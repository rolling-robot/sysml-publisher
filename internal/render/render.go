// Package render converts the representations of a page tree with
// asciidoctorg and titles its pages (PG-3, PG-8..10, MC-3): what every
// output needs, whatever it writes the pages to. The HTML site (package
// site) lays the result out as files; another output can take the same
// result.
package render

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"strings"
	"time"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
	"github.com/rolling-robot/sysml-publisher/macros"
)

// Options are how Tree renders.
type Options struct {
	// Logger receives debug records (page durations) and Asciidoctor's own
	// messages, with the page and model location (asciidocLog); nil uses
	// slog.Default.
	Logger *slog.Logger
	// Pages decides the macros' links between pages (MC-18); nil uses the
	// tree's relative links (tree.Tree.Href), as in the HTML site.
	Pages macros.Pages
	// Views renders the views sysml-view shows (MC-21); nil leaves every
	// sysml-view a failure (MC-19).
	Views macros.Views
}

// Result is a rendered page tree.
type Result struct {
	Tree *tree.Tree
	// Pages has every page of Tree.Pages; the root (the landing page, PT-5)
	// has none.
	Pages map[*tree.Node]*Page
	// Warnings are the macros' warnings, each once per page and macro, in
	// the order first seen, failing macros included, with their location
	// filled when found (MC-7, MC-19).
	Warnings []macros.Warning
}

// Page is a page's title and converted representations: its own, then
// the inherited ones, nearest first (PG-7). An empty ancestor page (PG-6)
// has a title and no representations.
type Page struct {
	Title          Title
	Own, Inherited []template.HTML // asciidoctorg output, secure safe mode
}

// Title is a page title: HTML for the page heading, which may carry inline
// markup, and Text, without markup, for places that take plain text (a
// browser tab, the sidebar). Text is HTML-escaped already.
type Title struct {
	HTML, Text template.HTML
}

// PlainTitle is the title s, which has no markup.
func PlainTitle(s string) Title {
	escaped := template.HTML(template.HTMLEscapeString(s))
	return Title{HTML: escaped, Text: escaped}
}

func (t Title) suffixed(n int) Title {
	suffix := template.HTML(fmt.Sprintf("-%d", n))
	return Title{HTML: t.HTML + suffix, Text: t.Text + suffix}
}

// Tree converts the representations of every page of t, on every page that
// shows them, since macros may render them differently per page (MC-3),
// then titles the pages. Cancelling ctx stops the rendering (CLI-7):
// between pages, inside a conversion at asciidoctorg's next checkpoint
// (before each block is parsed or converted, and at each macro), and inside
// macro evaluations. A macro that fails leaves a placeholder on its page
// and a warning, and rendering goes on (MC-19); only a cancelled ctx or a
// rendering bug returns an error.
func Tree(ctx context.Context, t *tree.Tree, opts Options) (*Result, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	var pages macros.Pages = t
	if opts.Pages != nil {
		pages = opts.Pages
	}
	ad, err := newAsciidoctor()
	if err != nil {
		return nil, fmt.Errorf("starting asciidoctorg: %w", err)
	}
	defer ad.Close()
	res := &Result{Tree: t, Pages: map[*tree.Node]*Page{}}
	r := &renderer{t: t, ad: ad, sources: newSources(t.Model.Files)}
	// One warning per page and macro (MC-7): a macro written twice on a
	// page, or shown there by two representations, warns once; every page
	// that shows it warns on its own. Macros warn while a representation is
	// converted, so the warning is placed in that one (MC-19).
	type key struct{ page, macro string }
	seen := map[key]bool{}
	warn := func(w macros.Warning) {
		k := key{w.Page, w.Macro}
		if seen[k] {
			return
		}
		seen[k] = true
		if r.current != nil && w.Location == "" {
			w.Location = r.sources.locate(*r.current, w.Macro)
		}
		res.Warnings = append(res.Warnings, w)
	}
	reg, err := macros.Register(ctx, ad, t.Model.Model, pages, warn)
	if err != nil {
		return nil, err
	}
	if opts.Views != nil {
		reg.SetViews(opts.Views)
	}
	if _, err := ad.RegisterLogHandler(asciidocLog{r: r, logger: logger}); err != nil {
		return nil, fmt.Errorf("routing asciidoctorg's messages: %w", err)
	}

	// Titles come from the converted bodies, so convert all bodies first.
	bodies := map[*tree.Node]*body{}
	for _, n := range t.Pages {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		start := time.Now()
		b, err := r.body(ctx, n)
		if err != nil {
			return nil, err
		}
		bodies[n] = b
		logger.DebugContext(ctx, "page converted", "element", n.QualifiedName,
			"representations", len(n.Own)+len(n.Inherited), "duration", time.Since(start))
	}
	assignTitles(t.Root, bodies, res.Pages)
	return res, nil
}

// newAsciidoctor starts asciidoctorg. Starting it replaces the process-wide
// default slog logger (a dependency installs its own handler, which drops
// debug records), so the previous default is put back.
func newAsciidoctor() (*asciidoctor.Asciidoctor, error) {
	previous := slog.Default()
	defer slog.SetDefault(previous)
	return asciidoctor.New()
}

type renderer struct {
	t       *tree.Tree
	ad      *asciidoctor.Asciidoctor
	sources *sources
	// current is the representation being converted and currentPage the
	// page it is converted for, for the location of macro warnings and of
	// Asciidoctor's messages.
	current     *tree.Rep
	currentPage *tree.Node
}

// body is a page's converted representations, and the document title of the
// first of them that has one (PG-8).
type body struct {
	own, inherited []template.HTML
	docTitle       *Title
}

// body converts the representations of n as shown on its page.
func (r *renderer) body(ctx context.Context, n *tree.Node) (*body, error) {
	b := &body{}
	for _, rep := range n.Own {
		h, err := r.convert(ctx, n, rep, b)
		if err != nil {
			return nil, err
		}
		b.own = append(b.own, h)
	}
	for _, rep := range n.Inherited {
		h, err := r.convert(ctx, n, rep, b)
		if err != nil {
			return nil, err
		}
		b.inherited = append(b.inherited, h)
	}
	return b, nil
}

// convert converts rep as shown on the page of n, which macros may refer to.
// The first document title found becomes the page title and is left out of
// the body; the titles of later representations stay in theirs (PG-10).
// Cancelling ctx stops the conversion; the error then satisfies
// errors.Is(err, ctx.Err()).
func (r *renderer) convert(ctx context.Context, n *tree.Node, rep tree.Rep, b *body) (template.HTML, error) {
	attrs := macros.Attributes(n.ID)
	if b.docTitle != nil {
		attrs = attrs.Set("showtitle", "")
	}
	r.current, r.currentPage = &rep, n
	doc, out, err := r.ad.ConvertDocumentContext(ctx, rep.Body, &asciidoctor.Options{Attributes: attrs})
	r.current, r.currentPage = nil, nil
	if err != nil {
		return "", fmt.Errorf("converting the representation of %s on the page of %s: %w", r.owner(rep).QualifiedName, n.QualifiedName, err)
	}
	if b.docTitle == nil && doc.HasHeader() {
		// Both are asciidoctorg output: the converted title, and the same
		// title with its markup removed.
		t := Title{HTML: template.HTML(doc.Doctitle())}
		if st := doc.StructuredDoctitle(true); st != nil {
			t.Text = template.HTML(st.Combined)
		}
		if strings.TrimSpace(string(t.Text)) != "" {
			b.docTitle = &t
		}
	}
	return template.HTML(out), nil // asciidoctorg output, secure safe mode
}

// assignTitles makes the pages of the children of parent, then of their
// children. A page takes its document title, or untitled-N when its
// representations have none (PG-8); an empty ancestor page takes its
// element's name (PG-6). A title a sibling already has gets a -N suffix
// (PG-9).
func assignTitles(parent *tree.Node, bodies map[*tree.Node]*body, pages map[*tree.Node]*Page) {
	untitled := 0
	taken := map[template.HTML]bool{}
	for _, c := range parent.Children {
		p := &Page{}
		b := bodies[c]
		switch {
		case !c.Documented():
			p.Title = PlainTitle(c.Name)
		case b != nil && b.docTitle != nil:
			p.Title = *b.docTitle
		default:
			untitled++
			p.Title = PlainTitle(fmt.Sprintf("untitled-%d", untitled))
		}
		if taken[p.Title.Text] {
			i := 2
			for taken[p.Title.suffixed(i).Text] {
				i++
			}
			p.Title = p.Title.suffixed(i)
		}
		taken[p.Title.Text] = true
		if b != nil {
			p.Own, p.Inherited = b.own, b.inherited
		}
		pages[c] = p
		assignTitles(c, bodies, pages)
	}
}

func (r *renderer) owner(rep tree.Rep) *tree.Node {
	if n := r.t.ByID[rep.Owner]; n != nil {
		return n
	}
	return &tree.Node{Name: rep.Owner, QualifiedName: rep.Owner}
}
