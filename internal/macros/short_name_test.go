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

// With a target, the element it names, read as sysml-name reads it (MC-15).
func TestShortNameTarget(t *testing.T) {
	tr, ad := setup(t)
	cases := []struct{ name, page, target, want string }{
		{"another element's short name", "M/a_b", "pd", "SN: (P)."},
		{"by its own short name", "M/a_b", "P", "SN: (P)."},
		{"an element without a short name", "M/pd", "'a<b'", "SN: ()."},
		{"an element without a page", "M/pd", "Other", "SN: ()."},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := ad.Convert("SN: (sysml-short-name:"+c.target+"[]).", &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, c.page))})
			if err != nil {
				t.Fatal(err)
			}
			if want := "<p>" + c.want + "</p>"; !strings.Contains(out, want) {
				t.Errorf("output %q does not contain %q", out, want)
			}
		})
	}
}
