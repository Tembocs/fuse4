package driver

import (
	"strings"
	"testing"
)

func TestHostedStdlibDiscovery(t *testing.T) {
	root := findStdlibRoot(t)
	sources, err := LoadStdlib(root)
	if err != nil {
		t.Fatalf("LoadStdlib: %s", err)
	}

	expected := []string{
		"full.chan",
		"full.env",
		"full.http",
		"full.io",
		"full.json",
		"full.net",
		"full.os",
		"full.path",
		"full.process",
		"full.random",
		"full.shared",
		"full.simd",
		"full.sys",
		"full.time",
		"full.timer",
	}
	for _, mod := range expected {
		if _, ok := sources[mod]; !ok {
			t.Errorf("missing expected hosted module: %s", mod)
		}
	}
}

func TestHostedModulesDoNotLeakIntoCore(t *testing.T) {
	root := findStdlibRoot(t)
	sources, err := LoadStdlib(root)
	if err != nil {
		t.Fatalf("LoadStdlib: %s", err)
	}

	// Core modules must not import from full.*.
	for modPath, src := range sources {
		if !strings.HasPrefix(modPath, "core.") {
			continue
		}
		lines := strings.Split(string(src), "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "import full.") {
				t.Errorf("core module %s imports from full tier at line %d: %s",
					modPath, i+1, trimmed)
			}
		}
	}
}

func TestHostedModulesCanImportCore(t *testing.T) {
	root := findStdlibRoot(t)
	sources, err := LoadStdlib(root)
	if err != nil {
		t.Fatalf("LoadStdlib: %s", err)
	}

	// Verify at least one full.* module imports from core.
	foundCoreImport := false
	for modPath, src := range sources {
		if !strings.HasPrefix(modPath, "full.") {
			continue
		}
		if strings.Contains(string(src), "import core.") {
			foundCoreImport = true
			break
		}
	}
	if !foundCoreImport {
		t.Error("expected at least one hosted module to import from core")
	}
}

func TestHostedDocCoverage(t *testing.T) {
	root := findStdlibRoot(t)
	sources, err := LoadStdlib(root)
	if err != nil {
		t.Fatalf("LoadStdlib: %s", err)
	}

	// Filter to full.* modules only.
	hosted := make(map[string][]byte)
	for modPath, src := range sources {
		if strings.HasPrefix(modPath, "full.") {
			hosted[modPath] = src
		}
	}

	undocumented := DocCoverage(hosted)
	for _, item := range undocumented {
		t.Logf("undocumented hosted API: %s", item)
	}
	t.Logf("hosted doc coverage: %d undocumented public items across %d modules",
		len(undocumented), len(hosted))
}

func TestTotalStdlibModuleCount(t *testing.T) {
	root := findStdlibRoot(t)
	sources, err := LoadStdlib(root)
	if err != nil {
		t.Fatalf("LoadStdlib: %s", err)
	}

	coreCount := 0
	fullCount := 0
	for modPath := range sources {
		if strings.HasPrefix(modPath, "core.") {
			coreCount++
		} else if strings.HasPrefix(modPath, "full.") {
			fullCount++
		}
	}

	if coreCount < 10 {
		t.Errorf("expected >= 10 core modules, got %d", coreCount)
	}
	if fullCount < 6 {
		t.Errorf("expected >= 6 full modules, got %d", fullCount)
	}
	t.Logf("stdlib: %d core + %d full = %d total modules", coreCount, fullCount, coreCount+fullCount)
}
