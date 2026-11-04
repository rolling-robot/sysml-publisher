package macros

import (
	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

// QualifiedName is the inline macro sysml-qualified-name:[], replaced by the
// qualified name of the page's element (MC-9).
const QualifiedName = "sysml-qualified-name"

func registerQualifiedName(g *asciidoctor.ExtensionGroup, t *tree.Tree) {
	registerText(g, t, QualifiedName, func(n *tree.Node) string { return n.QualifiedName })
}
