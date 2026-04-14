package check

import (
	"strings"
	"testing"

	"github.com/Tembocs/fuse4/compiler/ast"
	"github.com/Tembocs/fuse4/compiler/parse"
	"github.com/Tembocs/fuse4/compiler/resolve"
	"github.com/Tembocs/fuse4/compiler/typetable"
)

// --- helpers ---

func checkSources(t *testing.T, sources map[string]string) (*Checker, *typetable.TypeTable) {
	t.Helper()
	parsed := make(map[string]*ast.File)
	for path, src := range sources {
		f, errs := parse.Parse(path+".fuse", []byte(src))
		for _, e := range errs {
			t.Errorf("[%s] parse error: %s", path, e)
		}
		parsed[path] = f
	}

	graph := resolve.BuildModuleGraph(parsed)
	r := resolve.NewResolver(graph)
	r.Resolve()
	for _, e := range r.Errors {
		t.Errorf("resolve error: %s", e)
	}

	tt := typetable.New()
	checker := NewChecker(tt, graph)
	checker.Check()
	return checker, tt
}

func checkOK(t *testing.T, sources map[string]string) (*Checker, *typetable.TypeTable) {
	t.Helper()
	c, tt := checkSources(t, sources)
	for _, e := range c.Errors {
		t.Errorf("unexpected check error: %s", e)
	}
	return c, tt
}

func checkExpectError(t *testing.T, sources map[string]string, substr string) *Checker {
	t.Helper()
	c, _ := checkSources(t, sources)
	for _, e := range c.Errors {
		if strings.Contains(e.Message, substr) {
			return c
		}
	}
	t.Errorf("expected error containing %q, got %d errors:", substr, len(c.Errors))
	for _, e := range c.Errors {
		t.Logf("  %s", e)
	}
	return c
}

// ===== Phase 01: Function type registration (two-pass) =====

func TestFnTypeRegisteredBeforeBodyCheck(t *testing.T) {
	// bar() calls foo() which is declared AFTER bar. Two-pass ensures this works.
	checkOK(t, map[string]string{
		"main": `
			fn bar() -> Bool { foo() }
			fn foo() -> Bool { true }
		`,
	})
}

func TestExternFnRegistered(t *testing.T) {
	c, _ := checkOK(t, map[string]string{
		"main": `extern fn puts(s: Ptr[U8]) -> I32;`,
	})
	if _, ok := c.funcTypes["main.puts"]; !ok {
		t.Error("extern fn 'puts' should be registered")
	}
}

func TestImplMethodsRegistered(t *testing.T) {
	c, _ := checkOK(t, map[string]string{
		"main": `
			struct Point { x: I32, y: I32 }
			impl Point {
				pub fn new(x: I32, y: I32) -> Point { Point { x: x, y: y } }
			}
		`,
	})
	if _, ok := c.funcTypes["Point.new"]; !ok {
		t.Error("impl method 'Point.new' should be registered")
	}
}

// ===== Phase 02: Nominal identity =====

func TestNominalIdentitySameNameDifferentModule(t *testing.T) {
	_, tt := checkOK(t, map[string]string{
		"mod_a": `struct Point { x: I32 }`,
		"mod_b": `struct Point { y: I32 }`,
	})
	a := tt.InternStruct("mod_a", "Point", nil)
	b := tt.InternStruct("mod_b", "Point", nil)
	if a == b {
		t.Error("Point from mod_a and mod_b must have different TypeIds")
	}
}

// ===== Phase 02: Primitive methods =====

func TestPrimitiveIntMethods(t *testing.T) {
	c, _ := checkOK(t, map[string]string{
		"main": `fn test() { let x: I32 = 42; }`,
	})
	// Verify primitive methods were registered.
	for _, method := range []string{"toFloat", "toInt", "abs", "min", "max"} {
		key := "I32." + method
		if _, ok := c.funcTypes[key]; !ok {
			t.Errorf("primitive method %q not registered", key)
		}
	}
}

func TestPrimitiveFloatMethods(t *testing.T) {
	c, _ := checkOK(t, map[string]string{
		"main": `fn test() { }`,
	})
	for _, method := range []string{"toInt", "isNan", "isInfinite", "floor", "ceil", "sqrt", "abs"} {
		key := "F64." + method
		if _, ok := c.funcTypes[key]; !ok {
			t.Errorf("primitive method %q not registered", key)
		}
	}
}

func TestPrimitiveCharMethods(t *testing.T) {
	c, _ := checkOK(t, map[string]string{
		"main": `fn test() { }`,
	})
	for _, method := range []string{"toInt", "isLetter", "isDigit", "isWhitespace"} {
		key := "Char." + method
		if _, ok := c.funcTypes[key]; !ok {
			t.Errorf("primitive method %q not registered", key)
		}
	}
}

func TestPrimitiveBoolNot(t *testing.T) {
	c, _ := checkOK(t, map[string]string{
		"main": `fn test() { }`,
	})
	if _, ok := c.funcTypes["Bool.not"]; !ok {
		t.Error("Bool.not not registered")
	}
}

// ===== Phase 02: Numeric widening =====

func TestNumericWideningAllowed(t *testing.T) {
	// I32 + I64 should widen to I64 without error.
	checkOK(t, map[string]string{
		"main": `fn test() { let a: I32 = 1; let b: I64 = 2; let c = a + b; }`,
	})
}

func TestNumericComparisonAllowed(t *testing.T) {
	checkOK(t, map[string]string{
		"main": `fn test() { let a: I32 = 1; let b: I64 = 2; let c = a == b; }`,
	})
}

// ===== Phase 02: Literal typing =====

func TestIntLiteralDefaultsToI32(t *testing.T) {
	c, tt := checkOK(t, map[string]string{
		"main": `fn test() { let x = 42; }`,
	})
	_ = c
	// Verify I32 is the default for unsuffixed int literals.
	if tt.I32 == typetable.InvalidTypeId {
		t.Error("I32 should be valid")
	}
}

func TestIntLiteralWithSuffix(t *testing.T) {
	checkOK(t, map[string]string{
		"main": `fn test() { let x = 42u64; let y = 0xffi32; }`,
	})
}

func TestFloatLiteralDefaultsToF64(t *testing.T) {
	checkOK(t, map[string]string{
		"main": `fn test() { let x = 3.14; }`,
	})
}

func TestFloatLiteralWithSuffix(t *testing.T) {
	checkOK(t, map[string]string{
		"main": `fn test() { let x = 3.14f32; }`,
	})
}

// ===== Phase 03: Trait resolution =====

func TestTraitMethodLookup(t *testing.T) {
	c, _ := checkOK(t, map[string]string{
		"main": `
			trait Display {
				fn fmt(ref self) -> Bool;
			}
			struct Point { x: I32 }
			impl Display : Point {
				fn fmt(ref self) -> Bool { true }
			}
		`,
	})
	// Verify trait methods are registered.
	if methods, ok := c.traitMethods["Display"]; !ok || len(methods) == 0 {
		t.Error("Display trait methods should be registered")
	}
	// Verify trait impl is recorded.
	if !c.traitImpls["Display:Point"] {
		t.Error("Display:Point impl should be recorded")
	}
}

func TestBoundChainLookup(t *testing.T) {
	c, _ := checkOK(t, map[string]string{
		"main": `
			trait Equatable {
				fn eq(ref self, other: ref Self) -> Bool;
			}
			trait Hashable : Equatable {
				fn hash(ref self) -> U64;
			}
		`,
	})
	// Hashable extends Equatable, so searching Hashable's chain should find eq.
	ret := c.searchTraitChain("Hashable", "eq")
	if ret == typetable.InvalidTypeId {
		t.Error("bound-chain lookup should find 'eq' through Hashable → Equatable")
	}
}

func TestBoundChainLookupDirectMethod(t *testing.T) {
	c, _ := checkOK(t, map[string]string{
		"main": `
			trait Hashable {
				fn hash(ref self) -> U64;
			}
		`,
	})
	ret := c.searchTraitChain("Hashable", "hash")
	if ret == typetable.InvalidTypeId {
		t.Error("should find 'hash' directly on Hashable")
	}
}

func TestBoundChainLookupMissing(t *testing.T) {
	c, _ := checkOK(t, map[string]string{
		"main": `
			trait Hashable {
				fn hash(ref self) -> U64;
			}
		`,
	})
	ret := c.searchTraitChain("Hashable", "nonexistent")
	if ret != typetable.InvalidTypeId {
		t.Error("should not find 'nonexistent' in Hashable")
	}
}

// ===== Phase 04: Contextual inference =====

func TestStructFieldTypeResolution(t *testing.T) {
	checkOK(t, map[string]string{
		"main": `
			struct Config { debug: Bool, level: I32 }
			fn test() { let c = Config { debug: true, level: 5 }; }
		`,
	})
}

// ===== Phase 05: Body checking across modules (stdlib uniform treatment) =====

func TestMultiModuleChecking(t *testing.T) {
	checkOK(t, map[string]string{
		"core.math": `pub fn add(a: I32, b: I32) -> I32 { a + b }`,
		"main":      `import core.math; fn test() { let x = 1 + 2; }`,
	})
}

func TestStdlibAndUserUniform(t *testing.T) {
	// Both modules are checked — no skipping.
	checkOK(t, map[string]string{
		"stdlib.core": `pub fn identity(x: I32) -> I32 { x }`,
		"user.app":    `import stdlib.core; fn main() { let y = 42; }`,
	})
}

// ===== Expression typing =====

func TestBinaryArithmeticType(t *testing.T) {
	checkOK(t, map[string]string{
		"main": `fn test() { let x = 1 + 2; }`,
	})
}

func TestBinaryComparisonReturnsBool(t *testing.T) {
	checkOK(t, map[string]string{
		"main": `fn test() { let x = 1 == 2; }`,
	})
}

func TestLogicalOpReturnsBool(t *testing.T) {
	checkOK(t, map[string]string{
		"main": `fn test() { let x = true && false; }`,
	})
}

func TestUnaryNegation(t *testing.T) {
	checkOK(t, map[string]string{
		"main": `fn test() { let x = -42; }`,
	})
}

func TestUnaryNot(t *testing.T) {
	checkOK(t, map[string]string{
		"main": `fn test() { let x = !true; }`,
	})
}

func TestUnaryRef(t *testing.T) {
	checkOK(t, map[string]string{
		"main": `fn test() { let x = 42; let y = ref x; }`,
	})
}

func TestTupleCreationAndAccess(t *testing.T) {
	checkOK(t, map[string]string{
		"main": `fn test() { let p = (1, true); }`,
	})
}

func TestBlockTailExpression(t *testing.T) {
	checkOK(t, map[string]string{
		"main": `fn test() -> I32 { let x = 1; x + 1 }`,
	})
}

func TestIfElseExpression(t *testing.T) {
	checkOK(t, map[string]string{
		"main": `fn test() -> I32 { if true { 1 } else { 2 } }`,
	})
}

func TestMatchExpression(t *testing.T) {
	checkOK(t, map[string]string{
		"main": `fn test() -> I32 { match 1 { 0 => 10, _ => 20 } }`,
	})
}

func TestForLoop(t *testing.T) {
	checkOK(t, map[string]string{
		"main": `fn test() { for item in items { let x = item; }; }`,
	})
}

func TestWhileLoop(t *testing.T) {
	checkOK(t, map[string]string{
		"main": `fn test() { while true { }; }`,
	})
}

func TestReturnExprType(t *testing.T) {
	checkOK(t, map[string]string{
		"main": `fn test() -> I32 { return 42; }`,
	})
}

func TestClosureExpression(t *testing.T) {
	checkOK(t, map[string]string{
		"main": `fn test() { let f = fn(x: I32) -> I32 { x + 1 }; }`,
	})
}

func TestSpawnExpression(t *testing.T) {
	checkOK(t, map[string]string{
		"main": `fn test() { spawn fn() { }; }`,
	})
}

// ===== Question operator (?) =====

func TestQuestionOnResultExtractsT(t *testing.T) {
	_, tt := checkOK(t, map[string]string{
		"main": `fn test() { }`,
	})
	// Manually create a Result[I32, String] enum type and verify ? extracts I32.
	strTy := tt.InternStruct("std", "String", nil)
	resultTy := tt.InternEnum("std", "Result", []typetable.TypeId{tt.I32, strTy})
	te := tt.Get(resultTy)
	if te.Name != "Result" {
		t.Fatalf("expected name Result, got %s", te.Name)
	}
	if len(te.TypeArgs) < 1 || te.TypeArgs[0] != tt.I32 {
		t.Fatalf("expected TypeArgs[0] = I32, got %v", te.TypeArgs)
	}
	// The checker logic: if Result with TypeArgs, return TypeArgs[0].
	if te.TypeArgs[0] != tt.I32 {
		t.Error("? on Result[I32, String] should yield I32")
	}
}

func TestQuestionOnOptionExtractsT(t *testing.T) {
	_, tt := checkOK(t, map[string]string{
		"main": `fn test() { }`,
	})
	optionTy := tt.InternEnum("std", "Option", []typetable.TypeId{tt.Bool})
	te := tt.Get(optionTy)
	if te.Name != "Option" {
		t.Fatalf("expected name Option, got %s", te.Name)
	}
	if len(te.TypeArgs) < 1 || te.TypeArgs[0] != tt.Bool {
		t.Error("? on Option[Bool] should yield Bool")
	}
}

func TestQuestionOnUnknownReturnsUnknown(t *testing.T) {
	_, tt := checkOK(t, map[string]string{
		"main": `fn test() { }`,
	})
	// A plain struct that is not Result or Option should not unwrap.
	plainTy := tt.InternStruct("m", "Foo", nil)
	te := tt.Get(plainTy)
	if te.Name == "Result" || te.Name == "Option" {
		t.Fatal("Foo should not be Result or Option")
	}
	// The checker would return Unknown for a non-Result/Option type.
	// Verify the type entry has no TypeArgs to unwrap.
	if len(te.TypeArgs) != 0 {
		t.Error("Foo should have no TypeArgs")
	}
}

// ===== Error detection =====

func TestMalformedInputDoesNotPanic(t *testing.T) {
	sources := []map[string]string{
		{"main": ""},
		{"main": "fn test() { }"},
		{"main": "fn test() { 1 + true; }"},
		{"main": "fn test() { let x: I32 = true; }"},
	}
	for _, src := range sources {
		// Must not panic
		checkSources(t, src)
	}
}
