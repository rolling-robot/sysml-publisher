package macros

import (
	"fmt"
	"html"
	"strings"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
)

// Rep is the inline macro sysml-rep:[<element>][<language>], replaced by the
// body of the textual representation in that language that the element owns
// (MC-16): the page's element without a target, else the element the target
// names, read as sysml-name reads it (MC-8, MC-15). The body is inserted as
// plain text, not read as AsciiDoc.
const Rep = "sysml-rep"

func registerRep(g *asciidoctor.ExtensionGroup, e *env) {
	g.InlineMacro(Rep, asciidoctor.InlineMacroProcessorFunc(
		func(parent asciidoctor.Node, target string, attrs map[string]any) (*asciidoctor.PhraseNode, error) {
			n, err := pageElement(parent, e.t)
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
			fail := func(err error) error {
				return fmt.Errorf("%s on the page of %s: %w", macro, n.QualifiedName, err)
			}
			id, name := n.ID, n.QualifiedName
			if target != "" {
				if id, err = nameTarget(target, n, e); err != nil {
					return nil, fail(err)
				}
				if name, err = e.names.qualifiedName(id); err != nil {
					return nil, fail(err)
				}
			}
			reps, err := e.m.Representations(e.ctx, id)
			if err != nil {
				return nil, fail(err)
			}
			var bodies []string
			for _, r := range reps {
				if strings.EqualFold(r.Language, language) {
					bodies = append(bodies, strings.TrimSpace(r.Body))
				}
			}
			if len(bodies) == 0 {
				return nil, fmt.Errorf("%s on the page of %s: %s has no representation in language %q", macro, n.QualifiedName, name, language)
			}
			return asciidoctor.CreatePhraseNode(parent, "quoted", html.EscapeString(strings.Join(bodies, "\n")), nil, nil)
		}), &asciidoctor.MacroOptions{Regexp: optionalTargetRx(Rep), PositionalAttributes: []string{"language"}})
}
