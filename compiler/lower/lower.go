// Package lower owns HIR-to-MIR lowering and the preservation of
// semantic contracts into MIR.
package lower

import (
	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/hir"
	"github.com/Tembocs/fuse4/compiler/mir"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

// Lowerer translates HIR functions into MIR functions.
type Lowerer struct {
	Types  *typetable.TypeTable
	Errors []diagnostics.Diagnostic

	b     *mir.Builder
	vars  map[string]mir.LocalId // named variable → local
	loops []loopCtx              // loop context stack for break/continue
}

type loopCtx struct {
	BreakBlock    mir.BlockId
	ContinueBlock mir.BlockId
	BreakLocal    mir.LocalId // local to store break value
}

// LowerFunction lowers a single HIR function to MIR.
func (l *Lowerer) LowerFunction(fn *hir.Function) *mir.Function {
	// Build param locals.
	var params []mir.Local
	for i, p := range fn.Params {
		params = append(params, mir.Local{
			Id:   mir.LocalId(i),
			Name: p.Name,
			Type: p.Type,
		})
	}

	l.b = mir.NewBuilder(fn.Name, params, fn.ReturnType)
	l.vars = make(map[string]mir.LocalId)

	// Register params in var map.
	for i, p := range fn.Params {
		l.vars[p.Name] = mir.LocalId(i)
	}

	if fn.Body != nil {
		result := l.lowerExpr(fn.Body)
		// If the body doesn't diverge, emit return.
		if !l.b.IsSealed() {
			l.b.TermReturn(result)
		}
	}

	return l.b.Build()
}

// New creates a lowerer for the given type table.
func New(types *typetable.TypeTable) *Lowerer {
	return &Lowerer{Types: types}
}

// --- expression lowering ---

func (l *Lowerer) lowerExpr(e hir.Expr) mir.LocalId {
	if e == nil {
		return l.constUnit()
	}

	switch n := e.(type) {
	case *hir.LiteralExpr:
		return l.lowerLiteral(n)
	case *hir.IdentExpr:
		return l.lowerIdent(n)
	case *hir.BinaryExpr:
		return l.lowerBinary(n)
	case *hir.UnaryExpr:
		return l.lowerUnary(n)
	case *hir.AssignExpr:
		return l.lowerAssign(n)
	case *hir.CallExpr:
		return l.lowerCall(n)
	case *hir.IndexExpr:
		return l.lowerIndex(n)
	case *hir.FieldExpr:
		return l.lowerField(n)
	case *hir.QDotExpr:
		return l.lowerField(&hir.FieldExpr{Expr: n.Expr, Name: n.Name})
	case *hir.QuestionExpr:
		return l.lowerExpr(n.Expr) // simplified: unwrap
	case *hir.Block:
		return l.lowerBlock(n)
	case *hir.IfExpr:
		return l.lowerIf(n)
	case *hir.MatchExpr:
		return l.lowerMatch(n)
	case *hir.ForExpr:
		return l.lowerFor(n)
	case *hir.WhileExpr:
		return l.lowerWhile(n)
	case *hir.LoopExpr:
		return l.lowerLoop(n)
	case *hir.ReturnExpr:
		return l.lowerReturn(n)
	case *hir.BreakExpr:
		return l.lowerBreak(n)
	case *hir.ContinueExpr:
		return l.lowerContinue()
	case *hir.SpawnExpr:
		arg := l.lowerExpr(n.Expr)
		dest := l.b.NewTemp(l.Types.Unit)
		l.b.EmitCall(dest, arg, nil, l.Types.Unit, false)
		return dest
	case *hir.TupleExpr:
		return l.lowerTuple(n)
	case *hir.StructLitExpr:
		return l.lowerStructLit(n)
	case *hir.ClosureExpr:
		// Closures lower as function references (simplified).
		return l.constUnit()
	default:
		return l.constUnit()
	}
}

func (l *Lowerer) lowerLiteral(n *hir.LiteralExpr) mir.LocalId {
	dest := l.b.NewTemp(n.Meta().Type)
	l.b.EmitConst(dest, n.Meta().Type, n.Value)
	return dest
}

func (l *Lowerer) lowerIdent(n *hir.IdentExpr) mir.LocalId {
	if id, ok := l.vars[n.Name]; ok {
		return id
	}
	// Unknown ident — create a temp with unknown value.
	dest := l.b.NewTemp(n.Meta().Type)
	l.b.EmitConst(dest, n.Meta().Type, n.Name)
	return dest
}

func (l *Lowerer) lowerBinary(n *hir.BinaryExpr) mir.LocalId {
	left := l.lowerExpr(n.Left)
	right := l.lowerExpr(n.Right)
	dest := l.b.NewTemp(n.Meta().Type)
	l.b.EmitBinOp(dest, n.Op, left, right, n.Meta().Type)
	return dest
}

// lowerUnary handles unary operators. Per implementation contract,
// ref and mutref MUST lower to InstrBorrow, not a generic unary op.
func (l *Lowerer) lowerUnary(n *hir.UnaryExpr) mir.LocalId {
	operand := l.lowerExpr(n.Operand)
	dest := l.b.NewTemp(n.Meta().Type)

	switch n.Op {
	case "ref":
		l.b.EmitBorrow(dest, operand, n.Meta().Type, mir.BorrowShared)
	case "mutref":
		l.b.EmitBorrow(dest, operand, n.Meta().Type, mir.BorrowMutable)
	case "move":
		l.b.EmitMove(dest, operand, n.Meta().Type)
	default:
		l.b.EmitUnaryOp(dest, n.Op, operand, n.Meta().Type)
	}
	return dest
}

func (l *Lowerer) lowerAssign(n *hir.AssignExpr) mir.LocalId {
	val := l.lowerExpr(n.Value)
	target := l.lowerExpr(n.Target)
	l.b.EmitCopy(target, val, n.Value.Meta().Type)
	return l.constUnit()
}

// lowerCall disambiguates method calls from plain calls.
// Per contract: if callee is a FieldExpr, it's a method call with
// the field's object as first argument.
func (l *Lowerer) lowerCall(n *hir.CallExpr) mir.LocalId {
	dest := l.b.NewTemp(n.Meta().Type)

	if fe, ok := n.Callee.(*hir.FieldExpr); ok {
		// Method call: obj.method(args) → call(method, obj, args...)
		recv := l.lowerExpr(fe.Expr)
		var args []mir.LocalId
		args = append(args, recv)
		for _, a := range n.Args {
			args = append(args, l.lowerExpr(a))
		}
		// Method is referenced by name, create a callee local.
		callee := l.b.NewTemp(l.Types.Unknown)
		l.b.EmitConst(callee, l.Types.Unknown, fe.Name)
		l.b.EmitCall(dest, callee, args, n.Meta().Type, true)
		return dest
	}

	callee := l.lowerExpr(n.Callee)
	var args []mir.LocalId
	for _, a := range n.Args {
		args = append(args, l.lowerExpr(a))
	}
	l.b.EmitCall(dest, callee, args, n.Meta().Type, false)
	return dest
}

func (l *Lowerer) lowerIndex(n *hir.IndexExpr) mir.LocalId {
	src := l.lowerExpr(n.Expr)
	idx := l.lowerExpr(n.Index)
	dest := l.b.NewTemp(n.Meta().Type)
	l.b.EmitIndex(dest, src, idx, n.Meta().Type)
	return dest
}

func (l *Lowerer) lowerField(n *hir.FieldExpr) mir.LocalId {
	src := l.lowerExpr(n.Expr)
	dest := l.b.NewTemp(n.Meta().Type)
	l.b.EmitFieldRead(dest, src, n.Name, n.Meta().Type)
	return dest
}

func (l *Lowerer) lowerBlock(n *hir.Block) mir.LocalId {
	for _, s := range n.Stmts {
		l.lowerStmt(s)
		if l.b.IsSealed() {
			return l.constUnit() // diverged; no phantom locals
		}
	}
	if n.Tail != nil {
		return l.lowerExpr(n.Tail)
	}
	return l.constUnit()
}

func (l *Lowerer) lowerStmt(s hir.Stmt) {
	switch n := s.(type) {
	case *hir.LetStmt:
		local := l.b.NewLocal(n.Name, n.Type)
		l.vars[n.Name] = local
		if n.Value != nil {
			val := l.lowerExpr(n.Value)
			l.b.EmitCopy(local, val, n.Type)
		}
		if n.Meta().DestroyEnd {
			l.b.EmitDrop(local)
		}
	case *hir.VarStmt:
		local := l.b.NewLocal(n.Name, n.Type)
		l.vars[n.Name] = local
		if n.Value != nil {
			val := l.lowerExpr(n.Value)
			l.b.EmitCopy(local, val, n.Type)
		}
	case *hir.ExprStmt:
		l.lowerExpr(n.Expr)
	}
}

// --- control flow ---

func (l *Lowerer) lowerIf(n *hir.IfExpr) mir.LocalId {
	cond := l.lowerExpr(n.Cond)
	result := l.b.NewTemp(n.Meta().Type)

	thenBlock := l.b.NewBlock()
	elseBlock := l.b.NewBlock()
	joinBlock := l.b.NewBlock()

	l.b.TermBranch(cond, thenBlock, elseBlock)

	// Then branch
	l.b.SwitchToBlock(thenBlock)
	thenVal := l.lowerExpr(n.Then)
	if !l.b.IsSealed() {
		l.b.EmitCopy(result, thenVal, n.Meta().Type)
		l.b.TermGoto(joinBlock)
	}

	// Else branch
	l.b.SwitchToBlock(elseBlock)
	if n.Else != nil {
		elseVal := l.lowerExpr(n.Else)
		if !l.b.IsSealed() {
			l.b.EmitCopy(result, elseVal, n.Meta().Type)
			l.b.TermGoto(joinBlock)
		}
	} else {
		l.b.TermGoto(joinBlock)
	}

	l.b.SwitchToBlock(joinBlock)
	return result
}

func (l *Lowerer) lowerMatch(n *hir.MatchExpr) mir.LocalId {
	l.lowerExpr(n.Subject)
	result := l.b.NewTemp(n.Meta().Type)
	joinBlock := l.b.NewBlock()

	for _, arm := range n.Arms {
		armBlock := l.b.NewBlock()
		l.b.TermGoto(armBlock) // simplified: no real pattern dispatch yet
		l.b.SwitchToBlock(armBlock)
		val := l.lowerExpr(arm.Body)
		if !l.b.IsSealed() {
			l.b.EmitCopy(result, val, n.Meta().Type)
			l.b.TermGoto(joinBlock)
		}
	}

	l.b.SwitchToBlock(joinBlock)
	return result
}

func (l *Lowerer) lowerFor(n *hir.ForExpr) mir.LocalId {
	iterLocal := l.lowerExpr(n.Iterable)
	bindLocal := l.b.NewLocal(n.Binding, l.Types.Unknown)
	l.vars[n.Binding] = bindLocal

	headerBlock := l.b.NewBlock()
	bodyBlock := l.b.NewBlock()
	exitBlock := l.b.NewBlock()

	l.b.TermGoto(headerBlock)

	l.b.SwitchToBlock(headerBlock)
	// Simplified: unconditional loop, real iterator protocol comes later.
	l.b.TermBranch(iterLocal, bodyBlock, exitBlock)

	l.b.SwitchToBlock(bodyBlock)
	l.loops = append(l.loops, loopCtx{BreakBlock: exitBlock, ContinueBlock: headerBlock})
	l.lowerExpr(n.Body)
	l.loops = l.loops[:len(l.loops)-1]
	if !l.b.IsSealed() {
		l.b.TermGoto(headerBlock)
	}

	l.b.SwitchToBlock(exitBlock)
	return l.constUnit()
}

func (l *Lowerer) lowerWhile(n *hir.WhileExpr) mir.LocalId {
	headerBlock := l.b.NewBlock()
	bodyBlock := l.b.NewBlock()
	exitBlock := l.b.NewBlock()

	l.b.TermGoto(headerBlock)

	l.b.SwitchToBlock(headerBlock)
	cond := l.lowerExpr(n.Cond)
	l.b.TermBranch(cond, bodyBlock, exitBlock)

	l.b.SwitchToBlock(bodyBlock)
	l.loops = append(l.loops, loopCtx{BreakBlock: exitBlock, ContinueBlock: headerBlock})
	l.lowerExpr(n.Body)
	l.loops = l.loops[:len(l.loops)-1]
	if !l.b.IsSealed() {
		l.b.TermGoto(headerBlock)
	}

	l.b.SwitchToBlock(exitBlock)
	return l.constUnit()
}

func (l *Lowerer) lowerLoop(n *hir.LoopExpr) mir.LocalId {
	headerBlock := l.b.NewBlock()
	exitBlock := l.b.NewBlock()
	breakLocal := l.b.NewTemp(n.Meta().Type)

	l.b.TermGoto(headerBlock)

	l.b.SwitchToBlock(headerBlock)
	l.loops = append(l.loops, loopCtx{
		BreakBlock:    exitBlock,
		ContinueBlock: headerBlock,
		BreakLocal:    breakLocal,
	})
	l.lowerExpr(n.Body)
	l.loops = l.loops[:len(l.loops)-1]
	if !l.b.IsSealed() {
		l.b.TermGoto(headerBlock)
	}

	l.b.SwitchToBlock(exitBlock)
	return breakLocal
}

// lowerReturn seals the current block — no phantom locals after this point.
func (l *Lowerer) lowerReturn(n *hir.ReturnExpr) mir.LocalId {
	if n.Value != nil {
		val := l.lowerExpr(n.Value)
		l.b.TermReturn(val)
	} else {
		l.b.TermReturn(l.constUnit())
	}
	return l.constUnit() // unreachable, but satisfies return type
}

// lowerBreak seals the current block.
func (l *Lowerer) lowerBreak(n *hir.BreakExpr) mir.LocalId {
	if len(l.loops) == 0 {
		return l.constUnit()
	}
	ctx := l.loops[len(l.loops)-1]
	if n.Value != nil {
		val := l.lowerExpr(n.Value)
		l.b.EmitCopy(ctx.BreakLocal, val, l.Types.Unknown)
	}
	l.b.TermGoto(ctx.BreakBlock)
	return l.constUnit()
}

// lowerContinue seals the current block.
func (l *Lowerer) lowerContinue() mir.LocalId {
	if len(l.loops) == 0 {
		return l.constUnit()
	}
	ctx := l.loops[len(l.loops)-1]
	l.b.TermGoto(ctx.ContinueBlock)
	return l.constUnit()
}

// --- compound expressions ---

func (l *Lowerer) lowerTuple(n *hir.TupleExpr) mir.LocalId {
	var elems []mir.LocalId
	for _, e := range n.Elems {
		elems = append(elems, l.lowerExpr(e))
	}
	dest := l.b.NewTemp(n.Meta().Type)
	l.b.EmitTuple(dest, elems, n.Meta().Type)
	return dest
}

func (l *Lowerer) lowerStructLit(n *hir.StructLitExpr) mir.LocalId {
	var fields []mir.LocalId
	for _, f := range n.Fields {
		fields = append(fields, l.lowerExpr(f.Value))
	}
	dest := l.b.NewTemp(n.Meta().Type)
	l.b.EmitStructInit(dest, n.Name, fields, n.Meta().Type)
	return dest
}

// --- utility ---

func (l *Lowerer) constUnit() mir.LocalId {
	dest := l.b.NewTemp(l.Types.Unit)
	l.b.EmitConst(dest, l.Types.Unit, "()")
	return dest
}

func (l *Lowerer) errorf(span diagnostics.Span, format string, args ...any) {
	l.Errors = append(l.Errors, diagnostics.Errorf(span, format, args...))
}
