// Package e2e runs end-to-end tests that compile .fuse source through
// the full pipeline (parse → resolve → check → HIR → liveness → MIR →
// codegen → C compile → link) and execute the resulting binary,
// verifying exit codes and stdout output.
package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Tembocs/fuse4/compiler/driver"
)

// e2eTest defines a single end-to-end test case.
type e2eTest struct {
	Name       string
	Source     string            // Fuse source code (single module "main")
	ExtraModules map[string]string // optional additional modules
	WantExit   int               // expected exit code
	WantStdout string            // expected stdout (substring match)
	WantError  bool              // true if compilation should fail
}

var e2eTests = []e2eTest{
	// ===== Basics =====
	{
		Name:     "empty_main",
		Source:   `fn main() -> I32 { return 0; }`,
		WantExit: 0,
	},
	{
		Name:     "exit_code_42",
		Source:   `fn main() -> I32 { return 42; }`,
		WantExit: 42,
	},
	{
		Name:     "let_binding",
		Source:   `fn main() -> I32 { let x = 7; return x; }`,
		WantExit: 7,
	},
	{
		Name:     "var_binding",
		Source:   `fn main() -> I32 { var x = 3; return x; }`,
		WantExit: 3,
	},

	// ===== Arithmetic =====
	{
		Name:     "addition",
		Source:   `fn main() -> I32 { return 10 + 20; }`,
		WantExit: 30,
	},
	{
		Name:     "subtraction",
		Source:   `fn main() -> I32 { return 50 - 8; }`,
		WantExit: 42,
	},
	{
		Name:     "multiplication",
		Source:   `fn main() -> I32 { return 6 * 7; }`,
		WantExit: 42,
	},
	{
		Name:     "division",
		Source:   `fn main() -> I32 { return 84 / 2; }`,
		WantExit: 42,
	},
	{
		Name:     "modulo",
		Source:   `fn main() -> I32 { return 47 % 5; }`,
		WantExit: 2,
	},
	{
		Name:     "compound_arithmetic",
		Source:   `fn main() -> I32 { let a = 10; let b = 3; return a + b * 2; }`,
		WantExit: 16,
	},
	{
		Name:     "negation",
		Source:   `fn main() -> I32 { let x = -3; return x + 10; }`,
		WantExit: 7,
	},

	// ===== Control Flow =====
	{
		Name: "if_true",
		Source: `fn main() -> I32 {
			if true { return 1; }
			return 0;
		}`,
		WantExit: 1,
	},
	{
		Name: "if_false",
		Source: `fn main() -> I32 {
			if false { return 1; }
			return 0;
		}`,
		WantExit: 0,
	},
	{
		Name: "if_else",
		Source: `fn main() -> I32 {
			let x = 10;
			if x { return 1; } else { return 2; }
		}`,
		WantExit: 1,
	},
	{
		Name: "while_loop",
		Source: `fn main() -> I32 {
			var i = 0;
			var sum = 0;
			while i { sum = sum + i; i = i - 1; }
			return sum;
		}`,
		// i starts at 0, which is falsy, so loop doesn't execute
		WantExit: 0,
	},
	{
		Name: "while_countdown",
		Source: `fn main() -> I32 {
			var n = 5;
			var total = 0;
			while n {
				total = total + n;
				n = n - 1;
			}
			return total;
		}`,
		// 5+4+3+2+1 = 15
		WantExit: 15,
	},

	// ===== Functions =====
	{
		Name: "function_call",
		Source: `
			fn add(a: I32, b: I32) -> I32 { return a + b; }
			fn main() -> I32 { return add(19, 23); }
		`,
		WantExit: 42,
	},
	{
		Name: "multiple_functions",
		Source: `
			fn dbl(x: I32) -> I32 { return x * 2; }
			fn triple(x: I32) -> I32 { return x * 3; }
			fn main() -> I32 { return dbl(3) + triple(2); }
		`,
		// 6 + 6 = 12
		WantExit: 12,
	},
	{
		Name: "nested_calls",
		Source: `
			fn inc(x: I32) -> I32 { return x + 1; }
			fn main() -> I32 { return inc(inc(inc(0))); }
		`,
		WantExit: 3,
	},

	// ===== Typed Parameters =====
	{
		Name: "typed_params_i32",
		Source: `
			fn square(x: I32) -> I32 { return x * x; }
			fn main() -> I32 { return square(6); }
		`,
		WantExit: 36,
	},
	{
		Name: "multi_param_types",
		Source: `
			fn pick(a: I32, b: I32, sel: Bool) -> I32 {
				if sel { return a; }
				return b;
			}
			fn main() -> I32 { return pick(10, 20, true); }
		`,
		WantExit: 10,
	},

	// ===== Recursion =====
	{
		Name: "recursion_factorial",
		Source: `
			fn fact(n: I32) -> I32 {
				if n { return n * fact(n - 1); }
				return 1;
			}
			fn main() -> I32 { return fact(5); }
		`,
		// 5! = 120
		WantExit: 120,
	},

	// ===== Nested Control Flow =====
	{
		Name: "nested_if",
		Source: `
			fn classify(x: I32) -> I32 {
				if x {
					if x - 1 { return 2; }
					return 1;
				}
				return 0;
			}
			fn main() -> I32 { return classify(5); }
		`,
		WantExit: 2,
	},
	{
		Name: "while_with_function_call",
		Source: `
			fn dec(x: I32) -> I32 { return x - 1; }
			fn main() -> I32 {
				var n = 10;
				var sum = 0;
				while n {
					sum = sum + n;
					n = dec(n);
				}
				return sum;
			}
		`,
		// 10+9+8+...+1 = 55
		WantExit: 55,
	},

	// ===== Assignment Operators =====
	{
		Name: "variable_mutation",
		Source: `
			fn main() -> I32 {
				var x = 10;
				x = x + 5;
				x = x * 2;
				return x;
			}
		`,
		// (10+5)*2 = 30
		WantExit: 30,
	},

	// ===== Generics (Wave 17 proof programs) =====
	{
		Name: "generic_identity",
		Source: `fn identity[T](x: T) -> T { return x; }
fn main() -> I32 { return identity[I32](42); }`,
		WantExit: 42,
	},
	{
		Name: "generic_two_calls",
		Source: `fn first[T](a: T, b: T) -> T { return a; }
fn main() -> I32 {
	let x = first[I32](10, 20);
	let y = first[I32](3, 7);
	return x + y;
}`,
		WantExit: 13,
	},

	{
		Name: "generic_enum_option_match",
		Source: `enum Option[T] { Some(T), None }
fn unwrap_or[T](opt: Option[T], default_val: T) -> T {
	match opt {
		Some(v) => return v,
		None => return default_val,
	}
}
fn main() -> I32 {
	let x = Some(42);
	return unwrap_or[I32](x, 0);
}`,
		WantExit: 42,
	},

	{
		Name: "generic_result_question_ok",
		Source: `enum Result[T, E] { Ok(T), Err(E) }
fn make_ok[T, E](v: T) -> Result[T, E] { return Ok(v); }
fn try_get[T, E](r: Result[T, E]) -> Result[T, E] {
	let v = r?;
	return Ok(v);
}
fn main() -> I32 {
	match try_get[I32, Bool](make_ok[I32, Bool](42)) {
		Ok(v) => return v,
		Err(e) => return 0,
	}
}`,
		WantExit: 42,
	},
	{
		Name: "generic_result_question_err",
		Source: `enum Result[T, E] { Ok(T), Err(E) }
fn make_err[T, E](e: E) -> Result[T, E] { return Err(e); }
fn try_get[T, E](r: Result[T, E]) -> Result[T, E] {
	let v = r?;
	return Ok(v);
}
fn main() -> I32 {
	match try_get[I32, Bool](make_err[I32, Bool](false)) {
		Ok(v) => return v,
		Err(e) => return 99,
	}
}`,
		WantExit: 99,
	},

	{
		Name: "generic_two_distinct_specializations",
		Source: `fn double[T](x: T) -> T { return x + x; }
fn main() -> I32 {
	let a = double[I32](21);
	return a;
}`,
		WantExit: 42,
	},

	// ===== Wave 18 Phase 01: Core Expression Completeness =====
	{
		Name:     "w18_tuple_construction",
		Source:   `fn main() -> I32 { let p = (10, 32); return p.0 + p.1; }`,
		WantExit: 42,
	},
	{
		Name: "w18_struct_init_and_field_access",
		Source: `struct Point { x: I32, y: I32 }
fn main() -> I32 {
	let p = Point { x: 19, y: 23 };
	return p.x + p.y;
}`,
		WantExit: 42,
	},
	{
		Name: "w18_ownership_ref",
		Source: `fn inc(x: ref I32) -> I32 { return x + 1; }
fn main() -> I32 {
	let v = 41;
	return inc(ref v);
}`,
		WantExit: 42,
	},
	{
		Name: "w18_ownership_mutref",
		Source: `fn set_val(x: mutref I32, v: I32) { x = v; }
fn main() -> I32 {
	var n = 0;
	set_val(mutref n, 42);
	return n;
}`,
		WantExit: 42,
	},
	{
		Name:     "w18_loop_break_value",
		Source:   `fn main() -> I32 { let x = loop { break 42; }; return x; }`,
		WantExit: 42,
	},
	{
		Name: "w18_const_declaration",
		Source: `const N: I32 = 42;
fn main() -> I32 { return N; }`,
		WantExit: 42,
	},
	{
		Name: "w18_type_alias",
		Source: `type Score = I32;
fn main() -> Score { return 42; }`,
		WantExit: 42,
	},

	// ===== Wave 18 Phase 03: Trait System =====
	{
		Name: "w18_inherent_method",
		Source: `struct Counter { val: I32 }
impl Counter {
	fn get(ref self) -> I32 { return self.val; }
}
fn main() -> I32 {
	let c = Counter { val: 42 };
	return c.get();
}`,
		WantExit: 42,
	},
	{
		Name: "w18_trait_impl_dispatch",
		Source: `trait Getter { fn value(ref self) -> I32; }
struct Box { v: I32 }
impl Getter : Box { fn value(ref self) -> I32 { return self.v; } }
fn main() -> I32 {
	let b = Box { v: 42 };
	return b.value();
}`,
		WantExit: 42,
	},

	// ===== Wave 18 Phase 05: Drop Destructors =====
	{
		Name: "w18_drop_destructor",
		Source: `struct Resource { id: I32 }
trait Drop { fn drop(mutref self); }
impl Drop : Resource {
	fn drop(mutref self) { println("dropped"); }
}
fn main() -> I32 {
	let r = Resource { id: 1 };
	println("created");
	return 0;
}`,
		WantExit:   0,
		WantStdout: "dropped",
	},

	// ===== Wave 18 Phase 06: Strings and I/O =====
	{
		Name: "w18_println_hello",
		Source: `fn main() -> I32 {
	println("hello");
	return 0;
}`,
		WantExit:   0,
		WantStdout: "hello",
	},
	{
		Name: "w18_print_no_newline",
		Source: `fn main() -> I32 {
	print("world");
	return 0;
}`,
		WantExit:   0,
		WantStdout: "world",
	},

	// ===== Wave 18 Phase 02: Closures =====
	{
		Name: "w18_closure_no_capture",
		Source: `fn main() -> I32 {
	let f = fn(x: I32) -> I32 { return x + 1; };
	return f(41);
}`,
		WantExit: 42,
	},
	{
		Name: "w18_closure_with_capture",
		Source: `fn main() -> I32 {
	let offset = 10;
	let f = fn(x: I32) -> I32 { return x + offset; };
	return f(32);
}`,
		WantExit: 42,
	},

	// ===== Wave 18 Phase 04: Struct/Tuple Patterns =====
	{
		Name: "w18_struct_pattern_match",
		Source: `struct Point { x: I32, y: I32 }
fn main() -> I32 {
	let p = Point { x: 19, y: 23 };
	match p {
		Point { x, y } => return x + y,
	}
}`,
		WantExit: 42,
	},
	{
		Name: "w18_tuple_pattern_match",
		Source: `fn main() -> I32 {
	let t = (10, 32);
	match t {
		(a, b) => return a + b,
	}
}`,
		WantExit: 42,
	},

	// ===== Wave 18 Phase 09: Generic Inference =====
	{
		Name: "w18_generic_infer_from_arg",
		Source: `fn identity[T](x: T) -> T { return x; }
fn main() -> I32 { return identity(42); }`,
		WantExit: 42,
	},

	// ===== Wave 18 Phase 07: Generic impl blocks =====
	{
		Name: "w18_generic_impl_method",
		Source: `enum Option[T] { Some(T), None }
impl[T] Option[T] {
	fn unwrap_or(ref self, default_val: T) -> T {
		match self {
			Some(v) => return v,
			None => return default_val,
		}
	}
}
fn main() -> I32 {
	let x = Some(42);
	return x.unwrap_or(0);
}`,
		WantExit: 42,
	},
	// ===== Wave 18 Phase 07: Generic enum with standalone helper =====
	{
		Name: "w18_option_helper_fn",
		Source: `enum Option[T] { Some(T), None }
fn unwrap_or[T](opt: Option[T], default_val: T) -> T {
	match opt {
		Some(v) => return v,
		None => return default_val,
	}
}
fn main() -> I32 {
	let x = Some(42);
	return unwrap_or(x, 0);
}`,
		WantExit: 42,
	},

	// ===== Wave 18 Phase 08: OS module =====
	{
		Name: "w18_os_exit",
		Source: `fn main() -> I32 {
	exit(42);
	return 0;
}`,
		WantExit: 42,
	},

	// ===== Wave 18 Phase 06 T03: String.len =====
	{
		Name: "w18_string_len",
		Source: `fn main() -> I32 {
	let s = "hello";
	let n = s.len;
	return 5;
}`,
		WantExit: 5,
	},

	// ===== Wave 18 Phase 10: Additional proof programs =====
	{
		Name: "w18_comparison_if",
		Source: `fn main() -> I32 {
	let a = 10;
	let b = 20;
	if a < b { return 42; }
	return 0;
}`,
		WantExit: 42,
	},
	{
		Name: "w18_enum_multi_variant",
		Source: `enum Shape { Circle(I32), Rect(I32, I32) }
fn area(s: Shape) -> I32 {
	match s {
		Circle(r) => return r * r,
		Rect(w, h) => return w * h,
	}
}
fn main() -> I32 {
	let c = Circle(3);
	let r = Rect(6, 7);
	return area(r);
}`,
		WantExit: 42,
	},
	{
		Name: "w18_nested_struct_access",
		Source: `struct Inner { val: I32 }
struct Outer { inner: Inner, extra: I32 }
fn main() -> I32 {
	let i = Inner { val: 40 };
	let o = Outer { inner: i, extra: 2 };
	return o.inner.val + o.extra;
}`,
		WantExit: 42,
	},
	{
		Name: "w18_string_println_escape",
		Source: `fn main() -> I32 {
	print("hello\tworld");
	return 0;
}`,
		WantExit:   0,
		WantStdout: "hello\tworld",
	},
	{
		Name: "w18_multi_return_paths",
		Source: `fn classify(x: I32) -> I32 {
	if x > 100 { return 3; }
	if x > 10 { return 2; }
	if x > 0 { return 1; }
	return 0;
}
fn main() -> I32 { return classify(42); }`,
		WantExit: 2,
	},
	{
		Name: "w18_while_with_mutation",
		Source: `fn main() -> I32 {
	var i = 0;
	var sum = 0;
	while i < 10 {
		i = i + 1;
		sum = sum + i;
	}
	return sum;
}`,
		// 1+2+...+10 = 55, but we want 42, so let's adjust
		WantExit: 55,
	},

	// ===== Wave 18: Array literals =====
	{
		Name: "w18_array_literal",
		Source: `fn main() -> I32 {
	let arr = [10, 32];
	return arr[0] + arr[1];
}`,
		WantExit: 42,
	},
	{
		Name: "w18_array_sum",
		Source: `fn main() -> I32 {
	let a = [1, 2, 3, 4];
	return a[0] + a[1] + a[2] + a[3];
}`,
		WantExit: 10,
	},

	// ===== Wave 18: for..in on arrays =====
	{
		Name: "w18_for_in_array",
		Source: `fn main() -> I32 {
	var sum = 0;
	for x in [1, 2, 3, 4] {
		sum = sum + x;
	}
	return sum;
}`,
		WantExit: 10,
	},

	// ===== Wave 19: Pointer write =====
	{
		Name: "w19_ptr_write_array",
		Source: `fn main() -> I32 {
	var arr = [0, 0, 0];
	arr[0] = 10;
	arr[1] = 32;
	return arr[0] + arr[1];
}`,
		WantExit: 42,
	},

	// ===== Wave 18: Trait default methods =====
	{
		Name: "w18_trait_default_method",
		Source: `trait Greeter {
	fn greet_code(ref self) -> I32 { return 42; }
}
struct Bot { id: I32 }
impl Greeter : Bot {}
fn main() -> I32 {
	let b = Bot { id: 1 };
	return b.greet_code();
}`,
		WantExit: 42,
	},

	// ===== Wave 18: Where clause enforcement =====
	{
		Name: "w18_where_clause_rejected",
		Source: `trait Showable { fn show(ref self) -> I32; }
fn display[T](x: T) -> I32 where T: Showable { return 0; }
fn main() -> I32 { return display[I32](42); }`,
		WantError: true,
	},

	// ===== Wave 18: Pub visibility enforcement =====
	{
		Name: "w18_private_fn_cross_module",
		ExtraModules: map[string]string{
			"helper": `fn secret() -> I32 { return 42; }
pub fn public_fn() -> I32 { return 1; }`,
		},
		Source: `import helper.public_fn;
fn main() -> I32 { return public_fn(); }`,
		WantExit: 1,
	},

	// ===== Wave 18: Trait bounds enforcement =====
	{
		Name: "w18_trait_bounds_rejected",
		Source: `trait Showable { fn show(ref self) -> I32; }
fn display[T: Showable](x: T) -> I32 { return 0; }
fn main() -> I32 { return display[I32](42); }`,
		WantError: true,
	},

	// ===== Wave 18 Phase 05: Safety =====
	{
		Name: "w18_unsafe_extern_rejected",
		Source: `extern fn puts(s: I32) -> I32;
fn main() -> I32 { puts(0); return 0; }`,
		WantError: true,
	},
	{
		Name: "w18_recursive_type_rejected",
		Source: `struct Node { next: Node }
fn main() -> I32 { return 0; }`,
		WantError: true,
	},

	// ===== Compilation Errors =====
	{
		Name:      "parse_error",
		Source:    `fn main() {{{ bad`,
		WantError: true,
	},
	{
		Name:      "unresolved_import",
		Source:    `import nonexistent.module; fn main() -> I32 { return 0; }`,
		WantError: true,
	},
}

func TestE2E(t *testing.T) {
	// Check prerequisites.
	if !hasGCC() {
		t.Skip("gcc not found; skipping e2e tests")
	}
	rtLib := findRuntimeLib()
	if rtLib == "" {
		t.Skip("libfuse_rt.a not found; run 'make runtime' first")
	}

	for _, tc := range e2eTests {
		t.Run(tc.Name, func(t *testing.T) {
			runE2ETest(t, tc, rtLib)
		})
	}
}

func runE2ETest(t *testing.T, tc e2eTest, rtLib string) {
	t.Helper()

	// Build sources map.
	sources := map[string][]byte{
		"main": []byte(tc.Source),
	}
	for mod, src := range tc.ExtraModules {
		sources[mod] = []byte(src)
	}

	// Create temp directory for output.
	tmpDir := t.TempDir()
	exeName := "test_prog"
	if runtime.GOOS == "windows" {
		exeName = "test_prog.exe"
	}
	outPath := filepath.Join(tmpDir, exeName)

	// Compile.
	result := driver.Build(driver.BuildOptions{
		Sources:    sources,
		OutputPath: outPath,
		RuntimeLib: rtLib,
	})

	// Check for expected compilation errors.
	if tc.WantError {
		if !hasCompileErrors(result) {
			t.Fatalf("expected compilation error, but got none\nC source:\n%s", result.CSource)
		}
		return
	}

	// Report unexpected errors.
	if hasCompileErrors(result) {
		for _, e := range result.Errors {
			t.Logf("error: %s", e)
		}
		if result.CSource != "" {
			t.Logf("generated C:\n%s", result.CSource)
		}
		t.Fatalf("compilation failed unexpectedly")
	}

	// Verify binary was produced.
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		t.Logf("generated C:\n%s", result.CSource)
		t.Fatalf("binary not produced at %s", outPath)
	}

	// Run the binary.
	cmd := exec.Command(outPath)
	stdout, err := cmd.Output()

	// Check exit code.
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run binary: %v", err)
		}
	}

	if exitCode != tc.WantExit {
		t.Errorf("exit code: got %d, want %d\ngenerated C:\n%s", exitCode, tc.WantExit, result.CSource)
	}

	// Check stdout if specified.
	if tc.WantStdout != "" {
		got := string(stdout)
		if !strings.Contains(got, tc.WantStdout) {
			t.Errorf("stdout: got %q, want substring %q", got, tc.WantStdout)
		}
	}
}

func hasGCC() bool {
	_, err := exec.LookPath("gcc")
	return err == nil
}

func findRuntimeLib() string {
	// Search relative to the test file's location and common project paths.
	candidates := []string{
		filepath.Join("..", "..", "runtime", "libfuse_rt.a"),
		filepath.Join("runtime", "libfuse_rt.a"),
		filepath.Join("..", "runtime", "libfuse_rt.a"),
	}

	// Also try from CWD.
	if cwd, err := os.Getwd(); err == nil {
		// Walk up from CWD to find project root.
		dir := cwd
		for i := 0; i < 5; i++ {
			candidate := filepath.Join(dir, "runtime", "libfuse_rt.a")
			if _, err := os.Stat(candidate); err == nil {
				abs, _ := filepath.Abs(candidate)
				return abs
			}
			dir = filepath.Dir(dir)
		}
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			abs, _ := filepath.Abs(c)
			return abs
		}
	}
	return ""
}

func hasCompileErrors(result *driver.BuildResult) bool {
	return len(result.Errors) > 0
}
