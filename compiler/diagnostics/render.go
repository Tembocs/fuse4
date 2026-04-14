package diagnostics

import (
	"encoding/json"
	"fmt"
	"strings"
)

// RenderText renders diagnostics in human-readable format with spans and context.
func RenderText(diags []Diagnostic, sources map[string][]byte) string {
	var b strings.Builder
	for _, d := range diags {
		b.WriteString(d.Severity.String())
		if d.Span.File != "" {
			b.WriteString(fmt.Sprintf("[%s:%d:%d]", d.Span.File, d.Span.Start.Line, d.Span.Start.Col))
		}
		b.WriteString(": ")
		b.WriteString(d.Message)
		b.WriteByte('\n')

		// Show source context if available.
		if d.Span.File != "" && d.Span.Start.Line > 0 {
			if src, ok := sources[d.Span.File]; ok {
				line := getLine(src, d.Span.Start.Line)
				if line != "" {
					b.WriteString("  ")
					b.WriteString(line)
					b.WriteByte('\n')
					// Underline the span.
					col := d.Span.Start.Col
					if col > 0 {
						b.WriteString("  ")
						b.WriteString(strings.Repeat(" ", col-1))
						length := d.Span.End.Col - d.Span.Start.Col
						if length <= 0 {
							length = 1
						}
						b.WriteString(strings.Repeat("^", length))
						b.WriteByte('\n')
					}
				}
			}
		}
	}
	return b.String()
}

func getLine(src []byte, lineNum int) string {
	line := 1
	start := 0
	for i, ch := range src {
		if line == lineNum {
			end := i
			for end < len(src) && src[end] != '\n' {
				end++
			}
			return string(src[start:end])
		}
		if ch == '\n' {
			line++
			start = i + 1
		}
	}
	if line == lineNum {
		return string(src[start:])
	}
	return ""
}

// JSONDiagnostic is the machine-readable diagnostic format.
type JSONDiagnostic struct {
	Severity string `json:"severity"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	EndLine  int    `json:"end_line,omitempty"`
	EndCol   int    `json:"end_column,omitempty"`
	Message  string `json:"message"`
}

// RenderJSON renders diagnostics as a JSON array.
func RenderJSON(diags []Diagnostic) string {
	jdiags := make([]JSONDiagnostic, len(diags))
	for i, d := range diags {
		jdiags[i] = JSONDiagnostic{
			Severity: d.Severity.String(),
			File:     d.Span.File,
			Line:     d.Span.Start.Line,
			Column:   d.Span.Start.Col,
			EndLine:  d.Span.End.Line,
			EndCol:   d.Span.End.Col,
			Message:  d.Message,
		}
	}
	out, _ := json.MarshalIndent(jdiags, "", "  ")
	return string(out) + "\n"
}
