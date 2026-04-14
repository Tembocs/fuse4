package parse

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Tembocs/fuse4/compiler/ast"
)

// --- helpers ---

func parseOK(t *testing.T, src string) *ast.File {
	t.Helper()
	file, errs := Parse("test.fuse", []byte(src))
	for _, e := range errs {
		t.Errorf("unexpected diagnostic: %s", e)
	}
	return file
}

func parseWithErrors(src string) (*ast.File, int) {
	file, errs := Parse("test.fuse", []byte(src))
	return file, len(errs)
}

// ===== Wave 02 Phase 01: AST is syntax-only =====

func TestEmptyFile(t *testing.T) {
	f := parseOK(t, "")
	if len(f.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(f.Items))
	}
}

// ===== Wave 02 Phase 02 Task 01: Items and declarations =====

func TestImportSimple(t *testing.T) {
	f := parseOK(t, `import core.result.Result;`)
	if len(f.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(f.Items))
	}
	imp := f.Items[0].(*ast.ImportDecl)
	if len(imp.Path) != 3 || imp.Path[0] != "core" || imp.Path[2] != "Result" {
		t.Errorf("path: %v", imp.Path)
	}
}

func TestImportAlias(t *testing.T) {
	f := parseOK(t, `import full.chan.Chan as Ch;`)
	imp := f.Items[0].(*ast.ImportDecl)
	if imp.Alias != "Ch" {
		t.Errorf("alias: %q", imp.Alias)
	}
}

func TestFnDecl(t *testing.T) {
	f := parseOK(t, `fn foo() { }`)
	fn := f.Items[0].(*ast.FnDecl)
	if fn.Name != "foo" || fn.Public {
		t.Errorf("fn name=%q pub=%v", fn.Name, fn.Public)
	}
}

func TestFnDeclPubWithReturn(t *testing.T) {
	f := parseOK(t, `pub fn bar(x: Int) -> Bool { true }`)
	fn := f.Items[0].(*ast.FnDecl)
	if !fn.Public || fn.Name != "bar" {
		t.Errorf("fn pub=%v name=%q", fn.Public, fn.Name)
	}
	if len(fn.Params) != 1 {
		t.Fatalf("params: %d", len(fn.Params))
	}
	if fn.ReturnType == nil {
		t.Error("expected return type")
	}
}

func TestFnDeclOwnershipParams(t *testing.T) {
	f := parseOK(t, `fn process(queue: mutref Chan, data: ref String, sink: owned File) { }`)
	fn := f.Items[0].(*ast.FnDecl)
	if len(fn.Params) != 3 {
		t.Fatalf("params: %d", len(fn.Params))
	}
}

func TestFnDeclGeneric(t *testing.T) {
	f := parseOK(t, `fn map[T, U](items: List, f: Fn) -> List { }`)
	fn := f.Items[0].(*ast.FnDecl)
	if len(fn.GenericParams) != 2 {
		t.Errorf("generic params: %d", len(fn.GenericParams))
	}
}

func TestStructDecl(t *testing.T) {
	f := parseOK(t, `struct Point { x: F64, y: F64 }`)
	s := f.Items[0].(*ast.StructDecl)
	if s.Name != "Point" || len(s.Fields) != 2 {
		t.Errorf("struct name=%q fields=%d", s.Name, len(s.Fields))
	}
}

func TestStructDeclDecorator(t *testing.T) {
	f := parseOK(t, `@value struct WorkItem { id: U64, payload: String }`)
	s := f.Items[0].(*ast.StructDecl)
	if len(s.Decorators) != 1 || s.Decorators[0].Name != "value" {
		t.Errorf("decorators: %v", s.Decorators)
	}
}

func TestStructDeclGeneric(t *testing.T) {
	f := parseOK(t, `pub struct Pair[A, B] { first: A, second: B }`)
	s := f.Items[0].(*ast.StructDecl)
	if !s.Public || len(s.GenericParams) != 2 {
		t.Errorf("pub=%v generics=%d", s.Public, len(s.GenericParams))
	}
}

func TestEnumDecl(t *testing.T) {
	f := parseOK(t, `enum Color { Red, Green, Blue }`)
	e := f.Items[0].(*ast.EnumDecl)
	if e.Name != "Color" || len(e.Variants) != 3 {
		t.Errorf("enum name=%q variants=%d", e.Name, len(e.Variants))
	}
	for _, v := range e.Variants {
		if v.Kind != ast.VariantUnit {
			t.Errorf("variant %q kind=%d, want unit", v.Name, v.Kind)
		}
	}
}

func TestEnumTupleVariants(t *testing.T) {
	f := parseOK(t, `enum Shape { Circle(F64), Rect(F64, F64) }`)
	e := f.Items[0].(*ast.EnumDecl)
	if e.Variants[0].Kind != ast.VariantTuple || len(e.Variants[0].Types) != 1 {
		t.Error("Circle should be tuple(1)")
	}
	if e.Variants[1].Kind != ast.VariantTuple || len(e.Variants[1].Types) != 2 {
		t.Error("Rect should be tuple(2)")
	}
}

func TestEnumStructVariant(t *testing.T) {
	f := parseOK(t, `enum Msg { Quit, Data { payload: String, len: U64 } }`)
	e := f.Items[0].(*ast.EnumDecl)
	if e.Variants[0].Kind != ast.VariantUnit {
		t.Error("Quit should be unit")
	}
	if e.Variants[1].Kind != ast.VariantStruct || len(e.Variants[1].Fields) != 2 {
		t.Error("Data should be struct(2)")
	}
}

func TestEnumGeneric(t *testing.T) {
	f := parseOK(t, `enum Option[T] { Some(T), None }`)
	e := f.Items[0].(*ast.EnumDecl)
	if len(e.GenericParams) != 1 {
		t.Errorf("generics: %d", len(e.GenericParams))
	}
}

func TestTraitDecl(t *testing.T) {
	f := parseOK(t, `trait Display { fn fmt(ref self, f: mutref Formatter); }`)
	tr := f.Items[0].(*ast.TraitDecl)
	if tr.Name != "Display" || len(tr.Items) != 1 {
		t.Errorf("trait name=%q items=%d", tr.Name, len(tr.Items))
	}
}

func TestTraitWithSupertrait(t *testing.T) {
	f := parseOK(t, `trait Hashable : Equatable { fn hash(ref self) -> U64; }`)
	tr := f.Items[0].(*ast.TraitDecl)
	if len(tr.Supertraits) != 1 {
		t.Errorf("supertraits: %d", len(tr.Supertraits))
	}
}

func TestImplDecl(t *testing.T) {
	f := parseOK(t, `impl Point { pub fn new(x: F64, y: F64) -> Point { Point { x: x, y: y } } }`)
	impl := f.Items[0].(*ast.ImplDecl)
	if impl.Trait != nil {
		t.Error("expected inherent impl")
	}
	if len(impl.Items) != 1 {
		t.Errorf("items: %d", len(impl.Items))
	}
}

func TestImplTraitForType(t *testing.T) {
	f := parseOK(t, `impl Display : Point { fn fmt(ref self, f: mutref Formatter) { } }`)
	impl := f.Items[0].(*ast.ImplDecl)
	if impl.Trait == nil {
		t.Error("expected trait impl")
	}
}

func TestConstDecl(t *testing.T) {
	f := parseOK(t, `const MAX: I32 = 100;`)
	c := f.Items[0].(*ast.ConstDecl)
	if c.Name != "MAX" {
		t.Errorf("name: %q", c.Name)
	}
}

func TestTypeAlias(t *testing.T) {
	f := parseOK(t, `type Pair = (Int, Int);`)
	ta := f.Items[0].(*ast.TypeAliasDecl)
	if ta.Name != "Pair" {
		t.Errorf("name: %q", ta.Name)
	}
}

func TestExternFn(t *testing.T) {
	f := parseOK(t, `extern fn printf(fmt: Ptr[U8]) -> I32;`)
	ext := f.Items[0].(*ast.ExternFnDecl)
	if ext.Name != "printf" {
		t.Errorf("name: %q", ext.Name)
	}
}

// ===== Wave 02 Phase 02 Task 02: Expressions with precedence =====

func firstExpr(t *testing.T, src string) ast.Expr {
	t.Helper()
	f := parseOK(t, "fn test() { "+src+"; }")
	fn := f.Items[0].(*ast.FnDecl)
	if len(fn.Body.Stmts) == 0 {
		t.Fatal("no statements in body")
	}
	es := fn.Body.Stmts[0].(*ast.ExprStmt)
	return es.Expr
}

func TestLiterals(t *testing.T) {
	cases := []string{"42", "3.14", `"hello"`, `r"raw"`, "true", "false"}
	for _, c := range cases {
		t.Run(c, func(t *testing.T) {
			e := firstExpr(t, c)
			if _, ok := e.(*ast.LiteralExpr); !ok {
				t.Errorf("expected LiteralExpr, got %T", e)
			}
		})
	}
}

func TestBinaryPrecedence(t *testing.T) {
	// a + b * c  → Add(a, Mul(b, c))
	e := firstExpr(t, "a + b * c")
	bin := e.(*ast.BinaryExpr)
	if bin.Op.Literal != "+" {
		t.Errorf("top op: %q", bin.Op.Literal)
	}
	right := bin.Right.(*ast.BinaryExpr)
	if right.Op.Literal != "*" {
		t.Errorf("right op: %q", right.Op.Literal)
	}
}

func TestBinaryLeftAssoc(t *testing.T) {
	// a - b - c  → Sub(Sub(a, b), c)
	e := firstExpr(t, "a - b - c")
	bin := e.(*ast.BinaryExpr)
	if bin.Op.Literal != "-" {
		t.Errorf("top op: %q", bin.Op.Literal)
	}
	left := bin.Left.(*ast.BinaryExpr)
	if left.Op.Literal != "-" {
		t.Errorf("left op: %q", left.Op.Literal)
	}
}

func TestAssignRightAssoc(t *testing.T) {
	// a = b = c  → Assign(a, Assign(b, c))
	e := firstExpr(t, "a = b = c")
	assign := e.(*ast.AssignExpr)
	if _, ok := assign.Value.(*ast.AssignExpr); !ok {
		t.Errorf("right of assign should be AssignExpr, got %T", assign.Value)
	}
}

func TestCompoundAssign(t *testing.T) {
	ops := []string{"+=", "-=", "*=", "/=", "%="}
	for _, op := range ops {
		t.Run(op, func(t *testing.T) {
			e := firstExpr(t, "x "+op+" 1")
			assign := e.(*ast.AssignExpr)
			if assign.Op.Literal != op {
				t.Errorf("op: %q", assign.Op.Literal)
			}
		})
	}
}

func TestLogicalPrecedence(t *testing.T) {
	// a || b && c  → Or(a, And(b, c))
	e := firstExpr(t, "a || b && c")
	bin := e.(*ast.BinaryExpr)
	if bin.Op.Literal != "||" {
		t.Errorf("top: %q", bin.Op.Literal)
	}
	right := bin.Right.(*ast.BinaryExpr)
	if right.Op.Literal != "&&" {
		t.Errorf("right: %q", right.Op.Literal)
	}
}

func TestComparisonPrecedence(t *testing.T) {
	// a + 1 == b * 2
	e := firstExpr(t, "a + 1 == b * 2")
	bin := e.(*ast.BinaryExpr)
	if bin.Op.Literal != "==" {
		t.Errorf("top: %q", bin.Op.Literal)
	}
}

func TestUnaryPrefix(t *testing.T) {
	cases := []struct {
		src string
		op  string
	}{
		{"!x", "!"},
		{"-x", "-"},
		{"~x", "~"},
		{"ref x", "ref"},
		{"mutref x", "mutref"},
		{"move x", "move"},
	}
	for _, tc := range cases {
		t.Run(tc.op, func(t *testing.T) {
			e := firstExpr(t, tc.src)
			u := e.(*ast.UnaryExpr)
			if u.Op.Literal != tc.op {
				t.Errorf("op: %q", u.Op.Literal)
			}
		})
	}
}

func TestCallExpr(t *testing.T) {
	e := firstExpr(t, "foo(1, 2, 3)")
	call := e.(*ast.CallExpr)
	if len(call.Args) != 3 {
		t.Errorf("args: %d", len(call.Args))
	}
}

func TestChainedMethodCalls(t *testing.T) {
	e := firstExpr(t, "a.b().c()")
	// Should be Call(Field(Call(Field(a, b), []), c), [])
	outer := e.(*ast.CallExpr)
	outerField := outer.Callee.(*ast.FieldExpr)
	if outerField.Name != "c" {
		t.Errorf("outer field: %q", outerField.Name)
	}
	inner := outerField.Expr.(*ast.CallExpr)
	innerField := inner.Callee.(*ast.FieldExpr)
	if innerField.Name != "b" {
		t.Errorf("inner field: %q", innerField.Name)
	}
}

func TestIndexExpr(t *testing.T) {
	e := firstExpr(t, "arr[0]")
	idx := e.(*ast.IndexExpr)
	_ = idx
}

func TestFieldAccess(t *testing.T) {
	e := firstExpr(t, "obj.field")
	f := e.(*ast.FieldExpr)
	if f.Name != "field" {
		t.Errorf("field: %q", f.Name)
	}
}

func TestTupleFieldAccess(t *testing.T) {
	e := firstExpr(t, "pair.0")
	f := e.(*ast.FieldExpr)
	if f.Name != "0" {
		t.Errorf("field: %q", f.Name)
	}
}

func TestPostfixQuestion(t *testing.T) {
	e := firstExpr(t, "result?")
	q := e.(*ast.QuestionExpr)
	_ = q
}

func TestIfExpr(t *testing.T) {
	e := firstExpr(t, "if x { 1 } else { 2 }")
	ie := e.(*ast.IfExpr)
	if ie.Else == nil {
		t.Error("expected else branch")
	}
}

func TestIfElseIfChain(t *testing.T) {
	e := firstExpr(t, "if a { 1 } else if b { 2 } else { 3 }")
	ie := e.(*ast.IfExpr)
	elseIf := ie.Else.(*ast.IfExpr)
	if elseIf.Else == nil {
		t.Error("expected final else branch")
	}
}

func TestMatchExpr(t *testing.T) {
	src := `match x { Some(v) => v, None => 0 }`
	e := firstExpr(t, src)
	m := e.(*ast.MatchExpr)
	if len(m.Arms) != 2 {
		t.Errorf("arms: %d", len(m.Arms))
	}
	cp := m.Arms[0].Pattern.(*ast.ConstructorPat)
	if cp.Name != "Some" || len(cp.Args) != 1 {
		t.Errorf("arm[0] pattern: %q args=%d", cp.Name, len(cp.Args))
	}
}

func TestMatchWildcard(t *testing.T) {
	src := `match x { _ => 0 }`
	e := firstExpr(t, src)
	m := e.(*ast.MatchExpr)
	if _, ok := m.Arms[0].Pattern.(*ast.WildcardPat); !ok {
		t.Errorf("expected WildcardPat, got %T", m.Arms[0].Pattern)
	}
}

func TestForExpr(t *testing.T) {
	src := `for item in items { process(item); }`
	e := firstExpr(t, src)
	f := e.(*ast.ForExpr)
	if f.Binding != "item" {
		t.Errorf("binding: %q", f.Binding)
	}
}

func TestWhileExpr(t *testing.T) {
	e := firstExpr(t, `while running { tick(); }`)
	w := e.(*ast.WhileExpr)
	_ = w
}

func TestLoopExpr(t *testing.T) {
	e := firstExpr(t, `loop { break; }`)
	l := e.(*ast.LoopExpr)
	_ = l
}

func TestReturnExpr(t *testing.T) {
	e := firstExpr(t, `return 42`)
	r := e.(*ast.ReturnExpr)
	if r.Value == nil {
		t.Error("expected return value")
	}
}

func TestReturnVoid(t *testing.T) {
	// return followed by ; → nil value
	f := parseOK(t, "fn test() { return; }")
	fn := f.Items[0].(*ast.FnDecl)
	es := fn.Body.Stmts[0].(*ast.ExprStmt)
	r := es.Expr.(*ast.ReturnExpr)
	if r.Value != nil {
		t.Error("expected nil return value")
	}
}

func TestBreakContinue(t *testing.T) {
	f := parseOK(t, "fn test() { loop { break; continue; }; }")
	fn := f.Items[0].(*ast.FnDecl)
	block := fn.Body.Stmts[0].(*ast.ExprStmt).Expr.(*ast.LoopExpr)
	stmts := block.Body.Stmts
	if len(stmts) < 2 {
		t.Fatalf("loop stmts: %d", len(stmts))
	}
}

func TestSpawnExpr(t *testing.T) {
	e := firstExpr(t, `spawn fn() { }`)
	s := e.(*ast.SpawnExpr)
	if _, ok := s.Expr.(*ast.ClosureExpr); !ok {
		t.Errorf("spawn body: %T", s.Expr)
	}
}

func TestClosureExpr(t *testing.T) {
	e := firstExpr(t, `fn(x: Int) -> Int { x + 1 }`)
	c := e.(*ast.ClosureExpr)
	if len(c.Params) != 1 {
		t.Errorf("params: %d", len(c.Params))
	}
	if c.ReturnType == nil {
		t.Error("expected return type")
	}
}

func TestTupleExpr(t *testing.T) {
	e := firstExpr(t, "(1, 2, 3)")
	tuple := e.(*ast.TupleExpr)
	if len(tuple.Elems) != 3 {
		t.Errorf("elems: %d", len(tuple.Elems))
	}
}

func TestUnitExpr(t *testing.T) {
	e := firstExpr(t, "()")
	tuple := e.(*ast.TupleExpr)
	if len(tuple.Elems) != 0 {
		t.Errorf("elems: %d, want 0", len(tuple.Elems))
	}
}

func TestGroupedExpr(t *testing.T) {
	// (a + b) * c  → Mul(Add(a, b), c)
	e := firstExpr(t, "(a + b) * c")
	bin := e.(*ast.BinaryExpr)
	if bin.Op.Literal != "*" {
		t.Errorf("top: %q", bin.Op.Literal)
	}
	inner := bin.Left.(*ast.BinaryExpr)
	if inner.Op.Literal != "+" {
		t.Errorf("grouped: %q", inner.Op.Literal)
	}
}

func TestBlockExprTail(t *testing.T) {
	f := parseOK(t, "fn test() -> Int { let x = 1; x + 1 }")
	fn := f.Items[0].(*ast.FnDecl)
	if fn.Body.Tail == nil {
		t.Error("expected tail expression")
	}
}

// ===== Wave 02 Phase 02 Task 03: Type expressions =====

func TestPathType(t *testing.T) {
	f := parseOK(t, "fn test(x: Int) { }")
	fn := f.Items[0].(*ast.FnDecl)
	pt := fn.Params[0].Type.(*ast.PathType)
	if len(pt.Segments) != 1 || pt.Segments[0] != "Int" {
		t.Errorf("type: %v", pt.Segments)
	}
}

func TestGenericType(t *testing.T) {
	f := parseOK(t, "fn test(x: List[Int]) { }")
	fn := f.Items[0].(*ast.FnDecl)
	pt := fn.Params[0].Type.(*ast.PathType)
	if pt.Segments[0] != "List" || len(pt.TypeArgs) != 1 {
		t.Errorf("type: %v args=%d", pt.Segments, len(pt.TypeArgs))
	}
}

func TestNestedGenericType(t *testing.T) {
	f := parseOK(t, "fn test(x: Result[Option[Int], String]) { }")
	fn := f.Items[0].(*ast.FnDecl)
	pt := fn.Params[0].Type.(*ast.PathType)
	if pt.Segments[0] != "Result" || len(pt.TypeArgs) != 2 {
		t.Errorf("type: %v args=%d", pt.Segments, len(pt.TypeArgs))
	}
}

func TestUnitType(t *testing.T) {
	f := parseOK(t, "fn test() -> () { }")
	fn := f.Items[0].(*ast.FnDecl)
	tt := fn.ReturnType.(*ast.TupleType)
	if len(tt.Elems) != 0 {
		t.Error("expected unit type")
	}
}

func TestTupleType(t *testing.T) {
	f := parseOK(t, "fn test() -> (Int, Bool) { }")
	fn := f.Items[0].(*ast.FnDecl)
	tt := fn.ReturnType.(*ast.TupleType)
	if len(tt.Elems) != 2 {
		t.Errorf("tuple elems: %d", len(tt.Elems))
	}
}

func TestSliceType(t *testing.T) {
	f := parseOK(t, "fn test(x: [U8]) { }")
	fn := f.Items[0].(*ast.FnDecl)
	st := fn.Params[0].Type.(*ast.SliceType)
	_ = st
}

func TestArrayType(t *testing.T) {
	f := parseOK(t, "fn test(x: [U8; 256]) { }")
	fn := f.Items[0].(*ast.FnDecl)
	at := fn.Params[0].Type.(*ast.ArrayType)
	_ = at
}

func TestPtrType(t *testing.T) {
	f := parseOK(t, "fn test(x: Ptr[U8]) { }")
	fn := f.Items[0].(*ast.FnDecl)
	pt := fn.Params[0].Type.(*ast.PtrType)
	_ = pt
}

func TestQualifiedPathType(t *testing.T) {
	f := parseOK(t, "fn test(x: core.list.List[Int]) { }")
	fn := f.Items[0].(*ast.FnDecl)
	pt := fn.Params[0].Type.(*ast.PathType)
	if len(pt.Segments) != 3 || pt.Segments[0] != "core" {
		t.Errorf("segments: %v", pt.Segments)
	}
}

// ===== Wave 02 Phase 03: Ambiguity control =====

// Struct literal disambiguation: IDENT { field: value }
func TestStructLiteral(t *testing.T) {
	e := firstExpr(t, `Point { x: 1, y: 2 }`)
	sl := e.(*ast.StructLitExpr)
	if sl.Name != "Point" || len(sl.Fields) != 2 {
		t.Errorf("struct lit: name=%q fields=%d", sl.Name, len(sl.Fields))
	}
}

func TestEmptyStructLiteral(t *testing.T) {
	e := firstExpr(t, `Empty {}`)
	sl := e.(*ast.StructLitExpr)
	if sl.Name != "Empty" || len(sl.Fields) != 0 {
		t.Errorf("empty struct lit: name=%q fields=%d", sl.Name, len(sl.Fields))
	}
}

// IDENT { expr } → NOT a struct literal (no colon after first ident in braces)
func TestIdentFollowedByBlockIsNotStructLit(t *testing.T) {
	src := `fn test() { x { y; }; }`
	f := parseOK(t, src)
	fn := f.Items[0].(*ast.FnDecl)
	// First stmt should be the identifier 'x' as ExprStmt
	if len(fn.Body.Stmts) < 1 {
		t.Fatalf("stmts: %d", len(fn.Body.Stmts))
	}
	es := fn.Body.Stmts[0].(*ast.ExprStmt)
	if _, ok := es.Expr.(*ast.IdentExpr); !ok {
		t.Errorf("expected IdentExpr for 'x', got %T", es.Expr)
	}
}

// Optional chaining: expr?.field
func TestOptionalChaining(t *testing.T) {
	e := firstExpr(t, "x?.y?.z")
	outer := e.(*ast.QDotExpr)
	if outer.Name != "z" {
		t.Errorf("outer: %q", outer.Name)
	}
	inner := outer.Expr.(*ast.QDotExpr)
	if inner.Name != "y" {
		t.Errorf("inner: %q", inner.Name)
	}
}

func TestOptionalChainingWithCall(t *testing.T) {
	e := firstExpr(t, "obj?.method()")
	call := e.(*ast.CallExpr)
	qdot := call.Callee.(*ast.QDotExpr)
	if qdot.Name != "method" {
		t.Errorf("qdot name: %q", qdot.Name)
	}
}

func TestQuestionThenSemicolon(t *testing.T) {
	// x? should parse as QuestionExpr, not QDotExpr
	e := firstExpr(t, "x?")
	if _, ok := e.(*ast.QuestionExpr); !ok {
		t.Errorf("expected QuestionExpr, got %T", e)
	}
}

// ===== Error recovery =====

func TestMalformedInputDoesNotPanic(t *testing.T) {
	inputs := []string{
		"fn",
		"fn (",
		"struct {",
		"import ;",
		"fn test() { let = ; }",
		"fn test() { 1 + ; }",
		"}}}",
		"",
		"@",
		"fn test() { if { } }",
	}
	for _, src := range inputs {
		t.Run(src, func(t *testing.T) {
			// Must not panic
			_, _ = parseWithErrors(src)
		})
	}
}

// ===== Comprehensive program (golden-style sanity) =====

func TestFullProgram(t *testing.T) {
	src := `
import core.result.Result;
import full.chan.Chan;

@value struct WorkItem {
	id: U64,
	payload: String,
}

pub fn process(queue: mutref Chan[WorkItem]) -> Result[(), String] {
	let item = queue.recv()?;
	let upper = item.payload.toUpper();
	spawn fn() {
		log(upper);
	};
	return Ok(());
}
`
	f := parseOK(t, src)
	if len(f.Items) != 3 { // 2 imports + 1 struct + 1 fn... wait, 4
		// Actually: import, import, struct, fn = 4
		if len(f.Items) != 4 {
			t.Errorf("items: %d, want 4", len(f.Items))
		}
	}
}

// ===== Golden test harness =====

func printAST(node ast.Node, indent int) string {
	pad := strings.Repeat("  ", indent)
	var b strings.Builder

	switch n := node.(type) {
	case *ast.File:
		b.WriteString("(File\n")
		for _, item := range n.Items {
			b.WriteString(printAST(item, indent+1))
		}
		b.WriteString(")\n")

	case *ast.ImportDecl:
		b.WriteString(pad + "(Import " + strings.Join(n.Path, "."))
		if n.Alias != "" {
			b.WriteString(" as " + n.Alias)
		}
		b.WriteString(")\n")

	case *ast.FnDecl:
		pub := ""
		if n.Public {
			pub = "pub "
		}
		b.WriteString(pad + "(" + pub + "Fn " + n.Name)
		if len(n.GenericParams) > 0 {
			b.WriteString("[")
			for i, gp := range n.GenericParams {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(gp.Name)
			}
			b.WriteString("]")
		}
		b.WriteString("(")
		for i, p := range n.Params {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(p.Name)
		}
		b.WriteString(")")
		if n.ReturnType != nil {
			b.WriteString(" -> " + printTypeExpr(n.ReturnType))
		}
		b.WriteString(")\n")

	case *ast.StructDecl:
		pub := ""
		if n.Public {
			pub = "pub "
		}
		decs := ""
		for _, d := range n.Decorators {
			decs += "@" + d.Name + " "
		}
		b.WriteString(pad + "(" + decs + pub + "Struct " + n.Name)
		for _, f := range n.Fields {
			b.WriteString(" " + f.Name)
		}
		b.WriteString(")\n")

	case *ast.EnumDecl:
		b.WriteString(pad + "(Enum " + n.Name)
		for _, v := range n.Variants {
			b.WriteString(" " + v.Name)
		}
		b.WriteString(")\n")

	case *ast.TraitDecl:
		b.WriteString(pad + "(Trait " + n.Name + ")\n")

	case *ast.ImplDecl:
		b.WriteString(pad + "(Impl " + printTypeExpr(n.Target) + ")\n")

	case *ast.ConstDecl:
		b.WriteString(pad + "(Const " + n.Name + ")\n")

	case *ast.TypeAliasDecl:
		b.WriteString(pad + "(TypeAlias " + n.Name + ")\n")

	case *ast.ExternFnDecl:
		b.WriteString(pad + "(Extern " + n.Name + ")\n")

	default:
		b.WriteString(pad + "(unknown)\n")
	}

	return b.String()
}

func printTypeExpr(te ast.TypeExpr) string {
	switch t := te.(type) {
	case *ast.PathType:
		s := strings.Join(t.Segments, ".")
		if len(t.TypeArgs) > 0 {
			s += "["
			for i, a := range t.TypeArgs {
				if i > 0 {
					s += ", "
				}
				s += printTypeExpr(a)
			}
			s += "]"
		}
		return s
	case *ast.TupleType:
		if len(t.Elems) == 0 {
			return "()"
		}
		parts := make([]string, len(t.Elems))
		for i, e := range t.Elems {
			parts[i] = printTypeExpr(e)
		}
		return "(" + strings.Join(parts, ", ") + ")"
	case *ast.SliceType:
		return "[" + printTypeExpr(t.Elem) + "]"
	case *ast.ArrayType:
		return "[" + printTypeExpr(t.Elem) + "; N]"
	case *ast.PtrType:
		return "Ptr[" + printTypeExpr(t.Elem) + "]"
	default:
		return "<type>"
	}
}

func TestGoldenParser(t *testing.T) {
	fixtureDir := filepath.Join("..", "..", "tests", "fixtures", "parser")
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		t.Fatalf("cannot read fixture dir: %v", err)
	}

	update := os.Getenv("UPDATE_GOLDENS") != ""

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".fuse") {
			continue
		}
		base := strings.TrimSuffix(name, ".fuse")
		t.Run(base, func(t *testing.T) {
			srcPath := filepath.Join(fixtureDir, name)
			goldenPath := filepath.Join(fixtureDir, base+".golden")

			src, err := os.ReadFile(srcPath)
			if err != nil {
				t.Fatalf("read source: %v", err)
			}

			file, _ := Parse(name, src)
			got := printAST(file, 0)

			if update {
				if err := os.WriteFile(goldenPath, []byte(got), 0644); err != nil {
					t.Fatalf("update golden: %v", err)
				}
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden (run with UPDATE_GOLDENS=1 to generate): %v", err)
			}
			if got != string(want) {
				t.Errorf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", name, got, string(want))
			}
		})
	}
}
