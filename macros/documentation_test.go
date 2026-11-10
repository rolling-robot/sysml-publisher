package macros

import (
	"strings"
	"testing"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
)

func TestDocumentation(t *testing.T) {
	tr, ad := setup(t)
	src := "Before.\n\nsysml-doc::[]\n\nAfter."
	convert := func(dir string) string {
		t.Helper()
		out, err := ad.Convert(src, &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, dir).ID)})
		if err != nil {
			t.Fatalf("%s: %v", dir, err)
		}
		return out
	}

	out := convert("M/pd")
	for _, want := range []string{`<div class="openblock sysml-doc">`, "<p>pd documentation.</p>", "<p>Before.</p>", "<p>After.</p>"} {
		if !strings.Contains(out, want) {
			t.Errorf("M/pd: output %q does not contain %q", out, want)
		}
	}

	// The page's element's own docs (MC-2), not pd's; parsed as AsciiDoc,
	// escaped, in declaration order.
	out = convert("M/a_b")
	first := strings.Index(out, "The <strong>a&lt;b</strong> part.")
	second := strings.Index(out, "Second doc.")
	if first < 0 || second < first {
		t.Errorf("M/a_b: want both docs in order, got %q", out)
	}
	if strings.Contains(out, "pd documentation.") {
		t.Error("M/a_b: shows pd's doc, which belongs to the representation's owner")
	}

	// No doc: the macro line is dropped, the rest stays.
	out = convert("M/unnamed-1")
	if strings.Contains(out, "openblock") || strings.Contains(out, "sysml-doc") || !strings.Contains(out, "<p>After.</p>") {
		t.Errorf("M/unnamed-1: want only the surrounding paragraphs, got %q", out)
	}
}

// With a target, the docs of the element it names, read as sysml-name reads
// it (MC-15), whether it has a page or not.
func TestDocumentationTarget(t *testing.T) {
	tr, ad := setup(t)
	page := Attributes(pageAt(t, tr, "M/pd").ID)
	cases := []struct {
		name, target string
		want         []string // paragraphs in the doc block; nil: no block
	}{
		{"element with a page", "'a<b'", []string{"The <strong>a&lt;b</strong> part.", "Second doc."}},
		{"element without a page", "Other", []string{"Other&#8217;s documentation."}},
		{"element without docs", "A", nil},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := ad.Convert("sysml-doc::"+c.target+"[]\n\nAfter.", &asciidoctor.Options{Attributes: page})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(out, "pd documentation.") {
				t.Errorf("shows the page's own doc: %q", out)
			}
			if has := strings.Contains(out, `<div class="openblock sysml-doc">`); has != (c.want != nil) {
				t.Errorf("doc block present = %v, want %v: %q", has, c.want != nil, out)
			}
			for _, w := range c.want {
				if !strings.Contains(out, "<p>"+w+"</p>") {
					t.Errorf("output %q lacks paragraph %q", out, w)
				}
			}
		})
	}
	mustFail(t, ad, "sysml-doc::Nothing[]", page, `M::pd: sysml-doc::Nothing[]: the model has no element named "Nothing"`)
}
