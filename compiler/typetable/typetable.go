// Package typetable owns interned type identity via TypeId.
// Equality of types is integer comparison over interned entries.
package typetable

import "fmt"

// TypeId is a handle into the TypeTable. Type equality is TypeId equality.
type TypeId int

const (
	// InvalidTypeId is the zero value and signals an unset or erroneous type.
	InvalidTypeId TypeId = 0
)

// TypeKind classifies the shape of a type entry.
type TypeKind int

const (
	KindUnknown  TypeKind = iota
	KindUnit              // ()
	KindBool              // Bool
	KindChar              // Char
	KindInt               // I8..I128, ISize
	KindUint              // U8..U128, USize
	KindFloat             // F32, F64
	KindTuple             // (T1, T2, ...)
	KindArray             // [T; N]
	KindSlice             // [T]
	KindPtr               // Ptr[T]
	KindRef               // ref T (borrow)
	KindMutRef            // mutref T (mutable borrow)
	KindStruct            // user-defined struct
	KindEnum              // user-defined enum
	KindFunc              // function type
	KindGenericParam      // unresolved type parameter T
	KindNever             // ! / Never (diverging)
)

func (k TypeKind) String() string {
	names := [...]string{
		KindUnknown:      "Unknown",
		KindUnit:         "Unit",
		KindBool:         "Bool",
		KindChar:         "Char",
		KindInt:          "Int",
		KindUint:         "Uint",
		KindFloat:        "Float",
		KindTuple:        "Tuple",
		KindArray:        "Array",
		KindSlice:        "Slice",
		KindPtr:          "Ptr",
		KindRef:          "Ref",
		KindMutRef:       "MutRef",
		KindStruct:       "Struct",
		KindEnum:         "Enum",
		KindFunc:         "Func",
		KindGenericParam: "GenericParam",
		KindNever:        "Never",
	}
	if int(k) < len(names) {
		return names[k]
	}
	return fmt.Sprintf("TypeKind(%d)", k)
}

// TypeEntry is the interned data for a single type.
type TypeEntry struct {
	Kind TypeKind

	// Name is the declared name for nominal types (structs, enums, type params).
	Name string

	// Module is the defining module path for nominal identity.
	// Two types with the same name from different modules are distinct.
	Module string

	// BitSize is the width for numeric types (8, 16, 32, 64, 128, 0 for platform).
	BitSize int

	// Elem is the element type for Ptr, Ref, MutRef, Slice, Array.
	Elem TypeId

	// ArrayLen is the static length for array types.
	ArrayLen int

	// Fields holds TypeIds for tuple elements or function param types.
	Fields []TypeId

	// ReturnType is the return type for function types.
	ReturnType TypeId

	// TypeArgs holds concrete type arguments for generic instantiations.
	TypeArgs []TypeId
}

// TypeTable is the global canonical type store. All type references in HIR and
// MIR use TypeId values that index into this table.
type TypeTable struct {
	entries []TypeEntry
	intern  map[string]TypeId // dedup key → existing TypeId

	// Well-known primitive type IDs, set during Init().
	Unit    TypeId
	Bool    TypeId
	Char    TypeId
	I8      TypeId
	I16     TypeId
	I32     TypeId
	I64     TypeId
	I128    TypeId
	ISize   TypeId
	U8      TypeId
	U16     TypeId
	U32     TypeId
	U64     TypeId
	U128    TypeId
	USize   TypeId
	F32     TypeId
	F64     TypeId
	Never   TypeId
	Unknown TypeId
}

// New creates and initializes a TypeTable with all primitive types registered.
func New() *TypeTable {
	tt := &TypeTable{
		entries: []TypeEntry{{}}, // index 0 = InvalidTypeId (sentinel)
		intern:  make(map[string]TypeId),
	}
	tt.initPrimitives()
	return tt
}

func (tt *TypeTable) initPrimitives() {
	tt.Unknown = tt.insert(TypeEntry{Kind: KindUnknown, Name: "Unknown"})
	tt.Unit = tt.insert(TypeEntry{Kind: KindUnit, Name: "()"})
	tt.Bool = tt.insert(TypeEntry{Kind: KindBool, Name: "Bool"})
	tt.Char = tt.insert(TypeEntry{Kind: KindChar, Name: "Char"})
	tt.Never = tt.insert(TypeEntry{Kind: KindNever, Name: "Never"})

	tt.I8 = tt.insert(TypeEntry{Kind: KindInt, Name: "I8", BitSize: 8})
	tt.I16 = tt.insert(TypeEntry{Kind: KindInt, Name: "I16", BitSize: 16})
	tt.I32 = tt.insert(TypeEntry{Kind: KindInt, Name: "I32", BitSize: 32})
	tt.I64 = tt.insert(TypeEntry{Kind: KindInt, Name: "I64", BitSize: 64})
	tt.I128 = tt.insert(TypeEntry{Kind: KindInt, Name: "I128", BitSize: 128})
	tt.ISize = tt.insert(TypeEntry{Kind: KindInt, Name: "ISize", BitSize: 0})

	tt.U8 = tt.insert(TypeEntry{Kind: KindUint, Name: "U8", BitSize: 8})
	tt.U16 = tt.insert(TypeEntry{Kind: KindUint, Name: "U16", BitSize: 16})
	tt.U32 = tt.insert(TypeEntry{Kind: KindUint, Name: "U32", BitSize: 32})
	tt.U64 = tt.insert(TypeEntry{Kind: KindUint, Name: "U64", BitSize: 64})
	tt.U128 = tt.insert(TypeEntry{Kind: KindUint, Name: "U128", BitSize: 128})
	tt.USize = tt.insert(TypeEntry{Kind: KindUint, Name: "USize", BitSize: 0})

	tt.F32 = tt.insert(TypeEntry{Kind: KindFloat, Name: "F32", BitSize: 32})
	tt.F64 = tt.insert(TypeEntry{Kind: KindFloat, Name: "F64", BitSize: 64})
}

// Len returns the number of type entries (including the sentinel at index 0).
func (tt *TypeTable) Len() int {
	return len(tt.entries)
}

// Get returns the entry for a TypeId. Panics on out-of-range.
func (tt *TypeTable) Get(id TypeId) *TypeEntry {
	return &tt.entries[id]
}

// --- interning ---

func (tt *TypeTable) insert(e TypeEntry) TypeId {
	key := internKey(e)
	if existing, ok := tt.intern[key]; ok {
		return existing
	}
	id := TypeId(len(tt.entries))
	tt.entries = append(tt.entries, e)
	tt.intern[key] = id
	return id
}

// internKey produces a deterministic string key for deduplication.
func internKey(e TypeEntry) string {
	switch e.Kind {
	case KindUnit, KindBool, KindChar, KindNever, KindUnknown:
		return e.Name
	case KindInt, KindUint, KindFloat:
		return fmt.Sprintf("%s_%d", e.Name, e.BitSize)
	case KindPtr, KindRef, KindMutRef, KindSlice:
		return fmt.Sprintf("%s[%d]", e.Kind, e.Elem)
	case KindArray:
		return fmt.Sprintf("Array[%d;%d]", e.Elem, e.ArrayLen)
	case KindTuple:
		return fmt.Sprintf("Tuple%v", e.Fields)
	case KindStruct, KindEnum:
		return fmt.Sprintf("%s:%s:%s%v", e.Kind, e.Module, e.Name, e.TypeArgs)
	case KindFunc:
		return fmt.Sprintf("Func(%v)->%d", e.Fields, e.ReturnType)
	case KindGenericParam:
		return fmt.Sprintf("Param:%s:%s", e.Module, e.Name)
	}
	return fmt.Sprintf("?%v", e)
}

// --- public constructors for compound types ---

// Intern interns an arbitrary type entry and returns its canonical TypeId.
func (tt *TypeTable) Intern(e TypeEntry) TypeId {
	return tt.insert(e)
}

// InternTuple interns a tuple type.
func (tt *TypeTable) InternTuple(elems []TypeId) TypeId {
	if len(elems) == 0 {
		return tt.Unit
	}
	return tt.insert(TypeEntry{Kind: KindTuple, Fields: elems})
}

// InternSlice interns a slice type.
func (tt *TypeTable) InternSlice(elem TypeId) TypeId {
	return tt.insert(TypeEntry{Kind: KindSlice, Elem: elem})
}

// InternArray interns an array type.
func (tt *TypeTable) InternArray(elem TypeId, length int) TypeId {
	return tt.insert(TypeEntry{Kind: KindArray, Elem: elem, ArrayLen: length})
}

// InternPtr interns a raw pointer type.
func (tt *TypeTable) InternPtr(elem TypeId) TypeId {
	return tt.insert(TypeEntry{Kind: KindPtr, Elem: elem})
}

// InternRef interns a shared borrow type.
func (tt *TypeTable) InternRef(elem TypeId) TypeId {
	return tt.insert(TypeEntry{Kind: KindRef, Elem: elem})
}

// InternMutRef interns a mutable borrow type.
func (tt *TypeTable) InternMutRef(elem TypeId) TypeId {
	return tt.insert(TypeEntry{Kind: KindMutRef, Elem: elem})
}

// InternFunc interns a function type.
func (tt *TypeTable) InternFunc(params []TypeId, ret TypeId) TypeId {
	return tt.insert(TypeEntry{Kind: KindFunc, Fields: params, ReturnType: ret})
}

// InternStruct interns a struct type with nominal identity (module + name).
func (tt *TypeTable) InternStruct(module, name string, typeArgs []TypeId) TypeId {
	return tt.insert(TypeEntry{Kind: KindStruct, Module: module, Name: name, TypeArgs: typeArgs})
}

// InternEnum interns an enum type with nominal identity.
func (tt *TypeTable) InternEnum(module, name string, typeArgs []TypeId) TypeId {
	return tt.insert(TypeEntry{Kind: KindEnum, Module: module, Name: name, TypeArgs: typeArgs})
}

// InternGenericParam interns a generic type parameter.
func (tt *TypeTable) InternGenericParam(module, name string) TypeId {
	return tt.insert(TypeEntry{Kind: KindGenericParam, Module: module, Name: name})
}

// --- query helpers ---

// LookupPrimitive returns the TypeId for a primitive type name, or InvalidTypeId.
func (tt *TypeTable) LookupPrimitive(name string) TypeId {
	switch name {
	case "Bool":
		return tt.Bool
	case "Char":
		return tt.Char
	case "I8":
		return tt.I8
	case "I16":
		return tt.I16
	case "I32":
		return tt.I32
	case "I64":
		return tt.I64
	case "I128":
		return tt.I128
	case "ISize", "Int":
		return tt.ISize
	case "U8":
		return tt.U8
	case "U16":
		return tt.U16
	case "U32":
		return tt.U32
	case "U64":
		return tt.U64
	case "U128":
		return tt.U128
	case "USize":
		return tt.USize
	case "F32":
		return tt.F32
	case "F64":
		return tt.F64
	case "Float":
		return tt.F64
	case "Never":
		return tt.Never
	}
	return InvalidTypeId
}

// IsNumeric reports whether id is an integer or float type.
func (tt *TypeTable) IsNumeric(id TypeId) bool {
	k := tt.Get(id).Kind
	return k == KindInt || k == KindUint || k == KindFloat
}

// IsResolved reports whether a type has been fully resolved (not Unknown or GenericParam).
func (tt *TypeTable) IsResolved(id TypeId) bool {
	e := tt.Get(id)
	return e.Kind != KindUnknown && e.Kind != KindGenericParam
}
