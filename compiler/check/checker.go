// Package check owns semantic analysis and type checking.
//
// The checker operates in two passes (per language guide contract):
//  1. Signature pass: register all function types (top-level, impl methods, externs).
//  2. Body pass: check all function bodies and expressions.
//
// This guarantees that no function declaration retains Unknown type after checking.
package check

import (
	"github.com/Tembocs/fuse4/compiler/ast"
	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/resolve"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

// Checker performs type checking over a resolved module graph.
type Checker struct {
	Types  *typetable.TypeTable
	Graph  *resolve.ModuleGraph
	Errors []diagnostics.Diagnostic

	// funcTypes maps function names (module-qualified) to their function TypeId.
	funcTypes map[string]typetable.TypeId

	// structFields maps struct TypeId to ordered field info.
	structFields map[typetable.TypeId][]fieldInfo

	// traitMethods maps trait name to method signatures.
	traitMethods map[string][]methodSig

	// traitImpls maps "TraitName:TargetType" to true if an impl exists.
	traitImpls map[string]bool

	// traitSupers maps trait name to supertrait names.
	traitSupers map[string][]string

	// current context during body checking
	currentModule *resolve.Module
	currentReturn typetable.TypeId
	localScope    *resolve.Scope
}

type fieldInfo struct {
	Name string
	Type typetable.TypeId
}

type methodSig struct {
	Name       string
	ParamTypes []typetable.TypeId
	ReturnType typetable.TypeId
}

// NewChecker creates a checker for the given module graph and type table.
func NewChecker(types *typetable.TypeTable, graph *resolve.ModuleGraph) *Checker {
	return &Checker{
		Types:        types,
		Graph:        graph,
		funcTypes:    make(map[string]typetable.TypeId),
		structFields: make(map[typetable.TypeId][]fieldInfo),
		traitMethods: make(map[string][]methodSig),
		traitImpls:   make(map[string]bool),
		traitSupers:  make(map[string][]string),
	}
}

// Check runs all checking phases. Modules are checked uniformly (user and stdlib alike).
func (c *Checker) Check() {
	// Pass 1: register all signatures
	for _, key := range c.Graph.Order {
		mod := c.Graph.Modules[key]
		c.registerSignatures(mod)
	}

	// Register primitive methods after signatures but before body checking.
	c.registerPrimitiveMethods()

	// Pass 2: check all bodies
	for _, key := range c.Graph.Order {
		mod := c.Graph.Modules[key]
		c.checkBodies(mod)
	}
}

// --- Pass 1: signature registration ---

func (c *Checker) registerSignatures(mod *resolve.Module) {
	c.currentModule = mod
	for _, item := range mod.File.Items {
		switch n := item.(type) {
		case *ast.FnDecl:
			c.registerFn(mod, n)
		case *ast.ExternFnDecl:
			c.registerExternFn(mod, n)
		case *ast.StructDecl:
			c.registerStruct(mod, n)
		case *ast.EnumDecl:
			c.registerEnum(mod, n)
		case *ast.TraitDecl:
			c.registerTrait(mod, n)
		case *ast.ImplDecl:
			c.registerImpl(mod, n)
		}
	}
}

func (c *Checker) registerFn(mod *resolve.Module, fn *ast.FnDecl) {
	params := c.resolveParamTypes(fn.Params)
	ret := c.resolveTypeExprOr(fn.ReturnType, c.Types.Unit)
	fty := c.Types.InternFunc(params, ret)
	c.funcTypes[mod.Path.String()+"."+fn.Name] = fty
}

func (c *Checker) registerExternFn(mod *resolve.Module, fn *ast.ExternFnDecl) {
	params := c.resolveParamTypes(fn.Params)
	ret := c.resolveTypeExprOr(fn.ReturnType, c.Types.Unit)
	fty := c.Types.InternFunc(params, ret)
	c.funcTypes[mod.Path.String()+"."+fn.Name] = fty
}

func (c *Checker) registerStruct(mod *resolve.Module, s *ast.StructDecl) {
	sty := c.Types.InternStruct(mod.Path.String(), s.Name, nil)
	var fields []fieldInfo
	for _, f := range s.Fields {
		fty := c.resolveTypeExpr(f.Type)
		fields = append(fields, fieldInfo{Name: f.Name, Type: fty})
	}
	c.structFields[sty] = fields
}

func (c *Checker) registerEnum(_ *resolve.Module, e *ast.EnumDecl) {
	// Enum variants are already hoisted by the resolver.
	// Type registration is handled through the type table.
}

func (c *Checker) registerTrait(_ *resolve.Module, t *ast.TraitDecl) {
	var methods []methodSig
	for _, item := range t.Items {
		if fn, ok := item.(*ast.FnDecl); ok {
			params := c.resolveParamTypes(fn.Params)
			ret := c.resolveTypeExprOr(fn.ReturnType, c.Types.Unit)
			methods = append(methods, methodSig{
				Name:       fn.Name,
				ParamTypes: params,
				ReturnType: ret,
			})
		}
	}
	c.traitMethods[t.Name] = methods

	// Record supertraits.
	var supers []string
	for _, sup := range t.Supertraits {
		if pt, ok := sup.(*ast.PathType); ok && len(pt.Segments) == 1 {
			supers = append(supers, pt.Segments[0])
		}
	}
	c.traitSupers[t.Name] = supers
}

func (c *Checker) registerImpl(mod *resolve.Module, impl *ast.ImplDecl) {
	for _, item := range impl.Items {
		if fn, ok := item.(*ast.FnDecl); ok {
			targetName := typeExprName(impl.Target)
			c.registerFn(mod, fn)
			// Also register as method on the target type.
			params := c.resolveParamTypes(fn.Params)
			ret := c.resolveTypeExprOr(fn.ReturnType, c.Types.Unit)
			key := targetName + "." + fn.Name
			c.funcTypes[key] = c.Types.InternFunc(params, ret)
		}
	}
	// Record trait impl if present.
	if impl.Trait != nil {
		traitName := typeExprName(impl.Trait)
		targetName := typeExprName(impl.Target)
		if traitName != "" && targetName != "" {
			c.traitImpls[traitName+":"+targetName] = true
		}
	}
}

// --- Pass 2: body checking ---

func (c *Checker) checkBodies(mod *resolve.Module) {
	c.currentModule = mod
	for _, item := range mod.File.Items {
		switch n := item.(type) {
		case *ast.FnDecl:
			if n.Body != nil {
				c.checkFnBody(mod, n)
			}
		case *ast.ImplDecl:
			for _, implItem := range n.Items {
				if fn, ok := implItem.(*ast.FnDecl); ok && fn.Body != nil {
					c.checkFnBody(mod, fn)
				}
			}
		case *ast.ConstDecl:
			if n.Value != nil {
				c.checkExpr(n.Value)
			}
		}
	}
}

func (c *Checker) checkFnBody(mod *resolve.Module, fn *ast.FnDecl) {
	c.currentReturn = c.resolveTypeExprOr(fn.ReturnType, c.Types.Unit)
	c.localScope = resolve.NewScope(mod.Symbols)

	// Add parameters to local scope.
	for _, p := range fn.Params {
		pty := c.resolveTypeExprOr(p.Type, c.Types.Unknown)
		c.localScope.Define(&resolve.Symbol{
			Name: p.Name,
			Kind: resolve.SymParam,
			Span: p.Span,
		})
		_ = pty // param type is used during expression checking
	}

	c.checkBlock(fn.Body)
}

// --- error reporting ---

func (c *Checker) errorf(span diagnostics.Span, format string, args ...any) {
	c.Errors = append(c.Errors, diagnostics.Errorf(span, format, args...))
}

// --- helpers ---

func typeExprName(te ast.TypeExpr) string {
	if te == nil {
		return ""
	}
	if pt, ok := te.(*ast.PathType); ok && len(pt.Segments) > 0 {
		return pt.Segments[len(pt.Segments)-1]
	}
	return ""
}
