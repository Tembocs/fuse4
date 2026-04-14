package cc

import (
	"strings"
	"testing"
)

func TestDetectFindsCompiler(t *testing.T) {
	tc, err := Detect()
	if err != nil {
		t.Skipf("no C compiler available: %s", err)
	}
	if tc.Path == "" {
		t.Error("path should not be empty")
	}
	if tc.Name == "" {
		t.Error("name should not be empty")
	}
	t.Logf("detected: %s (%s) at %s", tc.Name, tc.Version, tc.Path)
}

func TestCompileArgsOptimized(t *testing.T) {
	tc := &Toolchain{Path: "/usr/bin/gcc", Name: "gcc"}
	args := tc.CompileArgs("input.c", BuildConfig{Optimize: true, OutputObj: "input.o"})
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-O2") {
		t.Errorf("expected -O2 in args: %s", joined)
	}
	if !strings.Contains(joined, "-o input.o") {
		t.Errorf("expected output flag: %s", joined)
	}
	if !strings.Contains(joined, "input.c") {
		t.Errorf("expected source file: %s", joined)
	}
}

func TestCompileArgsDebug(t *testing.T) {
	tc := &Toolchain{Path: "/usr/bin/gcc", Name: "gcc"}
	args := tc.CompileArgs("input.c", BuildConfig{Debug: true})
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "-g") {
		t.Errorf("expected -g in args: %s", joined)
	}
	if !strings.Contains(joined, "-O0") {
		t.Errorf("expected -O0 in args: %s", joined)
	}
}

func TestLinkArgsIncludesRuntime(t *testing.T) {
	tc := &Toolchain{Path: "/usr/bin/gcc", Name: "gcc"}
	args := tc.LinkArgs([]string{"a.o", "b.o"}, "/path/to/libfuse_rt.a", BuildConfig{OutputExe: "out"})
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "a.o") || !strings.Contains(joined, "b.o") {
		t.Errorf("expected objects: %s", joined)
	}
	if !strings.Contains(joined, "libfuse_rt.a") {
		t.Errorf("expected runtime lib: %s", joined)
	}
	if !strings.Contains(joined, "-o out") {
		t.Errorf("expected output: %s", joined)
	}
}

func TestLinkArgsNoRuntime(t *testing.T) {
	tc := &Toolchain{Path: "/usr/bin/gcc", Name: "gcc"}
	args := tc.LinkArgs([]string{"a.o"}, "", BuildConfig{OutputExe: "out"})
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "libfuse_rt") {
		t.Error("should not include runtime when empty")
	}
}
