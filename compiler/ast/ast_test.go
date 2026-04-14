package ast

import (
	"testing"

	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/lex"
)

var testSpan = diagnostics.Span{File: "test.fuse", Start: diagnostics.Pos{Line: 1, Col: 1}, End: diagnostics.Pos{Line: 1, Col: 10}}

// ===== Interface satisfaction =====
// Every concrete type must implement its expected marker interface.

func TestExprInterfaceSatisfaction(t *testing.T) {
	var exprs []Expr
	exprs = append(exprs,
		&LiteralExpr{Span: testSpan},
		&IdentExpr{Span: testSpan, Name: "x"},
		&UnaryExpr{Span: testSpan},
		&BinaryExpr{Span: testSpan},
		&AssignExpr{Span: testSpan},
		&CallExpr{Span: testSpan},
		&IndexExpr{Span: testSpan},
		&FieldExpr{Span: testSpan},
		&QDotExpr{Span: testSpan},
		&QuestionExpr{Span: testSpan},
		&BlockExpr{Span: testSpan},
		&IfExpr{Span: testSpan},
		&MatchExpr{Span: testSpan},
		&ForExpr{Span: testSpan},
		&WhileExpr{Span: testSpan},
		&LoopExpr{Span: testSpan},
		&TupleExpr{Span: testSpan},
		&StructLitExpr{Span: testSpan},
		&ClosureExpr{Span: testSpan},
		&SpawnExpr{Span: testSpan},
		&ReturnExpr{Span: testSpan},
		&BreakExpr{Span: testSpan},
		&ContinueExpr{Span: testSpan},
	)
	for _, e := range exprs {
		if e.NodeSpan() != testSpan {
			t.Errorf("%T: span mismatch", e)
		}
	}
}

func TestItemInterfaceSatisfaction(t *testing.T) {
	var items []Item
	items = append(items,
		&ImportDecl{Span: testSpan},
		&FnDecl{Span: testSpan},
		&StructDecl{Span: testSpan},
		&EnumDecl{Span: testSpan},
		&TraitDecl{Span: testSpan},
		&ImplDecl{Span: testSpan},
		&ConstDecl{Span: testSpan},
		&TypeAliasDecl{Span: testSpan},
		&ExternFnDecl{Span: testSpan},
	)
	for _, item := range items {
		if item.NodeSpan() != testSpan {
			t.Errorf("%T: span mismatch", item)
		}
	}
}

func TestStmtInterfaceSatisfaction(t *testing.T) {
	var stmts []Stmt
	stmts = append(stmts,
		&LetStmt{Span: testSpan},
		&VarStmt{Span: testSpan},
		&ExprStmt{Span: testSpan},
		&ItemStmt{Span: testSpan},
	)
	for _, s := range stmts {
		if s.NodeSpan() != testSpan {
			t.Errorf("%T: span mismatch", s)
		}
	}
}

func TestTypeExprInterfaceSatisfaction(t *testing.T) {
	var types []TypeExpr
	types = append(types,
		&PathType{Span: testSpan},
		&TupleType{Span: testSpan},
		&ArrayType{Span: testSpan},
		&SliceType{Span: testSpan},
		&PtrType{Span: testSpan},
	)
	for _, ty := range types {
		if ty.NodeSpan() != testSpan {
			t.Errorf("%T: span mismatch", ty)
		}
	}
}

func TestPatternInterfaceSatisfaction(t *testing.T) {
	var pats []Pattern
	pats = append(pats,
		&WildcardPat{Span: testSpan},
		&BindPat{Span: testSpan},
		&LitPat{Span: testSpan},
		&TuplePat{Span: testSpan},
		&ConstructorPat{Span: testSpan},
		&StructPat{Span: testSpan},
	)
	for _, p := range pats {
		if p.NodeSpan() != testSpan {
			t.Errorf("%T: span mismatch", p)
		}
	}
}

// ===== Span propagation =====

func TestFileNodeSpan(t *testing.T) {
	f := &File{Span: testSpan, Items: []Item{&FnDecl{Span: testSpan, Name: "main"}}}
	if f.NodeSpan() != testSpan {
		t.Error("File span mismatch")
	}
	if len(f.Items) != 1 {
		t.Error("File should have 1 item")
	}
}

// ===== Field and parameter construction =====

func TestFieldConstruction(t *testing.T) {
	f := Field{
		Span: testSpan,
		Name: "x",
		Type: &PathType{Span: testSpan, Segments: []string{"I32"}},
	}
	if f.Name != "x" {
		t.Error("field name")
	}
	if f.Type.NodeSpan() != testSpan {
		t.Error("field type span")
	}
}

func TestParamOwnershipKinds(t *testing.T) {
	cases := []struct {
		name      string
		ownership lex.TokenKind
	}{
		{"value", 0},
		{"ref", lex.KwRef},
		{"mutref", lex.KwMutref},
		{"owned", lex.KwOwned},
	}
	for _, tc := range cases {
		p := Param{Span: testSpan, Ownership: tc.ownership, Name: tc.name, Type: &PathType{Span: testSpan, Segments: []string{"I32"}}}
		if p.Ownership != tc.ownership {
			t.Errorf("%s: ownership %d, want %d", tc.name, p.Ownership, tc.ownership)
		}
	}
}

func TestGenericParamWithBounds(t *testing.T) {
	gp := GenericParam{
		Span:   testSpan,
		Name:   "T",
		Bounds: []TypeExpr{&PathType{Span: testSpan, Segments: []string{"Equatable"}}},
	}
	if gp.Name != "T" {
		t.Error("generic param name")
	}
	if len(gp.Bounds) != 1 {
		t.Error("generic param should have 1 bound")
	}
}

// ===== Variant kinds =====

func TestVariantKinds(t *testing.T) {
	unit := Variant{Span: testSpan, Name: "None", Kind: VariantUnit}
	tuple := Variant{Span: testSpan, Name: "Some", Kind: VariantTuple, Types: []TypeExpr{&PathType{Span: testSpan, Segments: []string{"T"}}}}
	struc := Variant{Span: testSpan, Name: "Named", Kind: VariantStruct, Fields: []Field{{Span: testSpan, Name: "val", Type: &PathType{Span: testSpan, Segments: []string{"I32"}}}}}

	if unit.Kind != VariantUnit {
		t.Error("unit variant kind")
	}
	if tuple.Kind != VariantTuple || len(tuple.Types) != 1 {
		t.Error("tuple variant")
	}
	if struc.Kind != VariantStruct || len(struc.Fields) != 1 {
		t.Error("struct variant")
	}
}

// ===== Expression node structure =====

func TestBinaryExprStructure(t *testing.T) {
	left := &LiteralExpr{Span: testSpan, Token: lex.Token{Kind: lex.IntLit, Literal: "1"}}
	right := &LiteralExpr{Span: testSpan, Token: lex.Token{Kind: lex.IntLit, Literal: "2"}}
	bin := &BinaryExpr{
		Span:  testSpan,
		Left:  left,
		Op:    lex.Token{Kind: lex.Plus, Literal: "+"},
		Right: right,
	}
	if bin.Left != left || bin.Right != right {
		t.Error("binary children")
	}
	if bin.Op.Literal != "+" {
		t.Error("binary op")
	}
}

func TestCallExprStructure(t *testing.T) {
	callee := &IdentExpr{Span: testSpan, Name: "foo"}
	arg := &LiteralExpr{Span: testSpan, Token: lex.Token{Kind: lex.IntLit, Literal: "42"}}
	call := &CallExpr{Span: testSpan, Callee: callee, Args: []Expr{arg}}
	if call.Callee.(*IdentExpr).Name != "foo" {
		t.Error("callee name")
	}
	if len(call.Args) != 1 {
		t.Error("call args count")
	}
}

func TestBlockExprTailMayBeNil(t *testing.T) {
	block := &BlockExpr{Span: testSpan, Stmts: nil, Tail: nil}
	if block.Tail != nil {
		t.Error("block tail should be nil")
	}
}

func TestBlockExprWithTail(t *testing.T) {
	tail := &LiteralExpr{Span: testSpan, Token: lex.Token{Kind: lex.IntLit, Literal: "1"}}
	block := &BlockExpr{Span: testSpan, Stmts: nil, Tail: tail}
	if block.Tail == nil {
		t.Error("block tail should not be nil")
	}
}

func TestIfExprElseChain(t *testing.T) {
	thenBlock := &BlockExpr{Span: testSpan}
	elseBlock := &BlockExpr{Span: testSpan}
	ifExpr := &IfExpr{
		Span: testSpan,
		Cond: &IdentExpr{Span: testSpan, Name: "cond"},
		Then: thenBlock,
		Else: elseBlock,
	}
	if ifExpr.Then != thenBlock {
		t.Error("then block")
	}
	if ifExpr.Else != elseBlock {
		t.Error("else block")
	}
}

func TestIfExprElseIfChain(t *testing.T) {
	inner := &IfExpr{Span: testSpan, Cond: &IdentExpr{Span: testSpan, Name: "b"}, Then: &BlockExpr{Span: testSpan}}
	outer := &IfExpr{
		Span: testSpan,
		Cond: &IdentExpr{Span: testSpan, Name: "a"},
		Then: &BlockExpr{Span: testSpan},
		Else: inner,
	}
	if _, ok := outer.Else.(*IfExpr); !ok {
		t.Error("else branch should be *IfExpr for else-if chain")
	}
}

func TestMatchExprArms(t *testing.T) {
	m := &MatchExpr{
		Span:    testSpan,
		Subject: &IdentExpr{Span: testSpan, Name: "x"},
		Arms: []MatchArm{
			{Span: testSpan, Pattern: &LitPat{Span: testSpan, Value: "1"}, Body: &LiteralExpr{Span: testSpan}},
			{Span: testSpan, Pattern: &WildcardPat{Span: testSpan}, Body: &LiteralExpr{Span: testSpan}},
		},
	}
	if len(m.Arms) != 2 {
		t.Error("match should have 2 arms")
	}
}

func TestMatchArmGuard(t *testing.T) {
	arm := MatchArm{
		Span:    testSpan,
		Pattern: &BindPat{Span: testSpan, Name: "n"},
		Guard:   &BinaryExpr{Span: testSpan, Left: &IdentExpr{Span: testSpan, Name: "n"}, Op: lex.Token{Kind: lex.Gt, Literal: ">"}, Right: &LiteralExpr{Span: testSpan}},
		Body:    &LiteralExpr{Span: testSpan},
	}
	if arm.Guard == nil {
		t.Error("match arm should have guard")
	}
}

func TestReturnExprValueMayBeNil(t *testing.T) {
	ret := &ReturnExpr{Span: testSpan, Value: nil}
	if ret.Value != nil {
		t.Error("return value should be nil")
	}
}

func TestClosureExprStructure(t *testing.T) {
	c := &ClosureExpr{
		Span:   testSpan,
		Params: []Param{{Span: testSpan, Name: "x", Type: &PathType{Span: testSpan, Segments: []string{"I32"}}}},
		Body:   &BlockExpr{Span: testSpan},
	}
	if len(c.Params) != 1 {
		t.Error("closure should have 1 param")
	}
	if c.ReturnType != nil {
		t.Error("closure return type should be nil when absent")
	}
}

// ===== Item structure =====

func TestImportDeclPath(t *testing.T) {
	imp := &ImportDecl{Span: testSpan, Path: []string{"core", "result", "Result"}, Alias: "Res"}
	if len(imp.Path) != 3 || imp.Alias != "Res" {
		t.Error("import path or alias")
	}
}

func TestFnDeclPublicAndGeneric(t *testing.T) {
	fn := &FnDecl{
		Span:          testSpan,
		Public:        true,
		Name:          "map",
		GenericParams: []GenericParam{{Span: testSpan, Name: "T"}, {Span: testSpan, Name: "U"}},
		Params:        []Param{{Span: testSpan, Name: "f"}},
		ReturnType:    &PathType{Span: testSpan, Segments: []string{"U"}},
		Body:          &BlockExpr{Span: testSpan},
	}
	if !fn.Public {
		t.Error("fn should be public")
	}
	if len(fn.GenericParams) != 2 {
		t.Error("fn should have 2 generic params")
	}
}

func TestStructDeclDecorators(t *testing.T) {
	s := &StructDecl{
		Span:       testSpan,
		Name:       "Point",
		Decorators: []Decorator{{Span: testSpan, Name: "value"}},
		Fields:     []Field{{Span: testSpan, Name: "x"}, {Span: testSpan, Name: "y"}},
	}
	if len(s.Decorators) != 1 || s.Decorators[0].Name != "value" {
		t.Error("struct decorators")
	}
	if len(s.Fields) != 2 {
		t.Error("struct should have 2 fields")
	}
}

func TestEnumDeclVariants(t *testing.T) {
	e := &EnumDecl{
		Span: testSpan,
		Name: "Option",
		GenericParams: []GenericParam{{Span: testSpan, Name: "T"}},
		Variants: []Variant{
			{Span: testSpan, Name: "Some", Kind: VariantTuple, Types: []TypeExpr{&PathType{Span: testSpan, Segments: []string{"T"}}}},
			{Span: testSpan, Name: "None", Kind: VariantUnit},
		},
	}
	if len(e.Variants) != 2 {
		t.Error("enum should have 2 variants")
	}
}

func TestTraitDeclSupertraits(t *testing.T) {
	tr := &TraitDecl{
		Span:        testSpan,
		Name:        "Comparable",
		Supertraits: []TypeExpr{&PathType{Span: testSpan, Segments: []string{"Equatable"}}},
	}
	if len(tr.Supertraits) != 1 {
		t.Error("trait should have 1 supertrait")
	}
}

func TestImplDeclTraitImpl(t *testing.T) {
	impl := &ImplDecl{
		Span:   testSpan,
		Target: &PathType{Span: testSpan, Segments: []string{"Point"}},
		Trait:  &PathType{Span: testSpan, Segments: []string{"Display"}},
		Items:  []Item{&FnDecl{Span: testSpan, Name: "fmt"}},
	}
	if impl.Trait == nil {
		t.Error("impl should have trait")
	}
}

func TestImplDeclInherentImpl(t *testing.T) {
	impl := &ImplDecl{
		Span:   testSpan,
		Target: &PathType{Span: testSpan, Segments: []string{"Point"}},
		Trait:  nil,
	}
	if impl.Trait != nil {
		t.Error("inherent impl should have nil trait")
	}
}

func TestExternFnDecl(t *testing.T) {
	ext := &ExternFnDecl{
		Span:       testSpan,
		Public:     true,
		Name:       "fuse_rt_panic",
		Params:     []Param{{Span: testSpan, Name: "msg"}},
		ReturnType: nil,
	}
	if ext.ReturnType != nil {
		t.Error("extern fn with void return should have nil return type")
	}
}

// ===== Statement structure =====

func TestLetStmtOptionalTypeAndValue(t *testing.T) {
	// let x: I32 = 5
	full := &LetStmt{
		Span:  testSpan,
		Name:  "x",
		Type:  &PathType{Span: testSpan, Segments: []string{"I32"}},
		Value: &LiteralExpr{Span: testSpan},
	}
	if full.Type == nil || full.Value == nil {
		t.Error("full let should have type and value")
	}

	// let x
	bare := &LetStmt{Span: testSpan, Name: "x"}
	if bare.Type != nil || bare.Value != nil {
		t.Error("bare let should have nil type and value")
	}
}

// ===== Type expression structure =====

func TestPathTypeGeneric(t *testing.T) {
	pt := &PathType{
		Span:     testSpan,
		Segments: []string{"core", "option", "Option"},
		TypeArgs: []TypeExpr{&PathType{Span: testSpan, Segments: []string{"I32"}}},
	}
	if len(pt.Segments) != 3 {
		t.Error("path segments")
	}
	if len(pt.TypeArgs) != 1 {
		t.Error("type args")
	}
}

func TestArrayTypeStructure(t *testing.T) {
	arr := &ArrayType{
		Span: testSpan,
		Elem: &PathType{Span: testSpan, Segments: []string{"U8"}},
		Size: &LiteralExpr{Span: testSpan, Token: lex.Token{Kind: lex.IntLit, Literal: "256"}},
	}
	if arr.Elem == nil || arr.Size == nil {
		t.Error("array type should have elem and size")
	}
}

// ===== Pattern structure =====

func TestConstructorPatStructure(t *testing.T) {
	pat := &ConstructorPat{
		Span: testSpan,
		Name: "Some",
		Args: []Pattern{&BindPat{Span: testSpan, Name: "x"}},
	}
	if pat.Name != "Some" || len(pat.Args) != 1 {
		t.Error("constructor pattern")
	}
}

func TestStructPatFieldShorthand(t *testing.T) {
	pat := &StructPat{
		Span: testSpan,
		Name: "Point",
		Fields: []FieldPat{
			{Span: testSpan, Name: "x", Pat: nil},         // shorthand
			{Span: testSpan, Name: "y", Pat: &BindPat{Span: testSpan, Name: "yy"}}, // explicit
		},
	}
	if pat.Fields[0].Pat != nil {
		t.Error("shorthand field should have nil pat")
	}
	if pat.Fields[1].Pat == nil {
		t.Error("explicit field should have non-nil pat")
	}
}

// ===== WhereClause =====

func TestWhereClauseStructure(t *testing.T) {
	wc := &WhereClause{
		Span: testSpan,
		Constraints: []WhereConstraint{
			{
				Span:   testSpan,
				Type:   &PathType{Span: testSpan, Segments: []string{"T"}},
				Bounds: []TypeExpr{&PathType{Span: testSpan, Segments: []string{"Display"}}},
			},
		},
	}
	if len(wc.Constraints) != 1 {
		t.Error("where clause should have 1 constraint")
	}
	if len(wc.Constraints[0].Bounds) != 1 {
		t.Error("constraint should have 1 bound")
	}
}

// ===== Decorator with args =====

func TestDecoratorWithArgs(t *testing.T) {
	d := Decorator{
		Span: testSpan,
		Name: "rank",
		Args: []Expr{&LiteralExpr{Span: testSpan, Token: lex.Token{Kind: lex.IntLit, Literal: "3"}}},
	}
	if d.Name != "rank" || len(d.Args) != 1 {
		t.Error("decorator with args")
	}
}

func TestDecoratorWithoutArgs(t *testing.T) {
	d := Decorator{Span: testSpan, Name: "value", Args: nil}
	if d.Args != nil {
		t.Error("decorator without args should have nil Args")
	}
}
