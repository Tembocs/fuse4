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

func TestBuildFunctionWithLet(t *testing.T) {
	result := Build(BuildOptions{
		Sources: map[string][]byte{
			"main": []byte("fn main() { let x = 42; }"),
		},
	})
	for _, e := range result.Errors {
		t.Errorf("error: %s", e)
	}
	if !strings.Contains(result.CSource, "42") {
		t.Errorf("C source should contain literal 42\n%s", result.CSource)
	}
}

func TestBuildFunctionWithReturn(t *testing.T) {
	result := Build(BuildOptions{
		Sources: map[string][]byte{
			"main": []byte("fn get() -> I32 { return 1; }"),
		},
	})
	for _, e := range result.Errors {
		t.Errorf("error: %s", e)
	}
	if !strings.Contains(result.CSource, "return") {
		t.Errorf("C source should contain return\n%s", result.CSource)
	}
}

func TestBuildFunctionWithArithmetic(t *testing.T) {
	result := Build(BuildOptions{
		Sources: map[string][]byte{
			"main": []byte("fn add() { let x = 1 + 2; }"),
		},
	})
	for _, e := range result.Errors {
		t.Errorf("error: %s", e)
	}
	if !strings.Contains(result.CSource, "+") {
		t.Errorf("C source should contain +\n%s", result.CSource)
	}
}

func TestBuildFunctionWithIfElse(t *testing.T) {
	result := Build(BuildOptions{
		Sources: map[string][]byte{
			"main": []byte("fn test() { if true { let x = 1; } }"),
		},
	})
	for _, e := range result.Errors {
		t.Errorf("error: %s", e)
	}
	hasIf := strings.Contains(result.CSource, "if")
	hasGoto := strings.Contains(result.CSource, "goto")
	if !hasIf && !hasGoto {
		t.Errorf("C source should contain if or goto\n%s", result.CSource)
	}
}

func TestBuildFunctionWithCall(t *testing.T) {
	result := Build(BuildOptions{
		Sources: map[string][]byte{
			"main": []byte("fn foo() { } fn main() { foo(); }"),
		},
	})
	for _, e := range result.Errors {
		t.Errorf("error: %s", e)
	}
	if !strings.Contains(result.CSource, "foo") {
		t.Errorf("C source should contain foo\n%s", result.CSource)
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
