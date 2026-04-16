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
	KindChannel           // channel type with element type
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
		KindChannel:      "Channel",
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

	// Fields holds TypeIds for tuple elements, function param types, or struct/enum fields.
	Fields []TypeId

	// FieldNames holds the declared field names for struct types (parallel to Fields).
	FieldNames []string

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
	case KindPtr, KindRef, KindMutRef, KindSlice, KindChannel:
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

// SetEnumFields sets the payload field types on an existing enum type entry.
// This is called after variant analysis to provide field info for codegen.
func (tt *TypeTable) SetEnumFields(id TypeId, fields []TypeId) {
	if int(id) < len(tt.entries) {
		tt.entries[id].Fields = fields
	}
}

// SetStructFields sets the named field info on an existing struct type entry.
// FieldNames and Fields must have the same length.
func (tt *TypeTable) SetStructFields(id TypeId, names []string, types []TypeId) {
	if int(id) < len(tt.entries) {
		tt.entries[id].Fields = types
		tt.entries[id].FieldNames = names
	}
}

// InternGenericParam interns a generic type parameter.
func (tt *TypeTable) InternGenericParam(module, name string) TypeId {
	return tt.insert(TypeEntry{Kind: KindGenericParam, Module: module, Name: name})
}

// InternChannel interns a channel type with an element type.
func (tt *TypeTable) InternChannel(elem TypeId) TypeId {
	return tt.insert(TypeEntry{Kind: KindChannel, Elem: elem, Name: "Chan"})
}

// RegisterStringType sets up the String type with data pointer and length fields.
// The module string matches `stdlib/core/string.fuse` (module `core.string`)
// so a literal's nominal TypeId matches user-declared fields of type String.
func (tt *TypeTable) RegisterStringType() TypeId {
	strTy := tt.InternStruct("core.string", "String", nil)
	ptrTy := tt.InternPtr(tt.U8)
	tt.SetStructFields(strTy, []string{"data", "len"}, []TypeId{ptrTy, tt.USize})
	return strTy
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

// HasGenericParam reports whether a type transitively references a
// KindGenericParam entry. Used by the backend to skip generic templates
// that must not reach C output (Rule 3.9; see learning-log L021).
//
// This is the single source of truth for "is this type a generic template."
// It walks pointer/borrow/slice/array/tuple/struct/enum/channel/func
// compositions through Elem, Fields, TypeArgs, and ReturnType.
func (tt *TypeTable) HasGenericParam(id TypeId) bool {
	return tt.hasGenericParamVisit(id, map[TypeId]bool{})
}

func (tt *TypeTable) hasGenericParamVisit(id TypeId, seen map[TypeId]bool) bool {
	if id == InvalidTypeId {
		return false
	}
	if seen[id] {
		return false
	}
	seen[id] = true
	e := tt.Get(id)
	if e.Kind == KindGenericParam {
		return true
	}
	if e.Elem != InvalidTypeId && tt.hasGenericParamVisit(e.Elem, seen) {
		return true
	}
	for _, f := range e.Fields {
		if tt.hasGenericParamVisit(f, seen) {
			return true
		}
	}
	for _, a := range e.TypeArgs {
		if tt.hasGenericParamVisit(a, seen) {
			return true
		}
	}
	if e.Kind == KindFunc {
		if tt.hasGenericParamVisit(e.ReturnType, seen) {
			return true
		}
	}
	return false
}

// BaseOf returns the TypeId of the generic template for a specialization —
// same module, same name, nil TypeArgs. Returns InvalidTypeId if the input
// is not a KindStruct/KindEnum specialization or if the template does not
// exist in the table.
//
// The template must already have been interned (typically during the
// checker's signature pass on the defining module). Callers must not fall
// back to a "canonical module" guess when BaseOf returns InvalidTypeId —
// that would re-introduce the L021 band-aid pattern. The correct response
// is a diagnostic at the resolution site.
func (tt *TypeTable) BaseOf(id TypeId) TypeId {
	if id == InvalidTypeId {
		return InvalidTypeId
	}
	e := tt.Get(id)
	if e.Kind != KindStruct && e.Kind != KindEnum {
		return InvalidTypeId
	}
	if len(e.TypeArgs) == 0 {
		return InvalidTypeId // not a specialization
	}
	key := internKey(TypeEntry{Kind: e.Kind, Module: e.Module, Name: e.Name})
	base, ok := tt.intern[key]
	if !ok {
		return InvalidTypeId
	}
	return base
}

// SubstituteFields returns the field names and substituted field types for a
// specialization given its base template's field layout. Generic parameter
// references in base fields (matched by Name) are replaced by the
// corresponding entry in typeArgs, recursing through Ptr/Ref/MutRef/Slice/
// Array/Tuple/Struct/Enum/Channel/Func compositions.
//
// Callers pass baseId (the template obtained via BaseOf) and the
// specialization's TypeArgs. If baseId is invalid the result is (nil, nil).
//
// Parameter names are taken from the template's own field types by position:
// the i-th GenericParam entry encountered in the template corresponds to
// typeArgs[i] via name matching. Name matching is used (not positional)
// because the template's fields reference params by name.
func (tt *TypeTable) SubstituteFields(baseId TypeId, typeArgs []TypeId) ([]string, []TypeId) {
	if baseId == InvalidTypeId {
		return nil, nil
	}
	base := tt.Get(baseId)
	if len(base.Fields) == 0 {
		return nil, nil
	}
	params := tt.collectParamNames(baseId)
	if len(params) == 0 {
		// Template has no generic params referenced in its fields — return a
		// copy so callers don't alias the template's slices.
		names := append([]string(nil), base.FieldNames...)
		types := append([]TypeId(nil), base.Fields...)
		return names, types
	}
	types := make([]TypeId, len(base.Fields))
	for i, f := range base.Fields {
		types[i] = tt.substituteType(f, params, typeArgs, map[TypeId]TypeId{})
	}
	var names []string
	if len(base.FieldNames) > 0 {
		names = append([]string(nil), base.FieldNames...)
	}
	return names, types
}

// collectParamNames returns the generic-parameter names referenced by the
// template's fields, in first-seen order.
func (tt *TypeTable) collectParamNames(id TypeId) []string {
	var params []string
	seenName := map[string]bool{}
	seenId := map[TypeId]bool{}
	var walk func(TypeId)
	walk = func(x TypeId) {
		if x == InvalidTypeId || seenId[x] {
			return
		}
		seenId[x] = true
		e := tt.Get(x)
		if e.Kind == KindGenericParam {
			if !seenName[e.Name] {
				seenName[e.Name] = true
				params = append(params, e.Name)
			}
			return
		}
		if e.Elem != InvalidTypeId {
			walk(e.Elem)
		}
		for _, f := range e.Fields {
			walk(f)
		}
		for _, a := range e.TypeArgs {
			walk(a)
		}
		if e.Kind == KindFunc {
			walk(e.ReturnType)
		}
	}
	base := tt.Get(id)
	for _, f := range base.Fields {
		walk(f)
	}
	return params
}

// substituteType is the recursive core of SubstituteFields. It mirrors the
// recursion in monomorph.Context.Substitute but lives on the TypeTable so
// that codegen can use it without importing monomorph. The duplication is
// documented; a future cleanup can switch monomorph.Substitute to delegate.
func (tt *TypeTable) substituteType(ty TypeId, params []string, args []TypeId, memo map[TypeId]TypeId) TypeId {
	if cached, ok := memo[ty]; ok {
		return cached
	}
	e := tt.Get(ty)
	if e.Kind == KindGenericParam {
		for i, name := range params {
			if e.Name == name && i < len(args) {
				memo[ty] = args[i]
				return args[i]
			}
		}
		memo[ty] = ty
		return ty
	}
	switch e.Kind {
	case KindRef:
		out := tt.InternRef(tt.substituteType(e.Elem, params, args, memo))
		memo[ty] = out
		return out
	case KindMutRef:
		out := tt.InternMutRef(tt.substituteType(e.Elem, params, args, memo))
		memo[ty] = out
		return out
	case KindPtr:
		out := tt.InternPtr(tt.substituteType(e.Elem, params, args, memo))
		memo[ty] = out
		return out
	case KindSlice:
		out := tt.InternSlice(tt.substituteType(e.Elem, params, args, memo))
		memo[ty] = out
		return out
	case KindArray:
		out := tt.InternArray(tt.substituteType(e.Elem, params, args, memo), e.ArrayLen)
		memo[ty] = out
		return out
	case KindChannel:
		out := tt.InternChannel(tt.substituteType(e.Elem, params, args, memo))
		memo[ty] = out
		return out
	case KindTuple:
		newFields := make([]TypeId, len(e.Fields))
		for i, f := range e.Fields {
			newFields[i] = tt.substituteType(f, params, args, memo)
		}
		out := tt.InternTuple(newFields)
		memo[ty] = out
		return out
	case KindFunc:
		newParams := make([]TypeId, len(e.Fields))
		for i, f := range e.Fields {
			newParams[i] = tt.substituteType(f, params, args, memo)
		}
		newRet := tt.substituteType(e.ReturnType, params, args, memo)
		out := tt.InternFunc(newParams, newRet)
		memo[ty] = out
		return out
	case KindStruct:
		if len(e.TypeArgs) == 0 {
			memo[ty] = ty
			return ty
		}
		newArgs := make([]TypeId, len(e.TypeArgs))
		for i, a := range e.TypeArgs {
			newArgs[i] = tt.substituteType(a, params, args, memo)
		}
		out := tt.InternStruct(e.Module, e.Name, newArgs)
		memo[ty] = out
		return out
	case KindEnum:
		if len(e.TypeArgs) == 0 {
			memo[ty] = ty
			return ty
		}
		newArgs := make([]TypeId, len(e.TypeArgs))
		for i, a := range e.TypeArgs {
			newArgs[i] = tt.substituteType(a, params, args, memo)
		}
		out := tt.InternEnum(e.Module, e.Name, newArgs)
		memo[ty] = out
		return out
	}
	memo[ty] = ty
	return ty
}
