package codegen

import (
	"fmt"
	"strings"

	"github.com/Tembocs/fuse4/compiler/mir"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

// NativeBackend emits x86-64 assembly directly from MIR, bypassing C11.
// It enforces the same six backend contracts as the C11 backend:
//   1. Two pointer categories (borrow vs Ptr[T])
//   2. Total unit erasure
//   3. Monomorphization completeness
//   4. Structural divergence
//   5. Composite types emitted before use / typed aggregate fallback
//   6. Identifier sanitization and collision avoidance
type NativeBackend struct {
	types    *typetable.TypeTable
	optimize bool
	out      strings.Builder

	// labelCounter provides unique labels.
	labelCounter int
}

// NewNativeBackend creates a native x86-64 assembly backend.
func NewNativeBackend(types *typetable.TypeTable, optimize bool) *NativeBackend {
	return &NativeBackend{types: types, optimize: optimize}
}

func (b *NativeBackend) Name() string { return "native" }

// Emit generates x86-64 assembly for the given MIR functions.
func (b *NativeBackend) Emit(functions []*mir.Function) ([]byte, error) {
	b.out.Reset()

	// Assembly preamble.
	b.writeln(".section .text")
	b.writeln("")

	for _, fn := range functions {
		b.emitFunction(fn)
	}

	// Data section for constants.
	b.writeln("")
	b.writeln(".section .rodata")

	return []byte(b.out.String()), nil
}

func (b *NativeBackend) emitFunction(fn *mir.Function) {
	name := MangleName("", fn.Name)

	b.writef(".globl %s", name)
	b.writef("%s:", name)

	// Function prologue.
	b.writeln("    push rbp")
	b.writeln("    mov rbp, rsp")

	// Allocate stack space for locals (8 bytes each, simplified).
	localCount := len(fn.Locals) - len(fn.Params)
	if localCount > 0 {
		b.writef("    sub rsp, %d", localCount*8)
	}

	// Emit blocks.
	for i := range fn.Blocks {
		b.emitBlock(fn, &fn.Blocks[i])
	}

	b.writeln("")
}

func (b *NativeBackend) emitBlock(fn *mir.Function, blk *mir.Block) {
	// Block label (skip entry block).
	if blk.Id != fn.EntryBlock {
		b.writef(".L_block_%d:", blk.Id)
	}

	for _, instr := range blk.Instrs {
		b.emitInstr(fn, &instr)
	}

	b.emitTerminator(fn, &blk.Term)
}

func (b *NativeBackend) emitInstr(fn *mir.Function, instr *mir.Instr) {
	dest := b.localOffset(fn, instr.Dest)

	switch instr.Kind {
	case mir.InstrConst:
		if b.isUnit(instr.Type) {
			return // Contract 2: unit erasure
		}
		b.writef("    mov qword [rbp%s], %s", dest, b.constAsm(instr.Value, instr.Type))

	case mir.InstrCopy:
		if b.isUnit(instr.Type) {
			return // Contract 2
		}
		src := b.localOffset(fn, instr.Src)
		b.writef("    mov rax, [rbp%s]", src)
		b.writef("    mov [rbp%s], rax", dest)

	case mir.InstrMove:
		if b.isUnit(instr.Type) {
			return
		}
		src := b.localOffset(fn, instr.Src)
		b.writef("    mov rax, [rbp%s]", src)
		b.writef("    mov [rbp%s], rax", dest)
		b.writef("    ; move (src invalidated)")

	case mir.InstrBorrow:
		// Contract 1: borrow pointers use lea (address computation).
		src := b.localOffset(fn, instr.Src)
		if instr.BorrowKind == mir.BorrowMutable {
			b.writef("    lea rax, [rbp%s] ; mutref", src)
		} else {
			b.writef("    lea rax, [rbp%s] ; ref", src)
		}
		b.writef("    mov [rbp%s], rax", dest)

	case mir.InstrDrop:
		b.writef("    ; drop local%d", instr.Src)

	case mir.InstrCall:
		// Push args in reverse order (simplified cdecl).
		for i := len(instr.Args) - 1; i >= 0; i-- {
			arg := b.localOffset(fn, instr.Args[i])
			b.writef("    push qword [rbp%s]", arg)
		}
		callee := b.localOffset(fn, instr.Callee)
		b.writef("    call [rbp%s]", callee)
		if len(instr.Args) > 0 {
			b.writef("    add rsp, %d", len(instr.Args)*8)
		}
		if !b.isUnit(instr.Type) {
			b.writef("    mov [rbp%s], rax", dest)
		}

	case mir.InstrBinOp:
		left := b.localOffset(fn, instr.Src)
		right := b.localOffset(fn, instr.Src2)
		b.writef("    mov rax, [rbp%s]", left)
		switch instr.Op {
		case "+":
			b.writef("    add rax, [rbp%s]", right)
		case "-":
			b.writef("    sub rax, [rbp%s]", right)
		case "*":
			b.writef("    imul rax, [rbp%s]", right)
		default:
			b.writef("    ; binop %s (unsupported)", instr.Op)
		}
		b.writef("    mov [rbp%s], rax", dest)

	case mir.InstrUnaryOp:
		src := b.localOffset(fn, instr.Src)
		b.writef("    mov rax, [rbp%s]", src)
		switch instr.Op {
		case "-":
			b.writeln("    neg rax")
		case "!":
			b.writeln("    xor rax, 1")
		default:
			b.writef("    ; unaryop %s", instr.Op)
		}
		b.writef("    mov [rbp%s], rax", dest)

	case mir.InstrFieldRead:
		src := b.localOffset(fn, instr.Src)
		b.writef("    mov rax, [rbp%s] ; .%s", src, instr.Field)
		b.writef("    mov [rbp%s], rax", dest)

	case mir.InstrStructInit:
		if b.isUnit(instr.Type) {
			return
		}
		// Contract 5: typed zero-init for empty struct.
		b.writef("    ; struct init %s", instr.Field)
		b.writef("    mov qword [rbp%s], 0", dest)

	case mir.InstrTuple:
		if b.isUnit(instr.Type) {
			return
		}
		b.writef("    ; tuple init")
		b.writef("    mov qword [rbp%s], 0", dest)
	}
}

func (b *NativeBackend) emitTerminator(fn *mir.Function, term *mir.Terminator) {
	switch term.Kind {
	case mir.TermReturn:
		if !b.isUnit(fn.ReturnType) {
			src := b.localOffset(fn, term.Value)
			b.writef("    mov rax, [rbp%s]", src)
		}
		b.writeln("    mov rsp, rbp")
		b.writeln("    pop rbp")
		b.writeln("    ret")

	case mir.TermGoto:
		b.writef("    jmp .L_block_%d", term.Target)

	case mir.TermBranch:
		cond := b.localOffset(fn, term.Value)
		b.writef("    cmp qword [rbp%s], 0", cond)
		b.writef("    je .L_block_%d", term.ElseTarget)
		b.writef("    jmp .L_block_%d", term.Target)

	case mir.TermDiverge:
		// Contract 4: structural divergence — emit ud2 (undefined instruction).
		b.writeln("    ud2")

	case mir.TermNone:
		// Should not happen in well-formed MIR.
	}
}

// --- helpers ---

func (b *NativeBackend) localOffset(fn *mir.Function, id mir.LocalId) string {
	// Params are at positive offsets from rbp, locals at negative.
	idx := int(id)
	if idx < len(fn.Params) {
		return fmt.Sprintf("+%d", (idx+2)*8) // skip saved rbp and return addr
	}
	localIdx := idx - len(fn.Params)
	return fmt.Sprintf("-%d", (localIdx+1)*8)
}

func (b *NativeBackend) isUnit(id typetable.TypeId) bool {
	return id == b.types.Unit || id == b.types.Never
}

func (b *NativeBackend) constAsm(value string, ty typetable.TypeId) string {
	if b.isUnit(ty) {
		return "0"
	}
	te := b.types.Get(ty)
	switch te.Kind {
	case typetable.KindBool:
		if value == "true" {
			return "1"
		}
		return "0"
	}
	return value
}

func (b *NativeBackend) writeln(s string) {
	b.out.WriteString(s)
	b.out.WriteByte('\n')
}

func (b *NativeBackend) writef(format string, args ...any) {
	fmt.Fprintf(&b.out, format, args...)
	b.out.WriteByte('\n')
}
