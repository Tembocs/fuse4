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
	"github.com/Tembocs/fuse4/compiler/lex"
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

	// constValues maps const names to their resolved types.
	constValues map[string]typetable.TypeId

	// typeAliases maps alias names to their underlying TypeId.
	typeAliases map[string]typetable.TypeId

	// ConstLiterals maps const names to their literal string values (for inlining).
	ConstLiterals map[string]string

	// localTypes maps local variable names to their resolved types (within the current function).
	localTypes map[string]typetable.TypeId

	// externFns tracks extern function names for unsafe enforcement.
	externFns map[string]bool

	// current context during body checking
	currentModule *resolve.Module
	currentReturn typetable.TypeId
	localScope    *resolve.Scope
	inUnsafe      bool
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
		constValues:      make(map[string]typetable.TypeId),
		ConstLiterals:    make(map[string]string),
		typeAliases:      make(map[string]typetable.TypeId),
		externFns:        make(map[string]bool),
	}
}

// Check runs all checking phases. Modules are checked uniformly (user and stdlib alike).
func (c *Checker) Check() {
	// Register built-in types.
	c.Types.RegisterStringType()

	// Pass 1: register all signatures
	for _, key := range c.Graph.Order {
		mod := c.Graph.Modules[key]
		c.registerSignatures(mod)
	}

	// Register primitive methods after signatures but before body checking.
	c.registerPrimitiveMethods()

	// Register built-in functions (print, println).
	c.registerBuiltinFunctions()

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
		case *ast.ConstDecl:
			c.registerConst(mod, n)
		case *ast.TypeAliasDecl:
			c.registerTypeAlias(mod, n)
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
	c.externFns[fn.Name] = true
}

func (c *Checker) registerStruct(mod *resolve.Module, s *ast.StructDecl) {
	sty := c.Types.InternStruct(mod.Path.String(), s.Name, nil)
	var fields []fieldInfo
	var fieldNames []string
	var fieldTypes []typetable.TypeId
	for _, f := range s.Fields {
		fty := c.resolveTypeExpr(f.Type)
		// Recursive type detection: a struct cannot contain itself directly.
		if fty == sty {
			c.errorf(f.Span, "recursive type: field '%s' in struct '%s' has the same type (use Ptr[%s] instead)",
				f.Name, s.Name, s.Name)
		}
		fields = append(fields, fieldInfo{Name: f.Name, Type: fty})
		fieldNames = append(fieldNames, f.Name)
		fieldTypes = append(fieldTypes, fty)
	}
	c.structFields[sty] = fields
	c.Types.SetStructFields(sty, fieldNames, fieldTypes)
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

func (c *Checker) registerBuiltinFunctions() {
	strTy := c.Types.InternStruct("core", "String", nil)
	// println(s: String) -> ()
	printlnTy := c.Types.InternFunc([]typetable.TypeId{strTy}, c.Types.Unit)
	c.funcTypes["println"] = printlnTy
	// print(s: String) -> ()
	printTy := c.Types.InternFunc([]typetable.TypeId{strTy}, c.Types.Unit)
	c.funcTypes["print"] = printTy
	// OS module built-ins
	// exit(code: I32) -> Never
	exitTy := c.Types.InternFunc([]typetable.TypeId{c.Types.I32}, c.Types.Never)
	c.funcTypes["exit"] = exitTy
	// argc() -> I32
	argcTy := c.Types.InternFunc(nil, c.Types.I32)
	c.funcTypes["argc"] = argcTy
}

func (c *Checker) registerTypeAlias(_ *resolve.Module, ta *ast.TypeAliasDecl) {
	underlying := c.resolveTypeExpr(ta.Type)
	c.typeAliases[ta.Name] = underlying
}

func (c *Checker) registerConst(mod *resolve.Module, cn *ast.ConstDecl) {
	ty := c.resolveTypeExprOr(cn.Type, c.Types.Unknown)
	if ty == c.Types.Unknown && cn.Value != nil {
		ty = c.checkExpr(cn.Value)
	}
	c.constValues[mod.Path.String()+"."+cn.Name] = ty
	c.constValues[cn.Name] = ty
	// Store literal value for inlining at use sites.
	if cn.Value != nil {
		if lit, ok := cn.Value.(*ast.LiteralExpr); ok {
			c.ConstLiterals[mod.Path.String()+"."+cn.Name] = lit.Token.Literal
			c.ConstLiterals[cn.Name] = lit.Token.Literal
		}
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
	targetName := typeExprName(impl.Target)
	targetType := c.resolveTypeExpr(impl.Target)

	for _, item := range impl.Items {
		if fn, ok := item.(*ast.FnDecl); ok {
			c.registerFn(mod, fn)
			// Resolve params with self → target type.
			params := c.resolveImplParamTypes(fn.Params, targetType)
			ret := c.resolveTypeExprOr(fn.ReturnType, c.Types.Unit)
			key := targetName + "." + fn.Name
			c.funcTypes[key] = c.Types.InternFunc(params, ret)
			// Also register module-qualified for direct lookup.
			qualKey := mod.Path.String() + "." + fn.Name
			c.funcTypes[qualKey] = c.Types.InternFunc(params, ret)
		}
	}
	// Record trait impl if present.
	if impl.Trait != nil {
		traitName := typeExprName(impl.Trait)
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
	c.localTypes = make(map[string]typetable.TypeId)

	// Add parameters to local scope with their types.
	for _, p := range fn.Params {
		pty := c.resolveTypeExprOr(p.Type, c.Types.Unknown)
		// For self params in impl methods, type was already set correctly
		// via resolveImplParamTypes; look it up from the function signature.
		if p.Name == "self" && p.Type == nil {
			qualName := mod.Path.String() + "." + fn.Name
			paramTypes := c.FuncParamTypes(qualName)
			if len(paramTypes) > 0 {
				pty = paramTypes[0]
			}
		} else {
			switch p.Ownership {
			case lex.KwRef:
				pty = c.Types.InternRef(pty)
			case lex.KwMutref:
				pty = c.Types.InternMutRef(pty)
			}
		}
		c.localScope.Define(&resolve.Symbol{
			Name: p.Name,
			Kind: resolve.SymParam,
			Span: p.Span,
		})
		c.localTypes[p.Name] = pty
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

// HasDropImpl returns true if the given type name has a Drop trait implementation.
func (c *Checker) HasDropImpl(typeName string) bool {
	return c.traitImpls["Drop:"+typeName]
}

// DropTypes returns a map of TypeIds that have Drop implementations.
func (c *Checker) DropTypes() map[typetable.TypeId]bool {
	result := make(map[typetable.TypeId]bool)
	for key := range c.traitImpls {
		// key format is "TraitName:TypeName"
		if len(key) > 5 && key[:5] == "Drop:" {
			typeName := key[5:]
			// Find the struct type for this name.
			for _, k := range c.Graph.Order {
				mod := c.Graph.Modules[k]
				for _, item := range mod.File.Items {
					if sd, ok := item.(*ast.StructDecl); ok && sd.Name == typeName {
						ty := c.Types.InternStruct(mod.Path.String(), typeName, nil)
						result[ty] = true
					}
				}
			}
		}
	}
	return result
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
