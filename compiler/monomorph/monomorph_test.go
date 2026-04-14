package monomorph

import (
	"testing"

	"github.com/Tembocs/fuse4/compiler/typetable"
)

func setup() (*typetable.TypeTable, *Context) {
	tt := typetable.New()
	ctx := NewContext(tt)
	return tt, ctx
}

func TestRecordDeduplicates(t *testing.T) {
	tt, ctx := setup()

	id1 := ctx.Record("Option", []typetable.TypeId{tt.I32})
	id2 := ctx.Record("Option", []typetable.TypeId{tt.I32})

	if id1 != id2 {
		t.Fatalf("expected same resolved ID for identical instantiation, got %d and %d", id1, id2)
	}
	if len(ctx.Instantiations) != 1 {
		t.Fatalf("expected 1 instantiation, got %d", len(ctx.Instantiations))
	}
}

func TestRecordRejectsPartialSpecialization(t *testing.T) {
	tt, ctx := setup()

	// Unknown type arg.
	id := ctx.Record("Option", []typetable.TypeId{tt.Unknown})
	if id != typetable.InvalidTypeId {
		t.Fatalf("expected InvalidTypeId for Unknown arg, got %d", id)
	}

	// GenericParam type arg.
	paramT := tt.InternGenericParam("test", "T")
	id = ctx.Record("Option", []typetable.TypeId{paramT})
	if id != typetable.InvalidTypeId {
		t.Fatalf("expected InvalidTypeId for GenericParam arg, got %d", id)
	}

	if len(ctx.Instantiations) != 0 {
		t.Fatalf("expected 0 instantiations, got %d", len(ctx.Instantiations))
	}
}

func TestSubstituteGenericParam(t *testing.T) {
	tt, ctx := setup()

	paramT := tt.InternGenericParam("mod", "T")
	result := ctx.Substitute(paramT, []string{"T"}, []typetable.TypeId{tt.I32})

	if result != tt.I32 {
		t.Fatalf("expected I32, got %d", result)
	}
}

func TestSubstituteCompound(t *testing.T) {
	tt, ctx := setup()

	paramT := tt.InternGenericParam("mod", "T")
	sliceT := tt.InternSlice(paramT)

	result := ctx.Substitute(sliceT, []string{"T"}, []typetable.TypeId{tt.I32})
	expected := tt.InternSlice(tt.I32)

	if result != expected {
		t.Fatalf("expected Slice[I32] (%d), got %d", expected, result)
	}
}

func TestSubstituteFunc(t *testing.T) {
	tt, ctx := setup()

	paramT := tt.InternGenericParam("mod", "T")
	funcTT := tt.InternFunc([]typetable.TypeId{paramT}, paramT)

	result := ctx.Substitute(funcTT, []string{"T"}, []typetable.TypeId{tt.I32})
	expected := tt.InternFunc([]typetable.TypeId{tt.I32}, tt.I32)

	if result != expected {
		t.Fatalf("expected fn(I32)->I32 (%d), got %d", expected, result)
	}
}

func TestIsGeneric(t *testing.T) {
	tt, ctx := setup()

	paramT := tt.InternGenericParam("mod", "T")
	if !ctx.IsGeneric(paramT) {
		t.Fatal("expected GenericParam to be generic")
	}

	if ctx.IsGeneric(tt.I32) {
		t.Fatal("expected I32 not to be generic")
	}

	if ctx.IsGeneric(tt.Bool) {
		t.Fatal("expected Bool not to be generic")
	}
}

func TestRecordMultipleInstantiations(t *testing.T) {
	tt, ctx := setup()

	id1 := ctx.Record("Option", []typetable.TypeId{tt.I32})
	id2 := ctx.Record("Option", []typetable.TypeId{tt.Bool})

	if id1 == id2 {
		t.Fatalf("expected distinct resolved IDs for different type args, both got %d", id1)
	}
	if id1 == typetable.InvalidTypeId || id2 == typetable.InvalidTypeId {
		t.Fatal("expected valid type IDs")
	}
	if len(ctx.Instantiations) != 2 {
		t.Fatalf("expected 2 instantiations, got %d", len(ctx.Instantiations))
	}
}

func TestRecordEmptyTypeArgs(t *testing.T) {
	_, ctx := setup()

	id := ctx.Record("NotGeneric", nil)
	if id != typetable.InvalidTypeId {
		t.Fatalf("expected InvalidTypeId for empty type args, got %d", id)
	}
}

func TestSubstituteRef(t *testing.T) {
	tt, ctx := setup()

	paramT := tt.InternGenericParam("mod", "T")
	refT := tt.InternRef(paramT)

	result := ctx.Substitute(refT, []string{"T"}, []typetable.TypeId{tt.I32})
	expected := tt.InternRef(tt.I32)

	if result != expected {
		t.Fatalf("expected Ref[I32] (%d), got %d", expected, result)
	}
}

func TestSubstituteMutRef(t *testing.T) {
	tt, ctx := setup()

	paramT := tt.InternGenericParam("mod", "T")
	mutRefT := tt.InternMutRef(paramT)

	result := ctx.Substitute(mutRefT, []string{"T"}, []typetable.TypeId{tt.I32})
	expected := tt.InternMutRef(tt.I32)

	if result != expected {
		t.Fatalf("expected MutRef[I32] (%d), got %d", expected, result)
	}
}

func TestSubstitutePtr(t *testing.T) {
	tt, ctx := setup()

	paramT := tt.InternGenericParam("mod", "T")
	ptrT := tt.InternPtr(paramT)

	result := ctx.Substitute(ptrT, []string{"T"}, []typetable.TypeId{tt.I32})
	expected := tt.InternPtr(tt.I32)

	if result != expected {
		t.Fatalf("expected Ptr[I32] (%d), got %d", expected, result)
	}
}

func TestSubstituteArray(t *testing.T) {
	tt, ctx := setup()

	paramT := tt.InternGenericParam("mod", "T")
	arrayT := tt.InternArray(paramT, 5)

	result := ctx.Substitute(arrayT, []string{"T"}, []typetable.TypeId{tt.I32})
	expected := tt.InternArray(tt.I32, 5)

	if result != expected {
		t.Fatalf("expected Array[I32;5] (%d), got %d", expected, result)
	}
}

func TestSubstituteTuple(t *testing.T) {
	tt, ctx := setup()

	paramT := tt.InternGenericParam("mod", "T")
	paramU := tt.InternGenericParam("mod", "U")
	tupleType := tt.InternTuple([]typetable.TypeId{paramT, paramU})

	result := ctx.Substitute(tupleType, []string{"T", "U"}, []typetable.TypeId{tt.I32, tt.Bool})
	expected := tt.InternTuple([]typetable.TypeId{tt.I32, tt.Bool})

	if result != expected {
		t.Fatalf("expected (I32, Bool) (%d), got %d", expected, result)
	}
}

func TestSubstituteStruct(t *testing.T) {
	tt, ctx := setup()

	paramT := tt.InternGenericParam("mod", "T")
	genericStruct := tt.InternStruct("core", "Vec", []typetable.TypeId{paramT})

	result := ctx.Substitute(genericStruct, []string{"T"}, []typetable.TypeId{tt.I32})
	expected := tt.InternStruct("core", "Vec", []typetable.TypeId{tt.I32})

	if result != expected {
		t.Fatalf("expected Vec[I32] (%d), got %d", expected, result)
	}
}

func TestSubstituteEnum(t *testing.T) {
	tt, ctx := setup()

	paramT := tt.InternGenericParam("mod", "T")
	genericEnum := tt.InternEnum("core", "Option", []typetable.TypeId{paramT})

	result := ctx.Substitute(genericEnum, []string{"T"}, []typetable.TypeId{tt.I32})
	expected := tt.InternEnum("core", "Option", []typetable.TypeId{tt.I32})

	if result != expected {
		t.Fatalf("expected Option[I32] (%d), got %d", expected, result)
	}
}

func TestSubstituteUnmatchedParam(t *testing.T) {
	tt, ctx := setup()

	paramT := tt.InternGenericParam("mod", "T")

	// Substitute with a different param name — should leave as is.
	result := ctx.Substitute(paramT, []string{"U"}, []typetable.TypeId{tt.I32})

	if result != paramT {
		t.Fatalf("expected unresolved param T (%d), got %d", paramT, result)
	}
}

func TestSubstituteConcreteLeavesUnchanged(t *testing.T) {
	tt, ctx := setup()

	// Substituting on a concrete type should return it unchanged.
	result := ctx.Substitute(tt.I32, []string{"T"}, []typetable.TypeId{tt.Bool})

	if result != tt.I32 {
		t.Fatalf("expected I32 unchanged (%d), got %d", tt.I32, result)
	}
}
