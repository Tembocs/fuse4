// specialize.go implements AST-level monomorphization.
//
// This runs between resolve and check in the driver pipeline.
// It finds generic function declarations, collects their concrete
// instantiation sites, generates specialized copies with type parameters
// replaced by concrete types, and rewrites call sites to reference the
// specialized names. After this pass, the checker sees only concrete functions.
package monomorph

import (
	"strings"

	"github.com/Tembocs/fuse4/compiler/ast"
	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/lex"
	"github.com/Tembocs/fuse4/compiler/resolve"
)

// instantiation records a concrete usage of a generic function.
type instantiation struct {
	genName  string   // original function name, e.g. "identity"
	typeArgs []string // concrete type names, e.g. ["I32"]
	specName string   // specialized name, e.g. "identity__I32"
}

// SpecializeModules performs AST-level monomorphization on a module graph.
// It mutates the AST in place: adding specialized function declarations
// and rewriting generic call sites to reference them.
func SpecializeModules(graph *resolve.ModuleGraph) {
	// --- Phase 1: index generic functions ---
	genericFns := map[string]*ast.FnDecl{}
	genericMod := map[string]string{} // fn name → module key
	for _, key := range graph.Order {
		mod := graph.Modules[key]
		for _, item := range mod.File.Items {
			if fn, ok := item.(*ast.FnDecl); ok && len(fn.GenericParams) > 0 {
				genericFns[fn.Name] = fn
				genericMod[fn.Name] = key
			}
		}
	}
	if len(genericFns) == 0 {
		return
	}

	// --- Phase 2: collect instantiation sites ---
	var insts []instantiation
	seen := map[string]bool{}
	for _, key := range graph.Order {
		mod := graph.Modules[key]
		for _, item := range mod.File.Items {
			fn, ok := item.(*ast.FnDecl)
			if !ok || fn.Body == nil {
				continue
			}
			collectCalls(fn.Body, genericFns, &insts, seen)
		}
	}
	if len(insts) == 0 {
		return
	}

	// --- Phase 3: generate specialized functions ---
	for _, inst := range insts {
		genFn := genericFns[inst.genName]
		modKey := genericMod[inst.genName]
		mod := graph.Modules[modKey]

		specFn := specializeFunction(genFn, inst.specName, inst.typeArgs)
		mod.File.Items = append(mod.File.Items, specFn)

		// Register in the module's symbol table so the checker can resolve it.
		mod.Symbols.Define(&resolve.Symbol{
			Name:   inst.specName,
			Kind:   resolve.SymFunc,
			Module: mod.Path,
		})
	}

	// --- Phase 4: rewrite call sites ---
	for _, key := range graph.Order {
		mod := graph.Modules[key]
		for _, item := range mod.File.Items {
			fn, ok := item.(*ast.FnDecl)
			if !ok || fn.Body == nil {
				continue
			}
			rewriteBlockCalls(fn.Body, genericFns)
		}
	}
}

// IsGenericFn returns true if a FnDecl has generic type parameters.
func IsGenericFn(fn *ast.FnDecl) bool {
	return len(fn.GenericParams) > 0
}

// --- instantiation collection ---

// collectCalls walks a block and finds generic call patterns:
// CallExpr(IndexExpr(IdentExpr(genName), IdentExpr(typeArg)), args)
func collectCalls(block *ast.BlockExpr, genericFns map[string]*ast.FnDecl, insts *[]instantiation, seen map[string]bool) {
	if block == nil {
		return
	}
	for _, s := range block.Stmts {
		collectStmtCalls(s, genericFns, insts, seen)
	}
	if block.Tail != nil {
		collectExprCalls(block.Tail, genericFns, insts, seen)
	}
}

func collectStmtCalls(s ast.Stmt, genericFns map[string]*ast.FnDecl, insts *[]instantiation, seen map[string]bool) {
	switch s := s.(type) {
	case *ast.LetStmt:
		if s.Value != nil {
			collectExprCalls(s.Value, genericFns, insts, seen)
		}
	case *ast.VarStmt:
		if s.Value != nil {
			collectExprCalls(s.Value, genericFns, insts, seen)
		}
	case *ast.ExprStmt:
		collectExprCalls(s.Expr, genericFns, insts, seen)
	}
}

func collectExprCalls(e ast.Expr, genericFns map[string]*ast.FnDecl, insts *[]instantiation, seen map[string]bool) {
	if e == nil {
		return
	}
	switch e := e.(type) {
	case *ast.CallExpr:
		// Check for explicit generic call pattern: call(index(ident, type_arg), args)
		if idx, ok := e.Callee.(*ast.IndexExpr); ok {
			if base, ok := idx.Expr.(*ast.IdentExpr); ok {
				if _, isGen := genericFns[base.Name]; isGen {
					typeArgs := extractTypeArgs(idx.Index)
					if len(typeArgs) > 0 {
						specName := makeSpecName(base.Name, typeArgs)
						if !seen[specName] {
							seen[specName] = true
							*insts = append(*insts, instantiation{
								genName:  base.Name,
								typeArgs: typeArgs,
								specName: specName,
							})
						}
					}
				}
			}
		}
		// Check for inferred generic call: call(ident(genName), args)
		// Infer type args from argument types (literals and idents).
		if ident, ok := e.Callee.(*ast.IdentExpr); ok {
			if genFn, isGen := genericFns[ident.Name]; isGen {
				typeArgs := inferTypeArgs(genFn, e.Args)
				if len(typeArgs) > 0 {
					specName := makeSpecName(ident.Name, typeArgs)
					if !seen[specName] {
						seen[specName] = true
						*insts = append(*insts, instantiation{
							genName:  ident.Name,
							typeArgs: typeArgs,
							specName: specName,
						})
					}
				}
			}
		}
		collectExprCalls(e.Callee, genericFns, insts, seen)
		for _, a := range e.Args {
			collectExprCalls(a, genericFns, insts, seen)
		}
	case *ast.BinaryExpr:
		collectExprCalls(e.Left, genericFns, insts, seen)
		collectExprCalls(e.Right, genericFns, insts, seen)
	case *ast.UnaryExpr:
		collectExprCalls(e.Operand, genericFns, insts, seen)
	case *ast.AssignExpr:
		collectExprCalls(e.Target, genericFns, insts, seen)
		collectExprCalls(e.Value, genericFns, insts, seen)
	case *ast.IndexExpr:
		collectExprCalls(e.Expr, genericFns, insts, seen)
		collectExprCalls(e.Index, genericFns, insts, seen)
	case *ast.FieldExpr:
		collectExprCalls(e.Expr, genericFns, insts, seen)
	case *ast.ReturnExpr:
		if e.Value != nil {
			collectExprCalls(e.Value, genericFns, insts, seen)
		}
	case *ast.BreakExpr:
		if e.Value != nil {
			collectExprCalls(e.Value, genericFns, insts, seen)
		}
	case *ast.BlockExpr:
		collectCalls(e, genericFns, insts, seen)
	case *ast.IfExpr:
		collectExprCalls(e.Cond, genericFns, insts, seen)
		collectCalls(e.Then, genericFns, insts, seen)
		if e.Else != nil {
			collectExprCalls(e.Else, genericFns, insts, seen)
		}
	case *ast.WhileExpr:
		collectExprCalls(e.Cond, genericFns, insts, seen)
		collectCalls(e.Body, genericFns, insts, seen)
	case *ast.LoopExpr:
		collectCalls(e.Body, genericFns, insts, seen)
	case *ast.ForExpr:
		collectExprCalls(e.Iterable, genericFns, insts, seen)
		collectCalls(e.Body, genericFns, insts, seen)
	case *ast.MatchExpr:
		collectExprCalls(e.Subject, genericFns, insts, seen)
		for _, arm := range e.Arms {
			collectExprCalls(arm.Body, genericFns, insts, seen)
			if arm.Guard != nil {
				collectExprCalls(arm.Guard, genericFns, insts, seen)
			}
		}
	case *ast.TupleExpr:
		for _, el := range e.Elems {
			collectExprCalls(el, genericFns, insts, seen)
		}
	case *ast.StructLitExpr:
		for _, f := range e.Fields {
			collectExprCalls(f.Value, genericFns, insts, seen)
		}
	case *ast.QuestionExpr:
		collectExprCalls(e.Expr, genericFns, insts, seen)
	case *ast.QDotExpr:
		collectExprCalls(e.Expr, genericFns, insts, seen)
	case *ast.SpawnExpr:
		collectExprCalls(e.Expr, genericFns, insts, seen)
	case *ast.ClosureExpr:
		collectCalls(e.Body, genericFns, insts, seen)
	}
}

// extractTypeArgs extracts type argument names from an index expression.
// For identity[I32], the index is IdentExpr("I32") → ["I32"]
// For try_get[I32, Bool], the index is TupleExpr([IdentExpr("I32"), IdentExpr("Bool")]) → ["I32", "Bool"]
func extractTypeArgs(index ast.Expr) []string {
	switch idx := index.(type) {
	case *ast.IdentExpr:
		return []string{idx.Name}
	case *ast.TupleExpr:
		var names []string
		for _, elem := range idx.Elems {
			if ident, ok := elem.(*ast.IdentExpr); ok {
				names = append(names, ident.Name)
			} else {
				return nil // non-ident type arg — not supported
			}
		}
		return names
	default:
		return nil
	}
}

// inferTypeArgs infers generic type arguments from call arguments.
// For identity[T](x: T) called as identity(42), infers T=I32.
func inferTypeArgs(genFn *ast.FnDecl, args []ast.Expr) []string {
	if len(genFn.GenericParams) == 0 || len(args) == 0 {
		return nil
	}

	// Build a map from generic param name to inferred type name.
	inferred := make(map[string]string)

	for i, param := range genFn.Params {
		if i >= len(args) {
			break
		}
		// Get the param's type name.
		paramTypeName := typeExprString(param.Type)
		if paramTypeName == "" {
			continue
		}
		// Check if this param type is one of the generic params.
		for _, gp := range genFn.GenericParams {
			if paramTypeName == gp.Name {
				// Infer the type from the argument expression.
				argType := inferExprType(args[i])
				if argType != "" {
					inferred[gp.Name] = argType
				}
			}
		}
	}

	// Build the type args array in order.
	if len(inferred) != len(genFn.GenericParams) {
		return nil // couldn't infer all type params
	}
	typeArgs := make([]string, len(genFn.GenericParams))
	for i, gp := range genFn.GenericParams {
		typeArgs[i] = inferred[gp.Name]
	}
	return typeArgs
}

// inferExprType guesses the type of an expression from its shape.
func inferExprType(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.LiteralExpr:
		return inferLiteralType(e)
	case *ast.IdentExpr:
		// Can't infer type of an ident without scope info.
		return ""
	default:
		return ""
	}
}

// inferLiteralType returns the default type name for a literal.
func inferLiteralType(e *ast.LiteralExpr) string {
	switch e.Token.Kind {
	case lex.IntLit:
		lit := e.Token.Literal
		suffixes := []struct{ s, t string }{
			{"i8", "I8"}, {"i16", "I16"}, {"i32", "I32"}, {"i64", "I64"},
			{"u8", "U8"}, {"u16", "U16"}, {"u32", "U32"}, {"u64", "U64"},
			{"isize", "ISize"}, {"usize", "USize"},
		}
		for _, sf := range suffixes {
			if len(lit) > len(sf.s) && lit[len(lit)-len(sf.s):] == sf.s {
				return sf.t
			}
		}
		return "I32"
	case lex.FloatLit:
		return "F64"
	case lex.StringLit, lex.RawStringLit:
		return "String"
	case lex.KwTrue, lex.KwFalse:
		return "Bool"
	}
	return ""
}

// typeExprString extracts the simple name from a type expression.
func typeExprString(te ast.TypeExpr) string {
	if te == nil {
		return ""
	}
	if pt, ok := te.(*ast.PathType); ok && len(pt.Segments) > 0 {
		return pt.Segments[len(pt.Segments)-1]
	}
	return ""
}

// makeSpecName generates a deterministic specialized function name.
func makeSpecName(genName string, typeArgs []string) string {
	return genName + "__" + strings.Join(typeArgs, "_")
}

// --- function specialization (AST cloning with type substitution) ---

// specializeFunction deep-clones a generic FnDecl, replacing type parameter
// references with concrete types.
func specializeFunction(genFn *ast.FnDecl, specName string, typeArgs []string) *ast.FnDecl {
	// Build substitution map: type param name → concrete type name.
	subst := map[string]string{}
	for i, gp := range genFn.GenericParams {
		if i < len(typeArgs) {
			subst[gp.Name] = typeArgs[i]
		}
	}

	// Clone params with substituted types.
	params := make([]ast.Param, len(genFn.Params))
	for i, p := range genFn.Params {
		params[i] = ast.Param{
			Span:      p.Span,
			Ownership: p.Ownership,
			Name:      p.Name,
			Type:      substTypeExpr(p.Type, subst),
		}
	}

	return &ast.FnDecl{
		Span:          genFn.Span,
		Public:        genFn.Public,
		Name:          specName,
		GenericParams: nil, // no longer generic
		Params:        params,
		ReturnType:    substTypeExpr(genFn.ReturnType, subst),
		Body:          cloneBlock(genFn.Body, subst),
	}
}

// substTypeExpr clones a type expression, substituting type parameter names.
func substTypeExpr(te ast.TypeExpr, subst map[string]string) ast.TypeExpr {
	if te == nil {
		return nil
	}
	switch t := te.(type) {
	case *ast.PathType:
		// If the single segment matches a type param, replace it.
		if len(t.Segments) == 1 {
			if concrete, ok := subst[t.Segments[0]]; ok {
				return &ast.PathType{
					Span:     t.Span,
					Segments: []string{concrete},
					TypeArgs: substTypeExprs(t.TypeArgs, subst),
				}
			}
		}
		return &ast.PathType{
			Span:     t.Span,
			Segments: append([]string{}, t.Segments...),
			TypeArgs: substTypeExprs(t.TypeArgs, subst),
		}
	case *ast.TupleType:
		return &ast.TupleType{
			Span:  t.Span,
			Elems: substTypeExprs(t.Elems, subst),
		}
	case *ast.ArrayType:
		return &ast.ArrayType{
			Span: t.Span,
			Elem: substTypeExpr(t.Elem, subst),
			Size: t.Size, // size is a constant, not type-dependent
		}
	case *ast.SliceType:
		return &ast.SliceType{
			Span: t.Span,
			Elem: substTypeExpr(t.Elem, subst),
		}
	case *ast.PtrType:
		return &ast.PtrType{
			Span: t.Span,
			Elem: substTypeExpr(t.Elem, subst),
		}
	default:
		return te
	}
}

func substTypeExprs(tes []ast.TypeExpr, subst map[string]string) []ast.TypeExpr {
	if len(tes) == 0 {
		return nil
	}
	out := make([]ast.TypeExpr, len(tes))
	for i, te := range tes {
		out[i] = substTypeExpr(te, subst)
	}
	return out
}

// --- AST deep cloning ---

func cloneBlock(block *ast.BlockExpr, subst map[string]string) *ast.BlockExpr {
	if block == nil {
		return nil
	}
	stmts := make([]ast.Stmt, len(block.Stmts))
	for i, s := range block.Stmts {
		stmts[i] = cloneStmt(s, subst)
	}
	return &ast.BlockExpr{
		Span:  block.Span,
		Stmts: stmts,
		Tail:  cloneExpr(block.Tail, subst),
	}
}

func cloneStmt(s ast.Stmt, subst map[string]string) ast.Stmt {
	if s == nil {
		return nil
	}
	switch s := s.(type) {
	case *ast.LetStmt:
		return &ast.LetStmt{
			Span:  s.Span,
			Name:  s.Name,
			Type:  substTypeExpr(s.Type, subst),
			Value: cloneExpr(s.Value, subst),
		}
	case *ast.VarStmt:
		return &ast.VarStmt{
			Span:  s.Span,
			Name:  s.Name,
			Type:  substTypeExpr(s.Type, subst),
			Value: cloneExpr(s.Value, subst),
		}
	case *ast.ExprStmt:
		return &ast.ExprStmt{
			Span: s.Span,
			Expr: cloneExpr(s.Expr, subst),
		}
	case *ast.ItemStmt:
		return s // item stmts are not cloned (nested fn decls etc.)
	default:
		return s
	}
}

func cloneExpr(e ast.Expr, subst map[string]string) ast.Expr {
	if e == nil {
		return nil
	}
	switch e := e.(type) {
	case *ast.LiteralExpr:
		return &ast.LiteralExpr{Span: e.Span, Token: e.Token}
	case *ast.IdentExpr:
		return &ast.IdentExpr{Span: e.Span, Name: e.Name}
	case *ast.BinaryExpr:
		return &ast.BinaryExpr{
			Span:  e.Span,
			Left:  cloneExpr(e.Left, subst),
			Op:    e.Op,
			Right: cloneExpr(e.Right, subst),
		}
	case *ast.UnaryExpr:
		return &ast.UnaryExpr{
			Span:    e.Span,
			Op:      e.Op,
			Operand: cloneExpr(e.Operand, subst),
		}
	case *ast.AssignExpr:
		return &ast.AssignExpr{
			Span:   e.Span,
			Target: cloneExpr(e.Target, subst),
			Op:     e.Op,
			Value:  cloneExpr(e.Value, subst),
		}
	case *ast.CallExpr:
		args := make([]ast.Expr, len(e.Args))
		for i, a := range e.Args {
			args[i] = cloneExpr(a, subst)
		}
		return &ast.CallExpr{
			Span:   e.Span,
			Callee: cloneExpr(e.Callee, subst),
			Args:   args,
		}
	case *ast.IndexExpr:
		return &ast.IndexExpr{
			Span:  e.Span,
			Expr:  cloneExpr(e.Expr, subst),
			Index: cloneExpr(e.Index, subst),
		}
	case *ast.FieldExpr:
		return &ast.FieldExpr{
			Span: e.Span,
			Expr: cloneExpr(e.Expr, subst),
			Name: e.Name,
		}
	case *ast.QDotExpr:
		return &ast.QDotExpr{
			Span: e.Span,
			Expr: cloneExpr(e.Expr, subst),
			Name: e.Name,
		}
	case *ast.QuestionExpr:
		return &ast.QuestionExpr{
			Span: e.Span,
			Expr: cloneExpr(e.Expr, subst),
		}
	case *ast.BlockExpr:
		return cloneBlock(e, subst)
	case *ast.IfExpr:
		return &ast.IfExpr{
			Span: e.Span,
			Cond: cloneExpr(e.Cond, subst),
			Then: cloneBlock(e.Then, subst),
			Else: cloneExpr(e.Else, subst),
		}
	case *ast.MatchExpr:
		arms := make([]ast.MatchArm, len(e.Arms))
		for i, arm := range e.Arms {
			arms[i] = ast.MatchArm{
				Span:    arm.Span,
				Pattern: arm.Pattern, // patterns don't contain type substitutions
				Guard:   cloneExpr(arm.Guard, subst),
				Body:    cloneExpr(arm.Body, subst),
			}
		}
		return &ast.MatchExpr{
			Span:    e.Span,
			Subject: cloneExpr(e.Subject, subst),
			Arms:    arms,
		}
	case *ast.ForExpr:
		return &ast.ForExpr{
			Span:     e.Span,
			Binding:  e.Binding,
			Iterable: cloneExpr(e.Iterable, subst),
			Body:     cloneBlock(e.Body, subst),
		}
	case *ast.WhileExpr:
		return &ast.WhileExpr{
			Span: e.Span,
			Cond: cloneExpr(e.Cond, subst),
			Body: cloneBlock(e.Body, subst),
		}
	case *ast.LoopExpr:
		return &ast.LoopExpr{
			Span: e.Span,
			Body: cloneBlock(e.Body, subst),
		}
	case *ast.ReturnExpr:
		return &ast.ReturnExpr{
			Span:  e.Span,
			Value: cloneExpr(e.Value, subst),
		}
	case *ast.BreakExpr:
		return &ast.BreakExpr{
			Span:  e.Span,
			Value: cloneExpr(e.Value, subst),
		}
	case *ast.ContinueExpr:
		return &ast.ContinueExpr{Span: e.Span}
	case *ast.TupleExpr:
		elems := make([]ast.Expr, len(e.Elems))
		for i, el := range e.Elems {
			elems[i] = cloneExpr(el, subst)
		}
		return &ast.TupleExpr{Span: e.Span, Elems: elems}
	case *ast.StructLitExpr:
		fields := make([]ast.FieldInit, len(e.Fields))
		for i, f := range e.Fields {
			fields[i] = ast.FieldInit{
				Span:  f.Span,
				Name:  f.Name,
				Value: cloneExpr(f.Value, subst),
			}
		}
		return &ast.StructLitExpr{Span: e.Span, Name: e.Name, Fields: fields}
	case *ast.SpawnExpr:
		return &ast.SpawnExpr{Span: e.Span, Expr: cloneExpr(e.Expr, subst)}
	case *ast.ClosureExpr:
		params := make([]ast.Param, len(e.Params))
		for i, p := range e.Params {
			params[i] = ast.Param{
				Span:      p.Span,
				Ownership: p.Ownership,
				Name:      p.Name,
				Type:      substTypeExpr(p.Type, subst),
			}
		}
		return &ast.ClosureExpr{
			Span:       e.Span,
			Params:     params,
			ReturnType: substTypeExpr(e.ReturnType, subst),
			Body:       cloneBlock(e.Body, subst),
		}
	default:
		return e
	}
}

// --- call site rewriting ---

// rewriteBlockCalls walks a block and replaces generic call patterns
// with direct calls to specialized function names.
func rewriteBlockCalls(block *ast.BlockExpr, genericFns map[string]*ast.FnDecl) {
	if block == nil {
		return
	}
	for _, s := range block.Stmts {
		rewriteStmtCalls(s, genericFns)
	}
	if block.Tail != nil {
		rewriteExprCalls(block.Tail, genericFns)
	}
}

func rewriteStmtCalls(s ast.Stmt, genericFns map[string]*ast.FnDecl) {
	switch s := s.(type) {
	case *ast.LetStmt:
		if s.Value != nil {
			rewriteExprCalls(s.Value, genericFns)
		}
	case *ast.VarStmt:
		if s.Value != nil {
			rewriteExprCalls(s.Value, genericFns)
		}
	case *ast.ExprStmt:
		rewriteExprCalls(s.Expr, genericFns)
	}
}

func rewriteExprCalls(e ast.Expr, genericFns map[string]*ast.FnDecl) {
	if e == nil {
		return
	}
	switch e := e.(type) {
	case *ast.CallExpr:
		// Rewrite generic call: call(index(ident(genName), ident(typeArg)), args)
		// → call(ident(specName), args)
		if idx, ok := e.Callee.(*ast.IndexExpr); ok {
			if base, ok := idx.Expr.(*ast.IdentExpr); ok {
				if _, isGen := genericFns[base.Name]; isGen {
					typeArgs := extractTypeArgs(idx.Index)
					if len(typeArgs) > 0 {
						specName := makeSpecName(base.Name, typeArgs)
						e.Callee = &ast.IdentExpr{
							Span: diagnostics.Span{},
							Name: specName,
						}
					}
				}
			}
		}
		// Also rewrite inferred generic calls: call(ident(genName), args) → call(ident(specName), args)
		if ident, ok := e.Callee.(*ast.IdentExpr); ok {
			if genFn, isGen := genericFns[ident.Name]; isGen {
				typeArgs := inferTypeArgs(genFn, e.Args)
				if len(typeArgs) > 0 {
					specName := makeSpecName(ident.Name, typeArgs)
					e.Callee = &ast.IdentExpr{
						Span: diagnostics.Span{},
						Name: specName,
					}
				}
			}
		}
		rewriteExprCalls(e.Callee, genericFns)
		for _, a := range e.Args {
			rewriteExprCalls(a, genericFns)
		}
	case *ast.BinaryExpr:
		rewriteExprCalls(e.Left, genericFns)
		rewriteExprCalls(e.Right, genericFns)
	case *ast.UnaryExpr:
		rewriteExprCalls(e.Operand, genericFns)
	case *ast.AssignExpr:
		rewriteExprCalls(e.Target, genericFns)
		rewriteExprCalls(e.Value, genericFns)
	case *ast.IndexExpr:
		rewriteExprCalls(e.Expr, genericFns)
		rewriteExprCalls(e.Index, genericFns)
	case *ast.FieldExpr:
		rewriteExprCalls(e.Expr, genericFns)
	case *ast.ReturnExpr:
		if e.Value != nil {
			rewriteExprCalls(e.Value, genericFns)
		}
	case *ast.BreakExpr:
		if e.Value != nil {
			rewriteExprCalls(e.Value, genericFns)
		}
	case *ast.BlockExpr:
		rewriteBlockCalls(e, genericFns)
	case *ast.IfExpr:
		rewriteExprCalls(e.Cond, genericFns)
		rewriteBlockCalls(e.Then, genericFns)
		if e.Else != nil {
			rewriteExprCalls(e.Else, genericFns)
		}
	case *ast.WhileExpr:
		rewriteExprCalls(e.Cond, genericFns)
		rewriteBlockCalls(e.Body, genericFns)
	case *ast.LoopExpr:
		rewriteBlockCalls(e.Body, genericFns)
	case *ast.ForExpr:
		rewriteExprCalls(e.Iterable, genericFns)
		rewriteBlockCalls(e.Body, genericFns)
	case *ast.MatchExpr:
		rewriteExprCalls(e.Subject, genericFns)
		for _, arm := range e.Arms {
			rewriteExprCalls(arm.Body, genericFns)
			if arm.Guard != nil {
				rewriteExprCalls(arm.Guard, genericFns)
			}
		}
	case *ast.TupleExpr:
		for _, el := range e.Elems {
			rewriteExprCalls(el, genericFns)
		}
	case *ast.StructLitExpr:
		for _, f := range e.Fields {
			rewriteExprCalls(f.Value, genericFns)
		}
	case *ast.QuestionExpr:
		rewriteExprCalls(e.Expr, genericFns)
	case *ast.QDotExpr:
		rewriteExprCalls(e.Expr, genericFns)
	case *ast.SpawnExpr:
		rewriteExprCalls(e.Expr, genericFns)
	case *ast.ClosureExpr:
		rewriteBlockCalls(e.Body, genericFns)
	}
}
