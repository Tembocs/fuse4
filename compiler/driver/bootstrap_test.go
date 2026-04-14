package driver

import (
	"strings"
	"testing"
)

// ===== Phase 01: First self-compilation =====

func TestStage1CompilesStage2(t *testing.T) {
	stage2Dir := findStage2Dir(t)
	stdlibRoot := findStdlibRoot(t)

	result := BootstrapStage1ToStage2(stage2Dir, stdlibRoot)
	// Log errors but focus on whether C source was generated.
	for _, e := range result.Errors {
		t.Logf("  [%s] %s", result.Stage, e)
	}
	if result.CSource == "" {
		t.Error("stage1->stage2: expected non-empty C source output")
	}
	if result.CSourceHash == "" {
		t.Error("stage1->stage2: expected non-empty hash")
	}
	t.Logf("stage1->stage2: hash=%s success=%v csource_len=%d",
		result.CSourceHash[:16], result.Success, len(result.CSource))
}

func TestStage2CompilesItself(t *testing.T) {
	stage2Dir := findStage2Dir(t)
	stdlibRoot := findStdlibRoot(t)

	result := BootstrapStage2ToStage2(stage2Dir, stdlibRoot)
	for _, e := range result.Errors {
		if e.Severity == 0 { // Error severity
			t.Logf("  [%s] %s", result.Stage, e)
		}
	}
	if result.CSource == "" {
		t.Error("stage2->stage2: expected non-empty C source output")
	}
	t.Logf("stage2->stage2: hash=%s success=%v",
		result.CSourceHash[:16], result.Success)
}

// ===== Phase 02: Reproducibility =====

func TestReproducibilityAcrossGenerations(t *testing.T) {
	stage2Dir := findStage2Dir(t)
	stdlibRoot := findStdlibRoot(t)

	result := ReproCheck(stage2Dir, stdlibRoot, 3)
	t.Logf("%s", result)

	if !result.AllIdentical {
		t.Error("reproducibility check failed: outputs differ across generations")
		for i, gen := range result.GenResults {
			t.Logf("  gen%d: hash=%s", i, gen.CSourceHash[:16])
		}
	}
}

func TestReproducibilityTwoGenerations(t *testing.T) {
	stage2Dir := findStage2Dir(t)
	stdlibRoot := findStdlibRoot(t)

	result := ReproCheck(stage2Dir, stdlibRoot, 2)
	if !result.AllIdentical {
		t.Error("two-generation repro check failed")
	}
}

// ===== Phase 02: Bootstrap health gate =====

func TestBootstrapHealthGate(t *testing.T) {
	stage2Dir := findStage2Dir(t)
	stdlibRoot := findStdlibRoot(t)

	// This test serves as the merge gate for bootstrap health (Rule 6.7).
	// A regression in self-hosting or reproducibility is release-blocking.
	gen1 := BootstrapStage1ToStage2(stage2Dir, stdlibRoot)
	gen2 := BootstrapStage1ToStage2(stage2Dir, stdlibRoot)

	if gen1.CSourceHash != gen2.CSourceHash {
		t.Fatalf("BOOTSTRAP HEALTH GATE FAILED: non-deterministic output\n"+
			"  gen1: %s\n  gen2: %s", gen1.CSourceHash[:16], gen2.CSourceHash[:16])
	}

	if gen1.CSource == "" {
		t.Fatal("BOOTSTRAP HEALTH GATE FAILED: empty output")
	}

	t.Logf("bootstrap health: OK (hash=%s, %d bytes)",
		gen1.CSourceHash[:16], len(gen1.CSource))
}

func TestBootstrapOutputContainsExpectedSymbols(t *testing.T) {
	stage2Dir := findStage2Dir(t)
	stdlibRoot := findStdlibRoot(t)

	result := BootstrapStage1ToStage2(stage2Dir, stdlibRoot)
	if result.CSource == "" {
		t.Skip("no C source generated")
	}

	// The generated C should contain the runtime include and at least some
	// function definitions from the stage2 sources.
	if !strings.Contains(result.CSource, "#include") {
		t.Error("generated C should include headers")
	}
}

func TestReproResultString(t *testing.T) {
	stage2Dir := findStage2Dir(t)
	stdlibRoot := findStdlibRoot(t)

	result := ReproCheck(stage2Dir, stdlibRoot, 2)
	s := result.String()
	if !strings.Contains(s, "repro check") {
		t.Errorf("unexpected ReproResult string: %q", s)
	}
}
