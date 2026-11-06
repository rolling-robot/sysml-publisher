// Package macros holds the tool's AsciiDoc macros (requirements MC-1..14).
// Each macro lives in its own file; Register adds them all to an asciidoctorg
// instance.
package macros

import (
	"context"
	"fmt"
	"html"
	"strings"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
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

// Register adds every macro to ad. The macros read the page tree t, report
// warnings to warn, and stop evaluating when ctx is cancelled. ctx is the
// context of the rendering ad is registered for; register again for another.
func Register(ctx context.Context, ad *asciidoctor.Asciidoctor, t *tree.Tree, warn Warn) error {
	if warn == nil {
		warn = func(Warning) {}
	}
	g := ad.Extensions().Group("sysml")
	registerName(g, t)
	registerRep(g, t)
	registerQualifiedName(g, t)
	registerShortName(g, t)
	registerMembersList(g, t)
	registerDoc(g, t)
	registerMetadataTable(g, t)
	registerValue(ctx, g, t, warn)
	if err := g.Register(); err != nil {
		return fmt.Errorf("registering the sysml macros: %w", err)
	}
	return nil
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

// registerText registers an inline macro name:[] replaced by the text that
// text gives for the page's element.
func registerText(g *asciidoctor.ExtensionGroup, t *tree.Tree, name string, text func(*tree.Node) string) {
	g.InlineMacro(name, asciidoctor.InlineMacroProcessorFunc(
		func(parent asciidoctor.Node, _ string, _ map[string]any) (*asciidoctor.PhraseNode, error) {
			n, err := pageElement(parent, t)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", name, err)
			}
			// An unquoted phrase is written out verbatim, so escape the text:
			// SysML names may contain <, > and &.
			return asciidoctor.CreatePhraseNode(parent, "quoted", html.EscapeString(text(n)), nil, nil)
		}), &asciidoctor.MacroOptions{Format: "short"})
}

// passthrough is AsciiDoc for finished HTML: an inline passthrough, which
// AsciiDoc substitutions leave alone. Text in list items and table cells
// goes through substitutions, which would read markup in a name (a*b*).
// A '+' would end the passthrough early; as a reference it is still a plus.
func passthrough(html string) string {
	return "+++" + strings.ReplaceAll(html, "+", "&#43;") + "+++"
}
