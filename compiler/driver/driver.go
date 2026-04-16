// Package driver owns the end-to-end orchestration of the Stage 1
// compiler pipeline.
package driver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Tembocs/fuse4/compiler/ast"
	"github.com/Tembocs/fuse4/compiler/cc"
	"github.com/Tembocs/fuse4/compiler/check"
	"github.com/Tembocs/fuse4/compiler/codegen"
	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/hir"
	"github.com/Tembocs/fuse4/compiler/liveness"
	"github.com/Tembocs/fuse4/compiler/lower"
	"github.com/Tembocs/fuse4/compiler/mir"
	"github.com/Tembocs/fuse4/compiler/monomorph"
	"github.com/Tembocs/fuse4/compiler/parse"
	"github.com/Tembocs/fuse4/compiler/resolve"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

// BuildOptions configures a compilation.
type BuildOptions struct {
	Sources        map[string][]byte // module path → source bytes
	OutputPath     string            // output executable path
	RuntimeLib     string            // path to libfuse_rt.a
	StdlibRoot     string            // explicit stdlib root; empty means auto-discover via StdlibRoot()
	Backend        string            // "c11" (default) or "native"
	Optimize       bool
	Debug          bool
	SkipAutoStdlib bool              // when true, the driver does not auto-load stdlib sources
}

// BuildResult holds the outcome of a compilation.
type BuildResult struct {
	Errors     []diagnostics.Diagnostic
	CSource    string // generated C (available even if compile/link fails)
	OutputPath string // final executable path
}

// Build runs the full compiler pipeline: parse → resolve → check → lower → codegen → compile → link.
func Build(opts BuildOptions) *BuildResult {
	result := &BuildResult{}

	// Phase 0: Auto-load stdlib sources unless explicitly skipped.
	// User sources take precedence: a user-provided module of the same
	// path as a stdlib module shadows the stdlib copy (see language-guide
	// §11.4 "Module loading").
	if !opts.SkipAutoStdlib {
		root := opts.StdlibRoot
		if root == "" {
			root = StdlibRoot()
		}
		if root == "" {
			result.Errors = append(result.Errors, diagnostics.Errorf(
				diagnostics.Span{}, "standard library not found: set FUSE_STDLIB_ROOT or pass BuildOptions.StdlibRoot"))
			return result
		}
		stdlibSources, err := LoadStdlib(root)
		if err != nil {
			result.Errors = append(result.Errors, diagnostics.Errorf(
				diagnostics.Span{}, "standard library not found at %s: %s", root, err))
			return result
		}
		if len(stdlibSources) == 0 {
			result.Errors = append(result.Errors, diagnostics.Errorf(
				diagnostics.Span{}, "standard library not found at %s: no .fuse files", root))
			return result
		}
		if opts.Sources == nil {
			opts.Sources = make(map[string][]byte)
		}
		for modPath, src := range stdlibSources {
			if _, userProvided := opts.Sources[modPath]; userProvided {
				continue // user-wins: shadow the stdlib copy
			}
			opts.Sources[modPath] = src
		}
	}

	// Phase 1: Parse all modules.
	parsed := make(map[string]*ast.File)
	for modPath, src := range opts.Sources {
		f, errs := parse.Parse(modPath+".fuse", src)
		result.Errors = append(result.Errors, errs...)
		parsed[modPath] = f
	}
	if hasErrors(result.Errors) {
		return result
	}

	// Phase 1.5: Enforce stdlib tier direction (Rule 5.4: ext → full → core).
	for modPath, f := range parsed {
		tier, isStdlib := stdlibTier(modPath)
		if !isStdlib {
			continue
		}
		for _, item := range f.Items {
			imp, ok := item.(*ast.ImportDecl)
			if !ok {
				continue
			}
			target := strings.Join(imp.Path, ".")
			targetTier, targetIsStdlib := stdlibTier(target)
			if !targetIsStdlib {
				continue
			}
			if !tierAllows(tier, targetTier) {
				result.Errors = append(result.Errors, diagnostics.Errorf(
					imp.Span,
					"module '%s' (%s) may not import '%s' (%s): stdlib tier direction is ext → full → core",
					modPath, tier, target, targetTier))
			}
		}
	}
	if hasErrors(result.Errors) {
		return result
	}

	// Phase 2: Resolve names and imports.
	graph := resolve.BuildModuleGraph(parsed)
	resolver := resolve.NewResolver(graph)
	resolver.Resolve()
	result.Errors = append(result.Errors, resolver.Errors...)
	if hasErrors(result.Errors) {
		return result
	}

	// Phase 2.5: Monomorphize — specialize generic functions at the AST level.
	// This runs before checking so the checker only sees concrete functions.
	monomorph.SpecializeModules(graph)

	// Phase 3: Type check.
	tt := typetable.New()
	checker := check.NewChecker(tt, graph)
	checker.Check()
	result.Errors = append(result.Errors, checker.Errors...)
	if hasErrors(result.Errors) {
		return result
	}

	// Phase 4: Build HIR, run ownership/liveness, lower to MIR.
	hirBuilder := hir.NewBuilder(tt)
	lowerer := lower.New(tt)
	var mirFunctions []*mir.Function

	for _, key := range graph.Order {
		mod := graph.Modules[key]
		for _, item := range mod.File.Items {
			switch it := item.(type) {
			case *ast.FnDecl:
				if it.Body == nil {
					continue
				}
				// Skip generic function originals — only their specialized
				// copies (produced by monomorph.SpecializeModules) are compiled.
				if monomorph.IsGenericFn(it) {
					continue
				}
				hirFn := buildHIRFunction(hirBuilder, tt, checker, mod.Path.String(), it)
				_, liveDiags := liveness.RunAll(hirFn)
				result.Errors = append(result.Errors, liveDiags...)
				mirFn := lowerer.LowerFunction(hirFn)
				mirFunctions = append(mirFunctions, mirFn)

			case *ast.ImplDecl:
				// Skip generic impl blocks — their specialized methods are
				// emitted as top-level functions by the monomorphizer.
				if monomorph.IsGenericImpl(it) {
					continue
				}
				for _, implItem := range it.Items {
					fn, ok := implItem.(*ast.FnDecl)
					if !ok || fn.Body == nil {
						continue
					}
					hirFn := buildHIRFunction(hirBuilder, tt, checker, mod.Path.String(), fn)
					_, liveDiags := liveness.RunAll(hirFn)
					result.Errors = append(result.Errors, liveDiags...)
					mirFn := lowerer.LowerFunction(hirFn)
					mirFunctions = append(mirFunctions, mirFn)
				}

			case *ast.ExternFnDecl:
				// Extern functions have no body — skip.
				continue
			}
		}
	}
	result.Errors = append(result.Errors, lowerer.Errors...)

	// Include lifted closure functions in the MIR output.
	mirFunctions = append(mirFunctions, lowerer.LiftedFunctions...)

	// Phase 5: Codegen — emit via selected backend.
	backendTarget := opts.Backend
	if backendTarget == "" {
		backendTarget = "c11"
	}
	backend := codegen.NewBackend(codegen.BackendConfig{
		Target:    backendTarget,
		Types:     tt,
		Optimize:  opts.Optimize,
		DropTypes: checker.DropTypes(),
	})
	output, err := backend.Emit(mirFunctions)
	if err != nil {
		result.Errors = append(result.Errors, diagnostics.Errorf(
			diagnostics.Span{}, "codegen: %s", err))
	}
	cSource := string(output)
	result.CSource = cSource
	if hasErrors(result.Errors) {
		return result
	}

	// Phase 6: Write C source and compile+link.
	if opts.OutputPath != "" {
		err := compileAndLink(cSource, opts)
		if err != nil {
			result.Errors = append(result.Errors, diagnostics.Errorf(
				diagnostics.Span{}, "build failed: %s", err))
		} else {
			result.OutputPath = opts.OutputPath
		}
	}

	return result
}

// compileAndLink writes the C source, invokes the C compiler, and links.
func compileAndLink(cSource string, opts BuildOptions) error {
	// Detect C compiler.
	toolchain, err := cc.Detect()
	if err != nil {
		return err
	}

	// Write generated C to a temp file.
	tmpDir, err := os.MkdirTemp("", "fuse-build-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	cPath := filepath.Join(tmpDir, "output.c")
	if err := os.WriteFile(cPath, []byte(cSource), 0644); err != nil {
		return fmt.Errorf("write C source: %w", err)
	}

	// Discover runtime library.
	rtLib := opts.RuntimeLib
	if rtLib == "" {
		rtLib = FindRuntimeLib()
	}

	// Derive runtime include directory from the runtime library path.
	var includeDirs []string
	if rtLib != "" {
		rtDir := filepath.Dir(rtLib)
		// Dev layout: runtime/libfuse_rt.a → runtime/include/
		includeDir := filepath.Join(rtDir, "include")
		if _, err := os.Stat(includeDir); err == nil {
			includeDirs = append(includeDirs, includeDir)
		}
		// Dev layout (CWD one level up): ../runtime/include/
		parentInclude := filepath.Join(filepath.Dir(rtDir), "runtime", "include")
		if _, err := os.Stat(parentInclude); err == nil {
			includeDirs = append(includeDirs, parentInclude)
		}
		// Packaged layout: lib/libfuse_rt.a → sibling include/
		siblingInclude := filepath.Join(filepath.Dir(rtDir), "include")
		if _, err := os.Stat(siblingInclude); err == nil {
			includeDirs = append(includeDirs, siblingInclude)
		}
	}

	// Compile.
	objPath := filepath.Join(tmpDir, "output.o")
	cfg := cc.BuildConfig{
		Optimize:    opts.Optimize,
		Debug:       opts.Debug,
		OutputObj:   objPath,
		IncludeDirs: includeDirs,
	}
	if err := toolchain.Compile(cPath, cfg); err != nil {
		return err
	}

	// Link.
	linkCfg := cc.BuildConfig{
		OutputExe: opts.OutputPath,
	}
	objects := []string{objPath}
	if err := toolchain.Link(objects, rtLib, linkCfg); err != nil {
		return err
	}

	return nil
}

// FindRuntimeLib searches for libfuse_rt.a in standard locations.
// Search order:
//  1. FUSE_RUNTIME_LIB environment variable (explicit override)
//  2. Relative to the executable (packaged distribution: <exe>/../lib/)
//  3. Relative to CWD (development: runtime/)
func FindRuntimeLib() string {
	if env := os.Getenv("FUSE_RUNTIME_LIB"); env != "" {
		return env
	}

	// Check relative to the executable (packaged layout: bin/fuse + lib/libfuse_rt.a).
	if exeRoot := exeDistRoot(); exeRoot != "" {
		c := filepath.Join(exeRoot, "lib", "libfuse_rt.a")
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	// Check relative to the current working directory (development layout).
	candidates := []string{
		"runtime/libfuse_rt.a",
		"../runtime/libfuse_rt.a",
		filepath.Join("..", "runtime", "libfuse_rt.a"),
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

// exeDistRoot returns the distribution root directory (the parent of bin/)
// when the running executable lives inside a bin/ directory. Returns ""
// if the executable location cannot be determined or isn't inside bin/.
func exeDistRoot() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return ""
	}
	binDir := filepath.Dir(exe)
	root := filepath.Dir(binDir)
	// Sanity: only return a root if the exe actually sits inside a bin/ dir.
	if filepath.Base(binDir) == "bin" {
		return root
	}
	return ""
}

// buildHIRFunction converts an AST FnDecl into a full HIR Function using the ast2hir bridge.
func buildHIRFunction(b *hir.Builder, tt *typetable.TypeTable, checker *check.Checker, modPath string, fn *ast.FnDecl) *hir.Function {
	bridge := &ast2hir{b: b, tt: tt, checker: checker, modPath: modPath}
	return bridge.lowerFunction(fn)
}

func hasErrors(errs []diagnostics.Diagnostic) bool {
	for _, e := range errs {
		if e.Severity == diagnostics.Error {
			return true
		}
	}
	return false
}

// stdlibTier returns the stdlib tier ("core", "full", "ext") of a module
// path, and whether the module belongs to the stdlib at all.
func stdlibTier(modPath string) (string, bool) {
	switch {
	case strings.HasPrefix(modPath, "core."), modPath == "core":
		return "core", true
	case strings.HasPrefix(modPath, "full."), modPath == "full":
		return "full", true
	case strings.HasPrefix(modPath, "ext."), modPath == "ext":
		return "ext", true
	}
	return "", false
}

// tierAllows reports whether a module in tier `src` is allowed to import
// from tier `dst`. Direction is `ext → full → core`; imports only flow
// toward `core`, never away from it (Rule 5.4).
func tierAllows(src, dst string) bool {
	switch src {
	case "core":
		return dst == "core"
	case "full":
		return dst == "core" || dst == "full"
	case "ext":
		return dst == "core" || dst == "full" || dst == "ext"
	}
	return true
}
