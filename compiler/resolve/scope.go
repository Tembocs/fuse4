package resolve

import "github.com/Tembocs/fuse4/compiler/diagnostics"

// SymbolKind classifies what a symbol refers to.
type SymbolKind int

const (
	SymFunc SymbolKind = iota
	SymStruct
	SymEnum
	SymEnumVariant
	SymTrait
	SymConst
	SymTypeAlias
	SymExternFn
	SymParam
	SymLocal
	SymImport  // an imported name
	SymModule  // a module reference
)

func (k SymbolKind) String() string {
	switch k {
	case SymFunc:
		return "function"
	case SymStruct:
		return "struct"
	case SymEnum:
		return "enum"
	case SymEnumVariant:
		return "enum variant"
	case SymTrait:
		return "trait"
	case SymConst:
		return "constant"
	case SymTypeAlias:
		return "type alias"
	case SymExternFn:
		return "extern function"
	case SymParam:
		return "parameter"
	case SymLocal:
		return "local"
	case SymImport:
		return "import"
	case SymModule:
		return "module"
	default:
		return "symbol"
	}
}

// Symbol is a resolved name binding.
type Symbol struct {
	Name   string
	Kind   SymbolKind
	Public bool
	Span   diagnostics.Span
	// Module is the defining module path (for qualified lookup).
	Module ModulePath
	// Parent is the enclosing type name for enum variants.
	Parent string
}

// Scope is a lexical scope that maps names to symbols.
// Scopes form a chain via the Parent pointer for nested lookup.
type Scope struct {
	Parent  *Scope
	Symbols map[string]*Symbol
}

// NewScope creates a scope with an optional parent.
func NewScope(parent *Scope) *Scope {
	return &Scope{
		Parent:  parent,
		Symbols: make(map[string]*Symbol),
	}
}

// Define adds a symbol to this scope. Returns false if already defined
// at this exact scope level.
func (s *Scope) Define(sym *Symbol) bool {
	if _, exists := s.Symbols[sym.Name]; exists {
		return false
	}
	s.Symbols[sym.Name] = sym
	return true
}

// Lookup searches this scope and its parents for a name.
func (s *Scope) Lookup(name string) *Symbol {
	for cur := s; cur != nil; cur = cur.Parent {
		if sym, ok := cur.Symbols[name]; ok {
			return sym
		}
	}
	return nil
}

// LookupLocal searches only this scope, not parents.
func (s *Scope) LookupLocal(name string) *Symbol {
	return s.Symbols[name]
}

// Names returns all symbol names in this scope (not parents), sorted.
func (s *Scope) Names() []string {
	names := make([]string, 0, len(s.Symbols))
	for n := range s.Symbols {
		names = append(names, n)
	}
	// Deterministic order
	for i := 1; i < len(names); i++ {
		for j := i; j > 0 && names[j] < names[j-1]; j-- {
			names[j], names[j-1] = names[j-1], names[j]
		}
	}
	return names
}
