package monomorph

import (
	"fmt"

	"github.com/Tembocs/fuse4/compiler/typetable"
)

// Instantiation records a concrete usage of a generic function or type.
type Instantiation struct {
	GenericName string             // e.g. "core.option.Option"
	TypeArgs    []typetable.TypeId // e.g. [I32]
	Resolved    typetable.TypeId   // the concrete type ID after substitution
}

// Context collects and validates all generic instantiations in a program.
type Context struct {
	Types          *typetable.TypeTable
	Instantiations []Instantiation
	seen           map[string]typetable.TypeId // dedup key → resolved TypeId
}

// NewContext creates a monomorphization context.
func NewContext(types *typetable.TypeTable) *Context {
	return &Context{
		Types: types,
		seen:  make(map[string]typetable.TypeId),
	}
}

// Record registers a concrete instantiation of a generic name with type args.
// Returns the resolved concrete TypeId. Deduplicates identical instantiations.
func (c *Context) Record(genericName string, typeArgs []typetable.TypeId) typetable.TypeId {
	if len(typeArgs) == 0 {
		return typetable.InvalidTypeId // not generic
	}

	key := c.deduplicationKey(genericName, typeArgs)
	if resolved, ok := c.seen[key]; ok {
		return resolved
	}

	// Validate: all type args must be concrete (not Unknown or GenericParam).
	for _, arg := range typeArgs {
		if !c.Types.IsResolved(arg) {
			return typetable.InvalidTypeId // partial specialization rejected
		}
	}

	// Create the concrete type.
	resolved := c.Types.InternStruct("__mono", genericName, typeArgs)

	c.seen[key] = resolved
	c.Instantiations = append(c.Instantiations, Instantiation{
		GenericName: genericName,
		TypeArgs:    typeArgs,
		Resolved:    resolved,
	})

	return resolved
}

// IsGeneric checks if a type has unresolved generic parameters.
func (c *Context) IsGeneric(ty typetable.TypeId) bool {
	te := c.Types.Get(ty)
	if te.Kind == typetable.KindGenericParam {
		return true
	}
	for _, arg := range te.TypeArgs {
		if c.IsGeneric(arg) {
			return true
		}
	}
	return false
}

// Substitute replaces generic parameters in a type with concrete args.
func (c *Context) Substitute(ty typetable.TypeId, params []string, args []typetable.TypeId) typetable.TypeId {
	te := c.Types.Get(ty)

	// If it's a generic param, look it up in the substitution map.
	if te.Kind == typetable.KindGenericParam {
		for i, name := range params {
			if te.Name == name && i < len(args) {
				return args[i]
			}
		}
		return ty // unresolved — leave as is
	}

	// Recursively substitute in compound types.
	switch te.Kind {
	case typetable.KindRef:
		inner := c.Substitute(te.Elem, params, args)
		return c.Types.InternRef(inner)
	case typetable.KindMutRef:
		inner := c.Substitute(te.Elem, params, args)
		return c.Types.InternMutRef(inner)
	case typetable.KindPtr:
		inner := c.Substitute(te.Elem, params, args)
		return c.Types.InternPtr(inner)
	case typetable.KindSlice:
		inner := c.Substitute(te.Elem, params, args)
		return c.Types.InternSlice(inner)
	case typetable.KindArray:
		inner := c.Substitute(te.Elem, params, args)
		return c.Types.InternArray(inner, te.ArrayLen)
	case typetable.KindTuple:
		newFields := make([]typetable.TypeId, len(te.Fields))
		for i, f := range te.Fields {
			newFields[i] = c.Substitute(f, params, args)
		}
		return c.Types.InternTuple(newFields)
	case typetable.KindFunc:
		newParams := make([]typetable.TypeId, len(te.Fields))
		for i, f := range te.Fields {
			newParams[i] = c.Substitute(f, params, args)
		}
		newRet := c.Substitute(te.ReturnType, params, args)
		return c.Types.InternFunc(newParams, newRet)
	case typetable.KindStruct:
		if len(te.TypeArgs) > 0 {
			newArgs := make([]typetable.TypeId, len(te.TypeArgs))
			for i, a := range te.TypeArgs {
				newArgs[i] = c.Substitute(a, params, args)
			}
			return c.Types.InternStruct(te.Module, te.Name, newArgs)
		}
	case typetable.KindEnum:
		if len(te.TypeArgs) > 0 {
			newArgs := make([]typetable.TypeId, len(te.TypeArgs))
			for i, a := range te.TypeArgs {
				newArgs[i] = c.Substitute(a, params, args)
			}
			return c.Types.InternEnum(te.Module, te.Name, newArgs)
		}
	}

	return ty
}

func (c *Context) deduplicationKey(name string, args []typetable.TypeId) string {
	key := name + ":"
	for i, a := range args {
		if i > 0 {
			key += ","
		}
		key += fmt.Sprintf("%d", a)
	}
	return key
}
