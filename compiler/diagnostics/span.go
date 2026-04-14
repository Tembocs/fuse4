package diagnostics

import "fmt"

// Pos is a position in a source file.
type Pos struct {
	Offset int // byte offset from start of file
	Line   int // 1-based line number
	Col    int // 1-based column (byte offset from line start)
}

func (p Pos) String() string {
	return fmt.Sprintf("%d:%d", p.Line, p.Col)
}

// Span is a range in a source file.
type Span struct {
	File  string
	Start Pos
	End   Pos
}

func (s Span) String() string {
	if s.File != "" {
		return fmt.Sprintf("%s:%s", s.File, s.Start)
	}
	return s.Start.String()
}
