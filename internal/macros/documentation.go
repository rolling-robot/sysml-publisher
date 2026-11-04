package macros

import (
	"fmt"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

// Doc is the block macro sysml-doc::[], replaced by the doc comments of the
// page's element, each parsed as AsciiDoc (MC-10).
const Doc = "sysml-doc"

// DocRole is the role of the block the documentation is put in, for styling.
const DocRole = "sysml-doc"

func registerDoc(g *asciidoctor.ExtensionGroup, t *tree.Tree) {
	g.BlockMacro(Doc, asciidoctor.BlockMacroProcessorFunc(
		func(parent asciidoctor.StructuralNode, _ string, _ map[string]any) (asciidoctor.StructuralNode, error) {
			n, err := pageElement(parent, t)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", Doc, err)
			}
			if len(n.Docs) == 0 {
				return nil, nil // nothing to insert; the macro line is dropped
			}
			block, err := asciidoctor.CreateOpenBlock(parent, nil, asciidoctor.Attributes{"role": DocRole}, nil)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", Doc, err)
			}
			for _, doc := range n.Docs {
				if err := asciidoctor.ParseContent(block, doc, nil); err != nil {
					return nil, fmt.Errorf("%s: parsing the doc of %s: %w", Doc, n.QualifiedName, err)
				}
			}
			return block, nil
		}), nil)
}
