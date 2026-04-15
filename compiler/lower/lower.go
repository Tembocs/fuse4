// Package lower owns HIR-to-MIR lowering and the preservation of
// semantic contracts into MIR.
package lower

import (
	"fmt"

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

	// LiftedFunctions holds closure bodies that were lifted to standalone functions.
	LiftedFunctions []*mir.Function

	// closureEnvs maps a closure reference local to its environment struct local.
	closureEnvs map[mir.LocalId]mir.LocalId
	// closureFns maps a closure reference local to its lifted function name.
	closureFns map[mir.LocalId]string
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
	l.closureEnvs = make(map[mir.LocalId]mir.LocalId)
	l.closureFns = make(map[mir.LocalId]string)

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
		return l.lowerQuestion(n)
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
		return l.lowerSpawn(n)
	case *hir.TupleExpr:
		return l.lowerTuple(n)
	case *hir.StructLitExpr:
		return l.lowerStructLit(n)
	case *hir.EnumInitExpr:
		return l.lowerEnumInit(n)
	case *hir.ClosureExpr:
		return l.lowerClosure(n)
	default:
		return l.constUnit()
	}
}

// lowerSpawn emits a call to fuse_rt_thread_spawn(fn, arg).
func (l *Lowerer) lowerSpawn(n *hir.SpawnExpr) mir.LocalId {
	fn := l.lowerExpr(n.Expr)

	// Create a reference to the runtime spawn function.
	spawnFn := l.b.NewTemp(l.Types.Unknown)
	l.b.EmitConst(spawnFn, l.Types.Unknown, "fuse_rt_thread_spawn")

	// Call spawn with the function as argument. The second arg (data) is null for now.
	nullArg := l.b.NewTemp(l.Types.Unknown)
	l.b.EmitConst(nullArg, l.Types.Unknown, "0")

	dest := l.b.NewTemp(l.Types.Unit)
	l.b.EmitCall(dest, spawnFn, []mir.LocalId{fn, nullArg}, l.Types.Unit, false)
	return dest
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
	// Top-level reference (function name, constant, etc.) — emit with mangled name
	// to match codegen's MangleName convention.
	dest := l.b.NewTemp(n.Meta().Type)
	mangledName := n.Name
	if n.Name != "main" {
		mangledName = "Fuse_" + n.Name
	}
	l.b.EmitConst(dest, n.Meta().Type, mangledName)
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
		// Method call: obj.method(args) → call(Fuse_method, &obj, args...)
		recv := l.lowerExpr(fe.Expr)
		// Pass receiver as a reference (borrow) for ref self methods.
		recvType := fe.Expr.Meta().Type
		refType := l.Types.InternRef(recvType)
		recvRef := l.b.NewTemp(refType)
		l.b.EmitBorrow(recvRef, recv, refType, mir.BorrowShared)
		var args []mir.LocalId
		args = append(args, recvRef)
		for _, a := range n.Args {
			args = append(args, l.lowerExpr(a))
		}
		// Mangle the method name for C emission.
		callee := l.b.NewTemp(l.Types.Unknown)
		l.b.EmitConst(callee, l.Types.Unknown, "Fuse_"+fe.Name)
		l.b.EmitCall(dest, callee, args, n.Meta().Type, true)
		return dest
	}

	// Direct function call: emit callee name as a const reference.
	var callee mir.LocalId
	if ident, ok := n.Callee.(*hir.IdentExpr); ok {
		// Built-in I/O functions: lower to runtime calls.
		if ident.Name == "println" || ident.Name == "print" {
			return l.lowerPrintCall(dest, ident.Name, n.Args)
		}
		// Check if this ident refers to a local that holds a closure reference.
		if localId, isLocal := l.vars[ident.Name]; isLocal {
			if fnName, isClosure := l.closureFns[localId]; isClosure {
				// Closure call: use the lifted function name and prepend env arg.
				callee = l.b.NewTemp(l.Types.Unknown)
				l.b.EmitConst(callee, l.Types.Unknown, fnName)
				var args []mir.LocalId
				if envId, hasEnv := l.closureEnvs[localId]; hasEnv {
					args = append(args, envId)
				}
				for _, a := range n.Args {
					args = append(args, l.lowerExpr(a))
				}
				l.b.EmitCall(dest, callee, args, n.Meta().Type, false)
				return dest
			}
		}
		// Direct call by name — emit the mangled function name.
		callee = l.b.NewTemp(l.Types.Unknown)
		mangledName := ident.Name
		if mangledName != "main" {
			if _, isLocal := l.vars[ident.Name]; !isLocal {
				mangledName = "Fuse_" + ident.Name
			}
		}
		l.b.EmitConst(callee, l.Types.Unknown, mangledName)
	} else {
		callee = l.lowerExpr(n.Callee)
	}
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

// lowerQuestion implements the ? operator: check discriminant, extract Ok/Some value
// on success, early-return Err/None on failure.
func (l *Lowerer) lowerQuestion(n *hir.QuestionExpr) mir.LocalId {
	subject := l.lowerExpr(n.Expr)

	// Read the discriminant tag.
	tag := l.b.NewTemp(l.Types.I32)
	l.b.EmitFieldRead(tag, subject, "_tag", l.Types.I32)

	// Compare: tag == 0 means success (Ok/Some).
	zero := l.b.NewTemp(l.Types.I32)
	l.b.EmitConst(zero, l.Types.I32, "0")
	cmp := l.b.NewTemp(l.Types.Bool)
	l.b.EmitBinOp(cmp, "==", tag, zero, l.Types.Bool)

	successBlock := l.b.NewBlock()
	failBlock := l.b.NewBlock()
	l.b.TermBranch(cmp, successBlock, failBlock)

	// Failure path: early return with the error/None value.
	l.b.SwitchToBlock(failBlock)
	l.b.TermReturn(subject)

	// Success path: extract the inner value from _f0.
	l.b.SwitchToBlock(successBlock)
	result := l.b.NewTemp(n.Meta().Type)
	l.b.EmitFieldRead(result, subject, "_f0", n.Meta().Type)
	return result
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
			// Propagate closure metadata: if the value is a closure reference,
			// map the let binding's local to the same closure info.
			if fnName, ok := l.closureFns[val]; ok {
				l.closureFns[local] = fnName
				if envId, hasEnv := l.closureEnvs[val]; hasEnv {
					l.closureEnvs[local] = envId
				}
			}
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
	subject := l.lowerExpr(n.Subject)
	result := l.b.NewTemp(n.Meta().Type)
	joinBlock := l.b.NewBlock()

	for i, arm := range n.Arms {
		armBlock := l.b.NewBlock()
		nextBlock := l.b.NewBlock()
		if i == len(n.Arms)-1 {
			nextBlock = joinBlock // last arm falls to join if no match
		}

		switch pat := arm.Pattern.(type) {
		case *hir.WildcardPattern:
			l.b.TermGoto(armBlock)
		case *hir.BindPattern:
			l.b.TermGoto(armBlock)
		case *hir.LiteralPattern:
			cmp := l.b.NewTemp(l.Types.Bool)
			l.b.EmitBinOp(cmp, "==", subject, l.emitConst(pat.Value, pat.Type), l.Types.Bool)
			l.b.TermBranch(cmp, armBlock, nextBlock)
		case *hir.ConstructorPattern:
			tag := l.b.NewTemp(l.Types.I32)
			l.b.EmitFieldRead(tag, subject, "_tag", l.Types.I32)
			expected := l.b.NewTemp(l.Types.I32)
			l.b.EmitConst(expected, l.Types.I32, fmt.Sprintf("%d", pat.Tag))
			cmp := l.b.NewTemp(l.Types.Bool)
			l.b.EmitBinOp(cmp, "==", tag, expected, l.Types.Bool)
			l.b.TermBranch(cmp, armBlock, nextBlock)
		default:
			// nil pattern or unknown: unconditional (backwards compat with PatternDesc)
			l.b.TermGoto(armBlock)
		}

		l.b.SwitchToBlock(armBlock)

		// Bind pattern variable if needed
		if pat, ok := arm.Pattern.(*hir.BindPattern); ok {
			local := l.b.NewLocal(pat.Name, pat.Type)
			l.vars[pat.Name] = local
			l.b.EmitCopy(local, subject, pat.Type)
		}

		// Handle constructor pattern bindings
		if pat, ok := arm.Pattern.(*hir.ConstructorPattern); ok {
			for j, arg := range pat.Args {
				if bp, ok := arg.(*hir.BindPattern); ok {
					local := l.b.NewLocal(bp.Name, bp.Type)
					l.vars[bp.Name] = local
					fieldName := fmt.Sprintf("_f%d", j)
					l.b.EmitFieldRead(local, subject, fieldName, bp.Type)
				}
			}
		}

		val := l.lowerExpr(arm.Body)
		if !l.b.IsSealed() {
			l.b.EmitCopy(result, val, n.Meta().Type)
			l.b.TermGoto(joinBlock)
		}

		if i < len(n.Arms)-1 {
			l.b.SwitchToBlock(nextBlock)
		}
	}

	l.b.SwitchToBlock(joinBlock)
	return result
}

func (l *Lowerer) emitConst(value string, ty typetable.TypeId) mir.LocalId {
	dest := l.b.NewTemp(ty)
	l.b.EmitConst(dest, ty, value)
	return dest
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
		l.b.EmitCopy(ctx.BreakLocal, val, n.Value.Meta().Type)
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

// lowerPrintCall emits a runtime call for print/println built-in functions.
func (l *Lowerer) lowerPrintCall(dest mir.LocalId, name string, args []hir.Expr) mir.LocalId {
	// Lower the string argument.
	if len(args) == 0 {
		if name == "println" {
			// println() with no args: just write a newline.
			callee := l.b.NewTemp(l.Types.Unknown)
			l.b.EmitConst(callee, l.Types.Unknown, "fuse_rt_io_write_stdout")
			nlData := l.b.NewTemp(l.Types.Unknown)
			l.b.EmitConst(nlData, l.Types.Unknown, "(uint8_t*)\"\\n\"")
			nlLen := l.b.NewTemp(l.Types.USize)
			l.b.EmitConst(nlLen, l.Types.USize, "1")
			l.b.EmitCall(dest, callee, []mir.LocalId{nlData, nlLen}, l.Types.Unit, false)
		}
		return dest
	}

	strArg := l.lowerExpr(args[0])

	// Extract .data and .len fields from the String struct.
	ptrTy := l.Types.InternPtr(l.Types.U8)
	dataLocal := l.b.NewTemp(ptrTy)
	l.b.EmitFieldRead(dataLocal, strArg, "data", ptrTy)
	lenLocal := l.b.NewTemp(l.Types.USize)
	l.b.EmitFieldRead(lenLocal, strArg, "len", l.Types.USize)

	// Call fuse_rt_io_write_stdout(data, len).
	callee := l.b.NewTemp(l.Types.Unknown)
	l.b.EmitConst(callee, l.Types.Unknown, "fuse_rt_io_write_stdout")
	l.b.EmitCall(dest, callee, []mir.LocalId{dataLocal, lenLocal}, l.Types.Unit, false)

	// For println, also write a newline.
	if name == "println" {
		nlCallee := l.b.NewTemp(l.Types.Unknown)
		l.b.EmitConst(nlCallee, l.Types.Unknown, "fuse_rt_io_write_stdout")
		nlData := l.b.NewTemp(ptrTy)
		l.b.EmitConst(nlData, ptrTy, "(uint8_t*)\"\\n\"")
		nlLen := l.b.NewTemp(l.Types.USize)
		l.b.EmitConst(nlLen, l.Types.USize, "1")
		nlDest := l.b.NewTemp(l.Types.Unit)
		l.b.EmitCall(nlDest, nlCallee, []mir.LocalId{nlData, nlLen}, l.Types.Unit, false)
	}

	return dest
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

func (l *Lowerer) lowerEnumInit(n *hir.EnumInitExpr) mir.LocalId {
	// Emit tag as first field, then payload args.
	tagLocal := l.b.NewTemp(l.Types.I32)
	l.b.EmitConst(tagLocal, l.Types.I32, fmt.Sprintf("%d", n.Tag))
	args := []mir.LocalId{tagLocal}
	for _, a := range n.Args {
		args = append(args, l.lowerExpr(a))
	}
	dest := l.b.NewTemp(n.Meta().Type)
	l.b.EmitEnumInit(dest, n.VariantName, args, n.Meta().Type)
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

// --- closure lowering ---

// lowerClosure performs capture analysis, lifts the closure body to a
// standalone function, and emits an environment struct at the call site.
func (l *Lowerer) lowerClosure(n *hir.ClosureExpr) mir.LocalId {
	// Phase 1: Capture analysis — find which outer variables the closure references.
	captures := l.analyzeCaptures(n)

	// Phase 2: Build the lifted function.
	// The lifted function takes an env parameter followed by the closure's own params.
	envType := l.Types.InternStruct("__closure", fmt.Sprintf("env_%d", len(l.LiftedFunctions)), nil)

	// Set the env struct fields so codegen can emit its definition.
	if len(captures) > 0 {
		var capNames []string
		var capTypes []typetable.TypeId
		for i, cap := range captures {
			capNames = append(capNames, fmt.Sprintf("_c%d", i))
			capTypes = append(capTypes, cap.Type)
		}
		l.Types.SetStructFields(envType, capNames, capTypes)
	}

	var liftedParams []mir.Local
	// Only add env param if the closure actually captures variables.
	if len(captures) > 0 {
		liftedParams = append(liftedParams, mir.Local{
			Id: 0, Name: "__env", Type: envType,
		})
	}
	// Then the closure's declared params.
	for i, p := range n.Params {
		paramId := mir.LocalId(i)
		if len(captures) > 0 {
			paramId = mir.LocalId(i + 1)
		}
		liftedParams = append(liftedParams, mir.Local{
			Id: paramId, Name: p.Name, Type: p.Type,
		})
	}

	liftedBuilder := mir.NewBuilder(
		fmt.Sprintf("__closure_%d", len(l.LiftedFunctions)),
		liftedParams,
		n.ReturnType,
	)

	// Create a child lowerer for the closure body.
	childLowerer := &Lowerer{
		Types: l.Types,
		b:     liftedBuilder,
		vars:  make(map[string]mir.LocalId),
	}

	// Register params in the child's var map.
	if len(captures) > 0 {
		envLocal := mir.LocalId(0)
		for i, p := range n.Params {
			childLowerer.vars[p.Name] = mir.LocalId(i + 1)
		}
		// Register captured variables — read from env struct fields.
		for i, cap := range captures {
			dest := childLowerer.b.NewLocal(cap.Name, cap.Type)
			childLowerer.vars[cap.Name] = dest
			childLowerer.b.EmitFieldRead(dest, envLocal, fmt.Sprintf("_c%d", i), cap.Type)
		}
	} else {
		for i, p := range n.Params {
			childLowerer.vars[p.Name] = mir.LocalId(i)
		}
	}

	// Lower the closure body in the child lowerer.
	if n.Body != nil {
		bodyResult := childLowerer.lowerExpr(n.Body)
		if !childLowerer.b.IsSealed() {
			childLowerer.b.TermReturn(bodyResult)
		}
	}

	liftedFn := liftedBuilder.Build()
	l.LiftedFunctions = append(l.LiftedFunctions, liftedFn)

	// Phase 3: At the closure expression site, emit a reference to the lifted function.
	// For closures without captures, we just emit the function name as a const.
	// For closures with captures, we build the env struct and pass it as the first arg
	// at the call site (handled by lowerCall when it detects a closure callee).
	liftedName := liftedFn.Name

	if len(captures) > 0 {
		// Build the environment struct at the closure site.
		envDest := l.b.NewTemp(envType)
		var captureLocals []mir.LocalId
		for _, cap := range captures {
			if id, ok := l.vars[cap.Name]; ok {
				captureLocals = append(captureLocals, id)
			} else {
				tmp := l.b.NewTemp(cap.Type)
				l.b.EmitConst(tmp, cap.Type, "0")
				captureLocals = append(captureLocals, tmp)
			}
		}
		l.b.EmitStructInit(envDest, fmt.Sprintf("env_%d", len(l.LiftedFunctions)-1), captureLocals, envType)

		// Store the lifted function name and env local for later call rewriting.
		fnRef := l.b.NewTemp(n.Meta().Type)
		l.b.EmitConst(fnRef, n.Meta().Type, "Fuse_"+liftedName)
		// Record env local for this closure so lowerCall can prepend it.
		l.closureEnvs[fnRef] = envDest
		l.closureFns[fnRef] = "Fuse_" + liftedName
		return fnRef
	}

	// No captures: just emit a reference to the lifted function.
	fnRef := l.b.NewTemp(n.Meta().Type)
	l.b.EmitConst(fnRef, n.Meta().Type, "Fuse_"+liftedName)
	l.closureFns[fnRef] = "Fuse_" + liftedName
	return fnRef
}

// capturedVar tracks a variable captured by a closure.
type capturedVar struct {
	Name string
	Type typetable.TypeId
}

// analyzeCaptures scans a closure body for references to outer variables.
func (l *Lowerer) analyzeCaptures(n *hir.ClosureExpr) []capturedVar {
	// Collect closure param names to exclude them from captures.
	paramNames := make(map[string]bool)
	for _, p := range n.Params {
		paramNames[p.Name] = true
	}

	var captures []capturedVar
	seen := make(map[string]bool)
	l.scanCaptures(n.Body, paramNames, seen, &captures)
	return captures
}

func (l *Lowerer) scanCaptures(e hir.Expr, params map[string]bool, seen map[string]bool, out *[]capturedVar) {
	if e == nil {
		return
	}
	switch n := e.(type) {
	case *hir.IdentExpr:
		if !params[n.Name] && !seen[n.Name] {
			if _, ok := l.vars[n.Name]; ok {
				seen[n.Name] = true
				*out = append(*out, capturedVar{Name: n.Name, Type: n.Meta().Type})
			}
		}
	case *hir.Block:
		for _, s := range n.Stmts {
			l.scanCapturesStmt(s, params, seen, out)
		}
		l.scanCaptures(n.Tail, params, seen, out)
	case *hir.BinaryExpr:
		l.scanCaptures(n.Left, params, seen, out)
		l.scanCaptures(n.Right, params, seen, out)
	case *hir.UnaryExpr:
		l.scanCaptures(n.Operand, params, seen, out)
	case *hir.CallExpr:
		l.scanCaptures(n.Callee, params, seen, out)
		for _, a := range n.Args {
			l.scanCaptures(a, params, seen, out)
		}
	case *hir.IfExpr:
		l.scanCaptures(n.Cond, params, seen, out)
		l.scanCaptures(n.Then, params, seen, out)
		l.scanCaptures(n.Else, params, seen, out)
	case *hir.ReturnExpr:
		l.scanCaptures(n.Value, params, seen, out)
	case *hir.AssignExpr:
		l.scanCaptures(n.Target, params, seen, out)
		l.scanCaptures(n.Value, params, seen, out)
	case *hir.IndexExpr:
		l.scanCaptures(n.Expr, params, seen, out)
		l.scanCaptures(n.Index, params, seen, out)
	case *hir.FieldExpr:
		l.scanCaptures(n.Expr, params, seen, out)
	}
}

func (l *Lowerer) scanCapturesStmt(s hir.Stmt, params map[string]bool, seen map[string]bool, out *[]capturedVar) {
	if s == nil {
		return
	}
	switch n := s.(type) {
	case *hir.LetStmt:
		l.scanCaptures(n.Value, params, seen, out)
	case *hir.VarStmt:
		l.scanCaptures(n.Value, params, seen, out)
	case *hir.ExprStmt:
		l.scanCaptures(n.Expr, params, seen, out)
	}
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
