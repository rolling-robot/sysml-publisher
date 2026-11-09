package model

import (
	"errors"
	"slices"
	"testing"

	"github.com/Open-MBEE/OpenSysML/client/opensysml"
)

func loadOpen(t *testing.T, dir string) *Model {
	t.Helper()
	m, err := load(t, dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	return m
}

func ids(refs []Ref) []string {
	out := make([]string, len(refs))
	for i, r := range refs {
		out[i] = r.ID
	}
	return out
}

// Props reads an element by qualified name, form id or positional id, and
// answers multi-valued properties in declaration order.
func TestProps(t *testing.T) {
	m := loadOpen(t, "unnamed")
	cases := []struct {
		name, id      string
		wantID        string
		wantQN        string
		wantMetaclass string
	}{
		{"qualified name", "U::pd", "U__pd", "U::pd", "PartDefinition"},
		{"form id", "U__pd", "U__pd", "U::pd", "PartDefinition"},
		{"positional id of an unnamed element", "U::@1", "U___401", "", "PartUsage"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p, err := m.Props(t.Context(), c.id, "owner")
			if err != nil {
				t.Fatal(err)
			}
			want := Ref{ID: c.wantID, QualifiedName: c.wantQN, Metaclass: c.wantMetaclass}
			if p.Element != want {
				t.Errorf("element = %+v, want %+v", p.Element, want)
			}
			if got := p.Ref("owner"); got.ID != "U" || got.Metaclass != "Package" {
				t.Errorf("owner = %+v, want the package U", got)
			}
		})
	}
	t.Run("members in declaration order", func(t *testing.T) {
		p, err := m.Props(t.Context(), "U", "ownedMember")
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"U__pd", "U___401", "U__plain", "U___403"}
		if got := ids(p.Refs("ownedMember")); !slices.Equal(got, want) {
			t.Errorf("ownedMember = %v, want %v", got, want)
		}
	})
	t.Run("a property the metaclass does not declare is empty", func(t *testing.T) {
		p, err := m.Props(t.Context(), "U", "type", "declaredName")
		if err != nil {
			t.Fatal(err)
		}
		if got := p.Refs("type"); got != nil {
			t.Errorf("type = %v, want none", got)
		}
		if got := p.Str("declaredName"); got != "U" {
			t.Errorf("declaredName = %q, want U", got)
		}
	})
	t.Run("unknown element", func(t *testing.T) {
		_, err := m.Props(t.Context(), "U::nothing", "owner")
		if !errors.Is(err, ErrNoSuchElement) {
			t.Errorf("err = %v, want ErrNoSuchElement", err)
		}
	})
}

// Props of the same element by another name reads from the cache: the
// answer is the one entry, keyed by the form id.
func TestPropsCache(t *testing.T) {
	m := loadOpen(t, "unnamed")
	if _, err := m.Props(t.Context(), "U::pd", "declaredName"); err != nil {
		t.Fatal(err)
	}
	if m.cache["U::pd"] != m.cache["U__pd"] || m.cache["U__pd"] == nil {
		t.Fatal("U::pd and U__pd are not one cache entry")
	}
	if _, err := m.Props(t.Context(), "U__pd", "declaredName"); err != nil {
		t.Fatal(err)
	}
	if _, ok := m.cache["U__pd"].values["declaredName"]; !ok {
		t.Error("declaredName not cached")
	}
}

// A usage typed by a library type refers to it with its library qualified
// name; across files, a usage's type is the other file's definition.
func TestPropsReferences(t *testing.T) {
	m := loadOpen(t, "crossfile")
	p, err := m.Props(t.Context(), "Top::x", "type")
	if err != nil {
		t.Fatal(err)
	}
	if got := p.Refs("type"); len(got) != 1 || got[0].ID != "Lib__pd" || got[0].Name() != "pd" {
		t.Errorf("Top::x type = %+v, want Lib::pd", got)
	}
	els, err := m.Query(t.Context(), opensysml.Query{Where: opensysml.InstanceOf("TextualRepresentation"), Select: []string{"owner", "language"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(els) != 1 || els[0].Properties["owner"] != "Lib::pd" || els[0].Properties["language"] != "asciidoc" {
		t.Fatalf("representations = %+v, want one owned by Lib::pd", els)
	}
	rep, err := m.Props(t.Context(), els[0].ID, "body")
	if err != nil {
		t.Fatal(err)
	}
	if got := rep.Str("body"); got != "From lib " {
		t.Errorf("body = %q, want %q", got, "From lib ")
	}
}

// Contains tells the model's own elements from the library's: a library
// type is referred to, but not read (AGENTS.md §13).
func TestContainsOnlyTheModel(t *testing.T) {
	m := loadOpen(t, "members-list")
	types, err := m.Types(t.Context(), "M::pd::mass")
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 1 || types[0].QualifiedName != "ScalarValues::Real" {
		t.Fatalf("mass types = %+v, want ScalarValues::Real", types)
	}
	if m.Contains(types[0].ID) {
		t.Error("Contains(ScalarValues::Real) = true, want false")
	}
	if !m.Contains("M__pd__mass") {
		t.Error("Contains(M::pd::mass) = false, want true")
	}
	if g, err := m.Generals(t.Context(), types[0].ID); err != nil || g != nil {
		t.Errorf("Generals(Real) = %v, %v; want none, unread", g, err)
	}
}

func TestRefName(t *testing.T) {
	cases := []struct{ qn, want string }{
		{"ScalarValues::Real", "Real"},
		{"M::'top speed'", "top speed"},
		{"M::'a::b'", "a::b"},
		{`M::'it\'s'`, "it's"},
		{"", ""},
		{"Top", "Top"},
	}
	for _, c := range cases {
		t.Run(c.qn, func(t *testing.T) {
			if got := (Ref{QualifiedName: c.qn}).Name(); got != c.want {
				t.Errorf("Name() = %q, want %q", got, c.want)
			}
		})
	}
}
