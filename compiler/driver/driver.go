// Package driver owns the end-to-end orchestration of the Stage 1
// compiler pipeline.
package driver

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Tembocs/fuse4/compiler/ast"
	"github.com/Tembocs/fuse4/compiler/cc"
	"github.com/Tembocs/fuse4/compiler/check"
	"github.com/Tembocs/fuse4/compiler/codegen"
	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/hir"
	"github.com/Tembocs/fuse4/compiler/liveness"
	"github.com/Tembocs/fuse4/compiler/lower"
	"github.com/Tembocs/fuse4/compiler/mir"
	"github.com/Tembocs/fuse4/compiler/parse"
	"github.com/Tembocs/fuse4/compiler/resolve"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

// BuildOptions configures a compilation.
type BuildOptions struct {
	Sources    map[string][]byte // module path → source bytes
	OutputPath string            // output executable path
	RuntimeLib string            // path to libfuse_rt.a
	Backend    string            // "c11" (default) or "native"
	Optimize   bool
	Debug      bool
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

	// Phase 2: Resolve names and imports.
	graph := resolve.BuildModuleGraph(parsed)
	resolver := resolve.NewResolver(graph)
	resolver.Resolve()
	result.Errors = append(result.Errors, resolver.Errors...)
	if hasErrors(result.Errors) {
		return result
	}

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
			fn, ok := item.(*ast.FnDecl)
			if !ok || fn.Body == nil {
				continue
			}

			// Build a minimal HIR function for lowering.
			hirFn := buildHIRFunction(hirBuilder, tt, fn)

			// Run liveness (exactly once per function, Rule 3.8).
			_, liveDiags := liveness.RunAll(hirFn)
			result.Errors = append(result.Errors, liveDiags...)

			// Lower to MIR.
			mirFn := lowerer.LowerFunction(hirFn)
			mirFunctions = append(mirFunctions, mirFn)
		}
	}
	result.Errors = append(result.Errors, lowerer.Errors...)

	// Phase 5: Codegen — emit via selected backend.
	backendTarget := opts.Backend
	if backendTarget == "" {
		backendTarget = "c11"
	}
	backend := codegen.NewBackend(codegen.BackendConfig{
		Target:   backendTarget,
		Types:    tt,
		Optimize: opts.Optimize,
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

	// Compile.
	objPath := filepath.Join(tmpDir, "output.o")
	cfg := cc.BuildConfig{
		Optimize:  opts.Optimize,
		Debug:     opts.Debug,
		OutputObj: objPath,
	}
	if err := toolchain.Compile(cPath, cfg); err != nil {
		return err
	}

	// Discover runtime library.
	rtLib := opts.RuntimeLib
	if rtLib == "" {
		rtLib = FindRuntimeLib()
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
func FindRuntimeLib() string {
	// Check relative to the current working directory.
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

// buildHIRFunction creates a minimal HIR function from an AST FnDecl.
// This is a simplified bridge — full AST-to-HIR lowering comes in later refinement.
func buildHIRFunction(b *hir.Builder, tt *typetable.TypeTable, fn *ast.FnDecl) *hir.Function {
	var params []hir.Param
	for _, p := range fn.Params {
		params = append(params, hir.Param{
			Span: p.Span,
			Name: p.Name,
			Type: tt.Unknown,
		})
	}

	retType := tt.Unit
	body := b.BlockExpr(fn.Body.Span, nil, nil, retType)

	return &hir.Function{
		Name:       fn.Name,
		Public:     fn.Public,
		Params:     params,
		ReturnType: retType,
		Body:       body,
	}
}

func hasErrors(errs []diagnostics.Diagnostic) bool {
	for _, e := range errs {
		if e.Severity == diagnostics.Error {
			return true
		}
	}
	return false
}
