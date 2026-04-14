// Package cc owns C compiler detection, invocation, and backend-toolchain
// interaction.
package cc

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Toolchain represents a detected C compiler.
type Toolchain struct {
	Path    string // absolute path to the compiler binary
	Name    string // "gcc", "clang", "cl"
	Version string
}

// Detect finds a usable C11 compiler on the host system.
// Search order: CC env var, gcc, clang, cc.
func Detect() (*Toolchain, error) {
	candidates := []string{"gcc", "clang", "cc"}
	if runtime.GOOS == "windows" {
		candidates = []string{"gcc", "clang", "cl", "cc"}
	}

	for _, name := range candidates {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		version := probeVersion(path)
		return &Toolchain{Path: path, Name: name, Version: version}, nil
	}
	return nil, fmt.Errorf("no C compiler found; install gcc or clang and ensure it is on PATH")
}

func probeVersion(path string) string {
	out, err := exec.Command(path, "--version").CombinedOutput()
	if err != nil {
		return "unknown"
	}
	line := strings.SplitN(string(out), "\n", 2)[0]
	return strings.TrimSpace(line)
}

// BuildConfig holds compilation and linking options.
type BuildConfig struct {
	Optimize  bool
	Debug     bool
	OutputExe string // output executable path
	OutputObj string // output object path (for compile-only)
}

// CompileArgs returns the arguments to compile a C source file to an object.
func (tc *Toolchain) CompileArgs(src string, cfg BuildConfig) []string {
	args := []string{"-c", "-std=c11", "-Wall"}
	if cfg.Optimize {
		args = append(args, "-O2")
	} else {
		args = append(args, "-O0")
	}
	if cfg.Debug {
		args = append(args, "-g")
	}
	if cfg.OutputObj != "" {
		args = append(args, "-o", cfg.OutputObj)
	}
	args = append(args, src)
	return args
}

// LinkArgs returns the arguments to link objects and the runtime into an executable.
func (tc *Toolchain) LinkArgs(objects []string, runtimeLib string, cfg BuildConfig) []string {
	args := make([]string, 0, len(objects)+6)
	args = append(args, objects...)
	if runtimeLib != "" {
		args = append(args, runtimeLib)
	}
	if cfg.OutputExe != "" {
		args = append(args, "-o", cfg.OutputExe)
	}
	// Platform link flags.
	if runtime.GOOS != "windows" {
		args = append(args, "-lpthread")
	}
	return args
}

// Run executes the C compiler with the given arguments.
func (tc *Toolchain) Run(args []string) (string, error) {
	cmd := exec.Command(tc.Path, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// Compile compiles a C source file to an object file.
func (tc *Toolchain) Compile(src string, cfg BuildConfig) error {
	args := tc.CompileArgs(src, cfg)
	output, err := tc.Run(args)
	if err != nil {
		return fmt.Errorf("compile %s: %s\n%s", src, err, output)
	}
	return nil
}

// Link links object files and the runtime into an executable.
func (tc *Toolchain) Link(objects []string, runtimeLib string, cfg BuildConfig) error {
	args := tc.LinkArgs(objects, runtimeLib, cfg)
	output, err := tc.Run(args)
	if err != nil {
		return fmt.Errorf("link: %s\n%s", err, output)
	}
	return nil
}
