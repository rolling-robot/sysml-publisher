package macros

import (
	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

// ShortName is the inline macro sysml-short-name:[], replaced by the short
// name of the page's element, or by nothing when it has none (MC-11).
const ShortName = "sysml-short-name"

func registerShortName(g *asciidoctor.ExtensionGroup, t *tree.Tree) {
	registerText(g, t, ShortName, func(n *tree.Node) string { return n.ShortName })
}
