// Package ast owns the syntax-only AST. AST nodes do not contain
// resolved symbols, inferred types, liveness, or backend metadata.
package ast

import (
	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/lex"
)

// Node is the common interface for every AST node.
type Node interface {
	NodeSpan() diagnostics.Span
}

// Item is a top-level or block-level declaration.
type Item interface {
	Node
	itemNode()
}

// Expr is an expression node.
type Expr interface {
	Node
	exprNode()
}

// Stmt is a statement node.
type Stmt interface {
	Node
	stmtNode()
}

// TypeExpr is a type expression node.
type TypeExpr interface {
	Node
	typeExprNode()
}

// Pattern is a destructuring pattern in match arms.
type Pattern interface {
	Node
	patternNode()
}

// ---------- File ----------

// File is the root AST node for a source file.
type File struct {
	Span  diagnostics.Span
	Items []Item
}

func (n *File) NodeSpan() diagnostics.Span { return n.Span }

// ---------- Shared supporting types ----------

// Field is a named field in a struct or struct-like enum variant.
type Field struct {
	Span diagnostics.Span
	Name string
	Type TypeExpr
}

// Param is a function or method parameter.
type Param struct {
	Span      diagnostics.Span
	Ownership lex.TokenKind // 0 = value, KwRef, KwMutref, KwOwned
	Name      string
	Type      TypeExpr
}

// GenericParam is a generic type parameter with optional trait bounds.
type GenericParam struct {
	Span   diagnostics.Span
	Name   string
	Bounds []TypeExpr
}

// WhereClause is an optional where clause on a declaration.
type WhereClause struct {
	Span        diagnostics.Span
	Constraints []WhereConstraint
}

// WhereConstraint binds a type to one or more trait bounds.
type WhereConstraint struct {
	Span   diagnostics.Span
	Type   TypeExpr
	Bounds []TypeExpr
}

// Decorator is a @ annotation such as @value or @rank(N).
type Decorator struct {
	Span diagnostics.Span
	Name string
	Args []Expr // may be nil
}

// MatchArm is a single arm in a match expression.
type MatchArm struct {
	Span    diagnostics.Span
	Pattern Pattern
	Guard   Expr // optional if-guard, may be nil
	Body    Expr
}

// FieldInit is a field initializer in a struct literal.
type FieldInit struct {
	Span  diagnostics.Span
	Name  string
	Value Expr
}

// Variant is an enum variant declaration.
type Variant struct {
	Span   diagnostics.Span
	Name   string
	Kind   VariantKind
	Types  []TypeExpr // tuple variant payload types
	Fields []Field    // struct variant fields
}

// VariantKind classifies an enum variant.
type VariantKind int

const (
	VariantUnit   VariantKind = iota // Name
	VariantTuple                     // Name(T1, T2)
	VariantStruct                    // Name { a: T1, b: T2 }
)
