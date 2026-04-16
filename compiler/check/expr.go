package check

import (
	"github.com/Tembocs/fuse4/compiler/ast"
	"github.com/Tembocs/fuse4/compiler/diagnostics"
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
	case *ast.ArrayLitExpr:
		return c.checkArrayLit(e)
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
		// The stdlib String type lives in module `core.string`. Using the
		// bare `core` module (its pre-auto-load placeholder) would produce
		// a distinct TypeId and break assignment to stdlib-typed fields.
		return c.Types.InternStruct("core.string", "String", nil)
	case lex.KwTrue, lex.KwFalse:
		return c.Types.Bool
	case lex.KwNone:
		// Option[T] is ambiguous without context. Prefer the current
		// expected type (set by checkReturn) so `return None;` in a
		// function returning `Option[Entry]` types as Option[Entry]
		// rather than Unknown. Without this the MIR EnumInit is typed
		// as Unknown and codegen falls back to `(int){...}`.
		if c.currentExpected != typetable.InvalidTypeId {
			ce := c.Types.Get(c.currentExpected)
			if ce.Kind == typetable.KindEnum && ce.Name == "Option" {
				return c.currentExpected
			}
		}
		if c.currentReturn != typetable.InvalidTypeId {
			re := c.Types.Get(c.currentReturn)
			if re.Kind == typetable.KindEnum && re.Name == "Option" {
				return c.currentReturn
			}
		}
		return c.Types.Unknown
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

// symbolType resolves a symbol to a TypeId. Function and extern-fn symbols
// yield their registered function type; struct and enum symbols yield the
// nominal type TypeId itself so that patterns like `String.new()` work as
// static method calls (fe.Expr is an identifier that names a type, not a
// value). Without the nominal TypeId the receiver is Unknown and
// lookupMethod cannot find the method.

func (c *Checker) symbolType(sym *resolve.Symbol) typetable.TypeId {
	// Look up function type from the registered signatures.
	modStr := sym.Module.String()
	key := modStr + "." + sym.Name
	if fty, ok := c.funcTypes[key]; ok {
		return fty
	}
	// Struct and enum symbols: return the nominal TypeId so static method
	// calls (`TypeName.method()`) can route through lookupMethod. Without
	// this, the receiver evaluates to Unknown and the lowerer emits
	// `Fuse_Unknown__method`.
	switch sym.Kind {
	case resolve.SymStruct:
		return c.Types.InternStruct(modStr, sym.Name, nil)
	case resolve.SymEnum:
		return c.Types.InternEnum(modStr, sym.Name, nil)
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
		// String + String → String concatenation.
		lte := c.Types.Get(lt)
		if e.Op.Kind == lex.Plus && lte.Kind == typetable.KindStruct && lte.Name == "String" {
			return lt
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

	// Note: Trait bounds are checked post-registration in checkSpecializedBounds.

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
		ret := c.lookupMethod(recvType, fe.Name, e.Span)
		return c.promoteToExpected(ret)
	}

	return c.Types.Unknown
}

// promoteToExpected upgrades a call's return type when the surrounding
// context expects a specialization of the same nominal type. Static
// methods on generic types (`List.new()`, `Map.new()`) return whichever
// specialization's signature registered first under `TypeName.method`;
// that is not necessarily the spec the caller wants. Using the
// caller's expected type aligns return-TypeArgs so isAssignableTo
// passes and emit sees the right spec name downstream.
func (c *Checker) promoteToExpected(ty typetable.TypeId) typetable.TypeId {
	if c.currentExpected == typetable.InvalidTypeId {
		return ty
	}
	te := c.Types.Get(ty)
	ee := c.Types.Get(c.currentExpected)
	if te.Kind != ee.Kind || te.Name != ee.Name || te.Module != ee.Module {
		return ty
	}
	if te.Kind != typetable.KindStruct && te.Kind != typetable.KindEnum {
		return ty
	}
	// Only upgrade when the expected side is strictly more specific.
	if len(ee.TypeArgs) == 0 {
		return ty
	}
	return c.currentExpected
}

// checkVariantConstructor checks if a call is to an enum variant constructor.
// Returns the concrete enum TypeId if so, or InvalidTypeId if not a variant call.
func (c *Checker) checkVariantConstructor(name string, e *ast.CallExpr) typetable.TypeId {
	// Look up the symbol to see if it's an enum variant. Fall through to
	// a graph-wide search because variant names (Some, None, Ok, Err) are
	// not imported into every module that uses them — monomorphized
	// specializations placed in a defining module may still reference
	// variants declared elsewhere (core.option etc.).
	var sym *resolve.Symbol
	if c.localScope != nil {
		sym = c.localScope.Lookup(name)
	}
	if sym == nil && c.currentModule != nil {
		sym = c.currentModule.Symbols.Lookup(name)
	}
	if (sym == nil || sym.Kind != resolve.SymEnumVariant) && c.Graph != nil {
		for _, key := range c.Graph.Order {
			mod := c.Graph.Modules[key]
			if mod == nil || mod.Symbols == nil {
				continue
			}
			s := mod.Symbols.LookupLocal(name)
			if s != nil && s.Kind == resolve.SymEnumVariant {
				sym = s
				break
			}
		}
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

	// Canonicalize the enum's nominal module — `sym.Module` is the module
	// where the variant (and therefore the enum) was declared, not the
	// currently-checked module. Using currentModule here would intern
	// `Result` under `core.bool` when `Ok(())` appears inside a Bool
	// Printable impl, causing the assignability check to fail even though
	// both sides textually match `Result` (L021-style module mismatch).
	modStr := sym.Module.String()

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
	case typetable.KindPtr:
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
	// Generic specialization: structFields is populated on the base
	// (nil TypeArgs) template, but the field access happens on a
	// specialization like `List[Entry]`. Substitute the specialization's
	// TypeArgs into the base field type so the access types as the
	// concrete element type rather than Unknown (task 3 cascade).
	if te.Kind == typetable.KindStruct && len(te.TypeArgs) > 0 {
		base := c.Types.BaseOf(recvType)
		if base != typetable.InvalidTypeId {
			_, subs := c.Types.SubstituteFields(base, te.TypeArgs)
			if baseFields, ok := c.structFields[base]; ok {
				for i, f := range baseFields {
					if f.Name == e.Name && i < len(subs) {
						return subs[i]
					}
				}
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
	exprTy := c.checkExpr(e.Expr)
	te := c.Types.Get(exprTy)

	// ?. on Option[T] or Result[T, E]: access field on the inner T value.
	// Returns Option[FieldType] (simplified: just the field type for bootstrap).
	if (te.Kind == typetable.KindEnum || te.Kind == typetable.KindStruct) &&
		(te.Name == "Option" || te.Name == "Result") && len(te.TypeArgs) > 0 {
		innerType := te.TypeArgs[0] // T from Option[T] or Result[T, E]
		// Look up the field on the inner type.
		fieldTy := c.FieldType(innerType, e.Name)
		if fieldTy != typetable.InvalidTypeId {
			return fieldTy
		}
		return innerType
	}

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
	iterTy := c.checkExpr(e.Iterable)
	prevScope := c.localScope
	c.localScope = resolve.NewScope(c.localScope)

	// Infer binding type from iterable: if array type, elem is the element type.
	bindTy := c.Types.Unknown
	te := c.Types.Get(iterTy)
	if te.Kind == typetable.KindArray || te.Kind == typetable.KindSlice {
		bindTy = te.Elem
	}

	c.localScope.Define(&resolve.Symbol{
		Name: e.Binding,
		Kind: resolve.SymLocal,
		Span: e.Span,
	})
	if c.localTypes != nil {
		c.localTypes[e.Binding] = bindTy
	}
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
		prev := c.currentExpected
		c.currentExpected = c.currentReturn
		valTy := c.checkExpr(e.Value)
		c.currentExpected = prev
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
	// Canonicalize under the struct's defining module. A struct named
	// List in user module `foo` must still intern under `core.list` when
	// the user's `List` is the auto-loaded stdlib type (L021 fix).
	if modStr, _, ok := c.resolveTypeName(e.Name); ok {
		// `List { ... }` inside a spec like `fn new() -> List[Entry]`
		// should type as `List[Entry]`, not the base `List`. Without
		// this the MIR local ends up typed as the generic template
		// and emit.fnHasGenericParam filters the whole function out.
		if exp := c.expectedType(); exp != typetable.InvalidTypeId {
			ee := c.Types.Get(exp)
			if (ee.Kind == typetable.KindStruct || ee.Kind == typetable.KindEnum) &&
				ee.Name == e.Name && ee.Module == modStr && len(ee.TypeArgs) > 0 {
				return exp
			}
		}
		return c.Types.InternStruct(modStr, e.Name, nil)
	}
	c.errorf(e.Span, "unknown struct '%s'", e.Name)
	return c.Types.Unknown
}

// expectedType returns the current context's expected type or
// InvalidTypeId when none is known. Updated by checkReturn /
// checkAssign / container initialization before descending into
// sub-expressions. Narrow: only StructLitExpr consumes it so far.
func (c *Checker) expectedType() typetable.TypeId {
	return c.currentExpected
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

// --- array literal ---

func (c *Checker) checkArrayLit(e *ast.ArrayLitExpr) typetable.TypeId {
	if len(e.Elems) == 0 {
		return c.Types.InternArray(c.Types.Unknown, 0)
	}
	var elemType typetable.TypeId
	for _, el := range e.Elems {
		ty := c.checkExpr(el)
		if elemType == 0 || elemType == c.Types.Unknown {
			elemType = ty
		}
	}
	return c.Types.InternArray(elemType, len(e.Elems))
}

// --- trait bounds ---

// checkGenericBounds validates that explicit type args satisfy the generic
// function's declared trait bounds (e.g., [T: Display] rejects types without Display).
func (c *Checker) checkGenericBounds(fnName string, typeArgExpr ast.Expr, span diagnostics.Span) {
	// Find the original generic function declaration.
	var genFn *ast.FnDecl
	for _, key := range c.Graph.Order {
		mod := c.Graph.Modules[key]
		for _, item := range mod.File.Items {
			if fn, ok := item.(*ast.FnDecl); ok && fn.Name == fnName && len(fn.GenericParams) > 0 {
				genFn = fn
				break
			}
		}
		if genFn != nil {
			break
		}
	}
	if genFn == nil {
		return
	}

	// Extract type arg names from the index expression.
	var typeArgNames []string
	switch idx := typeArgExpr.(type) {
	case *ast.IdentExpr:
		typeArgNames = []string{idx.Name}
	case *ast.TupleExpr:
		for _, el := range idx.Elems {
			if id, ok := el.(*ast.IdentExpr); ok {
				typeArgNames = append(typeArgNames, id.Name)
			}
		}
	}

	// Check each generic param's bounds against the concrete type arg.
	for i, gp := range genFn.GenericParams {
		if i >= len(typeArgNames) {
			break
		}
		concreteType := typeArgNames[i]
		for _, bound := range gp.Bounds {
			boundName := ""
			if pt, ok := bound.(*ast.PathType); ok && len(pt.Segments) > 0 {
				boundName = pt.Segments[0]
			}
			if boundName == "" {
				continue
			}
			// Check if the concrete type implements the bound trait.
			implKey := boundName + ":" + concreteType
			if !c.traitImpls[implKey] {
				c.errorf(span, "type '%s' does not implement trait '%s' (required by bound on '%s')",
					concreteType, boundName, gp.Name)
			}
		}
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
