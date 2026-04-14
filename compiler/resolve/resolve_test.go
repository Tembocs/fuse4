package resolve

import (
	"strings"
	"testing"

	"github.com/Tembocs/fuse4/compiler/ast"
	"github.com/Tembocs/fuse4/compiler/parse"
)

// --- helpers ---

// buildAndResolve parses a map of module paths to source code, builds a module
// graph, and runs the resolver. Returns the resolver (with errors) and graph.
func buildAndResolve(t *testing.T, sources map[string]string) (*Resolver, *ModuleGraph) {
	t.Helper()
	parsed := make(map[string]*ast.File)
	for path, src := range sources {
		f, errs := parse.Parse(path+".fuse", []byte(src))
		for _, e := range errs {
			t.Errorf("[%s] parse error: %s", path, e)
		}
		parsed[path] = f
	}

	graph := BuildModuleGraph(parsed)
	resolver := NewResolver(graph)
	resolver.Resolve()
	return resolver, graph
}

func resolveOK(t *testing.T, sources map[string]string) (*Resolver, *ModuleGraph) {
	t.Helper()
	r, g := buildAndResolve(t, sources)
	for _, e := range r.Errors {
		t.Errorf("unexpected error: %s", e)
	}
	return r, g
}

func resolveExpectErrors(t *testing.T, sources map[string]string) *Resolver {
	t.Helper()
	r, _ := buildAndResolve(t, sources)
	return r
}

func hasError(r *Resolver, substr string) bool {
	for _, e := range r.Errors {
		if strings.Contains(e.Message, substr) {
			return true
		}
	}
	return false
}

// ===== Phase 01: Module graph =====

func TestModuleGraphDeterministicOrder(t *testing.T) {
	_, g := resolveOK(t, map[string]string{
		"c": "",
		"a": "",
		"b": "",
	})
	if len(g.Order) != 3 {
		t.Fatalf("modules: %d", len(g.Order))
	}
	// Order must be sorted
	if g.Order[0] != "a" || g.Order[1] != "b" || g.Order[2] != "c" {
		t.Errorf("order: %v", g.Order)
	}
}

func TestModuleGraphLookup(t *testing.T) {
	_, g := resolveOK(t, map[string]string{
		"core.list": "pub fn new() { }",
	})
	mod := g.Lookup(ModulePath{"core", "list"})
	if mod == nil {
		t.Fatal("expected to find core.list")
	}
	if mod.Path.String() != "core.list" {
		t.Errorf("path: %q", mod.Path.String())
	}
}

// ===== Phase 02: Symbol table and scopes =====

func TestScopeDefineAndLookup(t *testing.T) {
	s := NewScope(nil)
	sym := &Symbol{Name: "x", Kind: SymLocal}
	if !s.Define(sym) {
		t.Fatal("first define should succeed")
	}
	if s.Define(sym) {
		t.Fatal("duplicate define should fail")
	}
	found := s.Lookup("x")
	if found == nil || found.Name != "x" {
		t.Error("lookup failed")
	}
}

func TestScopeNestedLookup(t *testing.T) {
	parent := NewScope(nil)
	parent.Define(&Symbol{Name: "x", Kind: SymLocal})
	child := NewScope(parent)
	child.Define(&Symbol{Name: "y", Kind: SymLocal})

	if child.Lookup("x") == nil {
		t.Error("child should find parent's x")
	}
	if child.Lookup("y") == nil {
		t.Error("child should find its own y")
	}
	if parent.Lookup("y") != nil {
		t.Error("parent should not find child's y")
	}
}

func TestScopeLookupLocal(t *testing.T) {
	parent := NewScope(nil)
	parent.Define(&Symbol{Name: "x", Kind: SymLocal})
	child := NewScope(parent)

	if child.LookupLocal("x") != nil {
		t.Error("LookupLocal should not search parent")
	}
}

func TestScopeNamesAreSorted(t *testing.T) {
	s := NewScope(nil)
	s.Define(&Symbol{Name: "z"})
	s.Define(&Symbol{Name: "a"})
	s.Define(&Symbol{Name: "m"})
	names := s.Names()
	if len(names) != 3 || names[0] != "a" || names[1] != "m" || names[2] != "z" {
		t.Errorf("names: %v", names)
	}
}

// ===== Phase 02: Top-level symbol indexing =====

func TestIndexFunctions(t *testing.T) {
	_, g := resolveOK(t, map[string]string{
		"main": "pub fn foo() { } fn bar() { }",
	})
	mod := g.Lookup(ModulePath{"main"})
	if mod.Symbols.Lookup("foo") == nil {
		t.Error("foo not indexed")
	}
	if mod.Symbols.Lookup("bar") == nil {
		t.Error("bar not indexed")
	}
	if mod.Symbols.Lookup("foo").Kind != SymFunc {
		t.Error("foo should be SymFunc")
	}
}

func TestIndexStructs(t *testing.T) {
	_, g := resolveOK(t, map[string]string{
		"main": "struct Point { x: F64, y: F64 }",
	})
	mod := g.Lookup(ModulePath{"main"})
	if s := mod.Symbols.Lookup("Point"); s == nil || s.Kind != SymStruct {
		t.Error("Point not indexed as struct")
	}
}

func TestIndexEnumWithVariantHoisting(t *testing.T) {
	_, g := resolveOK(t, map[string]string{
		"main": "enum Color { Red, Green, Blue }",
	})
	mod := g.Lookup(ModulePath{"main"})
	if s := mod.Symbols.Lookup("Color"); s == nil || s.Kind != SymEnum {
		t.Error("Color not indexed as enum")
	}
	// Variants should be hoisted into module scope.
	for _, name := range []string{"Red", "Green", "Blue"} {
		s := mod.Symbols.Lookup(name)
		if s == nil {
			t.Errorf("variant %q not hoisted", name)
			continue
		}
		if s.Kind != SymEnumVariant {
			t.Errorf("variant %q kind=%s", name, s.Kind)
		}
		if s.Parent != "Color" {
			t.Errorf("variant %q parent=%q", name, s.Parent)
		}
	}
}

func TestIndexTraitConstTypeExtern(t *testing.T) {
	_, g := resolveOK(t, map[string]string{
		"main": `
			trait Display { fn fmt(ref self); }
			const MAX: I32 = 100;
			type Pair = (Int, Int);
			extern fn puts(s: Ptr[U8]) -> I32;
		`,
	})
	mod := g.Lookup(ModulePath{"main"})
	cases := []struct {
		name string
		kind SymbolKind
	}{
		{"Display", SymTrait},
		{"MAX", SymConst},
		{"Pair", SymTypeAlias},
		{"puts", SymExternFn},
	}
	for _, tc := range cases {
		s := mod.Symbols.Lookup(tc.name)
		if s == nil {
			t.Errorf("%q not indexed", tc.name)
			continue
		}
		if s.Kind != tc.kind {
			t.Errorf("%q kind: got %s, want %s", tc.name, s.Kind, tc.kind)
		}
	}
}

func TestDuplicateDefinition(t *testing.T) {
	r := resolveExpectErrors(t, map[string]string{
		"main": "fn foo() { } fn foo() { }",
	})
	if !hasError(r, "duplicate definition") {
		t.Error("expected duplicate definition error")
	}
}

func TestVariantConflict(t *testing.T) {
	r := resolveExpectErrors(t, map[string]string{
		"main": `
			enum A { X, Y }
			enum B { X, Z }
		`,
	})
	if !hasError(r, "conflicts") {
		t.Error("expected variant conflict error")
	}
}

// ===== Phase 03: Import resolution =====

func TestImportModule(t *testing.T) {
	_, g := resolveOK(t, map[string]string{
		"core.list": "pub fn new() { }",
		"main":      "import core.list;",
	})
	mod := g.Lookup(ModulePath{"main"})
	s := mod.Symbols.Lookup("list")
	if s == nil {
		t.Fatal("import 'list' not resolved")
	}
	if s.Kind != SymModule {
		t.Errorf("kind: %s", s.Kind)
	}
}

func TestImportItemFromModule(t *testing.T) {
	_, g := resolveOK(t, map[string]string{
		"core.list": "pub struct List { }",
		"main":      "import core.list.List;",
	})
	mod := g.Lookup(ModulePath{"main"})
	s := mod.Symbols.Lookup("List")
	if s == nil {
		t.Fatal("import 'List' not resolved")
	}
	if s.Kind != SymStruct {
		t.Errorf("kind: %s", s.Kind)
	}
}

func TestImportWithAlias(t *testing.T) {
	_, g := resolveOK(t, map[string]string{
		"core.list": "pub struct List { }",
		"main":      "import core.list.List as L;",
	})
	mod := g.Lookup(ModulePath{"main"})
	if mod.Symbols.Lookup("L") == nil {
		t.Error("aliased import 'L' not found")
	}
	if mod.Symbols.Lookup("List") != nil {
		t.Error("original name 'List' should not be in scope")
	}
}

func TestImportModuleFirstFallback(t *testing.T) {
	// import core.list.List → no module "core.list.List" exists,
	// so fall back to item "List" in module "core.list".
	_, g := resolveOK(t, map[string]string{
		"core.list": "pub struct List { }",
		"main":      "import core.list.List;",
	})
	mod := g.Lookup(ModulePath{"main"})
	s := mod.Symbols.Lookup("List")
	if s == nil || s.Kind != SymStruct {
		t.Fatal("module-first fallback should resolve List as struct")
	}
}

func TestImportUnresolved(t *testing.T) {
	r := resolveExpectErrors(t, map[string]string{
		"main": "import nonexistent.module;",
	})
	if !hasError(r, "unresolved import") {
		t.Error("expected unresolved import error")
	}
}

func TestImportItemNotFound(t *testing.T) {
	r := resolveExpectErrors(t, map[string]string{
		"core.list": "pub fn new() { }",
		"main":      "import core.list.Nonexistent;",
	})
	if !hasError(r, "no exported item") {
		t.Error("expected 'no exported item' error")
	}
}

func TestImportConflict(t *testing.T) {
	r := resolveExpectErrors(t, map[string]string{
		"core.list": "pub struct List { }",
		"main":      "import core.list.List; import core.list.List;",
	})
	if !hasError(r, "conflicts") {
		t.Error("expected import conflict error")
	}
}

// ===== Phase 03: Qualified enum variant access =====

func TestQualifiedVariantResolution(t *testing.T) {
	_, g := resolveOK(t, map[string]string{
		"main": "enum Option { Some, None }",
	})
	mod := g.Lookup(ModulePath{"main"})

	// Option.Some should resolve
	sym := ResolveQualifiedVariant(mod.Symbols, "Option", "Some")
	if sym == nil {
		t.Fatal("Option.Some should resolve")
	}
	if sym.Kind != SymEnumVariant || sym.Parent != "Option" {
		t.Errorf("sym: kind=%s parent=%q", sym.Kind, sym.Parent)
	}

	// Option.None should resolve
	sym = ResolveQualifiedVariant(mod.Symbols, "Option", "None")
	if sym == nil {
		t.Fatal("Option.None should resolve")
	}

	// Option.Bogus should not resolve
	sym = ResolveQualifiedVariant(mod.Symbols, "Option", "Bogus")
	if sym != nil {
		t.Error("Option.Bogus should not resolve")
	}

	// Nonexistent.Some should not resolve
	sym = ResolveQualifiedVariant(mod.Symbols, "Nonexistent", "Some")
	if sym != nil {
		t.Error("Nonexistent.Some should not resolve")
	}
}

// ===== Phase 03: Import cycle detection =====

func TestImportCycleDetected(t *testing.T) {
	r := resolveExpectErrors(t, map[string]string{
		"a": "import b;",
		"b": "import a;",
	})
	if !hasError(r, "import cycle") {
		t.Error("expected import cycle error")
	}
}

func TestImportCycleThreeModules(t *testing.T) {
	r := resolveExpectErrors(t, map[string]string{
		"a": "import b;",
		"b": "import c;",
		"c": "import a;",
	})
	if !hasError(r, "import cycle") {
		t.Error("expected import cycle error")
	}
}

func TestNoCycleForDAG(t *testing.T) {
	// Diamond dependency: a→b, a→c, b→d, c→d (no cycle)
	resolveOK(t, map[string]string{
		"a": "import b; import c;",
		"b": "import d;",
		"c": "import d;",
		"d": "pub fn shared() { }",
	})
}

func TestSelfImportIsCycle(t *testing.T) {
	r := resolveExpectErrors(t, map[string]string{
		"a": "import a;",
	})
	if !hasError(r, "import cycle") {
		t.Error("expected import cycle error for self-import")
	}
}
