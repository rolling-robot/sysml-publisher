package macros

import (
	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

// QualifiedName is the inline macro sysml-qualified-name:[<element>],
// replaced by the qualified name of the page's element, or of the element
// the target names, read as sysml-name reads it (MC-9, MC-15).
const QualifiedName = "sysml-qualified-name"

func registerQualifiedName(g *asciidoctor.ExtensionGroup, t *tree.Tree) {
	registerText(g, t, QualifiedName,
		func(n *tree.Node) string { return n.QualifiedName },
		func(_ *metadataIndex, id string) string { return t.QualifiedName(id) })
}
