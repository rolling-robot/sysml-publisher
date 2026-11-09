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
// a value it cannot show. Its parts are kept apart so the receiver can tell
// repeats without parsing the text (MC-7: one warning per page and macro).
type Warning struct {
	// Page is the page the macro was converted for.
	Page *tree.Node
	// Macro is the macro as written: sysml-value:position[].
	Macro string
	// Message says what is wrong, without the page or the macro.
	Message string
}

// String is the warning as reported: "<page>: <macro>: <message>".
func (w Warning) String() string {
	return w.Page.QualifiedName + ": " + w.Macro + ": " + w.Message
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

// registerText registers the inline macro name:[<element>], replaced by text
// about an element: ofPage gives it for the page's element when there is no
// target; ofElement for the element a target names, read by nameTarget
// (MC-15, MC-8).
func registerText(g *asciidoctor.ExtensionGroup, e *env, name string, ofPage func(*tree.Node) string, ofElement func(id string) (string, error)) {
	g.InlineMacro(name, asciidoctor.InlineMacroProcessorFunc(
		func(parent asciidoctor.Node, target string, _ map[string]any) (*asciidoctor.PhraseNode, error) {
			n, err := pageElement(parent, e.t)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			// Inline macros run after special characters are escaped, so
			// M::'a<b' arrives as M::'a&lt;b'.
			target = html.UnescapeString(target)
			text := ofPage(n)
			if target != "" {
				id, err := nameTarget(target, n, e)
				if err == nil {
					text, err = ofElement(id)
				}
				if err != nil {
					return nil, fmt.Errorf("%s:%s[] on the page of %s: %w", name, target, n.QualifiedName, err)
				}
			}
			// An unquoted phrase is written out verbatim, so escape the text:
			// SysML names may contain <, > and &.
			return asciidoctor.CreatePhraseNode(parent, "quoted", html.EscapeString(text), nil, nil)
		}), &asciidoctor.MacroOptions{Regexp: optionalTargetRx(name)})
}

// passthrough is AsciiDoc for finished HTML: an inline passthrough, which
// AsciiDoc substitutions leave alone. Text in list items and table cells
// goes through substitutions, which would read markup in a name (a*b*).
// A '+' would end the passthrough early; as a reference it is still a plus.
func passthrough(html string) string {
	return "+++" + strings.ReplaceAll(html, "+", "&#43;") + "+++"
}
