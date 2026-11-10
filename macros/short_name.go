package macros

import (
	asciidoctor "github.com/rolling-robot/asciidoctorg"
)

// ShortName is the inline macro sysml-short-name:[<element>], replaced by the
// short name of the page's element, or of the element the target names
// (read as sysml-name reads it), or by nothing when it has none (MC-11,
// MC-15).
const ShortName = "sysml-short-name"

func registerShortName(g *asciidoctor.ExtensionGroup, e *env) {
	registerText(g, e, ShortName, func(id string) (string, error) { return e.m.ShortName(e.ctx, id) })
}
