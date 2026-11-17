// Copyright 2026 Petr Mukhachev
// SPDX-License-Identifier: MIT

package macros

import (
	"regexp"
	"slices"
	"strings"
	"testing"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
)

// items is a converted macro's list as plain text, tags stripped.
func items(out string) []string {
	var got []string
	for _, m := range listItemRx.FindAllStringSubmatch(out, -1) {
		got = append(got, regexp.MustCompile(`<[^>]*>`).ReplaceAllString(m[1], ""))
	}
	return got
}

// The expected lists are the // notes in testdata/ifaces. A connection is
// one item, then -> and the distinct part(s) on the other ends. The items
// come in model declaration order.
func TestIfaces(t *testing.T) {
	tr, ad := setupFixture(t, "ifaces")
	cases := []struct {
		name, page, target string
		want               []string
	}{
		{"no target: the page's part, x", "I/x", "", []string{
			"i1 -> wy", "i3 -> wy, z", "c1 -> wy", "s1 -> ", "i2 -> z"}},
		{"explicit target x", "I/x", "x", []string{
			"i1 -> wy", "i3 -> wy, z", "c1 -> wy", "s1 -> ", "i2 -> z"}},
		{"qualified target I::x", "I/x", "I::x", []string{
			"i1 -> wy", "i3 -> wy, z", "c1 -> wy", "s1 -> ", "i2 -> z"}},
		{"from y's page: the parts on the other ends", "I/y", "", []string{
			"i1 -> ex", "i3 -> ex, z", "c1 -> ex", "x4 -> z"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := ad.Convert("sysml-ifaces::"+c.target+"[]", &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, c.page).ID)})
			if err != nil {
				t.Fatal(err)
			}
			if got := items(out); !slices.Equal(got, c.want) {
				t.Errorf("items = %q, want %q", got, c.want)
			}
		})
	}
}

// The name attribute says how the connection and the partner parts are
// named: short (the default) uses each element's short name when it has one
// (x's is ex, y's wy), long always the name.
func TestIfacesSegmentName(t *testing.T) {
	tr, ad := setupFixture(t, "ifaces")
	page := Attributes(pageAt(t, tr, "I/x").ID)
	cases := []struct {
		name, src string
		want      []string
	}{
		{"default: the short name when it has one", "sysml-ifaces::[]", []string{
			"i1 -> wy", "i3 -> wy, z", "c1 -> wy", "s1 -> ", "i2 -> z"}},
		{"short", "sysml-ifaces::[name=short]", []string{
			"i1 -> wy", "i3 -> wy, z", "c1 -> wy", "s1 -> ", "i2 -> z"}},
		{"long: the names", "sysml-ifaces::[name=long]", []string{
			"i1 -> y", "i3 -> y, z", "c1 -> y", "s1 -> ", "i2 -> z"}},
		{"LONG spelled in capitals", "sysml-ifaces::[name=LONG]", []string{
			"i1 -> y", "i3 -> y, z", "c1 -> y", "s1 -> ", "i2 -> z"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := ad.Convert(c.src, &asciidoctor.Options{Attributes: page})
			if err != nil {
				t.Fatal(err)
			}
			if got := items(out); !slices.Equal(got, c.want) {
				t.Errorf("items = %q, want %q", got, c.want)
			}
		})
	}
}

// The connection and the partner parts link to their pages, an anchor the
// document's converter writes (MC-18): <a href> in html5, its own markup in
// another converter's output.
func TestIfacesLinks(t *testing.T) {
	tr, ad := setupFixture(t, "ifaces")
	base, err := ad.BuiltinConverter("xhtml5")
	if err != nil {
		t.Fatal(err)
	}
	other := asciidoctor.ConverterFunc(func(n asciidoctor.Node, transform string, opts map[string]any) (string, error) {
		if to, ok := n.Attr(AttrLink).(string); ok && transform == "inline_anchor" {
			return "[" + to + ": " + n.(*asciidoctor.PhraseNode).Text() + "]", nil
		}
		return base.Convert(n, transform, opts)
	})
	if err := ad.RegisterConverter(other, &asciidoctor.BackendTraits{Basebackend: "html", HTMLSyntax: "xml"}, "other"); err != nil {
		t.Fatal(err)
	}
	i1 := pageAt(t, tr, "I/i1").ID
	y := pageAt(t, tr, "I/y").ID
	cases := []struct{ backend, want string }{
		{"html5", `<a href="../i1/index.html">i1</a> -> <a href="../y/index.html">wy</a>`},
		{"other", "[" + i1 + ": i1] -> [" + y + ": wy]"},
	}
	for _, c := range cases {
		t.Run(c.backend, func(t *testing.T) {
			out, err := ad.Convert("sysml-ifaces::x[]", &asciidoctor.Options{Backend: c.backend, Attributes: Attributes(pageAt(t, tr, "I/x").ID)})
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out, c.want) {
				t.Errorf("output %q does not contain %q", out, c.want)
			}
		})
	}
}

// A part with no owned ports, or with ports nothing references, drops the
// macro line: no item, no placeholder.
func TestIfacesEmpty(t *testing.T) {
	tr, ad := setupFixture(t, "ifaces")
	for _, dir := range []string{"I/n", "I/m"} {
		t.Run(dir, func(t *testing.T) {
			out, err := ad.Convert("sysml-ifaces::[]", &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, dir).ID)})
			if err != nil {
				t.Fatal(err)
			}
			if got := items(out); len(got) != 0 {
				t.Errorf("items = %q, want none", got)
			}
			if strings.Contains(out, Ifaces) {
				t.Errorf("output %q holds a leftover macro or placeholder", out)
			}
		})
	}
}

func TestIfacesErrors(t *testing.T) {
	tr, ad := setupFixture(t, "ifaces")
	page := Attributes(pageAt(t, tr, "I/x").ID)
	cases := []struct{ name, src, want string }{
		{"not a part: a package", "sysml-ifaces::pkg[]",
			"I::x: sysml-ifaces::pkg[]: I::pkg is not a part (a Package); name a part"},
		{"not a part: an interface", "sysml-ifaces::i1[]",
			"I::x: sysml-ifaces::i1[]: I::i1 is not a part (an InterfaceUsage); name a part"},
		{"name not short or long", "sysml-ifaces::[name=tall]",
			`I::x: sysml-ifaces::[name=tall]: name "tall" is not short or long`},
		{"unknown target", "sysml-ifaces::nope[]",
			`I::x: sysml-ifaces::nope[]: the model has no element named "nope"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mustFail(t, ad, c.src, page, c.want)
		})
	}
}
