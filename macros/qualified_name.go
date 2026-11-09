package macros

import (
	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

// QualifiedName is the inline macro sysml-qualified-name:[<element>],
// replaced by the qualified name of the page's element, or of the element
// the target names, read as sysml-name reads it (MC-9, MC-15).
const QualifiedName = "sysml-qualified-name"

func registerQualifiedName(g *asciidoctor.ExtensionGroup, e *env) {
	registerText(g, e, QualifiedName,
		func(n *tree.Node) string { return n.QualifiedName },
		e.names.qualifiedName)
}
