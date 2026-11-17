// Copyright 2026 Petr Mukhachev
// SPDX-License-Identifier: MIT

package macros

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Open-MBEE/OpenSysML/client/opensysml"
	asciidoctor "github.com/rolling-robot/asciidoctorg"
)

// Ifaces is the block macro sysml-ifaces::[<element>][], replaced by a
// bullet list of the interfaces and connections the target's owned ports
// are on (MC-23): each item one connection, by name, -> , and the distinct
// part(s) on the other ends. Without a target the page's element is used; a
// target is read as sysml-name reads it (MC-8, MC-15). The model is searched
// once, a reverse index of ports to connections built on first use and
// shared by every page (MC-3). The attribute name says how the connection
// and the partner parts are named, as sysml-iface-ends' name does: short
// (the default) uses each element's short name when it has one, else its
// name; long always the name.
const Ifaces = "sysml-ifaces"

// IfacesRole is the role of the list, for styling.
const IfacesRole = "sysml-ifaces"

func registerIfaces(g *asciidoctor.ExtensionGroup, e *env) {
	block(g, e, Ifaces, nil,
		func(name, sep, target string, attrs map[string]any) string {
			attrsText := "[]"
			if nm := ifaceAttrs(attrs, "name"); nm != "" {
				attrsText = "[name=" + nm + "]"
			}
			return name + sep + target + attrsText
		},
		func(c call[asciidoctor.StructuralNode]) (asciidoctor.StructuralNode, error) {
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
			metaclass, err := e.metaclass(id)
			if err != nil {
				return nil, err
			}
			if !isPartLike(metaclass) {
				return nil, fmt.Errorf("%s is not a part (%s %s); name a part", name, indefinite(metaclass), metaclass)
			}
			ports, err := portsOf(e, id)
			if err != nil {
				return nil, err
			}
			cons, err := e.connectionsThrough(id, ports)
			if err != nil {
				return nil, err
			}
			if len(cons) == 0 {
				return nil, nil // no connections; the macro line is dropped
			}
			list, err := asciidoctor.CreateList(c.parent, "ulist", asciidoctor.Attributes{"role": IfacesRole})
			if err != nil {
				return nil, err
			}
			for _, con := range cons {
				partners, err := partnersOf(e, con, ports)
				if err != nil {
					return nil, err
				}
				text, err := ifaceText(c.parent, c.page, con, partners, e, segName)
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

// metaclass is the element's metamodel type: PartUsage, Package, ...
func (e *env) metaclass(id string) (string, error) {
	p, err := e.m.Props(e.ctx, id, "owner")
	if err != nil {
		return "", err
	}
	return p.Element.Metaclass, nil
}

// isPartLike is whether elements of metaclass type own ports: a part or a
// part definition. A package, an interface, a connection and more are not.
func isPartLike(metaclass string) bool {
	return metaclass == "PartUsage" || metaclass == "PartDefinition"
}

// indefinite is the English article before a metaclass name: "an" before a
// vowel, "a" otherwise, so the failure message reads well.
func indefinite(name string) string {
	if name == "" {
		return "a"
	}
	switch name[0] {
	case 'A', 'E', 'I', 'O', 'U':
		return "an"
	}
	return "a"
}

// portsOf are the owned ports of element id: its owned members whose
// metaclass is PortUsage, in declaration order.
func portsOf(e *env, id string) ([]string, error) {
	owned, err := e.m.Members(e.ctx, id)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, r := range owned {
		if r.Metaclass == "PortUsage" {
			out = append(out, r.ID)
		}
	}
	return out, nil
}

// ifaceConn is one connection the target's ports are on.
type ifaceConn struct {
	id   string
	ends []connectorEnd
}

// connIndex maps each element id in a connection's end paths to the
// connections whose ends reference it, so a target's ports find the
// connections through them without walking the model per page (MC-3).
type connIndex struct {
	byID    map[string]ifaceConn
	through map[string][]string // element id -> connection ids
}

// conns is the connection index, built on first use (MC-3): one query for
// every ConnectionUsage and InterfaceUsage, each read once with connectorEnds.
func (e *env) conns() (connIndex, error) {
	if e.connIndex != nil {
		return *e.connIndex, nil
	}
	idx := connIndex{byID: map[string]ifaceConn{}, through: map[string][]string{}}
	els, err := e.m.Query(e.ctx, opensysml.Query{Where: opensysml.InstanceOf("ConnectionUsage", "InterfaceUsage")})
	if err != nil {
		return idx, err
	}
	for _, q := range els {
		p, err := e.m.Props(e.ctx, q.ID, "owner")
		if err != nil {
			return idx, err
		}
		cid := p.Element.ID
		ends, err := connectorEnds(e, cid)
		if err != nil {
			return idx, err
		}
		con := ifaceConn{id: cid, ends: ends}
		idx.byID[cid] = con
		for _, end := range ends {
			for _, id := range end.path {
				if !slices.Contains(idx.through[id], con.id) {
					idx.through[id] = append(idx.through[id], con.id)
				}
			}
		}
	}
	e.connIndex = &idx
	return idx, nil
}

// connectionsThrough are the connections any of the target's ports is on,
// each once, in model order.
func (e *env) connectionsThrough(id string, ports []string) ([]ifaceConn, error) {
	idx, err := e.conns()
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var ids []string
	for _, port := range ports {
		for _, cid := range idx.through[port] {
			if !seen[cid] {
				seen[cid] = true
				ids = append(ids, cid)
			}
		}
	}
	// Keep the model's declaration order (the query returns it).
	var out []ifaceConn
	for _, cid := range ids {
		out = append(out, idx.byID[cid])
	}
	return out, nil
}

// partnersOf are the parts on the other ends of con, distinct: the ends that
// reference none of the target's ports, each the owner of the end's leaf
// feature, deduplicated. A self-loop, all ends on the target's own ports,
// gives none.
func partnersOf(e *env, con ifaceConn, ports []string) ([]string, error) {
	isTargetPort := map[string]bool{}
	for _, p := range ports {
		isTargetPort[p] = true
	}
	var out []string
	for _, end := range con.ends {
		if referencesAny(end.path, isTargetPort) {
			continue
		}
		if len(end.path) == 0 {
			continue // an interface def's end connects nothing
		}
		ownerChain, err := owners(e, end.path[len(end.path)-1])
		if err != nil {
			return nil, err
		}
		if len(ownerChain) == 0 {
			continue
		}
		part := ownerChain[len(ownerChain)-1]
		if !slices.Contains(out, part) {
			out = append(out, part)
		}
	}
	return out, nil
}

// referencesAny is whether an end's path holds any of the target's port ids.
func referencesAny(path []string, ports map[string]bool) bool {
	for _, id := range path {
		if ports[id] {
			return true
		}
	}
	return false
}

// ifaceText is a list item's AsciiDoc: the connection by name, linked to its
// page when it has one, -> , and the partner parts, each linked, joined with
// ", ". A connection with no partners (a self-loop) shows only the
// connection and an empty right side.
func ifaceText(parent asciidoctor.Node, page element, con ifaceConn, partners []string, e *env, seg segmentName) (string, error) {
	connName, err := segmentText(parent, page, e, seg, con.id)
	if err != nil {
		return "", err
	}
	var partnerTexts []string
	for _, p := range partners {
		text, err := segmentText(parent, page, e, seg, p)
		if err != nil {
			return "", err
		}
		partnerTexts = append(partnerTexts, text)
	}
	item := connName + " -> "
	if len(partnerTexts) > 0 {
		item += strings.Join(partnerTexts, ", ")
	}
	return passthrough(item), nil
}
