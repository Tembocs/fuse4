package codegen

import (
	"strings"
	"testing"

	"github.com/Tembocs/fuse4/compiler/mir"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

// ===== Backend interface =====

func TestBackendInterfaceC11(t *testing.T) {
	tt := typetable.New()
	b := NewBackend(BackendConfig{Target: "c11", Types: tt})
	if b.Name() != "c11" {
		t.Errorf("name: %q", b.Name())
	}
}

func TestBackendInterfaceNative(t *testing.T) {
	tt := typetable.New()
	b := NewBackend(BackendConfig{Target: "native", Types: tt})
	if b.Name() != "native" {
		t.Errorf("name: %q", b.Name())
	}
}

func TestDefaultBackendIsC11(t *testing.T) {
	tt := typetable.New()
	b := NewBackend(BackendConfig{Target: "", Types: tt})
	if b.Name() != "c11" {
		t.Errorf("default should be c11, got %q", b.Name())
	}
}

// ===== Native backend output =====

func TestNativeEmitSimpleFunction(t *testing.T) {
	tt := typetable.New()
	fn := buildSimpleFn(tt, "test", tt.I32, func(b *mir.Builder) {
		tmp := b.NewTemp(tt.I32)
		b.EmitConst(tmp, tt.I32, "42")
		b.TermReturn(tmp)
	})

	backend := NewNativeBackend(tt, false)
	out, err := backend.Emit([]*mir.Function{fn})
	if err != nil {
		t.Fatal(err)
	}
	asm := string(out)

	if !strings.Contains(asm, ".globl Fuse_test") {
		t.Error("should emit global symbol")
	}
	if !strings.Contains(asm, "Fuse_test:") {
		t.Error("should emit function label")
	}
	if !strings.Contains(asm, "push rbp") {
		t.Error("should emit prologue")
	}
	if !strings.Contains(asm, "ret") {
		t.Error("should emit ret")
	}
}

// ===== Contract 1: Borrow uses lea =====

func TestNativeBorrowUsesLea(t *testing.T) {
	tt := typetable.New()
	refTy := tt.InternRef(tt.I32)
	fn := buildSimpleFn(tt, "test", refTy, func(b *mir.Builder) {
		src := b.NewLocal("x", tt.I32)
		b.EmitConst(src, tt.I32, "1")
		dest := b.NewTemp(refTy)
		b.EmitBorrow(dest, src, refTy, mir.BorrowShared)
		b.TermReturn(dest)
	})

	backend := NewNativeBackend(tt, false)
	out, _ := backend.Emit([]*mir.Function{fn})
	asm := string(out)

	if !strings.Contains(asm, "lea") {
		t.Error("Contract 1: borrow should use lea (address computation)")
	}
	if !strings.Contains(asm, "; ref") {
		t.Error("borrow should be annotated as ref")
	}
}

func TestNativeMutrefBorrow(t *testing.T) {
	tt := typetable.New()
	mrefTy := tt.InternMutRef(tt.I32)
	fn := buildSimpleFn(tt, "test", mrefTy, func(b *mir.Builder) {
		src := b.NewLocal("x", tt.I32)
		b.EmitConst(src, tt.I32, "1")
		dest := b.NewTemp(mrefTy)
		b.EmitBorrow(dest, src, mrefTy, mir.BorrowMutable)
		b.TermReturn(dest)
	})

	backend := NewNativeBackend(tt, false)
	out, _ := backend.Emit([]*mir.Function{fn})
	if !strings.Contains(string(out), "; mutref") {
		t.Error("Contract 1: should annotate mutref")
	}
}

// ===== Contract 2: Unit erasure =====

func TestNativeUnitErasure(t *testing.T) {
	tt := typetable.New()
	fn := buildSimpleFn(tt, "test", tt.Unit, func(b *mir.Builder) {
		tmp := b.NewTemp(tt.Unit)
		b.EmitConst(tmp, tt.Unit, "()")
		b.TermReturn(tmp)
	})

	backend := NewNativeBackend(tt, false)
	out, _ := backend.Emit([]*mir.Function{fn})
	asm := string(out)

	// Unit const should not generate a mov instruction.
	lines := strings.Split(asm, "\n")
	for _, line := range lines {
		if strings.Contains(line, "mov") && strings.Contains(line, "()") {
			t.Error("Contract 2: unit value should be erased")
		}
	}
}

// ===== Contract 4: Divergence =====

func TestNativeDivergenceEmitsUd2(t *testing.T) {
	tt := typetable.New()
	fn := buildSimpleFn(tt, "test", tt.Never, func(b *mir.Builder) {
		b.TermDiverge()
	})

	backend := NewNativeBackend(tt, false)
	out, _ := backend.Emit([]*mir.Function{fn})
	if !strings.Contains(string(out), "ud2") {
		t.Error("Contract 4: divergence should emit ud2")
	}
}

// ===== Control flow =====

func TestNativeBranchEmitsJumps(t *testing.T) {
	tt := typetable.New()
	fn := buildSimpleFn(tt, "test", tt.I32, func(b *mir.Builder) {
		cond := b.NewTemp(tt.Bool)
		b.EmitConst(cond, tt.Bool, "true")
		thenBlk := b.NewBlock()
		elseBlk := b.NewBlock()
		b.TermBranch(cond, thenBlk, elseBlk)

		b.SwitchToBlock(thenBlk)
		t1 := b.NewTemp(tt.I32)
		b.EmitConst(t1, tt.I32, "1")
		b.TermReturn(t1)

		b.SwitchToBlock(elseBlk)
		t2 := b.NewTemp(tt.I32)
		b.EmitConst(t2, tt.I32, "2")
		b.TermReturn(t2)
	})

	backend := NewNativeBackend(tt, false)
	out, _ := backend.Emit([]*mir.Function{fn})
	asm := string(out)

	if !strings.Contains(asm, "je") {
		t.Error("branch should emit conditional jump")
	}
	if !strings.Contains(asm, "jmp") {
		t.Error("branch should emit unconditional jump")
	}
}

func TestNativeGotoEmitsJmp(t *testing.T) {
	tt := typetable.New()
	fn := buildSimpleFn(tt, "test", tt.Unit, func(b *mir.Builder) {
		target := b.NewBlock()
		b.TermGoto(target)
		b.SwitchToBlock(target)
		tmp := b.NewTemp(tt.Unit)
		b.EmitConst(tmp, tt.Unit, "()")
		b.TermReturn(tmp)
	})

	backend := NewNativeBackend(tt, false)
	out, _ := backend.Emit([]*mir.Function{fn})
	if !strings.Contains(string(out), "jmp .L_block_") {
		t.Error("goto should emit jmp")
	}
}

// ===== Binary ops =====

func TestNativeBinOpAdd(t *testing.T) {
	tt := typetable.New()
	fn := buildSimpleFn(tt, "test", tt.I32, func(b *mir.Builder) {
		a := b.NewTemp(tt.I32)
		b.EmitConst(a, tt.I32, "1")
		bv := b.NewTemp(tt.I32)
		b.EmitConst(bv, tt.I32, "2")
		c := b.NewTemp(tt.I32)
		b.EmitBinOp(c, "+", a, bv, tt.I32)
		b.TermReturn(c)
	})

	backend := NewNativeBackend(tt, false)
	out, _ := backend.Emit([]*mir.Function{fn})
	if !strings.Contains(string(out), "add rax") {
		t.Error("+ should emit add instruction")
	}
}

// ===== Stage2 through native path =====

func TestNativeBackendProducesOutput(t *testing.T) {
	tt := typetable.New()
	fns := []*mir.Function{
		buildSimpleFn(tt, "main", tt.I32, func(b *mir.Builder) {
			tmp := b.NewTemp(tt.I32)
			b.EmitConst(tmp, tt.I32, "0")
			b.TermReturn(tmp)
		}),
	}

	backend := NewNativeBackend(tt, false)
	out, err := backend.Emit(fns)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Error("native backend should produce non-empty output")
	}
	if !strings.Contains(string(out), ".section .text") {
		t.Error("should have text section")
	}
}

// ===== Both backends produce output for same MIR =====

func TestBothBackendsEmitSameMIR(t *testing.T) {
	tt := typetable.New()
	fn := buildSimpleFn(tt, "test", tt.I32, func(b *mir.Builder) {
		tmp := b.NewTemp(tt.I32)
		b.EmitConst(tmp, tt.I32, "42")
		b.TermReturn(tmp)
	})
	fns := []*mir.Function{fn}

	c11 := NewC11Backend(tt)
	c11Out, _ := c11.Emit(fns)

	native := NewNativeBackend(tt, false)
	nativeOut, _ := native.Emit(fns)

	if len(c11Out) == 0 {
		t.Error("c11 output empty")
	}
	if len(nativeOut) == 0 {
		t.Error("native output empty")
	}
	// Both should contain the function name.
	if !strings.Contains(string(c11Out), "Fuse_test") {
		t.Error("c11 should contain function name")
	}
	if !strings.Contains(string(nativeOut), "Fuse_test") {
		t.Error("native should contain function name")
	}
}
