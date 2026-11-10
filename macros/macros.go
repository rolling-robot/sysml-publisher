// Package macros holds the tool's AsciiDoc macros (requirements MC-1..14).
// Each macro lives in its own file; Register adds them all to an asciidoctorg
// instance.
package macros

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/model"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

// AttrElement is the document attribute naming the page's element by @id,
// set by the renderer on every representation it converts (MC-5).
const AttrElement = "sysml-element"

// Attributes are the document attributes a representation is converted with
// on the page of n.
func Attributes(n *tree.Node) asciidoctor.Attributes {
	return asciidoctor.Attributes{}.Set(AttrElement, n.ID)
}

// Warning is a problem a macro reports while still producing output, such as
// a value it cannot show, or a macro that failed and left a placeholder
// (MC-19). Its parts are kept apart so the receiver can tell
// repeats without parsing the text (MC-7: one warning per page and macro).
type Warning struct {
	// Page is the page the macro was converted for.
	Page *tree.Node
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
	s := w.Page.QualifiedName + ": " + w.Macro + ": " + w.Message
	if w.Location != "" {
		s = w.Location + ": " + s
	}
	return s
}

// Warn receives a warning from a macro. A nil Warn drops warnings.
type Warn func(Warning)

// Register adds every macro to ad. The macros read the page tree t and,
// through it, the model (DS-4); they report warnings to warn and stop
// reading and evaluating when ctx is cancelled. ctx is the context of the
// rendering ad is registered for; register again for another.
func Register(ctx context.Context, ad *asciidoctor.Asciidoctor, t *tree.Tree, warn Warn) error {
	if warn == nil {
		warn = func(Warning) {}
	}
	e := newEnv(ctx, t, warn)
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
		return fmt.Errorf("registering the sysml macros: %w", err)
	}
	return nil
}

// env is what every macro of one rendering shares: the rendering's
// context, the page tree, the model, and the lookups built on it, so a
// name or a metadata def read by one macro is not read again by the next.
type env struct {
	ctx   context.Context
	t     *tree.Tree
	m     *model.Model
	warn  Warn
	names *nameIndex
	md    *metadataIndex
	// kindIndex is made on first use, by kinds.
	kindIndex kindIndex
}

func newEnv(ctx context.Context, t *tree.Tree, warn Warn) *env {
	names := newNameIndex(ctx, t)
	return &env{
		ctx:   ctx,
		t:     t,
		m:     t.Model,
		warn:  warn,
		names: names,
		md:    &metadataIndex{ctx: ctx, m: t.Model, names: names},
	}
}

// pageElement is the element whose page is being rendered (MC-2).
func pageElement(parent asciidoctor.Node, t *tree.Tree) (*tree.Node, error) {
	id, _ := parent.Document().Attr(AttrElement).(string)
	if id == "" {
		return nil, fmt.Errorf("attribute %s is not set", AttrElement)
	}
	n := t.ByID[id]
	if n == nil {
		return nil, fmt.Errorf("attribute %s names %q, which has no page", AttrElement, id)
	}
	return n, nil
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
	page *tree.Node
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
			n, err := pageElement(parent, e.t)
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
			n, err := pageElement(parent, e.t)
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
func (e *env) fail(n *tree.Node, macro string, err error) error {
	if e.ctx.Err() != nil {
		return err
	}
	e.warn(Warning{Page: n, Macro: macro, Message: err.Error()})
	return nil
}

// failedText is the escaped text of a failing macro's placeholder: the
// macro as written and what went wrong.
func failedText(macro string, err error) string {
	return html.EscapeString(macro + ": " + err.Error())
}

// registerText registers the inline macro name:[<element>], replaced by text
// about an element: ofPage gives it for the page's element when there is no
// target; ofElement for the element a target names, read by nameTarget
// (MC-15, MC-8).
func registerText(g *asciidoctor.ExtensionGroup, e *env, name string, ofPage func(*tree.Node) string, ofElement func(id string) (string, error)) {
	inline(g, e, name, &asciidoctor.MacroOptions{Regexp: optionalTargetRx(name)}, nil,
		func(c call[asciidoctor.Node]) (string, error) {
			text, id := ofPage(c.page), c.page.ID
			if c.target != "" {
				var err error
				if id, err = nameTarget(c.target, c.page, e); err != nil {
					return "", err
				}
				if text, err = ofElement(id); err != nil {
					return "", err
				}
			}
			// An unquoted phrase is written out verbatim, so escape the text:
			// SysML names may contain <, > and &.
			return linkTo(c.page, e.t.ByID[id], html.EscapeString(text)), nil
		})
}

// linkTo is the HTML text linked, on the page of from, to the page to
// (MC-18): unlinked when to is nil (the element has no page), when to is
// from itself, or when the text is empty, which leaves nothing to click.
func linkTo(from, to *tree.Node, text string) string {
	if to == nil || to == from || text == "" {
		return text
	}
	return `<a href="` + tree.Href(from, to) + `">` + text + `</a>`
}

// passthrough is AsciiDoc for finished HTML: an inline passthrough, which
// AsciiDoc substitutions leave alone. Text in list items and table cells
// goes through substitutions, which would read markup in a name (a*b*).
// A '+' would end the passthrough early; as a reference it is still a plus.
func passthrough(html string) string {
	return "+++" + strings.ReplaceAll(html, "+", "&#43;") + "+++"
}
