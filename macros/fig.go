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
// is an interface, by a Mermaid figure of it (MC-25): the parts on its ends
// and the parts between them and their common ancestor, each end's feature,
// and one edge per pair of ends, drawn by the site's mermaid.js in the look
// of OpenSysML's Mermaid interconnection form. Without a target the page's
// element is used; a target is read as sysml-name reads it (MC-8, MC-15).
// A resolvable target that is not an interface is written out as the macro
// text, no warning raised. The figure draws exactly what the ends' chains
// pass through; the model is read the way sysml-ifaces reads it, the
// connections indexed once for every page (MC-3).
const Fig = "sysml-fig"

// FigRole is the role of the block a figure is put in, for styling.
const FigRole = "sysml-fig"

// AttrFig is the attribute of the block a figure is put in: the qualified
// name of the interface the figure is of. As links are anchor nodes
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
// resolvable one that is not an interface is written out; an interface is
// figured.
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
	if meta != "InterfaceUsage" {
		return figEcho(c.parent, c.macro)
	}
	ends, err := connectorEnds(e, id)
	if err != nil {
		return nil, err
	}
	t, err := e.figure(name, ends)
	if err != nil {
		return nil, err
	}
	return figBlock(c.parent, t)
}

// figEcho is the macro written out: a resolvable target that is not an
// interface leaves the text of the macro call where it was, no warning.
func figEcho(parent asciidoctor.StructuralNode, macro string) (asciidoctor.StructuralNode, error) {
	return asciidoctor.CreatePassBlock(parent,
		`<p class="`+FigRole+`-text">`+html.EscapeString(macro)+`</p>`, nil, nil)
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
	b.SetAttr(AttrFig, t.iface, true)
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
// first, as connectorEnds reads them.
func (e *env) figure(iface string, ends []connectorEnd) (figTree, error) {
	t := figTree{iface: iface}
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
	// The drawn elements are the chains cut below the common ancestor: one
	// node per element, each standing where it stands in its chain. An
	// element two ends reach is drawn once. A package in a chain is not
	// drawn — an interconnection shows no namespaces — so what it owns
	// stands where the package would have stood.
	// Every chain shares its start with the first, and the common ancestor
	// is the deepest of what they share, so it sits at the same index of
	// every chain; cutting there leaves what is drawn.
	k := slices.Index(chains[0], common)
	for i := range chains {
		chains[i] = chains[i][k+1:]
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
				if meta == "Package" {
					prev = id // not drawn: its members stand where it would
					continue
				}
				nm, err := e.names.name(id)
				if err != nil {
					return t, err
				}
				n = &figNode{elem: id, kind: "«" + figKeyword(meta) + "»", name: nm, box: isPartLike(meta)}
				byID[id] = n
				parentOf[id] = prev
				order = append(order, n)
			}
			prev = id
		}
	}
	for i, n := range order {
		n.id = "n" + fmt.Sprint(i)
	}
	// What a drawn element is nested in: its nearest drawn ancestor that
	// holds things, a part box; the top of the figure for one whose
	// ancestors are all packages or features.
	for _, n := range order {
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
	return t, nil
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
	children []*figNode
}

// figTree is one figure: what the interface is of, the drawn elements whose
// parent is the common ancestor (which itself is not drawn), and one edge
// per pair of ends, between the end features' node ids.
type figTree struct {
	iface string
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
