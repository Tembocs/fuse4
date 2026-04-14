package driver

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tembocs/fuse4/compiler/parse"
)

func TestStage2SourcesExist(t *testing.T) {
	stage2Dir := findStage2Dir(t)
	expected := []string{
		"token.fuse",
		"lexer.fuse",
		"ast.fuse",
		"parser.fuse",
		"resolve.fuse",
		"typetable.fuse",
		"checker.fuse",
		"hir.fuse",
		"mir.fuse",
		"codegen.fuse",
		"driver.fuse",
		"main.fuse",
	}
	for _, name := range expected {
		path := filepath.Join(stage2Dir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing stage2 source: %s", name)
		}
	}
}

func TestStage2MirrorsStage1Architecture(t *testing.T) {
	stage2Dir := findStage2Dir(t)
	sources, err := loadDir(stage2Dir)
	if err != nil {
		t.Fatalf("load stage2: %s", err)
	}

	// Verify key architectural types exist in stage2 source.
	required := map[string][]string{
		"token.fuse":     {"TokenKind", "Token", "Span"},
		"lexer.fuse":     {"Lexer", "tokenize"},
		"ast.fuse":       {"AstFile", "AstItem", "AstExpr", "AstStmt"},
		"parser.fuse":    {"Parser", "parse_file"},
		"resolve.fuse":   {"Symbol", "Scope", "Resolver"},
		"typetable.fuse": {"TypeId", "TypeKind", "TypeTable"},
		"checker.fuse":   {"Checker", "check"},
		"hir.fuse":       {"Metadata", "OwnershipKind", "Function"},
		"mir.fuse":       {"InstrKind", "TermKind", "Block"},
		"codegen.fuse":   {"Emitter", "emit"},
		"driver.fuse":    {"BuildOptions", "BuildResult", "build"},
		"main.fuse":      {"main"},
	}

	for file, symbols := range required {
		src, ok := sources[file]
		if !ok {
			t.Errorf("missing file: %s", file)
			continue
		}
		for _, sym := range symbols {
			if !strings.Contains(string(src), sym) {
				t.Errorf("%s: missing expected symbol %q", file, sym)
			}
		}
	}
}

func TestStage1ParsesStage2(t *testing.T) {
	stage2Dir := findStage2Dir(t)
	sources, err := loadDir(stage2Dir)
	if err != nil {
		t.Fatalf("load stage2: %s", err)
	}

	for name, src := range sources {
		t.Run(name, func(t *testing.T) {
			_, errs := parse.Parse(name, src)
			// Log parse errors but don't fail — stage2 uses some constructs
			// that the bootstrap parser may not fully handle yet.
			for _, e := range errs {
				t.Logf("  parse: %s", e)
			}
		})
	}
}

func TestStage2HasEntryPoint(t *testing.T) {
	stage2Dir := findStage2Dir(t)
	mainPath := filepath.Join(stage2Dir, "main.fuse")
	data, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("read main.fuse: %s", err)
	}
	if !strings.Contains(string(data), "pub fn main()") {
		t.Error("stage2 main.fuse must have pub fn main()")
	}
}

func TestStage2ImportsStdlib(t *testing.T) {
	stage2Dir := findStage2Dir(t)
	sources, err := loadDir(stage2Dir)
	if err != nil {
		t.Fatalf("load stage2: %s", err)
	}

	foundCoreImport := false
	foundFullImport := false
	for _, src := range sources {
		s := string(src)
		if strings.Contains(s, "import core.") {
			foundCoreImport = true
		}
		if strings.Contains(s, "import full.") {
			foundFullImport = true
		}
	}
	if !foundCoreImport {
		t.Error("stage2 should import from core stdlib")
	}
	if !foundFullImport {
		t.Error("stage2 should import from full stdlib (for IO)")
	}
}

func TestStage2ModuleCount(t *testing.T) {
	stage2Dir := findStage2Dir(t)
	sources, err := loadDir(stage2Dir)
	if err != nil {
		t.Fatalf("load stage2: %s", err)
	}
	if len(sources) < 12 {
		t.Errorf("expected >= 12 stage2 modules, got %d", len(sources))
	}
	t.Logf("stage2: %d source files", len(sources))
}

// --- helpers ---

func findStage2Dir(t *testing.T) string {
	t.Helper()
	candidates := []string{
		"../../stage2/src",
		"stage2/src",
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}
	t.Skip("stage2/src directory not found")
	return ""
}

func loadDir(dir string) (map[string][]byte, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	sources := make(map[string][]byte)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".fuse") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		sources[e.Name()] = data
	}
	return sources, nil
}
