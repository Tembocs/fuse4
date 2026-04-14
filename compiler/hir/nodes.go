package hir

import (
	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

// ===== Top-level declarations =====

// Function is a resolved function declaration.
type Function struct {
	nodeBase
	Name       string
	Public     bool
	Params     []Param
	ReturnType typetable.TypeId
	Body       *Block
	FuncType   typetable.TypeId // the function's own type
}

// Param is a function parameter with resolved type.
type Param struct {
	Span      diagnostics.Span
	Name      string
	Type      typetable.TypeId
	Ownership OwnershipKind
}

// StructDef is a resolved struct definition.
type StructDef struct {
	nodeBase
	Name       string
	Public     bool
	Fields     []FieldDef
	TypeId     typetable.TypeId
}

// FieldDef is a resolved struct field.
type FieldDef struct {
	Name string
	Type typetable.TypeId
}

// EnumDef is a resolved enum definition.
type EnumDef struct {
	nodeBase
	Name     string
	Public   bool
	Variants []VariantDef
	TypeId   typetable.TypeId
}

// VariantDef is a resolved enum variant.
type VariantDef struct {
	Name       string
	FieldTypes []typetable.TypeId // tuple variant payload types
	Fields     []FieldDef         // struct variant fields
}

// ===== Expressions =====

type LiteralExpr struct {
	nodeBase
	Value string
}

func (n *LiteralExpr) exprNode() {}

type IdentExpr struct {
	nodeBase
	Name string
}

func (n *IdentExpr) exprNode() {}

type BinaryExpr struct {
	nodeBase
	Op    string
	Left  Expr
	Right Expr
}

func (n *BinaryExpr) exprNode() {}

type UnaryExpr struct {
	nodeBase
	Op      string
	Operand Expr
}

func (n *UnaryExpr) exprNode() {}

type AssignExpr struct {
	nodeBase
	Op     string
	Target Expr
	Value  Expr
}

func (n *AssignExpr) exprNode() {}

type CallExpr struct {
	nodeBase
	Callee Expr
	Args   []Expr
}

func (n *CallExpr) exprNode() {}

type IndexExpr struct {
	nodeBase
	Expr  Expr
	Index Expr
}

func (n *IndexExpr) exprNode() {}

type FieldExpr struct {
	nodeBase
	Expr Expr
	Name string
}

func (n *FieldExpr) exprNode() {}

type QDotExpr struct {
	nodeBase
	Expr Expr
	Name string
}

func (n *QDotExpr) exprNode() {}

type QuestionExpr struct {
	nodeBase
	Expr Expr
}

func (n *QuestionExpr) exprNode() {}

type Block struct {
	nodeBase
	Stmts []Stmt
	Tail  Expr // may be nil
}

func (n *Block) exprNode() {}

type IfExpr struct {
	nodeBase
	Cond Expr
	Then *Block
	Else Expr // *Block or *IfExpr, may be nil
}

func (n *IfExpr) exprNode() {}

type MatchExpr struct {
	nodeBase
	Subject Expr
	Arms    []MatchArm
}

func (n *MatchExpr) exprNode() {}

type MatchArm struct {
	PatternDesc string // textual description for now; full pattern HIR in later waves
	Guard       Expr   // may be nil
	Body        Expr
}

type ForExpr struct {
	nodeBase
	Binding  string
	Iterable Expr
	Body     *Block
}

func (n *ForExpr) exprNode() {}

type WhileExpr struct {
	nodeBase
	Cond Expr
	Body *Block
}

func (n *WhileExpr) exprNode() {}

type LoopExpr struct {
	nodeBase
	Body *Block
}

func (n *LoopExpr) exprNode() {}

type ReturnExpr struct {
	nodeBase
	Value Expr // may be nil
}

func (n *ReturnExpr) exprNode() {}

type BreakExpr struct {
	nodeBase
	Value Expr // may be nil
}

func (n *BreakExpr) exprNode() {}

type ContinueExpr struct {
	nodeBase
}

func (n *ContinueExpr) exprNode() {}

type SpawnExpr struct {
	nodeBase
	Expr Expr
}

func (n *SpawnExpr) exprNode() {}

type TupleExpr struct {
	nodeBase
	Elems []Expr
}

func (n *TupleExpr) exprNode() {}

type StructLitExpr struct {
	nodeBase
	Name   string
	Fields []FieldInitHIR
}

func (n *StructLitExpr) exprNode() {}

type FieldInitHIR struct {
	Name  string
	Value Expr
}

type ClosureExpr struct {
	nodeBase
	Params     []Param
	ReturnType typetable.TypeId
	Body       *Block
}

func (n *ClosureExpr) exprNode() {}

// ===== Statements =====

type LetStmt struct {
	nodeBase
	Name  string
	Type  typetable.TypeId
	Value Expr // may be nil
}

func (n *LetStmt) stmtNode() {}

type VarStmt struct {
	nodeBase
	Name  string
	Type  typetable.TypeId
	Value Expr // may be nil
}

func (n *VarStmt) stmtNode() {}

type ExprStmt struct {
	nodeBase
	Expr Expr
}

func (n *ExprStmt) stmtNode() {}
