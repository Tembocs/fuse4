package driver

import (
	"strings"
	"testing"
)

func TestBuildEmptyModule(t *testing.T) {
	result := Build(BuildOptions{
		Sources: map[string][]byte{
			"main": []byte(""),
		},
	})
	if hasErrors(result.Errors) {
		for _, e := range result.Errors {
			t.Logf("error: %s", e)
		}
	}
}

func TestBuildSimpleFunction(t *testing.T) {
	result := Build(BuildOptions{
		Sources: map[string][]byte{
			"main": []byte("fn main() { }"),
		},
	})
	for _, e := range result.Errors {
		t.Errorf("error: %s", e)
	}
	if result.CSource == "" {
		t.Error("expected non-empty C source")
	}
	if !strings.Contains(result.CSource, "#include") {
		t.Error("C source should have includes")
	}
}

func TestBuildMultipleModules(t *testing.T) {
	result := Build(BuildOptions{
		Sources: map[string][]byte{
			"core.math": []byte("pub fn add(a: I32, b: I32) -> I32 { a + b }"),
			"main":      []byte("import core.math; fn main() { }"),
		},
	})
	for _, e := range result.Errors {
		t.Errorf("error: %s", e)
	}
	if result.CSource == "" {
		t.Error("expected non-empty C source")
	}
}

func TestBuildParseFails(t *testing.T) {
	result := Build(BuildOptions{
		Sources: map[string][]byte{
			"main": []byte("fn {{{ bad syntax"),
		},
	})
	if !hasErrors(result.Errors) {
		t.Error("expected parse errors")
	}
}

func TestBuildWithOutput(t *testing.T) {
	// Only test C source generation, not actual compile+link (may not have runtime built).
	result := Build(BuildOptions{
		Sources: map[string][]byte{
			"main": []byte("fn hello() { let x = 42; }"),
		},
		// No OutputPath — skips compile+link.
	})
	for _, e := range result.Errors {
		t.Errorf("error: %s", e)
	}
	if result.CSource == "" {
		t.Error("expected C source")
	}
}

func TestCSourceContainsFunction(t *testing.T) {
	result := Build(BuildOptions{
		Sources: map[string][]byte{
			"main": []byte("fn test_func() { let x = 1; }"),
		},
	})
	for _, e := range result.Errors {
		t.Errorf("error: %s", e)
	}
	if !strings.Contains(result.CSource, "test_func") {
		t.Errorf("C source should contain function name\n%s", result.CSource)
	}
}

func TestRuntimeLibDiscovery(t *testing.T) {
	// Just test that FindRuntimeLib doesn't panic.
	// It may or may not find the lib depending on CWD.
	_ = FindRuntimeLib()
}

func TestBuildResultHasCSource(t *testing.T) {
	result := Build(BuildOptions{
		Sources: map[string][]byte{
			"main": []byte("fn main() -> I32 { return 0; }"),
		},
	})
	if result.CSource == "" {
		t.Error("expected generated C source even without OutputPath")
	}
	if !strings.Contains(result.CSource, "return") {
		t.Error("C source should contain return statement")
	}
}

func TestBuildDiagnosticsAreCollected(t *testing.T) {
	// An import referencing a nonexistent module should produce a resolve error.
	result := Build(BuildOptions{
		Sources: map[string][]byte{
			"main": []byte("import nonexistent.module; fn main() { }"),
		},
	})
	if !hasErrors(result.Errors) {
		t.Error("expected resolve errors for nonexistent import")
	}
	found := false
	for _, e := range result.Errors {
		if strings.Contains(e.Message, "unresolved") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'unresolved import' diagnostic")
	}
}
