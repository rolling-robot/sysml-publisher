package macros

import (
	"fmt"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
)

// Doc is the block macro sysml-doc::[<element>][], replaced by the doc
// comments of the page's element, or of the element the target names (read
// as sysml-name reads it), each parsed as AsciiDoc (MC-10, MC-15). Macros in
// them still describe the page's element.
const Doc = "sysml-doc"

// DocRole is the role of the block the documentation is put in, for styling.
const DocRole = "sysml-doc"

func registerDoc(g *asciidoctor.ExtensionGroup, e *env) {
	block(g, e, Doc, nil, nil,
		func(c call[asciidoctor.StructuralNode]) (asciidoctor.StructuralNode, error) {
			id, owner := c.page.id, c.page.qualifiedName
			if c.target != "" {
				var err error
				if id, err = nameTarget(c.target, c.page, e); err != nil {
					return nil, err
				}
				if owner, err = e.names.qualifiedName(id); err != nil {
					return nil, err
				}
			}
			docs, err := e.m.Docs(e.ctx, id)
			if err != nil {
				return nil, err
			}
			if len(docs) == 0 {
				return nil, nil // nothing to insert; the macro line is dropped
			}
			block, err := asciidoctor.CreateOpenBlock(c.parent, nil, asciidoctor.Attributes{"role": DocRole}, nil)
			if err != nil {
				return nil, err
			}
			for _, doc := range docs {
				if err := asciidoctor.ParseContent(block, doc, nil); err != nil {
					return nil, fmt.Errorf("parsing the doc of %s: %w", owner, err)
				}
			}
			return block, nil
		})
}
