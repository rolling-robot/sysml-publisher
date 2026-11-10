package macros

import (
	asciidoctor "github.com/rolling-robot/asciidoctorg"
)

// QualifiedName is the inline macro sysml-qualified-name:[<element>],
// replaced by the qualified name of the page's element, or of the element
// the target names, read as sysml-name reads it (MC-9, MC-15).
const QualifiedName = "sysml-qualified-name"

func registerQualifiedName(g *asciidoctor.ExtensionGroup, e *env) {
	registerText(g, e, QualifiedName, e.names.qualifiedName)
}
