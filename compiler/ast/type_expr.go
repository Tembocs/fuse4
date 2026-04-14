package ast

import "github.com/Tembocs/fuse4/compiler/diagnostics"

// ---------- Path type (named type, possibly generic) ----------

// PathType covers simple names (Int), qualified paths (core.list.List),
// and generic instantiations (Result[T, E]).
type PathType struct {
	Span     diagnostics.Span
	Segments []string
	TypeArgs []TypeExpr // [] if not generic
}

func (n *PathType) NodeSpan() diagnostics.Span { return n.Span }
func (n *PathType) typeExprNode()               {}

// ---------- Tuple type ----------

type TupleType struct {
	Span  diagnostics.Span
	Elems []TypeExpr
}

func (n *TupleType) NodeSpan() diagnostics.Span { return n.Span }
func (n *TupleType) typeExprNode()               {}

// ---------- Array type ----------

type ArrayType struct {
	Span diagnostics.Span
	Elem TypeExpr
	Size Expr // compile-time constant expression
}

func (n *ArrayType) NodeSpan() diagnostics.Span { return n.Span }
func (n *ArrayType) typeExprNode()               {}

// ---------- Slice type ----------

type SliceType struct {
	Span diagnostics.Span
	Elem TypeExpr
}

func (n *SliceType) NodeSpan() diagnostics.Span { return n.Span }
func (n *SliceType) typeExprNode()               {}

// ---------- Pointer type ----------

type PtrType struct {
	Span diagnostics.Span
	Elem TypeExpr
}

func (n *PtrType) NodeSpan() diagnostics.Span { return n.Span }
func (n *PtrType) typeExprNode()               {}

// ========== Patterns ==========

// ---------- Wildcard: _ ----------

type WildcardPat struct {
	Span diagnostics.Span
}

func (n *WildcardPat) NodeSpan() diagnostics.Span { return n.Span }
func (n *WildcardPat) patternNode()                {}

// ---------- Binding: name ----------

type BindPat struct {
	Span diagnostics.Span
	Name string
}

func (n *BindPat) NodeSpan() diagnostics.Span { return n.Span }
func (n *BindPat) patternNode()                {}

// ---------- Literal: 42, "hello", true ----------

type LitPat struct {
	Span  diagnostics.Span
	Value string
}

func (n *LitPat) NodeSpan() diagnostics.Span { return n.Span }
func (n *LitPat) patternNode()                {}

// ---------- Tuple: (a, b, c) ----------

type TuplePat struct {
	Span  diagnostics.Span
	Elems []Pattern
}

func (n *TuplePat) NodeSpan() diagnostics.Span { return n.Span }
func (n *TuplePat) patternNode()                {}

// ---------- Constructor: Some(x), Ok(v) ----------

type ConstructorPat struct {
	Span diagnostics.Span
	Name string
	Args []Pattern
}

func (n *ConstructorPat) NodeSpan() diagnostics.Span { return n.Span }
func (n *ConstructorPat) patternNode()                {}

// ---------- Struct destructure: Point { x, y } ----------

type StructPat struct {
	Span   diagnostics.Span
	Name   string
	Fields []FieldPat
}

func (n *StructPat) NodeSpan() diagnostics.Span { return n.Span }
func (n *StructPat) patternNode()                {}

// FieldPat is a field in a struct pattern. If Pat is nil, the field name
// is used as a binding (shorthand).
type FieldPat struct {
	Span diagnostics.Span
	Name string
	Pat  Pattern // nil = shorthand binding
}
