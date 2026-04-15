package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Tembocs/fuse4/compiler/diagnostics"
	"github.com/Tembocs/fuse4/compiler/doc"
	"github.com/Tembocs/fuse4/compiler/driver"
	fusefmt "github.com/Tembocs/fuse4/compiler/fmt"
	"github.com/Tembocs/fuse4/compiler/repl"
	"github.com/Tembocs/fuse4/compiler/testrunner"
)

const version = "0.1.0-dev"

// Global state set early in main.
var useColor bool

func main() {
	// Detect color support before anything else.
	useColor = diagnostics.UseColor(os.Stderr)

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	// Check for global --no-color anywhere in args.
	filtered := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--no-color" {
			useColor = false
		} else {
			filtered = append(filtered, a)
		}
	}
	args = filtered

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
	case "version", "-v", "--version":
		fmt.Printf("fuse %s\n", version)
	case "help", "-h", "--help":
		printUsage()
	default:
		errMsg := fmt.Sprintf("unknown command %q", cmd)
		fmt.Fprintf(os.Stderr, "%s: %s\n\n", diagnostics.Colorize("error", diagnostics.ColorBoldRed, useColor), errMsg)
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

Global flags:
  --no-color   Disable colored output`)
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
		cliError("no input files")
		fmt.Fprintln(os.Stderr, "usage: fuse build [--output PATH] <file.fuse ...>")
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

	start := time.Now()
	result := driver.Build(driver.BuildOptions{
		Sources:    sources,
		OutputPath: outPath,
		Optimize:   *optimize,
		Debug:      *debug,
	})
	elapsed := time.Since(start)

	printDiagnostics(result.Errors, sourceMap, *jsonOut)
	if hasErrors(result.Errors) {
		printDiagSummary(result.Errors)
		os.Exit(1)
	}
	if result.OutputPath != "" {
		msg := fmt.Sprintf("built: %s", result.OutputPath)
		timing := fmt.Sprintf(" (%s)", formatDuration(elapsed))
		fmt.Fprintf(os.Stderr, "%s%s\n",
			diagnostics.Colorize(msg, diagnostics.ColorGreen, useColor),
			diagnostics.Colorize(timing, diagnostics.ColorDim, useColor))
	}
}

// --- run ---

func cmdRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "output diagnostics as JSON")
	optimize := fs.Bool("optimize", false, "enable optimizations")
	debug := fs.Bool("debug", false, "include debug info")
	fs.Parse(args)

	files := fs.Args()
	if len(files) == 0 {
		cliError("no input files")
		fmt.Fprintln(os.Stderr, "usage: fuse run [options] <file.fuse ...>")
		os.Exit(1)
	}

	sources, sourceMap := readSources(files)
	tmpPath := tempExePath()

	start := time.Now()
	result := driver.Build(driver.BuildOptions{
		Sources:    sources,
		OutputPath: tmpPath,
		Optimize:   *optimize,
		Debug:      *debug,
	})
	elapsed := time.Since(start)

	printDiagnostics(result.Errors, sourceMap, *jsonOut)
	if hasErrors(result.Errors) {
		printDiagSummary(result.Errors)
		os.Exit(1)
	}

	if result.OutputPath == "" {
		cliError("build produced no output")
		os.Exit(1)
	}
	defer os.Remove(result.OutputPath)

	// Print compile timing on stderr (dim, unobtrusive).
	timing := fmt.Sprintf("compiled in %s", formatDuration(elapsed))
	fmt.Fprintln(os.Stderr, diagnostics.Colorize(timing, diagnostics.ColorDim, useColor))

	// Execute the built binary, forwarding stdio.
	cmd := exec.Command(result.OutputPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		cliError(fmt.Sprintf("failed to execute: %s", err))
		os.Exit(1)
	}
}

// --- check ---

func cmdCheck(args []string) {
	fs := flag.NewFlagSet("check", flag.ExitOnError)
	jsonOut := fs.Bool("json", false, "output diagnostics as JSON")
	fs.Parse(args)

	files := fs.Args()
	if len(files) == 0 {
		cliError("no input files")
		fmt.Fprintln(os.Stderr, "usage: fuse check [options] <file.fuse ...>")
		os.Exit(1)
	}

	sources, sourceMap := readSources(files)

	start := time.Now()
	result := driver.Build(driver.BuildOptions{
		Sources: sources,
	})
	elapsed := time.Since(start)

	printDiagnostics(result.Errors, sourceMap, *jsonOut)
	if hasErrors(result.Errors) {
		printDiagSummary(result.Errors)
		os.Exit(1)
	}
	msg := "check: ok"
	timing := fmt.Sprintf(" (%s)", formatDuration(elapsed))
	fmt.Fprintf(os.Stderr, "%s%s\n",
		diagnostics.Colorize(msg, diagnostics.ColorGreen, useColor),
		diagnostics.Colorize(timing, diagnostics.ColorDim, useColor))
}

// --- fmt ---

func cmdFmt(args []string) {
	fs := flag.NewFlagSet("fmt", flag.ExitOnError)
	check := fs.Bool("check", false, "check if files are formatted (exit 1 if not)")
	write := fs.Bool("write", false, "write formatted output back to files")
	fs.Parse(args)

	files := fs.Args()
	if len(files) == 0 {
		cliError("no input files")
		fmt.Fprintln(os.Stderr, "usage: fuse fmt [--check] [--write] <file.fuse ...>")
		os.Exit(1)
	}

	exitCode := 0
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			cliError(fmt.Sprintf("cannot read %s: %s", path, err))
			os.Exit(1)
		}

		formatted, err := fusefmt.Format(path, data)
		if err != nil {
			cliError(fmt.Sprintf("format %s: %s", path, err))
			os.Exit(1)
		}

		if *check {
			if string(data) != string(formatted) {
				fmt.Fprintln(os.Stdout, path)
				exitCode = 1
			}
			continue
		}

		if *write || len(files) > 1 {
			if string(data) != string(formatted) {
				if err := os.WriteFile(path, formatted, 0644); err != nil {
					cliError(fmt.Sprintf("write %s: %s", path, err))
					os.Exit(1)
				}
				fmt.Fprintln(os.Stderr, diagnostics.Colorize("formatted: "+path, diagnostics.ColorGreen, useColor))
			}
			continue
		}

		// Single file, no flags: print to stdout.
		os.Stdout.Write(formatted)
	}
	os.Exit(exitCode)
}

// --- doc ---

func cmdDoc(args []string) {
	fs := flag.NewFlagSet("doc", flag.ExitOnError)
	all := fs.Bool("all", false, "include private items")
	output := fs.String("output", "", "write output to file instead of stdout")
	fs.Parse(args)

	files := fs.Args()
	if len(files) == 0 {
		cliError("no input files")
		fmt.Fprintln(os.Stderr, "usage: fuse doc [--all] [--output PATH] <file.fuse ...>")
		os.Exit(1)
	}

	var allMarkdown strings.Builder
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			cliError(fmt.Sprintf("cannot read %s: %s", path, err))
			os.Exit(1)
		}

		var items []doc.DocItem
		if *all {
			items = doc.Extract(data)
		} else {
			items = doc.ExtractPublic(data)
		}

		modName := fileToModulePath(path)
		md := doc.RenderMarkdown(items, modName)
		allMarkdown.WriteString(md)
	}

	if *output != "" {
		if err := os.WriteFile(*output, []byte(allMarkdown.String()), 0644); err != nil {
			cliError(fmt.Sprintf("write %s: %s", *output, err))
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, diagnostics.Colorize("wrote: "+*output, diagnostics.ColorGreen, useColor))
	} else {
		fmt.Print(allMarkdown.String())
	}
}

// --- test ---

func cmdTest(args []string) {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	verbose := fs.Bool("verbose", false, "show output from passing tests")
	fs.Parse(args)

	files := fs.Args()

	rtLib := driver.FindRuntimeLib()
	if rtLib == "" {
		cliError("runtime library not found (run 'make runtime' first)")
		os.Exit(1)
	}

	results, err := testrunner.Run(files, rtLib)
	if err != nil {
		cliError(err.Error())
		os.Exit(1)
	}

	testrunner.PrintReport(os.Stderr, results, useColor, *verbose)

	for _, r := range results {
		if !r.Passed {
			os.Exit(1)
		}
	}
}

// --- repl ---

func cmdRepl(args []string) {
	_ = args
	rtLib := driver.FindRuntimeLib()
	if rtLib == "" {
		cliError("runtime library not found (run 'make runtime' first)")
		os.Exit(1)
	}
	repl.Start(rtLib, useColor)
}

// --- helpers ---

func cliError(msg string) {
	label := diagnostics.Colorize("error", diagnostics.ColorBoldRed, useColor)
	fmt.Fprintf(os.Stderr, "%s: %s\n", label, msg)
}

func readSources(files []string) (map[string][]byte, map[string][]byte) {
	sources := make(map[string][]byte)
	sourceMap := make(map[string][]byte)
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			cliError(fmt.Sprintf("cannot read %s: %s", path, err))
			os.Exit(1)
		}
		modPath := fileToModulePath(path)
		sources[modPath] = data
		sourceMap[modPath+".fuse"] = data
	}
	return sources, sourceMap
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

func printDiagnostics(errs []diagnostics.Diagnostic, sourceMap map[string][]byte, jsonMode bool) {
	if len(errs) == 0 {
		return
	}
	if jsonMode {
		fmt.Fprint(os.Stderr, diagnostics.RenderJSON(errs))
	} else {
		fmt.Fprint(os.Stderr, diagnostics.RenderTextColor(errs, sourceMap, useColor))
	}
}

func printDiagSummary(errs []diagnostics.Diagnostic) {
	summary := diagnostics.DiagSummary(errs, useColor)
	if summary != "" {
		fmt.Fprintln(os.Stderr, summary)
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

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%d\u00b5s", d.Microseconds())
	}
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	return fmt.Sprintf("%.2fs", d.Seconds())
}
