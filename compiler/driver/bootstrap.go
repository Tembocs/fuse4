package driver

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Tembocs/fuse4/compiler/diagnostics"
)

// BootstrapResult holds the outcome of a bootstrap compilation attempt.
type BootstrapResult struct {
	Stage       string // "stage1->stage2", "stage2->stage2"
	CSource     string // generated C for this stage
	CSourceHash string // SHA-256 of the generated C
	Errors      []diagnostics.Diagnostic
	Success     bool
}

// BootstrapStage1ToStage2 compiles the stage2 Fuse sources using the stage1
// Go compiler pipeline. Returns the generated C source and diagnostics.
func BootstrapStage1ToStage2(stage2Dir, stdlibRoot string) *BootstrapResult {
	result := &BootstrapResult{Stage: "stage1->stage2"}

	// Load stage2 sources.
	stage2Sources, err := loadFuseSources(stage2Dir)
	if err != nil {
		result.Errors = append(result.Errors, diagnostics.Errorf(
			diagnostics.Span{}, "load stage2: %s", err))
		return result
	}

	// Load stdlib sources.
	stdlibSources, err := LoadStdlib(stdlibRoot)
	if err != nil {
		result.Errors = append(result.Errors, diagnostics.Errorf(
			diagnostics.Span{}, "load stdlib: %s", err))
		return result
	}

	// Merge: stage2 modules + stdlib modules.
	allSources := make(map[string][]byte)
	for k, v := range stdlibSources {
		allSources[k] = v
	}
	for k, v := range stage2Sources {
		allSources["stage2."+k] = v
	}

	// Run the stage1 compiler pipeline (check only — no native compile).
	buildResult := Build(BuildOptions{Sources: allSources})
	result.Errors = append(result.Errors, buildResult.Errors...)
	result.CSource = buildResult.CSource
	result.CSourceHash = hashString(buildResult.CSource)
	result.Success = !hasErrorDiags(buildResult.Errors)

	return result
}

// BootstrapStage2ToStage2 simulates stage2 compiling itself.
// In the current bootstrap state, this feeds the same sources through
// stage1 again (since stage2 is not yet a real executable) and verifies
// the output is identical — proving deterministic compilation.
func BootstrapStage2ToStage2(stage2Dir, stdlibRoot string) *BootstrapResult {
	result := &BootstrapResult{Stage: "stage2->stage2"}

	// In the bootstrap model, "stage2 compiles itself" means:
	// compile the stage2 sources a second time through stage1 and
	// verify byte-identical output (determinism gate).
	gen1 := BootstrapStage1ToStage2(stage2Dir, stdlibRoot)
	gen2 := BootstrapStage1ToStage2(stage2Dir, stdlibRoot)

	result.CSource = gen2.CSource
	result.CSourceHash = gen2.CSourceHash
	result.Errors = append(result.Errors, gen1.Errors...)
	result.Errors = append(result.Errors, gen2.Errors...)

	if gen1.CSourceHash != gen2.CSourceHash {
		result.Errors = append(result.Errors, diagnostics.Errorf(
			diagnostics.Span{},
			"reproducibility failure: gen1 hash %s != gen2 hash %s",
			gen1.CSourceHash, gen2.CSourceHash))
		result.Success = false
	} else {
		result.Success = gen1.Success && gen2.Success
	}

	return result
}

// ReproCheck compares N generations of bootstrap compilation and reports
// whether all generations produce identical output (Rule 7.1: same input, same bytes).
func ReproCheck(stage2Dir, stdlibRoot string, generations int) *ReproResult {
	result := &ReproResult{Generations: generations}

	var hashes []string
	for i := 0; i < generations; i++ {
		gen := BootstrapStage1ToStage2(stage2Dir, stdlibRoot)
		result.GenResults = append(result.GenResults, gen)
		hashes = append(hashes, gen.CSourceHash)
		if !gen.Success {
			result.AllSucceeded = false
		}
	}

	result.AllSucceeded = true
	result.AllIdentical = true
	for i := 1; i < len(hashes); i++ {
		if hashes[i] != hashes[0] {
			result.AllIdentical = false
			break
		}
	}
	for _, gen := range result.GenResults {
		if !gen.Success {
			result.AllSucceeded = false
		}
	}
	result.Hash = hashes[0]

	return result
}

// ReproResult holds the outcome of a multi-generation reproducibility check.
type ReproResult struct {
	Generations  int
	GenResults   []*BootstrapResult
	AllSucceeded bool
	AllIdentical bool
	Hash         string
}

func (r *ReproResult) String() string {
	status := "PASS"
	if !r.AllSucceeded {
		status = "FAIL (compilation errors)"
	} else if !r.AllIdentical {
		status = "FAIL (non-deterministic)"
	}
	return fmt.Sprintf("repro check: %d generations, %s, hash=%s",
		r.Generations, status, r.Hash[:16])
}

// --- helpers ---

func loadFuseSources(dir string) (map[string][]byte, error) {
	sources := make(map[string][]byte)
	names, err := readDirNames(dir)
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		if !strings.HasSuffix(name, ".fuse") {
			continue
		}
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		modPath := strings.TrimSuffix(name, ".fuse")
		sources[modPath] = data
	}
	return sources, nil
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", h)
}

func hasErrorDiags(diags []diagnostics.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == diagnostics.Error {
			return true
		}
	}
	return false
}

func readDirNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names, nil
}
