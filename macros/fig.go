// Copyright 2026 Petr Mukhachev
// SPDX-License-Identifier: MIT

package macros

import (
	"errors"
	"fmt"
	"html"
	"slices"
	"strings"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
)

// Fig is the block macro sysml-fig::[<element>][], replaced, when the target
// is an interface or a part, by a Mermaid figure of it (MC-25): an
// interface's figure shows the parts on its ends and the parts between them
// and their common ancestor, each end's feature, and one edge per pair of
// ends; a part's figure shows its owned ports, and, for every connection
// they are on, the other ends' features and their owner parts. The figures
// are drawn by the site's mermaid.js in the look of OpenSysML's Mermaid
// interconnection form. Without a target the page's element is used; a
// target is read as sysml-name reads it (MC-8, MC-15). A resolvable target
// that is neither an interface nor a part is written out as the macro text,
// no warning raised. The figure draws exactly what the ends' chains pass
// through; the model is read the way sysml-ifaces reads it, the connections
// indexed once for every page (MC-3).
const Fig = "sysml-fig"

// FigRole is the role of the block a figure is put in, for styling.
const FigRole = "sysml-fig"

// AttrFig is the attribute of the block a figure is put in: the qualified
// name of the element the figure is of. As links are anchor nodes
// (AttrLink) and a view an image block (AttrView), a figure is a pass block
// the document's converter writes verbatim; a converter for another output
// can tell the figure by AttrFig and take its Mermaid source from
// MermaidText.
const AttrFig = "sysml-fig"

// AttrMermaid is the attribute of the block a figure is put in: the
// figure's Mermaid source, for an output that renders Mermaid itself.
const AttrMermaid = "sysml-mermaid"

// figPre opens the block a figure is written in: a pre of class mermaid,
// which mermaid.js renders, in a div that styles it.
const figPre = `<div class="` + FigRole + `"><pre class="mermaid">`

// figPreClose closes it.
const figPreClose = "</pre></div>"

func registerFig(g *asciidoctor.ExtensionGroup, e *env) {
	block(g, e, Fig, nil, nil,
		func(c call[asciidoctor.StructuralNode]) (asciidoctor.StructuralNode, error) {
			return e.fig(c)
		})
}

// fig is one use of sysml-fig. A target that names no element fails; a
// resolvable one that is neither an interface nor a part is written out; an
// interface or a part is figured.
func (e *env) fig(c call[asciidoctor.StructuralNode]) (asciidoctor.StructuralNode, error) {
	id, name := c.page.id, c.page.qualifiedName
	if c.target != "" {
		var err error
		if id, err = nameTarget(c.target, c.page, e); err != nil {
			return nil, err
		}
		if name, err = e.names.qualifiedName(id); err != nil {
			return nil, err
		}
	}
	meta, err := e.metaclass(id)
	if err != nil {
		return nil, err
	}
	switch {
	case meta == "InterfaceUsage":
		ends, err := connectorEnds(e, id)
		if err != nil {
			return nil, err
		}
		t, err := e.figure(name, id, ends)
		if err != nil {
			return nil, err
		}
		return figBlock(c.parent, e.figLinks(c.page.id, t))
	case isPartLike(meta):
		t, err := e.partFigure(name, id, meta)
		if err != nil {
			return nil, err
		}
		return figBlock(c.parent, e.figLinks(c.page.id, t))
	default:
		return figEcho(c.parent, c.macro)
	}
}

// figEcho is the macro written out: a resolvable target that is neither an
// interface nor a part leaves the text of the macro call where it was, no
// warning.
func figEcho(parent asciidoctor.StructuralNode, macro string) (asciidoctor.StructuralNode, error) {
	return asciidoctor.CreatePassBlock(parent,
		`<p class="`+FigRole+`-text">`+html.EscapeString(macro)+`</p>`, nil, nil)
}

// figLinks sets each drawn node's href: the link from the page the figure
// is drawn on to the node element's page, empty when the element is the
// page's own or has no page (MC-18).
func (e *env) figLinks(page string, t figTree) figTree {
	var walk func([]*figNode)
	walk = func(nodes []*figNode) {
		for _, n := range nodes {
			if n.elem != page {
				if href, ok := e.href(page, n.elem); ok {
					n.href = href
				}
			}
			walk(n.children)
		}
	}
	for _, r := range t.roots {
		walk([]*figNode{r})
	}
	return t
}

// figBlock is a figure as a pass block: its content the Mermaid source in a
// pre of class mermaid, which the site's mermaid.js draws; the attributes
// carry the source (AttrMermaid) and what the figure is of (AttrFig), for
// an output that writes figures its own way.
func figBlock(parent asciidoctor.StructuralNode, t figTree) (asciidoctor.StructuralNode, error) {
	text := mermaidFigure(t)
	b, err := asciidoctor.CreatePassBlock(parent, figPre+html.EscapeString(text)+figPreClose, nil, nil)
	if err != nil {
		return nil, err
	}
	b.SetAttr(AttrFig, t.of, true)
	b.SetAttr(AttrMermaid, text, true)
	b.AddRole(FigRole)
	return b, nil
}

// MermaidText is the Mermaid source of a figure sysml-fig put in, for a
// converter that renders Mermaid itself: n is the pass block, carrying
// AttrFig. ok is false for any other node.
func MermaidText(n asciidoctor.Node) (text string, ok bool) {
	if _, isFig := n.Attr(AttrFig).(string); !isFig {
		return "", false
	}
	text, _ = n.Attr(AttrMermaid).(string)
	return text, true
}

// figure is the figure of one interface: its tree of drawn elements, each
// end's feature, and the edges. ends are the interface's ends, source
// first, as connectorEnds reads them. An interface with more than two ends
// is drawn as a node of its own, the ends' features joining it one edge
// each; two ends join directly.
func (e *env) figure(iface, id string, ends []connectorEnd) (figTree, error) {
	t := figTree{of: iface}
	if len(ends) < 2 {
		return t, fmt.Errorf("has %d ends; a figure needs at least two", len(ends))
	}
	// Each end's chain: the owners of its leaf feature, outermost first,
	// then the leaf itself. Two ends' chains share their start, which is
	// where the common ancestor is.
	leaf := make([]string, len(ends))
	chains := make([][]string, len(ends))
	for i, end := range ends {
		if len(end.path) == 0 {
			n := end.name
			if n == "" {
				n = "<unnamed>"
			}
			return t, fmt.Errorf("end %s connects nothing", n)
		}
		leaf[i] = end.path[len(end.path)-1]
		outer, err := owners(e, leaf[i])
		if err != nil {
			return t, err
		}
		chains[i] = append(slices.Clone(outer), leaf[i])
	}
	common, ok := commonAncestor(chains)
	if !ok {
		return t, errors.New("the ends' parts have no common ancestor; a figure needs them under one owner")
	}
	// The drawn elements are the chains cut at the common ancestor: one
	// node per element, the common ancestor the outermost, drawn whatever
	// it is, for the frame of the figure. A package further down a chain
	// is not drawn — an interconnection shows no namespaces — so what it
	// owns stands where the package would have stood.
	// Every chain shares its start with the first, and the common ancestor
	// is the deepest of what they share, so it sits at the same index of
	// every chain; cutting there leaves the common ancestor first.
	k := slices.Index(chains[0], common)
	for i := range chains {
		chains[i] = chains[i][k:]
	}
	byID := map[string]*figNode{}   // model id → node
	parentOf := map[string]string{} // a chain element's predecessor, by model id
	var order []*figNode
	for _, chain := range chains {
		prev := ""
		for _, id := range chain {
			if n := byID[id]; n == nil {
				meta, err := e.metaclass(id)
				if err != nil {
					return t, err
				}
				if meta == "Package" && id != common {
					parentOf[id] = prev
					prev = id // not drawn: its members stand where it would
					continue
				}
				nm, err := e.names.name(id)
				if err != nil {
					return t, err
				}
				n = &figNode{elem: id, kind: "«" + figKeyword(meta) + "»", name: nm, box: isPartLike(meta) || id == common}
				byID[id] = n
				parentOf[id] = prev
				order = append(order, n)
			}
			prev = id
		}
	}
	// More than two ends: the interface is drawn as a node of its own, the
	// ends' features joining it one edge each.
	var ifaceNode *figNode
	if len(ends) > 2 {
		nm, err := e.names.name(id)
		if err != nil {
			return t, err
		}
		ifaceNode = &figNode{elem: id, kind: "«interface»", name: nm}
		order = append(order, ifaceNode)
	}
	for i, n := range order {
		n.id = "n" + fmt.Sprint(i)
	}
	// What a drawn element is nested in: its nearest drawn ancestor that
	// holds things, a part box; the top of the figure for one whose
	// ancestors are all packages or features. The interface's node is
	// placed after, by its owners.
	for _, n := range order {
		if n == ifaceNode {
			continue
		}
		for p := parentOf[n.elem]; ; p = parentOf[p] {
			if p == "" {
				t.roots = append(t.roots, n)
				break
			}
			if box := byID[p]; box != nil && box.box {
				box.children = append(box.children, n)
				break
			}
		}
	}
	if ifaceNode != nil {
		// The interface is drawn where the model owns it: inside its
		// nearest drawn box ancestor, the frame included, else the top of
		// the figure.
		own, err := owners(e, id)
		if err != nil {
			return t, err
		}
		placed := false
		for i := len(own) - 1; i >= 0 && !placed; i-- {
			if b := byID[own[i]]; b != nil && b.box {
				b.children = append(b.children, ifaceNode)
				placed = true
			}
		}
		if !placed {
			t.roots = append(t.roots, ifaceNode)
		}
	}
	switch {
	case ifaceNode != nil:
		// One edge per end, the end's feature joining the interface's node;
		// two ends on the same feature draw one edge.
		seen := map[string]bool{}
		for i := range ends {
			a := byID[leaf[i]]
			if a == nil || seen[a.id] {
				continue
			}
			seen[a.id] = true
			t.edges = append(t.edges, [2]string{a.id, ifaceNode.id})
		}
	default:
		// One edge per pair of ends, joining the end features. An edge whose
		// ends are the same feature is not drawn: Mermaid draws no loop.
		for i := 0; i < len(ends); i++ {
			for j := i + 1; j < len(ends); j++ {
				a, b := byID[leaf[i]], byID[leaf[j]]
				if a == nil || b == nil || a.id == b.id {
					continue
				}
				t.edges = append(t.edges, [2]string{a.id, b.id})
			}
		}
	}
	return t, nil
}

// partFigure is the figure of one part: the part's box with its owned ports
// drawn inside, and, for every connection its ports are on, each end of it:
// an end whose path references one of the part's ports anchors at that
// port's node; another end's leaf feature draws inside a box for its
// immediate owner, the owner beside the part's box, or inside it when the
// model owns it there. A connection with more than two ends is drawn as a
// node of its own, the ends joining it one edge each; two ends join
// directly, as the interface's figure draws. An end that connects nothing
// draws no node and no edge.
func (e *env) partFigure(name, part, meta string) (figTree, error) {
	t := figTree{of: name}
	nm, err := e.names.name(part)
	if err != nil {
		return t, err
	}
	root := &figNode{elem: part, kind: "«" + figKeyword(meta) + "»", name: nm, box: true}
	byID := map[string]*figNode{part: root} // model id → the node drawn for it
	order := []*figNode{root}               // the nodes in drawing order
	parentOf := map[*figNode]*figNode{}     // where a node nests, nil: the top
	newNode := func(id string, box bool) (*figNode, error) {
		if n := byID[id]; n != nil {
			return n, nil
		}
		mc, err := e.metaclass(id)
		if err != nil {
			return nil, err
		}
		nm, err := e.names.name(id)
		if err != nil {
			return nil, err
		}
		n := &figNode{elem: id, kind: "«" + figKeyword(mc) + "»", name: nm, box: box}
		byID[id] = n
		order = append(order, n)
		return n, nil
	}
	ports, err := portsOf(e, part)
	if err != nil {
		return t, err
	}
	if len(ports) == 0 {
		return t, errors.New("has no ports; a figure needs at least one")
	}
	portSet := make(map[string]bool, len(ports))
	for _, p := range ports {
		n, err := newNode(p, false)
		if err != nil {
			return t, err
		}
		parentOf[n] = root
		portSet[p] = true
	}
	cons, err := e.connectionsThrough(part, ports)
	if err != nil {
		return t, err
	}
	// The nodes are created first and placed after, so a chain that passes
	// through a box another end creates later is still placed inside it.
	// pend holds each node with the owners chain its place walks.
	type pending struct {
		n     *figNode
		chain []string
	}
	var pend []pending
	var anchored [][]*figNode // per connection: the nodes its ends anchor at
	var connNodes []*figNode  // per connection: the connection's node, nil when its two ends join directly
	for _, con := range cons {
		var anchors []*figNode
		for _, end := range con.ends {
			if len(end.path) == 0 {
				continue // connects nothing
			}
			if pid := endPort(end.path, portSet); pid != "" {
				anchors = append(anchors, byID[pid])
				continue
			}
			leaf := end.path[len(end.path)-1]
			chain, err := owners(e, leaf)
			if err != nil {
				return t, err
			}
			if len(chain) == 0 {
				continue // a top-level feature: no owner to draw it in
			}
			obox, err := newNode(chain[len(chain)-1], true)
			if err != nil {
				return t, err
			}
			leafBox := false
			if mc, err := e.metaclass(leaf); err != nil {
				return t, err
			} else {
				leafBox = isPartLike(mc)
			}
			lnode, err := newNode(leaf, leafBox)
			if err != nil {
				return t, err
			}
			pend = append(pend, pending{obox, chain[:len(chain)-1]}, pending{lnode, chain})
			anchors = append(anchors, lnode)
		}
		// More than two ends: the connection is drawn as a node of its own
		// at the top of the figure, the ends joining it one edge each. Not
		// in parentOf, so the roots below make it a root.
		if len(anchors) > 2 {
			nm, err := e.names.name(con.id)
			if err != nil {
				return t, err
			}
			n := &figNode{elem: con.id, kind: "«" + figKeyword(con.meta) + "»", name: nm}
			order = append(order, n)
			connNodes = append(connNodes, n)
		} else {
			connNodes = append(connNodes, nil)
		}
		anchored = append(anchored, anchors)
	}
	place := func(n *figNode, chain []string) {
		if n == root {
			return
		}
		for i := len(chain) - 1; i >= 0; i-- {
			if a := byID[chain[i]]; a != nil && a.box {
				parentOf[n] = a
				return
			}
		}
		// no drawn ancestor: the top of the figure, beside the part's box
	}
	for _, p := range pend {
		place(p.n, p.chain)
	}
	t.roots = []*figNode{root}
	for _, n := range order[1:] {
		if slices.Contains(connNodes, n) {
			continue // the connection's node is placed below, by its owners
		}
		if p := parentOf[n]; p != nil {
			p.children = append(p.children, n)
		} else {
			t.roots = append(t.roots, n)
		}
	}
	// A connection's node is drawn where the model owns it: inside its
	// nearest drawn box ancestor, the part's box included, else the top of
	// the figure.
	for _, cn := range connNodes {
		if cn == nil {
			continue
		}
		own, err := owners(e, cn.elem)
		if err != nil {
			return t, err
		}
		placed := false
		for i := len(own) - 1; i >= 0 && !placed; i-- {
			if b := byID[own[i]]; b != nil && b.box {
				b.children = append(b.children, cn)
				placed = true
			}
		}
		if !placed {
			t.roots = append(t.roots, cn)
		}
	}
	for i, n := range order {
		n.id = "n" + fmt.Sprint(i)
	}
	for k, anchors := range anchored {
		if cn := connNodes[k]; cn != nil {
			// One edge per end, the anchor joining the connection's node;
			// two ends on the same anchor draw one edge.
			seen := map[string]bool{}
			for _, a := range anchors {
				if seen[a.id] {
					continue
				}
				seen[a.id] = true
				t.edges = append(t.edges, [2]string{a.id, cn.id})
			}
			continue
		}
		for i := 0; i < len(anchors); i++ {
			for j := i + 1; j < len(anchors); j++ {
				if anchors[i].id == anchors[j].id {
					continue
				}
				t.edges = append(t.edges, [2]string{anchors[i].id, anchors[j].id})
			}
		}
	}
	return t, nil
}

// endPort is the first of ports an end's path references, "" when it
// references none.
func endPort(path []string, ports map[string]bool) string {
	for _, id := range path {
		if ports[id] {
			return id
		}
	}
	return ""
}

// commonAncestor is the deepest element every chain passes through: the
// chains are outermost first, so it is the last element their shared start
// holds. ok is false when they share none.
func commonAncestor(chains [][]string) (string, bool) {
	i := 0
loop:
	for i < len(chains[0]) {
		id := chains[0][i]
		for _, c := range chains[1:] {
			if i >= len(c) || c[i] != id {
				break loop // the chains diverge here: what they share is before i
			}
		}
		i++
	}
	if i == 0 {
		return "", false
	}
	return chains[0][i-1], true
}

// figNode is one drawn element of a figure: a part drawn as a box (a
// Mermaid subgraph) or a feature drawn as a stadium node, with what is
// drawn inside it.
type figNode struct {
	id       string // the Mermaid id, n0, n1, ...
	elem     string // the model element's id
	kind     string // the notation keyword: «part», «port», ...
	name     string // the element's name, "" when it has none
	box      bool   // a part: drawn as a box
	href     string // the link to the element's page, "" when there is none
	children []*figNode
}

// figTree is one figure: the qualified name of the element it is of, the
// drawn elements whose parent is the top of the figure (the ends' common
// ancestor for an interface, the part's box with the parts beside it for a
// part), and the edges: one per pair of ends for a two-ended connection,
// one per end into the interface's node for one with more.
type figTree struct {
	of    string
	roots []*figNode
	edges [][2]string
}

// figKeyword is the notation keyword of a metaclass, as OpenSysML writes it
// in a rendering: part of PartUsage, port of PortUsage; the Usage or
// Definition suffix and the capitalisation go.
func figKeyword(metaclass string) string {
	k := strings.TrimSuffix(strings.TrimSuffix(metaclass, "Usage"), "Definition")
	return strings.ToLower(k)
}
