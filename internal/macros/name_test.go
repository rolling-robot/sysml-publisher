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
	m, err := model.Folder{Dir: filepath.Join("..", "..", "testdata", "macros")}.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	tr := tree.Build(m)
	ad, err := asciidoctor.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ad.Close() })
	if err := Register(ad, tr, nil); err != nil {
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
	cases := []struct{ dir, want string }{
		// The element that owns the representation.
		{"M/pd", "Name: pd."},
		// Inherited from pd: the page's element, not the owner (MC-2).
		{"M/a_b", "Name: a&lt;b."},
		// An element without a name (PG-5).
		{"M/unnamed-1", "Name: Unnamed."},
	}
	for _, c := range cases {
		n := pageAt(t, tr, c.dir)
		out, err := ad.Convert("Name: sysml-name:[].", &asciidoctor.Options{Attributes: Attributes(n)})
		if err != nil {
			t.Fatalf("%s: %v", c.dir, err)
		}
		if !strings.Contains(out, "<p>"+c.want+"</p>") {
			t.Errorf("%s: output %q does not contain %q", c.dir, out, c.want)
		}
	}
}

func TestNameWithoutPageFails(t *testing.T) {
	_, ad := setup(t)
	if _, err := ad.Convert("sysml-name:[]", nil); err == nil {
		t.Error("converting without the page attribute should fail")
	}
}
