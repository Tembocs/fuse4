package diagnostics

import "fmt"

// Diagnostic is a single compiler diagnostic message.
type Diagnostic struct {
	Severity Severity
	Span     Span
	Message  string
}

func (d Diagnostic) String() string {
	return fmt.Sprintf("%s: %s: %s", d.Span, d.Severity, d.Message)
}

// Errorf creates an error diagnostic at the given span.
func Errorf(span Span, format string, args ...any) Diagnostic {
	return Diagnostic{
		Severity: Error,
		Span:     span,
		Message:  fmt.Sprintf(format, args...),
	}
}
