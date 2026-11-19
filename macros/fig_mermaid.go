// Copyright 2026 Petr Mukhachev
// SPDX-License-Identifier: MIT

package macros

import (
	"fmt"
	"strings"
)

// The Mermaid figure of an interface (MC-25): a flowchart written in the
// look of OpenSysML's Mermaid interconnection form. The style constants
// below are that writer's frontmatter (OpenSysML internal/ir/view/
// mermaid.go, writeMermaidFrontmatter, at the commit go.mod pins): white
// fills, thin #181818 lines, Helvetica. If OpenSysML restyles its writer,
// copy the frontmatter again.
const mermaidConfig = `config:
  fontFamily: "Helvetica, Arial, sans-serif"
  theme: base
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

// mermaidFigure is the Mermaid source of a figure: the frontmatter with the
// interface for the title, a flowchart with a subgraph per part box
// (containment nested), a stadium node per end feature, and one thick edge
// per pair of ends, the way OpenSysML draws a connection.
func mermaidFigure(t figTree) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString(mermaidConfig)
	fmt.Fprintf(&b, "title: %s\n", mermaidYAMLString("«interface» "+t.iface))
	fmt.Fprintf(&b, "---\n%%%% %s — interconnection figure\n", t.iface)
	b.WriteString("flowchart LR\n")
	var write func(nodes []*figNode, indent string)
	write = func(nodes []*figNode, indent string) {
		for _, n := range nodes {
			if n.box {
				fmt.Fprintf(&b, "%ssubgraph %s [%s]\n", indent, n.id, figLabel(n.kind, n.name, false))
				fmt.Fprintf(&b, "%s  direction LR\n", indent)
				write(n.children, indent+"  ")
				fmt.Fprintf(&b, "%send\n", indent)
			} else {
				fmt.Fprintf(&b, "%s%s(%s)\n", indent, n.id, figLabel(n.kind, n.name, true))
			}
		}
	}
	write(t.roots, "  ")
	for i, edge := range t.edges {
		fmt.Fprintf(&b, "  %s === %s\n", edge[0], edge[1])
		fmt.Fprintf(&b, "  linkStyle %d stroke-width:3px\n", i)
	}
	return b.String()
}

// figLabel is a node's or box's label: the keyword line in italics and, when
// the element has a name, the name in bold. A box's label has both on one
// line; a stadium's carries the name on a second line. The text goes in a
// Mermaid markdown string ("`…`"), which takes a newline literally; a text
// with a backtick or a double quote cannot go in one, so such a label falls
// back to a double-quoted string, the text HTML-escaped and the newline a
// <br/>.
func figLabel(keyword, name string, multiline bool) string {
	text := "*" + keyword + "*"
	if name != "" {
		if multiline {
			text += "\n"
		} else {
			text += " "
		}
		text += "**" + name + "**"
	}
	if strings.ContainsAny(text, "`\"&<>") {
		return `"` + htmlEscapeLabel(text) + `"`
	}
	return "`" + text + "`"
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

// mermaidYAMLString writes s as a YAML double-quoted scalar: the characters
// YAML reads, escaped.
func mermaidYAMLString(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
