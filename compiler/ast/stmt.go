package ast

import "github.com/Tembocs/fuse4/compiler/diagnostics"

// ---------- Let binding ----------

type LetStmt struct {
	Span  diagnostics.Span
	Name  string
	Type  TypeExpr // nil if not annotated
	Value Expr     // nil if not initialized
}

func (n *LetStmt) NodeSpan() diagnostics.Span { return n.Span }
func (n *LetStmt) stmtNode()                   {}

// ---------- Var binding ----------

type VarStmt struct {
	Span  diagnostics.Span
	Name  string
	Type  TypeExpr // nil if not annotated
	Value Expr     // nil if not initialized
}

func (n *VarStmt) NodeSpan() diagnostics.Span { return n.Span }
func (n *VarStmt) stmtNode()                   {}

// ---------- Expression statement ----------

type ExprStmt struct {
	Span diagnostics.Span
	Expr Expr
}

func (n *ExprStmt) NodeSpan() diagnostics.Span { return n.Span }
func (n *ExprStmt) stmtNode()                   {}

// ---------- Item statement (declaration inside a block) ----------

type ItemStmt struct {
	Span diagnostics.Span
	Item Item
}

func (n *ItemStmt) NodeSpan() diagnostics.Span { return n.Span }
func (n *ItemStmt) stmtNode()                   {}
