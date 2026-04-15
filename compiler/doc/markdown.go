package doc

import (
	"fmt"
	"strings"
)

// RenderMarkdown renders extracted doc items as markdown.
func RenderMarkdown(items []DocItem, moduleName string) string {
	var b strings.Builder

	if moduleName != "" {
		fmt.Fprintf(&b, "# %s\n\n", moduleName)
	}

	// Group items by kind.
	groups := []struct {
		Title string
		Kinds []string
	}{
		{"Functions", []string{"fn"}},
		{"Structs", []string{"struct"}},
		{"Enums", []string{"enum"}},
		{"Traits", []string{"trait"}},
		{"Implementations", []string{"impl"}},
		{"Constants", []string{"const"}},
		{"Type Aliases", []string{"type"}},
		{"Extern Functions", []string{"extern fn"}},
	}

	for _, g := range groups {
		var matching []DocItem
		for _, item := range items {
			for _, k := range g.Kinds {
				if item.Kind == k {
					matching = append(matching, item)
					break
				}
			}
		}
		if len(matching) == 0 {
			continue
		}

		fmt.Fprintf(&b, "## %s\n\n", g.Title)
		for _, item := range matching {
			fmt.Fprintf(&b, "### %s\n\n", item.Name)
			if item.Signature != "" {
				fmt.Fprintf(&b, "```fuse\n%s\n```\n\n", item.Signature)
			}
			if len(item.DocLines) > 0 {
				for _, line := range item.DocLines {
					b.WriteString(line)
					b.WriteByte('\n')
				}
				b.WriteByte('\n')
			}
		}
	}

	return b.String()
}
