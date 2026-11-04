package macros

import (
	"fmt"
	"html"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

// Members is the block macro sysml-members::[], replaced by a bullet list of
// the owned members of the page's element (MC-12).
const Members = "sysml-members"

// MembersRole is the role of the list, for styling.
const MembersRole = "sysml-members"

func registerMembers(g *asciidoctor.ExtensionGroup, t *tree.Tree) {
	g.BlockMacro(Members, asciidoctor.BlockMacroProcessorFunc(
		func(parent asciidoctor.StructuralNode, _ string, _ map[string]any) (asciidoctor.StructuralNode, error) {
			n, err := pageElement(parent, t)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", Members, err)
			}
			if len(n.Members) == 0 {
				return nil, nil // nothing to list; the macro line is dropped
			}
			list, err := asciidoctor.CreateList(parent, "ulist", asciidoctor.Attributes{"role": MembersRole})
			if err != nil {
				return nil, fmt.Errorf("%s: %w", Members, err)
			}
			for _, m := range n.Members {
				item, err := asciidoctor.CreateListItem(list, memberText(n, m, t))
				if err != nil {
					return nil, fmt.Errorf("%s: %w", Members, err)
				}
				list.AddItem(item)
			}
			return list, nil
		}), nil)
}

// memberText is a list item's AsciiDoc: the member's name, linked to its
// page when it has one.
func memberText(page *tree.Node, m tree.Member, t *tree.Tree) string {
	name := html.EscapeString(m.Title)
	if target := t.ByID[m.ID]; target != nil {
		return passthrough(`<a href="` + tree.Href(page, target) + `">` + name + `</a>`)
	}
	return passthrough(name)
}
