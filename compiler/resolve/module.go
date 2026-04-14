// Package resolve owns module discovery, symbol indexing, scopes,
// import resolution, and HIR construction inputs.
package resolve

import (
	"sort"

	"github.com/Tembocs/fuse4/compiler/ast"
	"github.com/Tembocs/fuse4/compiler/diagnostics"
)

// ModulePath is a dot-separated module identifier such as ["core", "list"].
type ModulePath []string

func (p ModulePath) String() string {
	s := ""
	for i, seg := range p {
		if i > 0 {
			s += "."
		}
		s += seg
	}
	return s
}

// Module represents a single source module (one file).
type Module struct {
	Path    ModulePath
	File    *ast.File
	Symbols *Scope // module-level scope
}

// ModuleGraph is the set of all modules discovered in a compilation unit.
type ModuleGraph struct {
	Modules map[string]*Module // keyed by ModulePath.String()
	Order   []string           // deterministic traversal order
}

// NewModuleGraph creates an empty module graph.
func NewModuleGraph() *ModuleGraph {
	return &ModuleGraph{
		Modules: make(map[string]*Module),
	}
}

// Add registers a module. Deterministic order is maintained via sort on Finalize.
func (g *ModuleGraph) Add(m *Module) {
	key := m.Path.String()
	g.Modules[key] = m
}

// Finalize sorts module keys for deterministic iteration.
func (g *ModuleGraph) Finalize() {
	g.Order = make([]string, 0, len(g.Modules))
	for k := range g.Modules {
		g.Order = append(g.Order, k)
	}
	sort.Strings(g.Order)
}

// Lookup returns the module at the given path, or nil.
func (g *ModuleGraph) Lookup(path ModulePath) *Module {
	return g.Modules[path.String()]
}

// BuildModuleGraph parses a set of named source files into a module graph.
// Each entry maps a module path (e.g. "core.list") to its parsed AST.
// This is the Phase 01 entry point: it collects modules and their imports
// without performing semantic resolution.
func BuildModuleGraph(files map[string]*ast.File) *ModuleGraph {
	graph := NewModuleGraph()
	for pathStr, file := range files {
		modPath := splitModulePath(pathStr)
		mod := &Module{
			Path:    modPath,
			File:    file,
			Symbols: NewScope(nil),
		}
		graph.Add(mod)
	}
	graph.Finalize()
	return graph
}

func splitModulePath(s string) ModulePath {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// CollectImports extracts all import declarations from a module.
func CollectImports(mod *Module) []ast.ImportDecl {
	var imports []ast.ImportDecl
	for _, item := range mod.File.Items {
		if imp, ok := item.(*ast.ImportDecl); ok {
			imports = append(imports, *imp)
		}
	}
	return imports
}

// ImportEdge records a dependency from one module to another.
type ImportEdge struct {
	From     ModulePath
	To       ModulePath
	ItemName string           // non-empty if the import targets an item, not a module
	Alias    string           // non-empty if aliased
	Span     diagnostics.Span // location of the import statement
}
