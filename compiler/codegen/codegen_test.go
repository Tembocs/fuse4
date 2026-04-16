package codegen

import (
	"strings"
	"testing"

	"github.com/Tembocs/fuse4/compiler/mir"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

// --- helpers ---

func emitOne(t *testing.T, fn *mir.Function, tt *typetable.TypeTable) string {
	t.Helper()
	e := NewEmitter(tt)
	src := e.Emit([]*mir.Function{fn})
	for _, err := range e.Errors {
		t.Errorf("emit error: %s", err)
	}
	return src
}

func emitOneWithDropTypes(t *testing.T, fn *mir.Function, tt *typetable.TypeTable, dropTypes map[typetable.TypeId]bool) string {
	t.Helper()
	e := NewEmitter(tt)
	for k, v := range dropTypes {
		e.DropTypes[k] = v
	}
	src := e.Emit([]*mir.Function{fn})
	for _, err := range e.Errors {
		t.Errorf("emit error: %s", err)
	}
	return src
}

func buildSimpleFn(tt *typetable.TypeTable, name string, retType typetable.TypeId, setup func(b *mir.Builder)) *mir.Function {
	b := mir.NewBuilder(name, nil, retType)
	setup(b)
	return b.Build()
}

// ===== Phase 01: Identifier sanitization (Contract 6) =====

func TestSanitizeCKeywords(t *testing.T) {
	cases := []struct{ in, want string }{
		{"int", "fuse_int"},
		{"return", "fuse_return"},
		{"static", "fuse_static"},
		{"foo", "foo"},
	}
	for _, tc := range cases {
		got := SanitizeIdent(tc.in)
		if got != tc.want {
			t.Errorf("SanitizeIdent(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestSanitizeNumericFieldNames(t *testing.T) {
	got := SanitizeIdent("0")
	if got != "f_0" {
		t.Errorf("SanitizeIdent(\"0\") = %q, want \"f_0\"", got)
	}
}

func TestSanitizeDots(t *testing.T) {
	got := SanitizeIdent("core.list")
	if got != "core_list" {
		t.Errorf("got %q", got)
	}
}

// ===== Phase 01: Module-qualified mangling (Contract 6) =====

func TestMangleNameModuleQualified(t *testing.T) {
	a := MangleName("mod_a", "Point")
	b := MangleName("mod_b", "Point")
	if a == b {
		t.Error("same-name items from different modules must have different mangled names")
	}
	if !strings.HasPrefix(a, "Fuse_") {
		t.Errorf("mangled name should start with Fuse_: %q", a)
	}
}

func TestMangleNameNoModule(t *testing.T) {
	got := MangleName("", "main")
	if got != "main" {
		t.Errorf("got %q", got)
	}
}

// ===== Phase 01: Type emission before use (Contract 5) =====

func TestCompositeTypesEmittedBeforeFunctions(t *testing.T) {
	tt := typetable.New()
	sty := tt.InternStruct("m", "Point", nil)
	fn := buildSimpleFn(tt, "test", sty, func(b *mir.Builder) {
		tmp := b.NewTemp(sty)
		b.EmitStructInit(tmp, "Point", nil, sty)
		b.TermReturn(tmp)
	})

	src := emitOne(t, fn, tt)
	typedefIdx := strings.Index(src, "typedef struct Fuse_m__Point")
	fnIdx := strings.Index(src, "Fuse_test(")
	if typedefIdx < 0 {
		t.Fatal("struct typedef not found in output")
	}
	if fnIdx < 0 {
		t.Fatal("function not found in output")
	}
	if typedefIdx > fnIdx {
		t.Error("Contract 5: composite type must be emitted before function that uses it")
	}
}

func TestTupleTypeEmittedBeforeUse(t *testing.T) {
	tt := typetable.New()
	tty := tt.InternTuple([]typetable.TypeId{tt.I32, tt.Bool})
	fn := buildSimpleFn(tt, "test", tty, func(b *mir.Builder) {
		a := b.NewTemp(tt.I32)
		b.EmitConst(a, tt.I32, "1")
		c := b.NewTemp(tt.Bool)
		b.EmitConst(c, tt.Bool, "true")
		tmp := b.NewTemp(tty)
		b.EmitTuple(tmp, []mir.LocalId{a, c}, tty)
		b.TermReturn(tmp)
	})

	src := emitOne(t, fn, tt)
	if !strings.Contains(src, "typedef struct") {
		t.Error("tuple typedef should be emitted")
	}
}

// ===== Phase 02: Pointer categories (Contract 1) =====

func TestBorrowPointerUsesAddressOf(t *testing.T) {
	tt := typetable.New()
	refTy := tt.InternRef(tt.I32)
	fn := buildSimpleFn(tt, "test", refTy, func(b *mir.Builder) {
		src := b.NewLocal("x", tt.I32)
		b.EmitConst(src, tt.I32, "42")
		dest := b.NewTemp(refTy)
		b.EmitBorrow(dest, src, refTy, mir.BorrowShared)
		b.TermReturn(dest)
	})

	src := emitOne(t, fn, tt)
	if !strings.Contains(src, "&_l0") {
		t.Error("Contract 1: borrow pointer should use & (address-of)")
	}
	if !strings.Contains(src, "/* ref */") {
		t.Error("borrow should be annotated as ref")
	}
}

func TestMutrefBorrowAnnotation(t *testing.T) {
	tt := typetable.New()
	mrefTy := tt.InternMutRef(tt.I32)
	fn := buildSimpleFn(tt, "test", mrefTy, func(b *mir.Builder) {
		src := b.NewLocal("x", tt.I32)
		b.EmitConst(src, tt.I32, "42")
		dest := b.NewTemp(mrefTy)
		b.EmitBorrow(dest, src, mrefTy, mir.BorrowMutable)
		b.TermReturn(dest)
	})

	src := emitOne(t, fn, tt)
	if !strings.Contains(src, "/* mutref */") {
		t.Error("Contract 1: mutable borrow should be annotated as mutref")
	}
}

// ===== Phase 03: Unit erasure (Contract 2) =====

func TestUnitReturnIsVoid(t *testing.T) {
	tt := typetable.New()
	fn := buildSimpleFn(tt, "test", tt.Unit, func(b *mir.Builder) {
		tmp := b.NewTemp(tt.Unit)
		b.EmitConst(tmp, tt.Unit, "()")
		b.TermReturn(tmp)
	})

	src := emitOne(t, fn, tt)
	if !strings.Contains(src, "void Fuse_test(void)") {
		t.Errorf("Contract 2: unit return should emit void signature\n%s", src)
	}
}

func TestUnitLocalErased(t *testing.T) {
	tt := typetable.New()
	fn := buildSimpleFn(tt, "test", tt.I32, func(b *mir.Builder) {
		// Create a unit-typed local that should be erased.
		ulocal := b.NewTemp(tt.Unit)
		b.EmitConst(ulocal, tt.Unit, "()")
		ret := b.NewTemp(tt.I32)
		b.EmitConst(ret, tt.I32, "0")
		b.TermReturn(ret)
	})

	src := emitOne(t, fn, tt)
	// The unit local should not appear as a declaration.
	lines := strings.Split(src, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "void _l") && strings.HasSuffix(trimmed, ";") && !strings.Contains(trimmed, "(") {
			t.Errorf("Contract 2: unit-typed local should be erased, found: %q", trimmed)
		}
	}
}

func TestUnitParamErased(t *testing.T) {
	tt := typetable.New()
	params := []mir.Local{
		{Id: 0, Name: "x", Type: tt.Unit},
		{Id: 1, Name: "y", Type: tt.I32},
	}
	b := mir.NewBuilder("test", params, tt.I32)
	ret := b.NewTemp(tt.I32)
	b.EmitConst(ret, tt.I32, "0")
	b.TermReturn(ret)
	fn := b.Build()

	src := emitOne(t, fn, tt)
	// Signature should only have the I32 param, not the unit param.
	if strings.Contains(src, "void _l0") {
		t.Error("Contract 2: unit param should be erased from signature")
	}
	if !strings.Contains(src, "int32_t _l1") {
		t.Error("non-unit param should be present")
	}
}

// ===== Phase 03: Aggregate fallback (Contract 5) =====

func TestAggregateZeroInitializer(t *testing.T) {
	tt := typetable.New()
	sty := tt.InternStruct("m", "Foo", nil)
	fn := buildSimpleFn(tt, "test", sty, func(b *mir.Builder) {
		tmp := b.NewTemp(sty)
		b.EmitStructInit(tmp, "Foo", nil, sty)
		b.TermReturn(tmp)
	})

	src := emitOne(t, fn, tt)
	// Contract 5: aggregate fallback must be (Type){0}, not scalar 0.
	if !strings.Contains(src, "(Fuse_m__Foo){0}") {
		t.Errorf("Contract 5: aggregate fallback should be typed zero-init\n%s", src)
	}
}

// ===== Phase 04: Divergence (Contract 4) =====

func TestDivergenceEmitsUnreachable(t *testing.T) {
	tt := typetable.New()
	fn := buildSimpleFn(tt, "test", tt.Never, func(b *mir.Builder) {
		b.TermDiverge()
	})

	src := emitOne(t, fn, tt)
	if !strings.Contains(src, "__builtin_unreachable()") {
		t.Error("Contract 4: divergence should emit __builtin_unreachable()")
	}
}

func TestReturnTerminator(t *testing.T) {
	tt := typetable.New()
	fn := buildSimpleFn(tt, "test", tt.I32, func(b *mir.Builder) {
		tmp := b.NewTemp(tt.I32)
		b.EmitConst(tmp, tt.I32, "42")
		b.TermReturn(tmp)
	})

	src := emitOne(t, fn, tt)
	if !strings.Contains(src, "return _l") {
		t.Error("return terminator should emit return statement")
	}
}

// ===== Phase 04: Semantic equality note =====
// Non-scalar equality is simplified in this wave (emits raw ==).
// Full trait-based equality lowering is a later refinement.

// ===== Control flow =====

func TestBranchEmitsIfGoto(t *testing.T) {
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

	src := emitOne(t, fn, tt)
	if !strings.Contains(src, "if (") {
		t.Error("branch should emit if statement")
	}
	if !strings.Contains(src, "goto block_") {
		t.Error("branch should emit goto")
	}
}

func TestGotoTerminator(t *testing.T) {
	tt := typetable.New()
	fn := buildSimpleFn(tt, "test", tt.Unit, func(b *mir.Builder) {
		target := b.NewBlock()
		b.TermGoto(target)
		b.SwitchToBlock(target)
		tmp := b.NewTemp(tt.Unit)
		b.EmitConst(tmp, tt.Unit, "()")
		b.TermReturn(tmp)
	})

	src := emitOne(t, fn, tt)
	if !strings.Contains(src, "goto block_") {
		t.Error("goto terminator should emit goto")
	}
}

// ===== Calls =====

func TestCallEmission(t *testing.T) {
	tt := typetable.New()
	fn := buildSimpleFn(tt, "test", tt.I32, func(b *mir.Builder) {
		callee := b.NewTemp(tt.Unknown)
		b.EmitConst(callee, tt.Unknown, "foo")
		arg := b.NewTemp(tt.I32)
		b.EmitConst(arg, tt.I32, "1")
		dest := b.NewTemp(tt.I32)
		b.EmitCall(dest, callee, []mir.LocalId{arg}, tt.I32, false)
		b.TermReturn(dest)
	})

	src := emitOne(t, fn, tt)
	if !strings.Contains(src, "_l0(_l1)") {
		t.Errorf("call emission incorrect\n%s", src)
	}
}

// ===== Field read =====

func TestFieldReadEmission(t *testing.T) {
	tt := typetable.New()
	sty := tt.InternStruct("m", "Pt", nil)
	fn := buildSimpleFn(tt, "test", tt.I32, func(b *mir.Builder) {
		obj := b.NewTemp(sty)
		b.EmitStructInit(obj, "Pt", nil, sty)
		dest := b.NewTemp(tt.I32)
		b.EmitFieldRead(dest, obj, "x", tt.I32)
		b.TermReturn(dest)
	})

	src := emitOne(t, fn, tt)
	if !strings.Contains(src, ".x") {
		t.Error("field read should emit .fieldname")
	}
}

// ===== Binary op =====

func TestBinOpEmission(t *testing.T) {
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

	src := emitOne(t, fn, tt)
	if !strings.Contains(src, "_l0 + _l1") {
		t.Errorf("binop emission incorrect\n%s", src)
	}
}

// ===== Full pipeline: hello-world-level output =====

func TestHelloWorldOutput(t *testing.T) {
	tt := typetable.New()
	fn := buildSimpleFn(tt, "main", tt.I32, func(b *mir.Builder) {
		ret := b.NewTemp(tt.I32)
		b.EmitConst(ret, tt.I32, "0")
		b.TermReturn(ret)
	})

	src := emitOne(t, fn, tt)
	if !strings.Contains(src, "#include") {
		t.Error("output should have includes")
	}
	if !strings.Contains(src, "int32_t main(void)") {
		t.Errorf("output should have main function declaration\n%s", src)
	}
	if !strings.Contains(src, "return _l0") {
		t.Error("output should return")
	}
}

// ===== Destructor codegen (Drop trait) =====

func TestDropWithDropTrait(t *testing.T) {
	tt := typetable.New()
	sty := tt.InternStruct("m", "Resource", nil)
	fn := buildSimpleFn(tt, "test", tt.Unit, func(b *mir.Builder) {
		obj := b.NewLocal("r", sty)
		b.EmitStructInit(obj, "Resource", nil, sty)
		b.EmitDrop(obj)
		tmp := b.NewTemp(tt.Unit)
		b.EmitConst(tmp, tt.Unit, "()")
		b.TermReturn(tmp)
	})

	src := emitOneWithDropTypes(t, fn, tt, map[typetable.TypeId]bool{sty: true})
	if !strings.Contains(src, "Fuse_m__Resource_drop(&_l0)") {
		t.Errorf("Drop trait type should emit destructor call, got:\n%s", src)
	}
}

func TestDropWithoutDropTrait(t *testing.T) {
	tt := typetable.New()
	sty := tt.InternStruct("m", "Plain", nil)
	fn := buildSimpleFn(tt, "test", tt.Unit, func(b *mir.Builder) {
		obj := b.NewLocal("p", sty)
		b.EmitStructInit(obj, "Plain", nil, sty)
		b.EmitDrop(obj)
		tmp := b.NewTemp(tt.Unit)
		b.EmitConst(tmp, tt.Unit, "()")
		b.TermReturn(tmp)
	})

	src := emitOne(t, fn, tt)
	if strings.Contains(src, "_drop(") {
		t.Errorf("Type without Drop trait should not emit destructor call, got:\n%s", src)
	}
}

func TestDropUnitErased(t *testing.T) {
	tt := typetable.New()
	fn := buildSimpleFn(tt, "test", tt.Unit, func(b *mir.Builder) {
		u := b.NewTemp(tt.Unit)
		b.EmitConst(u, tt.Unit, "()")
		b.EmitDrop(u)
		b.TermReturn(u)
	})

	src := emitOne(t, fn, tt)
	if strings.Contains(src, "drop") {
		t.Errorf("Drop on unit type should emit nothing, got:\n%s", src)
	}
}

// ===== Section 2: generic template filter (L021) =====

// A MIR function whose params/locals/return type transitively reference
// a KindGenericParam is a template — it must produce no C output.
// Defends against regressions in which the driver's generic-original
// filter stops catching something; this is the backstop.
func TestGenericFunctionNotEmitted(t *testing.T) {
	tt := typetable.New()
	gp := tt.InternGenericParam("core.list", "T")

	// fn identity[T](x: T) -> T { ... } — template shape, not a
	// specialization. The emitter must emit no forward decl, no body,
	// no struct for T.
	params := []mir.Local{{Id: 0, Name: "x", Type: gp}}
	b := mir.NewBuilder("identity", params, gp)
	b.TermReturn(0)
	fn := b.Build()

	src := emitOne(t, fn, tt)
	if strings.Contains(src, "identity") {
		t.Errorf("generic function identity must not reach C output:\n%s", src)
	}
	// Also: no mangled reference to the generic param.
	if strings.Contains(src, "__T;") || strings.Contains(src, "__T ") {
		t.Errorf("generic param name T must not appear in C output:\n%s", src)
	}
}

// A generic struct type (Fields contain KindGenericParam, OR the type
// itself has TypeArgs that are generic params) must not produce a C
// typedef. Only its monomorphized specializations do.
func TestGenericStructTypedefNotEmitted(t *testing.T) {
	tt := typetable.New()
	gp := tt.InternGenericParam("core.list", "T")

	// Simulate `struct List[T] { items: T }` — the base template.
	listTy := tt.InternStruct("core.list", "List", []typetable.TypeId{gp})
	tt.SetStructFields(listTy, []string{"items"}, []typetable.TypeId{gp})

	// A non-generic function that mentions the generic type only in a
	// local variable — forces the emitter to at least consider it.
	// Since we also avoid the path where a generic function would be
	// skipped entirely, use a non-generic outer function.
	b := mir.NewBuilder("entry", nil, tt.Unit)
	_ = b.NewLocal("l", listTy) // carries the generic-template TypeId
	tmp := b.NewTemp(tt.Unit)
	b.EmitConst(tmp, tt.Unit, "()")
	b.TermReturn(tmp)
	fn := b.Build()

	src := emitOne(t, fn, tt)
	// No typedef for the generic template should appear.
	if strings.Contains(src, "typedef struct Fuse_core_list__List") {
		t.Errorf("generic struct template must not produce a typedef:\n%s", src)
	}
	// The function shouldn't have a local of a non-existent type either,
	// but current behavior still emits the local var declaration. What
	// must NOT happen: a typedef with the generic param name leaking.
	if strings.Contains(src, "__T;") {
		t.Errorf("generic param T leaked into C output:\n%s", src)
	}
}

// ===== Channel type emission =====

func TestChannelTypeEmission(t *testing.T) {
	tt := typetable.New()
	chanTy := tt.InternChannel(tt.I32)
	fn := buildSimpleFn(tt, "test", tt.Unit, func(b *mir.Builder) {
		ch := b.NewTemp(chanTy)
		b.EmitConst(ch, chanTy, "0")
		tmp := b.NewTemp(tt.Unit)
		b.EmitConst(tmp, tt.Unit, "()")
		b.TermReturn(tmp)
	})

	src := emitOne(t, fn, tt)
	// Should contain a channel typedef.
	if !strings.Contains(src, "typedef struct { void* _impl;") {
		t.Errorf("channel type should emit typedef with _impl field\n%s", src)
	}
	if !strings.Contains(src, "FuseChan_int32_t") {
		t.Errorf("channel type name should be FuseChan_int32_t\n%s", src)
	}
}
