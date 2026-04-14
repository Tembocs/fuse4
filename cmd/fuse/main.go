package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/driver"
)

const version = "0.1.0-dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "build":
		cmdBuild(args)
	case "run":
		cmdRun(args)
	case "check":
		cmdCheck(args)
	case "fmt":
		cmdFmt(args)
	case "doc":
		cmdDoc(args)
	case "test":
		cmdTest(args)
	case "repl":
		cmdRepl(args)
	case "version":
		fmt.Printf("fuse %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "fuse: unknown command %q\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`Usage: fuse <command> [options] [files...]

Commands:
  build     Compile Fuse source files into an executable
  run       Build and execute a Fuse program
  check     Type-check without compiling
  fmt       Format Fuse source files
  doc       Generate documentation
  test      Run tests
  repl      Start interactive REPL
  version   Print compiler version
  help      Show this message

Common flags:
  --json       Output diagnostics as JSON
  --optimize   Enable optimizations
  --debug      Include debug info
  --output     Output file path`)
}

// --- build ---

func cmdBuild(args []string) {
	fs := flag.NewFlagSet("build", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "output diagnostics as JSON")
	optimize := fs.Bool("optimize", false, "enable optimizations")
	debug := fs.Bool("debug", false, "include debug info")
	output := fs.String("output", "", "output file path")
	fs.Parse(args)

	files := fs.Args()
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "fuse build: no input files")
		os.Exit(1)
	}

	sources, sourceMap := readSources(files)
	outPath := *output
	if outPath == "" {
		outPath = "a.out"
		if isWindows() {
			outPath = "a.exe"
		}
	}

	result := driver.Build(driver.BuildOptions{
		Sources:    sources,
		OutputPath: outPath,
		Optimize:   *optimize,
		Debug:      *debug,
	})

	printDiagnostics(result.Errors, sourceMap, *jsonOut)
	if hasErrors(result.Errors) {
		os.Exit(1)
	}
	if result.OutputPath != "" {
		fmt.Fprintf(os.Stderr, "built: %s\n", result.OutputPath)
	}
}

// --- run ---

func cmdRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "output diagnostics as JSON")
	fs.Parse(args)

	files := fs.Args()
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "fuse run: no input files")
		os.Exit(1)
	}

	sources, sourceMap := readSources(files)
	result := driver.Build(driver.BuildOptions{
		Sources:    sources,
		OutputPath: tempExePath(),
	})

	printDiagnostics(result.Errors, sourceMap, *jsonOut)
	if hasErrors(result.Errors) {
		os.Exit(1)
	}

	if result.OutputPath != "" {
		// TODO: execute the built binary
		fmt.Fprintf(os.Stderr, "fuse run: execute %s (not yet wired)\n", result.OutputPath)
	}
}

// --- check ---

func cmdCheck(args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "output diagnostics as JSON")
	fs.Parse(args)

	files := fs.Args()
	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "fuse check: no input files")
		os.Exit(1)
	}

	sources, sourceMap := readSources(files)
	result := driver.Build(driver.BuildOptions{
		Sources: sources,
		// No OutputPath — check only, no compile+link.
	})

	printDiagnostics(result.Errors, sourceMap, *jsonOut)
	if hasErrors(result.Errors) {
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "check: ok")
}

// --- fmt ---

func cmdFmt(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "fuse fmt: no input files")
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "fuse fmt: formatter not yet implemented")
	os.Exit(1)
}

// --- doc ---

func cmdDoc(args []string) {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "fuse doc: no input files")
		os.Exit(1)
	}
	fmt.Fprintln(os.Stderr, "fuse doc: documentation generator not yet implemented")
	os.Exit(1)
}

// --- test ---

func cmdTest(args []string) {
	fmt.Fprintln(os.Stderr, "fuse test: test runner not yet implemented")
	_ = args
	os.Exit(1)
}

// --- repl ---

func cmdRepl(args []string) {
	fmt.Fprintln(os.Stderr, "fuse repl: REPL not yet implemented")
	_ = args
	os.Exit(1)
}

// --- helpers ---

func readSources(files []string) (map[string][]byte, map[string][]byte) {
	sources := make(map[string][]byte)
	sourceMap := make(map[string][]byte)
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fuse: cannot read %s: %s\n", path, err)
			os.Exit(1)
		}
		modPath := fileToModulePath(path)
		sources[modPath] = data
		sourceMap[modPath+".fuse"] = data
	}
	return sources, sourceMap
}

func fileToModulePath(path string) string {
	// Strip .fuse extension and replace separators with dots.
	p := path
	if strings.HasSuffix(p, ".fuse") {
		p = p[:len(p)-5]
	}
	p = strings.ReplaceAll(p, "/", ".")
	p = strings.ReplaceAll(p, "\\", ".")
	return p
}

func printDiagnostics(errs []diagnostics.Diagnostic, sourceMap map[string][]byte, jsonMode bool) {
	if len(errs) == 0 {
		return
	}
	if jsonMode {
		fmt.Fprint(os.Stderr, diagnostics.RenderJSON(errs))
	} else {
		fmt.Fprint(os.Stderr, diagnostics.RenderText(errs, sourceMap))
	}
}

func hasErrors(errs []diagnostics.Diagnostic) bool {
	for _, e := range errs {
		if e.Severity == diagnostics.Error {
			return true
		}
	}
	return false
}

func isWindows() bool {
	return os.PathSeparator == '\\'
}

func tempExePath() string {
	if isWindows() {
		return os.TempDir() + "\\fuse_run_tmp.exe"
	}
	return os.TempDir() + "/fuse_run_tmp"
}
