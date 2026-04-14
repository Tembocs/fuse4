package check

import (
	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

// primMethod describes a built-in method on a primitive type.
type primMethod struct {
	ReceiverKinds []typetable.TypeKind
	Name          string
	ReturnType    func(c *Checker) typetable.TypeId
}

// registerPrimitiveMethods populates function types for the primitive method
// surface defined in the language guide (section 3.6). This runs after
// signature registration but before body checking.
func (c *Checker) registerPrimitiveMethods() {
	// integer methods: toFloat, toInt, abs, min, max
	for _, name := range []string{"I8", "I16", "I32", "I64", "I128", "ISize",
		"U8", "U16", "U32", "U64", "U128", "USize"} {
		ty := c.Types.LookupPrimitive(name)
		c.registerPrimMethod(name, "toFloat", []typetable.TypeId{}, c.Types.F64)
		c.registerPrimMethod(name, "toInt", []typetable.TypeId{}, c.Types.I64)
		c.registerPrimMethod(name, "abs", []typetable.TypeId{}, ty)
		c.registerPrimMethod(name, "min", []typetable.TypeId{ty}, ty)
		c.registerPrimMethod(name, "max", []typetable.TypeId{ty}, ty)
	}

	// float methods: toInt, isNan, isInfinite, floor, ceil, sqrt, abs
	for _, name := range []string{"F32", "F64"} {
		ty := c.Types.LookupPrimitive(name)
		c.registerPrimMethod(name, "toInt", []typetable.TypeId{}, c.Types.I64)
		c.registerPrimMethod(name, "isNan", []typetable.TypeId{}, c.Types.Bool)
		c.registerPrimMethod(name, "isInfinite", []typetable.TypeId{}, c.Types.Bool)
		c.registerPrimMethod(name, "floor", []typetable.TypeId{}, ty)
		c.registerPrimMethod(name, "ceil", []typetable.TypeId{}, ty)
		c.registerPrimMethod(name, "sqrt", []typetable.TypeId{}, ty)
		c.registerPrimMethod(name, "abs", []typetable.TypeId{}, ty)
	}

	// Char methods: toInt, isLetter, isDigit, isWhitespace
	c.registerPrimMethod("Char", "toInt", []typetable.TypeId{}, c.Types.U32)
	c.registerPrimMethod("Char", "isLetter", []typetable.TypeId{}, c.Types.Bool)
	c.registerPrimMethod("Char", "isDigit", []typetable.TypeId{}, c.Types.Bool)
	c.registerPrimMethod("Char", "isWhitespace", []typetable.TypeId{}, c.Types.Bool)

	// Bool methods: not
	c.registerPrimMethod("Bool", "not", []typetable.TypeId{}, c.Types.Bool)
}

func (c *Checker) registerPrimMethod(typeName, methodName string, params []typetable.TypeId, ret typetable.TypeId) {
	fty := c.Types.InternFunc(params, ret)
	c.funcTypes[typeName+"."+methodName] = fty
}

// lookupMethod resolves a method call on a receiver type.
// It searches: struct methods, primitive methods, trait methods (with bound chain).
func (c *Checker) lookupMethod(recvType typetable.TypeId, method string, span diagnostics.Span) typetable.TypeId {
	te := c.Types.Get(recvType)

	// 1. Primitive methods (by type name).
	key := te.Name + "." + method
	if fty, ok := c.funcTypes[key]; ok {
		fe := c.Types.Get(fty)
		if fe.Kind == typetable.KindFunc {
			return fe.ReturnType
		}
	}

	// 2. Struct/enum inherent methods (registered during impl scanning).
	if te.Kind == typetable.KindStruct || te.Kind == typetable.KindEnum {
		if fty, ok := c.funcTypes[te.Name+"."+method]; ok {
			fe := c.Types.Get(fty)
			if fe.Kind == typetable.KindFunc {
				return fe.ReturnType
			}
		}
	}

	// 3. Trait methods — search all traits implemented by this type,
	//    including supertraits (bound-chain lookup).
	return c.lookupTraitMethod(recvType, method, span)
}

// lookupTraitMethod searches trait methods with supertrait chain traversal.
func (c *Checker) lookupTraitMethod(recvType typetable.TypeId, method string, span diagnostics.Span) typetable.TypeId {
	te := c.Types.Get(recvType)
	typeName := te.Name

	// Find all traits implemented by this type.
	for traitName := range c.traitMethods {
		implKey := traitName + ":" + typeName
		if !c.traitImpls[implKey] {
			continue
		}
		// Search this trait and its supertraits.
		if ret := c.searchTraitChain(traitName, method); ret != typetable.InvalidTypeId {
			return ret
		}
	}

	return c.Types.Unknown
}

// searchTraitChain recursively searches a trait and its supertraits for a method.
func (c *Checker) searchTraitChain(traitName, method string) typetable.TypeId {
	// Check this trait's methods.
	if methods, ok := c.traitMethods[traitName]; ok {
		for _, m := range methods {
			if m.Name == method {
				return m.ReturnType
			}
		}
	}

	// Recurse into supertraits.
	if supers, ok := c.traitSupers[traitName]; ok {
		for _, sup := range supers {
			if ret := c.searchTraitChain(sup, method); ret != typetable.InvalidTypeId {
				return ret
			}
		}
	}

	return typetable.InvalidTypeId
}
