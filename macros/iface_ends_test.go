package macros

import (
	"html"
	"regexp"
	"slices"
	"strings"
	"testing"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
)

// listItemRx is the text of a ulist item, links and all.
var listItemRx = regexp.MustCompile(`(?s)<li>\s*<p>(.*?)</p>\s*</li>`)

// The expected lists are the // notes in testdata/iface-ends.
func TestIfaceEnds(t *testing.T) {
	tr, ad := setupFixture(t, "iface-ends")
	cases := []struct {
		name, page, target string
		want               []string
	}{
		{"connect (a, b): source and target", "I/sys", "i1", []string{"source: x.p", "target: y.p"}},
		{"connect a to b", "I/sys", "i2", []string{"source: x.q", "target: y.q"}},
		{"ends named by an interface def", "I/sys", "i4", []string{"a: x.p", "b: y.p", "c: z.p"}},
		{"n-ary, unnamed ends", "I/sys", "i5", []string{"x.p", "y.p", "z.p"}},
		{"source declared second, listed first", "I/sys", "i8", []string{"source: x.r", "target: z.r"}},
		{"a connection", "I/sys", "c1", []string{"source: x.p", "target: z.p"}},
		{"qualified target", "I/sys", "I::sys::i2", []string{"source: x.q", "target: y.q"}},
		{"no target: the page's interface", "I/sys/i6", "", []string{"supplier: x.p", "consumer: y.p"}},
		{"no target: an interface def, ends connect nothing", "I/D", "", []string{"supplier", "consumer"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := ad.Convert("sysml-iface-ends::"+c.target+"[]", &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, c.page).ID)})
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, m := range listItemRx.FindAllStringSubmatch(out, -1) {
				got = append(got, regexp.MustCompile(`<[^>]*>`).ReplaceAllString(m[1], ""))
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("items = %q, want %q", got, c.want)
			}
		})
	}
}

// A segment whose element has a page is a link to it, an anchor the
// document's converter writes (MC-18): <a href> in html5, its own markup in
// another converter's output.
func TestIfaceEndsLinks(t *testing.T) {
	tr, ad := setupFixture(t, "iface-ends")
	base, err := ad.BuiltinConverter("xhtml5")
	if err != nil {
		t.Fatal(err)
	}
	other := asciidoctor.ConverterFunc(func(n asciidoctor.Node, transform string, opts map[string]any) (string, error) {
		if to, ok := n.Attr(AttrLink).(string); ok && transform == "inline_anchor" {
			return "[" + to + ": " + n.(*asciidoctor.PhraseNode).Text() + "]", nil
		}
		return base.Convert(n, transform, opts)
	})
	if err := ad.RegisterConverter(other, &asciidoctor.BackendTraits{Basebackend: "html", HTMLSyntax: "xml"}, "other"); err != nil {
		t.Fatal(err)
	}
	x := pageAt(t, tr, "I/sys/x").ID
	cases := []struct{ backend, want string }{
		{"html5", `source: <a href="x/index.html">x</a>.p`},
		{"other", "source: [" + x + ": x].p"},
	}
	for _, c := range cases {
		t.Run(c.backend, func(t *testing.T) {
			out, err := ad.Convert("sysml-iface-ends::i1[]", &asciidoctor.Options{Backend: c.backend, Attributes: Attributes(pageAt(t, tr, "I/sys").ID)})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out, c.want) {
				t.Errorf("output %q does not contain %q", out, c.want)
			}
		})
	}
}

// depth shows the last segments of the connected feature's qualified path,
// joined by ::; all shows the whole path. Without it the chain is shown as
// written, joined by dots.
func TestIfaceEndsDepth(t *testing.T) {
	tr, ad := setupFixture(t, "iface-ends")
	page := Attributes(pageAt(t, tr, "I/sys").ID)
	cases := []struct {
		name, src string
		want      []string
	}{
		{"no depth: as written", "sysml-iface-ends::i1[]", []string{"source: x.p", "target: y.p"}},
		{"depth 1: the port", "sysml-iface-ends::i1[depth=1]", []string{"source: p", "target: p"}},
		{"depth as the positional attribute", "sysml-iface-ends::i1[1]", []string{"source: p", "target: p"}},
		{"depth 2: part and port", "sysml-iface-ends::i1[depth=2]", []string{"source: x::p", "target: y::p"}},
		{"depth 3: the owner of the part too", "sysml-iface-ends::i1[depth=3]", []string{"source: sys::x::p", "target: sys::y::p"}},
		{"all: the qualified path", "sysml-iface-ends::i1[depth=all]", []string{"source: I::sys::x::p", "target: I::sys::y::p"}},
		{"all in capitals", "sysml-iface-ends::i1[depth=ALL]", []string{"source: I::sys::x::p", "target: I::sys::y::p"}},
		{"deeper than the path: the whole path", "sysml-iface-ends::i1[depth=9]", []string{"source: I::sys::x::p", "target: I::sys::y::p"}},
		{"quoted name with depth", "sysml-iface-ends::i9[depth=2]", []string{"source: x::'de-icing fluid'", "target: y::'de-icing fluid'"}},
		{"quoted name as written", "sysml-iface-ends::i9[]", []string{"source: x.de-icing fluid", "target: y.de-icing fluid"}},
		{"interface def ends: names only", "sysml-iface-ends::D[depth=all]", []string{"supplier", "consumer"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := ad.Convert(c.src, &asciidoctor.Options{Attributes: page})
			if err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, m := range listItemRx.FindAllStringSubmatch(out, -1) {
				got = append(got, html.UnescapeString(regexp.MustCompile(`<[^>]*>`).ReplaceAllString(m[1], "")))
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("items = %q, want %q", got, c.want)
			}
		})
	}
}

// With depth, every segment that has a page is a link, owners included.
func TestIfaceEndsDepthLinks(t *testing.T) {
	tr, ad := setupFixture(t, "iface-ends")
	out, err := ad.Convert("sysml-iface-ends::i1[depth=all]", &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, "I/sys").ID)})
	if err != nil {
		t.Fatal(err)
	}
	if want := `source: <a href="../index.html">I</a>::<a href="index.html">sys</a>::<a href="x/index.html">x</a>::p`; !strings.Contains(out, want) {
		t.Errorf("output %q does not contain %q", out, want)
	}
}

func TestIfaceEndsErrors(t *testing.T) {
	tr, ad := setupFixture(t, "iface-ends")
	page := Attributes(pageAt(t, tr, "I/sys").ID)
	cases := []struct{ name, src, want string }{
		{"no ends: the page's part", "sysml-iface-ends::[]", "I::sys: sysml-iface-ends::[]: I::sys has no ends; name an interface or connection"},
		{"no ends: a target", "sysml-iface-ends::x[]", "I::sys: sysml-iface-ends::x[]: I::sys::x has no ends; name an interface or connection"},
		{"depth zero", "sysml-iface-ends::i1[depth=0]", `I::sys: sysml-iface-ends::i1[depth=0]: depth "0" is not a positive number or all`},
		{"depth not a number", "sysml-iface-ends::i1[depth=two]", `I::sys: sysml-iface-ends::i1[depth=two]: depth "two" is not a positive number or all`},
		{"unknown target", "sysml-iface-ends::nope[]", `I::sys: sysml-iface-ends::nope[]: the model has no element named "nope"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mustFail(t, ad, c.src, page, c.want)
		})
	}
}
