package site

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rolling-robot/sysml-publisher/internal/model"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
	"github.com/rolling-robot/sysml-publisher/macros"
)

func render(t *testing.T, fixture string) *Site {
	t.Helper()
	m, err := model.Folder{Dir: filepath.Join("..", "..", "testdata", fixture)}.Load(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	tr, err := tree.Build(t.Context(), m)
	if err != nil {
		t.Fatal(err)
	}
	s, err := Render(t.Context(), tr, nil)
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

// A failing macro leaves a placeholder on every page that shows it and a
// warning naming the file and line where it is written, the page and the
// macro, and the run goes on (MC-19); the cases are the // notes in
// testdata/macroerror/macroerror.sysml. OpenSysML's wording of the
// evaluation error is not pinned.
func TestMacroFailureLeavesPlaceholder(t *testing.T) {
	s := render(t, "macroerror")
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
	s := render(t, "warnings")
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

// Starting asciidoctorg replaces the default slog logger (a dependency
// installs a handler that drops debug records); Render must put it back.
func TestRenderKeepsTheDefaultLogger(t *testing.T) {
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })
	mine := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(mine)
	render(t, "smoke")
	if slog.Default().Handler() != mine.Handler() {
		t.Errorf("default logger replaced by %T", slog.Default().Handler())
	}
}

// Cancelling stops a conversion itself, not only the loop between pages
// (CLI-7): a representation without model-reading macros, which nothing
// but asciidoctorg's checkpoints could stop, fails with the context's
// error, and converts when the context is live.
func TestConversionStopsWhenCancelled(t *testing.T) {
	m, err := model.Folder{Dir: filepath.Join("..", "..", "testdata", "titles")}.Load(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	tr, err := tree.Build(t.Context(), m)
	if err != nil {
		t.Fatal(err)
	}
	ad, err := newAsciidoctor()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ad.Close() })
	if _, err := macros.Register(t.Context(), ad, m.Model, tr, nil); err != nil {
		t.Fatal(err)
	}
	r := &renderer{t: tr, ad: ad, sources: newSources(m.Files)}
	n := tr.Pages[0]
	rep := tree.Rep{ID: "plain", Body: "Plain *text*.\n\n* one\n* two\n"}

	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := r.convert(ctx, n, rep, &body{}); !errors.Is(err, context.Canceled) {
		t.Errorf("cancelled conversion: err = %v, want context.Canceled", err)
	}
	out, err := r.convert(t.Context(), n, rep, &body{})
	if err != nil || !strings.Contains(string(out), "<strong>text</strong>") {
		t.Errorf("live conversion: out %q, err %v", out, err)
	}
}
