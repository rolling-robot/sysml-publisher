// Package macros holds the sysml-* AsciiDoc macros (requirements MC-1..20),
// which put facts from a SysML v2 model into AsciiDoc text: names, docs,
// members, metadata, evaluated values. Each macro lives in its own file;
// Register adds them all to an asciidoctorg instance.
//
// The package is usable from other modules. It reads the model through
// package sysml, and links to the host's pages through Pages:
//
//	m, err := sysml.Open(ctx, client, parsed) // an OpenSysML-parsed model
//	ad, err := asciidoctor.New()
//	r, err := macros.Register(ctx, ad, m, pages, warn)
//	err = r.Inline("my-macro", func(c *macros.Call) (string, error) { ... })
//	html, err := ad.Convert(body, &asciidoctor.Options{Attributes: macros.Attributes(elementID)})
//
// Macros of the caller's own (Registry.Inline, Registry.Block) get a Call:
// the page's element, the target, name lookup, links, evaluation and
// warnings, the way the sysml-* macros have them; an error they return
// leaves a placeholder and a warning instead of stopping the conversion.
package macros

import (
	"context"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/sysml"
)

// AttrElement is the document attribute naming the page's element by id,
// set by the renderer on every representation it converts (MC-5).
const AttrElement = "sysml-element"

// Attributes are the document attributes a representation is converted with
// for the element id: on its page, macros describe that element (MC-2).
func Attributes(id string) asciidoctor.Attributes {
	return asciidoctor.Attributes{}.Set(AttrElement, id)
}

// AttrLink is the attribute of the anchor nodes macros link with (MC-18):
// the id of the element whose page the link goes to. A link between pages
// is an anchor node of type link, converted by the document's converter
// like any link: html5 writes <a href>, and a converter for another output
// can tell these links by AttrLink and write its own (Confluence's
// <ac:link>).
const AttrLink = "sysml-link"

// Pages is how the host places elements on pages, for the links macros
// write (MC-18, MC-12, MC-17). A nil Pages links nothing.
type Pages interface {
	// Href is the link, relative to the page of element from, to the page
	// of element to (both by id); ok is false when to has no page.
	Href(from, to string) (href string, ok bool)
}

// Warning is a problem a macro reports while still producing output, such as
// a value it cannot show, or a macro that failed and left a placeholder
// (MC-19). Its parts are kept apart so the receiver can tell
// repeats without parsing the text (MC-7: one warning per page and macro).
type Warning struct {
	// Page is the qualified name (or positional name, M::@2) of the element
	// the macro was converted for.
	Page string
	// Macro is the macro as written: sysml-value:position[].
	Macro string
	// Message says what is wrong, without the page or the macro.
	Message string
	// Location is where the macro is written in the model, file:line, or
	// empty when unknown. Macros leave it empty: they see the
	// representation's text, not its file; the renderer fills it in.
	Location string
}

// String is the warning as reported: "<location>: <page>: <macro>:
// <message>", without the location when it is unknown.
func (w Warning) String() string {
	s := w.Page + ": " + w.Macro + ": " + w.Message
	if w.Location != "" {
		s = w.Location + ": " + s
	}
	return s
}

// Warn receives a warning from a macro. A nil Warn drops warnings.
type Warn func(Warning)

// Register adds every sysml-* macro to ad. The macros read the model m
// (DS-4), link to the pages pages gives, report warnings to warn and stop
// reading and evaluating when ctx is cancelled. ctx is the context of the
// rendering ad is registered for; register again for another. m must stay
// open while ad converts. The Registry adds macros of the caller's own
// (MC-20).
func Register(ctx context.Context, ad *asciidoctor.Asciidoctor, m *sysml.Model, pages Pages, warn Warn) (*Registry, error) {
	if m == nil {
		return nil, errors.New("registering the sysml macros: no model")
	}
	if warn == nil {
		warn = func(Warning) {}
	}
	e := newEnv(ctx, m, pages, warn)
	g := ad.Extensions().Group("sysml")
	registerName(g, e)
	registerRep(g, e)
	registerQualifiedName(g, e)
	registerShortName(g, e)
	registerMembersList(g, e)
	registerIfaceEnds(g, e)
	registerDoc(g, e)
	registerMetadataTable(g, e)
	registerValue(g, e)
	if err := g.Register(); err != nil {
		return nil, fmt.Errorf("registering the sysml macros: %w", err)
	}
	return &Registry{ad: ad, e: e}, nil
}

// env is what every macro of one rendering shares: the rendering's
// context, the model, the host's pages, and the lookups built on the
// model, so a name or a metadata def read by one macro is not read again by
// the next.
type env struct {
	ctx   context.Context
	m     *sysml.Model
	pages Pages
	warn  Warn
	names *nameIndex
	md    *metadataIndex
	// kindIndex is made on first use, by kinds.
	kindIndex kindIndex
}

func newEnv(ctx context.Context, m *sysml.Model, pages Pages, warn Warn) *env {
	names := newNameIndex(ctx, m)
	return &env{
		ctx:   ctx,
		m:     m,
		pages: pages,
		warn:  warn,
		names: names,
		md:    &metadataIndex{ctx: ctx, m: m, names: names},
	}
}

// element is the element a representation is converted for: the page's
// element (MC-2), which macros without a target describe.
type element struct {
	id string
	// qualifiedName is its qualified name, or its positional name (M::@2);
	// what warnings name and what Evaluate takes as subject.
	qualifiedName string
}

// pageElement is the element whose page is being rendered (MC-2), named by
// the document attribute sysml-element. A missing attribute, or one naming
// no element of the model, is the renderer's mistake: an error.
func pageElement(parent asciidoctor.Node, e *env) (element, error) {
	id, _ := parent.Document().Attr(AttrElement).(string)
	if id == "" {
		return element{}, fmt.Errorf("attribute %s is not set", AttrElement)
	}
	qn, err := e.m.QualifiedName(e.ctx, id)
	if err != nil {
		return element{}, fmt.Errorf("attribute %s names %q: %w", AttrElement, id, err)
	}
	return element{id: id, qualifiedName: qn}, nil
}

// href is the link from the page of element from to the page of element
// to, and whether there is one: none when to has no page.
func (e *env) href(from, to string) (string, bool) {
	if e.pages == nil {
		return "", false
	}
	return e.pages.Href(from, to)
}

// optionalTargetRx is the pattern of an inline macro whose target is
// optional, name:[] or name:<target>[]: Asciidoctor's own inline forms take
// a target always (long) or never (short). A target runs up to "[" without
// spaces, except inside a quoted segment: name:M::'top speed'[]. A leading
// backslash escapes the macro, as for every inline macro.
func optionalTargetRx(name string) string {
	return `\\?` + regexp.QuoteMeta(name) + `:((?:'(?:[^'\\]|\\.)*'|[^\s\['])*)\[(|.*?[^\\])\]`
}

// FailedRole is the class of the placeholder a failing macro leaves on the
// page (MC-19), for styling.
const FailedRole = "sysml-failed"

// call is one use of a macro on the page of an element: what a macro's body
// gets from inline or block.
type call[P asciidoctor.Node] struct {
	parent P
	// page is the element whose page is being rendered (MC-2).
	page element
	// target is the target as written, unescaped.
	target string
	attrs  map[string]any
	// macro is the macro as written, for warnings: sysml-value:mass[].
	macro string
}

// written is how a macro is written in warnings and placeholders, from its
// name, the separator (":" inline, "::" block), its target and attributes.
type written func(name, sep, target string, attrs map[string]any) string

// plainWritten writes a macro without its attributes: name:target[].
func plainWritten(name, sep, target string, _ map[string]any) string {
	return name + sep + target + "[]"
}

// inline registers the inline macro name. body returns the HTML that
// replaces the macro; an error from it is the macro failing (fail). Targets
// arrive unescaped: inline macros run after special characters are escaped,
// so M::'a<b' arrives as M::'a&lt;b'.
func inline(g *asciidoctor.ExtensionGroup, e *env, name string, opts *asciidoctor.MacroOptions, w written, body func(c call[asciidoctor.Node]) (string, error)) {
	if w == nil {
		w = plainWritten
	}
	g.InlineMacro(name, asciidoctor.InlineMacroProcessorFunc(
		func(parent asciidoctor.Node, target string, attrs map[string]any) (*asciidoctor.PhraseNode, error) {
			n, err := pageElement(parent, e)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			target = html.UnescapeString(target)
			c := call[asciidoctor.Node]{parent: parent, page: n, target: target, attrs: attrs, macro: w(name, ":", target, attrs)}
			out, err := body(c)
			if err != nil {
				if err := e.fail(n, c.macro, err); err != nil {
					return nil, err
				}
				out = `<span class="` + FailedRole + `">` + failedText(c.macro, err) + `</span>`
			}
			return asciidoctor.CreatePhraseNode(parent, "quoted", out, nil, nil)
		}), opts)
}

// block registers the block macro name. body returns the block that
// replaces the macro, or nil to drop the macro line; an error from it is the
// macro failing (fail).
func block(g *asciidoctor.ExtensionGroup, e *env, name string, opts *asciidoctor.MacroOptions, w written, body func(c call[asciidoctor.StructuralNode]) (asciidoctor.StructuralNode, error)) {
	if w == nil {
		w = plainWritten
	}
	g.BlockMacro(name, asciidoctor.BlockMacroProcessorFunc(
		func(parent asciidoctor.StructuralNode, target string, attrs map[string]any) (asciidoctor.StructuralNode, error) {
			n, err := pageElement(parent, e)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			c := call[asciidoctor.StructuralNode]{parent: parent, page: n, target: target, attrs: attrs, macro: w(name, "::", target, attrs)}
			out, err := body(c)
			if err == nil {
				return out, nil
			}
			if err := e.fail(n, c.macro, err); err != nil {
				return nil, err
			}
			return asciidoctor.CreatePassBlock(parent,
				`<div class="`+FailedRole+`"><p>`+failedText(c.macro, err)+`</p></div>`, nil, nil)
		}), opts)
}

// fail handles a macro that cannot do its job (MC-7, MC-19): it reports a
// warning naming the page and the macro, and the caller leaves a placeholder
// in the macro's place, so the run goes on and every failing page is
// reported. A cancelled rendering is no macro failure: its error is returned,
// which stops the conversion.
func (e *env) fail(n element, macro string, err error) error {
	if e.ctx.Err() != nil {
		return err
	}
	e.warn(Warning{Page: n.qualifiedName, Macro: macro, Message: err.Error()})
	return nil
}

// failedText is the escaped text of a failing macro's placeholder: the
// macro as written and what went wrong.
func failedText(macro string, err error) string {
	return html.EscapeString(macro + ": " + err.Error())
}

// registerText registers the inline macro name:[<element>], replaced by text
// about an element: ofElement gives it for the page's element when there is
// no target, else for the element the target names, read by nameTarget
// (MC-15, MC-8). The text links to the element's page (MC-18).
func registerText(g *asciidoctor.ExtensionGroup, e *env, name string, ofElement func(id string) (string, error)) {
	inline(g, e, name, &asciidoctor.MacroOptions{Regexp: optionalTargetRx(name)}, nil,
		func(c call[asciidoctor.Node]) (string, error) {
			id := c.page.id
			if c.target != "" {
				var err error
				if id, err = nameTarget(c.target, c.page, e); err != nil {
					return "", err
				}
			}
			text, err := ofElement(id)
			if err != nil {
				return "", err
			}
			// An unquoted phrase is written out verbatim, so escape the text:
			// SysML names may contain <, > and &.
			return e.linkTo(c.parent, c.page.id, id, html.EscapeString(text))
		})
}

// linkTo is the HTML text linked, on the page of element from, to the page
// of element to (MC-18): unlinked when to has no page, when to is from
// itself, or when the text is empty, which leaves nothing to click.
func (e *env) linkTo(parent asciidoctor.Node, from, to, text string) (string, error) {
	href, ok := e.href(from, to)
	if !ok || from == to || text == "" {
		return text, nil
	}
	return link(parent, to, href, text)
}

// link is text, HTML already, linked to href, the page of element to: an
// anchor node carrying AttrLink, converted by the document's converter, so
// the output decides the markup.
func link(parent asciidoctor.Node, to, href, text string) (string, error) {
	a, err := asciidoctor.CreateAnchor(parent, text, asciidoctor.Attributes{AttrLink: to},
		map[string]any{"type": "link", "target": html.EscapeString(href)})
	if err != nil {
		return "", err
	}
	return a.Convert()
}

// passthrough is AsciiDoc for finished HTML: an inline passthrough, which
// AsciiDoc substitutions leave alone. Text in list items and table cells
// goes through substitutions, which would read markup in a name (a*b*).
// A '+' would end the passthrough early; as a reference it is still a plus.
func passthrough(html string) string {
	return "+++" + strings.ReplaceAll(html, "+", "&#43;") + "+++"
}
