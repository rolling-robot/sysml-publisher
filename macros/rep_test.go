package macros

import (
	"strings"
	"testing"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
)

// The expectations are the // notes in testdata/rep/rep.sysml.
func TestRep(t *testing.T) {
	tr, ad := setupFixture(t, "rep")
	cases := []struct{ name, page, src, want string }{
		{"own element", "R/Engine", "sysml-rep:[python]", "power = torque * speed"},
		{"quoted language", "R/Engine", `sysml-rep:["python"]`, "power = torque * speed"},
		{"any letter case, several joined in order", "R/Engine", "sysml-rep:[LATEX]", "P = \\tau \\omega\n\\omega = 2 \\pi n"},
		{"another element, escaped", "R/Engine", "sysml-rep:Other[ocl]", "self.mass &gt; 0 and a &lt; b"},
		{"qualified name", "R/Engine", "sysml-rep:R::Other[OCL]", "self.mass &gt; 0 and a &lt; b"},
		{"short name, from a usage's page", "R/e", "sysml-rep:E[python]", "power = torque * speed"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := ad.Convert("Rep: "+c.src+".", &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, c.page).ID)})
			if err != nil {
				t.Fatal(err)
			}
			if want := "<p>Rep: " + c.want + ".</p>"; !strings.Contains(out, want) {
				t.Errorf("output %q does not contain %q", out, want)
			}
		})
	}
}

func TestRepErrors(t *testing.T) {
	tr, ad := setupFixture(t, "rep")
	cases := []struct{ name, page, src, want string }{
		{"own representations only", "R/e", "sysml-rep:[python]",
			`R::e: sysml-rep:[python]: R::e has no representation in language "python"`},
		{"no language", "R/Engine", "sysml-rep:Other[]",
			"R::Engine: sysml-rep:Other[]: no language named; write sysml-rep:[<element>][<language>]"},
		{"unknown element", "R/Engine", "sysml-rep:Nothing[ocl]",
			`R::Engine: sysml-rep:Nothing[ocl]: the model has no element named "Nothing"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mustFail(t, ad, c.src, Attributes(pageAt(t, tr, c.page).ID), c.want)
		})
	}
}
