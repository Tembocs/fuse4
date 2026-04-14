package check

import (
	"github.com/Tembocs/fuse4/compiler/ast"
	"github.com/Tembocs/fuse4/compiler/lex"
	"github.com/Tembocs/fuse4/compiler/resolve"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

// checkExpr type-checks an expression, records the resolved type, and returns it.
func (c *Checker) checkExpr(expr ast.Expr) typetable.TypeId {
	if expr == nil {
		return c.Types.Unit
	}
	ty := c.resolveExpr(expr)
	if c.ExprTypes != nil {
		c.ExprTypes[expr] = ty
	}
	return ty
}

// resolveExpr dispatches to the appropriate type-checking method for the expression.
func (c *Checker) resolveExpr(expr ast.Expr) typetable.TypeId {
	switch e := expr.(type) {
	case *ast.LiteralExpr:
		return c.checkLiteral(e)
	case *ast.IdentExpr:
		return c.checkIdent(e)
	case *ast.BinaryExpr:
		return c.checkBinary(e)
	case *ast.UnaryExpr:
		return c.checkUnary(e)
	case *ast.AssignExpr:
		return c.checkAssign(e)
	case *ast.CallExpr:
		return c.checkCall(e)
	case *ast.IndexExpr:
		return c.checkIndex(e)
	case *ast.FieldExpr:
		return c.checkField(e)
	case *ast.QDotExpr:
		return c.checkQDot(e)
	case *ast.QuestionExpr:
		return c.checkQuestion(e)
	case *ast.BlockExpr:
		return c.checkBlock(e)
	case *ast.IfExpr:
		return c.checkIf(e)
	case *ast.MatchExpr:
		return c.checkMatch(e)
	case *ast.ForExpr:
		return c.checkFor(e)
	case *ast.WhileExpr:
		return c.checkWhile(e)
	case *ast.LoopExpr:
		return c.checkLoop(e)
	case *ast.ReturnExpr:
		return c.checkReturn(e)
	case *ast.BreakExpr:
		return c.checkBreak(e)
	case *ast.ContinueExpr:
		return c.Types.Never
	case *ast.SpawnExpr:
		c.checkExpr(e.Expr)
		return c.Types.Unit
	case *ast.TupleExpr:
		return c.checkTuple(e)
	case *ast.StructLitExpr:
		return c.checkStructLit(e)
	case *ast.ClosureExpr:
		return c.checkClosure(e)
	default:
		return c.Types.Unknown
	}
}

// --- literals ---

func (c *Checker) checkLiteral(e *ast.LiteralExpr) typetable.TypeId {
	switch e.Token.Kind {
	case lex.IntLit:
		return c.inferIntLiteral(e.Token.Literal)
	case lex.FloatLit:
		return c.inferFloatLiteral(e.Token.Literal)
	case lex.StringLit, lex.RawStringLit:
		return c.Types.InternStruct("core", "String", nil)
	case lex.KwTrue, lex.KwFalse:
		return c.Types.Bool
	case lex.KwNone:
		return c.Types.Unknown // needs context to determine Option[T]
	default:
		return c.Types.Unknown
	}
}

func (c *Checker) inferIntLiteral(lit string) typetable.TypeId {
	// Check for explicit suffix.
	suffixes := map[string]typetable.TypeId{
		"i8": c.Types.I8, "i16": c.Types.I16, "i32": c.Types.I32,
		"i64": c.Types.I64, "i128": c.Types.I128, "isize": c.Types.ISize,
		"u8": c.Types.U8, "u16": c.Types.U16, "u32": c.Types.U32,
		"u64": c.Types.U64, "u128": c.Types.U128, "usize": c.Types.USize,
	}
	for suffix, ty := range suffixes {
		if len(lit) > len(suffix) && lit[len(lit)-len(suffix):] == suffix {
			return ty
		}
	}
	// Default: I32
	return c.Types.I32
}

func (c *Checker) inferFloatLiteral(lit string) typetable.TypeId {
	if len(lit) >= 3 && lit[len(lit)-3:] == "f32" {
		return c.Types.F32
	}
	if len(lit) >= 3 && lit[len(lit)-3:] == "f64" {
		return c.Types.F64
	}
	return c.Types.F64 // default
}

// --- identifiers ---

func (c *Checker) checkIdent(e *ast.IdentExpr) typetable.TypeId {
	// Check local scope first.
	if c.localScope != nil {
		if sym := c.localScope.Lookup(e.Name); sym != nil {
			return c.symbolType(sym)
		}
	}
	// Check module scope.
	if c.currentModule != nil {
		if sym := c.currentModule.Symbols.Lookup(e.Name); sym != nil {
			return c.symbolType(sym)
		}
	}
	// Lookup as primitive constructor (Ok, Err, Some, None).
	if prim := c.Types.LookupPrimitive(e.Name); prim != typetable.InvalidTypeId {
		return prim
	}
	return c.Types.Unknown
}

func (c *Checker) symbolType(sym *resolve.Symbol) typetable.TypeId {
	// Look up function type from the registered signatures.
	modStr := sym.Module.String()
	key := modStr + "." + sym.Name
	if fty, ok := c.funcTypes[key]; ok {
		return fty
	}
	return c.Types.Unknown
}

// --- binary ---

func (c *Checker) checkBinary(e *ast.BinaryExpr) typetable.TypeId {
	lt := c.checkExpr(e.Left)
	rt := c.checkExpr(e.Right)

	switch e.Op.Kind {
	case lex.EqEq, lex.BangEq, lex.Lt, lex.Gt, lex.LtEq, lex.GtEq:
		// Comparison operators always return Bool.
		if c.Types.IsNumeric(lt) && c.Types.IsNumeric(rt) {
			if c.numericWiden(lt, rt) == typetable.InvalidTypeId {
				c.errorf(e.Span, "cannot compare %s and %s",
					c.Types.Get(lt).Name, c.Types.Get(rt).Name)
			}
		}
		return c.Types.Bool

	case lex.AmpAmp, lex.PipePipe:
		// Logical operators require Bool operands and return Bool.
		return c.Types.Bool

	case lex.Plus, lex.Minus, lex.Star, lex.Slash, lex.Percent:
		if c.Types.IsNumeric(lt) && c.Types.IsNumeric(rt) {
			widened := c.numericWiden(lt, rt)
			if widened != typetable.InvalidTypeId {
				return widened
			}
			c.errorf(e.Span, "cannot apply %s between %s and %s",
				e.Op.Literal, c.Types.Get(lt).Name, c.Types.Get(rt).Name)
		}
		return lt

	case lex.Amp, lex.Pipe, lex.Caret, lex.Shl, lex.Shr:
		// Bitwise operators require matching signedness and width.
		if lt == rt && c.Types.IsNumeric(lt) {
			return lt
		}
		return lt

	default:
		return lt
	}
}

// --- unary ---

func (c *Checker) checkUnary(e *ast.UnaryExpr) typetable.TypeId {
	inner := c.checkExpr(e.Operand)

	switch e.Op.Kind {
	case lex.Bang:
		return c.Types.Bool
	case lex.Minus:
		return inner
	case lex.Tilde:
		return inner
	case lex.KwRef:
		return c.Types.InternRef(inner)
	case lex.KwMutref:
		return c.Types.InternMutRef(inner)
	case lex.KwOwned, lex.KwMove:
		return inner
	default:
		return inner
	}
}

// --- assign ---

func (c *Checker) checkAssign(e *ast.AssignExpr) typetable.TypeId {
	lt := c.checkExpr(e.Target)
	rt := c.checkExpr(e.Value)
	if !c.isAssignableTo(rt, lt) {
		c.errorf(e.Span, "cannot assign %s to %s",
			c.Types.Get(rt).Name, c.Types.Get(lt).Name)
	}
	return c.Types.Unit
}

// --- call ---

func (c *Checker) checkCall(e *ast.CallExpr) typetable.TypeId {
	calleeType := c.checkExpr(e.Callee)
	for _, arg := range e.Args {
		c.checkExpr(arg)
	}

	ce := c.Types.Get(calleeType)
	if ce.Kind == typetable.KindFunc {
		return ce.ReturnType
	}

	// If callee is a field access, try method lookup.
	if fe, ok := e.Callee.(*ast.FieldExpr); ok {
		recvType := c.checkExpr(fe.Expr)
		return c.lookupMethod(recvType, fe.Name, e.Span)
	}

	return c.Types.Unknown
}

// --- index ---

func (c *Checker) checkIndex(e *ast.IndexExpr) typetable.TypeId {
	exprType := c.checkExpr(e.Expr)
	c.checkExpr(e.Index)

	te := c.Types.Get(exprType)
	switch te.Kind {
	case typetable.KindArray, typetable.KindSlice:
		return te.Elem
	}
	return c.Types.Unknown
}

// --- field access ---

func (c *Checker) checkField(e *ast.FieldExpr) typetable.TypeId {
	recvType := c.checkExpr(e.Expr)
	te := c.Types.Get(recvType)

	// Tuple field access by index.
	if te.Kind == typetable.KindTuple {
		idx := parseFieldIndex(e.Name)
		if idx >= 0 && idx < len(te.Fields) {
			return te.Fields[idx]
		}
		c.errorf(e.Span, "tuple index %s out of range", e.Name)
		return c.Types.Unknown
	}

	// Struct field access.
	if fields, ok := c.structFields[recvType]; ok {
		for _, f := range fields {
			if f.Name == e.Name {
				return f.Type
			}
		}
	}

	// Could be a method — resolved during call checking.
	return c.Types.Unknown
}

func parseFieldIndex(name string) int {
	n := 0
	for _, ch := range name {
		if ch < '0' || ch > '9' {
			return -1
		}
		n = n*10 + int(ch-'0')
	}
	return n
}

// --- optional chaining ---

func (c *Checker) checkQDot(e *ast.QDotExpr) typetable.TypeId {
	c.checkExpr(e.Expr)
	// ?. returns the field type wrapped in Option — simplified for now.
	return c.Types.Unknown
}

// --- postfix ? ---

func (c *Checker) checkQuestion(e *ast.QuestionExpr) typetable.TypeId {
	inner := c.checkExpr(e.Expr)
	te := c.Types.Get(inner)

	// ? on Result[T, E] → T (propagates E)
	// ? on Option[T] → T (propagates None)
	if te.Kind == typetable.KindEnum || te.Kind == typetable.KindStruct {
		switch te.Name {
		case "Result":
			if len(te.TypeArgs) >= 1 {
				return te.TypeArgs[0] // T from Result[T, E]
			}
		case "Option":
			if len(te.TypeArgs) >= 1 {
				return te.TypeArgs[0] // T from Option[T]
			}
		}
	}

	// If we can't determine the inner type, return Unknown.
	return c.Types.Unknown
}

// --- block ---

func (c *Checker) checkBlock(e *ast.BlockExpr) typetable.TypeId {
	prevScope := c.localScope
	c.localScope = resolve.NewScope(c.localScope)
	defer func() { c.localScope = prevScope }()

	for _, stmt := range e.Stmts {
		c.checkStmt(stmt)
	}

	if e.Tail != nil {
		return c.checkExpr(e.Tail)
	}
	return c.Types.Unit
}

func (c *Checker) checkStmt(stmt ast.Stmt) {
	switch s := stmt.(type) {
	case *ast.LetStmt:
		ty := c.resolveTypeExprOr(s.Type, c.Types.Unknown)
		if s.Value != nil {
			valTy := c.checkExpr(s.Value)
			if ty == c.Types.Unknown {
				ty = valTy
			}
		}
		if c.localScope != nil {
			c.localScope.Define(&resolve.Symbol{
				Name: s.Name,
				Kind: resolve.SymLocal,
				Span: s.Span,
			})
		}
	case *ast.VarStmt:
		ty := c.resolveTypeExprOr(s.Type, c.Types.Unknown)
		if s.Value != nil {
			valTy := c.checkExpr(s.Value)
			if ty == c.Types.Unknown {
				ty = valTy
			}
		}
		if c.localScope != nil {
			c.localScope.Define(&resolve.Symbol{
				Name: s.Name,
				Kind: resolve.SymLocal,
				Span: s.Span,
			})
		}
	case *ast.ExprStmt:
		c.checkExpr(s.Expr)
	case *ast.ItemStmt:
		// nested item declarations — skip for now
	}
}

// --- if ---

func (c *Checker) checkIf(e *ast.IfExpr) typetable.TypeId {
	c.checkExpr(e.Cond)
	thenTy := c.checkBlock(e.Then)
	if e.Else != nil {
		elseTy := c.checkExpr(e.Else)
		if c.isAssignableTo(elseTy, thenTy) {
			return thenTy
		}
		return thenTy
	}
	return c.Types.Unit
}

// --- match ---

func (c *Checker) checkMatch(e *ast.MatchExpr) typetable.TypeId {
	c.checkExpr(e.Subject)
	var armType typetable.TypeId
	for _, arm := range e.Arms {
		if arm.Guard != nil {
			c.checkExpr(arm.Guard)
		}
		ty := c.checkExpr(arm.Body)
		if armType == 0 {
			armType = ty
		}
	}
	if armType == 0 {
		return c.Types.Unit
	}
	return armType
}

// --- loops ---

func (c *Checker) checkFor(e *ast.ForExpr) typetable.TypeId {
	c.checkExpr(e.Iterable)
	prevScope := c.localScope
	c.localScope = resolve.NewScope(c.localScope)
	c.localScope.Define(&resolve.Symbol{
		Name: e.Binding,
		Kind: resolve.SymLocal,
		Span: e.Span,
	})
	c.checkBlock(e.Body)
	c.localScope = prevScope
	return c.Types.Unit
}

func (c *Checker) checkWhile(e *ast.WhileExpr) typetable.TypeId {
	c.checkExpr(e.Cond)
	c.checkBlock(e.Body)
	return c.Types.Unit
}

func (c *Checker) checkLoop(e *ast.LoopExpr) typetable.TypeId {
	c.checkBlock(e.Body)
	return c.Types.Never // infinite loop diverges unless broken
}

// --- return / break ---

func (c *Checker) checkReturn(e *ast.ReturnExpr) typetable.TypeId {
	if e.Value != nil {
		valTy := c.checkExpr(e.Value)
		if !c.isAssignableTo(valTy, c.currentReturn) {
			c.errorf(e.Span, "return type mismatch: got %s, expected %s",
				c.Types.Get(valTy).Name, c.Types.Get(c.currentReturn).Name)
		}
	}
	return c.Types.Never
}

func (c *Checker) checkBreak(e *ast.BreakExpr) typetable.TypeId {
	if e.Value != nil {
		c.checkExpr(e.Value)
	}
	return c.Types.Never
}

// --- tuple ---

func (c *Checker) checkTuple(e *ast.TupleExpr) typetable.TypeId {
	if len(e.Elems) == 0 {
		return c.Types.Unit
	}
	elems := make([]typetable.TypeId, len(e.Elems))
	for i, el := range e.Elems {
		elems[i] = c.checkExpr(el)
	}
	return c.Types.InternTuple(elems)
}

// --- struct literal ---

func (c *Checker) checkStructLit(e *ast.StructLitExpr) typetable.TypeId {
	for _, f := range e.Fields {
		c.checkExpr(f.Value)
	}
	modStr := ""
	if c.currentModule != nil {
		modStr = c.currentModule.Path.String()
	}
	return c.Types.InternStruct(modStr, e.Name, nil)
}

// --- closure ---

func (c *Checker) checkClosure(e *ast.ClosureExpr) typetable.TypeId {
	params := c.resolveParamTypes(e.Params)
	ret := c.resolveTypeExprOr(e.ReturnType, c.Types.Unit)

	prevScope := c.localScope
	prevReturn := c.currentReturn
	c.localScope = resolve.NewScope(c.localScope)
	c.currentReturn = ret

	for _, p := range e.Params {
		c.localScope.Define(&resolve.Symbol{
			Name: p.Name,
			Kind: resolve.SymParam,
			Span: p.Span,
		})
	}

	c.checkBlock(e.Body)

	c.localScope = prevScope
	c.currentReturn = prevReturn

	return c.Types.InternFunc(params, ret)
}
