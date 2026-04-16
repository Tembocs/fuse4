package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Tembocs/fuse4/compiler/ast"
	"github.com/Tembocs/fuse4/compiler/check"
	"github.com/Tembocs/fuse4/compiler/monomorph"
	"github.com/Tembocs/fuse4/compiler/parse"
	"github.com/Tembocs/fuse4/compiler/resolve"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

// TestStdlibCoreCompiles verifies that each stdlib/core/ file compiles in isolation.
func TestStdlibCoreCompiles(t *testing.T) {
	root := findProjectRoot()
	if root == "" {
		t.Skip("project root not found")
	}

	coreFiles := []string{
		// Traits (individual files)
		"stdlib/core/equatable.fuse",
		"stdlib/core/comparable.fuse",
		"stdlib/core/hashable.fuse",
		"stdlib/core/printable.fuse",
		"stdlib/core/debuggable.fuse",
		"stdlib/core/traits.fuse",
		// Core types
		"stdlib/core/option.fuse",
		"stdlib/core/result.fuse",
		"stdlib/core/string.fuse",
		"stdlib/core/fmt.fuse",
		// Primitive type modules
		"stdlib/core/bool.fuse",
		"stdlib/core/int.fuse",
		"stdlib/core/int8.fuse",
		"stdlib/core/int32.fuse",
		"stdlib/core/float.fuse",
		"stdlib/core/float32.fuse",
		"stdlib/core/uint8.fuse",
		"stdlib/core/uint32.fuse",
		"stdlib/core/uint64.fuse",
		// Collections
		"stdlib/core/list.fuse",
		"stdlib/core/map.fuse",
		"stdlib/core/set.fuse",
		// Utilities
		"stdlib/core/hash.fuse",
		"stdlib/core/math.fuse",
		// Runtime bridge
		"stdlib/core/rt_bridge/alloc.fuse",
		"stdlib/core/rt_bridge/panic.fuse",
		"stdlib/core/rt_bridge/intrinsics.fuse",
	}

	// Each core file is compiled alongside its siblings so explicit imports
	// (e.g. `import core.option.Option` in list.fuse) can resolve. Before
	// Section 3 of STDLIB_INTEGRATION_TASKS.md these files used phantom
	// types instead of imports, so standalone compilation worked; now they
	// state their real dependencies and need the full graph.
	allCore := map[string]*ast.File{}
	for _, f := range coreFiles {
		src, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			continue
		}
		parsed, errs := parse.Parse(filepath.Base(f), src)
		if len(errs) > 0 {
			continue
		}
		allCore[coreModuleNameFor(f)] = parsed
	}
	for _, f := range coreFiles {
		t.Run(filepath.Base(f), func(t *testing.T) {
			compileStdlibFileWithDeps(t, f, allCore)
		})
	}
}

// coreModuleNameFor maps a stdlib/core/ file path to its module name.
func coreModuleNameFor(path string) string {
	base := filepath.Base(path)
	base = base[:len(base)-5]
	// Handle rt_bridge subdir — it maps to core.rt_bridge.<name>.
	if filepath.Base(filepath.Dir(path)) == "rt_bridge" {
		return "core.rt_bridge." + base
	}
	return "core." + base
}

// compileStdlibFileWithDeps type-checks one stdlib file alongside a set
// of already-parsed sibling modules so that cross-module imports resolve.
func compileStdlibFileWithDeps(t *testing.T, path string, all map[string]*ast.File) {
	t.Helper()
	target := coreModuleNameFor(path)
	parsed, ok := all[target]
	if !ok {
		t.Fatalf("file %q missing from dep set", path)
	}

	files := map[string]*ast.File{target: parsed}
	for k, v := range all {
		if k == target {
			continue
		}
		files[k] = v
	}

	graph := resolve.BuildModuleGraph(files)
	resolver := resolve.NewResolver(graph)
	resolver.Resolve()
	if len(resolver.Errors) > 0 {
		t.Fatalf("resolve: %v", resolver.Errors[0])
	}

	monomorph.SpecializeModules(graph)

	tt := typetable.New()
	chk := check.NewChecker(tt, graph)
	chk.Check()
	if len(chk.Errors) > 0 {
		t.Fatalf("check: %v", chk.Errors[0])
	}
}

// TestStdlibFullCompiles verifies that stdlib/full/ files compile with their
// core dependencies provided.
func TestStdlibFullCompiles(t *testing.T) {
	root := findProjectRoot()
	if root == "" {
		t.Skip("project root not found")
	}

	// Load core dependencies that full/ modules import.
	// The path→module map is defined once; each sub-test parses fresh
	// AST copies so the monomorphizer's in-place mutations on one
	// iteration do not leak into the next (Section 5 scanner reaches
	// struct-field types which can inject specs referencing types the
	// next test's graph does not contain).
	coreDeps := map[string]string{
		"stdlib/core/option.fuse":    "core.option",
		"stdlib/core/result.fuse":    "core.result",
		"stdlib/core/string.fuse":    "core.string",
		"stdlib/core/traits.fuse":    "core.traits",
		"stdlib/core/equatable.fuse": "core.equatable",
		"stdlib/core/list.fuse":      "core.list",
		"stdlib/core/map.fuse":       "core.map",
		"stdlib/core/hash.fuse":      "core.hash",
	}
	reparseCore := func() map[string]*ast.File {
		out := map[string]*ast.File{}
		for path, modName := range coreDeps {
			src, err := os.ReadFile(filepath.Join(root, path))
			if err != nil {
				continue
			}
			parsed, errs := parse.Parse(filepath.Base(path), src)
			if len(errs) > 0 {
				continue
			}
			out[modName] = parsed
		}
		return out
	}

	fullFiles := []string{
		"stdlib/full/chan.fuse",
		"stdlib/full/env.fuse",
		"stdlib/full/http.fuse",
		"stdlib/full/io.fuse",
		"stdlib/full/json.fuse",
		"stdlib/full/net.fuse",
		"stdlib/full/os.fuse",
		"stdlib/full/path.fuse",
		"stdlib/full/process.fuse",
		"stdlib/full/random.fuse",
		"stdlib/full/shared.fuse",
		"stdlib/full/simd.fuse",
		"stdlib/full/sys.fuse",
		"stdlib/full/time.fuse",
		"stdlib/full/timer.fuse",
	}

	for _, f := range fullFiles {
		t.Run(filepath.Base(f), func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join(root, f))
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			modName := filepath.Base(f)
			modName = modName[:len(modName)-5]

			parsed, errs := parse.Parse(filepath.Base(f), src)
			if len(errs) > 0 {
				t.Fatalf("parse: %v", errs[0])
			}

			// Build module graph with fresh core deps + this full module.
			files := reparseCore()
			files["full."+modName] = parsed

			graph := resolve.BuildModuleGraph(files)
			resolver := resolve.NewResolver(graph)
			resolver.Resolve()
			if len(resolver.Errors) > 0 {
				t.Fatalf("resolve: %v", resolver.Errors[0])
			}

			monomorph.SpecializeModules(graph)

			tt := typetable.New()
			chk := check.NewChecker(tt, graph)
			chk.Check()
			if len(chk.Errors) > 0 {
				t.Fatalf("check: %v", chk.Errors[0])
			}
		})
	}
}


// TestStdlibExtCompiles verifies that stdlib/ext/ files compile with their
// core and full dependencies provided.
func TestStdlibExtCompiles(t *testing.T) {
	root := findProjectRoot()
	if root == "" {
		t.Skip("project root not found")
	}

	// Load core dependencies. Re-parsed per sub-test to avoid cross-test
	// monomorph mutations (see TestStdlibFullCompiles for the same
	// pattern). Without this, a spec added to core.list from one ext
	// file's graph leaks into the next and references types not in
	// its dep set.
	coreDeps := map[string]string{
		"stdlib/core/option.fuse":    "core.option",
		"stdlib/core/result.fuse":    "core.result",
		"stdlib/core/string.fuse":    "core.string",
		"stdlib/core/traits.fuse":    "core.traits",
		"stdlib/core/equatable.fuse": "core.equatable",
		"stdlib/core/list.fuse":      "core.list",
		"stdlib/core/map.fuse":       "core.map",
		"stdlib/core/hash.fuse":      "core.hash",
	}
	reparseCore := func() map[string]*ast.File {
		out := map[string]*ast.File{}
		for path, modName := range coreDeps {
			src, err := os.ReadFile(filepath.Join(root, path))
			if err != nil {
				continue
			}
			parsed, errs := parse.Parse(filepath.Base(path), src)
			if len(errs) > 0 {
				continue
			}
			out[modName] = parsed
		}
		return out
	}

	extFiles := []string{
		"stdlib/ext/argparse.fuse",
		"stdlib/ext/crypto.fuse",
		"stdlib/ext/http_server.fuse",
		"stdlib/ext/json_schema.fuse",
		"stdlib/ext/jsonrpc.fuse",
		"stdlib/ext/log.fuse",
		"stdlib/ext/regex.fuse",
		"stdlib/ext/test.fuse",
		"stdlib/ext/toml.fuse",
		"stdlib/ext/uri.fuse",
		"stdlib/ext/yaml.fuse",
	}

	for _, f := range extFiles {
		t.Run(filepath.Base(f), func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join(root, f))
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			modName := filepath.Base(f)
			modName = modName[:len(modName)-5]

			parsed, errs := parse.Parse(filepath.Base(f), src)
			if len(errs) > 0 {
				t.Fatalf("parse: %v", errs[0])
			}

			files := reparseCore()
			files["ext."+modName] = parsed

			graph := resolve.BuildModuleGraph(files)
			resolver := resolve.NewResolver(graph)
			resolver.Resolve()
			if len(resolver.Errors) > 0 {
				t.Fatalf("resolve: %v", resolver.Errors[0])
			}

			monomorph.SpecializeModules(graph)

			tt := typetable.New()
			chk := check.NewChecker(tt, graph)
			chk.Check()
			if len(chk.Errors) > 0 {
				t.Fatalf("check: %v", chk.Errors[0])
			}
		})
	}
}

func findProjectRoot() string {
	cwd, _ := os.Getwd()
	dir := cwd
	for i := 0; i < 5; i++ {
		if _, err := os.Stat(filepath.Join(dir, "stdlib")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	return ""
}
