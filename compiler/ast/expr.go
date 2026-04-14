package ast

import (
	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/lex"
)

// ---------- Literal ----------

type LiteralExpr struct {
	Span  diagnostics.Span
	Token lex.Token
}

func (n *LiteralExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *LiteralExpr) exprNode()                   {}

// ---------- Identifier ----------

type IdentExpr struct {
	Span diagnostics.Span
	Name string
}

func (n *IdentExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *IdentExpr) exprNode()                   {}

// ---------- Unary ----------

type UnaryExpr struct {
	Span    diagnostics.Span
	Op      lex.Token
	Operand Expr
}

func (n *UnaryExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *UnaryExpr) exprNode()                   {}

// ---------- Binary ----------

type BinaryExpr struct {
	Span  diagnostics.Span
	Left  Expr
	Op    lex.Token
	Right Expr
}

func (n *BinaryExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *BinaryExpr) exprNode()                   {}

// ---------- Assign ----------

type AssignExpr struct {
	Span   diagnostics.Span
	Target Expr
	Op     lex.Token // =, +=, -=, etc.
	Value  Expr
}

func (n *AssignExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *AssignExpr) exprNode()                   {}

// ---------- Call ----------

type CallExpr struct {
	Span   diagnostics.Span
	Callee Expr
	Args   []Expr
}

func (n *CallExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *CallExpr) exprNode()                   {}

// ---------- Index ----------

type IndexExpr struct {
	Span  diagnostics.Span
	Expr  Expr
	Index Expr
}

func (n *IndexExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *IndexExpr) exprNode()                   {}

// ---------- Field access ----------

type FieldExpr struct {
	Span diagnostics.Span
	Expr Expr
	Name string
}

func (n *FieldExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *FieldExpr) exprNode()                   {}

// ---------- Optional chaining ----------

type QDotExpr struct {
	Span diagnostics.Span
	Expr Expr
	Name string
}

func (n *QDotExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *QDotExpr) exprNode()                   {}

// ---------- Postfix ? ----------

type QuestionExpr struct {
	Span diagnostics.Span
	Expr Expr
}

func (n *QuestionExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *QuestionExpr) exprNode()                   {}

// ---------- Block ----------

type BlockExpr struct {
	Span  diagnostics.Span
	Stmts []Stmt
	Tail  Expr // trailing expression without semicolon, may be nil
}

func (n *BlockExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *BlockExpr) exprNode()                   {}

// ---------- If ----------

type IfExpr struct {
	Span diagnostics.Span
	Cond Expr
	Then *BlockExpr
	Else Expr // *BlockExpr or *IfExpr (else-if chain), may be nil
}

func (n *IfExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *IfExpr) exprNode()                   {}

// ---------- Match ----------

type MatchExpr struct {
	Span    diagnostics.Span
	Subject Expr
	Arms    []MatchArm
}

func (n *MatchExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *MatchExpr) exprNode()                   {}

// ---------- For ----------

type ForExpr struct {
	Span     diagnostics.Span
	Binding  string
	Iterable Expr
	Body     *BlockExpr
}

func (n *ForExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *ForExpr) exprNode()                   {}

// ---------- While ----------

type WhileExpr struct {
	Span diagnostics.Span
	Cond Expr
	Body *BlockExpr
}

func (n *WhileExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *WhileExpr) exprNode()                   {}

// ---------- Loop ----------

type LoopExpr struct {
	Span diagnostics.Span
	Body *BlockExpr
}

func (n *LoopExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *LoopExpr) exprNode()                   {}

// ---------- Tuple ----------

type TupleExpr struct {
	Span  diagnostics.Span
	Elems []Expr
}

func (n *TupleExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *TupleExpr) exprNode()                   {}

// ---------- Struct literal ----------

type StructLitExpr struct {
	Span   diagnostics.Span
	Name   string
	Fields []FieldInit
}

func (n *StructLitExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *StructLitExpr) exprNode()                   {}

// ---------- Closure ----------

type ClosureExpr struct {
	Span       diagnostics.Span
	Params     []Param
	ReturnType TypeExpr // nil if absent
	Body       *BlockExpr
}

func (n *ClosureExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *ClosureExpr) exprNode()                   {}

// ---------- Spawn ----------

type SpawnExpr struct {
	Span diagnostics.Span
	Expr Expr
}

func (n *SpawnExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *SpawnExpr) exprNode()                   {}

// ---------- Return ----------

type ReturnExpr struct {
	Span  diagnostics.Span
	Value Expr // may be nil
}

func (n *ReturnExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *ReturnExpr) exprNode()                   {}

// ---------- Break ----------

type BreakExpr struct {
	Span  diagnostics.Span
	Value Expr // may be nil
}

func (n *BreakExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *BreakExpr) exprNode()                   {}

// ---------- Continue ----------

type ContinueExpr struct {
	Span diagnostics.Span
}

func (n *ContinueExpr) NodeSpan() diagnostics.Span { return n.Span }
func (n *ContinueExpr) exprNode()                   {}
