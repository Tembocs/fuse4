package lex

import (
	"fmt"

	"github.com/Tembocs/fuse4/compiler/diagnostics"
)

// TokenKind classifies a lexical token.
type TokenKind int

const (
	// Special tokens
	EOF     TokenKind = iota
	Illegal           // unrecognized character or malformed token

	// Literals
	Ident        // identifier
	IntLit       // integer literal (may include suffix)
	FloatLit     // float literal (may include suffix)
	StringLit    // "..." string literal
	RawStringLit // r"..." or r#"..."# raw string literal

	// ---- Keywords ----

	KwFn
	KwPub
	KwStruct
	KwEnum
	KwTrait
	KwImpl
	KwFor
	KwIn
	KwWhile
	KwLoop
	KwIf
	KwElse
	KwMatch
	KwReturn
	KwLet
	KwVar
	KwMove
	KwRef
	KwMutref
	KwOwned
	KwUnsafe
	KwSpawn
	KwChan
	KwImport
	KwAs
	KwMod
	KwUse
	KwType
	KwConst
	KwStatic
	KwExtern
	KwBreak
	KwContinue
	KwWhere
	KwSelfType  // Self
	KwSelfValue // self
	KwTrue
	KwFalse
	KwNone
	KwSome

	// ---- Operators ----

	Plus    // +
	Minus   // -
	Star    // *
	Slash   // /
	Percent // %

	Amp   // &
	Pipe  // |
	Caret // ^
	Tilde // ~
	Shl   // <<
	Shr   // >>

	Bang     // !
	AmpAmp   // &&
	PipePipe // ||

	EqEq   // ==
	BangEq // !=
	Lt     // <
	Gt     // >
	LtEq   // <=
	GtEq   // >=

	// ---- Assignment operators ----

	Eq        // =
	PlusEq    // +=
	MinusEq   // -=
	StarEq    // *=
	SlashEq   // /=
	PercentEq // %=
	AmpEq    // &=
	PipeEq   // |=
	CaretEq  // ^=
	ShlEq    // <<=
	ShrEq    // >>=

	// ---- Punctuation / Delimiters ----

	LParen     // (
	RParen     // )
	LBrack     // [
	RBrack     // ]
	LBrace     // {
	RBrace     // }
	Comma      // ,
	Semi       // ;
	Colon      // :
	Dot        // .
	Arrow      // ->
	FatArrow   // =>
	ColonColon // ::
	At         // @
	Hash       // #
	Question   // ?
	QDot       // ?.
)

var kindNames = [...]string{
	EOF:     "EOF",
	Illegal: "ILLEGAL",

	Ident:        "IDENT",
	IntLit:       "INT",
	FloatLit:     "FLOAT",
	StringLit:    "STRING",
	RawStringLit: "RAWSTRING",

	KwFn:        "fn",
	KwPub:       "pub",
	KwStruct:    "struct",
	KwEnum:      "enum",
	KwTrait:     "trait",
	KwImpl:      "impl",
	KwFor:       "for",
	KwIn:        "in",
	KwWhile:     "while",
	KwLoop:      "loop",
	KwIf:        "if",
	KwElse:      "else",
	KwMatch:     "match",
	KwReturn:    "return",
	KwLet:       "let",
	KwVar:       "var",
	KwMove:      "move",
	KwRef:       "ref",
	KwMutref:    "mutref",
	KwOwned:     "owned",
	KwUnsafe:    "unsafe",
	KwSpawn:     "spawn",
	KwChan:      "chan",
	KwImport:    "import",
	KwAs:        "as",
	KwMod:       "mod",
	KwUse:       "use",
	KwType:      "type",
	KwConst:     "const",
	KwStatic:    "static",
	KwExtern:    "extern",
	KwBreak:     "break",
	KwContinue:  "continue",
	KwWhere:     "where",
	KwSelfType:  "Self",
	KwSelfValue: "self",
	KwTrue:      "true",
	KwFalse:     "false",
	KwNone:      "None",
	KwSome:      "Some",

	Plus:    "+",
	Minus:   "-",
	Star:    "*",
	Slash:   "/",
	Percent: "%",

	Amp:   "&",
	Pipe:  "|",
	Caret: "^",
	Tilde: "~",
	Shl:   "<<",
	Shr:   ">>",

	Bang:     "!",
	AmpAmp:   "&&",
	PipePipe: "||",

	EqEq:   "==",
	BangEq: "!=",
	Lt:     "<",
	Gt:     ">",
	LtEq:   "<=",
	GtEq:   ">=",

	Eq:        "=",
	PlusEq:    "+=",
	MinusEq:   "-=",
	StarEq:    "*=",
	SlashEq:   "/=",
	PercentEq: "%=",
	AmpEq:    "&=",
	PipeEq:   "|=",
	CaretEq:  "^=",
	ShlEq:    "<<=",
	ShrEq:    ">>=",

	LParen:     "(",
	RParen:     ")",
	LBrack:     "[",
	RBrack:     "]",
	LBrace:     "{",
	RBrace:     "}",
	Comma:      ",",
	Semi:       ";",
	Colon:      ":",
	Dot:        ".",
	Arrow:      "->",
	FatArrow:   "=>",
	ColonColon: "::",
	At:         "@",
	Hash:       "#",
	Question:   "?",
	QDot:       "?.",
}

func (k TokenKind) String() string {
	if int(k) < len(kindNames) && kindNames[k] != "" {
		return kindNames[k]
	}
	return fmt.Sprintf("TokenKind(%d)", int(k))
}

// keywords maps keyword text to its token kind.
var keywords = map[string]TokenKind{
	"fn":       KwFn,
	"pub":      KwPub,
	"struct":   KwStruct,
	"enum":     KwEnum,
	"trait":    KwTrait,
	"impl":    KwImpl,
	"for":      KwFor,
	"in":       KwIn,
	"while":    KwWhile,
	"loop":     KwLoop,
	"if":       KwIf,
	"else":     KwElse,
	"match":    KwMatch,
	"return":   KwReturn,
	"let":      KwLet,
	"var":      KwVar,
	"move":     KwMove,
	"ref":      KwRef,
	"mutref":   KwMutref,
	"owned":    KwOwned,
	"unsafe":   KwUnsafe,
	"spawn":    KwSpawn,
	"chan":      KwChan,
	"import":   KwImport,
	"as":       KwAs,
	"mod":      KwMod,
	"use":      KwUse,
	"type":     KwType,
	"const":    KwConst,
	"static":   KwStatic,
	"extern":   KwExtern,
	"break":    KwBreak,
	"continue": KwContinue,
	"where":    KwWhere,
	"Self":     KwSelfType,
	"self":     KwSelfValue,
	"true":     KwTrue,
	"false":    KwFalse,
	"None":     KwNone,
	"Some":     KwSome,
}

// LookupIdent returns the keyword token kind for s, or Ident if s is not a keyword.
func LookupIdent(s string) TokenKind {
	if k, ok := keywords[s]; ok {
		return k
	}
	return Ident
}

// Token is a single lexical token.
type Token struct {
	Kind    TokenKind
	Literal string
	Span    diagnostics.Span
}

func (t Token) String() string {
	return fmt.Sprintf("%s %s %q", t.Span.Start, t.Kind, t.Literal)
}

// IsKeyword reports whether this token is a keyword.
func (t Token) IsKeyword() bool {
	return t.Kind >= KwFn && t.Kind <= KwSome
}
