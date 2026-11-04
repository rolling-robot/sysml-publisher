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
		out, err := ad.Convert(src, &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, dir))})
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
