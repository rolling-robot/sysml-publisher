package macros

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/model"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

// setup builds the page tree of the macros fixture and an asciidoctorg
// instance with the macros registered.
func setup(t *testing.T) (*tree.Tree, *asciidoctor.Asciidoctor) {
	t.Helper()
	m, err := model.Folder{Dir: filepath.Join("..", "testdata", "macros")}.Load(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	tr, err := tree.Build(t.Context(), m)
	if err != nil {
		t.Fatal(err)
	}
	ad, err := asciidoctor.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ad.Close() })
	if _, err := Register(context.Background(), ad, tr.Model.Model, tr, record(t, ad)); err != nil {
		t.Fatal(err)
	}
	return tr, ad
}

func pageAt(t *testing.T, tr *tree.Tree, dir string) *tree.Node {
	t.Helper()
	for _, n := range tr.Pages {
		if n.Dir == dir {
			return n
		}
	}
	t.Fatalf("no page at %s", dir)
	return nil
}

func TestName(t *testing.T) {
	tr, ad := setup(t)
	cases := []struct{ name, page, src, want string }{
		// No target: the page's element.
		{"own element", "M/pd", "sysml-name:[]", "pd"},
		{"inherited representation: the page's element (MC-2)", "M/a_b", "sysml-name:[]", "a&lt;b"},
		{"element without a name (PG-5)", "M/unnamed-1", "sysml-name:[]", "Unnamed"},
		// A target (MC-15), looked up from the page of a<b, linked to its page
		// (MC-18).
		{"qualified name", "M/a_b", "sysml-name:M::pd[]", `<a href="../pd/index.html">pd</a>`},
		{"relative name", "M/a_b", "sysml-name:pd[]", `<a href="../pd/index.html">pd</a>`},
		{"short name", "M/a_b", "sysml-name:P[]", `<a href="../pd/index.html">pd</a>`},
		{"qualified, short last segment", "M/a_b", "sysml-name:M::P[]", `<a href="../pd/index.html">pd</a>`},
		{"quoted segment, escaped", "M/pd", "sysml-name:M::'a<b'[]", `<a href="../a_b/index.html">a&lt;b</a>`},
		// A link only to another page (MC-18).
		{"the page's own element, named", "M/pd", "sysml-name:M::pd[]", "pd"},
		{"element without a page, quoted name with a space", "M/pd", "sysml-name:M::Other::'top speed'[]", "top speed"},
		{"relative, quoted name with a space", "M/pd", "sysml-name:Other::'top speed'[]", "top speed"},
		{"escaped macro stays text", "M/pd", `\sysml-name:pd[]`, "sysml-name:pd[]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := ad.Convert("Name: "+c.src+".", &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, c.page).ID)})
			if err != nil {
				t.Fatal(err)
			}
			if want := "<p>Name: " + c.want + ".</p>"; !strings.Contains(out, want) {
				t.Errorf("output %q does not contain %q", out, want)
			}
		})
	}
}

func TestNameErrors(t *testing.T) {
	tr, ad := setup(t)
	page := Attributes(pageAt(t, tr, "M/pd").ID)
	cases := []struct{ name, src, want string }{
		{"unknown element", "sysml-name:Nothing[]",
			`M::pd: sysml-name:Nothing[]: the model has no element named "Nothing"`},
		{"ambiguous plain name", "sysml-name:x[]",
			`M::pd: sysml-name:x[]: "x" names several elements (M::A::x, M::B::x); use a qualified name`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mustFail(t, ad, c.src, page, c.want)
		})
	}
}

func TestNameWithoutPageFails(t *testing.T) {
	_, ad := setup(t)
	if _, err := ad.Convert("sysml-name:[]", nil); err == nil {
		t.Error("converting without the page attribute should fail")
	}
}

// sysml-name:[<element>::]@<metadata>[::<attribute>][]: the names of
// metadata applied to an element and of its attributes (MC-8). The cases
// use testdata/metadata-table.
func TestNameMetadata(t *testing.T) {
	tr, ad := setupFixture(t, "metadata-table")
	cases := []struct{ name, page, target, want string }{
		{"metadata on the page's element", "Meta/bound", "@Info", "Info"},
		{"its attribute", "Meta/bound", "@Info::owner", "owner"},
		{"short name of the metadata def", "Meta/specialized", "@drv", "Derived"},
		{"attribute of a supertype of the metadata def", "Meta/specialized", "@drv::origin", "origin"},
		{"metadata on another element, qualified", "Meta/specialized", "Meta::bound::@Info::count", "count"},
		{"metadata on another element, relative", "Meta/specialized", "bound::@Info::count", "count"},
		{"prefix form #Info", "Meta/defaults", "@Info::sum", "sum"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := ad.Convert("N: sysml-name:"+c.target+"[].", &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, c.page).ID)})
			if err != nil {
				t.Fatal(err)
			}
			if want := "<p>N: " + c.want + ".</p>"; !strings.Contains(out, want) {
				t.Errorf("output %q does not contain %q", out, want)
			}
		})
	}
}

func TestNameMetadataErrors(t *testing.T) {
	tr, ad := setupFixture(t, "metadata-table")
	page := Attributes(pageAt(t, tr, "Meta/bound").ID)
	cases := []struct{ name, target, want string }{
		{"metadata the element does not carry", "@Derived", "Meta::bound has no @Derived metadata"},
		{"unknown attribute", "@Info::nothing", `metadata def Info has no attribute "nothing"`},
		{"unknown metadata def", "@Missing", `the model has no metadata def "Missing"`},
		{"too many segments", "@Info::owner::x", "write [<element>::]@<metadata>[::<attribute>]"},
		{"no metadata name", "@::owner", "write [<element>::]@<metadata>[::<attribute>]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mustFail(t, ad, "sysml-name:"+c.target+"[]", page, "Meta::bound: sysml-name:"+c.target+"[]: "+c.want)
		})
	}
}
