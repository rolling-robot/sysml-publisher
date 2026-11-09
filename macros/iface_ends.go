package macros

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/model"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

// IfaceEnds is the block macro sysml-iface-ends::[<element>][], replaced by
// a bullet list of the ends of an interface (or any connection): each end's
// name and the feature it connects, engine.air, its segments linked to their
// pages (MC-17). The source end comes first when there is one. Without a
// target the page's element is used; a target is read as sysml-name reads
// it (MC-8, MC-15). The attribute depth (named, or the first positional
// one) shows the last depth segments of the feature's qualified path,
// joined by :: (1: air, 2: engine::air), all the whole path
// (Model::aircraft::engine::air).
const IfaceEnds = "sysml-iface-ends"

// IfaceEndsRole is the role of the list, for styling.
const IfaceEndsRole = "sysml-iface-ends"

func registerIfaceEnds(g *asciidoctor.ExtensionGroup, t *tree.Tree) {
	md := newMetadataIndex(t.Model)
	g.BlockMacro(IfaceEnds, asciidoctor.BlockMacroProcessorFunc(
		func(parent asciidoctor.StructuralNode, target string, attrs map[string]any) (asciidoctor.StructuralNode, error) {
			n, err := pageElement(parent, t)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", IfaceEnds, err)
			}
			depthText, _ := attrs["depth"].(string)
			depthText = strings.TrimSpace(depthText)
			macro := fmt.Sprintf("%s::%s[]", IfaceEnds, target)
			if depthText != "" {
				macro = fmt.Sprintf("%s::%s[depth=%s]", IfaceEnds, target, depthText)
			}
			depth, err := parseDepth(depthText)
			if err != nil {
				return nil, fmt.Errorf("%s on the page of %s: %w", macro, n.QualifiedName, err)
			}
			id, name := n.ID, n.QualifiedName
			if target != "" {
				if id, err = nameTarget(target, n, md); err != nil {
					return nil, fmt.Errorf("%s on the page of %s: %w", macro, n.QualifiedName, err)
				}
				name = md.names.qualifiedName(id)
			}
			ends := connectorEnds(t.Model, id)
			if len(ends) == 0 {
				return nil, fmt.Errorf("%s on the page of %s: %s has no ends; name an interface or connection", macro, n.QualifiedName, name)
			}
			list, err := asciidoctor.CreateList(parent, "ulist", asciidoctor.Attributes{"role": IfaceEndsRole})
			if err != nil {
				return nil, fmt.Errorf("%s: %w", IfaceEnds, err)
			}
			for _, e := range ends {
				item, err := asciidoctor.CreateListItem(list, endText(n, e, t, md.names, depth))
				if err != nil {
					return nil, fmt.Errorf("%s: %w", IfaceEnds, err)
				}
				list.AddItem(item)
			}
			return list, nil
		}), &asciidoctor.MacroOptions{PositionalAttributes: []string{"depth"}})
}

// Depths of sysml-iface-ends: depthAsWritten without the attribute, the
// chain as the model writes it (engine.air); depthAll for depth=all;
// otherwise the number of trailing segments of the qualified path.
const (
	depthAsWritten = 0
	depthAll       = -1
)

// parseDepth reads the depth attribute: empty, a positive number or all.
func parseDepth(s string) (int, error) {
	switch {
	case s == "":
		return depthAsWritten, nil
	case strings.EqualFold(s, "all"):
		return depthAll, nil
	}
	d, err := strconv.Atoi(s)
	if err != nil || d < 1 {
		return 0, fmt.Errorf("depth %q is not a positive number or all", s)
	}
	return d, nil
}

// connectorEnd is one end of a connection.
type connectorEnd struct {
	name string   // the end's name (source, supplier), "" when it has none
	path []string // ids of the features it connects, a chain: engine, air
}

// connectorEnds are the ends the element owns, the source first, then the
// rest in declaration order. The source is the end named source, the name
// OpenSysML gives the first end of connect a to b and connect (a, b); ends
// named by an interface def (supplier, consumer) keep the def's order.
func connectorEnds(m *model.Model, id string) []connectorEnd {
	var ends []connectorEnd
	for _, fid := range m.Elements[id].Refs("ownedFeature") {
		f := m.Elements[fid]
		if isEnd, _ := f["isEnd"].(bool); !isEnd {
			continue
		}
		e := connectorEnd{name: m.Elements[f.Ref("owningMembership")].Str("memberName")}
		if e.name == "" {
			e.name = f.Str("declaredName")
		}
		for _, rid := range f.Refs("ownedRelationship") {
			r := m.Elements[rid]
			if r.Type() != "ReferenceSubsetting" {
				continue
			}
			ref := r.Ref("general")
			if chain := m.Elements[ref].Refs("chainingFeature"); len(chain) > 0 {
				e.path = chain
			} else {
				e.path = []string{ref}
			}
		}
		ends = append(ends, e)
	}
	for i, e := range ends {
		if e.name == "source" && i > 0 {
			ends = append(append([]connectorEnd{e}, ends[:i]...), ends[i+1:]...)
			break
		}
	}
	return ends
}

// endText is a list item's AsciiDoc: the end's name and the feature it
// connects, each segment linked to its page when it has one. With
// depthAsWritten the feature is the chain as written (source: engine.air);
// otherwise it is the feature's qualified path, the owners of the chain's
// first element followed by the chain, cut to its last depth segments and
// joined by :: (source: aircraft::engine::air). An end that connects nothing
// (an interface def's) shows its name only.
func endText(page *tree.Node, e connectorEnd, t *tree.Tree, names *nameIndex, depth int) string {
	ids, sep := e.path, "."
	if depth != depthAsWritten && len(e.path) > 0 {
		ids = append(owners(t.Model, e.path[0]), e.path...)
		if depth != depthAll && depth < len(ids) {
			ids = ids[len(ids)-depth:]
		}
		sep = "::"
	}
	segs := make([]string, len(ids))
	for i, id := range ids {
		name := names.name(id)
		if sep == "::" {
			name = quoteName(name)
		}
		segs[i] = html.EscapeString(name)
		if target := t.ByID[id]; target != nil {
			segs[i] = `<a href="` + tree.Href(page, target) + `">` + segs[i] + `</a>`
		}
	}
	path := strings.Join(segs, sep)
	switch {
	case e.name == "":
		return passthrough(path)
	case path == "":
		return passthrough(html.EscapeString(e.name))
	}
	return passthrough(html.EscapeString(e.name) + ": " + path)
}

// owners are the elements that own id, outermost first, up to a top-level
// element: the root namespace, which has no owner, is left out.
func owners(m *model.Model, id string) []string {
	var out []string
	for o := m.Elements[id].Ref("owner"); o != ""; o = m.Elements[o].Ref("owner") {
		if m.Elements[o].Ref("owner") == "" {
			break // the root namespace
		}
		out = append([]string{o}, out...)
	}
	return out
}

// basicNameRx is a SysML name that needs no quotes.
var basicNameRx = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// quoteName writes a name as a qualified name segment: quoted, as SysML
// writes it, when it is not a basic name ('de-icing fluid').
func quoteName(name string) string {
	if basicNameRx.MatchString(name) {
		return name
	}
	return "'" + strings.NewReplacer(`\`, `\\`, `'`, `\'`).Replace(name) + "'"
}
