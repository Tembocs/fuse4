package main

import (
	"testing"
)

func TestFileToModulePath(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"main.fuse", "main"},
		{"core/list.fuse", "core.list"},
		{"core\\list.fuse", "core.list"},
		{"src/core/list.fuse", "src.core.list"},
		{"noext", "noext"},
	}
	for _, tc := range cases {
		got := fileToModulePath(tc.input)
		if got != tc.want {
			t.Errorf("fileToModulePath(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestIsWindows(t *testing.T) {
	// Just verify it doesn't panic and returns a bool.
	_ = isWindows()
}

func TestTempExePath(t *testing.T) {
	p := tempExePath()
	if p == "" {
		t.Error("tempExePath should not be empty")
	}
}

func TestPrintUsageDoesNotPanic(t *testing.T) {
	// Just verify it doesn't panic.
	printUsage()
}
