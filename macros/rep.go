package macros

import (
	"fmt"
	"html"
	"strings"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/model"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

// Rep is the inline macro sysml-rep:[<element>][<language>], replaced by the
// body of the textual representation in that language that the element owns
// (MC-16): the page's element without a target, else the element the target
// names, read as sysml-name reads it (MC-8, MC-15). The body is inserted as
// plain text, not read as AsciiDoc.
const Rep = "sysml-rep"

func registerRep(g *asciidoctor.ExtensionGroup, t *tree.Tree) {
	md := newMetadataIndex(t.Model)
	reps := repsByOwner(t.Model)
	g.InlineMacro(Rep, asciidoctor.InlineMacroProcessorFunc(
		func(parent asciidoctor.Node, target string, attrs map[string]any) (*asciidoctor.PhraseNode, error) {
			n, err := pageElement(parent, t)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", Rep, err)
			}
			// Inline macros run after special characters are escaped.
			target = html.UnescapeString(target)
			language, _ := attrs["language"].(string)
			language = strings.TrimSpace(html.UnescapeString(language))
			macro := fmt.Sprintf("%s:%s[%s]", Rep, target, language)
			if language == "" {
				return nil, fmt.Errorf("%s on the page of %s: no language named; write %s:[<element>][<language>]", macro, n.QualifiedName, Rep)
			}
			id, name := n.ID, n.QualifiedName
			if target != "" {
				if id, err = nameTarget(target, n, md); err != nil {
					return nil, fmt.Errorf("%s on the page of %s: %w", macro, n.QualifiedName, err)
				}
				name = md.names.qualifiedName(id)
			}
			var bodies []string
			for _, r := range reps[id] {
				if strings.EqualFold(r.Str("language"), language) {
					bodies = append(bodies, strings.TrimSpace(r.Str("body")))
				}
			}
			if len(bodies) == 0 {
				return nil, fmt.Errorf("%s on the page of %s: %s has no representation in language %q", macro, n.QualifiedName, name, language)
			}
			return asciidoctor.CreatePhraseNode(parent, "quoted", html.EscapeString(strings.Join(bodies, "\n")), nil, nil)
		}), &asciidoctor.MacroOptions{Regexp: optionalTargetRx(Rep), PositionalAttributes: []string{"language"}})
}

// repsByOwner are the textual representations of every element, in any
// language, by owner id, in model order.
func repsByOwner(m *model.Model) map[string][]model.Element {
	out := map[string][]model.Element{}
	for _, id := range m.Order {
		if e := m.Elements[id]; e.Type() == "TextualRepresentation" {
			out[e.Ref("owner")] = append(out[e.Ref("owner")], e)
		}
	}
	return out
}
