// Package doc owns documentation extraction and rendering for Fuse source.
package doc

import (
	"strings"
)

// DocItem represents a documentable item extracted from source.
type DocItem struct {
	Kind      string   // "fn", "struct", "enum", "trait", "impl", "const", "type", "extern fn"
	Name      string   // item name
	Public    bool     // true if declared with pub
	Signature string   // declaration line(s) up to the body
	DocLines  []string // stripped /// comment lines preceding the item
}

// Extract scans source bytes and returns all documentable items with
// their associated /// doc comments.
func Extract(src []byte) []DocItem {
	lines := strings.Split(string(src), "\n")
	var items []DocItem
	var docBlock []string

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])

		// Accumulate doc comment blocks.
		if strings.HasPrefix(trimmed, "///") {
			text := strings.TrimPrefix(trimmed, "///")
			if len(text) > 0 && text[0] == ' ' {
				text = text[1:]
			}
			docBlock = append(docBlock, text)
			continue
		}

		// Check if this line starts a documentable item.
		if item, ok := parseItemLine(trimmed, lines, i); ok {
			item.DocLines = docBlock
			items = append(items, item)
		}

		// Reset doc block for non-comment, non-blank lines.
		if trimmed != "" {
			docBlock = nil
		}
	}

	return items
}

// ExtractPublic returns only public items.
func ExtractPublic(src []byte) []DocItem {
	all := Extract(src)
	var pub []DocItem
	for _, item := range all {
		if item.Public {
			pub = append(pub, item)
		}
	}
	return pub
}

// parseItemLine checks if a line declares a documentable item.
func parseItemLine(line string, lines []string, idx int) (DocItem, bool) {
	pub := false
	rest := line

	if strings.HasPrefix(rest, "pub ") {
		pub = true
		rest = strings.TrimPrefix(rest, "pub ")
	}

	kind, name, sig := "", "", ""

	switch {
	case strings.HasPrefix(rest, "fn "):
		kind = "fn"
		name = extractName(rest[3:])
		sig = extractSignature(lines, idx)
	case strings.HasPrefix(rest, "struct "):
		kind = "struct"
		name = extractName(rest[7:])
		sig = extractSignature(lines, idx)
	case strings.HasPrefix(rest, "enum "):
		kind = "enum"
		name = extractName(rest[5:])
		sig = extractSignature(lines, idx)
	case strings.HasPrefix(rest, "trait "):
		kind = "trait"
		name = extractName(rest[6:])
		sig = extractSignature(lines, idx)
	case strings.HasPrefix(rest, "impl "):
		kind = "impl"
		name = extractName(rest[5:])
		sig = extractSignature(lines, idx)
	case strings.HasPrefix(rest, "const "):
		kind = "const"
		name = extractName(rest[6:])
		sig = strings.TrimSpace(line)
	case strings.HasPrefix(rest, "type "):
		kind = "type"
		name = extractName(rest[5:])
		sig = strings.TrimSpace(line)
	case strings.HasPrefix(rest, "extern fn "):
		kind = "extern fn"
		name = extractName(rest[10:])
		sig = extractSignature(lines, idx)
	default:
		return DocItem{}, false
	}

	if name == "" {
		return DocItem{}, false
	}

	return DocItem{
		Kind:      kind,
		Name:      name,
		Public:    pub,
		Signature: sig,
	}, true
}

// extractName pulls the identifier from the start of s, stopping at
// punctuation like (, {, [, <, :, or whitespace.
func extractName(s string) string {
	s = strings.TrimSpace(s)
	end := 0
	for end < len(s) {
		ch := s[end]
		if ch == '(' || ch == '{' || ch == '[' || ch == '<' || ch == ':' || ch == ' ' || ch == '\t' {
			break
		}
		end++
	}
	return s[:end]
}

// extractSignature gathers the declaration line(s) up to the first '{' or ';'.
func extractSignature(lines []string, start int) string {
	var sig strings.Builder
	for i := start; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		sig.WriteString(trimmed)
		if strings.Contains(trimmed, "{") {
			// Trim from the { onward.
			result := sig.String()
			if idx := strings.Index(result, "{"); idx >= 0 {
				result = strings.TrimSpace(result[:idx])
			}
			return result
		}
		if strings.HasSuffix(trimmed, ";") {
			return sig.String()
		}
		sig.WriteByte(' ')
	}
	return strings.TrimSpace(sig.String())
}
