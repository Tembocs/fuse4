package hir

import (
	"testing"

	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

var zeroSpan = diagnostics.Span{}

// ===== Builder enforcement (Rule 3.3) =====

func TestBuilderSetsDefaultMetadata(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder(tt)

	lit := b.Literal(zeroSpan, "42", tt.I32)
	if lit.Meta().Type != tt.I32 {
		t.Errorf("type: %d", lit.Meta().Type)
	}
	if lit.Meta().Ownership != OwnerValue {
		t.Errorf("ownership: %s", lit.Meta().Ownership)
	}
}

func TestBuilderReturnDiverges(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder(tt)

	ret := b.Return(zeroSpan, nil)
	if !ret.Meta().Diverges {
		t.Error("return should have Diverges=true")
	}
	if ret.Meta().Type != tt.Never {
		t.Error("return should have Never type")
	}
}

func TestBuilderBreakDiverges(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder(tt)

	brk := b.Break(zeroSpan, nil)
	if !brk.Meta().Diverges {
		t.Error("break should have Diverges=true")
	}
}

func TestBuilderContinueDiverges(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder(tt)

	cont := b.Continue(zeroSpan)
	if !cont.Meta().Diverges {
		t.Error("continue should have Diverges=true")
	}
}

func TestBuilderAssignHasUnitType(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder(tt)

	lhs := b.Ident(zeroSpan, "x", tt.I32)
	rhs := b.Literal(zeroSpan, "1", tt.I32)
	assign := b.Assign(zeroSpan, "=", lhs, rhs)
	if assign.Meta().Type != tt.Unit {
		t.Errorf("assign type: %d, want Unit(%d)", assign.Meta().Type, tt.Unit)
	}
}

func TestBuilderBlockExpr(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder(tt)

	tail := b.Literal(zeroSpan, "true", tt.Bool)
	block := b.BlockExpr(zeroSpan, nil, tail, tt.Bool)
	if block.Tail == nil {
		t.Error("block should have tail")
	}
	if block.Meta().Type != tt.Bool {
		t.Error("block type should be Bool")
	}
}

func TestBuilderCallExpr(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder(tt)

	callee := b.Ident(zeroSpan, "foo", tt.InternFunc([]typetable.TypeId{tt.I32}, tt.Bool))
	arg := b.Literal(zeroSpan, "1", tt.I32)
	call := b.Call(zeroSpan, callee, []Expr{arg}, tt.Bool)
	if call.Meta().Type != tt.Bool {
		t.Errorf("call return type: %d", call.Meta().Type)
	}
}

// ===== HIR is distinct from AST (Rule 3.2) =====

func TestHIRNodeInterface(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder(tt)

	// HIR expressions implement hir.Expr, not ast.Expr
	var e Expr = b.Literal(zeroSpan, "1", tt.I32)
	if e.Meta() == nil {
		t.Error("HIR node must have metadata")
	}
}

// ===== Pass manifest (Rule 3.4) =====

func TestDefaultPassManifestIsValid(t *testing.T) {
	_, errs := NewPassManifest(DefaultPasses())
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("manifest error: %s", e)
		}
	}
}

func TestPassManifestDetectsMissingWrite(t *testing.T) {
	passes := []PassDecl{
		{Name: "bad_pass", Reads: []MetadataKey{MDType}, Writes: nil},
	}
	_, errs := NewPassManifest(passes)
	if len(errs) == 0 {
		t.Error("expected manifest validation error")
	}
}

func TestPassManifestOrderMatters(t *testing.T) {
	passes := []PassDecl{
		{Name: "liveness", Reads: []MetadataKey{MDOwnership}, Writes: []MetadataKey{MDLiveAfter}},
		{Name: "check", Reads: nil, Writes: []MetadataKey{MDOwnership}},
	}
	_, errs := NewPassManifest(passes)
	if len(errs) == 0 {
		t.Error("liveness reading ownership before check writes it should fail")
	}
}

// ===== Invariant walkers (Rule 3.5) =====

func TestInvariantWalkerCatchesMissingDiverges(t *testing.T) {
	tt := typetable.New()

	// Manually construct a return node WITHOUT Diverges (simulating a bug).
	ret := &ReturnExpr{
		nodeBase: nodeBase{
			MD: Metadata{Type: tt.Never, Diverges: false}, // bug: should be true
		},
	}
	fn := &Function{Body: &Block{
		nodeBase: nodeBase{MD: Metadata{Type: tt.Unit}},
		Stmts:    []Stmt{&ExprStmt{nodeBase: nodeBase{MD: Metadata{Type: tt.Unit}}, Expr: ret}},
	}}

	violations := WalkInvariants(fn, "check")
	found := false
	for _, v := range violations {
		if v.Message == "return expression missing Diverges flag" {
			found = true
		}
	}
	if !found {
		t.Error("walker should catch missing Diverges on return")
	}
}

func TestInvariantWalkerPassesForCorrectHIR(t *testing.T) {
	tt := typetable.New()
	b := NewBuilder(tt)

	ret := b.Return(zeroSpan, b.Literal(zeroSpan, "0", tt.I32))
	block := b.BlockExpr(zeroSpan, []Stmt{b.ExprStatement(zeroSpan, ret)}, nil, tt.Never)
	fn := &Function{Body: block}

	violations := WalkInvariants(fn, "check")
	if len(violations) > 0 {
		for _, v := range violations {
			t.Errorf("unexpected violation: %s", v)
		}
	}
}

// ===== Deterministic IR collections (Rule 3.6) =====
// The TypeTable uses a slice (not map iteration) for entries, and the
// module graph uses sorted order. This test ensures TypeTable IDs are
// stable across repeated construction.

func TestTypeIdStability(t *testing.T) {
	for i := 0; i < 5; i++ {
		tt := typetable.New()
		if tt.I32 != typetable.New().I32 {
			t.Fatal("I32 TypeId should be stable across table constructions")
		}
	}
}
