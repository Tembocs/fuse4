// Package hir owns the semantically rich HIR and HIR builders.
// HIR nodes carry metadata fields used by later passes.
//
// HIR is distinct from AST (syntax-only) and MIR (explicit low-level).
// HIR nodes MUST be constructed through builder functions, not ad hoc.
package hir

import (
	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

// NodeId uniquely identifies an HIR node within a function or module.
type NodeId int

// Metadata carries the semantic annotations that later passes read and write.
// Every HIR node embeds a Metadata value.
type Metadata struct {
	Type       typetable.TypeId
	Ownership  OwnershipKind
	Diverges   bool  // true if this expression never returns
	LiveAfter  []int // populated by liveness pass (slot indices alive after this node)
	DestroyEnd bool  // true if this node is the last use requiring destruction
}

// OwnershipKind classifies the ownership context of an expression.
type OwnershipKind int

const (
	OwnerValue  OwnershipKind = iota // ordinary by-value
	OwnerRef                         // shared borrow
	OwnerMutRef                      // mutable borrow
	OwnerOwned                       // transferring ownership
	OwnerMoved                       // value has been moved
)

func (k OwnershipKind) String() string {
	switch k {
	case OwnerValue:
		return "value"
	case OwnerRef:
		return "ref"
	case OwnerMutRef:
		return "mutref"
	case OwnerOwned:
		return "owned"
	case OwnerMoved:
		return "moved"
	default:
		return "?"
	}
}

// --- Node interfaces ---

// Node is the common interface for all HIR nodes.
type Node interface {
	NodeSpan() diagnostics.Span
	Meta() *Metadata
}

// Expr is an HIR expression node.
type Expr interface {
	Node
	exprNode()
}

// Stmt is an HIR statement node.
type Stmt interface {
	Node
	stmtNode()
}

// --- base type embedded in all HIR nodes ---

type nodeBase struct {
	Span diagnostics.Span
	MD   Metadata
}

func (n *nodeBase) NodeSpan() diagnostics.Span { return n.Span }
func (n *nodeBase) Meta() *Metadata            { return &n.MD }
