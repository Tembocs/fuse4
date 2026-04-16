package driver

import (
	"os"
	"path/filepath"
	"strings"
)

// StdlibRoot returns the path to the stdlib directory.
// Search order:
//  1. FUSE_STDLIB_ROOT environment variable (explicit override)
//  2. Relative to the executable (packaged distribution: <exe>/../stdlib/)
//  3. Relative to CWD (development: stdlib/)
func StdlibRoot() string {
	if env := os.Getenv("FUSE_STDLIB_ROOT"); env != "" {
		return env
	}

	// Check relative to the executable (packaged layout).
	if exeRoot := exeDistRoot(); exeRoot != "" {
		c := filepath.Join(exeRoot, "stdlib")
		if info, err := os.Stat(c); err == nil && info.IsDir() {
			return c
		}
	}

	return "stdlib"
}

// LoadStdlib discovers all .fuse files under the stdlib root and returns
// them as a module-path → source-bytes map suitable for Build().
func LoadStdlib(root string) (map[string][]byte, error) {
	return loadStdlibFiltered(root, nil)
}

// LoadStdlibCore loads only the `core/` tier of the standard library. This
// is the default for the driver's auto-load path: the language guide
// (§11.4 "Module loading") says `core/` is auto-loaded while `full/` and
// `ext/` are loaded on demand. Loading everything would pull ext modules
// (argparse, toml, yaml, …) into every user program's compilation unit
// and surface codegen errors for types the user never referenced.
func LoadStdlibCore(root string) (map[string][]byte, error) {
	return loadStdlibFiltered(root, func(rel string) bool {
		return strings.HasPrefix(rel, "core"+string(filepath.Separator)) ||
			strings.HasPrefix(rel, "core/")
	})
}

// loadStdlibFiltered walks `root` and returns a module-path → bytes map.
// When `keep` is non-nil it is called with the path relative to `root`
// (using the local OS separator) and the entry is included only when
// keep returns true.
func loadStdlibFiltered(root string, keep func(rel string) bool) (map[string][]byte, error) {
	sources := make(map[string][]byte)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".fuse") {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if keep != nil && !keep(rel) {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Convert file path to module path:
		// stdlib/core/string.fuse → core.string
		modPath := filePathToModulePath(rel)
		sources[modPath] = data
		return nil
	})

	return sources, err
}

func filePathToModulePath(rel string) string {
	// Remove .fuse extension.
	if strings.HasSuffix(rel, ".fuse") {
		rel = rel[:len(rel)-5]
	}
	// Replace path separators with dots.
	rel = strings.ReplaceAll(rel, string(filepath.Separator), ".")
	rel = strings.ReplaceAll(rel, "/", ".")
	return rel
}

// DocCoverage checks that all public items in the stdlib have doc comments.
// Returns a list of undocumented public items as "module.Name" strings.
func DocCoverage(sources map[string][]byte) []string {
	var undocumented []string
	for modPath, src := range sources {
		lines := strings.Split(string(src), "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)
			if !strings.HasPrefix(trimmed, "pub ") {
				continue
			}
			// Check if the preceding line is a doc comment.
			if i > 0 {
				prev := strings.TrimSpace(lines[i-1])
				if strings.HasPrefix(prev, "///") || strings.HasPrefix(prev, "*/") {
					continue
				}
			}
			// Extract item name.
			name := extractItemName(trimmed)
			if name != "" {
				undocumented = append(undocumented, modPath+"."+name)
			}
		}
	}
	return undocumented
}

func extractItemName(line string) string {
	// "pub fn foo(..." → "foo"
	// "pub struct Foo {" → "Foo"
	// "pub trait Foo {" → "Foo"
	parts := strings.Fields(line)
	if len(parts) < 3 {
		return ""
	}
	// parts[0] = "pub", parts[1] = keyword, parts[2] = name(...)
	name := parts[2]
	// Strip trailing punctuation.
	for _, ch := range []byte{'(', '{', '[', ':', '<'} {
		if idx := strings.IndexByte(name, ch); idx >= 0 {
			name = name[:idx]
		}
	}
	return name
}
