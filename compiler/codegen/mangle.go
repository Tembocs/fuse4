// Package codegen owns the bootstrap C11 backend.
package codegen

import (
	"strings"

	"github.com/Tembocs/fuse4/compiler/typetable"
)

// C keywords that must be escaped in identifiers.
var cKeywords = map[string]bool{
	"auto": true, "break": true, "case": true, "char": true,
	"const": true, "continue": true, "default": true, "do": true,
	"double": true, "else": true, "enum": true, "extern": true,
	"float": true, "for": true, "goto": true, "if": true,
	"inline": true, "int": true, "long": true, "register": true,
	"restrict": true, "return": true, "short": true, "signed": true,
	"sizeof": true, "static": true, "struct": true, "switch": true,
	"typedef": true, "union": true, "unsigned": true, "void": true,
	"volatile": true, "while": true, "_Bool": true, "_Complex": true,
	"_Imaginary": true, "_Alignas": true, "_Alignof": true,
	"_Atomic": true, "_Generic": true, "_Noreturn": true,
	"_Static_assert": true, "_Thread_local": true,
}

// SanitizeIdent makes a name safe for C: escapes C keywords and replaces
// illegal characters. Deterministic and stable across builds.
func SanitizeIdent(name string) string {
	name = strings.ReplaceAll(name, ".", "_")
	name = strings.ReplaceAll(name, "[", "_")
	name = strings.ReplaceAll(name, "]", "_")
	name = strings.ReplaceAll(name, ",", "_")
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "(", "_")
	name = strings.ReplaceAll(name, ")", "_")

	// Sanitize numeric field names (e.g. tuple field "0" → "f_0").
	if len(name) > 0 && name[0] >= '0' && name[0] <= '9' {
		name = "f_" + name
	}

	if cKeywords[name] {
		return "fuse_" + name
	}
	return name
}

// MangleName produces a module-qualified C identifier from a module path and
// item name. Same-name items from different modules must not collide.
// The "main" function is emitted as "main" (C entry point).
func MangleName(module, name string) string {
	if name == "main" {
		return "main"
	}
	if module == "" {
		return "Fuse_" + SanitizeIdent(name)
	}
	return "Fuse_" + SanitizeIdent(module) + "__" + SanitizeIdent(name)
}

// MangleType produces the C type name for a Fuse type.
func MangleType(tt *typetable.TypeTable, id typetable.TypeId) string {
	e := tt.Get(id)
	switch e.Kind {
	case typetable.KindUnit:
		return "void"
	case typetable.KindBool:
		return "bool"
	case typetable.KindChar:
		return "uint32_t"
	case typetable.KindInt:
		return intCType(e.Name, e.BitSize, true)
	case typetable.KindUint:
		return intCType(e.Name, e.BitSize, false)
	case typetable.KindFloat:
		if e.BitSize == 32 {
			return "float"
		}
		return "double"
	case typetable.KindNever:
		return "void"
	case typetable.KindPtr:
		return MangleType(tt, e.Elem) + "*"
	case typetable.KindRef, typetable.KindMutRef:
		// Borrow pointers are C pointers but tracked separately.
		return MangleType(tt, e.Elem) + "*"
	case typetable.KindSlice:
		return "FuseSlice_" + SanitizeIdent(MangleType(tt, e.Elem))
	case typetable.KindArray:
		return "FuseArray_" + SanitizeIdent(MangleType(tt, e.Elem))
	case typetable.KindTuple:
		return MangleTupleName(tt, e.Fields)
	case typetable.KindStruct:
		if len(e.TypeArgs) > 0 {
			return MangleName(e.Module, e.Name) + mangleTypeArgs(tt, e.TypeArgs)
		}
		return MangleName(e.Module, e.Name)
	case typetable.KindEnum:
		if len(e.TypeArgs) > 0 {
			return MangleName(e.Module, e.Name) + mangleTypeArgs(tt, e.TypeArgs)
		}
		return MangleName(e.Module, e.Name)
	case typetable.KindChannel:
		return "FuseChan_" + SanitizeIdent(MangleType(tt, e.Elem))
	case typetable.KindFunc:
		return "FuseFunc" // simplified; full function pointer mangling in later waves
	case typetable.KindUnknown:
		return "/* UNKNOWN */ int"
	default:
		return "int"
	}
}

func intCType(name string, bits int, signed bool) string {
	switch bits {
	case 8:
		if signed {
			return "int8_t"
		}
		return "uint8_t"
	case 16:
		if signed {
			return "int16_t"
		}
		return "uint16_t"
	case 32:
		if signed {
			return "int32_t"
		}
		return "uint32_t"
	case 64:
		if signed {
			return "int64_t"
		}
		return "uint64_t"
	case 128:
		if signed {
			return "__int128"
		}
		return "unsigned __int128"
	default:
		// Platform-width (ISize/USize)
		if signed {
			return "intptr_t"
		}
		return "uintptr_t"
	}
}

// mangleTypeArgs appends type argument names to a mangled identifier.
func mangleTypeArgs(tt *typetable.TypeTable, args []typetable.TypeId) string {
	s := "__"
	for i, a := range args {
		if i > 0 {
			s += "_"
		}
		s += SanitizeIdent(MangleType(tt, a))
	}
	return s
}

// MangleTupleName generates a stable name for a tuple type.
func MangleTupleName(tt *typetable.TypeTable, fields []typetable.TypeId) string {
	parts := make([]string, len(fields))
	for i, f := range fields {
		parts[i] = SanitizeIdent(MangleType(tt, f))
	}
	return "FuseTuple_" + strings.Join(parts, "_")
}
