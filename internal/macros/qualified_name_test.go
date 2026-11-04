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
