package diagnostics

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestRenderTextBasic(t *testing.T) {
	diags := []Diagnostic{
		Errorf(Span{File: "test.fuse", Start: Pos{Line: 3, Col: 5}}, "unexpected token"),
	}
	out := RenderText(diags, nil)
	if !strings.Contains(out, "error") {
		t.Error("should contain severity")
	}
	if !strings.Contains(out, "test.fuse:3:5") {
		t.Error("should contain span")
	}
	if !strings.Contains(out, "unexpected token") {
		t.Error("should contain message")
	}
}

func TestRenderTextWithSourceContext(t *testing.T) {
	src := []byte("fn main() {\n  let x = ;\n}\n")
	sources := map[string][]byte{"test.fuse": src}
	diags := []Diagnostic{
		Errorf(Span{
			File:  "test.fuse",
			Start: Pos{Line: 2, Col: 11},
			End:   Pos{Line: 2, Col: 12},
		}, "expected expression"),
	}
	out := RenderText(diags, sources)
	if !strings.Contains(out, "let x = ;") {
		t.Errorf("should show source line, got:\n%s", out)
	}
	if !strings.Contains(out, "^") {
		t.Error("should show underline caret")
	}
}

func TestRenderTextMultipleDiagnostics(t *testing.T) {
	diags := []Diagnostic{
		Errorf(Span{File: "a.fuse", Start: Pos{Line: 1, Col: 1}}, "first error"),
		Errorf(Span{File: "b.fuse", Start: Pos{Line: 5, Col: 10}}, "second error"),
	}
	out := RenderText(diags, nil)
	if strings.Count(out, "error[") != 2 {
		t.Errorf("expected 2 errors in output:\n%s", out)
	}
}

func TestRenderTextEmpty(t *testing.T) {
	out := RenderText(nil, nil)
	if out != "" {
		t.Error("empty diagnostics should produce empty output")
	}
}

func TestRenderJSONFormat(t *testing.T) {
	diags := []Diagnostic{
		Errorf(Span{
			File:  "test.fuse",
			Start: Pos{Line: 1, Col: 5},
			End:   Pos{Line: 1, Col: 10},
		}, "type mismatch"),
	}
	out := RenderJSON(diags)

	var parsed []JSONDiagnostic
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON: %s\n%s", err, out)
	}
	if len(parsed) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(parsed))
	}
	d := parsed[0]
	if d.Severity != "error" {
		t.Errorf("severity: %q", d.Severity)
	}
	if d.File != "test.fuse" {
		t.Errorf("file: %q", d.File)
	}
	if d.Line != 1 || d.Column != 5 {
		t.Errorf("position: %d:%d", d.Line, d.Column)
	}
	if d.Message != "type mismatch" {
		t.Errorf("message: %q", d.Message)
	}
}

func TestRenderJSONEmpty(t *testing.T) {
	out := RenderJSON(nil)
	var parsed []JSONDiagnostic
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON: %s", err)
	}
	if len(parsed) != 0 {
		t.Error("empty diagnostics should produce empty array")
	}
}

func TestRenderJSONMultiple(t *testing.T) {
	diags := []Diagnostic{
		Errorf(Span{}, "err1"),
		Errorf(Span{}, "err2"),
	}
	out := RenderJSON(diags)
	var parsed []JSONDiagnostic
	json.Unmarshal([]byte(out), &parsed)
	if len(parsed) != 2 {
		t.Errorf("expected 2, got %d", len(parsed))
	}
}
