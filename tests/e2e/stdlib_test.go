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
		"stdlib/core/traits.fuse",
		"stdlib/core/option.fuse",
		"stdlib/core/result.fuse",
		"stdlib/core/primitives.fuse",
		"stdlib/core/string.fuse",
		"stdlib/core/hash.fuse",
		"stdlib/core/collections.fuse",
		"stdlib/core/rt_bridge/alloc.fuse",
		"stdlib/core/rt_bridge/panic.fuse",
		"stdlib/core/rt_bridge/intrinsics.fuse",
	}

	for _, f := range coreFiles {
		t.Run(filepath.Base(f), func(t *testing.T) {
			src, err := os.ReadFile(filepath.Join(root, f))
			if err != nil {
				t.Fatalf("read: %v", err)
			}
			compileStdlibFile(t, filepath.Base(f), src)
		})
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
	coreDeps := map[string]string{
		"stdlib/core/option.fuse":  "core.option",
		"stdlib/core/result.fuse":  "core.result",
		"stdlib/core/string.fuse":  "core.string",
		"stdlib/core/traits.fuse":  "core.traits",
	}
	coreFiles := map[string]*ast.File{}
	for path, modName := range coreDeps {
		src, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			continue
		}
		parsed, errs := parse.Parse(filepath.Base(path), src)
		if len(errs) > 0 {
			continue
		}
		coreFiles[modName] = parsed
	}

	fullFiles := []string{
		"stdlib/full/io.fuse",
		"stdlib/full/os.fuse",
		"stdlib/full/thread.fuse",
		"stdlib/full/sync.fuse",
		"stdlib/full/time.fuse",
		"stdlib/full/chan.fuse",
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

			// Build module graph with core deps + this full module.
			files := map[string]*ast.File{}
			for k, v := range coreFiles {
				files[k] = v
			}
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

func compileStdlibFile(t *testing.T, name string, src []byte) {
	t.Helper()
	modName := name[:len(name)-5] // strip .fuse

	parsed, errs := parse.Parse(name, src)
	if len(errs) > 0 {
		t.Fatalf("parse: %v", errs[0])
	}

	files := map[string]*ast.File{modName: parsed}
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
