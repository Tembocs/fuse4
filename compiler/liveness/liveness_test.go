package liveness

import (
	"testing"

	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/hir"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

var zs = diagnostics.Span{}

func makeBuilder() (*hir.Builder, *typetable.TypeTable) {
	tt := typetable.New()
	return hir.NewBuilder(tt), tt
}

// ===== Phase 01: Ownership analysis =====

func TestOwnershipValueDefault(t *testing.T) {
	b, tt := makeBuilder()
	lit := b.Literal(zs, "42", tt.I32)
	block := b.BlockExpr(zs, nil, lit, tt.I32)
	fn := &hir.Function{Body: block}

	oa := &OwnershipAnalyzer{}
	oa.AnalyzeOwnership(fn)

	if lit.Meta().Ownership != hir.OwnerValue {
		t.Errorf("literal should be OwnerValue, got %s", lit.Meta().Ownership)
	}
}

func TestOwnershipRefPropagation(t *testing.T) {
	b, tt := makeBuilder()
	x := b.Ident(zs, "x", tt.I32)
	refX := b.Unary(zs, "ref", x, tt.InternRef(tt.I32))
	block := b.BlockExpr(zs, nil, refX, tt.InternRef(tt.I32))
	fn := &hir.Function{Body: block}

	oa := &OwnershipAnalyzer{}
	oa.AnalyzeOwnership(fn)

	if x.Meta().Ownership != hir.OwnerRef {
		t.Errorf("ref operand should be OwnerRef, got %s", x.Meta().Ownership)
	}
}

func TestOwnershipMutrefPropagation(t *testing.T) {
	b, tt := makeBuilder()
	x := b.Ident(zs, "x", tt.I32)
	mutrefX := b.Unary(zs, "mutref", x, tt.InternMutRef(tt.I32))
	block := b.BlockExpr(zs, nil, mutrefX, tt.InternMutRef(tt.I32))
	fn := &hir.Function{Body: block}

	oa := &OwnershipAnalyzer{}
	oa.AnalyzeOwnership(fn)

	if x.Meta().Ownership != hir.OwnerMutRef {
		t.Errorf("mutref operand should be OwnerMutRef, got %s", x.Meta().Ownership)
	}
}

func TestOwnershipMovePropagation(t *testing.T) {
	b, tt := makeBuilder()
	x := b.Ident(zs, "x", tt.I32)
	moveX := b.Unary(zs, "move", x, tt.I32)
	block := b.BlockExpr(zs, nil, moveX, tt.I32)
	fn := &hir.Function{Body: block}

	oa := &OwnershipAnalyzer{}
	oa.AnalyzeOwnership(fn)

	if x.Meta().Ownership != hir.OwnerMoved {
		t.Errorf("move operand should be OwnerMoved, got %s", x.Meta().Ownership)
	}
}

func TestOwnershipSpawnIsOwned(t *testing.T) {
	b, tt := makeBuilder()
	closure := b.Closure(zs, nil, tt.Unit, b.BlockExpr(zs, nil, nil, tt.Unit), tt.InternFunc(nil, tt.Unit))
	spawn := b.Spawn(zs, closure)
	block := b.BlockExpr(zs, []hir.Stmt{b.ExprStatement(zs, spawn)}, nil, tt.Unit)
	fn := &hir.Function{Body: block}

	oa := &OwnershipAnalyzer{}
	oa.AnalyzeOwnership(fn)

	if closure.Meta().Ownership != hir.OwnerOwned {
		t.Errorf("spawn operand should be OwnerOwned, got %s", closure.Meta().Ownership)
	}
}

// ===== Phase 02: Liveness computation =====

func TestLivenessTracksParams(t *testing.T) {
	b, tt := makeBuilder()
	x := b.Ident(zs, "x", tt.I32)
	block := b.BlockExpr(zs, nil, x, tt.I32)
	fn := &hir.Function{
		Body:   block,
		Params: []hir.Param{{Span: zs, Name: "x", Type: tt.I32, Ownership: hir.OwnerValue}},
	}

	result := ComputeLiveness(fn)
	if len(result.Slots) != 1 || result.Slots[0].Name != "x" {
		t.Errorf("expected 1 param slot 'x', got %v", result.Slots)
	}
}

func TestLivenessTracksLocals(t *testing.T) {
	b, tt := makeBuilder()
	letStmt := b.Let(zs, "y", tt.I32, b.Literal(zs, "1", tt.I32))
	yIdent := b.Ident(zs, "y", tt.I32)
	block := b.BlockExpr(zs, []hir.Stmt{letStmt}, yIdent, tt.I32)
	fn := &hir.Function{Body: block}

	result := ComputeLiveness(fn)
	found := false
	for _, s := range result.Slots {
		if s.Name == "y" {
			found = true
		}
	}
	if !found {
		t.Error("local 'y' should be tracked")
	}
}

func TestLivenessRecordsLastUse(t *testing.T) {
	b, tt := makeBuilder()
	xUse := b.Ident(zs, "x", tt.I32)
	block := b.BlockExpr(zs, nil, xUse, tt.I32)
	fn := &hir.Function{
		Body:   block,
		Params: []hir.Param{{Span: zs, Name: "x", Type: tt.I32}},
	}

	result := ComputeLiveness(fn)
	if last, ok := result.LastUses["x"]; !ok || last != xUse {
		t.Error("last use of 'x' should be the ident expression")
	}
}

func TestLivenessMultipleUses(t *testing.T) {
	b, tt := makeBuilder()
	x1 := b.Ident(zs, "x", tt.I32)
	x2 := b.Ident(zs, "x", tt.I32)
	add := b.Binary(zs, "+", x1, x2, tt.I32)
	block := b.BlockExpr(zs, nil, add, tt.I32)
	fn := &hir.Function{
		Body:   block,
		Params: []hir.Param{{Span: zs, Name: "x", Type: tt.I32}},
	}

	result := ComputeLiveness(fn)
	// Last use should be x2 (the right operand, walked second).
	if last, ok := result.LastUses["x"]; !ok || last != x2 {
		t.Error("last use of 'x' should be the second ident")
	}
}

func TestLiveAfterPopulated(t *testing.T) {
	b, tt := makeBuilder()
	xUse := b.Ident(zs, "x", tt.I32)
	block := b.BlockExpr(zs, nil, xUse, tt.I32)
	fn := &hir.Function{
		Body:   block,
		Params: []hir.Param{{Span: zs, Name: "x", Type: tt.I32}},
	}

	ComputeLiveness(fn)
	if xUse.Meta().LiveAfter == nil {
		t.Error("LiveAfter should be populated on ident use")
	}
}

// ===== Phase 03: Deterministic destruction =====

func TestDestroyFlagOnLastUseOfOwnedLocal(t *testing.T) {
	b, tt := makeBuilder()
	xUse := b.Ident(zs, "x", tt.I32)
	block := b.BlockExpr(zs, nil, xUse, tt.I32)
	fn := &hir.Function{
		Body:   block,
		Params: []hir.Param{{Span: zs, Name: "x", Type: tt.I32, Ownership: hir.OwnerValue}},
	}

	result, errs := RunAll(fn)
	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}

	_ = result
	if !xUse.Meta().DestroyEnd {
		t.Error("last use of owned local 'x' should have DestroyEnd=true")
	}
}

func TestNoDestroyFlagOnBorrowedParam(t *testing.T) {
	b, tt := makeBuilder()
	xUse := b.Ident(zs, "x", tt.I32)
	block := b.BlockExpr(zs, nil, xUse, tt.I32)
	fn := &hir.Function{
		Body:   block,
		Params: []hir.Param{{Span: zs, Name: "x", Type: tt.I32, Ownership: hir.OwnerRef}},
	}

	RunAll(fn)
	if xUse.Meta().DestroyEnd {
		t.Error("borrowed param 'x' should NOT have DestroyEnd=true")
	}
}

func TestDestroyWithMultipleLocals(t *testing.T) {
	b, tt := makeBuilder()
	letA := b.Let(zs, "a", tt.I32, b.Literal(zs, "1", tt.I32))
	letB := b.Let(zs, "b", tt.I32, b.Literal(zs, "2", tt.I32))
	aUse := b.Ident(zs, "a", tt.I32)
	bUse := b.Ident(zs, "b", tt.I32)
	add := b.Binary(zs, "+", aUse, bUse, tt.I32)
	block := b.BlockExpr(zs, []hir.Stmt{letA, letB}, add, tt.I32)
	fn := &hir.Function{Body: block}

	RunAll(fn)

	if !aUse.Meta().DestroyEnd {
		t.Error("last use of 'a' should have DestroyEnd")
	}
	if !bUse.Meta().DestroyEnd {
		t.Error("last use of 'b' should have DestroyEnd")
	}
}

func TestDestroyWithEarlyReturn(t *testing.T) {
	b, tt := makeBuilder()
	xUse := b.Ident(zs, "x", tt.I32)
	ret := b.Return(zs, xUse)
	block := b.BlockExpr(zs, []hir.Stmt{b.ExprStatement(zs, ret)}, nil, tt.Never)
	fn := &hir.Function{
		Body:   block,
		Params: []hir.Param{{Span: zs, Name: "x", Type: tt.I32, Ownership: hir.OwnerValue}},
	}

	RunAll(fn)

	if !xUse.Meta().DestroyEnd {
		t.Error("'x' used in return should have DestroyEnd")
	}
}

func TestDestroyWithLoop(t *testing.T) {
	b, tt := makeBuilder()
	xUse := b.Ident(zs, "x", tt.I32)
	brk := b.Break(zs, xUse)
	loopBody := b.BlockExpr(zs, []hir.Stmt{b.ExprStatement(zs, brk)}, nil, tt.Never)
	loop := b.Loop(zs, loopBody, tt.I32)
	block := b.BlockExpr(zs, nil, loop, tt.I32)
	fn := &hir.Function{
		Body:   block,
		Params: []hir.Param{{Span: zs, Name: "x", Type: tt.I32, Ownership: hir.OwnerValue}},
	}

	RunAll(fn)

	if !xUse.Meta().DestroyEnd {
		t.Error("'x' used in break inside loop should have DestroyEnd")
	}
}

// ===== RunAll integration =====

func TestRunAllPipeline(t *testing.T) {
	b, tt := makeBuilder()
	x := b.Ident(zs, "x", tt.I32)
	refX := b.Unary(zs, "ref", x, tt.InternRef(tt.I32))
	block := b.BlockExpr(zs, nil, refX, tt.InternRef(tt.I32))
	fn := &hir.Function{
		Body:   block,
		Params: []hir.Param{{Span: zs, Name: "x", Type: tt.I32, Ownership: hir.OwnerValue}},
	}

	result, errs := RunAll(fn)
	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Ownership should be set.
	if x.Meta().Ownership != hir.OwnerRef {
		t.Errorf("ownership: %s", x.Meta().Ownership)
	}
	// LiveAfter should be populated.
	if x.Meta().LiveAfter == nil {
		t.Error("LiveAfter should be set")
	}
	// DestroyEnd should be set for owned param.
	if !x.Meta().DestroyEnd {
		t.Error("DestroyEnd should be true for last use of owned param")
	}
}

func TestNilBodyDoesNotPanic(t *testing.T) {
	fn := &hir.Function{Body: nil}
	result, errs := RunAll(fn)
	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if result == nil {
		t.Fatal("expected non-nil result even for nil body")
	}
}
