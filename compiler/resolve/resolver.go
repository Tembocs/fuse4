package resolve

import (
	"github.com/Tembocs/fuse4/compiler/ast"
	"github.com/Tembocs/fuse4/compiler/diagnostics"
)

// Resolver performs name resolution across a module graph.
type Resolver struct {
	Graph  *ModuleGraph
	Errors []diagnostics.Diagnostic
}

// NewResolver creates a resolver for the given module graph.
func NewResolver(graph *ModuleGraph) *Resolver {
	return &Resolver{Graph: graph}
}

// Resolve runs all resolution phases:
//  1. Index top-level symbols in each module.
//  2. Hoist enum variants into module scope.
//  3. Resolve imports with module-first fallback.
//  4. Detect import cycles.
func (r *Resolver) Resolve() {
	// Phase 1 + 2: index symbols and hoist variants
	for _, key := range r.Graph.Order {
		mod := r.Graph.Modules[key]
		r.indexModule(mod)
	}

	// Phase 3: resolve imports
	for _, key := range r.Graph.Order {
		mod := r.Graph.Modules[key]
		r.resolveImports(mod)
	}

	// Phase 4: cycle detection
	r.detectCycles()
}

// ---------- Phase 1+2: index top-level symbols ----------

func (r *Resolver) indexModule(mod *Module) {
	for _, item := range mod.File.Items {
		switch n := item.(type) {
		case *ast.FnDecl:
			r.define(mod, n.Name, SymFunc, n.Public, n.Span)
		case *ast.StructDecl:
			r.define(mod, n.Name, SymStruct, n.Public, n.Span)
		case *ast.EnumDecl:
			r.define(mod, n.Name, SymEnum, n.Public, n.Span)
			// Hoist enum variants into module scope (per language guide).
			for _, v := range n.Variants {
				r.defineVariant(mod, n.Name, v)
			}
		case *ast.TraitDecl:
			r.define(mod, n.Name, SymTrait, n.Public, n.Span)
		case *ast.ConstDecl:
			r.define(mod, n.Name, SymConst, n.Public, n.Span)
		case *ast.TypeAliasDecl:
			r.define(mod, n.Name, SymTypeAlias, n.Public, n.Span)
		case *ast.ExternFnDecl:
			r.define(mod, n.Name, SymExternFn, n.Public, n.Span)
		case *ast.ImportDecl:
			// imports are handled in resolveImports
		}
	}
}

func (r *Resolver) define(mod *Module, name string, kind SymbolKind, public bool, span diagnostics.Span) {
	sym := &Symbol{
		Name:   name,
		Kind:   kind,
		Public: public,
		Span:   span,
		Module: mod.Path,
	}
	if !mod.Symbols.Define(sym) {
		prev := mod.Symbols.LookupLocal(name)
		r.errorf(span, "duplicate definition of %q (previous at %s)", name, prev.Span)
	}
}

func (r *Resolver) defineVariant(mod *Module, enumName string, v ast.Variant) {
	sym := &Symbol{
		Name:   v.Name,
		Kind:   SymEnumVariant,
		Public: true, // variants are public if the enum is
		Span:   v.Span,
		Module: mod.Path,
		Parent: enumName,
	}
	if !mod.Symbols.Define(sym) {
		prev := mod.Symbols.LookupLocal(v.Name)
		// Conflict: two enums in the same module export the same variant name.
		r.errorf(v.Span,
			"enum variant %q conflicts with existing %s %q (at %s)",
			v.Name, prev.Kind, prev.Name, prev.Span)
	}
}

// ---------- Phase 3: resolve imports ----------

func (r *Resolver) resolveImports(mod *Module) {
	for _, item := range mod.File.Items {
		imp, ok := item.(*ast.ImportDecl)
		if !ok {
			continue
		}
		r.resolveImport(mod, imp)
	}
}

// resolveImport implements the module-first import resolution contract:
//
//	First, treat the full dotted path as a module path. If a module exists
//	at that path, import the entire module. Otherwise, treat the final
//	segment as an item name inside the preceding module path.
func (r *Resolver) resolveImport(mod *Module, imp *ast.ImportDecl) {
	path := imp.Path

	// Try 1: full path as module
	fullPath := ModulePath(path)
	if target := r.Graph.Lookup(fullPath); target != nil {
		name := imp.Alias
		if name == "" {
			name = path[len(path)-1]
		}
		sym := &Symbol{
			Name:   name,
			Kind:   SymModule,
			Public: false,
			Span:   imp.Span,
			Module: fullPath,
		}
		if !mod.Symbols.Define(sym) {
			r.errorf(imp.Span, "import %q conflicts with existing symbol %q", name, name)
		}
		return
	}

	// Try 2: last segment is an item name in the parent module
	if len(path) >= 2 {
		modPath := ModulePath(path[:len(path)-1])
		itemName := path[len(path)-1]
		if target := r.Graph.Lookup(modPath); target != nil {
			srcSym := target.Symbols.LookupLocal(itemName)
			if srcSym != nil {
				name := imp.Alias
				if name == "" {
					name = itemName
				}
				sym := &Symbol{
					Name:   name,
					Kind:   srcSym.Kind,
					Public: false,
					Span:   imp.Span,
					Module: srcSym.Module,
					Parent: srcSym.Parent,
				}
				if !mod.Symbols.Define(sym) {
					r.errorf(imp.Span, "import %q conflicts with existing symbol %q", name, name)
				}
				return
			}
			r.errorf(imp.Span, "module %s has no exported item %q", modPath, itemName)
			return
		}
	}

	r.errorf(imp.Span, "unresolved import: %s", fullPath)
}

// ---------- Phase 4: import cycle detection ----------

func (r *Resolver) detectCycles() {
	// Build adjacency list from import edges.
	adj := make(map[string][]string) // module path → imported module paths
	for _, key := range r.Graph.Order {
		mod := r.Graph.Modules[key]
		for _, item := range mod.File.Items {
			imp, ok := item.(*ast.ImportDecl)
			if !ok {
				continue
			}
			edge := r.resolveImportTarget(imp)
			if edge != "" {
				adj[key] = append(adj[key], edge)
			}
		}
	}

	// Standard DFS cycle detection with path tracking.
	const (
		white = 0 // unvisited
		gray  = 1 // in current DFS path
		black = 2 // fully visited
	)
	color := make(map[string]int)
	parent := make(map[string]string) // for reconstructing cycle path

	var dfs func(node string) bool
	dfs = func(node string) bool {
		color[node] = gray
		for _, next := range adj[node] {
			switch color[next] {
			case gray:
				// Found a cycle — reconstruct path.
				cycle := r.buildCyclePath(next, node, parent)
				mod := r.Graph.Modules[node]
				span := diagnostics.Span{}
				if mod != nil && mod.File != nil {
					span = mod.File.Span
				}
				r.errorf(span, "import cycle detected: %s", cycle)
				return true
			case white:
				parent[next] = node
				if dfs(next) {
					return true
				}
			}
		}
		color[node] = black
		return false
	}

	for _, key := range r.Graph.Order {
		if color[key] == white {
			dfs(key)
		}
	}
}

// resolveImportTarget returns the module path string that an import targets,
// using the same module-first fallback logic but only for adjacency building.
func (r *Resolver) resolveImportTarget(imp *ast.ImportDecl) string {
	path := imp.Path

	// Full path as module
	fullPath := ModulePath(path)
	if r.Graph.Lookup(fullPath) != nil {
		return fullPath.String()
	}

	// Parent module
	if len(path) >= 2 {
		modPath := ModulePath(path[:len(path)-1])
		if r.Graph.Lookup(modPath) != nil {
			return modPath.String()
		}
	}

	return ""
}

func (r *Resolver) buildCyclePath(cycleStart, cycleEnd string, parent map[string]string) string {
	var path []string
	path = append(path, cycleStart)
	cur := cycleEnd
	for cur != cycleStart {
		path = append(path, cur)
		cur = parent[cur]
	}
	// Reverse to get the forward path.
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	path = append(path, cycleStart) // close the cycle
	result := path[0]
	for _, p := range path[1:] {
		result += " -> " + p
	}
	return result
}

// ---------- Qualified enum variant resolution ----------

// ResolveQualifiedVariant looks up EnumName.Variant in a module scope.
// Returns the variant symbol if found, nil otherwise.
func ResolveQualifiedVariant(scope *Scope, enumName, variantName string) *Symbol {
	// Check that enumName is an enum in scope.
	enumSym := scope.Lookup(enumName)
	if enumSym == nil || enumSym.Kind != SymEnum {
		return nil
	}
	// The variant should be hoisted into the same scope with Parent == enumName.
	varSym := scope.Lookup(variantName)
	if varSym != nil && varSym.Kind == SymEnumVariant && varSym.Parent == enumName {
		return varSym
	}
	return nil
}

// ---------- errors ----------

func (r *Resolver) errorf(span diagnostics.Span, format string, args ...any) {
	r.Errors = append(r.Errors, diagnostics.Errorf(span, format, args...))
}
