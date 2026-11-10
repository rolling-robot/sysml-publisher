package macros

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
)

var itemRx = regexp.MustCompile(`<li>\s*<p>(.*?)</p>`)

// The expectations are the // notes in testdata/members-list/members-list.sysml.
func TestMembersList(t *testing.T) {
	tr, ad := setupFixture(t, "members-list")
	cases := []struct {
		name, page, macro string
		want              []string // list items as HTML; nil: no list
	}{
		{"all members with types", "M/pd", "sysml-members-list::[]", []string{
			"mass (Real)",
			"a*b*&#43;&#43;&#43;c",
			`<a href="sub/index.html">sub</a>`,
			"Unnamed (other)",
			"wheels (Wheel)",
			"front (Wheel)",
			"spare (Sport)",
			"Racing",
			"both (Wheel, Sport)",
			"link",
		}},
		{"typed, subclasses and subsets at any depth", "M/pd", "sysml-members-list::Wheel[]",
			[]string{"wheels", "front", "spare", "Racing", "both"}},
		{"qualified name", "M/pd", "sysml-members-list::M::Sport[]", []string{"spare", "Racing", "both"}},
		{"typed only", "M/pd", "sysml-members-list::other[]", []string{"Unnamed"}},
		{"name relative to the page's element", "M/pd", "sysml-members-list::wheels[]", []string{"front"}},
		{"nothing specializes it", "M/pd", "sysml-members-list::sub[]", nil},
		// mass : Real is typed by the library's Real.
		{"library element", "M/pd", "sysml-members-list::ScalarValues::Real[]", []string{"mass"}},
		// Short names, in every segment and every lookup.
		{"short name", "M/pd", "sysml-members-list::W[]", []string{"wheels", "front", "spare", "Racing", "both"}},
		{"qualified, all segments short", "M/pd", "sysml-members-list::MM::W[]", []string{"wheels", "front", "spare", "Racing", "both"}},
		{"relative short name", "M/pd", "sysml-members-list::ws[]", []string{"front"}},
		{"relative qualified, short last segment", "M/pd", "sysml-members-list::pd::ws[]", []string{"front"}},
		// Element kinds: exact metaclass, any letter case.
		{"kind", "M/pd", "sysml-members-list::partUsage[]", []string{
			`<a href="sub/index.html">sub</a>`, "Unnamed", "wheels", "front", "spare", "both"}},
		{"kind, interface", "M/pd", "sysml-members-list::InterfaceUsage[]", []string{"link"}},
		{"kind, lower case", "M/pd", "sysml-members-list::attributeusage[]", []string{"mass"}},
		{"kind without members", "M/pd", "sysml-members-list::ActionUsage[]", nil},
		{"the page's element's members only", "M/leaf", "sysml-members-list::[]", []string{"q"}},
		{"no members", "M/pd/sub", "sysml-members-list::[]", nil},
		{"relative name inside the page's element", "M/Car", "sysml-members-list::front[]", nil},
		{"redefinition of a subset, by plain name", "M/Car", "sysml-members-list::wheels[]", []string{"front"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := ad.Convert(c.macro+"\n\nAfter.", &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, c.page))})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out, "<p>After.</p>") {
				t.Errorf("the text after the macro is lost: %q", out)
			}
			var got []string
			for _, m := range itemRx.FindAllStringSubmatch(out, -1) {
				got = append(got, m[1])
			}
			if !slices.Equal(got, c.want) {
				t.Errorf("items:\n got %q\nwant %q", got, c.want)
			}
			hasList := strings.Contains(out, `<div class="ulist sysml-members-list">`)
			if hasList != (c.want != nil) {
				t.Errorf("list with role sysml-members-list present = %v, want %v: %q", hasList, c.want != nil, out)
			}
		})
	}
}

func TestMembersListErrors(t *testing.T) {
	tr, ad := setupFixture(t, "members-list")
	page := Attributes(pageAt(t, tr, "M/pd"))
	cases := []struct{ name, macro, want string }{
		{"unknown element", "sysml-members-list::Nothing[]",
			`M::pd: sysml-members-list::Nothing[]: the model has no element named "Nothing", and it is no element kind (such as partUsage)`},
		{"ambiguous plain name", "sysml-members-list::x[]",
			`M::pd: sysml-members-list::x[]: "x" names several elements (M::A::x, M::B::x); use a qualified name`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mustFail(t, ad, c.macro, page, c.want)
		})
	}
}
