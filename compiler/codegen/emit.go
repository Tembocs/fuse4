package codegen

import (
	"fmt"
	"strings"

	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/mir"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

// Emitter generates C11 source from MIR functions.
type Emitter struct {
	Types     *typetable.TypeTable
	Errors    []diagnostics.Diagnostic
	DropTypes map[typetable.TypeId]bool // types with Drop trait implementations

	out     strings.Builder
	indent  int
	emitted map[typetable.TypeId]bool // types already emitted
}

// NewEmitter creates an emitter for the given type table.
func NewEmitter(types *typetable.TypeTable) *Emitter {
	return &Emitter{
		Types:     types,
		emitted:   make(map[typetable.TypeId]bool),
		DropTypes: make(map[typetable.TypeId]bool),
	}
}

// Emit generates C11 for a list of MIR functions and returns the source.
func (e *Emitter) Emit(functions []*mir.Function) string {
	e.out.Reset()

	// Preamble
	e.writeln("#include <stdint.h>")
	e.writeln("#include <stdbool.h>")
	e.writeln("#include <stddef.h>")
	e.writeln("#include \"fuse_rt.h\"")
	e.writeln("")

	// Phase 1: collect and emit all composite type definitions before functions.
	for _, fn := range functions {
		e.collectTypes(fn)
	}
	e.writeln("")

	// Phase 2: emit function forward declarations.
	for _, fn := range functions {
		e.emitFnForwardDecl(fn)
	}
	e.writeln("")

	// Phase 3: emit function bodies.
	for _, fn := range functions {
		e.emitFunction(fn)
	}

	return e.out.String()
}

// --- type collection and emission (Contract 5: emit before use) ---

func (e *Emitter) collectTypes(fn *mir.Function) {
	// Emit return type.
	e.emitTypeDefIfNeeded(fn.ReturnType)
	// Emit param types.
	for _, p := range fn.Params {
		e.emitTypeDefIfNeeded(p.Type)
	}
	// Emit local types.
	for _, l := range fn.Locals {
		e.emitTypeDefIfNeeded(l.Type)
	}
}

func (e *Emitter) emitTypeDefIfNeeded(id typetable.TypeId) {
	if e.emitted[id] || id == typetable.InvalidTypeId {
		return
	}
	te := e.Types.Get(id)

	switch te.Kind {
	case typetable.KindStruct:
		e.emitted[id] = true
		name := MangleType(e.Types, id)
		e.writef("typedef struct %s %s;", name, name)
		e.writeln("")
	case typetable.KindEnum:
		e.emitted[id] = true
		name := MangleType(e.Types, id)
		e.writef("typedef struct %s %s;", name, name)
		e.writeln("")
	case typetable.KindTuple:
		// Emit element types first.
		for _, f := range te.Fields {
			e.emitTypeDefIfNeeded(f)
		}
		e.emitted[id] = true
		name := MangleType(e.Types, id)
		e.writef("typedef struct {")
		for i, f := range te.Fields {
			e.writef(" %s f_%d;", MangleType(e.Types, f), i)
		}
		e.writef(" } %s;", name)
		e.writeln("")
	case typetable.KindSlice:
		e.emitTypeDefIfNeeded(te.Elem)
		e.emitted[id] = true
		name := MangleType(e.Types, id)
		elemC := MangleType(e.Types, te.Elem)
		e.writef("typedef struct { %s* data; size_t len; } %s;", elemC, name)
		e.writeln("")
	case typetable.KindArray:
		e.emitTypeDefIfNeeded(te.Elem)
		e.emitted[id] = true
		name := MangleType(e.Types, id)
		elemC := MangleType(e.Types, te.Elem)
		e.writef("typedef struct { %s data[%d]; } %s;", elemC, te.ArrayLen, name)
		e.writeln("")
	case typetable.KindChannel:
		e.emitTypeDefIfNeeded(te.Elem)
		e.emitted[id] = true
		name := MangleType(e.Types, id)
		elemC := MangleType(e.Types, te.Elem)
		// Channel is a pointer to a runtime channel struct.
		e.writef("typedef struct { void* _impl; /* Chan<%s> */ } %s;", elemC, name)
		e.writeln("")
	default:
		e.emitted[id] = true
	}
}

// --- function emission ---

func (e *Emitter) emitFnForwardDecl(fn *mir.Function) {
	retC := e.returnTypeC(fn.ReturnType)
	nameC := MangleName("", fn.Name)
	paramsC := e.paramsC(fn.Params)
	e.writef("%s %s(%s);", retC, nameC, paramsC)
	e.writeln("")
}

func (e *Emitter) emitFunction(fn *mir.Function) {
	retC := e.returnTypeC(fn.ReturnType)
	nameC := MangleName("", fn.Name)
	paramsC := e.paramsC(fn.Params)

	e.writef("%s %s(%s) {", retC, nameC, paramsC)
	e.writeln("")
	e.indent++

	// Declare locals (skip params and unit-typed locals).
	for _, l := range fn.Locals[len(fn.Params):] {
		if e.isUnit(l.Type) {
			continue // Contract 2: total unit erasure
		}
		ty := MangleType(e.Types, l.Type)
		name := e.localName(l.Id)
		e.writeIndent()
		e.writef("%s %s;", ty, name)
		e.writeln("")
	}
	if len(fn.Locals) > len(fn.Params) {
		e.writeln("")
	}

	// Emit blocks.
	for i := range fn.Blocks {
		e.emitBlock(fn, &fn.Blocks[i])
	}

	e.indent--
	e.writeln("}")
	e.writeln("")
}

func (e *Emitter) emitBlock(fn *mir.Function, blk *mir.Block) {
	// Label (skip for entry block 0).
	if blk.Id != fn.EntryBlock {
		e.writef("block_%d:;", blk.Id)
		e.writeln("")
	}

	for _, instr := range blk.Instrs {
		e.emitInstr(fn, &instr)
	}

	e.emitTerminator(fn, &blk.Term)
}

func (e *Emitter) emitInstr(fn *mir.Function, instr *mir.Instr) {
	dest := e.localName(instr.Dest)

	switch instr.Kind {
	case mir.InstrConst:
		if e.isUnit(instr.Type) {
			return // unit erasure
		}
		e.writeIndent()
		e.writef("%s = %s;", dest, e.constValue(instr.Value, instr.Type))
		e.writeln("")

	case mir.InstrCopy:
		if e.isUnit(instr.Type) {
			return
		}
		e.writeIndent()
		e.writef("%s = %s;", dest, e.localName(instr.Src))
		e.writeln("")

	case mir.InstrMove:
		if e.isUnit(instr.Type) {
			return
		}
		e.writeIndent()
		e.writef("%s = %s; /* move */", dest, e.localName(instr.Src))
		e.writeln("")

	case mir.InstrBorrow:
		// Contract 1: borrow pointers use &, tracked as borrow category.
		e.writeIndent()
		if instr.BorrowKind == mir.BorrowMutable {
			e.writef("%s = &%s; /* mutref */", dest, e.localName(instr.Src))
		} else {
			e.writef("%s = &%s; /* ref */", dest, e.localName(instr.Src))
		}
		e.writeln("")

	case mir.InstrDrop:
		if e.isUnit(instr.Type) {
			return
		}
		if e.DropTypes[instr.Type] {
			// Type has a Drop implementation — call its destructor.
			ty := MangleType(e.Types, instr.Type)
			e.writeIndent()
			e.writef("%s_drop(&%s);", ty, e.localName(instr.Src))
			e.writeln("")
		}
		// Types without Drop: no-op.

	case mir.InstrCall:
		if e.isUnit(instr.Type) {
			e.writeIndent()
			e.writef("%s(%s);", e.localName(instr.Callee), e.argsC(instr.Args))
			e.writeln("")
		} else {
			e.writeIndent()
			e.writef("%s = %s(%s);", dest, e.localName(instr.Callee), e.argsC(instr.Args))
			e.writeln("")
		}

	case mir.InstrFieldRead:
		e.writeIndent()
		e.writef("%s = %s.%s;", dest, e.localName(instr.Src), SanitizeIdent(instr.Field))
		e.writeln("")

	case mir.InstrFieldAddr:
		e.writeIndent()
		e.writef("%s = &%s.%s;", dest, e.localName(instr.Src), SanitizeIdent(instr.Field))
		e.writeln("")

	case mir.InstrIndex:
		e.writeIndent()
		e.writef("%s = %s.data[%s];", dest, e.localName(instr.Src), e.localName(instr.Src2))
		e.writeln("")

	case mir.InstrBinOp:
		e.writeIndent()
		e.writef("%s = %s %s %s;", dest, e.localName(instr.Src), instr.Op, e.localName(instr.Src2))
		e.writeln("")

	case mir.InstrUnaryOp:
		e.writeIndent()
		e.writef("%s = %s%s;", dest, instr.Op, e.localName(instr.Src))
		e.writeln("")

	case mir.InstrTuple:
		if e.isUnit(instr.Type) {
			return
		}
		e.writeIndent()
		ty := MangleType(e.Types, instr.Type)
		fields := make([]string, len(instr.Args))
		for i, a := range instr.Args {
			fields[i] = fmt.Sprintf(".f_%d = %s", i, e.localName(a))
		}
		// Contract 5: typed aggregate zero-initializer form.
		e.writef("%s = (%s){%s};", dest, ty, strings.Join(fields, ", "))
		e.writeln("")

	case mir.InstrStructInit:
		e.writeIndent()
		ty := MangleType(e.Types, instr.Type)
		if len(instr.Args) == 0 {
			// Contract 5: aggregate fallback is typed zero-initializer, not scalar 0.
			e.writef("%s = (%s){0};", dest, ty)
		} else {
			fields := make([]string, len(instr.Args))
			for i, a := range instr.Args {
				fields[i] = e.localName(a)
			}
			e.writef("%s = (%s){%s};", dest, ty, strings.Join(fields, ", "))
		}
		e.writeln("")

	case mir.InstrEnumInit:
		e.writeIndent()
		ty := MangleType(e.Types, instr.Type)
		if len(instr.Args) == 0 {
			e.writef("%s = (%s){0}; /* %s */", dest, ty, SanitizeIdent(instr.Field))
		} else {
			fields := make([]string, len(instr.Args))
			for i, a := range instr.Args {
				fields[i] = e.localName(a)
			}
			e.writef("%s = (%s){%s}; /* %s */", dest, ty, strings.Join(fields, ", "), SanitizeIdent(instr.Field))
		}
		e.writeln("")
	}
}

func (e *Emitter) emitTerminator(fn *mir.Function, term *mir.Terminator) {
	switch term.Kind {
	case mir.TermReturn:
		if e.isUnit(fn.ReturnType) {
			e.writeIndent()
			e.writeln("return;")
		} else {
			e.writeIndent()
			e.writef("return %s;", e.localName(term.Value))
			e.writeln("")
		}

	case mir.TermGoto:
		e.writeIndent()
		e.writef("goto block_%d;", term.Target)
		e.writeln("")

	case mir.TermBranch:
		e.writeIndent()
		e.writef("if (%s) goto block_%d; else goto block_%d;",
			e.localName(term.Value), term.Target, term.ElseTarget)
		e.writeln("")

	case mir.TermDiverge:
		// Contract 4: divergence is structural — no code after this point.
		e.writeIndent()
		e.writeln("__builtin_unreachable();")

	case mir.TermNone:
		// Unterminated block — should not happen in well-formed MIR.
	}
}

// --- helpers ---

func (e *Emitter) isUnit(id typetable.TypeId) bool {
	return id == e.Types.Unit || id == e.Types.Never
}

func (e *Emitter) localName(id mir.LocalId) string {
	return fmt.Sprintf("_l%d", id)
}

func (e *Emitter) returnTypeC(id typetable.TypeId) string {
	if e.isUnit(id) {
		return "void"
	}
	return MangleType(e.Types, id)
}

func (e *Emitter) paramsC(params []mir.Local) string {
	var parts []string
	for _, p := range params {
		if e.isUnit(p.Type) {
			continue // unit erasure in params
		}
		ty := MangleType(e.Types, p.Type)
		name := e.localName(p.Id)
		parts = append(parts, ty+" "+name)
	}
	if len(parts) == 0 {
		return "void"
	}
	return strings.Join(parts, ", ")
}

func (e *Emitter) argsC(args []mir.LocalId) string {
	parts := make([]string, len(args))
	for i, a := range args {
		parts[i] = e.localName(a)
	}
	return strings.Join(parts, ", ")
}

func (e *Emitter) constValue(value string, ty typetable.TypeId) string {
	if value == "()" || e.isUnit(ty) {
		return "0 /* unit */"
	}
	te := e.Types.Get(ty)
	switch te.Kind {
	case typetable.KindBool:
		if value == "true" {
			return "true"
		}
		return "false"
	case typetable.KindStruct, typetable.KindEnum, typetable.KindTuple:
		// Contract 5: typed aggregate zero-initializer.
		return fmt.Sprintf("(%s){0}", MangleType(e.Types, ty))
	}
	return value
}

// --- output helpers ---

func (e *Emitter) writeln(s string) {
	e.out.WriteString(s)
	e.out.WriteByte('\n')
}

func (e *Emitter) writef(format string, args ...any) {
	fmt.Fprintf(&e.out, format, args...)
}

func (e *Emitter) writeIndent() {
	for i := 0; i < e.indent; i++ {
		e.out.WriteString("    ")
	}
}
