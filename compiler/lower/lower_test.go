package lower

import (
	"testing"

	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/hir"
	"github.com/Tembocs/fuse4/compiler/mir"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

var zs = diagnostics.Span{}

func lowerFn(t *testing.T, fn *hir.Function, tt *typetable.TypeTable) *mir.Function {
	t.Helper()
	l := New(tt)
	result := l.LowerFunction(fn)
	for _, e := range l.Errors {
		t.Errorf("lower error: %s", e)
	}
	return result
}

func makeBuilder() (*hir.Builder, *typetable.TypeTable) {
	tt := typetable.New()
	return hir.NewBuilder(tt), tt
}

// ===== Phase 01: MIR instruction set =====

func TestLowerLiteral(t *testing.T) {
	b, tt := makeBuilder()
	lit := b.Literal(zs, "42", tt.I32)
	block := b.BlockExpr(zs, nil, lit, tt.I32)
	fn := &hir.Function{Name: "test", Body: block, ReturnType: tt.I32}

	mf := lowerFn(t, fn, tt)
	if mf.Name != "test" {
		t.Errorf("name: %q", mf.Name)
	}
	if len(mf.Blocks) == 0 {
		t.Fatal("no blocks")
	}
	// Entry block should have a const instruction.
	found := false
	for _, instr := range mf.Blocks[0].Instrs {
		if instr.Kind == mir.InstrConst && instr.Value == "42" {
			found = true
		}
	}
	if !found {
		t.Error("expected const instruction for literal 42")
	}
}

func TestLowerBinaryOp(t *testing.T) {
	b, tt := makeBuilder()
	left := b.Literal(zs, "1", tt.I32)
	right := b.Literal(zs, "2", tt.I32)
	add := b.Binary(zs, "+", left, right, tt.I32)
	block := b.BlockExpr(zs, nil, add, tt.I32)
	fn := &hir.Function{Name: "test", Body: block, ReturnType: tt.I32}

	mf := lowerFn(t, fn, tt)
	found := false
	for _, instr := range mf.Blocks[0].Instrs {
		if instr.Kind == mir.InstrBinOp && instr.Op == "+" {
			found = true
		}
	}
	if !found {
		t.Error("expected binop instruction for +")
	}
}

// ===== Phase 01: Borrow instructions (Rule: ref/mutref → InstrBorrow) =====

func TestLowerRefIsBorrowInstr(t *testing.T) {
	b, tt := makeBuilder()
	x := b.Ident(zs, "x", tt.I32)
	refX := b.Unary(zs, "ref", x, tt.InternRef(tt.I32))
	block := b.BlockExpr(zs, nil, refX, tt.InternRef(tt.I32))
	fn := &hir.Function{
		Name: "test", Body: block, ReturnType: tt.InternRef(tt.I32),
		Params: []hir.Param{{Span: zs, Name: "x", Type: tt.I32}},
	}

	mf := lowerFn(t, fn, tt)
	found := false
	for _, blk := range mf.Blocks {
		for _, instr := range blk.Instrs {
			if instr.Kind == mir.InstrBorrow && instr.BorrowKind == mir.BorrowShared {
				found = true
			}
		}
	}
	if !found {
		t.Error("ref must lower to InstrBorrow(BorrowShared), not generic unary")
	}
}

func TestLowerMutrefIsBorrowInstr(t *testing.T) {
	b, tt := makeBuilder()
	x := b.Ident(zs, "x", tt.I32)
	mutrefX := b.Unary(zs, "mutref", x, tt.InternMutRef(tt.I32))
	block := b.BlockExpr(zs, nil, mutrefX, tt.InternMutRef(tt.I32))
	fn := &hir.Function{
		Name: "test", Body: block, ReturnType: tt.InternMutRef(tt.I32),
		Params: []hir.Param{{Span: zs, Name: "x", Type: tt.I32}},
	}

	mf := lowerFn(t, fn, tt)
	found := false
	for _, blk := range mf.Blocks {
		for _, instr := range blk.Instrs {
			if instr.Kind == mir.InstrBorrow && instr.BorrowKind == mir.BorrowMutable {
				found = true
			}
		}
	}
	if !found {
		t.Error("mutref must lower to InstrBorrow(BorrowMutable)")
	}
}

func TestLowerMoveIsInstrMove(t *testing.T) {
	b, tt := makeBuilder()
	x := b.Ident(zs, "x", tt.I32)
	moveX := b.Unary(zs, "move", x, tt.I32)
	block := b.BlockExpr(zs, nil, moveX, tt.I32)
	fn := &hir.Function{
		Name: "test", Body: block, ReturnType: tt.I32,
		Params: []hir.Param{{Span: zs, Name: "x", Type: tt.I32}},
	}

	mf := lowerFn(t, fn, tt)
	found := false
	for _, blk := range mf.Blocks {
		for _, instr := range blk.Instrs {
			if instr.Kind == mir.InstrMove {
				found = true
			}
		}
	}
	if !found {
		t.Error("move must lower to InstrMove")
	}
}

// ===== Phase 02: Control flow =====

func TestLowerIfCreatesBlocks(t *testing.T) {
	b, tt := makeBuilder()
	cond := b.Literal(zs, "true", tt.Bool)
	thenBlock := b.BlockExpr(zs, nil, b.Literal(zs, "1", tt.I32), tt.I32)
	elseBlock := b.BlockExpr(zs, nil, b.Literal(zs, "2", tt.I32), tt.I32)
	ifExpr := b.If(zs, cond, thenBlock, elseBlock, tt.I32)
	block := b.BlockExpr(zs, nil, ifExpr, tt.I32)
	fn := &hir.Function{Name: "test", Body: block, ReturnType: tt.I32}

	mf := lowerFn(t, fn, tt)
	// Should have multiple blocks: entry, then, else, join.
	if len(mf.Blocks) < 4 {
		t.Errorf("expected >= 4 blocks for if/else, got %d", len(mf.Blocks))
	}
	// Entry block should end with a branch.
	entry := mf.Blocks[0]
	if entry.Term.Kind != mir.TermBranch {
		t.Errorf("entry terminator: %d, want TermBranch", entry.Term.Kind)
	}
}

func TestLowerReturnSealsBlock(t *testing.T) {
	b, tt := makeBuilder()
	ret := b.Return(zs, b.Literal(zs, "42", tt.I32))
	block := b.BlockExpr(zs, []hir.Stmt{b.ExprStatement(zs, ret)}, nil, tt.Never)
	fn := &hir.Function{Name: "test", Body: block, ReturnType: tt.I32}

	mf := lowerFn(t, fn, tt)
	// Entry block should be sealed with TermReturn.
	entry := mf.Blocks[0]
	if !entry.Sealed {
		t.Error("block with return should be sealed")
	}
	if entry.Term.Kind != mir.TermReturn {
		t.Errorf("terminator: %d, want TermReturn", entry.Term.Kind)
	}
}

func TestLowerWhileLoop(t *testing.T) {
	b, tt := makeBuilder()
	cond := b.Literal(zs, "true", tt.Bool)
	body := b.BlockExpr(zs, nil, nil, tt.Unit)
	while := b.While(zs, cond, body)
	block := b.BlockExpr(zs, nil, while, tt.Unit)
	fn := &hir.Function{Name: "test", Body: block, ReturnType: tt.Unit}

	mf := lowerFn(t, fn, tt)
	// While creates: header, body, exit blocks.
	if len(mf.Blocks) < 4 {
		t.Errorf("expected >= 4 blocks for while, got %d", len(mf.Blocks))
	}
}

func TestLowerLoopWithBreak(t *testing.T) {
	b, tt := makeBuilder()
	brk := b.Break(zs, b.Literal(zs, "1", tt.I32))
	body := b.BlockExpr(zs, []hir.Stmt{b.ExprStatement(zs, brk)}, nil, tt.Never)
	loop := b.Loop(zs, body, tt.I32)
	block := b.BlockExpr(zs, nil, loop, tt.I32)
	fn := &hir.Function{Name: "test", Body: block, ReturnType: tt.I32}

	mf := lowerFn(t, fn, tt)
	// Break should seal the block and goto exit.
	foundGoto := false
	for _, blk := range mf.Blocks {
		if blk.Term.Kind == mir.TermGoto {
			foundGoto = true
		}
	}
	if !foundGoto {
		t.Error("break should produce a goto to exit block")
	}
}

func TestLowerContinueSealsBlock(t *testing.T) {
	b, tt := makeBuilder()
	cont := b.Continue(zs)
	body := b.BlockExpr(zs, []hir.Stmt{b.ExprStatement(zs, cont)}, nil, tt.Never)
	loop := b.Loop(zs, body, tt.Never)
	block := b.BlockExpr(zs, nil, loop, tt.Never)
	fn := &hir.Function{Name: "test", Body: block, ReturnType: tt.Never}

	mf := lowerFn(t, fn, tt)
	// Continue should seal the block and goto header.
	sealed := 0
	for _, blk := range mf.Blocks {
		if blk.Sealed {
			sealed++
		}
	}
	if sealed < 2 {
		t.Errorf("expected >= 2 sealed blocks, got %d", sealed)
	}
}

// ===== Phase 02: Divergence is structural =====

func TestDivergenceNoPhantomLocals(t *testing.T) {
	b, tt := makeBuilder()
	// return 1; let x = 2;  ← x should never be created (dead code after return)
	ret := b.Return(zs, b.Literal(zs, "1", tt.I32))
	letX := b.Let(zs, "x", tt.I32, b.Literal(zs, "2", tt.I32))
	block := b.BlockExpr(zs, []hir.Stmt{
		b.ExprStatement(zs, ret),
		letX,
	}, nil, tt.Never)
	fn := &hir.Function{Name: "test", Body: block, ReturnType: tt.I32}

	mf := lowerFn(t, fn, tt)
	// After return, the block is sealed. The let x should not produce a local
	// in the entry block's instructions.
	entry := mf.Blocks[0]
	if !entry.Sealed {
		t.Error("block should be sealed after return")
	}
	// No instruction for "x = 2" should appear after the return terminator.
	foundX := false
	for _, local := range mf.Locals {
		if local.Name == "x" {
			foundX = true
		}
	}
	// x should NOT be allocated because the block was sealed before the let.
	if foundX {
		t.Error("phantom local 'x' should not exist after diverging return")
	}
}

// ===== Phase 03: Method call disambiguation =====

func TestMethodCallLowersCorrectly(t *testing.T) {
	b, tt := makeBuilder()
	obj := b.Ident(zs, "obj", tt.InternStruct("m", "Foo", nil))
	field := b.Field(zs, obj, "bar", tt.InternFunc(nil, tt.I32))
	call := b.Call(zs, field, nil, tt.I32)
	block := b.BlockExpr(zs, nil, call, tt.I32)
	fn := &hir.Function{
		Name: "test", Body: block, ReturnType: tt.I32,
		Params: []hir.Param{{Span: zs, Name: "obj", Type: tt.InternStruct("m", "Foo", nil)}},
	}

	mf := lowerFn(t, fn, tt)
	// Should produce a method call (IsMethod=true), not a field read.
	foundMethodCall := false
	foundFieldRead := false
	for _, blk := range mf.Blocks {
		for _, instr := range blk.Instrs {
			if instr.Kind == mir.InstrCall && instr.IsMethod {
				foundMethodCall = true
			}
			if instr.Kind == mir.InstrFieldRead && instr.Field == "bar" {
				foundFieldRead = true
			}
		}
	}
	if !foundMethodCall {
		t.Error("obj.bar() should lower as method call (IsMethod=true)")
	}
	if foundFieldRead {
		t.Error("obj.bar() in call position should NOT produce a field read")
	}
}

func TestFieldReadLowersCorrectly(t *testing.T) {
	b, tt := makeBuilder()
	obj := b.Ident(zs, "obj", tt.InternStruct("m", "Foo", nil))
	field := b.Field(zs, obj, "x", tt.I32)
	block := b.BlockExpr(zs, nil, field, tt.I32)
	fn := &hir.Function{
		Name: "test", Body: block, ReturnType: tt.I32,
		Params: []hir.Param{{Span: zs, Name: "obj", Type: tt.InternStruct("m", "Foo", nil)}},
	}

	mf := lowerFn(t, fn, tt)
	found := false
	for _, blk := range mf.Blocks {
		for _, instr := range blk.Instrs {
			if instr.Kind == mir.InstrFieldRead && instr.Field == "x" {
				found = true
			}
		}
	}
	if !found {
		t.Error("obj.x (not in call position) should lower as field read")
	}
}

// ===== Phase 03: Struct and tuple construction =====

func TestLowerStructLit(t *testing.T) {
	b, tt := makeBuilder()
	sty := tt.InternStruct("m", "Point", nil)
	slit := b.StructLit(zs, "Point", []hir.FieldInitHIR{
		{Name: "x", Value: b.Literal(zs, "1", tt.I32)},
		{Name: "y", Value: b.Literal(zs, "2", tt.I32)},
	}, sty)
	block := b.BlockExpr(zs, nil, slit, sty)
	fn := &hir.Function{Name: "test", Body: block, ReturnType: sty}

	mf := lowerFn(t, fn, tt)
	found := false
	for _, blk := range mf.Blocks {
		for _, instr := range blk.Instrs {
			if instr.Kind == mir.InstrStructInit && instr.Field == "Point" {
				found = true
			}
		}
	}
	if !found {
		t.Error("struct literal should produce InstrStructInit")
	}
}

func TestLowerTuple(t *testing.T) {
	b, tt := makeBuilder()
	tty := tt.InternTuple([]typetable.TypeId{tt.I32, tt.Bool})
	tuple := b.Tuple(zs, []hir.Expr{
		b.Literal(zs, "1", tt.I32),
		b.Literal(zs, "true", tt.Bool),
	}, tty)
	block := b.BlockExpr(zs, nil, tuple, tty)
	fn := &hir.Function{Name: "test", Body: block, ReturnType: tty}

	mf := lowerFn(t, fn, tt)
	found := false
	for _, blk := range mf.Blocks {
		for _, instr := range blk.Instrs {
			if instr.Kind == mir.InstrTuple {
				found = true
			}
		}
	}
	if !found {
		t.Error("tuple should produce InstrTuple")
	}
}

// ===== Builder tests =====

func TestMIRBuilderSealedBlockIgnoresInstructions(t *testing.T) {
	tt := typetable.New()
	b := mir.NewBuilder("test", nil, tt.Unit)
	tmp := b.NewTemp(tt.I32)
	b.EmitConst(tmp, tt.I32, "1")
	b.TermReturn(tmp) // seals the block

	// Further instructions should be silently ignored.
	tmp2 := b.NewTemp(tt.I32)
	b.EmitConst(tmp2, tt.I32, "2")

	mf := b.Build()
	entry := mf.Blocks[0]
	if len(entry.Instrs) != 1 {
		t.Errorf("sealed block should have 1 instruction, got %d", len(entry.Instrs))
	}
}

// ===== Pattern matching dispatch =====

func TestMatchLiteralPatterns(t *testing.T) {
	b, tt := makeBuilder()
	subject := b.Literal(zs, "1", tt.I32)
	arms := []hir.MatchArm{
		{
			Pattern: &hir.LiteralPattern{Value: "1", Type: tt.I32},
			Body:    b.Literal(zs, "10", tt.I32),
		},
		{
			Pattern: &hir.LiteralPattern{Value: "2", Type: tt.I32},
			Body:    b.Literal(zs, "20", tt.I32),
		},
		{
			Pattern: &hir.WildcardPattern{},
			Body:    b.Literal(zs, "0", tt.I32),
		},
	}
	matchExpr := b.Match(zs, subject, arms, tt.I32)
	block := b.BlockExpr(zs, nil, matchExpr, tt.I32)
	fn := &hir.Function{Name: "test", Body: block, ReturnType: tt.I32}

	mf := lowerFn(t, fn, tt)

	// Literal patterns should produce TermBranch (conditional dispatch).
	branchCount := 0
	for _, blk := range mf.Blocks {
		if blk.Term.Kind == mir.TermBranch {
			branchCount++
		}
	}
	if branchCount < 2 {
		t.Errorf("expected >= 2 branch terminators for literal patterns, got %d", branchCount)
	}

	// Should have == comparison binops.
	eqCount := 0
	for _, blk := range mf.Blocks {
		for _, instr := range blk.Instrs {
			if instr.Kind == mir.InstrBinOp && instr.Op == "==" {
				eqCount++
			}
		}
	}
	if eqCount < 2 {
		t.Errorf("expected >= 2 equality comparisons for literal patterns, got %d", eqCount)
	}
}

func TestMatchWildcardPattern(t *testing.T) {
	b, tt := makeBuilder()
	subject := b.Literal(zs, "42", tt.I32)
	arms := []hir.MatchArm{
		{
			Pattern: &hir.WildcardPattern{},
			Body:    b.Literal(zs, "99", tt.I32),
		},
	}
	matchExpr := b.Match(zs, subject, arms, tt.I32)
	block := b.BlockExpr(zs, nil, matchExpr, tt.I32)
	fn := &hir.Function{Name: "test", Body: block, ReturnType: tt.I32}

	mf := lowerFn(t, fn, tt)

	// Wildcard should produce an unconditional goto, no branches.
	for _, blk := range mf.Blocks {
		if blk.Term.Kind == mir.TermBranch {
			t.Error("wildcard pattern should not produce a branch terminator")
		}
	}
}

func TestMatchBindPattern(t *testing.T) {
	b, tt := makeBuilder()
	subject := b.Literal(zs, "42", tt.I32)
	arms := []hir.MatchArm{
		{
			Pattern: &hir.BindPattern{Name: "x", Type: tt.I32},
			Body:    b.Ident(zs, "x", tt.I32),
		},
	}
	matchExpr := b.Match(zs, subject, arms, tt.I32)
	block := b.BlockExpr(zs, nil, matchExpr, tt.I32)
	fn := &hir.Function{Name: "test", Body: block, ReturnType: tt.I32}

	mf := lowerFn(t, fn, tt)

	// Bind pattern should create a named local "x" and copy the subject into it.
	foundLocal := false
	for _, local := range mf.Locals {
		if local.Name == "x" {
			foundLocal = true
		}
	}
	if !foundLocal {
		t.Error("bind pattern should create a named local 'x'")
	}

	// Should have a copy instruction for the binding.
	foundCopy := false
	for _, blk := range mf.Blocks {
		for _, instr := range blk.Instrs {
			if instr.Kind == mir.InstrCopy {
				for _, local := range mf.Locals {
					if local.Id == instr.Dest && local.Name == "x" {
						foundCopy = true
					}
				}
			}
		}
	}
	if !foundCopy {
		t.Error("bind pattern should emit a copy of the subject into 'x'")
	}
}

func TestMatchConstructorPattern(t *testing.T) {
	b, tt := makeBuilder()
	enumTy := tt.InternStruct("m", "Color", nil)
	subject := b.Ident(zs, "c", enumTy)
	arms := []hir.MatchArm{
		{
			Pattern: &hir.ConstructorPattern{
				Name: "Red",
				Tag:  0,
				Args: []hir.Pattern{
					&hir.BindPattern{Name: "r", Type: tt.I32},
				},
				Type: enumTy,
			},
			Body: b.Ident(zs, "r", tt.I32),
		},
		{
			Pattern: &hir.WildcardPattern{},
			Body:    b.Literal(zs, "0", tt.I32),
		},
	}
	matchExpr := b.Match(zs, subject, arms, tt.I32)
	block := b.BlockExpr(zs, nil, matchExpr, tt.I32)
	fn := &hir.Function{
		Name: "test", Body: block, ReturnType: tt.I32,
		Params: []hir.Param{{Span: zs, Name: "c", Type: enumTy}},
	}

	mf := lowerFn(t, fn, tt)

	// Constructor pattern should read _tag field.
	foundTagRead := false
	for _, blk := range mf.Blocks {
		for _, instr := range blk.Instrs {
			if instr.Kind == mir.InstrFieldRead && instr.Field == "_tag" {
				foundTagRead = true
			}
		}
	}
	if !foundTagRead {
		t.Error("constructor pattern should emit a _tag field read")
	}

	// Should branch on the tag comparison.
	foundBranch := false
	for _, blk := range mf.Blocks {
		if blk.Term.Kind == mir.TermBranch {
			foundBranch = true
		}
	}
	if !foundBranch {
		t.Error("constructor pattern should emit a branch terminator")
	}

	// Should create a named local "r" from field read _f0.
	foundFieldBind := false
	for _, blk := range mf.Blocks {
		for _, instr := range blk.Instrs {
			if instr.Kind == mir.InstrFieldRead && instr.Field == "_f0" {
				foundFieldBind = true
			}
		}
	}
	if !foundFieldBind {
		t.Error("constructor pattern should emit _f0 field read for bound arg")
	}
}

func TestQuestionLowersToBranch(t *testing.T) {
	b, tt := makeBuilder()
	// Create a Result[I32, Bool] enum type.
	resultTy := tt.InternEnum("std", "Result", []typetable.TypeId{tt.I32, tt.Bool})
	// Build: expr? where expr is an identifier of type Result[I32, Bool].
	expr := b.Ident(zs, "r", resultTy)
	question := b.Question(zs, expr, tt.I32)
	block := b.BlockExpr(zs, nil, question, tt.I32)
	fn := &hir.Function{
		Name: "test", Body: block, ReturnType: tt.I32,
		Params: []hir.Param{{Span: zs, Name: "r", Type: resultTy}},
	}

	mf := lowerFn(t, fn, tt)

	// Should have a branch terminator (for the tag check).
	foundBranch := false
	for _, blk := range mf.Blocks {
		if blk.Term.Kind == mir.TermBranch {
			foundBranch = true
		}
	}
	if !foundBranch {
		t.Error("? operator should produce a branch terminator")
	}

	// Should read the _tag field.
	foundTagRead := false
	for _, blk := range mf.Blocks {
		for _, instr := range blk.Instrs {
			if instr.Kind == mir.InstrFieldRead && instr.Field == "_tag" {
				foundTagRead = true
			}
		}
	}
	if !foundTagRead {
		t.Error("? operator should emit a _tag field read")
	}

	// Should read the _f0 field (extracting the Ok value).
	foundF0Read := false
	for _, blk := range mf.Blocks {
		for _, instr := range blk.Instrs {
			if instr.Kind == mir.InstrFieldRead && instr.Field == "_f0" {
				foundF0Read = true
			}
		}
	}
	if !foundF0Read {
		t.Error("? operator should emit a _f0 field read to extract the inner value")
	}

	// Failure path should have a TermReturn (early return with error).
	foundReturn := false
	for _, blk := range mf.Blocks {
		if blk.Term.Kind == mir.TermReturn {
			foundReturn = true
		}
	}
	if !foundReturn {
		t.Error("? operator failure path should produce a return terminator")
	}
}

// ===== Spawn lowering =====

func TestSpawnLowersToRuntimeCall(t *testing.T) {
	b, tt := makeBuilder()
	// spawn someFunc()
	fn := b.Ident(zs, "someFunc", tt.InternFunc(nil, tt.Unit))
	spawn := b.Spawn(zs, fn)
	block := b.BlockExpr(zs, nil, spawn, tt.Unit)
	hirFn := &hir.Function{Name: "test", Body: block, ReturnType: tt.Unit}

	mf := lowerFn(t, hirFn, tt)

	// The MIR should contain a const instruction with value "fuse_rt_thread_spawn"
	// and a call instruction using it as the callee.
	foundSpawnConst := false
	foundCall := false
	for _, blk := range mf.Blocks {
		for _, instr := range blk.Instrs {
			if instr.Kind == mir.InstrConst && instr.Value == "fuse_rt_thread_spawn" {
				foundSpawnConst = true
			}
			if instr.Kind == mir.InstrCall && len(instr.Args) == 2 {
				foundCall = true
			}
		}
	}
	if !foundSpawnConst {
		t.Error("spawn should emit a const for fuse_rt_thread_spawn")
	}
	if !foundCall {
		t.Error("spawn should emit a call with 2 args (fn, null)")
	}
}

// ===== Closure lowering =====

func TestClosureLoweringProducesLiftedFunction(t *testing.T) {
	b, tt := makeBuilder()
	// Closure: |x: I32| -> I32 { x }
	closureBody := b.BlockExpr(zs, nil, b.Ident(zs, "x", tt.I32), tt.I32)
	closure := b.Closure(zs, []hir.Param{
		{Span: zs, Name: "x", Type: tt.I32},
	}, tt.I32, closureBody, tt.InternFunc([]typetable.TypeId{tt.I32}, tt.I32))
	block := b.BlockExpr(zs, nil, closure, tt.InternFunc([]typetable.TypeId{tt.I32}, tt.I32))
	fn := &hir.Function{Name: "test", Body: block, ReturnType: tt.InternFunc([]typetable.TypeId{tt.I32}, tt.I32)}

	l := New(tt)
	l.LowerFunction(fn)

	if len(l.LiftedFunctions) != 1 {
		t.Fatalf("expected 1 lifted function, got %d", len(l.LiftedFunctions))
	}
	lifted := l.LiftedFunctions[0]
	if lifted.Name != "__closure_0" {
		t.Errorf("lifted function name: %q, want __closure_0", lifted.Name)
	}
}

func TestClosureCapturesOuterVariable(t *testing.T) {
	b, tt := makeBuilder()
	// fn test() {
	//   let y = 10;
	//   let f = |x: I32| -> I32 { x + y };  // captures y
	// }
	closureBody := b.BlockExpr(zs, nil,
		b.Binary(zs, "+", b.Ident(zs, "x", tt.I32), b.Ident(zs, "y", tt.I32), tt.I32),
		tt.I32,
	)
	fnTy := tt.InternFunc([]typetable.TypeId{tt.I32}, tt.I32)
	closure := b.Closure(zs, []hir.Param{
		{Span: zs, Name: "x", Type: tt.I32},
	}, tt.I32, closureBody, fnTy)

	letY := b.Let(zs, "y", tt.I32, b.Literal(zs, "10", tt.I32))
	block := b.BlockExpr(zs, []hir.Stmt{letY}, closure, fnTy)
	fn := &hir.Function{Name: "test", Body: block, ReturnType: fnTy}

	l := New(tt)
	mf := l.LowerFunction(fn)

	if len(l.LiftedFunctions) != 1 {
		t.Fatalf("expected 1 lifted function, got %d", len(l.LiftedFunctions))
	}

	// The outer function should contain an InstrStructInit for the env.
	foundEnvInit := false
	for _, blk := range mf.Blocks {
		for _, instr := range blk.Instrs {
			if instr.Kind == mir.InstrStructInit && instr.Field == "env_0" {
				foundEnvInit = true
				if len(instr.Args) != 1 {
					t.Errorf("env struct init should have 1 captured field, got %d", len(instr.Args))
				}
			}
		}
	}
	if !foundEnvInit {
		t.Error("closure with captures should emit InstrStructInit for env")
	}

	// The lifted function should have a field read for the captured var.
	lifted := l.LiftedFunctions[0]
	foundCapRead := false
	for _, blk := range lifted.Blocks {
		for _, instr := range blk.Instrs {
			if instr.Kind == mir.InstrFieldRead && instr.Field == "_c0" {
				foundCapRead = true
			}
		}
	}
	if !foundCapRead {
		t.Error("lifted function should read captured variable from env via _c0 field")
	}
}

func TestClosureWithNoCaptures(t *testing.T) {
	b, tt := makeBuilder()
	// Closure: |x: I32| -> I32 { x }  — no outer variables referenced
	closureBody := b.BlockExpr(zs, nil, b.Ident(zs, "x", tt.I32), tt.I32)
	fnTy := tt.InternFunc([]typetable.TypeId{tt.I32}, tt.I32)
	closure := b.Closure(zs, []hir.Param{
		{Span: zs, Name: "x", Type: tt.I32},
	}, tt.I32, closureBody, fnTy)
	block := b.BlockExpr(zs, nil, closure, fnTy)
	fn := &hir.Function{Name: "test", Body: block, ReturnType: fnTy}

	l := New(tt)
	mf := l.LowerFunction(fn)

	if len(l.LiftedFunctions) != 1 {
		t.Fatalf("expected 1 lifted function, got %d", len(l.LiftedFunctions))
	}

	// The env struct init should have zero captured fields.
	foundEnvInit := false
	for _, blk := range mf.Blocks {
		for _, instr := range blk.Instrs {
			if instr.Kind == mir.InstrStructInit && instr.Field == "env_0" {
				foundEnvInit = true
				if len(instr.Args) != 0 {
					t.Errorf("env struct init for no-capture closure should have 0 fields, got %d", len(instr.Args))
				}
			}
		}
	}
	if !foundEnvInit {
		t.Error("closure should emit InstrStructInit for env even with no captures")
	}
}

func TestNilBodyDoesNotPanic(t *testing.T) {
	tt := typetable.New()
	fn := &hir.Function{Name: "test", Body: nil, ReturnType: tt.Unit}
	l := New(tt)
	mf := l.LowerFunction(fn)
	if mf == nil {
		t.Fatal("expected non-nil MIR function")
	}
}
