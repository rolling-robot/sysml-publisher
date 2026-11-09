package macros

import (
	"strings"
	"testing"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
)

func TestQualifiedName(t *testing.T) {
	tr, ad := setup(t)
	cases := []struct{ dir, want string }{
		{"M/pd", "QN: M::pd."},
		// Inherited from pd: the page's element (MC-2); the quoted name
		// keeps its quotes and is escaped.
		{"M/a_b", "QN: M::&#39;a&lt;b&#39;."},
		// An element without a name has no qualified name; its positional
		// name is given instead.
		{"M/unnamed-1", "QN: M::@2."},
	}
	for _, c := range cases {
		n := pageAt(t, tr, c.dir)
		out, err := ad.Convert("QN: sysml-qualified-name:[].", &asciidoctor.Options{Attributes: Attributes(n)})
		if err != nil {
			t.Fatalf("%s: %v", c.dir, err)
		}
		if !strings.Contains(out, "<p>"+c.want+"</p>") {
			t.Errorf("%s: output %q does not contain %q", c.dir, out, c.want)
		}
	}
}

// With a target, the element it names, read as sysml-name reads it (MC-15).
func TestQualifiedNameTarget(t *testing.T) {
	tr, ad := setup(t)
	cases := []struct{ name, page, target, want string }{
		{"quoted name, escaped", "M/pd", "'a<b'", "M::&#39;a&lt;b&#39;"},
		{"short name", "M/a_b", "P", "M::pd"},
		{"element without a page", "M/pd", "Other::'top speed'", "M::Other::&#39;top speed&#39;"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := ad.Convert("QN: sysml-qualified-name:"+c.target+"[].", &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, c.page))})
			if err != nil {
				t.Fatal(err)
			}
			if want := "<p>QN: " + c.want + ".</p>"; !strings.Contains(out, want) {
				t.Errorf("output %q does not contain %q", out, want)
			}
		})
	}
	_, err := ad.Convert("sysml-qualified-name:Nothing[]", &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, "M/pd"))})
	if want := `sysml-qualified-name:Nothing[] on the page of M::pd: the model has no element named "Nothing"`; err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("err = %v, want one containing %q", err, want)
	}
}
