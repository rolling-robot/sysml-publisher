package macros

import (
	"strings"
	"testing"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
)

func TestShortName(t *testing.T) {
	tr, ad := setup(t)
	cases := []struct{ dir, want string }{
		// part def <P> pd
		{"M/pd", "SN: (P)."},
		// Inherited from pd: the page's element (MC-2), which has no short
		// name, so the macro gives nothing.
		{"M/a_b", "SN: ()."},
		{"M/unnamed-1", "SN: ()."},
	}
	for _, c := range cases {
		n := pageAt(t, tr, c.dir)
		out, err := ad.Convert("SN: (sysml-short-name:[]).", &asciidoctor.Options{Attributes: Attributes(n)})
		if err != nil {
			t.Fatalf("%s: %v", c.dir, err)
		}
		if !strings.Contains(out, "<p>"+c.want+"</p>") {
			t.Errorf("%s: output %q does not contain %q", c.dir, out, c.want)
		}
	}
}
