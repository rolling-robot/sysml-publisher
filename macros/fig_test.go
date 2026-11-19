// Copyright 2026 Petr Mukhachev
// SPDX-License-Identifier: MIT

package macros

import (
	"html"
	"regexp"
	"strings"
	"testing"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
)

// mermaidRx finds the pre a figure is written in: the text spans lines.
var mermaidRx = regexp.MustCompile(`(?s)<pre class="mermaid">(.*?)</pre>`)

// mermaidOf is the Mermaid source of the first figure in converted HTML:
// the pre's text, unescaped.
func mermaidOf(out string) string {
	m := mermaidRx.FindStringSubmatch(out)
	if m == nil {
		return ""
	}
	return html.UnescapeString(m[1])
}

// figSource is the whole Mermaid source a figure of iface with body is
// expected to be: the style frontmatter and the flowchart, no title.
func figSource(iface, body string) string {
	return "---\n" + mermaidConfig +
		"---\n%% " + iface + " — interconnection figure\n" +
		"flowchart LR\n" + body
}

// The figures of the // notes in testdata/figs, on the page of vehicle: the
// parts on the ends and between them and their common ancestor, the end
// features, and one edge per pair of ends; nothing else of the model.
func TestFig(t *testing.T) {
	tr, ad := setupFixture(t, "figs")
	attrs := Attributes(pageAt(t, tr, "F/vehicle").ID)
	cases := []struct{ name, target, iface, body string }{
		{"the parts on the ends", "i1", "F::vehicle::i1",
			"  subgraph n0 [\"`*«part»*\n**vehicle**`\"]\n    direction LR\n    subgraph n1 [\"`*«part»*\n**a**`\"]\n      direction LR\n      n2(\"`*«port»*\n**p**`\")\n    end\n    subgraph n3 [\"`*«part»*\n**b**`\"]\n      direction LR\n      n4(\"`*«port»*\n**q**`\")\n    end\n  end\n  n2 === n4\n  linkStyle 0 stroke-width:3px\nclassDef figbox fill:#FFFFFF,stroke:#181818,color:#000000\nclass n0,n1,n3 figbox\nclassDef figport fill:#FFFFFF,stroke:#181818,color:#000000\nclass n2,n4 figport\n",
		},
		{"an end nested in a part", "i2", "F::vehicle::i2",
			"  subgraph n0 [\"`*«part»*\n**vehicle**`\"]\n    direction LR\n    subgraph n1 [\"`*«part»*\n**a**`\"]\n      direction LR\n      subgraph n2 [\"`*«part»*\n**sub**`\"]\n        direction LR\n        n3(\"`*«port»*\n**deep**`\")\n      end\n    end\n    subgraph n4 [\"`*«part»*\n**b**`\"]\n      direction LR\n      n5(\"`*«port»*\n**q**`\")\n    end\n  end\n  n3 === n5\n  linkStyle 0 stroke-width:3px\nclassDef figbox fill:#FFFFFF,stroke:#181818,color:#000000\nclass n0,n1,n2,n4 figbox\nclassDef figport fill:#FFFFFF,stroke:#181818,color:#000000\nclass n3,n5 figport\n",
		},
		{"ends in different packages", "ixy", "F::ixy",
			"  subgraph n0 [\"`*«package»*\n**F**`\"]\n    direction LR\n    subgraph n1 [\"`*«part»*\n**px**`\"]\n      direction LR\n      n2(\"`*«port»*\n**p**`\")\n    end\n    subgraph n3 [\"`*«part»*\n**py**`\"]\n      direction LR\n      n4(\"`*«port»*\n**p**`\")\n    end\n  end\n  n2 === n4\n  linkStyle 0 stroke-width:3px\nclassDef figbox fill:#FFFFFF,stroke:#181818,color:#000000\nclass n0,n1,n3 figbox\nclassDef figport fill:#FFFFFF,stroke:#181818,color:#000000\nclass n2,n4 figport\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := ad.Convert("sysml-fig::"+c.target+"[]", &asciidoctor.Options{Attributes: attrs})
			if err != nil {
				t.Fatal(err)
			}
			want := figSource(c.iface, c.body)
			if got := mermaidOf(out); got != want {
				t.Errorf("figure\n got %q\nwant %q", got, want)
			}
		})
	}
}

// A resolvable target that is not an interface, and no target on a page
// whose element is not one, leave the macro text where it was, no warning
// (MC-25).
func TestFigEchoes(t *testing.T) {
	tr, ad := setupFixture(t, "figs")
	attrs := Attributes(pageAt(t, tr, "F/vehicle").ID)
	cases := []struct{ name, src, want string }{
		{"a part", "sysml-fig::a[]", `<p class="sysml-fig-text">sysml-fig::a[]</p>`},
		{"an interface def", "sysml-fig::Itself[]", `<p class="sysml-fig-text">sysml-fig::Itself[]</p>`},
		{"no target: the page's element", "sysml-fig::[]", `<p class="sysml-fig-text">sysml-fig::[]</p>`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			*recorded[ad] = nil
			out, err := ad.Convert(c.src, &asciidoctor.Options{Attributes: attrs})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out, c.want) {
				t.Errorf("output lacks %q:\n%s", c.want, out)
			}
			if ws := *recorded[ad]; len(ws) != 0 {
				t.Errorf("warnings %q, want none", ws)
			}
		})
	}
}

// A target that names no element fails the macro (MC-19).
func TestFigUnknownTargetFails(t *testing.T) {
	tr, ad := setupFixture(t, "figs")
	attrs := Attributes(pageAt(t, tr, "F/vehicle").ID)
	mustFail(t, ad, "sysml-fig::nope[]", attrs,
		`F::vehicle: sysml-fig::nope[]: the model has no element named "nope"`)
}

// A figure is a pass block the document's converter writes (MC-25): html5
// writes its pre verbatim; another converter tells the block by AttrFig and
// takes the Mermaid source from MermaidText.
func TestFigWrittenByTheConverter(t *testing.T) {
	tr, ad := setupFixture(t, "figs")
	base, err := ad.BuiltinConverter("xhtml5")
	if err != nil {
		t.Fatal(err)
	}
	wiki := asciidoctor.ConverterFunc(func(n asciidoctor.Node, transform string, opts map[string]any) (string, error) {
		if qn, ok := n.Attr(AttrFig).(string); ok {
			text, _ := MermaidText(n)
			return `<fig of="` + qn + `" as="` + transform + `" text="` + html.EscapeString(text) + `"/>`, nil
		}
		return base.Convert(n, transform, opts)
	})
	if err := ad.RegisterConverter(wiki, &asciidoctor.BackendTraits{Basebackend: "html", HTMLSyntax: "xml"}, "wiki"); err != nil {
		t.Fatal(err)
	}
	attrs := Attributes(pageAt(t, tr, "F/vehicle").ID)
	for _, out := range []struct{ backend, want string }{
		{"html5", `<div class="sysml-fig"><pre class="mermaid">`},
		{"wiki", `<fig of="F::vehicle::i1" as="pass" text="`},
	} {
		got, err := ad.Convert("sysml-fig::i1[]", &asciidoctor.Options{Backend: out.backend, Attributes: attrs})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(got, out.want) {
			t.Errorf("%s: %q does not contain %q", out.backend, got, out.want)
		}
	}
}

// commonAncestor is the deepest element every chain passes through.
func TestCommonAncestor(t *testing.T) {
	cases := []struct {
		name   string
		chains [][]string
		want   string
		ok     bool
	}{
		{"a shared start", [][]string{{"m", "v", "a"}, {"m", "v", "b"}}, "v", true},
		{"a chain inside another", [][]string{{"m", "v", "a"}, {"m", "v", "a", "s"}}, "a", true},
		{"the chains are the same", [][]string{{"m", "a"}, {"m", "a"}}, "a", true},
		{"nothing shared", [][]string{{"x", "a"}, {"y", "b"}}, "", false},
		{"diverging at the start", [][]string{{"x", "a"}, {"y", "b"}, {"x", "c"}}, "", false},
		{"one end only", [][]string{{"m", "a"}}, "a", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, ok := commonAncestor(c.chains)
			if got != c.want || ok != c.ok {
				t.Errorf("commonAncestor(%q) = %q, %v; want %q, %v", c.chains, got, ok, c.want, c.ok)
			}
		})
	}
}

// A label carries the keyword and the name; a name with a backtick or a
// double quote falls back to a double-quoted string, HTML-escaped.
func TestFigLabel(t *testing.T) {
	cases := []struct {
		name    string
		keyword string
		label   string
		want    string
	}{
		{"an unnamed port", "«port»", "", "\"`*«port»*`\""},
		{"a named port", "«port»", "p", "\"`*«port»*\n**p**`\""},
		{"a name with a backtick", "«part»", "a`b", "\"*«part»*<br/>**a`b**\""},
		{"a name with a double quote", "«part»", `a"b`, `"*«part»*<br/>**a&quot;b**"`},
		{"a name with a less-than", "«port»", "a<b", `"*«port»*<br/>**a&lt;b**"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := mermaidString(figLabel(c.keyword, c.label)); got != c.want {
				t.Errorf("figLabel = %q, want %q", got, c.want)
			}
		})
	}
}
