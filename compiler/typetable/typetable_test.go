package typetable

import "testing"

func TestNewTableHasPrimitives(t *testing.T) {
	tt := New()
	// All primitives should have non-zero IDs.
	prims := []struct {
		name string
		id   TypeId
	}{
		{"Unit", tt.Unit}, {"Bool", tt.Bool}, {"Char", tt.Char},
		{"I8", tt.I8}, {"I16", tt.I16}, {"I32", tt.I32}, {"I64", tt.I64},
		{"I128", tt.I128}, {"ISize", tt.ISize},
		{"U8", tt.U8}, {"U16", tt.U16}, {"U32", tt.U32}, {"U64", tt.U64},
		{"U128", tt.U128}, {"USize", tt.USize},
		{"F32", tt.F32}, {"F64", tt.F64},
		{"Never", tt.Never}, {"Unknown", tt.Unknown},
	}
	seen := make(map[TypeId]string)
	for _, p := range prims {
		if p.id == InvalidTypeId {
			t.Errorf("%s has InvalidTypeId", p.name)
		}
		if prev, ok := seen[p.id]; ok {
			t.Errorf("%s and %s share TypeId %d", p.name, prev, p.id)
		}
		seen[p.id] = p.name
	}
}

func TestLookupPrimitive(t *testing.T) {
	tt := New()
	cases := []struct {
		name string
		want TypeId
	}{
		{"Bool", tt.Bool},
		{"I32", tt.I32},
		{"Int", tt.ISize},    // alias
		{"Float", tt.F64},    // alias
		{"Never", tt.Never},
		{"Nonexistent", InvalidTypeId},
	}
	for _, tc := range cases {
		got := tt.LookupPrimitive(tc.name)
		if got != tc.want {
			t.Errorf("LookupPrimitive(%q): got %d, want %d", tc.name, got, tc.want)
		}
	}
}

func TestInternDeduplicates(t *testing.T) {
	tt := New()
	a := tt.InternSlice(tt.I32)
	b := tt.InternSlice(tt.I32)
	if a != b {
		t.Errorf("same slice type got different IDs: %d vs %d", a, b)
	}
}

func TestDifferentTypesGetDifferentIds(t *testing.T) {
	tt := New()
	a := tt.InternSlice(tt.I32)
	b := tt.InternSlice(tt.I64)
	if a == b {
		t.Error("[I32] and [I64] should have different IDs")
	}
}

func TestInternTuple(t *testing.T) {
	tt := New()
	t1 := tt.InternTuple([]TypeId{tt.I32, tt.Bool})
	t2 := tt.InternTuple([]TypeId{tt.I32, tt.Bool})
	if t1 != t2 {
		t.Error("same tuple type should dedup")
	}
	t3 := tt.InternTuple([]TypeId{tt.Bool, tt.I32})
	if t1 == t3 {
		t.Error("(I32, Bool) and (Bool, I32) should differ")
	}
}

func TestEmptyTupleIsUnit(t *testing.T) {
	tt := New()
	id := tt.InternTuple(nil)
	if id != tt.Unit {
		t.Errorf("empty tuple should be Unit, got %d", id)
	}
}

func TestInternArray(t *testing.T) {
	tt := New()
	a := tt.InternArray(tt.U8, 256)
	b := tt.InternArray(tt.U8, 256)
	c := tt.InternArray(tt.U8, 128)
	if a != b {
		t.Error("same array should dedup")
	}
	if a == c {
		t.Error("different lengths should differ")
	}
}

func TestInternPtr(t *testing.T) {
	tt := New()
	a := tt.InternPtr(tt.U8)
	e := tt.Get(a)
	if e.Kind != KindPtr || e.Elem != tt.U8 {
		t.Errorf("Ptr[U8]: kind=%s elem=%d", e.Kind, e.Elem)
	}
}

func TestInternRefAndMutRef(t *testing.T) {
	tt := New()
	r := tt.InternRef(tt.I32)
	m := tt.InternMutRef(tt.I32)
	if r == m {
		t.Error("ref and mutref should differ")
	}
	if tt.Get(r).Kind != KindRef {
		t.Error("ref kind")
	}
	if tt.Get(m).Kind != KindMutRef {
		t.Error("mutref kind")
	}
}

func TestInternFunc(t *testing.T) {
	tt := New()
	f := tt.InternFunc([]TypeId{tt.I32, tt.I32}, tt.Bool)
	e := tt.Get(f)
	if e.Kind != KindFunc || len(e.Fields) != 2 || e.ReturnType != tt.Bool {
		t.Errorf("func: kind=%s params=%d ret=%d", e.Kind, len(e.Fields), e.ReturnType)
	}
}

func TestNominalIdentity(t *testing.T) {
	tt := New()
	a := tt.InternStruct("mod_a", "Point", nil)
	b := tt.InternStruct("mod_b", "Point", nil)
	if a == b {
		t.Error("same-name structs from different modules must be distinct")
	}
	c := tt.InternStruct("mod_a", "Point", nil)
	if a != c {
		t.Error("same module+name should dedup")
	}
}

func TestGenericInstantiationsAreDistinct(t *testing.T) {
	tt := New()
	a := tt.InternStruct("core", "Option", []TypeId{tt.I32})
	b := tt.InternStruct("core", "Option", []TypeId{tt.I64})
	if a == b {
		t.Error("Option[I32] and Option[I64] must be distinct")
	}
}

func TestGenericParam(t *testing.T) {
	tt := New()
	p := tt.InternGenericParam("core.list", "T")
	e := tt.Get(p)
	if e.Kind != KindGenericParam || e.Name != "T" {
		t.Errorf("generic param: kind=%s name=%q", e.Kind, e.Name)
	}
}

func TestIsNumeric(t *testing.T) {
	tt := New()
	if !tt.IsNumeric(tt.I32) {
		t.Error("I32 should be numeric")
	}
	if !tt.IsNumeric(tt.F64) {
		t.Error("F64 should be numeric")
	}
	if tt.IsNumeric(tt.Bool) {
		t.Error("Bool should not be numeric")
	}
}

func TestInternChannel(t *testing.T) {
	tt := New()
	ch := tt.InternChannel(tt.I32)
	if ch == InvalidTypeId {
		t.Fatal("InternChannel returned InvalidTypeId")
	}
	e := tt.Get(ch)
	if e.Kind != KindChannel {
		t.Errorf("expected KindChannel, got %s", e.Kind)
	}
	if e.Elem != tt.I32 {
		t.Errorf("expected Elem == I32 (%d), got %d", tt.I32, e.Elem)
	}
}

func TestInternChannelDeduplicates(t *testing.T) {
	tt := New()
	a := tt.InternChannel(tt.I32)
	b := tt.InternChannel(tt.I32)
	if a != b {
		t.Errorf("same channel type got different IDs: %d vs %d", a, b)
	}
}

func TestInternChannelDifferentElem(t *testing.T) {
	tt := New()
	a := tt.InternChannel(tt.I32)
	b := tt.InternChannel(tt.Bool)
	if a == b {
		t.Error("Chan[I32] and Chan[Bool] should have different IDs")
	}
}

func TestIsResolved(t *testing.T) {
	tt := New()
	if !tt.IsResolved(tt.I32) {
		t.Error("I32 should be resolved")
	}
	if tt.IsResolved(tt.Unknown) {
		t.Error("Unknown should not be resolved")
	}
	p := tt.InternGenericParam("m", "T")
	if tt.IsResolved(p) {
		t.Error("generic param should not be resolved")
	}
}

func TestBaseOfReturnsTemplateForSpecialization(t *testing.T) {
	tt := New()
	// Template interned first with its field layout.
	base := tt.InternStruct("core.list", "List", nil)
	pt := tt.InternGenericParam("core.list", "T")
	tt.SetStructFields(base, []string{"data", "len"}, []TypeId{tt.InternPtr(pt), tt.USize})

	spec := tt.InternStruct("core.list", "List", []TypeId{tt.I32})
	got := tt.BaseOf(spec)
	if got != base {
		t.Errorf("BaseOf(List[I32]) = %d, want template %d", got, base)
	}
}

func TestBaseOfReturnsInvalidForTemplate(t *testing.T) {
	tt := New()
	base := tt.InternStruct("core.list", "List", nil)
	if got := tt.BaseOf(base); got != InvalidTypeId {
		t.Errorf("BaseOf(template) = %d, want InvalidTypeId", got)
	}
}

func TestBaseOfReturnsInvalidForPrimitive(t *testing.T) {
	tt := New()
	if got := tt.BaseOf(tt.I32); got != InvalidTypeId {
		t.Errorf("BaseOf(I32) = %d, want InvalidTypeId", got)
	}
}

func TestBaseOfReturnsInvalidWhenTemplateAbsent(t *testing.T) {
	tt := New()
	// Specialization interned without ever registering the template. L021
	// guarantees this shouldn't happen after Section 3a, but BaseOf must
	// still return InvalidTypeId rather than synthesize a template.
	spec := tt.InternStruct("foo", "Ghost", []TypeId{tt.I32})
	if got := tt.BaseOf(spec); got != InvalidTypeId {
		t.Errorf("BaseOf(spec without template) = %d, want InvalidTypeId", got)
	}
}

func TestSubstituteFieldsBasic(t *testing.T) {
	tt := New()
	base := tt.InternStruct("core.list", "List", nil)
	pt := tt.InternGenericParam("core.list", "T")
	ptrT := tt.InternPtr(pt)
	tt.SetStructFields(base, []string{"data", "len"}, []TypeId{ptrT, tt.USize})

	names, types := tt.SubstituteFields(base, []TypeId{tt.I32})
	if len(types) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(types))
	}
	if len(names) != 2 || names[0] != "data" || names[1] != "len" {
		t.Errorf("unexpected names: %v", names)
	}
	got0 := tt.Get(types[0])
	if got0.Kind != KindPtr {
		t.Fatalf("field 0 kind = %s, want Ptr", got0.Kind)
	}
	if got0.Elem != tt.I32 {
		t.Errorf("field 0 elem = %d, want I32 (%d)", got0.Elem, tt.I32)
	}
	if types[1] != tt.USize {
		t.Errorf("field 1 = %d, want USize (%d)", types[1], tt.USize)
	}
}

func TestSubstituteFieldsNestedGeneric(t *testing.T) {
	tt := New()
	// Option[T] with a Some(T) variant payload.
	opt := tt.InternEnum("core.option", "Option", nil)
	pt := tt.InternGenericParam("core.option", "T")
	tt.SetEnumFields(opt, []TypeId{pt})

	// Node[T] { value: T, next: Option[T] }.
	node := tt.InternStruct("m", "Node", nil)
	ntp := tt.InternGenericParam("m", "T")
	optOfT := tt.InternEnum("core.option", "Option", []TypeId{ntp})
	tt.SetStructFields(node, []string{"value", "next"}, []TypeId{ntp, optOfT})

	_, types := tt.SubstituteFields(node, []TypeId{tt.I32})
	if types[0] != tt.I32 {
		t.Errorf("value type = %d, want I32", types[0])
	}
	got1 := tt.Get(types[1])
	if got1.Kind != KindEnum || got1.Name != "Option" || got1.Module != "core.option" {
		t.Errorf("next type = kind=%s name=%s module=%s", got1.Kind, got1.Name, got1.Module)
	}
	if len(got1.TypeArgs) != 1 || got1.TypeArgs[0] != tt.I32 {
		t.Errorf("next TypeArgs = %v, want [I32]", got1.TypeArgs)
	}
}

func TestSubstituteFieldsInvalidBase(t *testing.T) {
	tt := New()
	names, types := tt.SubstituteFields(InvalidTypeId, []TypeId{tt.I32})
	if names != nil || types != nil {
		t.Errorf("expected (nil, nil), got (%v, %v)", names, types)
	}
}
