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
	// Check local variable types first (let/var bindings and params).
	if c.localTypes != nil {
		if ty, ok := c.localTypes[e.Name]; ok {
			// Auto-deref: when a local has ref/mutref type, the expression
			// evaluates to the inner type (borrows are transparent in expressions).
			te := c.Types.Get(ty)
			if te.Kind == typetable.KindRef || te.Kind == typetable.KindMutRef {
				return te.Elem
			}
			return ty
		}
	}
	// Check local scope for other symbols.
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
	// Lookup as constant.
	if ty, ok := c.constValues[e.Name]; ok {
		return ty
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
	// Check if this is an enum variant constructor: Some(42), Ok(val), etc.
	if ident, ok := e.Callee.(*ast.IdentExpr); ok {
		if enumTy := c.checkVariantConstructor(ident.Name, e); enumTy != typetable.InvalidTypeId {
			return enumTy
		}
		// Enforce unsafe: extern function calls require unsafe context.
		if c.externFns[ident.Name] && !c.inUnsafe {
			c.errorf(e.Span, "call to extern function '%s' requires unsafe block", ident.Name)
		}
	}

	calleeType := c.checkExpr(e.Callee)
	for _, arg := range e.Args {
		c.checkExpr(arg)
	}

	ce := c.Types.Get(calleeType)
	if ce.Kind == typetable.KindFunc {
		// Unify argument types with parameter types: upgrade Unknown type args
		// to concrete ones when the parameter type is more specific.
		for i, arg := range e.Args {
			if i < len(ce.Fields) {
				c.unifyExprType(arg, ce.Fields[i])
			}
		}
		return ce.ReturnType
	}

	// If callee is a field access, try method lookup.
	if fe, ok := e.Callee.(*ast.FieldExpr); ok {
		recvType := c.checkExpr(fe.Expr)
		return c.lookupMethod(recvType, fe.Name, e.Span)
	}

	return c.Types.Unknown
}

// checkVariantConstructor checks if a call is to an enum variant constructor.
// Returns the concrete enum TypeId if so, or InvalidTypeId if not a variant call.
func (c *Checker) checkVariantConstructor(name string, e *ast.CallExpr) typetable.TypeId {
	// Look up the symbol to see if it's an enum variant.
	var sym *resolve.Symbol
	if c.localScope != nil {
		sym = c.localScope.Lookup(name)
	}
	if sym == nil && c.currentModule != nil {
		sym = c.currentModule.Symbols.Lookup(name)
	}
	if sym == nil || sym.Kind != resolve.SymEnumVariant {
		return typetable.InvalidTypeId
	}

	// Found a variant. Check arg types to infer generic type args.
	argTypes := make([]typetable.TypeId, len(e.Args))
	for i, arg := range e.Args {
		argTypes[i] = c.checkExpr(arg)
	}

	// Look up the parent enum's variant info.
	enumName := sym.Parent
	variants, ok := c.EnumVariants[enumName]
	if !ok {
		return typetable.InvalidTypeId
	}

	// Find the enum declaration to get generic params.
	var enumDecl *ast.EnumDecl
	if c.currentModule != nil {
		for _, item := range c.currentModule.File.Items {
			if ed, ok := item.(*ast.EnumDecl); ok && ed.Name == enumName {
				enumDecl = ed
				break
			}
		}
	}
	// Also search other modules.
	if enumDecl == nil {
		for _, key := range c.Graph.Order {
			mod := c.Graph.Modules[key]
			for _, item := range mod.File.Items {
				if ed, ok := item.(*ast.EnumDecl); ok && ed.Name == enumName {
					enumDecl = ed
					break
				}
			}
		}
	}

	modStr := ""
	if c.currentModule != nil {
		modStr = c.currentModule.Path.String()
	}

	// Determine concrete type args from the constructor arguments.
	var typeArgs []typetable.TypeId
	if enumDecl != nil && len(enumDecl.GenericParams) > 0 {
		// Generic enum: infer type args from constructor arg types.
		// For each generic param, find the first variant payload that uses it
		// and match against the concrete arg type.
		typeArgs = make([]typetable.TypeId, len(enumDecl.GenericParams))
		for gi, gp := range enumDecl.GenericParams {
			// Find this variant and match type params to arg types.
			for _, v := range variants {
				if v.Name == name {
					for pi, pt := range v.PayloadTypes {
						te := c.Types.Get(pt)
						if te.Kind == typetable.KindGenericParam && te.Name == gp.Name && pi < len(argTypes) {
							typeArgs[gi] = argTypes[pi]
						}
					}
				}
			}
			// Try to infer from the current function's return type context.
			if typeArgs[gi] == typetable.InvalidTypeId || typeArgs[gi] == c.Types.Unknown {
				retEntry := c.Types.Get(c.currentReturn)
				if retEntry.Name == enumName && gi < len(retEntry.TypeArgs) {
					typeArgs[gi] = retEntry.TypeArgs[gi]
				}
			}
			// Fallback to Unknown if not inferred.
			if typeArgs[gi] == typetable.InvalidTypeId {
				typeArgs[gi] = c.Types.Unknown
			}
		}
	}

	// Create the concrete enum type.
	enumTy := c.Types.InternEnum(modStr, enumName, typeArgs)

	// Register concrete variant info with resolved payload types.
	if _, exists := c.EnumTypeVariants[enumTy]; !exists {
		concreteVars := make([]VariantDef, len(variants))
		// Track the maximum payload types across all variants for the struct fields.
		var maxPayloads []typetable.TypeId
		for i, v := range variants {
			payloads := make([]typetable.TypeId, len(v.PayloadTypes))
			for j, pt := range v.PayloadTypes {
				resolved := pt
				// Substitute generic params with concrete type args.
				te := c.Types.Get(pt)
				if te.Kind == typetable.KindGenericParam && enumDecl != nil {
					for gi, gp := range enumDecl.GenericParams {
						if te.Name == gp.Name && gi < len(typeArgs) {
							resolved = typeArgs[gi]
						}
					}
				}
				payloads[j] = resolved
			}
			concreteVars[i] = VariantDef{Name: v.Name, Tag: v.Tag, PayloadTypes: payloads}
			// Use the widest variant's payloads for the struct fields.
			if len(payloads) > len(maxPayloads) {
				maxPayloads = payloads
			}
		}
		c.EnumTypeVariants[enumTy] = concreteVars
		// Set enum fields on the type table so codegen can emit the struct.
		c.Types.SetEnumFields(enumTy, maxPayloads)
	}

	return enumTy
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
	prevUnsafe := c.inUnsafe
	c.localScope = resolve.NewScope(c.localScope)
	if e.Unsafe {
		c.inUnsafe = true
	}
	defer func() { c.localScope = prevScope; c.inUnsafe = prevUnsafe }()

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
		if c.localTypes != nil {
			c.localTypes[s.Name] = ty
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
		if c.localTypes != nil {
			c.localTypes[s.Name] = ty
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
	// If the body contains a break with a value, the loop's type is the break value type.
	// Otherwise it diverges.
	if breakTy := c.findBreakType(e.Body); breakTy != typetable.InvalidTypeId {
		return breakTy
	}
	return c.Types.Never // infinite loop diverges unless broken
}

// findBreakType scans a block for break expressions with values and returns their type.
func (c *Checker) findBreakType(block *ast.BlockExpr) typetable.TypeId {
	if block == nil {
		return typetable.InvalidTypeId
	}
	for _, stmt := range block.Stmts {
		if es, ok := stmt.(*ast.ExprStmt); ok {
			if br, ok := es.Expr.(*ast.BreakExpr); ok && br.Value != nil {
				if ty, ok := c.ExprTypes[br.Value]; ok {
					return ty
				}
			}
		}
	}
	return typetable.InvalidTypeId
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

// unifyExprType upgrades an expression's type when the expected type is more
// specific (i.e., has concrete type args where the expression has Unknown).
// This handles cases like passing Ok(42) (Result[I32, Unknown]) to a param
// expecting Result[I32, Bool].
func (c *Checker) unifyExprType(expr ast.Expr, expected typetable.TypeId) {
	if c.ExprTypes == nil {
		return
	}
	actual, ok := c.ExprTypes[expr]
	if !ok || actual == expected {
		return
	}
	ae := c.Types.Get(actual)
	ee := c.Types.Get(expected)
	if ae.Kind != ee.Kind || ae.Name != ee.Name || ae.Module != ee.Module {
		return
	}
	if len(ae.TypeArgs) != len(ee.TypeArgs) || len(ae.TypeArgs) == 0 {
		return
	}
	// Check if actual has Unknown type args that expected fills in.
	needsUpgrade := false
	for i := range ae.TypeArgs {
		if ae.TypeArgs[i] == c.Types.Unknown && ee.TypeArgs[i] != c.Types.Unknown {
			needsUpgrade = true
			break
		}
	}
	if needsUpgrade {
		c.ExprTypes[expr] = expected
	}
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
