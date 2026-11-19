// Copyright 2026 Petr Mukhachev
// SPDX-License-Identifier: MIT

package site

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rolling-robot/sysml-publisher/internal/model"
	"github.com/rolling-robot/sysml-publisher/internal/servicetest"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

func renderSite(t *testing.T, fixture string) *Site {
	return renderSiteOptions(t, fixture, Options{})
}

func renderSiteOptions(t *testing.T, fixture string, o Options) *Site {
	t.Helper()
	m, err := model.Folder{Dir: filepath.Join("..", "..", "testdata", fixture), API: servicetest.API(t)}.Load(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	tr, err := tree.Build(t.Context(), m)
	if err != nil {
		t.Fatal(err)
	}
	s, err := Render(t.Context(), tr, nil, nil, o)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func mustContain(t *testing.T, s *Site, file string, wants ...string) {
	t.Helper()
	html := string(s.Files[file])
	if html == "" {
		t.Fatalf("%s not rendered", file)
	}
	for _, w := range wants {
		if !strings.Contains(html, w) {
			t.Errorf("%s does not contain %q", file, w)
		}
	}
}

func TestRenderSmoke(t *testing.T) {
	s := renderSite(t, "smoke")
	want := []string{"Model/aaaa/index.html", "Model/bbb/index.html", "Model/index.html", "Model/pd/index.html", "index.html"}
	if got := s.Pages(); !reflect.DeepEqual(got, want) {
		t.Fatalf("pages = %v, want %v", got, want)
	}
	// pd's own representation, rendered by asciidoctorg as a paragraph.
	mustContain(t, s, "Model/pd/index.html",
		"<h1>Element: pd</h1>",
		`<p class="qualified-name">Model::pd</p>`,
		`<div class="paragraph">`, "Textual representation",
		`href="../../assets/layout.css"`)
	// aaaa shows pd's representation as inherited, with no label (PG-7).
	mustContain(t, s, "Model/aaaa/index.html",
		`<section class="rep inherited">`,
		"Textual representation")
	if strings.Contains(string(s.Files["Model/aaaa/index.html"]), `class="from"`) {
		t.Error("inherited representation has an owner label (PG-7)")
	}
	// The sidebar lists the whole tree, marks the current page, and links
	// the empty ancestor page too (SB-1..3). The branch holding the
	// current page's path is a collapsible details, open (SB-6).
	mustContain(t, s, "Model/aaaa/index.html",
		`<aside class="sidebar"`,
		`<a href="index.html" aria-current="page">Element: aaaa</a>`,
		`<li class="empty"><details data-branch="Model" open><summary><a href="../index.html">Model</a></summary>`,
		`<a href="../bbb/index.html">Element: bbb</a>`)
	mustContain(t, s, "Model/index.html", `<p class="empty-page">`)
	mustContain(t, s, "index.html", "<h1>"+LandingTitle+"</h1>", `<a href="Model/pd/index.html">Element: pd</a>`)
	for _, f := range []string{"assets/asciidoctor.css", "assets/layout.css", "assets/sort.js", "assets/sidebar.js", Marker} {
		if len(s.Files[f]) == 0 {
			t.Errorf("%s missing", f)
		}
	}
	// Every page loads the table-sorting (SB-4) and sidebar-resizing (SB-5)
	// scripts, from the same assets folder as the stylesheets.
	mustContain(t, s, "Model/pd/index.html",
		`<script src="../../assets/sort.js" defer></script>`,
		`<script src="../../assets/sidebar.js" defer></script>`,
		`<div class="sidebar-resize" role="separator" aria-orientation="vertical"`)
}

// The sidebar's branches are collapsible details (SB-6): the branches on the
// current page's path are open, the others closed, a leaf is a plain link,
// and the landing page opens nothing.
func TestSidebarBranchesCollapsible(t *testing.T) {
	s := renderSite(t, "smoke")
	mustContain(t, s, "Model/aaaa/index.html",
		`<details data-branch="Model" open><summary><a href="../index.html">Model</a></summary>`)
	mustContain(t, s, "Model/index.html",
		`<details data-branch="Model" open><summary><a href="index.html" aria-current="page">Model</a></summary>`)
	// The landing page: the branch is there, closed.
	mustContain(t, s, "index.html", `<details data-branch="Model"><summary>`)
	// A leaf is a plain link, no details of its own.
	mustContain(t, s, "Model/aaaa/index.html", `<li><a href="index.html" aria-current="page">Element: aaaa</a></li>`)
	if strings.Contains(string(s.Files["Model/aaaa/index.html"]), `data-branch="Model/aaaa"`) {
		t.Error("leaf aaaa is a branch")
	}
}

func TestHrefEscapesSegments(t *testing.T) {
	from := &tree.Node{Dir: "A/b c"}
	to := &tree.Node{Dir: "A/x#y"}
	if got, want := tree.Href(from, to), "../x%23y/index.html"; got != want {
		t.Errorf("href = %q, want %q", got, want)
	}
}

// canonical joins the base URL and a page's folder (CFG-5); no base URL, no
// canonical link.
func TestCanonical(t *testing.T) {
	cases := []struct{ base, dir, want string }{
		{"https://example.com", "", "https://example.com/"},
		{"https://example.com/", "", "https://example.com/"},
		{"https://example.com/docs", "Model/pd", "https://example.com/docs/Model/pd/"},
		{"https://example.com/docs/", "Model/pd", "https://example.com/docs/Model/pd/"},
		{"https://example.com", "A/b c", "https://example.com/A/b%20c/"},
	}
	for _, c := range cases {
		if got := canonical(c.base, c.dir); got != c.want {
			t.Errorf("canonical(%q, %q) = %q, want %q", c.base, c.dir, got, c.want)
		}
	}
	if got := canonical("", "A"); got != "" {
		t.Errorf("canonical(\"\") = %q, want \"\"", got)
	}
}

// Options' title (CFG-4) is the landing page's and the tabs' suffix; the
// base URL (CFG-5) puts a canonical link on every page.
func TestOptionsTitleAndCanonical(t *testing.T) {
	s := renderSiteOptions(t, "smoke", Options{Title: "Model docs", BaseURL: "https://example.com/docs/"})
	mustContain(t, s, "index.html",
		"<h1>Model docs</h1>",
		`<a class="home" href="index.html" aria-current="page">Model docs</a>`,
		`<link rel="canonical" href="https://example.com/docs/">`)
	mustContain(t, s, "Model/pd/index.html",
		"<title>Element: pd — Model docs</title>",
		`<link rel="canonical" href="https://example.com/docs/Model/pd/">`)
	// Without a base URL no page carries a canonical link.
	s = renderSite(t, "smoke")
	for _, f := range s.Pages() {
		if strings.Contains(string(s.Files[f]), "canonical") {
			t.Errorf("%s has a canonical link without a base URL", f)
		}
	}
}

func TestWriteReplacesOnlyItsOwnFolder(t *testing.T) {
	s := &Site{Files: map[string][]byte{"index.html": []byte("x"), Marker: []byte("m")}}
	root := t.TempDir()

	// A folder the user made is not replaced.
	user := filepath.Join(root, "user")
	if err := os.MkdirAll(user, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(user, "notes.txt"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.Write(user); !errors.Is(err, ErrNotASite) {
		t.Fatalf("Write(user folder) = %v, want ErrNotASite", err)
	}
	if _, err := os.Stat(filepath.Join(user, "notes.txt")); err != nil {
		t.Fatal("the user's file was touched")
	}

	// A new folder is created; a second run replaces it, dropping stale pages.
	out := filepath.Join(root, "site")
	if err := s.Write(out); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(out, "Old", "index.html")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stale, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := s.Write(out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !errors.Is(err, os.ErrNotExist) {
		t.Error("stale page survived a rerun (CLI-4)")
	}
}

// pd's representation uses sysml-element-name:[], so it renders differently
// on each page that shows it (MC-2, MC-3).
func TestRepresentationRendersPerPage(t *testing.T) {
	s := renderSite(t, "macros")
	mustContain(t, s, "M/pd/index.html", "<p>Name: pd.</p>")
	mustContain(t, s, "M/a_b/index.html", "<p>Name: a&lt;b.</p>")
	mustContain(t, s, "M/unnamed-1/index.html", "<p>Name: Unnamed.</p>")
}

// A failing macro leaves a placeholder on every page that shows it and a
// warning naming the file and line where it is written, the page and the
// macro, and the run goes on (MC-19); the cases are the // notes in
// testdata/macroerror/macroerror.sysml. OpenSysML's wording of the
// evaluation error is not pinned.
func TestMacroFailureLeavesPlaceholder(t *testing.T) {
	s := renderSite(t, "macroerror")
	file := filepath.Join("..", "..", "testdata", "macroerror", "macroerror.sysml")
	// message is empty where it is OpenSysML's.
	want := []struct{ prefix, message string }{
		{file + ":21: M::a: sysml-value:mas[]: ", ""},
		{file + ":21: M::pd: sysml-value:mas[]: ", ""},
		{file + ":26: M::b: sysml-doc::Nothing[]: ", `the model has no element named "Nothing"`},
	}
	if len(s.Warnings) != len(want) {
		t.Fatalf("warnings:\n got %q\nwant %v", s.Warnings, want)
	}
	for i, w := range want {
		got, ok := strings.CutPrefix(s.Warnings[i], w.prefix)
		if !ok || got == "" || (w.message != "" && got != w.message) {
			t.Errorf("warning %d:\n got %q\nwant %q%s", i, s.Warnings[i], w.prefix, w.message)
		}
	}
	mustContain(t, s, "M/a/index.html", `<span class="sysml-failed">sysml-value:mas[]: `)
	mustContain(t, s, "M/pd/index.html", `<span class="sysml-failed">sysml-value:mas[]: `)
	mustContain(t, s, "M/b/index.html",
		`<div class="sysml-failed"><p>sysml-doc::Nothing[]: the model has no element named &#34;Nothing&#34;</p></div>`)
}

// Macro warnings are reported once per page and macro (MC-7); the cases are
// the // notes in testdata/warnings/warnings.sysml.
func TestWarningsOncePerPageAndMacro(t *testing.T) {
	s := renderSite(t, "warnings")
	// Each warning is placed where the macro is first written in the
	// representation that warned first on that page: Tagged's own, else
	// Pump's (PG-7: nearest first).
	file := filepath.Join("..", "..", "testdata", "warnings", "warnings.sysml")
	var want []string
	for _, c := range []struct{ page, position string }{
		{"W::Pump", "25"}, {"W::Tagged", "32"}, {"W::a", "32"}, {"W::b", "25"},
	} {
		want = append(want,
			file+":"+c.position+": "+c.page+": sysml-value:position[]: position is composite; shown as [composite]",
			file+":27: "+c.page+": sysml-value:home[]: home is composite; shown as [composite]")
	}
	if !reflect.DeepEqual(s.Warnings, want) {
		t.Errorf("warnings:\n got %q\nwant %q", s.Warnings, want)
	}
}

// Page titles come from the first representation's document title, with
// macros resolved; untitled-N and -N suffixes number siblings (PG-8, PG-9).
func TestPageTitles(t *testing.T) {
	s := renderSite(t, "titles")
	cases := []struct{ dir, h1, sidebar string }{
		{"T/titled", "Element: titled", "Element: titled"},
		{"T/plain1", "untitled-1", "untitled-1"},
		{"T/plain2", "untitled-2", "untitled-2"},
		{"T/dupA", "Same", "Same"},
		{"T/dupB", "Same-2", "Same-2"},
		// The first representation has no title; the second gives it.
		{"T/two", "Second", "Second"},
		// Inherited: the title is resolved on the usage's page.
		{"T/u", "Element: u", "Element: u"},
		// The heading keeps inline markup; the tab and sidebar do not.
		{"T/markup", "A <strong>bold</strong> &lt;thing&gt;", "A bold &lt;thing&gt;"},
	}
	for _, c := range cases {
		f := c.dir + "/index.html"
		mustContain(t, s, f, "<h1>"+c.h1+"</h1>", "<title>"+c.sidebar+" — "+LandingTitle+"</title>")
		mustContain(t, s, "index.html", ">"+c.sidebar+"</a>")
	}
	// Titles after the one that names the page stay in their bodies (PG-10).
	mustContain(t, s, "T/two/index.html", "<h1>Third</h1>", "Third body.")
	if strings.Contains(string(s.Files["T/two/index.html"]), "<h1>Second</h1>\n<div") {
		t.Error("the title used for the page should not repeat in its body")
	}
	// The empty package page keeps the element name (PG-6).
	mustContain(t, s, "T/index.html", "<h1>T</h1>")
}

// A page whose representation shows a figure (MC-25) loads the site's
// mermaid.js, which draws it; pages without a figure, and the landing page,
// load nothing, and the site carries the asset only when a figure needs it.
func TestMermaidLoadsOnFigurePages(t *testing.T) {
	s := renderSite(t, "figs")
	if _, ok := s.Files["assets/mermaid.min.js"]; !ok {
		t.Error("the site carries no mermaid.min.js though a page shows a figure")
	}
	mustContain(t, s, "F/vehicle/index.html", `<script src="../../assets/mermaid.min.js" defer></script>`)
	if strings.Contains(string(s.Files["F/vehicle/i1/index.html"]), "mermaid.min.js") {
		t.Error("the page of i1 loads mermaid.js though its body has no figure")
	}
	if strings.Contains(string(s.Files["index.html"]), "mermaid.min.js") {
		t.Error("the landing page loads mermaid.js though it shows no figure")
	}

	// A site with no figure carries no asset and no script.
	s = renderSite(t, "ifaces")
	if _, ok := s.Files["assets/mermaid.min.js"]; ok {
		t.Error("the site carries mermaid.min.js though no page shows a figure")
	}
	mustContain(t, s, "I/x/index.html", `<script src="../../assets/sort.js" defer></script>`)
	if strings.Contains(string(s.Files["I/x/index.html"]), "mermaid.min.js") {
		t.Error("the page of x loads mermaid.js though it shows no figure")
	}
}
