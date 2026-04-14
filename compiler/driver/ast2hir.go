package driver

import (
	"fmt"

	"github.com/Tembocs/fuse4/compiler/ast"
	"github.com/Tembocs/fuse4/compiler/check"
	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/hir"
	"github.com/Tembocs/fuse4/compiler/lex"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

func zeroSpan() diagnostics.Span { return diagnostics.Span{} }

// ast2hir converts AST nodes to HIR nodes using the Builder.
type ast2hir struct {
	b       *hir.Builder
	tt      *typetable.TypeTable
	checker *check.Checker
	modPath string // current module path for qualified name lookups
}

// typeOf returns the checker's resolved type for an AST expression, falling
// back to Unknown if the checker is unavailable or the expression was not recorded.
func (a *ast2hir) typeOf(e ast.Expr) typetable.TypeId {
	if a.checker != nil && a.checker.ExprTypes != nil {
		if ty, ok := a.checker.ExprTypes[e]; ok {
			return ty
		}
	}
	return a.tt.Unknown
}

// valueTypeOf returns the checker's resolved type, but substitutes Unknown for
// Unit or Never. The lowerer allocates result temps for control-flow expressions
// (if, match, loop, block); if those temps have Unit/Never type, codegen omits
// their declarations, which breaks dead-code references. Using Unknown keeps
// the temp declared as a harmless int placeholder.
func (a *ast2hir) valueTypeOf(e ast.Expr) typetable.TypeId {
	ty := a.typeOf(e)
	if ty == a.tt.Unit || ty == a.tt.Never {
		return a.tt.Unknown
	}
	return ty
}

// typeExprString extracts the simple type name from an AST type expression.
func typeExprString(te ast.TypeExpr) string {
	if te == nil {
		return ""
	}
	if pt, ok := te.(*ast.PathType); ok && len(pt.Segments) > 0 {
		return pt.Segments[len(pt.Segments)-1]
	}
	return ""
}

// lowerFunction translates an AST FnDecl into a complete HIR Function.
func (a *ast2hir) lowerFunction(fn *ast.FnDecl) *hir.Function {
	var params []hir.Param
	for _, p := range fn.Params {
		pty := a.tt.Unknown
		if a.checker != nil {
			pty = a.checker.Types.LookupPrimitive(typeExprString(p.Type))
			if pty == typetable.InvalidTypeId {
				pty = a.tt.Unknown
			}
		}
		params = append(params, hir.Param{
			Span: p.Span,
			Name: p.Name,
			Type: pty,
		})
	}

	retType := a.tt.Unit
	if fn.ReturnType != nil {
		if a.checker != nil {
			qualName := a.modPath + "." + fn.Name
			retType = a.checker.FuncReturnType(qualName)
		} else {
			retType = a.tt.Unknown
		}
	}

	var body *hir.Block
	if fn.Body != nil {
		body = a.lowerBlock(fn.Body)
	} else {
		body = a.b.BlockExpr(fn.Span, nil, nil, retType)
	}

	return &hir.Function{
		Name:       fn.Name,
		Public:     fn.Public,
		Params:     params,
		ReturnType: retType,
		Body:       body,
	}
}

// lowerBlock translates an AST BlockExpr into an HIR Block.
func (a *ast2hir) lowerBlock(block *ast.BlockExpr) *hir.Block {
	if block == nil {
		return a.b.BlockExpr(zeroSpan(), nil, nil, a.tt.Unit)
	}

	var stmts []hir.Stmt
	for _, s := range block.Stmts {
		lowered := a.lowerStmt(s)
		if lowered != nil {
			stmts = append(stmts, lowered)
		}
	}

	var tail hir.Expr
	if block.Tail != nil {
		tail = a.lowerExpr(block.Tail)
	}

	ty := a.tt.Unit
	if tail != nil {
		ty = a.valueTypeOf(block.Tail)
	}

	return a.b.BlockExpr(block.Span, stmts, tail, ty)
}

// lowerStmt translates an AST Stmt into an HIR Stmt.
func (a *ast2hir) lowerStmt(s ast.Stmt) hir.Stmt {
	if s == nil {
		return nil
	}

	switch s := s.(type) {
	case *ast.LetStmt:
		var val hir.Expr
		ty := a.tt.Unknown
		if s.Value != nil {
			val = a.lowerExpr(s.Value)
			ty = a.typeOf(s.Value)
		}
		return a.b.Let(s.Span, s.Name, ty, val)

	case *ast.VarStmt:
		var val hir.Expr
		ty := a.tt.Unknown
		if s.Value != nil {
			val = a.lowerExpr(s.Value)
			ty = a.typeOf(s.Value)
		}
		return a.b.Var(s.Span, s.Name, ty, val)

	case *ast.ExprStmt:
		expr := a.lowerExpr(s.Expr)
		if expr == nil {
			return nil
		}
		return a.b.ExprStatement(s.Span, expr)

	case *ast.ItemStmt:
		// Item statements (e.g. nested fn decl) are not lowered here.
		return nil

	default:
		return nil
	}
}

// lowerExpr translates an AST Expr into an HIR Expr.
func (a *ast2hir) lowerExpr(e ast.Expr) hir.Expr {
	if e == nil {
		return nil
	}

	switch e := e.(type) {
	case *ast.LiteralExpr:
		return a.b.Literal(e.Span, e.Token.Literal, a.typeOf(e))

	case *ast.IdentExpr:
		return a.b.Ident(e.Span, e.Name, a.typeOf(e))

	case *ast.BinaryExpr:
		left := a.lowerExpr(e.Left)
		right := a.lowerExpr(e.Right)
		op := e.Op.Literal
		if op == "" {
			op = opFromTokenKind(e.Op.Kind)
		}
		return a.b.Binary(e.Span, op, left, right, a.typeOf(e))

	case *ast.UnaryExpr:
		operand := a.lowerExpr(e.Operand)
		op := unaryOpFromToken(e.Op)
		return a.b.Unary(e.Span, op, operand, a.typeOf(e))

	case *ast.AssignExpr:
		target := a.lowerExpr(e.Target)
		value := a.lowerExpr(e.Value)
		op := e.Op.Literal
		if op == "" {
			op = opFromTokenKind(e.Op.Kind)
		}
		return a.b.Assign(e.Span, op, target, value)

	case *ast.CallExpr:
		callee := a.lowerExpr(e.Callee)
		var args []hir.Expr
		for _, arg := range e.Args {
			args = append(args, a.lowerExpr(arg))
		}
		return a.b.Call(e.Span, callee, args, a.typeOf(e))

	case *ast.IndexExpr:
		expr := a.lowerExpr(e.Expr)
		index := a.lowerExpr(e.Index)
		return a.b.Index(e.Span, expr, index, a.typeOf(e))

	case *ast.FieldExpr:
		expr := a.lowerExpr(e.Expr)
		return a.b.Field(e.Span, expr, e.Name, a.typeOf(e))

	case *ast.QDotExpr:
		expr := a.lowerExpr(e.Expr)
		return a.b.QDot(e.Span, expr, e.Name, a.typeOf(e))

	case *ast.QuestionExpr:
		expr := a.lowerExpr(e.Expr)
		return a.b.Question(e.Span, expr, a.typeOf(e))

	case *ast.BlockExpr:
		return a.lowerBlock(e)

	case *ast.IfExpr:
		cond := a.lowerExpr(e.Cond)
		then := a.lowerBlock(e.Then)
		var els hir.Expr
		if e.Else != nil {
			els = a.lowerExpr(e.Else)
		}
		return a.b.If(e.Span, cond, then, els, a.valueTypeOf(e))

	case *ast.MatchExpr:
		subject := a.lowerExpr(e.Subject)
		subjectTy := a.typeOf(e.Subject)
		var arms []hir.MatchArm
		for _, arm := range e.Arms {
			body := a.lowerExpr(arm.Body)
			var guard hir.Expr
			if arm.Guard != nil {
				guard = a.lowerExpr(arm.Guard)
			}
			desc := patternDesc(arm.Pattern)
			pat := a.lowerPattern(arm.Pattern, subjectTy)
			arms = append(arms, hir.MatchArm{
				Pattern:     pat,
				PatternDesc: desc,
				Guard:       guard,
				Body:        body,
			})
		}
		return a.b.Match(e.Span, subject, arms, a.valueTypeOf(e))

	case *ast.ForExpr:
		iter := a.lowerExpr(e.Iterable)
		body := a.lowerBlock(e.Body)
		return a.b.For(e.Span, e.Binding, iter, body)

	case *ast.WhileExpr:
		cond := a.lowerExpr(e.Cond)
		body := a.lowerBlock(e.Body)
		return a.b.While(e.Span, cond, body)

	case *ast.LoopExpr:
		body := a.lowerBlock(e.Body)
		return a.b.Loop(e.Span, body, a.valueTypeOf(e))

	case *ast.ReturnExpr:
		var val hir.Expr
		if e.Value != nil {
			val = a.lowerExpr(e.Value)
		}
		return a.b.Return(e.Span, val)

	case *ast.BreakExpr:
		var val hir.Expr
		if e.Value != nil {
			val = a.lowerExpr(e.Value)
		}
		return a.b.Break(e.Span, val)

	case *ast.ContinueExpr:
		return a.b.Continue(e.Span)

	case *ast.TupleExpr:
		var elems []hir.Expr
		for _, elem := range e.Elems {
			elems = append(elems, a.lowerExpr(elem))
		}
		return a.b.Tuple(e.Span, elems, a.typeOf(e))

	case *ast.StructLitExpr:
		var fields []hir.FieldInitHIR
		for _, f := range e.Fields {
			fields = append(fields, hir.FieldInitHIR{
				Name:  f.Name,
				Value: a.lowerExpr(f.Value),
			})
		}
		return a.b.StructLit(e.Span, e.Name, fields, a.typeOf(e))

	case *ast.SpawnExpr:
		expr := a.lowerExpr(e.Expr)
		return a.b.Spawn(e.Span, expr)

	case *ast.ClosureExpr:
		var params []hir.Param
		for _, p := range e.Params {
			pty := a.tt.Unknown
			if a.checker != nil {
				pty = a.checker.Types.LookupPrimitive(typeExprString(p.Type))
				if pty == typetable.InvalidTypeId {
					pty = a.tt.Unknown
				}
			}
			params = append(params, hir.Param{
				Span: p.Span,
				Name: p.Name,
				Type: pty,
			})
		}
		body := a.lowerBlock(e.Body)
		closureTy := a.typeOf(e)
		retTy := a.tt.Unknown
		if a.checker != nil && closureTy != a.tt.Unknown {
			ce := a.checker.Types.Get(closureTy)
			if ce.Kind == typetable.KindFunc {
				retTy = ce.ReturnType
			}
		}
		return a.b.Closure(e.Span, params, retTy, body, closureTy)

	default:
		return nil
	}
}

// unaryOpFromToken maps a unary operator token to its string representation.
func unaryOpFromToken(tok lex.Token) string {
	if tok.Literal != "" {
		return tok.Literal
	}
	return opFromTokenKind(tok.Kind)
}

// opFromTokenKind maps a token kind to its operator string.
func opFromTokenKind(kind lex.TokenKind) string {
	switch kind {
	case lex.Plus:
		return "+"
	case lex.Minus:
		return "-"
	case lex.Star:
		return "*"
	case lex.Slash:
		return "/"
	case lex.Percent:
		return "%"
	case lex.Bang:
		return "!"
	case lex.Amp:
		return "&"
	case lex.Pipe:
		return "|"
	case lex.Caret:
		return "^"
	case lex.Tilde:
		return "~"
	case lex.Shl:
		return "<<"
	case lex.Shr:
		return ">>"
	case lex.AmpAmp:
		return "&&"
	case lex.PipePipe:
		return "||"
	case lex.EqEq:
		return "=="
	case lex.BangEq:
		return "!="
	case lex.Lt:
		return "<"
	case lex.Gt:
		return ">"
	case lex.LtEq:
		return "<="
	case lex.GtEq:
		return ">="
	case lex.Eq:
		return "="
	case lex.PlusEq:
		return "+="
	case lex.MinusEq:
		return "-="
	case lex.StarEq:
		return "*="
	case lex.SlashEq:
		return "/="
	case lex.PercentEq:
		return "%="
	case lex.KwRef:
		return "ref"
	case lex.KwMutref:
		return "mutref"
	default:
		return fmt.Sprintf("op(%d)", kind)
	}
}

// lowerPattern translates an AST pattern into an HIR pattern, using the
// subject's resolved type to populate type annotations.
func (a *ast2hir) lowerPattern(p ast.Pattern, subjectTy typetable.TypeId) hir.Pattern {
	if p == nil {
		return &hir.WildcardPattern{}
	}
	switch p := p.(type) {
	case *ast.WildcardPat:
		return &hir.WildcardPattern{}
	case *ast.LitPat:
		return &hir.LiteralPattern{Value: p.Value, Type: subjectTy}
	case *ast.BindPat:
		return &hir.BindPattern{Name: p.Name, Type: subjectTy}
	case *ast.ConstructorPat:
		var args []hir.Pattern
		for _, arg := range p.Args {
			args = append(args, a.lowerPattern(arg, a.tt.Unknown))
		}
		return &hir.ConstructorPattern{Name: p.Name, Args: args}
	default:
		return &hir.WildcardPattern{}
	}
}

// patternDesc produces a textual description of an AST pattern for HIR MatchArm.
func patternDesc(p ast.Pattern) string {
	if p == nil {
		return "_"
	}
	switch p := p.(type) {
	case *ast.WildcardPat:
		return "_"
	case *ast.BindPat:
		return p.Name
	case *ast.LitPat:
		return p.Value
	case *ast.ConstructorPat:
		return p.Name
	case *ast.StructPat:
		return p.Name
	case *ast.TuplePat:
		return "(..)"
	default:
		return "_"
	}
}
