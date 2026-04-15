package lex

import (
	"unicode"
	"unicode/utf8"

	"github.com/Tembocs/fuse4/compiler/diagnostics"
)

// Lexer tokenizes Fuse source code.
type Lexer struct {
	src      []byte
	filename string

	pos  int // current read position (byte offset)
	line int // current line (1-based)
	col  int // current column (1-based, byte offset from line start)

	// start of the token currently being scanned
	startPos  int
	startLine int
	startCol  int

	tokens       []Token
	errors       []diagnostics.Diagnostic
	keepComments bool // when true, emit LineComment/BlockComment tokens
}

// New creates a lexer for the given source.
func New(filename string, src []byte) *Lexer {
	return &Lexer{
		src:      src,
		filename: filename,
		pos:      0,
		line:     1,
		col:      1,
	}
}

// Tokenize lexes the full source and returns all tokens and diagnostics.
// The token slice always ends with an EOF token.
func (l *Lexer) Tokenize() ([]Token, []diagnostics.Diagnostic) {
	l.checkBOM()

	for !l.atEnd() {
		l.skipWhitespace()
		if l.atEnd() {
			break
		}
		l.markStart()
		l.scanToken()
	}

	l.markStart()
	l.emit(EOF, "")
	return l.tokens, l.errors
}

// TokenizeWithComments lexes the full source, preserving line and block
// comments as LineComment and BlockComment tokens in the stream. This is
// used by the formatter to round-trip comments through formatting.
func (l *Lexer) TokenizeWithComments() ([]Token, []diagnostics.Diagnostic) {
	l.keepComments = true
	return l.Tokenize()
}

// --- character access ---

func (l *Lexer) atEnd() bool {
	return l.pos >= len(l.src)
}

// peek returns the current byte without advancing, or 0 at end.
func (l *Lexer) peek() byte {
	if l.pos >= len(l.src) {
		return 0
	}
	return l.src[l.pos]
}

// peekAt returns the byte at pos+offset, or 0 if out of bounds.
func (l *Lexer) peekAt(offset int) byte {
	p := l.pos + offset
	if p >= len(l.src) || p < 0 {
		return 0
	}
	return l.src[p]
}

// advance consumes one byte and returns it. Tracks line/col.
func (l *Lexer) advance() byte {
	if l.pos >= len(l.src) {
		return 0
	}
	ch := l.src[l.pos]
	l.pos++
	if ch == '\n' {
		l.line++
		l.col = 1
	} else {
		l.col++
	}
	return ch
}

// match advances if the current byte equals expected.
func (l *Lexer) match(expected byte) bool {
	if l.pos < len(l.src) && l.src[l.pos] == expected {
		l.advance()
		return true
	}
	return false
}

// --- token construction ---

func (l *Lexer) markStart() {
	l.startPos = l.pos
	l.startLine = l.line
	l.startCol = l.col
}

func (l *Lexer) currentSpan() diagnostics.Span {
	return diagnostics.Span{
		File: l.filename,
		Start: diagnostics.Pos{
			Offset: l.startPos,
			Line:   l.startLine,
			Col:    l.startCol,
		},
		End: diagnostics.Pos{
			Offset: l.pos,
			Line:   l.line,
			Col:    l.col,
		},
	}
}

func (l *Lexer) emit(kind TokenKind, literal string) {
	l.tokens = append(l.tokens, Token{
		Kind:    kind,
		Literal: literal,
		Span:    l.currentSpan(),
	})
}

func (l *Lexer) emitCurrent(kind TokenKind) {
	l.emit(kind, string(l.src[l.startPos:l.pos]))
}

func (l *Lexer) errorf(format string, args ...any) {
	l.errors = append(l.errors, diagnostics.Errorf(l.currentSpan(), format, args...))
}

// --- BOM check ---

func (l *Lexer) checkBOM() {
	if len(l.src) >= 3 && l.src[0] == 0xEF && l.src[1] == 0xBB && l.src[2] == 0xBF {
		l.markStart()
		l.pos = 3
		l.col = 4
		l.errorf("BOM is not permitted in Fuse source files")
	}
}

// --- whitespace ---

func (l *Lexer) skipWhitespace() {
	for !l.atEnd() {
		ch := l.peek()
		switch ch {
		case ' ', '\t':
			l.advance()
		case '\r':
			l.advance()
			// normalize CRLF: the \n is consumed next iteration
		case '\n':
			l.advance()
		default:
			return
		}
	}
}

// --- scanner dispatch ---

func (l *Lexer) scanToken() {
	ch := l.peek()

	// line comment
	if ch == '/' && l.peekAt(1) == '/' {
		l.scanLineComment()
		return
	}
	// block comment
	if ch == '/' && l.peekAt(1) == '*' {
		l.scanBlockComment()
		return
	}

	// string literal
	if ch == '"' {
		l.scanString()
		return
	}

	// raw string literal or identifier starting with 'r'
	if ch == 'r' && l.isRawStringStart() {
		l.scanRawString()
		return
	}

	// number literal
	if isDigit(ch) {
		l.scanNumber()
		return
	}

	// identifier or keyword
	if isIdentStart(ch) {
		l.scanIdent()
		return
	}

	// operators and punctuation
	l.scanPunct()
}

// --- comments ---

func (l *Lexer) scanLineComment() {
	l.advance() // /
	l.advance() // /
	for !l.atEnd() && l.peek() != '\n' {
		l.advance()
	}
	if l.keepComments {
		l.emitCurrent(LineComment)
	}
	// the \n will be consumed by skipWhitespace
}

func (l *Lexer) scanBlockComment() {
	l.advance() // /
	l.advance() // *
	depth := 1
	for !l.atEnd() && depth > 0 {
		if l.peek() == '/' && l.peekAt(1) == '*' {
			l.advance()
			l.advance()
			depth++
		} else if l.peek() == '*' && l.peekAt(1) == '/' {
			l.advance()
			l.advance()
			depth--
		} else {
			l.advance()
		}
	}
	if depth > 0 {
		l.errorf("unterminated block comment")
	} else if l.keepComments {
		l.emitCurrent(BlockComment)
	}
}

// --- identifiers and keywords ---

func (l *Lexer) scanIdent() {
	for !l.atEnd() {
		ch := l.peek()
		if ch < 0x80 {
			if !isIdentContinue(ch) {
				break
			}
			l.advance()
		} else {
			// multi-byte UTF-8 rune
			r, size := utf8.DecodeRune(l.src[l.pos:])
			if !isIdentContinueRune(r) {
				break
			}
			l.pos += size
			l.col += size
		}
	}
	text := string(l.src[l.startPos:l.pos])
	kind := LookupIdent(text)
	l.emit(kind, text)
}

// --- numbers ---

func (l *Lexer) scanNumber() {
	ch := l.peek()

	if ch == '0' {
		next := l.peekAt(1)
		switch next {
		case 'x', 'X':
			l.advance() // 0
			l.advance() // x
			if !isHexDigit(l.peek()) {
				l.errorf("expected hexadecimal digit after 0x")
				l.emitCurrent(Illegal)
				return
			}
			for !l.atEnd() && isHexDigit(l.peek()) {
				l.advance()
			}
			l.consumeIntSuffix()
			l.emitCurrent(IntLit)
			return
		case 'o', 'O':
			l.advance() // 0
			l.advance() // o
			if !isOctDigit(l.peek()) {
				l.errorf("expected octal digit after 0o")
				l.emitCurrent(Illegal)
				return
			}
			for !l.atEnd() && isOctDigit(l.peek()) {
				l.advance()
			}
			l.consumeIntSuffix()
			l.emitCurrent(IntLit)
			return
		case 'b', 'B':
			l.advance() // 0
			l.advance() // b
			if !isBinDigit(l.peek()) {
				l.errorf("expected binary digit after 0b")
				l.emitCurrent(Illegal)
				return
			}
			for !l.atEnd() && isBinDigit(l.peek()) {
				l.advance()
			}
			l.consumeIntSuffix()
			l.emitCurrent(IntLit)
			return
		}
	}

	// decimal integer or float
	for !l.atEnd() && isDigit(l.peek()) {
		l.advance()
	}

	isFloat := false

	// check for decimal point: '.' followed by digit (not '..' or '.method')
	if l.peek() == '.' && isDigit(l.peekAt(1)) {
		isFloat = true
		l.advance() // .
		for !l.atEnd() && isDigit(l.peek()) {
			l.advance()
		}
	}

	// check for exponent
	if l.peek() == 'e' || l.peek() == 'E' {
		isFloat = true
		l.advance() // e
		if l.peek() == '+' || l.peek() == '-' {
			l.advance()
		}
		if !isDigit(l.peek()) {
			l.errorf("expected digit in exponent")
			l.emitCurrent(Illegal)
			return
		}
		for !l.atEnd() && isDigit(l.peek()) {
			l.advance()
		}
	}

	if isFloat {
		l.consumeFloatSuffix()
		l.emitCurrent(FloatLit)
	} else {
		l.consumeIntSuffix()
		l.emitCurrent(IntLit)
	}
}

// consumeIntSuffix tries to consume a valid integer type suffix.
func (l *Lexer) consumeIntSuffix() {
	l.trySuffix(intSuffixes)
}

// consumeFloatSuffix tries to consume a valid float type suffix.
func (l *Lexer) consumeFloatSuffix() {
	l.trySuffix(floatSuffixes)
}

var intSuffixes = []string{
	"i128", "isize", "i64", "i32", "i16", "i8",
	"u128", "usize", "u64", "u32", "u16", "u8",
}

var floatSuffixes = []string{
	"f64", "f32",
}

// trySuffix checks if the remaining source at current position starts with
// one of the given suffixes, and that the character after the suffix is not
// an identifier-continue character. If so, it consumes the suffix.
func (l *Lexer) trySuffix(suffixes []string) {
	for _, s := range suffixes {
		end := l.pos + len(s)
		if end > len(l.src) {
			continue
		}
		if string(l.src[l.pos:end]) == s {
			// suffix must not be followed by identifier-continue
			if end < len(l.src) && isIdentContinue(l.src[end]) {
				continue
			}
			l.pos = end
			l.col += len(s)
			return
		}
	}
}

// --- strings ---

func (l *Lexer) scanString() {
	l.advance() // opening "
	for !l.atEnd() {
		ch := l.peek()
		if ch == '"' {
			l.advance()
			l.emitCurrent(StringLit)
			return
		}
		if ch == '\\' {
			l.advance() // backslash
			if l.atEnd() {
				l.errorf("unterminated string literal")
				l.emitCurrent(Illegal)
				return
			}
			esc := l.peek()
			switch esc {
			case 'n', 'r', 't', '\\', '"', '0':
				l.advance()
			case 'u':
				l.advance() // u
				if !l.match('{') {
					l.errorf("expected '{' in unicode escape")
					continue
				}
				for !l.atEnd() && l.peek() != '}' {
					if !isHexDigit(l.peek()) {
						l.errorf("expected hex digit in unicode escape")
						break
					}
					l.advance()
				}
				if !l.match('}') {
					l.errorf("expected '}' in unicode escape")
				}
			default:
				l.errorf("unknown escape sequence '\\%c'", esc)
				l.advance()
			}
			continue
		}
		if ch == '\n' {
			l.errorf("unterminated string literal")
			l.emitCurrent(Illegal)
			return
		}
		l.advance()
	}
	l.errorf("unterminated string literal")
	l.emitCurrent(Illegal)
}

// --- raw strings ---

// isRawStringStart checks whether the source at current position (which is 'r')
// begins a valid raw string prefix: r followed by zero or more # then ".
// Per implementation contract: r#abc is NOT a raw string start.
func (l *Lexer) isRawStringStart() bool {
	p := l.pos + 1 // skip the 'r'
	for p < len(l.src) && l.src[p] == '#' {
		p++
	}
	return p < len(l.src) && l.src[p] == '"'
}

func (l *Lexer) scanRawString() {
	l.advance() // r

	hashes := 0
	for l.peek() == '#' {
		l.advance()
		hashes++
	}
	l.advance() // opening "

	for !l.atEnd() {
		if l.peek() == '"' {
			l.advance() // closing "
			// count closing hashes
			matched := 0
			for matched < hashes && l.peek() == '#' {
				l.advance()
				matched++
			}
			if matched == hashes {
				l.emitCurrent(RawStringLit)
				return
			}
			// not enough hashes — this " is part of the string body
		} else {
			l.advance()
		}
	}
	l.errorf("unterminated raw string literal")
	l.emitCurrent(Illegal)
}

// --- punctuation and operators ---

func (l *Lexer) scanPunct() {
	ch := l.advance()

	switch ch {
	case '(':
		l.emitCurrent(LParen)
	case ')':
		l.emitCurrent(RParen)
	case '[':
		l.emitCurrent(LBrack)
	case ']':
		l.emitCurrent(RBrack)
	case '{':
		l.emitCurrent(LBrace)
	case '}':
		l.emitCurrent(RBrace)
	case ',':
		l.emitCurrent(Comma)
	case ';':
		l.emitCurrent(Semi)
	case '~':
		l.emitCurrent(Tilde)
	case '@':
		l.emitCurrent(At)
	case '#':
		l.emitCurrent(Hash)

	case ':':
		if l.match(':') {
			l.emitCurrent(ColonColon)
		} else {
			l.emitCurrent(Colon)
		}

	case '.':
		l.emitCurrent(Dot)

	case '?':
		if l.match('.') {
			l.emitCurrent(QDot)
		} else {
			l.emitCurrent(Question)
		}

	case '+':
		if l.match('=') {
			l.emitCurrent(PlusEq)
		} else {
			l.emitCurrent(Plus)
		}
	case '-':
		if l.match('>') {
			l.emitCurrent(Arrow)
		} else if l.match('=') {
			l.emitCurrent(MinusEq)
		} else {
			l.emitCurrent(Minus)
		}
	case '*':
		if l.match('=') {
			l.emitCurrent(StarEq)
		} else {
			l.emitCurrent(Star)
		}
	case '/':
		if l.match('=') {
			l.emitCurrent(SlashEq)
		} else {
			l.emitCurrent(Slash)
		}
	case '%':
		if l.match('=') {
			l.emitCurrent(PercentEq)
		} else {
			l.emitCurrent(Percent)
		}

	case '&':
		if l.match('&') {
			l.emitCurrent(AmpAmp)
		} else if l.match('=') {
			l.emitCurrent(AmpEq)
		} else {
			l.emitCurrent(Amp)
		}
	case '|':
		if l.match('|') {
			l.emitCurrent(PipePipe)
		} else if l.match('=') {
			l.emitCurrent(PipeEq)
		} else {
			l.emitCurrent(Pipe)
		}
	case '^':
		if l.match('=') {
			l.emitCurrent(CaretEq)
		} else {
			l.emitCurrent(Caret)
		}

	case '!':
		if l.match('=') {
			l.emitCurrent(BangEq)
		} else {
			l.emitCurrent(Bang)
		}

	case '=':
		if l.match('=') {
			l.emitCurrent(EqEq)
		} else if l.match('>') {
			l.emitCurrent(FatArrow)
		} else {
			l.emitCurrent(Eq)
		}

	case '<':
		if l.match('<') {
			if l.match('=') {
				l.emitCurrent(ShlEq)
			} else {
				l.emitCurrent(Shl)
			}
		} else if l.match('=') {
			l.emitCurrent(LtEq)
		} else {
			l.emitCurrent(Lt)
		}
	case '>':
		if l.match('>') {
			if l.match('=') {
				l.emitCurrent(ShrEq)
			} else {
				l.emitCurrent(Shr)
			}
		} else if l.match('=') {
			l.emitCurrent(GtEq)
		} else {
			l.emitCurrent(Gt)
		}

	default:
		l.errorf("unexpected character %q", ch)
		l.emitCurrent(Illegal)
	}
}

// --- character predicates ---

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isHexDigit(ch byte) bool {
	return (ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f') || (ch >= 'A' && ch <= 'F')
}

func isOctDigit(ch byte) bool {
	return ch >= '0' && ch <= '7'
}

func isBinDigit(ch byte) bool {
	return ch == '0' || ch == '1'
}

func isIdentStart(ch byte) bool {
	if ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
		return true
	}
	// Handle multi-byte Unicode letters: decode the rune.
	if ch >= 0x80 {
		return true // let isIdentStartRune check properly at call sites that need it
	}
	return false
}

func isIdentContinue(ch byte) bool {
	if ch == '_' || (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') {
		return true
	}
	if ch >= 0x80 {
		return true
	}
	return false
}

// isIdentStartRune checks the Unicode property for identifier start.
func isIdentStartRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r)
}

// isIdentContinueRune checks the Unicode property for identifier continuation.
func isIdentContinueRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}

