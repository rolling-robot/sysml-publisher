package macros_test

import (
	"context"
	"html"
	"slices"
	"strings"
	"testing"

	"github.com/Open-MBEE/OpenSysML/client/opensysml"
	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/macros"
	"github.com/rolling-robot/sysml-publisher/sysml"
)

// host is what another module sets up: the model, an asciidoctorg instance
// with the macros, its pages, and the warnings it received.
type host struct {
	m        *sysml.Model
	ad       *asciidoctor.Asciidoctor
	r        *macros.Registry
	car      string // element ids
	wheel    string
	warnings []string
}

func newHost(t *testing.T, ctx context.Context) *host {
	t.Helper()
	client, err := opensysml.New()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := client.ParseFiles(ctx, []string{"../testdata/external/vehicles.sysml"})
	if err != nil {
		client.Close()
		t.Fatal(err)
	}
	m, err := sysml.Open(ctx, client, parsed)
	if err != nil {
		client.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	h := &host{m: m}
	for qn, id := range map[string]*string{"Vehicles::Car": &h.car, "Vehicles::Wheel": &h.wheel} {
		p, err := m.Props(ctx, qn, "name")
		if err != nil {
			t.Fatal(err)
		}
		*id = p.Element.ID
	}
	if h.ad, err = asciidoctor.New(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { h.ad.Close() })
	site := pages{h.car: "car.html", h.wheel: "wheel.html"}
	if h.r, err = macros.Register(ctx, h.ad, m, site, func(w macros.Warning) { h.warnings = append(h.warnings, w.String()) }); err != nil {
		t.Fatal(err)
	}
	return h
}

// convert converts src for the element id, the warnings reset.
func (h *host) convert(t *testing.T, src, id string) string {
	t.Helper()
	h.warnings = nil
	out, err := h.ad.Convert(src, &asciidoctor.Options{Attributes: macros.Attributes(id)})
	if err != nil {
		t.Fatalf("converting %q: %v", src, err)
	}
	return out
}

// A custom inline macro gets what sysml-* macros have (MC-20): the element
// the target names, read as sysml-name reads it, its name, links to pages,
// evaluation formatted as sysml-value formats it, and attributes.
func TestCustomInline(t *testing.T) {
	h := newHost(t, t.Context())
	err := h.r.Inline("my-name", func(c *macros.Call) (string, error) {
		el, err := c.Element()
		if err != nil {
			return "", err
		}
		name, err := c.Name(el.ID)
		if err != nil {
			return "", err
		}
		if prefix := c.Attr("prefix"); prefix != "" {
			name = prefix + " " + name
		}
		return c.Link(el.ID, html.EscapeString(name)), nil
	}, "prefix")
	if err != nil {
		t.Fatal(err)
	}
	err = h.r.Inline("my-mass", func(c *macros.Call) (string, error) {
		v, err := c.Evaluate(c.Page, "mass")
		if err != nil {
			return "", err
		}
		return html.EscapeString(macros.FormatValue(v)), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, src, page, want string }{
		{"page's element: no link to itself", "my-name:[]", h.car, "<p>Car</p>"},
		{"short name, linked", "my-name:W[]", h.car, `<p><a href="wheel.html">Wheel</a></p>`},
		{"relative name, no page: no link", "my-name:front[]", h.car, "<p>front</p>"},
		{"positional attribute", "my-name:W[The]", h.car, `<p><a href="wheel.html">The Wheel</a></p>`},
		{"named attribute, trimmed, escaped", "my-name:[prefix= a<b ]", h.car, "<p>a&lt;b Car</p>"},
		{"evaluation, formatted", "Mass: my-mass:[].", h.car, "<p>Mass: 1200.</p>"},
		{"inside text", "The my-name:W[] rolls.", h.car, `<p>The <a href="wheel.html">Wheel</a> rolls.</p>`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if out := h.convert(t, c.src, c.page); !strings.Contains(out, c.want) {
				t.Errorf("output %q lacks %q", out, c.want)
			}
			if h.warnings != nil {
				t.Errorf("warnings: %q", h.warnings)
			}
		})
	}
}

// A custom macro that fails leaves a placeholder and a warning, as a
// sysml-* macro does (MC-19, MC-20), and the rest is converted.
func TestCustomFailure(t *testing.T) {
	h := newHost(t, t.Context())
	err := h.r.Inline("my-name", func(c *macros.Call) (string, error) {
		el, err := c.Element()
		if err != nil {
			return "", err
		}
		return html.EscapeString(el.QualifiedName), nil
	})
	if err != nil {
		t.Fatal(err)
	}
	out := h.convert(t, "A my-name:Boat[] and my-name:W[].", h.car)
	want := `<p>A <span class="sysml-failed">my-name:Boat[]: the model has no element named &#34;Boat&#34;</span> and Vehicles::Wheel.</p>`
	if !strings.Contains(out, want) {
		t.Errorf("output %q lacks %q", out, want)
	}
	if w := []string{`Vehicles::Car: my-name:Boat[]: the model has no element named "Boat"`}; !slices.Equal(h.warnings, w) {
		t.Errorf("warnings %q, want %q", h.warnings, w)
	}
}

// A custom block macro builds a block under its parent; Resolve and
// Passthrough work as for sysml-members-list; Warn reports without failing.
func TestCustomBlock(t *testing.T) {
	h := newHost(t, t.Context())
	err := h.r.Block("my-parts", func(c *macros.Call, parent asciidoctor.StructuralNode) (asciidoctor.StructuralNode, error) {
		owner := c.Page
		if c.Target != "" {
			var err error
			if owner, err = c.Resolve(c.Target); err != nil {
				return nil, err
			}
		}
		members, err := c.Model().Members(c.Context(), owner.ID)
		if err != nil {
			return nil, err
		}
		list, err := asciidoctor.CreateList(parent, "ulist", nil)
		if err != nil {
			return nil, err
		}
		for _, m := range members {
			if m.Metaclass != "PartUsage" {
				continue
			}
			name, err := c.Name(m.ID)
			if err != nil {
				return nil, err
			}
			item, err := asciidoctor.CreateListItem(list, macros.Passthrough(html.EscapeString(name+" *x*")))
			if err != nil {
				return nil, err
			}
			list.AddItem(item)
		}
		if len(list.Items()) == 0 {
			c.Warn(owner.QualifiedName + " has no parts")
			return nil, nil
		}
		return list, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Run("list, markup left alone", func(t *testing.T) {
		out := h.convert(t, "my-parts::[]", h.car)
		for _, want := range []string{"<p>front *x*</p>", "<p>rear *x*</p>"} {
			if !strings.Contains(out, want) {
				t.Errorf("output %q lacks %q", out, want)
			}
		}
	})
	t.Run("warning without failing", func(t *testing.T) {
		out := h.convert(t, "Before.\n\nmy-parts::W[]", h.car)
		if strings.Contains(out, "sysml-failed") || !strings.Contains(out, "<p>Before.</p>") {
			t.Errorf("output %q: want Before. and no placeholder", out)
		}
		if w := []string{"Vehicles::Car: my-parts::W[]: Vehicles::Wheel has no parts"}; !slices.Equal(h.warnings, w) {
			t.Errorf("warnings %q, want %q", h.warnings, w)
		}
	})
	t.Run("failure", func(t *testing.T) {
		out := h.convert(t, "my-parts::Boat[]", h.car)
		want := `<div class="sysml-failed"><p>my-parts::Boat[]: the model has no element named &#34;Boat&#34;</p></div>`
		if !strings.Contains(out, want) {
			t.Errorf("output %q lacks %q", out, want)
		}
	})
}

// Custom names must be macro names, and not take the sysml- prefix, which
// is the package's (MC-1, MC-20).
func TestCustomNames(t *testing.T) {
	h := newHost(t, t.Context())
	nop := func(*macros.Call) (string, error) { return "", nil }
	for _, name := range []string{"sysml-mine", "SYSML-mine", "", "1x", "my macro", "my:x"} {
		if err := h.r.Inline(name, nop); err == nil {
			t.Errorf("Inline(%q) accepted", name)
		}
	}
	if err := h.r.Inline("my_x-2", nop); err != nil {
		t.Errorf("Inline(my_x-2): %v", err)
	}
}
