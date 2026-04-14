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
