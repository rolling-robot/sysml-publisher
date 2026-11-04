package macros

import (
	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

// Name is the inline macro sysml-name:[], replaced by the name of the page's
// element (MC-8).
const Name = "sysml-name"

func registerName(g *asciidoctor.ExtensionGroup, t *tree.Tree) {
	registerText(g, t, Name, func(n *tree.Node) string { return n.Name })
}
