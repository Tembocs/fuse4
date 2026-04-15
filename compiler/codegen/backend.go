package codegen

import (
	"github.com/Tembocs/fuse4/compiler/mir"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

// Backend is the common interface for all code generation backends.
// Both the C11 backend and the native backend implement this interface,
// ensuring the same semantic contracts are enforced regardless of target.
type Backend interface {
	// Name returns the backend identifier ("c11" or "native").
	Name() string

	// Emit generates output for a list of MIR functions.
	// Returns the generated source/binary as bytes.
	Emit(functions []*mir.Function) ([]byte, error)
}

// BackendConfig selects and configures the code generation backend.
type BackendConfig struct {
	// Target selects the backend: "c11" or "native".
	Target string

	// Types is the global type table.
	Types *typetable.TypeTable

	// Optimize enables backend optimizations.
	Optimize bool

	// DropTypes maps TypeIds that have Drop trait implementations.
	DropTypes map[typetable.TypeId]bool
}

// NewBackend creates the appropriate backend based on config.
func NewBackend(cfg BackendConfig) Backend {
	switch cfg.Target {
	case "native":
		return NewNativeBackend(cfg.Types, cfg.Optimize)
	default:
		return NewC11Backend(cfg.Types, cfg.DropTypes)
	}
}

// C11Backend wraps the existing Emitter as a Backend.
type C11Backend struct {
	emitter *Emitter
}

// NewC11Backend creates the C11 code generation backend.
func NewC11Backend(types *typetable.TypeTable, dropTypes map[typetable.TypeId]bool) *C11Backend {
	e := NewEmitter(types)
	if dropTypes != nil {
		e.DropTypes = dropTypes
	}
	return &C11Backend{emitter: e}
}

func (b *C11Backend) Name() string { return "c11" }

func (b *C11Backend) Emit(functions []*mir.Function) ([]byte, error) {
	src := b.emitter.Emit(functions)
	return []byte(src), nil
}
