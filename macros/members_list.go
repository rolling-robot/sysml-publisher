package macros

import (
	"errors"
	"fmt"
	"html"
	"strings"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
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

func registerMembersList(g *asciidoctor.ExtensionGroup, e *env) {
	block(g, e, MembersList, nil, nil,
		func(c call[asciidoctor.StructuralNode]) (asciidoctor.StructuralNode, error) {
			n := c.page
			all, err := members(e, n.id)
			if err != nil {
				return nil, err
			}
			members := all
			if c.target != "" {
				keep, err := membersFilter(c.target, n, e)
				if err != nil {
					return nil, err
				}
				members = nil
				for _, m := range all {
					ok, err := keep(m)
					if err != nil {
						return nil, err
					}
					if ok {
						members = append(members, m)
					}
				}
			}
			if len(members) == 0 {
				return nil, nil // nothing to list; the macro line is dropped
			}
			list, err := asciidoctor.CreateList(c.parent, "ulist", asciidoctor.Attributes{"role": MembersListRole})
			if err != nil {
				return nil, err
			}
			for _, m := range members {
				item, err := asciidoctor.CreateListItem(list, memberText(n, m, e, c.target == ""))
				if err != nil {
					return nil, err
				}
				list.AddItem(item)
			}
			return list, nil
		})
}

// member is an owned member as sysml-members-list shows it.
type member struct {
	id        string
	name      string // its name, or sysml.UnnamedName
	metaclass string // PartUsage, AttributeUsage, ...
	// types are the names of its types (part w : Wheel gives Wheel), in
	// declaration order, including those a usage gets through what it
	// subsets or redefines; none for a definition or an untyped usage.
	types []string
}

// notMembers are the metaclasses of owned members that sysml-members-list
// leaves out (MC-12): annotations, which annotate the owner rather than make
// it up, and multiplicities, which are part of the owner's declaration
// (OpenSysML gives every usage one, its implicit [1..1] included).
var notMembers = map[string]bool{
	"Comment": true, "Documentation": true, "TextualRepresentation": true,
	"MetadataUsage": true, "MetadataFeature": true,
	"Multiplicity": true, "MultiplicityRange": true,
}

// members are the owned members of element id, in declaration order,
// other than annotations and multiplicities.
func members(e *env, id string) ([]member, error) {
	owned, err := e.m.Members(e.ctx, id)
	if err != nil {
		return nil, err
	}
	var out []member
	for _, r := range owned {
		if notMembers[r.Metaclass] {
			continue
		}
		name, err := e.names.name(r.ID)
		if err != nil {
			return nil, err
		}
		types, err := e.m.Types(e.ctx, r.ID)
		if err != nil {
			return nil, err
		}
		var typeNames []string
		for _, typ := range types {
			tn := typ.Name() // from the qualified name: library types need no read
			if tn == "" {
				if tn, err = e.m.Name(e.ctx, typ.ID); err != nil {
					return nil, err
				}
			}
			if tn != "" {
				typeNames = append(typeNames, tn)
			}
		}
		out = append(out, member{id: r.ID, name: name, metaclass: r.Metaclass, types: typeNames})
	}
	return out, nil
}

// memberText is a list item's AsciiDoc: the member's name, linked to its
// page when it has one, and its types in parentheses when withTypes is set
// and it has any: mass (Real).
func memberText(page element, m member, e *env, withTypes bool) string {
	text := e.linkTo(page.id, m.id, html.EscapeString(m.name))
	if withTypes && len(m.types) > 0 {
		text += " (" + html.EscapeString(strings.Join(m.types, ", ")) + ")"
	}
	return passthrough(text)
}

// membersFilter is which members target keeps: an element's specializations
// when target names an element, else the members of the kind it names.
func membersFilter(target string, n element, e *env) (func(member) (bool, error), error) {
	general, err := e.names.resolve(target, n, nil)
	if err == nil {
		return func(m member) (bool, error) { return e.m.Specializes(e.ctx, m.id, general.ID) }, nil
	}
	if !errors.Is(err, errNoSuchName) {
		return nil, err // ambiguous: not read as a kind
	}
	kinds, err := e.kinds()
	if err != nil {
		return nil, err
	}
	kind, ok := kinds.lookup(target)
	if !ok {
		return nil, fmt.Errorf("the model has no element named %q, and it is no element kind (such as partUsage)", target)
	}
	return func(m member) (bool, error) { return m.metaclass == kind, nil }, nil
}
