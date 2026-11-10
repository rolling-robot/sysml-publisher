package render

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/rolling-robot/sysml-publisher/internal/model"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
	"github.com/rolling-robot/sysml-publisher/macros"
)

func fixture(name string) string { return filepath.Join("..", "..", "testdata", name) }

// build loads a fixture and builds its page tree.
func build(t *testing.T, name string) (*model.Model, *tree.Tree) {
	t.Helper()
	m, err := model.Folder{Dir: fixture(name)}.Load(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	tr, err := tree.Build(t.Context(), m)
	if err != nil {
		t.Fatal(err)
	}
	return m, tr
}

func page(t *testing.T, res *Result, qualifiedName string) *Page {
	t.Helper()
	for _, n := range res.Tree.Pages {
		if n.QualifiedName == qualifiedName {
			if p := res.Pages[n]; p != nil {
				return p
			}
		}
	}
	t.Fatalf("no page for %s", qualifiedName)
	return nil
}

// otherPages links every element to "page:<id>".
type otherPages struct{}

func (otherPages) Href(_, to string) (string, bool) { return "page:" + to, true }

// The macros' links between pages are the output's (MC-18): the tree's
// relative links by default, the given Pages' otherwise. The cases are the
// // notes in testdata/links/links.sysml.
func TestLinksFollowPages(t *testing.T) {
	_, tr := build(t, "links")
	var b *tree.Node
	for _, n := range tr.Pages {
		if n.QualifiedName == "L::b" {
			b = n
		}
	}
	if b == nil {
		t.Fatal("no page for L::b")
	}
	cases := []struct {
		name  string
		pages macros.Pages
		want  string
	}{
		{"tree", nil, `<a href="../b/index.html">b</a>`},
		{"given", otherPages{}, `<a href="page:` + b.ID + `">b</a>`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := Tree(t.Context(), tr, Options{Pages: c.pages})
			if err != nil {
				t.Fatal(err)
			}
			a := page(t, res, "L::a")
			if len(a.Own) != 1 || !strings.Contains(string(a.Own[0]), c.want) {
				t.Errorf("L::a's body %q, want it to contain %q", a.Own, c.want)
			}
		})
	}
}

// Starting asciidoctorg replaces the default slog logger (a dependency
// installs a handler that drops debug records); Tree must put it back.
func TestTreeKeepsTheDefaultLogger(t *testing.T) {
	previous := slog.Default()
	t.Cleanup(func() { slog.SetDefault(previous) })
	mine := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(mine)
	_, tr := build(t, "smoke")
	if _, err := Tree(t.Context(), tr, Options{}); err != nil {
		t.Fatal(err)
	}
	if slog.Default().Handler() != mine.Handler() {
		t.Errorf("default logger replaced by %T", slog.Default().Handler())
	}
}

// Cancelling stops a conversion itself, not only the loop between pages
// (CLI-7): a representation without model-reading macros, which nothing
// but asciidoctorg's checkpoints could stop, fails with the context's
// error, and converts when the context is live.
func TestConversionStopsWhenCancelled(t *testing.T) {
	m, tr := build(t, "titles")
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

// Asciidoctor's own messages reach the run's logger as records with the
// page's element and the model line they are about (CLI-6), once per page
// that shows the representation; the cases are the // notes in
// testdata/asciidoc-messages/messages.sysml. Macro warnings are not mixed in.
func TestAsciidocMessagesAreLogged(t *testing.T) {
	_, tr := build(t, "asciidoc-messages")
	var buf strings.Builder
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
	res, err := Tree(t.Context(), tr, Options{Logger: logger})
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(fixture("asciidoc-messages"), "messages.sysml")
	var got []string
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if strings.Contains(line, "asciidoc:") {
			got = append(got, line)
		}
	}
	var want []string
	for _, el := range []string{"M::pd", "M::u"} {
		want = append(want, `level=WARN msg="asciidoc: section title out of sequence: expected level 1, got level 3" element=`+el+" location="+file+":18")
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("records:\n got %q\nwant %q", got, want)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("macro warnings %q, want none", res.Warnings)
	}
}
