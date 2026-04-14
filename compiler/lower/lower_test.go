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

func TestNilBodyDoesNotPanic(t *testing.T) {
	tt := typetable.New()
	fn := &hir.Function{Name: "test", Body: nil, ReturnType: tt.Unit}
	l := New(tt)
	mf := l.LowerFunction(fn)
	if mf == nil {
		t.Fatal("expected non-nil MIR function")
	}
}
