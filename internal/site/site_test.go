package site

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rolling-robot/sysml-publisher/internal/model"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

func render(t *testing.T, fixture string) *Site {
	t.Helper()
	m, err := model.Folder{Dir: filepath.Join("..", "..", "testdata", fixture)}.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	s, err := Render(tree.Build(m))
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
	s := render(t, "smoke")
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
	// the empty ancestor page too (SB-1..3).
	mustContain(t, s, "Model/aaaa/index.html",
		`<aside class="sidebar"`,
		`<a href="index.html" aria-current="page">Element: aaaa</a>`,
		`<li class="empty"><a href="../index.html">Model</a>`,
		`<a href="../bbb/index.html">Element: bbb</a>`)
	mustContain(t, s, "Model/index.html", `<p class="empty-page">`)
	mustContain(t, s, "index.html", "<h1>"+LandingTitle+"</h1>", `<a href="Model/pd/index.html">Element: pd</a>`)
	for _, f := range []string{"assets/asciidoctor.css", "assets/layout.css", Marker} {
		if len(s.Files[f]) == 0 {
			t.Errorf("%s missing", f)
		}
	}
}

func TestHrefEscapesSegments(t *testing.T) {
	from := &tree.Node{Dir: "A/b c"}
	to := &tree.Node{Dir: "A/x#y"}
	if got, want := tree.Href(from, to), "../x%23y/index.html"; got != want {
		t.Errorf("href = %q, want %q", got, want)
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
	s := render(t, "macros")
	mustContain(t, s, "M/pd/index.html", "<p>Name: pd.</p>")
	mustContain(t, s, "M/a_b/index.html", "<p>Name: a&lt;b.</p>")
	mustContain(t, s, "M/unnamed-1/index.html", "<p>Name: Unnamed.</p>")
}

// Page titles come from the first representation's document title, with
// macros resolved; untitled-N and -N suffixes number siblings (PG-8, PG-9).
func TestPageTitles(t *testing.T) {
	s := render(t, "titles")
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
