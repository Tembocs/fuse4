package mir

import (
	"testing"

	"github.com/Tembocs/fuse4/compiler/typetable"
)

// ===== Builder construction =====

func TestNewBuilderCreatesEntryBlock(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("main", nil, tt.Unit)
	fn := b.Build()
	if len(fn.Blocks) != 1 {
		t.Fatalf("expected 1 block, got %d", len(fn.Blocks))
	}
	if fn.EntryBlock != 0 {
		t.Errorf("entry block should be 0, got %d", fn.EntryBlock)
	}
}

func TestNewBuilderCopiesParams(t *testing.T) {
	tt := typetable.New()
	params := []Local{
		{Id: 0, Name: "a", Type: tt.I32},
		{Id: 1, Name: "b", Type: tt.Bool},
	}
	b := NewBuilder("add", params, tt.I32)
	fn := b.Build()
	if len(fn.Locals) != 2 {
		t.Fatalf("expected 2 locals, got %d", len(fn.Locals))
	}
	if fn.Locals[0].Name != "a" || fn.Locals[1].Name != "b" {
		t.Error("param locals not copied correctly")
	}
}

func TestBuilderFunctionName(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("foo", nil, tt.Unit)
	if b.Build().Name != "foo" {
		t.Error("function name mismatch")
	}
}

func TestBuilderReturnType(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("get", nil, tt.I64)
	if b.Build().ReturnType != tt.I64 {
		t.Error("return type mismatch")
	}
}

// ===== Local allocation =====

func TestNewLocalAllocatesSequentialIds(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	x := b.NewLocal("x", tt.I32)
	y := b.NewLocal("y", tt.Bool)
	if x != 0 {
		t.Errorf("first local id: got %d, want 0", x)
	}
	if y != 1 {
		t.Errorf("second local id: got %d, want 1", y)
	}
}

func TestNewLocalWithParamsStartsAfterParams(t *testing.T) {
	tt := typetable.New()
	params := []Local{{Id: 0, Name: "a", Type: tt.I32}, {Id: 1, Name: "b", Type: tt.I32}}
	b := NewBuilder("f", params, tt.I32)
	x := b.NewLocal("x", tt.Bool)
	if x != 2 {
		t.Errorf("local after 2 params should have id 2, got %d", x)
	}
}

func TestNewTempHasEmptyName(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	tmp := b.NewTemp(tt.I32)
	fn := b.Build()
	if fn.Locals[tmp].Name != "" {
		t.Errorf("temp should have empty name, got %q", fn.Locals[tmp].Name)
	}
}

func TestNewLocalHasCorrectType(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	x := b.NewLocal("x", tt.F64)
	fn := b.Build()
	if fn.Locals[x].Type != tt.F64 {
		t.Errorf("local type: got %d, want %d", fn.Locals[x].Type, tt.F64)
	}
}

// ===== Block management =====

func TestNewBlockAllocatesSequentialIds(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	// entry block is 0
	b1 := b.NewBlock()
	b2 := b.NewBlock()
	if b1 != 1 {
		t.Errorf("second block id: got %d, want 1", b1)
	}
	if b2 != 2 {
		t.Errorf("third block id: got %d, want 2", b2)
	}
}

func TestCurrentBlockIsEntryByDefault(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	if b.CurrentBlock() != 0 {
		t.Errorf("current block should be entry (0), got %d", b.CurrentBlock())
	}
}

func TestSwitchToBlock(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	blk := b.NewBlock()
	b.SwitchToBlock(blk)
	if b.CurrentBlock() != blk {
		t.Error("SwitchToBlock did not update current block")
	}
}

// ===== Instruction emission =====

func TestEmitConst(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	dest := b.NewLocal("x", tt.I32)
	b.EmitConst(dest, tt.I32, "42")
	fn := b.Build()
	instrs := fn.Blocks[0].Instrs
	if len(instrs) != 1 {
		t.Fatalf("expected 1 instruction, got %d", len(instrs))
	}
	instr := instrs[0]
	if instr.Kind != InstrConst {
		t.Errorf("kind: got %d, want InstrConst", instr.Kind)
	}
	if instr.Dest != dest {
		t.Errorf("dest: got %d, want %d", instr.Dest, dest)
	}
	if instr.Value != "42" {
		t.Errorf("value: got %q, want %q", instr.Value, "42")
	}
	if instr.Type != tt.I32 {
		t.Errorf("type: got %d, want %d", instr.Type, tt.I32)
	}
}

func TestEmitCopy(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	x := b.NewLocal("x", tt.I32)
	y := b.NewLocal("y", tt.I32)
	b.EmitCopy(y, x, tt.I32)
	instr := b.Build().Blocks[0].Instrs[0]
	if instr.Kind != InstrCopy || instr.Dest != y || instr.Src != x {
		t.Error("copy instruction fields")
	}
}

func TestEmitMove(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	x := b.NewLocal("x", tt.I32)
	y := b.NewLocal("y", tt.I32)
	b.EmitMove(y, x, tt.I32)
	instr := b.Build().Blocks[0].Instrs[0]
	if instr.Kind != InstrMove || instr.Dest != y || instr.Src != x {
		t.Error("move instruction fields")
	}
}

func TestEmitBorrowShared(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	x := b.NewLocal("x", tt.I32)
	r := b.NewLocal("r", tt.InternRef(tt.I32))
	b.EmitBorrow(r, x, tt.InternRef(tt.I32), BorrowShared)
	instr := b.Build().Blocks[0].Instrs[0]
	if instr.Kind != InstrBorrow || instr.BorrowKind != BorrowShared {
		t.Error("shared borrow")
	}
}

func TestEmitBorrowMutable(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	x := b.NewLocal("x", tt.I32)
	m := b.NewLocal("m", tt.InternMutRef(tt.I32))
	b.EmitBorrow(m, x, tt.InternMutRef(tt.I32), BorrowMutable)
	instr := b.Build().Blocks[0].Instrs[0]
	if instr.Kind != InstrBorrow || instr.BorrowKind != BorrowMutable {
		t.Error("mutable borrow")
	}
}

func TestEmitDrop(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	x := b.NewLocal("x", tt.I32)
	b.EmitDrop(x)
	instr := b.Build().Blocks[0].Instrs[0]
	if instr.Kind != InstrDrop || instr.Src != x {
		t.Error("drop instruction")
	}
}

func TestEmitCall(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	callee := b.NewLocal("fn", tt.InternFunc([]typetable.TypeId{tt.I32}, tt.Bool))
	arg := b.NewLocal("a", tt.I32)
	dest := b.NewTemp(tt.Bool)
	b.EmitCall(dest, callee, []LocalId{arg}, tt.Bool, false)
	instr := b.Build().Blocks[0].Instrs[0]
	if instr.Kind != InstrCall {
		t.Error("call kind")
	}
	if instr.Callee != callee {
		t.Error("callee")
	}
	if len(instr.Args) != 1 || instr.Args[0] != arg {
		t.Error("call args")
	}
	if instr.IsMethod {
		t.Error("should not be method call")
	}
}

func TestEmitMethodCall(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	callee := b.NewLocal("method", tt.InternFunc(nil, tt.Unit))
	self := b.NewLocal("self", tt.I32)
	dest := b.NewTemp(tt.Unit)
	b.EmitCall(dest, callee, []LocalId{self}, tt.Unit, true)
	instr := b.Build().Blocks[0].Instrs[0]
	if !instr.IsMethod {
		t.Error("should be method call")
	}
}

func TestEmitFieldRead(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	src := b.NewLocal("p", tt.InternStruct("mod", "Point", nil))
	dest := b.NewTemp(tt.I32)
	b.EmitFieldRead(dest, src, "x", tt.I32)
	instr := b.Build().Blocks[0].Instrs[0]
	if instr.Kind != InstrFieldRead || instr.Field != "x" || instr.Src != src {
		t.Error("field read")
	}
}

func TestEmitFieldAddr(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	src := b.NewLocal("p", tt.InternStruct("mod", "Point", nil))
	dest := b.NewTemp(tt.InternRef(tt.I32))
	b.EmitFieldAddr(dest, src, "y", tt.InternRef(tt.I32))
	instr := b.Build().Blocks[0].Instrs[0]
	if instr.Kind != InstrFieldAddr || instr.Field != "y" {
		t.Error("field addr")
	}
}

func TestEmitIndex(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	arr := b.NewLocal("arr", tt.InternSlice(tt.I32))
	idx := b.NewLocal("i", tt.USize)
	dest := b.NewTemp(tt.I32)
	b.EmitIndex(dest, arr, idx, tt.I32)
	instr := b.Build().Blocks[0].Instrs[0]
	if instr.Kind != InstrIndex || instr.Src != arr || instr.Src2 != idx {
		t.Error("index instruction")
	}
}

func TestEmitBinOp(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	left := b.NewLocal("a", tt.I32)
	right := b.NewLocal("b", tt.I32)
	dest := b.NewTemp(tt.I32)
	b.EmitBinOp(dest, "+", left, right, tt.I32)
	instr := b.Build().Blocks[0].Instrs[0]
	if instr.Kind != InstrBinOp || instr.Op != "+" || instr.Src != left || instr.Src2 != right {
		t.Error("binop instruction")
	}
}

func TestEmitUnaryOp(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	x := b.NewLocal("x", tt.I32)
	dest := b.NewTemp(tt.I32)
	b.EmitUnaryOp(dest, "-", x, tt.I32)
	instr := b.Build().Blocks[0].Instrs[0]
	if instr.Kind != InstrUnaryOp || instr.Op != "-" || instr.Src != x {
		t.Error("unary op instruction")
	}
}

func TestEmitTuple(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	a := b.NewLocal("a", tt.I32)
	bLocal := b.NewLocal("b", tt.Bool)
	dest := b.NewTemp(tt.InternTuple([]typetable.TypeId{tt.I32, tt.Bool}))
	b.EmitTuple(dest, []LocalId{a, bLocal}, tt.InternTuple([]typetable.TypeId{tt.I32, tt.Bool}))
	instr := b.Build().Blocks[0].Instrs[0]
	if instr.Kind != InstrTuple || len(instr.Args) != 2 {
		t.Error("tuple instruction")
	}
}

func TestEmitStructInit(t *testing.T) {
	tt := typetable.New()
	pointTy := tt.InternStruct("mod", "Point", nil)
	b := NewBuilder("f", nil, tt.Unit)
	x := b.NewLocal("x", tt.I32)
	y := b.NewLocal("y", tt.I32)
	dest := b.NewTemp(pointTy)
	b.EmitStructInit(dest, "Point", []LocalId{x, y}, pointTy)
	instr := b.Build().Blocks[0].Instrs[0]
	if instr.Kind != InstrStructInit || instr.Field != "Point" || len(instr.Args) != 2 {
		t.Error("struct init instruction")
	}
}

func TestEmitEnumInit(t *testing.T) {
	tt := typetable.New()
	optTy := tt.InternEnum("core", "Option", nil)
	b := NewBuilder("f", nil, tt.Unit)
	val := b.NewLocal("v", tt.I32)
	dest := b.NewTemp(optTy)
	b.EmitEnumInit(dest, "Some", []LocalId{val}, optTy)
	instr := b.Build().Blocks[0].Instrs[0]
	if instr.Kind != InstrEnumInit || instr.Field != "Some" || len(instr.Args) != 1 {
		t.Error("enum init instruction")
	}
}

// ===== Terminators =====

func TestTermReturn(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.I32)
	x := b.NewLocal("x", tt.I32)
	b.EmitConst(x, tt.I32, "0")
	b.TermReturn(x)
	fn := b.Build()
	term := fn.Blocks[0].Term
	if term.Kind != TermReturn {
		t.Errorf("term kind: got %d, want TermReturn", term.Kind)
	}
	if term.Value != x {
		t.Errorf("return value: got %d, want %d", term.Value, x)
	}
}

func TestTermGoto(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	target := b.NewBlock()
	b.TermGoto(target)
	term := b.Build().Blocks[0].Term
	if term.Kind != TermGoto || term.Target != target {
		t.Error("goto terminator")
	}
}

func TestTermBranch(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	cond := b.NewLocal("c", tt.Bool)
	thenBlk := b.NewBlock()
	elseBlk := b.NewBlock()
	b.TermBranch(cond, thenBlk, elseBlk)
	term := b.Build().Blocks[0].Term
	if term.Kind != TermBranch {
		t.Error("branch kind")
	}
	if term.Value != cond {
		t.Error("branch condition")
	}
	if term.Target != thenBlk || term.ElseTarget != elseBlk {
		t.Error("branch targets")
	}
}

func TestTermDiverge(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	b.TermDiverge()
	term := b.Build().Blocks[0].Term
	if term.Kind != TermDiverge {
		t.Error("diverge terminator")
	}
}

// ===== Block sealing =====

func TestBlockSealedAfterTerminator(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	if b.IsSealed() {
		t.Error("block should not be sealed initially")
	}
	b.TermDiverge()
	if !b.IsSealed() {
		t.Error("block should be sealed after terminator")
	}
}

func TestSealedBlockIgnoresInstructions(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	x := b.NewLocal("x", tt.I32)
	b.TermDiverge()
	// This should be silently dropped
	b.EmitConst(x, tt.I32, "42")
	fn := b.Build()
	if len(fn.Blocks[0].Instrs) != 0 {
		t.Error("sealed block should not accept new instructions")
	}
}

func TestSealedBlockIgnoresSecondTerminator(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.Unit)
	target := b.NewBlock()
	b.TermGoto(target)
	// Second terminator should be ignored
	b.TermDiverge()
	term := b.Build().Blocks[0].Term
	if term.Kind != TermGoto {
		t.Error("first terminator should win")
	}
}

// ===== Multi-block function =====

func TestMultiBlockFunction(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.I32)

	// Entry block: branch on condition
	cond := b.NewLocal("c", tt.Bool)
	b.EmitConst(cond, tt.Bool, "true")
	thenBlk := b.NewBlock()
	elseBlk := b.NewBlock()
	b.TermBranch(cond, thenBlk, elseBlk)

	// Then block: return 1
	b.SwitchToBlock(thenBlk)
	x := b.NewLocal("x", tt.I32)
	b.EmitConst(x, tt.I32, "1")
	b.TermReturn(x)

	// Else block: return 0
	b.SwitchToBlock(elseBlk)
	y := b.NewLocal("y", tt.I32)
	b.EmitConst(y, tt.I32, "0")
	b.TermReturn(y)

	fn := b.Build()
	if len(fn.Blocks) != 3 {
		t.Fatalf("expected 3 blocks, got %d", len(fn.Blocks))
	}
	// All blocks should be sealed
	for i, blk := range fn.Blocks {
		if !blk.Sealed {
			t.Errorf("block %d should be sealed", i)
		}
	}
}

// ===== Instruction kinds are distinct =====

func TestInstrKindsAreDistinct(t *testing.T) {
	kinds := []InstrKind{
		InstrConst, InstrCopy, InstrMove, InstrBorrow, InstrDrop,
		InstrCall, InstrFieldRead, InstrFieldAddr, InstrIndex,
		InstrBinOp, InstrUnaryOp, InstrTuple, InstrStructInit,
		InstrEnumInit, InstrCast,
	}
	seen := make(map[InstrKind]bool)
	for _, k := range kinds {
		if seen[k] {
			t.Errorf("duplicate InstrKind: %d", k)
		}
		seen[k] = true
	}
}

func TestTermKindsAreDistinct(t *testing.T) {
	kinds := []TermKind{
		TermNone, TermReturn, TermGoto, TermBranch, TermSwitch, TermDiverge,
	}
	seen := make(map[TermKind]bool)
	for _, k := range kinds {
		if seen[k] {
			t.Errorf("duplicate TermKind: %d", k)
		}
		seen[k] = true
	}
}

func TestBorrowKindsAreDistinct(t *testing.T) {
	if BorrowShared == BorrowMutable {
		t.Error("BorrowShared and BorrowMutable should be distinct")
	}
}

// ===== Multiple instructions in one block =====

func TestMultipleInstructionsInBlock(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder("f", nil, tt.I32)
	x := b.NewLocal("x", tt.I32)
	y := b.NewLocal("y", tt.I32)
	z := b.NewLocal("z", tt.I32)
	b.EmitConst(x, tt.I32, "1")
	b.EmitConst(y, tt.I32, "2")
	b.EmitBinOp(z, "+", x, y, tt.I32)
	b.TermReturn(z)

	fn := b.Build()
	if len(fn.Blocks[0].Instrs) != 3 {
		t.Fatalf("expected 3 instructions, got %d", len(fn.Blocks[0].Instrs))
	}
	if fn.Blocks[0].Instrs[0].Kind != InstrConst {
		t.Error("first instr should be const")
	}
	if fn.Blocks[0].Instrs[2].Kind != InstrBinOp {
		t.Error("third instr should be binop")
	}
}
