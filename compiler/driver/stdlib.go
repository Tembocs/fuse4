package driver

import (
	"os"
	"path/filepath"
	"strings"
)

// StdlibRoot is the default path to the stdlib directory, relative to the
// project root. Override via FUSE_STDLIB_ROOT environment variable.
func StdlibRoot() string {
	if env := os.Getenv("FUSE_STDLIB_ROOT"); env != "" {
		return env
	}
	return "stdlib"
}

// LoadStdlib discovers all .fuse files under the stdlib root and returns
// them as a module-path → source-bytes map suitable for Build().
func LoadStdlib(root string) (map[string][]byte, error) {
	sources := make(map[string][]byte)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".fuse") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Convert file path to module path:
		// stdlib/core/string.fuse → core.string
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
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
