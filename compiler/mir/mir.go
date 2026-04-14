// Package mir owns the explicit mid-level IR used by backends.
//
// MIR is a flat, block-based representation. Each function is a list of
// basic blocks, each block is a list of instructions plus a terminator.
// MIR is distinct from AST (syntax) and HIR (semantic).
package mir

import "github.com/Tembocs/fuse4/compiler/typetable"

// LocalId identifies a local variable or temporary in a function.
type LocalId int

// BlockId identifies a basic block in a function.
type BlockId int

// Function is the MIR representation of a single function.
type Function struct {
	Name       string
	Params     []Local
	ReturnType typetable.TypeId
	Locals     []Local
	Blocks     []Block
	EntryBlock BlockId
}

// Local is a local variable or temporary.
type Local struct {
	Id   LocalId
	Name string // empty for temporaries
	Type typetable.TypeId
}

// Block is a basic block: a sequence of instructions followed by a terminator.
type Block struct {
	Id     BlockId
	Instrs []Instr
	Term   Terminator
	Sealed bool // true once a terminator is set; no more instructions allowed
}

// --- Instructions ---

// InstrKind classifies an MIR instruction.
type InstrKind int

const (
	InstrConst       InstrKind = iota // dest = constant value
	InstrCopy                         // dest = src
	InstrMove                         // dest = move src (src invalidated)
	InstrBorrow                       // dest = ref/mutref src
	InstrDrop                         // drop(src)
	InstrCall                         // dest = callee(args...)
	InstrFieldRead                    // dest = src.field
	InstrFieldAddr                    // dest = &src.field
	InstrIndex                        // dest = src[index]
	InstrBinOp                        // dest = left op right
	InstrUnaryOp                      // dest = op operand
	InstrTuple                        // dest = (elems...)
	InstrStructInit                   // dest = StructName { fields... }
	InstrEnumInit                     // dest = EnumVariant(args...)
	InstrCast                         // dest = src as Type
)

// Instr is a single MIR instruction.
type Instr struct {
	Kind     InstrKind
	Dest     LocalId
	Type     typetable.TypeId
	Src      LocalId   // for copy, move, borrow, drop, field, index
	Src2     LocalId   // for binop right, index value
	Args     []LocalId // for call arguments, tuple/struct elements
	Callee   LocalId   // for calls
	Field    string    // for field read/addr
	Op       string    // for binop/unaryop
	Value    string    // for const literal
	IsMethod bool      // for calls: true if method (first arg is receiver)

	// BorrowKind distinguishes ref from mutref in InstrBorrow.
	BorrowKind BorrowKind
}

// BorrowKind distinguishes shared from mutable borrows.
type BorrowKind int

const (
	BorrowShared  BorrowKind = iota // ref
	BorrowMutable                   // mutref
)

// --- Terminators ---

// TermKind classifies a block terminator.
type TermKind int

const (
	TermNone       TermKind = iota // block not yet terminated
	TermReturn                     // return value
	TermGoto                       // unconditional jump
	TermBranch                     // conditional branch
	TermSwitch                     // multi-way branch (match)
	TermDiverge                    // unreachable (after panic/abort)
)

// Terminator ends a basic block.
type Terminator struct {
	Kind      TermKind
	Value     LocalId   // return value, branch condition
	Target    BlockId   // goto target, branch-true target
	ElseTarget BlockId  // branch-false target
	Targets   []BlockId // switch targets
}
