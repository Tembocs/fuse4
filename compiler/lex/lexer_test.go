package lex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- helpers ---

func tokenize(t *testing.T, src string) []Token {
	t.Helper()
	l := New("test.fuse", []byte(src))
	tokens, errs := l.Tokenize()
	for _, e := range errs {
		t.Errorf("unexpected diagnostic: %s", e)
	}
	return tokens
}

func tokenizeWithErrors(src string) ([]Token, int) {
	l := New("test.fuse", []byte(src))
	tokens, errs := l.Tokenize()
	return tokens, len(errs)
}

func expectTokens(t *testing.T, src string, expected []TokenKind) {
	t.Helper()
	tokens := tokenize(t, src)
	// strip trailing EOF for comparison
	got := make([]TokenKind, 0, len(tokens))
	for _, tok := range tokens {
		if tok.Kind == EOF {
			continue
		}
		got = append(got, tok.Kind)
	}
	if len(got) != len(expected) {
		t.Fatalf("token count mismatch: got %d, want %d\ngot:  %v\nwant: %v", len(got), len(expected), got, expected)
	}
	for i := range got {
		if got[i] != expected[i] {
			t.Errorf("token[%d]: got %s, want %s", i, got[i], expected[i])
		}
	}
}

func expectSingleToken(t *testing.T, src string, kind TokenKind, literal string) {
	t.Helper()
	tokens := tokenize(t, src)
	if len(tokens) != 2 { // token + EOF
		t.Fatalf("expected 2 tokens (token+EOF), got %d: %v", len(tokens), tokens)
	}
	if tokens[0].Kind != kind {
		t.Errorf("kind: got %s, want %s", tokens[0].Kind, kind)
	}
	if tokens[0].Literal != literal {
		t.Errorf("literal: got %q, want %q", tokens[0].Literal, literal)
	}
}

// ===== Wave 01 Phase 01: Token Kinds =====

func TestEOFOnEmptyInput(t *testing.T) {
	tokens := tokenize(t, "")
	if len(tokens) != 1 || tokens[0].Kind != EOF {
		t.Fatalf("expected single EOF token, got %v", tokens)
	}
}

func TestEOFOnWhitespace(t *testing.T) {
	tokens := tokenize(t, "   \t\n\r\n  ")
	if len(tokens) != 1 || tokens[0].Kind != EOF {
		t.Fatalf("expected single EOF token, got %v", tokens)
	}
}

// ===== Wave 01 Phase 02: Identifiers and Keywords =====

func TestIdentifiers(t *testing.T) {
	cases := []struct {
		src  string
		kind TokenKind
	}{
		{"foo", Ident},
		{"_bar", Ident},
		{"_123", Ident},
		{"camelCase", Ident},
		{"UPPER", Ident},
		{"a", Ident},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			expectSingleToken(t, tc.src, tc.kind, tc.src)
		})
	}
}

func TestAllKeywords(t *testing.T) {
	cases := []struct {
		src  string
		kind TokenKind
	}{
		{"fn", KwFn},
		{"pub", KwPub},
		{"struct", KwStruct},
		{"enum", KwEnum},
		{"trait", KwTrait},
		{"impl", KwImpl},
		{"for", KwFor},
		{"in", KwIn},
		{"while", KwWhile},
		{"loop", KwLoop},
		{"if", KwIf},
		{"else", KwElse},
		{"match", KwMatch},
		{"return", KwReturn},
		{"let", KwLet},
		{"var", KwVar},
		{"move", KwMove},
		{"ref", KwRef},
		{"mutref", KwMutref},
		{"owned", KwOwned},
		{"unsafe", KwUnsafe},
		{"spawn", KwSpawn},
		{"chan", KwChan},
		{"import", KwImport},
		{"as", KwAs},
		{"mod", KwMod},
		{"use", KwUse},
		{"type", KwType},
		{"const", KwConst},
		{"static", KwStatic},
		{"extern", KwExtern},
		{"break", KwBreak},
		{"continue", KwContinue},
		{"where", KwWhere},
		{"Self", KwSelfType},
		{"self", KwSelfValue},
		{"true", KwTrue},
		{"false", KwFalse},
		{"None", KwNone},
		{"Some", KwSome},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			expectSingleToken(t, tc.src, tc.kind, tc.src)
		})
	}
}

func TestKeywordPrefixIsIdent(t *testing.T) {
	// "fns" is not a keyword, it's an identifier
	expectSingleToken(t, "fns", Ident, "fns")
	expectSingleToken(t, "letter", Ident, "letter")
	expectSingleToken(t, "import_path", Ident, "import_path")
}

// ===== Wave 01 Phase 02: Integer Literals =====

func TestDecimalIntegers(t *testing.T) {
	cases := []string{"0", "1", "42", "9001", "123456789"}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			expectSingleToken(t, src, IntLit, src)
		})
	}
}

func TestHexIntegers(t *testing.T) {
	cases := []string{"0x0", "0x2A", "0xff", "0xDEAD"}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			expectSingleToken(t, src, IntLit, src)
		})
	}
}

func TestOctalIntegers(t *testing.T) {
	cases := []string{"0o0", "0o52", "0o777"}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			expectSingleToken(t, src, IntLit, src)
		})
	}
}

func TestBinaryIntegers(t *testing.T) {
	cases := []string{"0b0", "0b1", "0b101010", "0b1111"}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			expectSingleToken(t, src, IntLit, src)
		})
	}
}

func TestIntegerSuffixes(t *testing.T) {
	cases := []string{
		"64usize", "0xffu8", "0b1010i32",
		"42i8", "42i16", "42i32", "42i64", "42i128", "42isize",
		"42u8", "42u16", "42u32", "42u64", "42u128", "42usize",
	}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			expectSingleToken(t, src, IntLit, src)
		})
	}
}

func TestInvalidHexLiteral(t *testing.T) {
	_, errs := tokenizeWithErrors("0x")
	if errs == 0 {
		t.Error("expected error for 0x without digits")
	}
}

func TestInvalidOctalLiteral(t *testing.T) {
	_, errs := tokenizeWithErrors("0o")
	if errs == 0 {
		t.Error("expected error for 0o without digits")
	}
}

func TestInvalidBinaryLiteral(t *testing.T) {
	_, errs := tokenizeWithErrors("0b")
	if errs == 0 {
		t.Error("expected error for 0b without digits")
	}
}

// ===== Wave 01 Phase 02: Float Literals =====

func TestFloatLiterals(t *testing.T) {
	cases := []string{"1.0", "3.14", "0.5", "100.001"}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			expectSingleToken(t, src, FloatLit, src)
		})
	}
}

func TestFloatExponents(t *testing.T) {
	cases := []string{"6.02e23", "1e10", "1E10", "1e+10", "1e-10"}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			expectSingleToken(t, src, FloatLit, src)
		})
	}
}

func TestFloatSuffixes(t *testing.T) {
	expectSingleToken(t, "3.14f32", FloatLit, "3.14f32")
	expectSingleToken(t, "1.0f64", FloatLit, "1.0f64")
}

func TestInvalidExponent(t *testing.T) {
	_, errs := tokenizeWithErrors("1e")
	if errs == 0 {
		t.Error("expected error for exponent without digits")
	}
}

// ===== Wave 01 Phase 02: String Literals =====

func TestSimpleStrings(t *testing.T) {
	expectSingleToken(t, `"hello"`, StringLit, `"hello"`)
	expectSingleToken(t, `""`, StringLit, `""`)
	expectSingleToken(t, `"a b c"`, StringLit, `"a b c"`)
}

func TestStringEscapes(t *testing.T) {
	expectSingleToken(t, `"hello\nworld"`, StringLit, `"hello\nworld"`)
	expectSingleToken(t, `"tab\there"`, StringLit, `"tab\there"`)
	expectSingleToken(t, `"cr\rhere"`, StringLit, `"cr\rhere"`)
	expectSingleToken(t, `"slash\\"`, StringLit, `"slash\\"`)
	expectSingleToken(t, `"quote\""`, StringLit, `"quote\""`)
}

func TestUnterminatedString(t *testing.T) {
	_, errs := tokenizeWithErrors(`"unterminated`)
	if errs == 0 {
		t.Error("expected error for unterminated string")
	}
}

func TestStringWithNewline(t *testing.T) {
	_, errs := tokenizeWithErrors("\"line1\nline2\"")
	if errs == 0 {
		t.Error("expected error for newline in string")
	}
}

// ===== Wave 01 Phase 02: Comments =====

func TestLineComment(t *testing.T) {
	tokens := tokenize(t, "foo // this is a comment\nbar")
	expectTokens(t, "foo // this is a comment\nbar", []TokenKind{Ident, Ident})
	if tokens[0].Literal != "foo" || tokens[1].Literal != "bar" {
		t.Errorf("wrong literals: %q, %q", tokens[0].Literal, tokens[1].Literal)
	}
}

func TestBlockComment(t *testing.T) {
	expectTokens(t, "foo /* comment */ bar", []TokenKind{Ident, Ident})
}

func TestNestedBlockComment(t *testing.T) {
	expectTokens(t, "a /* outer /* inner */ still outer */ b", []TokenKind{Ident, Ident})
}

func TestDeeplyNestedBlockComment(t *testing.T) {
	expectTokens(t, "x /* 1 /* 2 /* 3 */ 2 */ 1 */ y", []TokenKind{Ident, Ident})
}

func TestUnterminatedBlockComment(t *testing.T) {
	_, errs := tokenizeWithErrors("/* unterminated")
	if errs == 0 {
		t.Error("expected error for unterminated block comment")
	}
}

// ===== Wave 01 Phase 02: All Operators and Punctuation =====

func TestSingleCharPunctuation(t *testing.T) {
	cases := []struct {
		src  string
		kind TokenKind
	}{
		{"(", LParen}, {")", RParen},
		{"[", LBrack}, {"]", RBrack},
		{"{", LBrace}, {"}", RBrace},
		{",", Comma}, {";", Semi},
		{"~", Tilde}, {"@", At}, {"#", Hash},
		{".", Dot},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			expectSingleToken(t, tc.src, tc.kind, tc.src)
		})
	}
}

func TestMultiCharOperators(t *testing.T) {
	cases := []struct {
		src  string
		kind TokenKind
	}{
		{"+", Plus}, {"-", Minus}, {"*", Star}, {"/", Slash}, {"%", Percent},
		{"&", Amp}, {"|", Pipe}, {"^", Caret},
		{"!", Bang},
		{"<<", Shl}, {">>", Shr},
		{"&&", AmpAmp}, {"||", PipePipe},
		{"==", EqEq}, {"!=", BangEq},
		{"<", Lt}, {">", Gt}, {"<=", LtEq}, {">=", GtEq},
		{"=", Eq},
		{"+=", PlusEq}, {"-=", MinusEq}, {"*=", StarEq}, {"/=", SlashEq}, {"%=", PercentEq},
		{"&=", AmpEq}, {"|=", PipeEq}, {"^=", CaretEq},
		{"<<=", ShlEq}, {">>=", ShrEq},
		{"->", Arrow}, {"=>", FatArrow},
		{"::", ColonColon}, {":", Colon},
		{"?", Question}, {"?.", QDot},
	}
	for _, tc := range cases {
		t.Run(tc.src, func(t *testing.T) {
			expectSingleToken(t, tc.src, tc.kind, tc.src)
		})
	}
}

// ===== Wave 01 Phase 03: Raw String Edge Cases =====

func TestRawStringSimple(t *testing.T) {
	expectSingleToken(t, `r"hello"`, RawStringLit, `r"hello"`)
}

func TestRawStringWithHashes(t *testing.T) {
	expectSingleToken(t, `r#"hello"#`, RawStringLit, `r#"hello"#`)
	expectSingleToken(t, `r##"hello"##`, RawStringLit, `r##"hello"##`)
}

func TestRawStringContainingQuote(t *testing.T) {
	expectSingleToken(t, `r#"say "hi""#`, RawStringLit, `r#"say "hi""#`)
}

func TestRawStringContainingHash(t *testing.T) {
	expectSingleToken(t, `r##"has a #"# inside"##`, RawStringLit, `r##"has a #"# inside"##`)
}

// Implementation contract: r#abc must NOT enter raw-string mode.
// It must tokenize as: IDENT("r") HASH("#") IDENT("abc")
func TestRHashIdentIsNotRawString(t *testing.T) {
	tokens := tokenize(t, "r#abc")
	kinds := make([]TokenKind, 0)
	for _, tok := range tokens {
		if tok.Kind == EOF {
			continue
		}
		kinds = append(kinds, tok.Kind)
	}
	expected := []TokenKind{Ident, Hash, Ident}
	if len(kinds) != len(expected) {
		t.Fatalf("r#abc: got %d tokens %v, want %d tokens %v", len(kinds), kinds, len(expected), expected)
	}
	for i := range kinds {
		if kinds[i] != expected[i] {
			t.Errorf("r#abc token[%d]: got %s, want %s", i, kinds[i], expected[i])
		}
	}
	if tokens[0].Literal != "r" {
		t.Errorf("first token literal: got %q, want %q", tokens[0].Literal, "r")
	}
}

func TestUnterminatedRawString(t *testing.T) {
	_, errs := tokenizeWithErrors(`r"unterminated`)
	if errs == 0 {
		t.Error("expected error for unterminated raw string")
	}
}

// ===== Wave 01 Phase 03: ?. Longest-Match =====

// Implementation contract: ?. is emitted as one token, not ? followed by .
func TestQDotIsSingleToken(t *testing.T) {
	tokens := tokenize(t, "x?.y")
	kinds := make([]TokenKind, 0)
	for _, tok := range tokens {
		if tok.Kind == EOF {
			continue
		}
		kinds = append(kinds, tok.Kind)
	}
	expected := []TokenKind{Ident, QDot, Ident}
	if len(kinds) != len(expected) {
		t.Fatalf("x?.y: got %v, want %v", kinds, expected)
	}
	for i := range kinds {
		if kinds[i] != expected[i] {
			t.Errorf("x?.y token[%d]: got %s, want %s", i, kinds[i], expected[i])
		}
	}
}

func TestQuestionAloneBeforeNonDot(t *testing.T) {
	tokens := tokenize(t, "x?;")
	kinds := make([]TokenKind, 0)
	for _, tok := range tokens {
		if tok.Kind == EOF {
			continue
		}
		kinds = append(kinds, tok.Kind)
	}
	expected := []TokenKind{Ident, Question, Semi}
	if len(kinds) != len(expected) {
		t.Fatalf("x?; got %v, want %v", kinds, expected)
	}
}

// ===== Wave 01 Phase 03: BOM Rejection =====

func TestBOMIsRejected(t *testing.T) {
	bom := []byte{0xEF, 0xBB, 0xBF}
	src := append(bom, []byte("fn main() {}")...)
	l := New("test.fuse", src)
	_, errs := l.Tokenize()
	if len(errs) == 0 {
		t.Error("expected BOM diagnostic")
	}
	found := false
	for _, e := range errs {
		if strings.Contains(e.Message, "BOM") {
			found = true
		}
	}
	if !found {
		t.Error("expected BOM-specific diagnostic message")
	}
}

func TestBOMStillTokenizesRest(t *testing.T) {
	bom := []byte{0xEF, 0xBB, 0xBF}
	src := append(bom, []byte("fn")...)
	l := New("test.fuse", src)
	tokens, _ := l.Tokenize()
	// Should still produce fn + EOF even with BOM error
	foundFn := false
	for _, tok := range tokens {
		if tok.Kind == KwFn {
			foundFn = true
		}
	}
	if !foundFn {
		t.Error("expected fn keyword after BOM")
	}
}

// ===== Wave 01 Phase 03: Span Correctness =====

func TestSpanPositions(t *testing.T) {
	tokens := tokenize(t, "fn main")
	// fn is at 1:1, main is at 1:4
	if tokens[0].Span.Start.Line != 1 || tokens[0].Span.Start.Col != 1 {
		t.Errorf("fn span start: got %d:%d, want 1:1", tokens[0].Span.Start.Line, tokens[0].Span.Start.Col)
	}
	if tokens[1].Span.Start.Line != 1 || tokens[1].Span.Start.Col != 4 {
		t.Errorf("main span start: got %d:%d, want 1:4", tokens[1].Span.Start.Line, tokens[1].Span.Start.Col)
	}
}

func TestMultiLineSpans(t *testing.T) {
	tokens := tokenize(t, "a\nb\nc")
	if tokens[0].Span.Start.Line != 1 {
		t.Errorf("a: expected line 1, got %d", tokens[0].Span.Start.Line)
	}
	if tokens[1].Span.Start.Line != 2 {
		t.Errorf("b: expected line 2, got %d", tokens[1].Span.Start.Line)
	}
	if tokens[2].Span.Start.Line != 3 {
		t.Errorf("c: expected line 3, got %d", tokens[2].Span.Start.Line)
	}
}

// ===== Wave 01 Phase 03: CRLF Normalization =====

func TestCRLFNormalization(t *testing.T) {
	tokens := tokenize(t, "a\r\nb")
	kinds := make([]TokenKind, 0)
	for _, tok := range tokens {
		if tok.Kind == EOF {
			continue
		}
		kinds = append(kinds, tok.Kind)
	}
	if len(kinds) != 2 {
		t.Fatalf("expected 2 tokens, got %d: %v", len(kinds), kinds)
	}
	if tokens[1].Span.Start.Line != 2 {
		t.Errorf("b after CRLF: expected line 2, got %d", tokens[1].Span.Start.Line)
	}
}

// ===== Wave 01 Phase 03: Compound expressions =====

func TestFunctionSignature(t *testing.T) {
	src := `pub fn process(queue: mutref Chan[WorkItem]) -> Result[(), String]`
	tokens := tokenize(t, src)
	// Just verify it tokenizes without errors and has reasonable tokens
	if len(tokens) < 10 {
		t.Errorf("expected many tokens for function signature, got %d", len(tokens))
	}
}

func TestStructDecl(t *testing.T) {
	src := "@value struct WorkItem {\n\tid: U64,\n\tpayload: String,\n}"
	tokens := tokenize(t, src)
	if len(tokens) < 8 {
		t.Errorf("expected many tokens for struct decl, got %d", len(tokens))
	}
	if tokens[0].Kind != At {
		t.Errorf("expected @ first, got %s", tokens[0].Kind)
	}
	if tokens[1].Kind != Ident || tokens[1].Literal != "value" {
		t.Errorf("expected ident 'value', got %s %q", tokens[1].Kind, tokens[1].Literal)
	}
	if tokens[2].Kind != KwStruct {
		t.Errorf("expected struct keyword, got %s", tokens[2].Kind)
	}
}

func TestOptionalChainExpression(t *testing.T) {
	src := "result?.value?.name"
	tokens := tokenize(t, src)
	expected := []TokenKind{Ident, QDot, Ident, QDot, Ident}
	kinds := make([]TokenKind, 0)
	for _, tok := range tokens {
		if tok.Kind == EOF {
			continue
		}
		kinds = append(kinds, tok.Kind)
	}
	if len(kinds) != len(expected) {
		t.Fatalf("got %v, want %v", kinds, expected)
	}
	for i := range kinds {
		if kinds[i] != expected[i] {
			t.Errorf("token[%d]: got %s, want %s", i, kinds[i], expected[i])
		}
	}
}

func TestErrorPropagation(t *testing.T) {
	src := "queue.recv()?"
	tokens := tokenize(t, src)
	// last non-EOF should be ?
	lastNonEOF := tokens[len(tokens)-2]
	if lastNonEOF.Kind != Question {
		t.Errorf("expected ? at end, got %s", lastNonEOF.Kind)
	}
}

// ===== Wave 01 Phase 03: Illegal Character =====

func TestIllegalCharacter(t *testing.T) {
	_, errs := tokenizeWithErrors("$")
	if errs == 0 {
		t.Error("expected error for illegal character $")
	}
}

// ===== Golden Test Harness =====

func formatTokens(tokens []Token) string {
	var b strings.Builder
	for _, tok := range tokens {
		s := tok.Span.Start
		b.WriteString(strings.Replace(
			strings.Join([]string{
				padInt(s.Line), ":", padInt(s.Col), " ",
				tok.Kind.String(), " ",
				quote(tok.Literal),
			}, ""),
			"\n", "\\n", -1,
		))
		b.WriteByte('\n')
	}
	return b.String()
}

func padInt(n int) string {
	s := strings.Builder{}
	if n < 10 {
		s.WriteByte(' ')
	}
	s.WriteString(intToStr(n))
	return s.String()
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, c := range s {
		switch c {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			b.WriteRune(c)
		}
	}
	b.WriteByte('"')
	return b.String()
}

func TestGoldenLexer(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "tests", "fixtures", "lexer")
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatalf("cannot read fixture dir: %v", err)
	}

	update := os.Getenv("UPDATE_GOLDENS") != ""

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".fuse") {
			continue
		}
		base := strings.TrimSuffix(name, ".fuse")
		t.Run(base, func(t *testing.T) {
			srcPath := filepath.Join(fixtureDir, name)
			goldenPath := filepath.Join(fixtureDir, base+".golden")

			src, err := os.ReadFile(srcPath)
			if err != nil {
				t.Fatalf("read source: %v", err)
			}

			l := New(name, src)
			tokens, errs := l.Tokenize()
			_ = errs // golden captures tokens only; errors are tested separately

			got := formatTokens(tokens)

			if update {
				if err := os.WriteFile(goldenPath, []byte(got), 0644); err != nil {
					t.Fatalf("update golden: %v", err)
				}
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden (run with UPDATE_GOLDENS=1 to generate): %v", err)
			}
			if got != string(want) {
				t.Errorf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, string(want))
			}
		})
	}
}
