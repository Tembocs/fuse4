package hir

import (
	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

// Builder constructs HIR nodes with guaranteed metadata defaults.
// All HIR nodes MUST be created through Builder methods, not ad hoc (Rule 3.3).
type Builder struct {
	Types *typetable.TypeTable
}

// NewBuilder creates a builder backed by the given type table.
func NewBuilder(types *typetable.TypeTable) *Builder {
	return &Builder{Types: types}
}

func (b *Builder) base(span diagnostics.Span) nodeBase {
	return nodeBase{
		Span: span,
		MD: Metadata{
			Type:      b.Types.Unknown,
			Ownership: OwnerValue,
		},
	}
}

func (b *Builder) baseTyped(span diagnostics.Span, ty typetable.TypeId) nodeBase {
	return nodeBase{
		Span: span,
		MD: Metadata{
			Type:      ty,
			Ownership: OwnerValue,
		},
	}
}

// --- Expression builders ---

func (b *Builder) Literal(span diagnostics.Span, value string, ty typetable.TypeId) *LiteralExpr {
	return &LiteralExpr{nodeBase: b.baseTyped(span, ty), Value: value}
}

func (b *Builder) Ident(span diagnostics.Span, name string, ty typetable.TypeId) *IdentExpr {
	return &IdentExpr{nodeBase: b.baseTyped(span, ty), Name: name}
}

func (b *Builder) Binary(span diagnostics.Span, op string, left, right Expr, ty typetable.TypeId) *BinaryExpr {
	return &BinaryExpr{nodeBase: b.baseTyped(span, ty), Op: op, Left: left, Right: right}
}

func (b *Builder) Unary(span diagnostics.Span, op string, operand Expr, ty typetable.TypeId) *UnaryExpr {
	return &UnaryExpr{nodeBase: b.baseTyped(span, ty), Op: op, Operand: operand}
}

func (b *Builder) Assign(span diagnostics.Span, op string, target, value Expr) *AssignExpr {
	return &AssignExpr{
		nodeBase: b.baseTyped(span, b.Types.Unit),
		Op:       op,
		Target:   target,
		Value:    value,
	}
}

func (b *Builder) Call(span diagnostics.Span, callee Expr, args []Expr, retType typetable.TypeId) *CallExpr {
	return &CallExpr{nodeBase: b.baseTyped(span, retType), Callee: callee, Args: args}
}

func (b *Builder) Index(span diagnostics.Span, expr, index Expr, elemType typetable.TypeId) *IndexExpr {
	return &IndexExpr{nodeBase: b.baseTyped(span, elemType), Expr: expr, Index: index}
}

func (b *Builder) Field(span diagnostics.Span, expr Expr, name string, ty typetable.TypeId) *FieldExpr {
	return &FieldExpr{nodeBase: b.baseTyped(span, ty), Expr: expr, Name: name}
}

func (b *Builder) QDot(span diagnostics.Span, expr Expr, name string, ty typetable.TypeId) *QDotExpr {
	return &QDotExpr{nodeBase: b.baseTyped(span, ty), Expr: expr, Name: name}
}

func (b *Builder) Question(span diagnostics.Span, expr Expr, ty typetable.TypeId) *QuestionExpr {
	return &QuestionExpr{nodeBase: b.baseTyped(span, ty), Expr: expr}
}

func (b *Builder) BlockExpr(span diagnostics.Span, stmts []Stmt, tail Expr, ty typetable.TypeId) *Block {
	return &Block{nodeBase: b.baseTyped(span, ty), Stmts: stmts, Tail: tail}
}

func (b *Builder) If(span diagnostics.Span, cond Expr, then *Block, els Expr, ty typetable.TypeId) *IfExpr {
	return &IfExpr{nodeBase: b.baseTyped(span, ty), Cond: cond, Then: then, Else: els}
}

func (b *Builder) Match(span diagnostics.Span, subject Expr, arms []MatchArm, ty typetable.TypeId) *MatchExpr {
	return &MatchExpr{nodeBase: b.baseTyped(span, ty), Subject: subject, Arms: arms}
}

func (b *Builder) For(span diagnostics.Span, binding string, iter Expr, body *Block) *ForExpr {
	return &ForExpr{
		nodeBase: b.baseTyped(span, b.Types.Unit),
		Binding:  binding,
		Iterable: iter,
		Body:     body,
	}
}

func (b *Builder) While(span diagnostics.Span, cond Expr, body *Block) *WhileExpr {
	return &WhileExpr{nodeBase: b.baseTyped(span, b.Types.Unit), Cond: cond, Body: body}
}

func (b *Builder) Loop(span diagnostics.Span, body *Block, ty typetable.TypeId) *LoopExpr {
	return &LoopExpr{nodeBase: b.baseTyped(span, ty), Body: body}
}

func (b *Builder) Return(span diagnostics.Span, value Expr) *ReturnExpr {
	n := &ReturnExpr{nodeBase: b.baseTyped(span, b.Types.Never), Value: value}
	n.MD.Diverges = true
	return n
}

func (b *Builder) Break(span diagnostics.Span, value Expr) *BreakExpr {
	n := &BreakExpr{nodeBase: b.baseTyped(span, b.Types.Never), Value: value}
	n.MD.Diverges = true
	return n
}

func (b *Builder) Continue(span diagnostics.Span) *ContinueExpr {
	n := &ContinueExpr{nodeBase: b.baseTyped(span, b.Types.Never)}
	n.MD.Diverges = true
	return n
}

func (b *Builder) Spawn(span diagnostics.Span, expr Expr) *SpawnExpr {
	return &SpawnExpr{nodeBase: b.baseTyped(span, b.Types.Unit), Expr: expr}
}

func (b *Builder) Tuple(span diagnostics.Span, elems []Expr, ty typetable.TypeId) *TupleExpr {
	return &TupleExpr{nodeBase: b.baseTyped(span, ty), Elems: elems}
}

func (b *Builder) StructLit(span diagnostics.Span, name string, fields []FieldInitHIR, ty typetable.TypeId) *StructLitExpr {
	return &StructLitExpr{nodeBase: b.baseTyped(span, ty), Name: name, Fields: fields}
}

func (b *Builder) EnumInit(span diagnostics.Span, variantName string, tag int, args []Expr, ty typetable.TypeId) *EnumInitExpr {
	return &EnumInitExpr{nodeBase: b.baseTyped(span, ty), VariantName: variantName, Tag: tag, Args: args}
}

func (b *Builder) Closure(span diagnostics.Span, params []Param, retType typetable.TypeId, body *Block, ty typetable.TypeId) *ClosureExpr {
	return &ClosureExpr{nodeBase: b.baseTyped(span, ty), Params: params, ReturnType: retType, Body: body}
}

// --- Statement builders ---

func (b *Builder) Let(span diagnostics.Span, name string, ty typetable.TypeId, value Expr) *LetStmt {
	return &LetStmt{nodeBase: b.baseTyped(span, b.Types.Unit), Name: name, Type: ty, Value: value}
}

func (b *Builder) Var(span diagnostics.Span, name string, ty typetable.TypeId, value Expr) *VarStmt {
	return &VarStmt{nodeBase: b.baseTyped(span, b.Types.Unit), Name: name, Type: ty, Value: value}
}

func (b *Builder) ExprStatement(span diagnostics.Span, expr Expr) *ExprStmt {
	return &ExprStmt{nodeBase: b.baseTyped(span, b.Types.Unit), Expr: expr}
}
