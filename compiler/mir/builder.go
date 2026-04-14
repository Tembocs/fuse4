package mir

import "github.com/Tembocs/fuse4/compiler/typetable"

// Builder constructs MIR functions with centralized block and local management.
type Builder struct {
	Fn        *Function
	nextLocal LocalId
	nextBlock BlockId
	current   BlockId // current block being built
}

// NewBuilder starts building a MIR function.
func NewBuilder(name string, params []Local, retType typetable.TypeId) *Builder {
	b := &Builder{
		Fn: &Function{
			Name:       name,
			Params:     params,
			ReturnType: retType,
		},
		nextLocal: LocalId(len(params)),
	}
	// Copy params as initial locals.
	b.Fn.Locals = append(b.Fn.Locals, params...)

	// Create entry block.
	entry := b.NewBlock()
	b.Fn.EntryBlock = entry
	b.current = entry
	return b
}

// --- locals ---

// NewLocal creates a named local.
func (b *Builder) NewLocal(name string, ty typetable.TypeId) LocalId {
	id := b.nextLocal
	b.nextLocal++
	b.Fn.Locals = append(b.Fn.Locals, Local{Id: id, Name: name, Type: ty})
	return id
}

// NewTemp creates an unnamed temporary.
func (b *Builder) NewTemp(ty typetable.TypeId) LocalId {
	id := b.nextLocal
	b.nextLocal++
	b.Fn.Locals = append(b.Fn.Locals, Local{Id: id, Type: ty})
	return id
}

// --- blocks ---

// NewBlock creates a new basic block and returns its ID.
func (b *Builder) NewBlock() BlockId {
	id := b.nextBlock
	b.nextBlock++
	b.Fn.Blocks = append(b.Fn.Blocks, Block{Id: id})
	return id
}

// CurrentBlock returns the block currently being built.
func (b *Builder) CurrentBlock() BlockId {
	return b.current
}

// SwitchToBlock sets the current block for emitting instructions.
func (b *Builder) SwitchToBlock(id BlockId) {
	b.current = id
}

// IsSealed reports whether the current block already has a terminator.
func (b *Builder) IsSealed() bool {
	return b.Fn.Blocks[b.current].Sealed
}

// --- emit instructions ---

func (b *Builder) emit(instr Instr) {
	blk := &b.Fn.Blocks[b.current]
	if blk.Sealed {
		return // do not add instructions after a terminator (Rule: sealed blocks)
	}
	blk.Instrs = append(blk.Instrs, instr)
}

// EmitConst emits: dest = literal_value
func (b *Builder) EmitConst(dest LocalId, ty typetable.TypeId, value string) {
	b.emit(Instr{Kind: InstrConst, Dest: dest, Type: ty, Value: value})
}

// EmitCopy emits: dest = src
func (b *Builder) EmitCopy(dest, src LocalId, ty typetable.TypeId) {
	b.emit(Instr{Kind: InstrCopy, Dest: dest, Type: ty, Src: src})
}

// EmitMove emits: dest = move src
func (b *Builder) EmitMove(dest, src LocalId, ty typetable.TypeId) {
	b.emit(Instr{Kind: InstrMove, Dest: dest, Type: ty, Src: src})
}

// EmitBorrow emits: dest = ref/mutref src
func (b *Builder) EmitBorrow(dest, src LocalId, ty typetable.TypeId, kind BorrowKind) {
	b.emit(Instr{Kind: InstrBorrow, Dest: dest, Type: ty, Src: src, BorrowKind: kind})
}

// EmitDrop emits: drop(src)
func (b *Builder) EmitDrop(src LocalId) {
	ty := b.Fn.Locals[src].Type
	b.emit(Instr{Kind: InstrDrop, Src: src, Type: ty})
}

// EmitCall emits: dest = callee(args...)
func (b *Builder) EmitCall(dest, callee LocalId, args []LocalId, retType typetable.TypeId, isMethod bool) {
	b.emit(Instr{Kind: InstrCall, Dest: dest, Type: retType, Callee: callee, Args: args, IsMethod: isMethod})
}

// EmitFieldRead emits: dest = src.field
func (b *Builder) EmitFieldRead(dest, src LocalId, field string, ty typetable.TypeId) {
	b.emit(Instr{Kind: InstrFieldRead, Dest: dest, Type: ty, Src: src, Field: field})
}

// EmitFieldAddr emits: dest = &src.field
func (b *Builder) EmitFieldAddr(dest, src LocalId, field string, ty typetable.TypeId) {
	b.emit(Instr{Kind: InstrFieldAddr, Dest: dest, Type: ty, Src: src, Field: field})
}

// EmitIndex emits: dest = src[index]
func (b *Builder) EmitIndex(dest, src, index LocalId, ty typetable.TypeId) {
	b.emit(Instr{Kind: InstrIndex, Dest: dest, Type: ty, Src: src, Src2: index})
}

// EmitBinOp emits: dest = left op right
func (b *Builder) EmitBinOp(dest LocalId, op string, left, right LocalId, ty typetable.TypeId) {
	b.emit(Instr{Kind: InstrBinOp, Dest: dest, Type: ty, Src: left, Src2: right, Op: op})
}

// EmitUnaryOp emits: dest = op operand
func (b *Builder) EmitUnaryOp(dest LocalId, op string, operand LocalId, ty typetable.TypeId) {
	b.emit(Instr{Kind: InstrUnaryOp, Dest: dest, Type: ty, Src: operand, Op: op})
}

// EmitTuple emits: dest = (elems...)
func (b *Builder) EmitTuple(dest LocalId, elems []LocalId, ty typetable.TypeId) {
	b.emit(Instr{Kind: InstrTuple, Dest: dest, Type: ty, Args: elems})
}

// EmitStructInit emits: dest = StructName { fields... }
func (b *Builder) EmitStructInit(dest LocalId, name string, fields []LocalId, ty typetable.TypeId) {
	b.emit(Instr{Kind: InstrStructInit, Dest: dest, Type: ty, Args: fields, Field: name})
}

// EmitEnumInit emits: dest = VariantName(args...)
func (b *Builder) EmitEnumInit(dest LocalId, variant string, args []LocalId, ty typetable.TypeId) {
	b.emit(Instr{Kind: InstrEnumInit, Dest: dest, Type: ty, Args: args, Field: variant})
}

// --- terminators ---

func (b *Builder) seal(term Terminator) {
	blk := &b.Fn.Blocks[b.current]
	if blk.Sealed {
		return
	}
	blk.Term = term
	blk.Sealed = true
}

// TermReturn emits: return value
func (b *Builder) TermReturn(value LocalId) {
	b.seal(Terminator{Kind: TermReturn, Value: value})
}

// TermGoto emits: goto target
func (b *Builder) TermGoto(target BlockId) {
	b.seal(Terminator{Kind: TermGoto, Target: target})
}

// TermBranch emits: if cond goto thenBlock else elseBlock
func (b *Builder) TermBranch(cond LocalId, thenBlock, elseBlock BlockId) {
	b.seal(Terminator{Kind: TermBranch, Value: cond, Target: thenBlock, ElseTarget: elseBlock})
}

// TermDiverge emits: unreachable (no successors)
func (b *Builder) TermDiverge() {
	b.seal(Terminator{Kind: TermDiverge})
}

// Build finalizes and returns the MIR function.
func (b *Builder) Build() *Function {
	return b.Fn
}
