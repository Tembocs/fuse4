// Package repl owns the interactive read-eval-print loop for Fuse.
//
// The REPL wraps user input into a synthetic main function, compiles it
// through the full pipeline, executes the result, and prints stdout.
// Top-level items (fn, struct, enum, trait, impl, const, type) persist
// across iterations so the user can build up definitions interactively.
package repl

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/driver"
)

// Start runs the interactive REPL.
func Start(rtLib string, color bool) {
	prompt := "fuse> "
	contPrompt := "  ... "
	if color {
		prompt = diagnostics.Colorize("fuse> ", diagnostics.ColorGreen, true)
		contPrompt = diagnostics.Colorize("  ... ", diagnostics.ColorDim, true)
	}

	fmt.Println("fuse 0.1.0-dev REPL")
	fmt.Println("type :help for help, :quit to exit")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	var preamble strings.Builder // accumulated top-level items
	counter := 0

	for {
		fmt.Fprint(os.Stderr, prompt)
		if !scanner.Scan() {
			fmt.Fprintln(os.Stderr)
			break
		}
		line := scanner.Text()

		// Handle special commands.
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if trimmed == ":quit" || trimmed == ":q" || trimmed == ":exit" {
			break
		}
		if trimmed == ":help" || trimmed == ":h" {
			printHelp()
			continue
		}
		if trimmed == ":clear" {
			preamble.Reset()
			fmt.Fprintln(os.Stderr, diagnostics.Colorize("cleared definitions", diagnostics.ColorDim, color))
			continue
		}

		// Multi-line: if braces are unbalanced, keep reading.
		input := line
		for braceDepth(input) > 0 {
			fmt.Fprint(os.Stderr, contPrompt)
			if !scanner.Scan() {
				break
			}
			input += "\n" + scanner.Text()
		}

		// Determine if the input is a top-level item or an expression/statement.
		if isTopLevel(strings.TrimSpace(input)) {
			preamble.WriteString(input)
			preamble.WriteByte('\n')
			fmt.Fprintln(os.Stderr, diagnostics.Colorize("defined", diagnostics.ColorDim, color))
			continue
		}

		// Wrap in a main function.
		counter++
		var src strings.Builder
		src.WriteString(preamble.String())
		src.WriteString("fn main() -> I32 {\n")
		src.WriteString("    ")
		src.WriteString(input)
		src.WriteString("\n    return 0;\n}\n")

		// Compile and run.
		output, ok := compileAndRun(src.String(), rtLib, counter)
		if !ok {
			// Error already printed by compileAndRun.
			continue
		}
		if output != "" {
			fmt.Print(output)
		}
	}
}

func compileAndRun(src string, rtLib string, counter int) (string, bool) {
	sources := map[string][]byte{
		"main": []byte(src),
	}

	tmpDir, err := os.MkdirTemp("", "fuse-repl-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		return "", false
	}
	defer os.RemoveAll(tmpDir)

	exeName := fmt.Sprintf("repl_%d", counter)
	if runtime.GOOS == "windows" {
		exeName += ".exe"
	}
	outPath := filepath.Join(tmpDir, exeName)

	result := driver.Build(driver.BuildOptions{
		Sources:    sources,
		OutputPath: outPath,
		RuntimeLib: rtLib,
	})

	if len(result.Errors) > 0 {
		for _, d := range result.Errors {
			if d.Severity == diagnostics.Error {
				fmt.Fprintf(os.Stderr, "%s: %s\n",
					diagnostics.Colorize("error", diagnostics.ColorBoldRed, true),
					d.Message)
			}
		}
		return "", false
	}

	cmd := exec.Command(outPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 0 {
				return string(out), true
			}
		} else {
			fmt.Fprintf(os.Stderr, "run error: %s\n", err)
			return "", false
		}
	}
	return string(out), true
}

func braceDepth(s string) int {
	depth := 0
	inString := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if ch == '"' && (i == 0 || s[i-1] != '\\') {
			inString = !inString
		}
		if inString {
			continue
		}
		if ch == '{' {
			depth++
		} else if ch == '}' {
			depth--
		}
	}
	return depth
}

func isTopLevel(s string) bool {
	// Check if the input starts with a top-level keyword.
	prefixes := []string{
		"fn ", "pub fn ", "struct ", "pub struct ",
		"enum ", "pub enum ", "trait ", "pub trait ",
		"impl ", "impl[", "const ", "pub const ",
		"type ", "pub type ", "extern ", "import ",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func printHelp() {
	fmt.Fprintln(os.Stderr, `REPL commands:
  :help, :h     Show this help
  :quit, :q     Exit the REPL
  :clear        Clear accumulated definitions

Enter expressions or statements to evaluate them.
Enter top-level items (fn, struct, enum, etc.) to define them.
Definitions persist across evaluations until :clear.`)
}
