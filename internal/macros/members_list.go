package macros

import (
	"errors"
	"fmt"
	"html"
	"strings"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

// MembersList is the block macro sysml-members-list::[<element or kind>][],
// replaced by a bullet list of the owned members of the page's element
// (MC-12). Without a target every member is listed, each with its types in
// parentheses; with an element, only the members typed by, subclassing or
// subsetting it; with an element kind (partUsage), only members of that
// kind.
const MembersList = "sysml-members-list"

// MembersListRole is the role of the list, for styling.
const MembersListRole = "sysml-members-list"

func registerMembersList(g *asciidoctor.ExtensionGroup, t *tree.Tree) {
	names := newNameIndex(t.Model)
	kinds := newKindIndex(t.Model)
	g.BlockMacro(MembersList, asciidoctor.BlockMacroProcessorFunc(
		func(parent asciidoctor.StructuralNode, target string, _ map[string]any) (asciidoctor.StructuralNode, error) {
			n, err := pageElement(parent, t)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", MembersList, err)
			}
			members := n.Members
			if target != "" {
				keep, err := membersFilter(target, n, t, names, kinds)
				if err != nil {
					return nil, fmt.Errorf("%s::%s[] on the page of %s: %w", MembersList, target, n.QualifiedName, err)
				}
				members = nil
				for _, m := range n.Members {
					if keep(m) {
						members = append(members, m)
					}
				}
			}
			if len(members) == 0 {
				return nil, nil // nothing to list; the macro line is dropped
			}
			list, err := asciidoctor.CreateList(parent, "ulist", asciidoctor.Attributes{"role": MembersListRole})
			if err != nil {
				return nil, fmt.Errorf("%s: %w", MembersList, err)
			}
			for _, m := range members {
				item, err := asciidoctor.CreateListItem(list, memberText(n, m, t, target == ""))
				if err != nil {
					return nil, fmt.Errorf("%s: %w", MembersList, err)
				}
				list.AddItem(item)
			}
			return list, nil
		}), nil)
}

// memberText is a list item's AsciiDoc: the member's name, linked to its
// page when it has one, and its types in parentheses when withTypes is set
// and it has any: mass (Real).
func memberText(page *tree.Node, m tree.Member, t *tree.Tree, withTypes bool) string {
	text := html.EscapeString(m.Name)
	if target := t.ByID[m.ID]; target != nil {
		text = `<a href="` + tree.Href(page, target) + `">` + text + `</a>`
	}
	if withTypes && len(m.Types) > 0 {
		text += " (" + html.EscapeString(strings.Join(m.Types, ", ")) + ")"
	}
	return passthrough(text)
}

// membersFilter is which members target keeps: an element's specializations
// when target names an element, else the members of the kind it names.
func membersFilter(target string, n *tree.Node, t *tree.Tree, names *nameIndex, kinds kindIndex) (func(tree.Member) bool, error) {
	general, err := names.resolve(target, n, nil)
	if err == nil {
		return func(m tree.Member) bool { return t.Specializes(m.ID, general) }, nil
	}
	if !errors.Is(err, errNoSuchName) {
		return nil, err // ambiguous: not read as a kind
	}
	kind, ok := kinds.lookup(target)
	if !ok {
		return nil, fmt.Errorf("the model has no element named %q, and it is no element kind (such as partUsage)", target)
	}
	return func(m tree.Member) bool { return t.Model.Elements[m.ID].Type() == kind }, nil
}
