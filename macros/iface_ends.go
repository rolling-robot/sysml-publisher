package macros

import (
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
)

// IfaceEnds is the block macro sysml-iface-ends::[<element>][], replaced by
// a bullet list of the ends of an interface (or any connection): each end's
// feature, engine.air, its segments linked to their pages (MC-17). The
// source end comes first when there is one. Without a target the page's
// element is used; a target is read as sysml-name reads it (MC-8, MC-15).
// The attribute depth (named, or the first positional one) shows the last
// depth segments of the feature's qualified path, joined by " :: " (1: air,
// 2: engine :: air), all the whole path (Model :: aircraft :: engine :: air).
// The attribute name says how each segment is named: short (the default)
// uses the element's short name when it has one, falling back to its name;
// long always uses the name. Names are shown without single quotes.
const IfaceEnds = "sysml-iface-ends"

// IfaceEndsRole is the role of the list, for styling.
const IfaceEndsRole = "sysml-iface-ends"

func registerIfaceEnds(g *asciidoctor.ExtensionGroup, e *env) {
	block(g, e, IfaceEnds, &asciidoctor.MacroOptions{PositionalAttributes: []string{"depth"}},
		func(name, sep, target string, attrs map[string]any) string {
			attrsText := ""
			if depth := ifaceDepth(attrs); depth != "" {
				attrsText = "[depth=" + depth + "]"
			} else if nm, _ := attrs["name"].(string); nm != "" {
				attrsText = "[name=" + strings.TrimSpace(nm) + "]"
			}
			if attrsText == "" {
				attrsText = "[]"
			}
			return name + sep + target + attrsText
		},
		func(c call[asciidoctor.StructuralNode]) (asciidoctor.StructuralNode, error) {
			depth, err := parseDepth(ifaceDepth(c.attrs))
			if err != nil {
				return nil, err
			}
			segName, err := parseSegmentName(ifaceAttrs(c.attrs, "name"))
			if err != nil {
				return nil, err
			}
			id, name := c.page.id, c.page.qualifiedName
			if c.target != "" {
				if id, err = nameTarget(c.target, c.page, e); err != nil {
					return nil, err
				}
				if name, err = e.names.qualifiedName(id); err != nil {
					return nil, err
				}
			}
			ends, err := connectorEnds(e, id)
			if err != nil {
				return nil, err
			}
			if len(ends) == 0 {
				return nil, fmt.Errorf("%s has no ends; name an interface or connection", name)
			}
			list, err := asciidoctor.CreateList(c.parent, "ulist", asciidoctor.Attributes{"role": IfaceEndsRole})
			if err != nil {
				return nil, err
			}
			for _, end := range ends {
				text, err := endText(c.parent, c.page, end, e, depth, segName)
				if err != nil {
					return nil, err
				}
				item, err := asciidoctor.CreateListItem(list, text)
				if err != nil {
					return nil, err
				}
				list.AddItem(item)
			}
			return list, nil
		})
}

// ifaceDepth is the depth attribute of a sysml-iface-ends macro, trimmed.
func ifaceDepth(attrs map[string]any) string {
	return ifaceAttrs(attrs, "depth")
}

// ifaceAttrs is a named attribute of a sysml-iface-ends macro, trimmed.
func ifaceAttrs(attrs map[string]any, key string) string {
	v, _ := attrs[key].(string)
	return strings.TrimSpace(v)
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

// setSegment's name mode: segmentShort uses each element's short name when
// it has one, else its name; segmentLong always uses the name. The default,
// without the name attribute, is segmentShort.
type segmentName int

const (
	segmentShort segmentName = iota
	segmentLong
)

// parseSegmentName reads the name attribute: empty or short, else long.
func parseSegmentName(s string) (segmentName, error) {
	switch strings.ToLower(s) {
	case "", "short":
		return segmentShort, nil
	case "long":
		return segmentLong, nil
	}
	return 0, fmt.Errorf("name %q is not short or long", s)
}

// segmentText is the text of one path segment, an element's name (or short
// name), single quotes stripped, and linked to its page when it has one.
func segmentText(parent asciidoctor.Node, page element, e *env, seg segmentName, id string) (string, error) {
	name, err := e.names.name(id)
	if err != nil {
		return "", err
	}
	if seg == segmentShort {
		if short, err := e.m.ShortName(e.ctx, id); err != nil {
			return "", err
		} else if short != "" {
			name = short
		}
	}
	text := html.EscapeString(strings.ReplaceAll(name, "'", ""))
	if href, ok := e.href(page.id, id); ok {
		return link(parent, id, href, text)
	}
	return text, nil
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
func connectorEnds(e *env, id string) ([]connectorEnd, error) {
	p, err := e.m.Props(e.ctx, id, "ownedFeature")
	if err != nil {
		return nil, err
	}
	var ends []connectorEnd
	for _, f := range p.Refs("ownedFeature") {
		fp, err := e.m.Props(e.ctx, f.ID, "isEnd", "name", "ownedReferenceSubsetting")
		if err != nil {
			return nil, err
		}
		if !fp.Bool("isEnd") {
			continue
		}
		end := connectorEnd{name: fp.Str("name")}
		for _, rs := range fp.Refs("ownedReferenceSubsetting") {
			rp, err := e.m.Props(e.ctx, rs.ID, "referencedFeature")
			if err != nil {
				return nil, err
			}
			ref := rp.Ref("referencedFeature").ID
			cp, err := e.m.Props(e.ctx, ref, "chainingFeature")
			if err != nil {
				return nil, err
			}
			if chain := cp.Refs("chainingFeature"); len(chain) > 0 {
				end.path = nil
				for _, c := range chain {
					end.path = append(end.path, c.ID)
				}
			} else {
				end.path = []string{ref}
			}
		}
		ends = append(ends, end)
	}
	for i, end := range ends {
		if end.name == "source" && i > 0 {
			ends = append(append([]connectorEnd{end}, ends[:i]...), ends[i+1:]...)
			break
		}
	}
	return ends, nil
}

// endText is a list item's AsciiDoc: the feature the end connects, each
// segment linked to its page when it has one. With depthAsWritten the
// feature is the chain as written (engine.air); otherwise it is the
// feature's qualified path, the owners of the chain's first element followed
// by the chain, cut to its last depth segments and joined by " :: "
// (aircraft :: engine :: air). seg names each segment by short name when
// available (segmentShort) or always by name (segmentLong). An end that
// connects nothing (an interface def's) shows its name only.
func endText(parent asciidoctor.Node, page element, end connectorEnd, e *env, depth int, seg segmentName) (string, error) {
	ids, sep := end.path, "."
	if depth != depthAsWritten && len(end.path) > 0 {
		outer, err := owners(e, end.path[0])
		if err != nil {
			return "", err
		}
		ids = append(outer, end.path...)
		if depth != depthAll && depth < len(ids) {
			ids = ids[len(ids)-depth:]
		}
		sep = " :: "
	}
	segs := make([]string, len(ids))
	for i, id := range ids {
		segText, err := segmentText(parent, page, e, seg, id)
		if err != nil {
			return "", err
		}
		segs[i] = segText
	}
	path := strings.Join(segs, sep)
	if path == "" {
		return passthrough(html.EscapeString(end.name)), nil
	}
	return passthrough(path), nil
}

// owners are the elements that own id, outermost first, up to a top-level
// element: the root namespace, which has no owner, is left out.
func owners(e *env, id string) ([]string, error) {
	var out []string
	for {
		o, err := e.m.Owner(e.ctx, id)
		if err != nil || o.ID == "" {
			return out, err
		}
		oo, err := e.m.Owner(e.ctx, o.ID)
		if err != nil || oo.ID == "" {
			return out, err // o is the root namespace
		}
		out = append([]string{o.ID}, out...)
		id = o.ID
	}
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
