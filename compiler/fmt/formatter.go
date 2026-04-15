// Package fmt owns the Fuse source code formatter.
//
// The formatter operates on the token stream (not the AST) so that
// comments are preserved. It normalizes whitespace, indentation, and
// spacing while maintaining the logical structure of the source.
package fmt

import (
	"bytes"
	"strings"

	"github.com/Tembocs/fuse4/compiler/lex"
)

// Format formats a Fuse source file and returns the formatted bytes.
func Format(filename string, src []byte) ([]byte, error) {
	l := lex.New(filename, src)
	tokens, _ := l.TokenizeWithComments()
	f := &formatter{
		tokens: tokens,
		indent: 0,
	}
	f.format()
	return f.out.Bytes(), nil
}

type formatter struct {
	tokens []lex.Token
	pos    int
	indent int
	out    bytes.Buffer

	// Tracks whether we just wrote a newline (are at line start).
	atLineStart  bool
	// Tracks whether we just emitted a blank line separator.
	blankEmitted bool
	// The previous non-whitespace token kind emitted.
	prevKind lex.TokenKind
	// The previous token (for context decisions).
	prevToken lex.Token
}

func (f *formatter) peek() lex.Token {
	if f.pos < len(f.tokens) {
		return f.tokens[f.pos]
	}
	return lex.Token{Kind: lex.EOF}
}

func (f *formatter) peekAt(offset int) lex.Token {
	idx := f.pos + offset
	if idx >= 0 && idx < len(f.tokens) {
		return f.tokens[idx]
	}
	return lex.Token{Kind: lex.EOF}
}

func (f *formatter) advance() lex.Token {
	t := f.peek()
	if t.Kind != lex.EOF {
		f.pos++
	}
	return t
}

func (f *formatter) format() {
	f.atLineStart = true
	for f.peek().Kind != lex.EOF {
		f.formatToken()
	}
	// Ensure file ends with a single newline.
	f.ensureTrailingNewline()
}

func (f *formatter) formatToken() {
	tok := f.peek()

	switch tok.Kind {
	case lex.LineComment:
		f.emitLineComment()
	case lex.BlockComment:
		f.emitBlockComment()
	case lex.LBrace:
		f.emitOpenBrace()
	case lex.RBrace:
		f.emitCloseBrace()
	case lex.Semi:
		f.emitSemicolon()
	case lex.Comma:
		f.emitComma()
	case lex.Colon:
		f.emitColon()
	case lex.LParen:
		f.emitLParen()
	case lex.RParen:
		f.emitRParen()
	case lex.LBrack:
		f.emitLBrack()
	case lex.RBrack:
		f.emitRBrack()
	case lex.Arrow:
		f.emitSpaced()
	case lex.FatArrow:
		f.emitSpaced()
	case lex.Dot, lex.QDot:
		f.emitNoSpace()
	default:
		if isBinaryOp(tok.Kind) && !isUnaryContext(f.prevKind) {
			f.emitSpaced()
		} else {
			f.emitDefault()
		}
	}
}

// --- emit helpers ---

func (f *formatter) writeIndent() {
	for i := 0; i < f.indent; i++ {
		f.out.WriteString("    ")
	}
	f.atLineStart = false
	f.blankEmitted = false
}

func (f *formatter) newline() {
	f.out.WriteByte('\n')
	f.atLineStart = true
	f.blankEmitted = false
}

func (f *formatter) blankLine() {
	if !f.blankEmitted {
		f.out.WriteByte('\n')
		f.blankEmitted = true
	}
}

func (f *formatter) space() {
	if !f.atLineStart {
		f.out.WriteByte(' ')
	}
}

func (f *formatter) writeToken(tok lex.Token) {
	if f.atLineStart {
		f.writeIndent()
	}
	f.out.WriteString(tok.Literal)
	f.prevKind = tok.Kind
	f.prevToken = tok
	f.atLineStart = false
	f.blankEmitted = false
}

// emitDefault emits a token with standard spacing (space before if not at line start).
func (f *formatter) emitDefault() {
	tok := f.advance()
	if !f.atLineStart && needsSpaceBefore(f.prevKind, tok.Kind) {
		f.space()
	}
	f.writeToken(tok)
}

// emitSpaced emits a token with spaces on both sides (binary ops, arrows).
func (f *formatter) emitSpaced() {
	tok := f.advance()
	if !f.atLineStart {
		f.space()
	}
	f.writeToken(tok)
	// Space after is handled by the next token's needsSpaceBefore.
}

// emitNoSpace emits a token with no space before it (dots).
func (f *formatter) emitNoSpace() {
	tok := f.advance()
	f.writeToken(tok)
}

func (f *formatter) emitOpenBrace() {
	tok := f.advance()
	if !f.atLineStart {
		f.space()
	}
	f.writeToken(tok)
	f.indent++
	f.newline()
}

func (f *formatter) emitCloseBrace() {
	f.indent--
	tok := f.advance()
	if !f.atLineStart {
		f.newline()
	}
	f.writeToken(tok)
	// Newline after }, unless followed by else, comma, semicolon, or rparen.
	next := f.peek()
	if next.Kind == lex.KwElse {
		f.space()
	} else if next.Kind != lex.Comma && next.Kind != lex.Semi &&
		next.Kind != lex.RParen && next.Kind != lex.EOF {
		f.newline()
		// Blank line between top-level items.
		if f.indent == 0 && isTopLevelStart(next.Kind) {
			f.blankLine()
		}
	}
}

func (f *formatter) emitSemicolon() {
	tok := f.advance()
	f.writeToken(tok)
	// Newline after semicolon, unless next is } (handled by closeBrace) or EOF.
	next := f.peek()
	if next.Kind != lex.RBrace && next.Kind != lex.EOF {
		f.newline()
	}
}

func (f *formatter) emitComma() {
	tok := f.advance()
	f.writeToken(tok)
	// Space after comma (on same line).
	next := f.peek()
	if next.Kind != lex.RBrace && next.Kind != lex.RBrack &&
		next.Kind != lex.RParen && next.Kind != lex.EOF {
		// Don't newline after commas in short lists; just space.
		f.space()
	}
}

func (f *formatter) emitColon() {
	tok := f.advance()
	// No space before colon.
	f.writeToken(tok)
	// Space after colon.
}

func (f *formatter) emitLParen() {
	tok := f.advance()
	// No space before ( when preceded by ident, keyword, or ].
	f.writeToken(tok)
}

func (f *formatter) emitRParen() {
	tok := f.advance()
	f.writeToken(tok)
}

func (f *formatter) emitLBrack() {
	tok := f.advance()
	// No space before [ when preceded by ident or ].
	if !f.atLineStart && f.prevKind != lex.Ident && f.prevKind != lex.RBrack &&
		f.prevKind != lex.RParen && !isLiteral(f.prevKind) {
		f.space()
	}
	f.writeToken(tok)
}

func (f *formatter) emitRBrack() {
	tok := f.advance()
	f.writeToken(tok)
}

func (f *formatter) emitLineComment() {
	tok := f.advance()
	if !f.atLineStart {
		// Trailing comment on a line — add space before.
		f.space()
	} else {
		f.writeIndent()
		f.atLineStart = false
	}
	f.out.WriteString(tok.Literal)
	f.prevKind = tok.Kind
	f.prevToken = tok
	f.newline()
}

func (f *formatter) emitBlockComment() {
	tok := f.advance()
	if !f.atLineStart {
		f.space()
	} else {
		f.writeIndent()
		f.atLineStart = false
	}
	f.out.WriteString(tok.Literal)
	f.prevKind = tok.Kind
	f.prevToken = tok
}

func (f *formatter) ensureTrailingNewline() {
	b := f.out.Bytes()
	if len(b) == 0 {
		return
	}
	// Remove trailing blank lines, keep exactly one newline.
	trimmed := bytes.TrimRight(b, "\n\r ")
	f.out.Reset()
	f.out.Write(trimmed)
	f.out.WriteByte('\n')
}

// --- classification helpers ---

func needsSpaceBefore(prev, cur lex.TokenKind) bool {
	// No space after open delimiters.
	if prev == lex.LParen || prev == lex.LBrack {
		return false
	}
	// No space before close delimiters.
	if cur == lex.RParen || cur == lex.RBrack {
		return false
	}
	// No space before comma, semicolon, colon.
	if cur == lex.Comma || cur == lex.Semi || cur == lex.Colon {
		return false
	}
	// No space after dot.
	if prev == lex.Dot || prev == lex.QDot {
		return false
	}
	// No space before dot.
	if cur == lex.Dot || cur == lex.QDot {
		return false
	}
	// No space before ( when preceded by ident, keyword, or ]
	if cur == lex.LParen {
		if prev == lex.Ident || prev == lex.RBrack || prev == lex.RParen ||
			prev == lex.KwSelfValue || prev == lex.Gt ||
			isLiteral(prev) {
			return false
		}
	}
	// No space before [ when preceded by ident (indexing).
	if cur == lex.LBrack {
		if prev == lex.Ident || prev == lex.RBrack || prev == lex.RParen ||
			isLiteral(prev) {
			return false
		}
	}
	// No space before ? (postfix try).
	if cur == lex.Question {
		return false
	}
	return true
}

func isBinaryOp(k lex.TokenKind) bool {
	switch k {
	case lex.Plus, lex.Minus, lex.Star, lex.Slash, lex.Percent,
		lex.Amp, lex.Pipe, lex.Caret, lex.Shl, lex.Shr,
		lex.AmpAmp, lex.PipePipe,
		lex.EqEq, lex.BangEq, lex.Lt, lex.Gt, lex.LtEq, lex.GtEq,
		lex.Eq, lex.PlusEq, lex.MinusEq, lex.StarEq, lex.SlashEq,
		lex.PercentEq, lex.AmpEq, lex.PipeEq, lex.CaretEq,
		lex.ShlEq, lex.ShrEq:
		return true
	}
	return false
}

func isUnaryContext(prev lex.TokenKind) bool {
	// A minus/star/bang after these tokens is unary, not binary.
	switch prev {
	case lex.EOF, lex.LParen, lex.LBrack, lex.Comma, lex.Semi,
		lex.Eq, lex.PlusEq, lex.MinusEq, lex.StarEq, lex.SlashEq,
		lex.PercentEq, lex.AmpEq, lex.PipeEq, lex.CaretEq,
		lex.ShlEq, lex.ShrEq,
		lex.KwReturn, lex.KwLet, lex.KwVar, lex.Arrow, lex.FatArrow,
		lex.LBrace, lex.Colon:
		return true
	}
	return false
}

func isTopLevelStart(k lex.TokenKind) bool {
	switch k {
	case lex.KwFn, lex.KwPub, lex.KwStruct, lex.KwEnum, lex.KwTrait,
		lex.KwImpl, lex.KwConst, lex.KwType, lex.KwExtern, lex.KwImport,
		lex.KwUnsafe, lex.At:
		return true
	}
	return false
}

func isLiteral(k lex.TokenKind) bool {
	switch k {
	case lex.IntLit, lex.FloatLit, lex.StringLit, lex.RawStringLit,
		lex.KwTrue, lex.KwFalse:
		return true
	}
	return false
}

// Check reports whether the source is already formatted.
// Returns true if the source matches the formatted output.
func Check(filename string, src []byte) (bool, error) {
	formatted, err := Format(filename, src)
	if err != nil {
		return false, err
	}
	return bytes.Equal(src, formatted), nil
}

// FormatString is a convenience wrapper for tests.
func FormatString(src string) string {
	out, _ := Format("input.fuse", []byte(src))
	return string(out)
}

// DiffFiles returns the list of filenames whose content differs from
// the formatted output. Used by `fuse fmt --check`.
func DiffFiles(files map[string][]byte) []string {
	var diffs []string
	for name, src := range files {
		ok, _ := Check(name, src)
		if !ok {
			diffs = append(diffs, strings.TrimSuffix(name, ".fuse"))
		}
	}
	return diffs
}
