package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCLIRunHello (task 16g): fuse run hello.fuse prints "hello" and exits 0.
func TestCLIRunHello(t *testing.T) {
	fuseBin := buildFuseCLI(t)
	if !hasGCC() {
		t.Skip("gcc not found; skipping CLI e2e tests")
	}
	rtLib := findRuntimeLib()
	if rtLib == "" {
		t.Skip("libfuse_rt.a not found; run 'make runtime' first")
	}

	tmpDir := t.TempDir()
	helloPath := filepath.Join(tmpDir, "hello.fuse")
	err := os.WriteFile(helloPath, []byte(`fn main() -> I32 {
	println("hello");
	return 0;
}
`), 0644)
	if err != nil {
		t.Fatalf("write hello.fuse: %v", err)
	}

	cmd := exec.Command(fuseBin, "run", "--no-color", helloPath)
	cmd.Dir = projectRoot(t) // run from project root so runtime lib is found
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	stdout, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			t.Fatalf("fuse run exited %d; stderr: %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		t.Fatalf("fuse run failed: %v", err)
	}

	got := string(stdout)
	if !strings.Contains(got, "hello") {
		t.Errorf("stdout = %q, want substring %q", got, "hello")
	}
}

// TestCLICheckBad (task 16h): fuse check bad.fuse reports error and exits 1.
func TestCLICheckBad(t *testing.T) {
	fuseBin := buildFuseCLI(t)

	tmpDir := t.TempDir()
	badPath := filepath.Join(tmpDir, "bad.fuse")
	err := os.WriteFile(badPath, []byte("fn main() -> I32 {{{ bad syntax\n"), 0644)
	if err != nil {
		t.Fatalf("write bad.fuse: %v", err)
	}

	cmd := exec.Command(fuseBin, "check", "--no-color", badPath)
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected fuse check to fail, but it succeeded; output: %s", string(output))
	}

	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got: %v", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("exit code = %d, want 1; output: %s", exitErr.ExitCode(), string(output))
	}
}

// TestCLIVersion verifies `fuse version` works.
func TestCLIVersion(t *testing.T) {
	fuseBin := buildFuseCLI(t)

	cmd := exec.Command(fuseBin, "version")
	stdout, err := cmd.Output()
	if err != nil {
		t.Fatalf("fuse version failed: %v", err)
	}

	got := strings.TrimSpace(string(stdout))
	if !strings.HasPrefix(got, "fuse ") {
		t.Errorf("version output = %q, want prefix %q", got, "fuse ")
	}
}

// TestCLIHelp verifies `fuse help` works.
func TestCLIHelp(t *testing.T) {
	fuseBin := buildFuseCLI(t)

	cmd := exec.Command(fuseBin, "help")
	stdout, err := cmd.Output()
	if err != nil {
		t.Fatalf("fuse help failed: %v", err)
	}

	got := string(stdout)
	if !strings.Contains(got, "Commands:") {
		t.Errorf("help output missing 'Commands:', got: %s", got)
	}
}

// buildFuseCLI compiles the fuse binary and returns the path.
func buildFuseCLI(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	bin := filepath.Join(tmpDir, "fuse")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	// Find project root by walking up from the test file location.
	root := projectRoot(t)

	cmd := exec.Command("go", "build", "-o", bin, "./cmd/fuse")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go build failed: %v\n%s", err, string(out))
	}
	return bin
}

// projectRoot finds the project root directory.
func projectRoot(t *testing.T) string {
	t.Helper()
	// Start from CWD and walk up looking for go.mod.
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 10; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Fatalf("could not find project root (go.mod)")
	return ""
}
