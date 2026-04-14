package hir

import (
	"fmt"

	"github.com/Tembocs/fuse4/compiler/typetable"
)

// InvariantViolation describes a structural check failure.
type InvariantViolation struct {
	Node    Node
	Message string
}

func (v InvariantViolation) Error() string {
	return fmt.Sprintf("%s: %s", v.Node.NodeSpan(), v.Message)
}

// WalkInvariants runs structural invariant checks on an HIR function body.
// It is intended to run at pass boundaries in debug and CI (Rule 3.5).
//
// Current checks:
//   - No node has Unknown type after the check pass.
//   - Diverging nodes (return, break, continue) have Diverges set.
//   - Block tail expressions must not have Unknown type.
func WalkInvariants(fn *Function, afterPass string) []InvariantViolation {
	var violations []InvariantViolation
	if fn.Body == nil {
		return nil
	}
	walkExpr(fn.Body, afterPass, &violations)
	return violations
}

func walkExpr(e Expr, afterPass string, out *[]InvariantViolation) {
	if e == nil {
		return
	}

	md := e.Meta()

	// After "check" pass, no node should have Unknown type.
	if afterPass == "check" || afterPass == "liveness" || afterPass == "lower" {
		if md.Type == typetable.InvalidTypeId {
			*out = append(*out, InvariantViolation{
				Node:    e,
				Message: fmt.Sprintf("node has InvalidTypeId after %q pass", afterPass),
			})
		}
	}

	// Diverging nodes must have Diverges flag.
	switch n := e.(type) {
	case *ReturnExpr:
		if !md.Diverges {
			*out = append(*out, InvariantViolation{
				Node:    e,
				Message: "return expression missing Diverges flag",
			})
		}
		walkExpr(n.Value, afterPass, out)
	case *BreakExpr:
		if !md.Diverges {
			*out = append(*out, InvariantViolation{
				Node:    e,
				Message: "break expression missing Diverges flag",
			})
		}
		walkExpr(n.Value, afterPass, out)
	case *ContinueExpr:
		if !md.Diverges {
			*out = append(*out, InvariantViolation{
				Node:    e,
				Message: "continue expression missing Diverges flag",
			})
		}
	case *Block:
		for _, s := range n.Stmts {
			walkStmt(s, afterPass, out)
		}
		walkExpr(n.Tail, afterPass, out)
	case *BinaryExpr:
		walkExpr(n.Left, afterPass, out)
		walkExpr(n.Right, afterPass, out)
	case *UnaryExpr:
		walkExpr(n.Operand, afterPass, out)
	case *AssignExpr:
		walkExpr(n.Target, afterPass, out)
		walkExpr(n.Value, afterPass, out)
	case *CallExpr:
		walkExpr(n.Callee, afterPass, out)
		for _, a := range n.Args {
			walkExpr(a, afterPass, out)
		}
	case *IndexExpr:
		walkExpr(n.Expr, afterPass, out)
		walkExpr(n.Index, afterPass, out)
	case *FieldExpr:
		walkExpr(n.Expr, afterPass, out)
	case *QDotExpr:
		walkExpr(n.Expr, afterPass, out)
	case *QuestionExpr:
		walkExpr(n.Expr, afterPass, out)
	case *IfExpr:
		walkExpr(n.Cond, afterPass, out)
		walkExpr(n.Then, afterPass, out)
		walkExpr(n.Else, afterPass, out)
	case *MatchExpr:
		walkExpr(n.Subject, afterPass, out)
		for _, arm := range n.Arms {
			walkExpr(arm.Guard, afterPass, out)
			walkExpr(arm.Body, afterPass, out)
		}
	case *ForExpr:
		walkExpr(n.Iterable, afterPass, out)
		walkExpr(n.Body, afterPass, out)
	case *WhileExpr:
		walkExpr(n.Cond, afterPass, out)
		walkExpr(n.Body, afterPass, out)
	case *LoopExpr:
		walkExpr(n.Body, afterPass, out)
	case *SpawnExpr:
		walkExpr(n.Expr, afterPass, out)
	case *TupleExpr:
		for _, el := range n.Elems {
			walkExpr(el, afterPass, out)
		}
	case *StructLitExpr:
		for _, f := range n.Fields {
			walkExpr(f.Value, afterPass, out)
		}
	case *ClosureExpr:
		walkExpr(n.Body, afterPass, out)
	case *LiteralExpr, *IdentExpr:
		// leaf nodes — no children to walk
	}
}

func walkStmt(s Stmt, afterPass string, out *[]InvariantViolation) {
	if s == nil {
		return
	}
	switch n := s.(type) {
	case *LetStmt:
		walkExpr(n.Value, afterPass, out)
	case *VarStmt:
		walkExpr(n.Value, afterPass, out)
	case *ExprStmt:
		walkExpr(n.Expr, afterPass, out)
	}
}
