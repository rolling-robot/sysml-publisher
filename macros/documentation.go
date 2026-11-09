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
	g.BlockMacro(Doc, asciidoctor.BlockMacroProcessorFunc(
		func(parent asciidoctor.StructuralNode, target string, _ map[string]any) (asciidoctor.StructuralNode, error) {
			n, err := pageElement(parent, e.t)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", Doc, err)
			}
			docs, owner := n.Docs, n.QualifiedName
			if target != "" {
				id, err := nameTarget(target, n, e)
				if err == nil {
					docs, err = e.m.Docs(e.ctx, id)
				}
				if err == nil {
					owner, err = e.names.qualifiedName(id)
				}
				if err != nil {
					return nil, fmt.Errorf("%s::%s[] on the page of %s: %w", Doc, target, n.QualifiedName, err)
				}
			}
			if len(docs) == 0 {
				return nil, nil // nothing to insert; the macro line is dropped
			}
			block, err := asciidoctor.CreateOpenBlock(parent, nil, asciidoctor.Attributes{"role": DocRole}, nil)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", Doc, err)
			}
			for _, doc := range docs {
				if err := asciidoctor.ParseContent(block, doc, nil); err != nil {
					return nil, fmt.Errorf("%s: parsing the doc of %s: %w", Doc, owner, err)
				}
			}
			return block, nil
		}), nil)
}
