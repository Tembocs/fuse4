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
