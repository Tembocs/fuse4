// Package liveness owns the single liveness computation and
// ownership context analysis for the Fuse compiler.
//
// Liveness is computed once per function (Rule 3.8) and the results
// are stored on HIR Metadata. Later passes consume this data without
// recomputation.
package liveness

import (
	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/hir"
)

// OwnershipAnalyzer walks HIR and assigns ownership metadata to each node
// based on its syntactic context and the ownership forms in the language.
type OwnershipAnalyzer struct {
	Errors []diagnostics.Diagnostic
}

// AnalyzeOwnership sets ownership metadata on all nodes in a function body.
func (a *OwnershipAnalyzer) AnalyzeOwnership(fn *hir.Function) {
	if fn.Body == nil {
		return
	}
	// Parameters carry their declared ownership.
	for i := range fn.Params {
		// Param ownership is already set during construction; nothing to adjust.
		_ = fn.Params[i]
	}
	a.walkExpr(fn.Body, hir.OwnerValue)
}

func (a *OwnershipAnalyzer) walkExpr(e hir.Expr, ctx hir.OwnershipKind) {
	if e == nil {
		return
	}
	e.Meta().Ownership = ctx

	switch n := e.(type) {
	case *hir.Block:
		for _, s := range n.Stmts {
			a.walkStmt(s)
		}
		if n.Tail != nil {
			a.walkExpr(n.Tail, ctx)
		}
	case *hir.BinaryExpr:
		a.walkExpr(n.Left, hir.OwnerValue)
		a.walkExpr(n.Right, hir.OwnerValue)
	case *hir.UnaryExpr:
		switch n.Op {
		case "ref":
			a.walkExpr(n.Operand, hir.OwnerRef)
		case "mutref":
			a.walkExpr(n.Operand, hir.OwnerMutRef)
		case "move":
			a.walkExpr(n.Operand, hir.OwnerMoved)
		case "owned":
			a.walkExpr(n.Operand, hir.OwnerOwned)
		default:
			a.walkExpr(n.Operand, hir.OwnerValue)
		}
	case *hir.AssignExpr:
		a.walkExpr(n.Target, hir.OwnerValue)
		a.walkExpr(n.Value, hir.OwnerValue)
	case *hir.CallExpr:
		a.walkExpr(n.Callee, hir.OwnerValue)
		for _, arg := range n.Args {
			a.walkExpr(arg, hir.OwnerValue)
		}
	case *hir.IndexExpr:
		a.walkExpr(n.Expr, hir.OwnerValue)
		a.walkExpr(n.Index, hir.OwnerValue)
	case *hir.FieldExpr:
		a.walkExpr(n.Expr, hir.OwnerValue)
	case *hir.QDotExpr:
		a.walkExpr(n.Expr, hir.OwnerRef)
	case *hir.QuestionExpr:
		a.walkExpr(n.Expr, hir.OwnerValue)
	case *hir.IfExpr:
		a.walkExpr(n.Cond, hir.OwnerValue)
		a.walkExpr(n.Then, ctx)
		a.walkExpr(n.Else, ctx)
	case *hir.MatchExpr:
		a.walkExpr(n.Subject, hir.OwnerValue)
		for _, arm := range n.Arms {
			a.walkExpr(arm.Guard, hir.OwnerValue)
			a.walkExpr(arm.Body, ctx)
		}
	case *hir.ForExpr:
		a.walkExpr(n.Iterable, hir.OwnerValue)
		a.walkExpr(n.Body, hir.OwnerValue)
	case *hir.WhileExpr:
		a.walkExpr(n.Cond, hir.OwnerValue)
		a.walkExpr(n.Body, hir.OwnerValue)
	case *hir.LoopExpr:
		a.walkExpr(n.Body, hir.OwnerValue)
	case *hir.ReturnExpr:
		a.walkExpr(n.Value, hir.OwnerValue)
	case *hir.BreakExpr:
		a.walkExpr(n.Value, hir.OwnerValue)
	case *hir.SpawnExpr:
		a.walkExpr(n.Expr, hir.OwnerOwned)
	case *hir.TupleExpr:
		for _, el := range n.Elems {
			a.walkExpr(el, hir.OwnerValue)
		}
	case *hir.StructLitExpr:
		for _, f := range n.Fields {
			a.walkExpr(f.Value, hir.OwnerValue)
		}
	case *hir.ClosureExpr:
		a.walkExpr(n.Body, hir.OwnerValue)
	case *hir.LiteralExpr, *hir.IdentExpr, *hir.ContinueExpr:
		// leaf nodes
	}
}

func (a *OwnershipAnalyzer) walkStmt(s hir.Stmt) {
	if s == nil {
		return
	}
	switch n := s.(type) {
	case *hir.LetStmt:
		a.walkExpr(n.Value, hir.OwnerValue)
	case *hir.VarStmt:
		a.walkExpr(n.Value, hir.OwnerValue)
	case *hir.ExprStmt:
		a.walkExpr(n.Expr, hir.OwnerValue)
	}
}
