// Package testrunner owns compiler-driven test execution workflows.
//
// Convention: test functions are named test_* with signature
// fn test_xxx() -> I32, where 0 = pass and nonzero = fail.
// Test files are named *_test.fuse.
package testrunner

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/driver"
)

// TestResult holds the outcome of a single test function.
type TestResult struct {
	Name     string
	Passed   bool
	ExitCode int
	Duration time.Duration
	Output   string // captured stdout+stderr
}

// Run discovers and runs test functions in the given files.
// If files is empty, it discovers *_test.fuse in the current directory.
func Run(files []string, rtLib string) ([]TestResult, error) {
	if len(files) == 0 {
		discovered, err := discoverTestFiles(".")
		if err != nil {
			return nil, err
		}
		files = discovered
	}

	var results []TestResult
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		testFns := findTestFunctions(string(data))
		if len(testFns) == 0 {
			continue
		}

		modPath := fileToModulePath(path)
		for _, fnName := range testFns {
			r := runSingleTest(modPath, data, fnName, rtLib)
			results = append(results, r)
		}
	}
	return results, nil
}

// discoverTestFiles finds *_test.fuse files in the given directory.
func discoverTestFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), "_test.fuse") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files, nil
}

// findTestFunctions scans source text for fn test_* declarations.
func findTestFunctions(src string) []string {
	var fns []string
	lines := strings.Split(src, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		rest := trimmed
		if strings.HasPrefix(rest, "pub ") {
			rest = rest[4:]
		}
		if !strings.HasPrefix(rest, "fn test_") {
			continue
		}
		name := rest[3:] // skip "fn "
		if idx := strings.IndexByte(name, '('); idx > 0 {
			name = name[:idx]
		}
		name = strings.TrimSpace(name)
		if name != "" {
			fns = append(fns, name)
		}
	}
	return fns
}

// runSingleTest compiles and executes a single test function.
func runSingleTest(modPath string, src []byte, fnName string, rtLib string) TestResult {
	syntheticMain := "fn main() -> I32 { return " + fnName + "(); }\n"

	sources := map[string][]byte{
		modPath: src,
		"main":  []byte(syntheticMain),
	}

	tmpDir, err := os.MkdirTemp("", "fuse-test-*")
	if err != nil {
		return TestResult{Name: fnName, Passed: false, Output: err.Error()}
	}
	defer os.RemoveAll(tmpDir)

	exeName := "test_bin"
	if runtime.GOOS == "windows" {
		exeName = "test_bin.exe"
	}
	outPath := filepath.Join(tmpDir, exeName)

	start := time.Now()
	result := driver.Build(driver.BuildOptions{
		Sources:    sources,
		OutputPath: outPath,
		RuntimeLib: rtLib,
	})
	if hasErrors(result.Errors) {
		var errMsg strings.Builder
		for _, e := range result.Errors {
			errMsg.WriteString(e.Message)
			errMsg.WriteByte('\n')
		}
		return TestResult{
			Name:     fnName,
			Passed:   false,
			Duration: time.Since(start),
			Output:   "compile error: " + errMsg.String(),
		}
	}

	cmd := exec.Command(outPath)
	out, err := cmd.CombinedOutput()
	elapsed := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return TestResult{
				Name:     fnName,
				Passed:   false,
				Duration: elapsed,
				Output:   err.Error(),
			}
		}
	}

	return TestResult{
		Name:     fnName,
		Passed:   exitCode == 0,
		ExitCode: exitCode,
		Duration: elapsed,
		Output:   string(out),
	}
}

func fileToModulePath(path string) string {
	p := path
	if strings.HasSuffix(p, ".fuse") {
		p = p[:len(p)-5]
	}
	p = strings.ReplaceAll(p, "/", ".")
	p = strings.ReplaceAll(p, "\\", ".")
	return p
}

func hasErrors(errs []diagnostics.Diagnostic) bool {
	for _, e := range errs {
		if e.Severity == diagnostics.Error {
			return true
		}
	}
	return false
}
