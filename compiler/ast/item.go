package ast

import "github.com/Tembocs/fuse4/compiler/diagnostics"

// ---------- Import ----------

type ImportDecl struct {
	Span  diagnostics.Span
	Path  []string // e.g. ["core", "result", "Result"]
	Alias string   // optional "as" alias, "" if absent
}

func (n *ImportDecl) NodeSpan() diagnostics.Span { return n.Span }
func (n *ImportDecl) itemNode()                   {}

// ---------- Function ----------

type FnDecl struct {
	Span          diagnostics.Span
	Public        bool
	Name          string
	GenericParams []GenericParam
	Params        []Param
	ReturnType    TypeExpr     // nil if no explicit return type
	Where         *WhereClause // nil if absent
	Body          *BlockExpr
}

func (n *FnDecl) NodeSpan() diagnostics.Span { return n.Span }
func (n *FnDecl) itemNode()                   {}

// ---------- Struct ----------

type StructDecl struct {
	Span          diagnostics.Span
	Public        bool
	Decorators    []Decorator
	Name          string
	GenericParams []GenericParam
	Fields        []Field
}

func (n *StructDecl) NodeSpan() diagnostics.Span { return n.Span }
func (n *StructDecl) itemNode()                   {}

// ---------- Enum ----------

type EnumDecl struct {
	Span          diagnostics.Span
	Public        bool
	Name          string
	GenericParams []GenericParam
	Variants      []Variant
}

func (n *EnumDecl) NodeSpan() diagnostics.Span { return n.Span }
func (n *EnumDecl) itemNode()                   {}

// ---------- Trait ----------

type TraitDecl struct {
	Span          diagnostics.Span
	Public        bool
	Name          string
	GenericParams []GenericParam
	Supertraits   []TypeExpr
	Items         []Item // method signatures and associated items
}

func (n *TraitDecl) NodeSpan() diagnostics.Span { return n.Span }
func (n *TraitDecl) itemNode()                   {}

// ---------- Impl ----------

type ImplDecl struct {
	Span          diagnostics.Span
	GenericParams []GenericParam
	Target        TypeExpr     // the type being implemented
	Trait         TypeExpr     // nil if inherent impl
	Where         *WhereClause // nil if absent
	Items         []Item       // methods
}

func (n *ImplDecl) NodeSpan() diagnostics.Span { return n.Span }
func (n *ImplDecl) itemNode()                   {}

// ---------- Const ----------

type ConstDecl struct {
	Span   diagnostics.Span
	Public bool
	Name   string
	Type   TypeExpr // nil if inferred
	Value  Expr
}

func (n *ConstDecl) NodeSpan() diagnostics.Span { return n.Span }
func (n *ConstDecl) itemNode()                   {}

// ---------- Type alias ----------

type TypeAliasDecl struct {
	Span          diagnostics.Span
	Public        bool
	Name          string
	GenericParams []GenericParam
	Type          TypeExpr
}

func (n *TypeAliasDecl) NodeSpan() diagnostics.Span { return n.Span }
func (n *TypeAliasDecl) itemNode()                   {}

// ---------- Extern function ----------

type ExternFnDecl struct {
	Span       diagnostics.Span
	Public     bool
	Name       string
	Params     []Param
	ReturnType TypeExpr // nil if void
}

func (n *ExternFnDecl) NodeSpan() diagnostics.Span { return n.Span }
func (n *ExternFnDecl) itemNode()                   {}
