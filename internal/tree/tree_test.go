package tree

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rolling-robot/sysml-publisher/internal/model"
)

func build(t *testing.T, fixture string) *Tree {
	t.Helper()
	m, err := model.Folder{Dir: filepath.Join("..", "..", "testdata", fixture)}.Load(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	tr, err := Build(t.Context(), m)
	if err != nil {
		t.Fatal(err)
	}
	return tr
}

func dirs(tr *Tree) []string {
	var out []string
	for _, n := range tr.Pages {
		out = append(out, n.Dir)
	}
	return out
}

func page(t *testing.T, tr *Tree, dir string) *Node {
	t.Helper()
	for _, n := range tr.Pages {
		if n.Dir == dir {
			return n
		}
	}
	t.Fatalf("no page at %s; pages: %v", dir, dirs(tr))
	return nil
}

// bodies are the trimmed bodies of reps, each prefixed by its owner's name.
func bodies(tr *Tree, reps []Rep) []string {
	var out []string
	for _, r := range reps {
		s := strings.TrimSpace(r.Body)
		if n := tr.ByID[r.Owner]; n != nil {
			s = n.Name + ": " + s
		}
		out = append(out, s)
	}
	return out
}

func TestSmokeModel(t *testing.T) {
	tr := build(t, "smoke")
	want := []string{"Model", "Model/pd", "Model/aaaa", "Model/bbb"}
	if got := dirs(tr); !reflect.DeepEqual(got, want) {
		t.Fatalf("pages = %v, want %v", got, want)
	}
	if n := page(t, tr, "Model"); n.Documented() {
		t.Error("package Model should be an empty ancestor page (PG-6)")
	}
	// pd's one representation, owned by pd.
	isPdRep := func(reps []Rep) bool {
		return len(reps) == 1 && reps[0].Owner == "Model__pd" && strings.Contains(reps[0].Body, "Textual representation")
	}
	if n := page(t, tr, "Model/pd"); !isPdRep(n.Own) {
		t.Errorf("pd own = %v", n.Own)
	}
	for _, dir := range []string{"Model/aaaa", "Model/bbb"} {
		n := page(t, tr, dir)
		if len(n.Own) != 0 {
			t.Errorf("%s own = %v, want none", dir, n.Own)
		}
		if !isPdRep(n.Inherited) {
			t.Errorf("%s inherited = %v", dir, n.Inherited)
		}
	}
	if len(tr.Warnings) != 0 {
		t.Errorf("warnings = %v, want none", tr.Warnings)
	}
}

func TestInheritance(t *testing.T) {
	tr := build(t, "inherit")
	cases := []struct {
		dir            string
		own, inherited []string
	}{
		{"Inherit/pd", []string{"pd: pd text"}, nil},
		// A definition inherits from its supertypes.
		{"Inherit/pd2", []string{"pd2: pd2 text"}, []string{"pd: pd text"}},
		// A usage inherits through its type, then the type's supertypes.
		{"Inherit/x", []string{"x: x text"}, []string{"pd2: pd2 text", "pd: pd text"}},
		// Subsetting: y :> x inherits x's own and x's inherited, nearest first.
		{"Inherit/y", nil, []string{"x: x text", "pd2: pd2 text", "pd: pd text"}},
		// A diamond D :> B, C with both :> A shows A once.
		{"Inherit/D", nil, []string{"A: A text"}},
	}
	for _, c := range cases {
		n := page(t, tr, c.dir)
		if got := bodies(tr, n.Own); !reflect.DeepEqual(got, c.own) {
			t.Errorf("%s own = %v, want %v", c.dir, got, c.own)
		}
		if got := bodies(tr, n.Inherited); !reflect.DeepEqual(got, c.inherited) {
			t.Errorf("%s inherited = %v, want %v", c.dir, got, c.inherited)
		}
	}
}

func TestOtherLanguagesAreSkipped(t *testing.T) {
	tr := build(t, "otherlang")
	if got, want := dirs(tr), []string{"Other", "Other/up"}; !reflect.DeepEqual(got, want) {
		t.Errorf("pages = %v, want %v", got, want)
	}
	if len(tr.Warnings) != 1 || !strings.Contains(tr.Warnings[0], "Other::md") || !strings.Contains(tr.Warnings[0], `"markdown"`) {
		t.Errorf("warnings = %v, want one naming Other::md and markdown", tr.Warnings)
	}
}

func TestUnnamedElements(t *testing.T) {
	tr := build(t, "unnamed")
	if got, want := dirs(tr), []string{"U", "U/pd", "U/unnamed-1", "U/unnamed-2"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("pages = %v, want %v", got, want)
	}
	for _, dir := range []string{"U/unnamed-1", "U/unnamed-2"} {
		if n := page(t, tr, dir); n.Name != UnnamedName || !n.Unnamed {
			t.Errorf("%s name = %q unnamed = %v", dir, n.Name, n.Unnamed)
		}
	}
	if len(tr.Warnings) != 2 {
		t.Errorf("warnings = %v, want one per unnamed page", tr.Warnings)
	}
}

func TestNestedAndPruned(t *testing.T) {
	tr := build(t, "nested")
	if got, want := dirs(tr), []string{"Outer", "Outer/Inner", "Outer/Inner/q"}; !reflect.DeepEqual(got, want) {
		t.Errorf("pages = %v, want %v (Empty pruned, PT-4)", got, want)
	}
}

func TestCrossFile(t *testing.T) {
	tr := build(t, "crossfile")
	if got, want := dirs(tr), []string{"Lib", "Lib/pd", "Top", "Top/x"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("pages = %v, want %v", got, want)
	}
	if got := bodies(tr, page(t, tr, "Top/x").Inherited); !reflect.DeepEqual(got, []string{"pd: From lib"}) {
		t.Errorf("Top/x inherited = %v", got)
	}
}

func TestSegment(t *testing.T) {
	for in, want := range map[string]string{
		"pd":      "pd",
		"my part": "my part",
		"a/b":     "a_b",
		"..":      "_..",
		`x:y*"z"`: "x_y__z_",
		"":        "_",
	} {
		if got := segment(in); got != want {
			t.Errorf("segment(%q) = %q, want %q", in, got, want)
		}
	}
	used := map[string]bool{}
	if a, b := unique("p", used), unique("P", used); a != "p" || b != "P-2" {
		t.Errorf("unique = %q, %q; want p, P-2 (case-insensitive filesystems)", a, b)
	}
}

// An unnamed element's page carries its positional name (PG-5).
func TestUnnamedNode(t *testing.T) {
	n := page(t, build(t, "macros"), "M/unnamed-1")
	if n.QualifiedName != "M::@2" || n.Name != UnnamedName || !n.Unnamed {
		t.Errorf("name %q, qualified %q, unnamed %v; want Unnamed, M::@2, true", n.Name, n.QualifiedName, n.Unnamed)
	}
}
