package macros_test

import (
	"html"
	"strings"
	"testing"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/macros"
)

// wiki is a converter for another output: it writes the links between pages
// (anchors with macros.AttrLink) its own way, as a Confluence converter
// writes <ac:link>, and hands everything else to html5.
type wiki struct{ html *asciidoctor.BuiltinConverter }

func (w wiki) Convert(n asciidoctor.Node, transform string, opts map[string]any) (string, error) {
	if transform == "inline_anchor" {
		if to, ok := n.Attr(macros.AttrLink).(string); ok {
			return `<page-link to="` + html.EscapeString(to) + `">` + n.(*asciidoctor.PhraseNode).Text() + `</page-link>`, nil
		}
	}
	return w.html.Convert(n, transform, opts)
}

// Links between pages are anchor nodes the document's converter writes
// (MC-18): html5 gives <a href>, another converter its own markup, for
// every macro that links, custom ones included. Other links stay the
// converter's ordinary links.
func TestLinksAreWrittenByTheConverter(t *testing.T) {
	h := newHost(t, t.Context())
	base, err := h.ad.BuiltinConverter("xhtml5")
	if err != nil {
		t.Fatal(err)
	}
	if err := h.ad.RegisterConverter(wiki{base}, &asciidoctor.BackendTraits{Basebackend: "html", HTMLSyntax: "xml"}, "wiki"); err != nil {
		t.Fatal(err)
	}
	if err := h.r.Inline("my-link", func(c *macros.Call) (string, error) {
		el, err := c.Element()
		if err != nil {
			return "", err
		}
		return c.Link(el.ID, "the "+html.EscapeString(c.Target)), nil
	}); err != nil {
		t.Fatal(err)
	}
	p, err := h.m.Props(t.Context(), "Vehicles", "name")
	if err != nil {
		t.Fatal(err)
	}
	pkg := p.Element.ID
	link := func(to, text string) string { return `<page-link to="` + to + `">` + text + `</page-link>` }

	cases := []struct {
		name, src, page string
		html, wiki      string
	}{
		{"sysml-name", "sysml-name:Wheel[]", h.car,
			`<a href="wheel.html">Wheel</a>`, link(h.wheel, "Wheel")},
		{"sysml-members-list", "sysml-members-list::[]", pkg,
			`<a href="wheel.html">Wheel</a>`, link(h.wheel, "Wheel")},
		{"custom", "my-link:Wheel[]", h.car,
			`<a href="wheel.html">the Wheel</a>`, link(h.wheel, "the Wheel")},
		{"not a page link", "https://example.org[elsewhere]", h.car,
			`<a href="https://example.org">elsewhere</a>`, `<a href="https://example.org">elsewhere</a>`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			for _, out := range []struct{ backend, want string }{{"html5", c.html}, {"wiki", c.wiki}} {
				got, err := h.ad.Convert(c.src, &asciidoctor.Options{Backend: out.backend, Attributes: macros.Attributes(c.page)})
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(got, out.want) {
					t.Errorf("%s: %q does not contain %q", out.backend, got, out.want)
				}
				if len(h.warnings) != 0 {
					t.Errorf("%s: warnings %q", out.backend, h.warnings)
				}
			}
		})
	}
}
