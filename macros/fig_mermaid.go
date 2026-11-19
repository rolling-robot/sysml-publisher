// Copyright 2026 Petr Mukhachev
// SPDX-License-Identifier: MIT

package macros

import (
	"fmt"
	"strings"
)

// The Mermaid figure of an interface (MC-25): a flowchart written in the
// look of OpenSysML's Mermaid interconnection form. The style constants
// below pin what that writer's frontmatter states (OpenSysML internal/ir/
// view/mermaid.go, writeMermaidFrontmatter, at the commit go.mod pins):
// white fills, thin #181818 lines, Helvetica. The layout is ELK (mermaid
// 12's own), which places a box's contents below all of its title's lines;
// the look classic and the classDefs pin the Pilot's white-on-#181818 on
// top of the theme a mermaid version defaults to. If OpenSysML restyles
// its writer, copy the frontmatter again.
const mermaidConfig = `config:
  fontFamily: "Helvetica, Arial, sans-serif"
  theme: base
  look: classic
  layout: elk
  themeCSS: ".edgeLabel rect { opacity: 1 !important; } .cluster-label .nodeLabel { text-align: center; }"
  themeVariables:
    fontFamily: "Helvetica, Arial, sans-serif"
    fontSize: "14px"
    primaryColor: "#FFFFFF"
    secondaryColor: "#FFFFFF"
    tertiaryColor: "#FFFFFF"
    background: "#FFFFFF"
    primaryBorderColor: "#181818"
    primaryTextColor: "#000000"
    lineColor: "#181818"
    textColor: "#000000"
    noteBkgColor: "#FEFFDD"
    noteBorderColor: "#181818"
    noteTextColor: "#000000"
    clusterBkg: "#FFFFFF"
    clusterBorder: "#181818"
    edgeLabelBackground: "#FFFFFF"
`

// The classDefs the figure states, so its boxes and its stadium nodes draw
// in the Pilot's white-on-#181818 whatever the theme and look a mermaid
// version fills from.
const (
	mermaidBoxClass  = "classDef figbox fill:#FFFFFF,stroke:#181818,color:#000000\n"
	mermaidPortClass = "classDef figport fill:#FFFFFF,stroke:#181818,color:#000000\n"
)

// mermaidFigure is the Mermaid source of a figure: the style frontmatter, a
// flowchart with a subgraph per part box (containment nested), a stadium
// node per end feature, and one thick edge per pair of ends, the way
// OpenSysML draws a connection. The figure carries no title: the page names
// the interface.
func mermaidFigure(t figTree) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(mermaidConfig)
	fmt.Fprintf(&b, "---\n%%%% %s — interconnection figure\n", t.iface)
	b.WriteString("flowchart LR\n")
	var boxes, ports []string
	var write func(nodes []*figNode, indent string)
	write = func(nodes []*figNode, indent string) {
		for _, n := range nodes {
			if n.box {
				boxes = append(boxes, n.id)
				fmt.Fprintf(&b, "%ssubgraph %s [%s]\n", indent, n.id, mermaidString(figLabel(n.kind, n.name)))
				fmt.Fprintf(&b, "%s  direction LR\n", indent)
				write(n.children, indent+"  ")
				fmt.Fprintf(&b, "%send\n", indent)
			} else {
				ports = append(ports, n.id)
				fmt.Fprintf(&b, "%s%s(%s)\n", indent, n.id, mermaidString(figLabel(n.kind, n.name)))
			}
		}
	}
	write(t.roots, "  ")
	for i, edge := range t.edges {
		fmt.Fprintf(&b, "  %s === %s\n", edge[0], edge[1])
		fmt.Fprintf(&b, "  linkStyle %d stroke-width:3px\n", i)
	}
	if len(boxes) > 0 {
		b.WriteString(mermaidBoxClass)
		fmt.Fprintf(&b, "class %s figbox\n", strings.Join(boxes, ","))
	}
	if len(ports) > 0 {
		b.WriteString(mermaidPortClass)
		fmt.Fprintf(&b, "class %s figport\n", strings.Join(ports, ","))
	}
	return b.String()
}

// figLabel is a node's or box's label text: the keyword line in italics and,
// when the element has a name, the name in bold on a line under it.
func figLabel(kind, name string) string {
	text := "*" + kind + "*"
	if name != "" {
		text += "\n**" + name + "**"
	}
	return text
}

// mermaidString is text as a Mermaid string: a markdown string ("`…`",
// written inside the double quotes every Mermaid string is written in) when
// text holds nothing a markdown string cannot — a backtick, a double quote,
// <, > or & — else a double-quoted string with the text HTML-escaped and the
// newline a <br/>. A box's and a stadium's label are the same string; a
// subgraph's title carries it in [..], a stadium's in (..).
func mermaidString(text string) string {
	if strings.ContainsAny(text, "`\"&<>") {
		return `"` + htmlEscapeLabel(text) + `"`
	}
	return "\"`" + text + "`\""
}

// htmlEscapeLabel escapes a label for a double-quoted Mermaid string: the
// characters HTML reads, the double quote included, and the newline as a
// line break.
func htmlEscapeLabel(text string) string {
	text = strings.ReplaceAll(text, "&", "&amp;")
	text = strings.ReplaceAll(text, "<", "&lt;")
	text = strings.ReplaceAll(text, ">", "&gt;")
	text = strings.ReplaceAll(text, `"`, "&quot;")
	return strings.ReplaceAll(text, "\n", "<br/>")
}
