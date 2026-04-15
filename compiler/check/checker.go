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

	// ExprTypes maps AST expression pointers to their resolved TypeId.
	// Populated during body checking for use by the AST-to-HIR bridge.
	ExprTypes map[ast.Expr]typetable.TypeId

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

	// EnumVariants maps enum name to variant definitions (ordered by tag).
	// Used by the bridge and codegen for variant construction and struct emission.
	EnumVariants map[string][]VariantDef

	// EnumTypeVariants maps concrete enum TypeId to variant defs (with resolved types).
	// Populated when concrete enum types are created (e.g. Option[I32]).
	EnumTypeVariants map[typetable.TypeId][]VariantDef

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

// VariantDef describes an enum variant for codegen and the bridge.
type VariantDef struct {
	Name         string
	Tag          int
	PayloadTypes []typetable.TypeId // empty for unit variants
}

// NewChecker creates a checker for the given module graph and type table.
func NewChecker(types *typetable.TypeTable, graph *resolve.ModuleGraph) *Checker {
	return &Checker{
		Types:            types,
		Graph:            graph,
		ExprTypes:        make(map[ast.Expr]typetable.TypeId),
		funcTypes:        make(map[string]typetable.TypeId),
		structFields:     make(map[typetable.TypeId][]fieldInfo),
		traitMethods:     make(map[string][]methodSig),
		traitImpls:       make(map[string]bool),
		traitSupers:      make(map[string][]string),
		EnumVariants:     make(map[string][]VariantDef),
		EnumTypeVariants: make(map[typetable.TypeId][]VariantDef),
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

func (c *Checker) registerEnum(mod *resolve.Module, e *ast.EnumDecl) {
	modStr := mod.Path.String()

	// Build a map of generic param names → GenericParam TypeIds.
	gpMap := map[string]typetable.TypeId{}
	for _, gp := range e.GenericParams {
		gpMap[gp.Name] = c.Types.InternGenericParam(modStr, gp.Name)
	}

	// Register variant definitions with tag numbers and payload types.
	var variants []VariantDef
	for i, v := range e.Variants {
		var payloads []typetable.TypeId
		for _, pt := range v.Types {
			// Check if the type is a generic param name.
			name := typeExprName(pt)
			if gpTy, ok := gpMap[name]; ok {
				payloads = append(payloads, gpTy)
			} else {
				payloads = append(payloads, c.resolveTypeExpr(pt))
			}
		}
		variants = append(variants, VariantDef{
			Name:         v.Name,
			Tag:          i,
			PayloadTypes: payloads,
		})
	}
	c.EnumVariants[e.Name] = variants

	// For non-generic enums, register the concrete type variants and fields.
	if len(e.GenericParams) == 0 {
		ety := c.Types.InternEnum(modStr, e.Name, nil)
		c.EnumTypeVariants[ety] = variants
		var maxPayloads []typetable.TypeId
		for _, v := range variants {
			if len(v.PayloadTypes) > len(maxPayloads) {
				maxPayloads = v.PayloadTypes
			}
		}
		c.Types.SetEnumFields(ety, maxPayloads)
	}
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

// --- public accessors for the AST-to-HIR bridge ---

// FuncType returns the function TypeId for a qualified name, or InvalidTypeId.
func (c *Checker) FuncType(name string) typetable.TypeId {
	if fty, ok := c.funcTypes[name]; ok {
		return fty
	}
	return typetable.InvalidTypeId
}

// FieldType returns the type of a named field on a struct type.
func (c *Checker) FieldType(structTy typetable.TypeId, fieldName string) typetable.TypeId {
	for _, f := range c.structFields[structTy] {
		if f.Name == fieldName {
			return f.Type
		}
	}
	return typetable.InvalidTypeId
}

// VariantTag returns the tag number for an enum variant, or -1 if not found.
// Uses the module graph's deterministic Order to search.
func (c *Checker) VariantTag(variantName string) int {
	for _, key := range c.Graph.Order {
		mod := c.Graph.Modules[key]
		sym := mod.Symbols.Lookup(variantName)
		if sym != nil && sym.Kind == resolve.SymEnumVariant {
			if variants, ok := c.EnumVariants[sym.Parent]; ok {
				for _, v := range variants {
					if v.Name == variantName {
						return v.Tag
					}
				}
			}
		}
	}
	return -1
}

// IsVariantConstructor returns true if the given name is an enum variant.
func (c *Checker) IsVariantConstructor(name string) bool {
	for _, key := range c.Graph.Order {
		mod := c.Graph.Modules[key]
		sym := mod.Symbols.Lookup(name)
		if sym != nil && sym.Kind == resolve.SymEnumVariant {
			return true
		}
	}
	return false
}

// GetEnumTypeVariants returns variant defs for a concrete enum TypeId.
func (c *Checker) GetEnumTypeVariants(ty typetable.TypeId) []VariantDef {
	return c.EnumTypeVariants[ty]
}

// FuncReturnType returns the return type of a named function.
func (c *Checker) FuncReturnType(qualifiedName string) typetable.TypeId {
	if fty, ok := c.funcTypes[qualifiedName]; ok {
		fe := c.Types.Get(fty)
		if fe.Kind == typetable.KindFunc {
			return fe.ReturnType
		}
	}
	return c.Types.Unknown
}

// FuncParamTypes returns the parameter types of a named function.
func (c *Checker) FuncParamTypes(qualifiedName string) []typetable.TypeId {
	if fty, ok := c.funcTypes[qualifiedName]; ok {
		fe := c.Types.Get(fty)
		if fe.Kind == typetable.KindFunc {
			return fe.Fields
		}
	}
	return nil
}
