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
	inline(g, e, Rep, &asciidoctor.MacroOptions{Regexp: optionalTargetRx(Rep), PositionalAttributes: []string{"language"}},
		func(name, sep, target string, attrs map[string]any) string {
			return name + sep + target + "[" + repLanguage(attrs) + "]"
		},
		func(c call[asciidoctor.Node]) (string, error) {
			language := repLanguage(c.attrs)
			if language == "" {
				return "", fmt.Errorf("no language named; write %s:[<element>][<language>]", Rep)
			}
			id, name := c.page.id, c.page.qualifiedName
			if c.target != "" {
				var err error
				if id, err = nameTarget(c.target, c.page, e); err != nil {
					return "", err
				}
				if name, err = e.names.qualifiedName(id); err != nil {
					return "", err
				}
			}
			reps, err := e.m.Representations(e.ctx, id)
			if err != nil {
				return "", err
			}
			var bodies []string
			for _, r := range reps {
				if strings.EqualFold(r.Language, language) {
					bodies = append(bodies, strings.TrimSpace(r.Body))
				}
			}
			if len(bodies) == 0 {
				return "", fmt.Errorf("%s has no representation in language %q", name, language)
			}
			return html.EscapeString(strings.Join(bodies, "\n")), nil
		})
}

// repLanguage is the language attribute of a sysml-rep macro, unescaped
// (inline macros run after special characters are escaped) and trimmed.
func repLanguage(attrs map[string]any) string {
	language, _ := attrs["language"].(string)
	return strings.TrimSpace(html.UnescapeString(language))
}
