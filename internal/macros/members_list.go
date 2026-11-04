package macros

import (
	"fmt"
	"html"
	"slices"
	"strings"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/model"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

// MembersList is the block macro sysml-members-list::[<element>][], replaced
// by a bullet list of the owned members of the page's element (MC-12).
// Without an element every member is listed, each with its types in
// parentheses; with one, only the members that subclass or subset it.
const MembersList = "sysml-members-list"

// MembersListRole is the role of the list, for styling.
const MembersListRole = "sysml-members-list"

func registerMembersList(g *asciidoctor.ExtensionGroup, t *tree.Tree) {
	names := newNameIndex(t.Model)
	g.BlockMacro(MembersList, asciidoctor.BlockMacroProcessorFunc(
		func(parent asciidoctor.StructuralNode, target string, _ map[string]any) (asciidoctor.StructuralNode, error) {
			n, err := pageElement(parent, t)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", MembersList, err)
			}
			members := n.Members
			if target != "" {
				general, err := names.resolve(target, n)
				if err != nil {
					return nil, fmt.Errorf("%s::%s[] on the page of %s: %w", MembersList, target, n.QualifiedName, err)
				}
				members = nil
				for _, m := range n.Members {
					if t.Specializes(m.ID, general) {
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

// nameIndex finds the element a macro names.
type nameIndex struct {
	byQualified map[string]string   // qualified name → id
	byName      map[string][]string // declared name → ids, in model order
	m           *model.Model
}

func newNameIndex(m *model.Model) *nameIndex {
	x := &nameIndex{byQualified: map[string]string{}, byName: map[string][]string{}, m: m}
	for _, id := range m.Order {
		e := m.Elements[id]
		if qn := e.Str("qualifiedName"); qn != "" {
			if _, dup := x.byQualified[qn]; !dup {
				x.byQualified[qn] = id
			}
		}
		if name := e.Str("declaredName"); name != "" {
			x.byName[name] = append(x.byName[name], id)
		}
	}
	// Library elements the model refers to can be named too: Real, or
	// ScalarValues::Real. Sorted, so an ambiguity is reported the same way
	// on every run.
	ids := make([]string, 0, len(m.Library))
	for id := range m.Library {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		lib := m.Library[id]
		if _, dup := x.byQualified[lib.QualifiedName]; !dup {
			x.byQualified[lib.QualifiedName] = id
		}
		x.byName[lib.Name()] = append(x.byName[lib.Name()], id)
	}
	return x
}

// resolve finds the element name refers to, seen from the page of n: a
// qualified name as written; else the name inside n's element, then inside
// each enclosing namespace outward; else a plain name that names one element
// in the model.
func (x *nameIndex) resolve(name string, n *tree.Node) (string, error) {
	if id, ok := x.byQualified[name]; ok {
		return id, nil
	}
	for scope := n.QualifiedName; scope != ""; scope = enclosing(scope) {
		if id, ok := x.byQualified[scope+"::"+name]; ok {
			return id, nil
		}
	}
	ids := x.byName[name]
	switch len(ids) {
	case 0:
		return "", fmt.Errorf("the model has no element named %q", name)
	case 1:
		return ids[0], nil
	}
	var qns []string
	for _, id := range ids {
		qn := x.m.Elements[id].Str("qualifiedName")
		if lib, ok := x.m.Library[id]; ok {
			qn = lib.QualifiedName
		}
		qns = append(qns, qn)
	}
	return "", fmt.Errorf("%q names several elements (%s); use a qualified name", name, strings.Join(qns, ", "))
}

// enclosing is the namespace a qualified name is declared in: M for M::pd,
// "" for M.
func enclosing(qualifiedName string) string {
	if i := strings.LastIndex(qualifiedName, "::"); i >= 0 {
		return qualifiedName[:i]
	}
	return ""
}
