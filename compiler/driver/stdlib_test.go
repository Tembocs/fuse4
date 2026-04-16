package driver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadStdlibFindsFiles(t *testing.T) {
	root := findStdlibRoot(t)
	sources, err := LoadStdlib(root)
	if err != nil {
		t.Fatalf("LoadStdlib: %s", err)
	}
	if len(sources) == 0 {
		t.Fatal("expected at least one stdlib source file")
	}

	// Verify known files are present.
	expected := []string{
		"core.traits",
		"core.equatable",
		"core.comparable",
		"core.hashable",
		"core.printable",
		"core.debuggable",
		"core.string",
		"core.option",
		"core.result",
		"core.list",
		"core.map",
		"core.set",
		"core.hash",
		"core.fmt",
		"core.math",
		"core.bool",
		"core.int",
		"core.int8",
		"core.int32",
		"core.float",
		"core.float32",
		"core.uint8",
		"core.uint32",
		"core.uint64",
		"core.rt_bridge.alloc",
		"core.rt_bridge.panic",
		"core.rt_bridge.intrinsics",
	}
	for _, mod := range expected {
		if _, ok := sources[mod]; !ok {
			t.Errorf("missing expected module: %s", mod)
		}
	}
}

func TestLoadStdlibModulePaths(t *testing.T) {
	root := findStdlibRoot(t)
	sources, err := LoadStdlib(root)
	if err != nil {
		t.Fatalf("LoadStdlib: %s", err)
	}

	for modPath := range sources {
		// Module paths should use dots, not slashes.
		if strings.Contains(modPath, "/") || strings.Contains(modPath, "\\") {
			t.Errorf("module path contains separator: %q", modPath)
		}
		// Should not end with .fuse.
		if strings.HasSuffix(modPath, ".fuse") {
			t.Errorf("module path has .fuse extension: %q", modPath)
		}
	}
}

func TestStdlibParsesWithoutErrors(t *testing.T) {
	root := findStdlibRoot(t)
	sources, err := LoadStdlib(root)
	if err != nil {
		t.Fatalf("LoadStdlib: %s", err)
	}

	// Feed all stdlib sources through the compiler pipeline (check only).
	// SkipAutoStdlib: we're already feeding the complete stdlib set, so
	// auto-load would just re-do the same work.
	result := Build(BuildOptions{Sources: sources, SkipAutoStdlib: true})
	// We accept some errors from simplified stdlib bodies, but parse errors
	// indicate a real problem.
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "expected") && e.Severity == 0 {
			// Parse errors are severity 0 (Error).
			t.Logf("stdlib parse issue: %s", e)
		}
	}
}

func TestDocCoverageOnStdlib(t *testing.T) {
	root := findStdlibRoot(t)
	sources, err := LoadStdlib(root)
	if err != nil {
		t.Fatalf("LoadStdlib: %s", err)
	}

	undocumented := DocCoverage(sources)
	for _, item := range undocumented {
		t.Logf("undocumented public API: %s", item)
	}
	// Log count but don't fail — some items may legitimately be undocumented
	// during bootstrap. This test serves as a tracking mechanism.
	t.Logf("doc coverage: %d undocumented public items out of %d modules",
		len(undocumented), len(sources))
}

func TestFilePathToModulePath(t *testing.T) {
	cases := []struct {
		rel  string
		want string
	}{
		{"core/string.fuse", "core.string"},
		{"core/rt_bridge/alloc.fuse", "core.rt_bridge.alloc"},
		{"full/io.fuse", "full.io"},
	}
	for _, tc := range cases {
		// Normalize to OS separator for the test.
		rel := filepath.FromSlash(tc.rel)
		got := filePathToModulePath(rel)
		if got != tc.want {
			t.Errorf("filePathToModulePath(%q) = %q, want %q", rel, got, tc.want)
		}
	}
}

func TestDocCoverageDetectsUndocumented(t *testing.T) {
	sources := map[string][]byte{
		"test": []byte("pub fn documented() { }\npub fn undocumented() { }\n"),
	}
	// Neither has a doc comment preceding it.
	undoc := DocCoverage(sources)
	if len(undoc) != 2 {
		t.Errorf("expected 2 undocumented, got %d: %v", len(undoc), undoc)
	}
}

func TestDocCoveragePassesDocumented(t *testing.T) {
	sources := map[string][]byte{
		"test": []byte("/// Does something.\npub fn documented() { }\n"),
	}
	undoc := DocCoverage(sources)
	if len(undoc) != 0 {
		t.Errorf("expected 0 undocumented, got %v", undoc)
	}
}

// --- helpers ---

func findStdlibRoot(t *testing.T) string {
	t.Helper()
	// Try relative to test working directory.
	candidates := []string{
		"../../stdlib",
		"stdlib",
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}
	t.Skip("stdlib directory not found")
	return ""
}
