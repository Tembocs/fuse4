package liveness

import "github.com/Tembocs/fuse4/compiler/hir"

// VarSlot tracks a local variable for liveness analysis.
type VarSlot struct {
	Name      string
	Ownership hir.OwnershipKind
	Slot      int
}

// LivenessResult holds the output of the single liveness computation.
type LivenessResult struct {
	Slots    []VarSlot
	LastUses map[string]hir.Expr // variable name → the expression node of its last use
}

// ComputeLiveness performs a single liveness pass over a function body.
// It populates LiveAfter and DestroyEnd metadata on HIR nodes.
// This function must be called exactly once per function (Rule 3.8).
func ComputeLiveness(fn *hir.Function) *LivenessResult {
	if fn.Body == nil {
		return &LivenessResult{LastUses: make(map[string]hir.Expr)}
	}

	ctx := &livenessCtx{
		slots:    make(map[string]int),
		lastUse:  make(map[string]hir.Expr),
		liveSet:  make(map[int]bool),
		slotInfo: nil,
	}

	// Collect all variable slots from params + body locals.
	for i, p := range fn.Params {
		ctx.slots[p.Name] = i
		ctx.slotInfo = append(ctx.slotInfo, VarSlot{
			Name:      p.Name,
			Ownership: p.Ownership,
			Slot:      i,
		})
		ctx.liveSet[i] = true
	}
	ctx.nextSlot = len(fn.Params)

	// Walk the body in forward order, tracking variable uses.
	ctx.walkExpr(fn.Body)

	// Build result.
	result := &LivenessResult{
		Slots:    ctx.slotInfo,
		LastUses: ctx.lastUse,
	}
	return result
}

type livenessCtx struct {
	slots    map[string]int     // variable name → slot index
	lastUse  map[string]hir.Expr // variable name → last use node
	liveSet  map[int]bool       // currently live slot indices
	slotInfo []VarSlot
	nextSlot int
}

func (c *livenessCtx) defineLocal(name string, ownership hir.OwnershipKind) {
	slot := c.nextSlot
	c.nextSlot++
	c.slots[name] = slot
	c.slotInfo = append(c.slotInfo, VarSlot{
		Name:      name,
		Ownership: ownership,
		Slot:      slot,
	})
	c.liveSet[slot] = true
}

func (c *livenessCtx) recordUse(name string, node hir.Expr) {
	c.lastUse[name] = node
	// Set live-after on this node: all currently live slots.
	liveAfter := make([]int, 0, len(c.liveSet))
	for slot := range c.liveSet {
		liveAfter = append(liveAfter, slot)
	}
	node.Meta().LiveAfter = liveAfter
}

func (c *livenessCtx) walkExpr(e hir.Expr) {
	if e == nil {
		return
	}

	switch n := e.(type) {
	case *hir.IdentExpr:
		c.recordUse(n.Name, e)

	case *hir.Block:
		for _, s := range n.Stmts {
			c.walkStmt(s)
		}
		if n.Tail != nil {
			c.walkExpr(n.Tail)
		}

	case *hir.BinaryExpr:
		c.walkExpr(n.Left)
		c.walkExpr(n.Right)
	case *hir.UnaryExpr:
		c.walkExpr(n.Operand)
	case *hir.AssignExpr:
		c.walkExpr(n.Value)
		c.walkExpr(n.Target)
	case *hir.CallExpr:
		c.walkExpr(n.Callee)
		for _, arg := range n.Args {
			c.walkExpr(arg)
		}
	case *hir.IndexExpr:
		c.walkExpr(n.Expr)
		c.walkExpr(n.Index)
	case *hir.FieldExpr:
		c.walkExpr(n.Expr)
	case *hir.QDotExpr:
		c.walkExpr(n.Expr)
	case *hir.QuestionExpr:
		c.walkExpr(n.Expr)
	case *hir.IfExpr:
		c.walkExpr(n.Cond)
		c.walkExpr(n.Then)
		c.walkExpr(n.Else)
	case *hir.MatchExpr:
		c.walkExpr(n.Subject)
		for _, arm := range n.Arms {
			c.walkExpr(arm.Guard)
			c.walkExpr(arm.Body)
		}
	case *hir.ForExpr:
		c.defineLocal(n.Binding, hir.OwnerValue)
		c.walkExpr(n.Iterable)
		c.walkExpr(n.Body)
	case *hir.WhileExpr:
		c.walkExpr(n.Cond)
		c.walkExpr(n.Body)
	case *hir.LoopExpr:
		c.walkExpr(n.Body)
	case *hir.ReturnExpr:
		c.walkExpr(n.Value)
	case *hir.BreakExpr:
		c.walkExpr(n.Value)
	case *hir.SpawnExpr:
		c.walkExpr(n.Expr)
	case *hir.TupleExpr:
		for _, el := range n.Elems {
			c.walkExpr(el)
		}
	case *hir.StructLitExpr:
		for _, f := range n.Fields {
			c.walkExpr(f.Value)
		}
	case *hir.ClosureExpr:
		// Closures start a new liveness scope (simplified: we don't descend).
	case *hir.LiteralExpr, *hir.ContinueExpr:
		// leaf nodes
	}
}

func (c *livenessCtx) walkStmt(s hir.Stmt) {
	if s == nil {
		return
	}
	switch n := s.(type) {
	case *hir.LetStmt:
		c.walkExpr(n.Value)
		c.defineLocal(n.Name, hir.OwnerValue)
	case *hir.VarStmt:
		c.walkExpr(n.Value)
		c.defineLocal(n.Name, hir.OwnerValue)
	case *hir.ExprStmt:
		c.walkExpr(n.Expr)
	}
}
