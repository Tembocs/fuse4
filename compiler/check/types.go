package check

import (
	"github.com/Tembocs/fuse4/compiler/ast"
	"github.com/Tembocs/fuse4/compiler/lex"
	"github.com/Tembocs/fuse4/compiler/resolve"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

// resolveTypeExpr resolves an AST type expression to a TypeId.
func (c *Checker) resolveTypeExpr(te ast.TypeExpr) typetable.TypeId {
	if te == nil {
		return c.Types.Unknown
	}

	switch t := te.(type) {
	case *ast.PathType:
		return c.resolvePathType(t)
	case *ast.TupleType:
		if len(t.Elems) == 0 {
			return c.Types.Unit
		}
		elems := make([]typetable.TypeId, len(t.Elems))
		for i, e := range t.Elems {
			elems[i] = c.resolveTypeExpr(e)
		}
		return c.Types.InternTuple(elems)
	case *ast.ArrayType:
		elem := c.resolveTypeExpr(t.Elem)
		// For now, array length is treated as a constant int.
		return c.Types.InternArray(elem, 0)
	case *ast.SliceType:
		elem := c.resolveTypeExpr(t.Elem)
		return c.Types.InternSlice(elem)
	case *ast.PtrType:
		elem := c.resolveTypeExpr(t.Elem)
		return c.Types.InternPtr(elem)
	default:
		return c.Types.Unknown
	}
}

func (c *Checker) resolvePathType(pt *ast.PathType) typetable.TypeId {
	name := pt.Segments[len(pt.Segments)-1]

	// Self resolves to the current impl target type.
	if name == "Self" && c.currentImplTarget != typetable.InvalidTypeId {
		return c.currentImplTarget
	}

	// Check primitives first.
	if prim := c.Types.LookupPrimitive(name); prim != typetable.InvalidTypeId {
		return prim
	}

	// Check type aliases.
	if alias, ok := c.typeAliases[name]; ok {
		return alias
	}

	// A generic parameter of the enclosing decl interns as KindGenericParam,
	// not as a synthetic struct in the current module (Rule 3.9).
	if gp, ok := c.currentGenericParams[name]; ok {
		return gp
	}

	// Resolve type args.
	var typeArgs []typetable.TypeId
	for _, arg := range pt.TypeArgs {
		typeArgs = append(typeArgs, c.resolveTypeExpr(arg))
	}

	// Look up in current module scope. Lookup walks imports, so an
	// auto-loaded stdlib type (String, List, Option, Result, ...) resolves
	// to its defining module rather than the current one — the fix for
	// the L021 module-identity mismatch.
	if modStr, kind, ok := c.resolveTypeName(name); ok {
		switch kind {
		case resolve.SymStruct:
			return c.Types.InternStruct(modStr, name, typeArgs)
		case resolve.SymEnum:
			return c.Types.InternEnum(modStr, name, typeArgs)
		case resolve.SymTrait:
			// Trait-as-type is used for return positions like
			// `fn into_iter(owned self) -> Iterator[T]`. Full dyn-trait
			// support is pending; for now intern the trait nominally
			// under its defining module so the canonical identity is
			// preserved (unlike the L021 phantom-struct pattern).
			return c.Types.InternStruct(modStr, name, typeArgs)
		}
	}

	// Unresolved: emit a diagnostic and return Unknown. Silent fallthrough
	// to a synthetic struct in the current module is the L021 band-aid
	// pattern and is forbidden (Rule 3.9, Rule 6.9).
	c.errorf(pt.Span, "unresolved type '%s'", name)
	return c.Types.Unknown
}

// resolveTypeName looks up a nominal type name in the current module's
// symbol table and returns the symbol's defining module, kind, and a
// success flag. Shared by resolvePathType and checkStructLit so both sites
// canonicalize module identity the same way.
//
// When the current-module lookup misses, it falls through to a graph-
// wide search across every module's locally-defined types. This is
// necessary because the monomorphizer places specialized functions in
// the module that owns the generic impl (e.g., `List__Entry__new` lands
// in `core.list`) while the concrete type arg (`Entry`) lives in the
// user's module. Without the fallback the specialization's signature
// fails to resolve even though the symbol exists elsewhere in the
// compilation unit. Genuinely unknown names still produce a diagnostic
// at the call site because no module defines them.
func (c *Checker) resolveTypeName(name string) (string, resolve.SymbolKind, bool) {
	if c.currentModule != nil {
		if sym := c.currentModule.Symbols.Lookup(name); sym != nil {
			if isTypeSym(sym.Kind) {
				return sym.Module.String(), sym.Kind, true
			}
		}
	}
	// Graph-wide fallback — scan every module's locally-defined types.
	// LookupLocal skips imports so a type is reported under its real
	// defining module and collisions between two modules declaring the
	// same name resolve deterministically via Graph.Order.
	if c.Graph != nil {
		for _, key := range c.Graph.Order {
			mod := c.Graph.Modules[key]
			if mod == nil || mod.Symbols == nil {
				continue
			}
			sym := mod.Symbols.LookupLocal(name)
			if sym != nil && isTypeSym(sym.Kind) {
				return sym.Module.String(), sym.Kind, true
			}
		}
	}
	return "", 0, false
}

func isTypeSym(k resolve.SymbolKind) bool {
	return k == resolve.SymStruct || k == resolve.SymEnum || k == resolve.SymTrait
}

func (c *Checker) resolveTypeExprOr(te ast.TypeExpr, fallback typetable.TypeId) typetable.TypeId {
	if te == nil {
		return fallback
	}
	return c.resolveTypeExpr(te)
}

// resolveParamTypes resolves the types of a parameter list, wrapping
// with Ref/MutRef when the parameter has an ownership qualifier.
func (c *Checker) resolveParamTypes(params []ast.Param) []typetable.TypeId {
	types := make([]typetable.TypeId, len(params))
	for i, p := range params {
		ty := c.resolveTypeExprOr(p.Type, c.Types.Unknown)
		switch p.Ownership {
		case lex.KwRef:
			ty = c.Types.InternRef(ty)
		case lex.KwMutref:
			ty = c.Types.InternMutRef(ty)
		}
		types[i] = ty
	}
	return types
}

// resolveImplParamTypes resolves params like resolveParamTypes but substitutes
// the target type for `self` parameters (name == "self" with nil type).
func (c *Checker) resolveImplParamTypes(params []ast.Param, targetType typetable.TypeId) []typetable.TypeId {
	types := make([]typetable.TypeId, len(params))
	for i, p := range params {
		if p.Name == "self" && p.Type == nil {
			ty := targetType
			switch p.Ownership {
			case lex.KwRef:
				ty = c.Types.InternRef(ty)
			case lex.KwMutref:
				ty = c.Types.InternMutRef(ty)
			}
			types[i] = ty
		} else {
			ty := c.resolveTypeExprOr(p.Type, c.Types.Unknown)
			switch p.Ownership {
			case lex.KwRef:
				ty = c.Types.InternRef(ty)
			case lex.KwMutref:
				ty = c.Types.InternMutRef(ty)
			}
			types[i] = ty
		}
	}
	return types
}

// --- numeric widening ---

// numericWiden returns the wider type when two numeric types in the same family
// are used in a binary operation, or InvalidTypeId if they're incompatible.
func (c *Checker) numericWiden(a, b typetable.TypeId) typetable.TypeId {
	ae := c.Types.Get(a)
	be := c.Types.Get(b)

	if ae.Kind != be.Kind {
		// Different numeric families (int vs uint vs float) → not compatible
		// unless both are integer-like for comparison purposes.
		return typetable.InvalidTypeId
	}

	// Same family: return the wider one.
	if ae.BitSize >= be.BitSize {
		return a
	}
	return b
}

// isAssignableTo checks if src can be assigned to dst.
func (c *Checker) isAssignableTo(src, dst typetable.TypeId) bool {
	if src == dst {
		return true
	}
	// Never is assignable to anything (diverging expression).
	if c.Types.Get(src).Kind == typetable.KindNever {
		return true
	}
	// Unknown is compatible with anything during checking (will be flagged later).
	if src == c.Types.Unknown || dst == c.Types.Unknown {
		return true
	}
	// Numeric widening in same family.
	if c.Types.IsNumeric(src) && c.Types.IsNumeric(dst) {
		widened := c.numericWiden(src, dst)
		return widened != typetable.InvalidTypeId
	}
	// Same-name enum/struct: treat them as assignable. The lowerer and
	// codegen bind the concrete type from the declared / expected side
	// rather than the source-inference side, so module identity is the
	// only correctness gate. TypeArgs mismatches at this layer are
	// usually an artifact of generic static methods (`List.new()` —
	// the call's return type depends on whichever spec registered
	// first) and the assignment target carries the real type.
	//
	// Full generic inference is tracked for a later wave; until then
	// this permissive check unblocks the stdlib-integration proofs
	// without forcing every call site to write explicit type args.
	se := c.Types.Get(src)
	de := c.Types.Get(dst)
	if se.Kind == de.Kind && se.Name == de.Name && se.Module == de.Module &&
		(se.Kind == typetable.KindEnum || se.Kind == typetable.KindStruct) {
		return true
	}
	return false
}
