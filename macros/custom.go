package macros

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/Open-MBEE/OpenSysML/client/opensysml"
	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/sysml"
)

// Registry is the macros registered on one asciidoctorg instance for one
// rendering, as Register returns it. Inline and Block add macros of the
// caller's own that work like the sysml-* ones (MC-20): they read the same
// model, name elements the same way (MC-15), link through the same Pages,
// and a failing one leaves a placeholder and a warning (MC-19).
type Registry struct {
	ad *asciidoctor.Asciidoctor
	e  *env
}

// Element is an element a macro is converted for or names.
type Element struct {
	ID string
	// QualifiedName is its qualified name, or its positional name (M::@2)
	// when it has no name; the subject Evaluate takes.
	QualifiedName string
}

// Call is one use of a custom macro in a representation converted for an
// element: what the macro's function gets.
type Call struct {
	// Page is the element whose page is converted (MC-2).
	Page Element
	// Target is the target as written, unescaped; "" when there is none.
	Target string
	// Macro is the macro as written (my-mass:engine[]), as warnings name it.
	Macro string

	attrs  map[string]any
	parent asciidoctor.Node
	e      *env
}

// InlineFunc makes the HTML that replaces an inline macro. The HTML is
// written out as is: escape model text (html.EscapeString). An error is the
// macro failing: a placeholder takes its place and a warning is reported
// (MC-19).
type InlineFunc func(c *Call) (string, error)

// BlockFunc makes the block that replaces a block macro, created under
// parent (asciidoctor.CreateOpenBlock, CreateList, ...), or nil to drop the
// macro line. An error is the macro failing, as for InlineFunc.
type BlockFunc func(c *Call, parent asciidoctor.StructuralNode) (asciidoctor.StructuralNode, error)

// customNameRx is what a custom macro may be called.
var customNameRx = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)

// checkName refuses names Asciidoctor cannot read as a macro and the sysml-
// prefix, which is this package's (MC-1): a later version may add a macro
// of that name.
func checkName(name string) error {
	switch {
	case !customNameRx.MatchString(name):
		return fmt.Errorf("macro name %q: use letters, digits, - and _, starting with a letter", name)
	case strings.HasPrefix(strings.ToLower(name), "sysml-"):
		return fmt.Errorf("macro name %q: the sysml- prefix is reserved for this package's macros", name)
	}
	return nil
}

// Inline adds the inline macro name:[<target>][<attributes>], whose target
// is optional and may hold quoted names with spaces, as sysml-name's does.
// positional names the positional attributes, in order (Attr reads them).
// The macro is active once Inline returns.
func (r *Registry) Inline(name string, fn InlineFunc, positional ...string) error {
	if err := checkName(name); err != nil {
		return err
	}
	g := r.ad.Extensions().Group("")
	inline(g, r.e, name, &asciidoctor.MacroOptions{Regexp: optionalTargetRx(name), PositionalAttributes: positional}, nil,
		func(c call[asciidoctor.Node]) (string, error) {
			return fn(r.call(c.page, c.target, c.macro, c.attrs, c.parent))
		})
	if err := g.Register(); err != nil {
		return fmt.Errorf("registering %s: %w", name, err)
	}
	return nil
}

// Block adds the block macro name::[<target>][<attributes>], on a line of
// its own. positional names the positional attributes, in order. The
// macro is active once Block returns.
func (r *Registry) Block(name string, fn BlockFunc, positional ...string) error {
	if err := checkName(name); err != nil {
		return err
	}
	g := r.ad.Extensions().Group("")
	block(g, r.e, name, &asciidoctor.MacroOptions{PositionalAttributes: positional}, nil,
		func(c call[asciidoctor.StructuralNode]) (asciidoctor.StructuralNode, error) {
			return fn(r.call(c.page, c.target, c.macro, c.attrs, c.parent), c.parent)
		})
	if err := g.Register(); err != nil {
		return fmt.Errorf("registering %s: %w", name, err)
	}
	return nil
}

func (r *Registry) call(page element, target, macro string, attrs map[string]any, parent asciidoctor.Node) *Call {
	return &Call{
		Page:   Element{ID: page.id, QualifiedName: page.qualifiedName},
		Target: target,
		Macro:  macro,
		attrs:  attrs,
		parent: parent,
		e:      r.e,
	}
}

// Context is the rendering's context: pass it to model reads, which stop
// when it is cancelled.
func (c *Call) Context() context.Context { return c.e.ctx }

// Model is the model the macros read.
func (c *Call) Model() *sysml.Model { return c.e.m }

// Parent is the node the macro is converted in.
func (c *Call) Parent() asciidoctor.Node { return c.parent }

// Attr is the attribute name as written (named, or positional when Inline
// or Block named its position), trimmed; "" when it is not given.
func (c *Call) Attr(name string) string {
	v, _ := c.attrs[name].(string)
	return strings.TrimSpace(html.UnescapeString(v))
}

// Element is the element the macro is about: the page's element without a
// target, else the element the target names, read as sysml-name reads it
// (MC-8, MC-15), metadata forms ([<element>::]@<metadata>[::<attribute>])
// included.
func (c *Call) Element() (Element, error) {
	if c.Target == "" {
		return c.Page, nil
	}
	id, err := nameTarget(c.Target, element{id: c.Page.ID, qualifiedName: c.Page.QualifiedName}, c.e)
	if err != nil {
		return Element{}, err
	}
	return c.element(id)
}

// Resolve is the element name names, read from the page's element as every
// macro reads a name (MC-15): a qualified name, a name inside the page's
// element or an element around it, a library qualified name, or a name
// only one element has; each segment a name or a short name.
func (c *Call) Resolve(name string) (Element, error) {
	ref, err := c.e.names.resolve(name, element{id: c.Page.ID, qualifiedName: c.Page.QualifiedName}, nil)
	if err != nil {
		return Element{}, err
	}
	return c.element(ref.ID)
}

func (c *Call) element(id string) (Element, error) {
	qn, err := c.e.names.qualifiedName(id)
	if err != nil {
		return Element{}, err
	}
	return Element{ID: id, QualifiedName: qn}, nil
}

// Name is the element's name as sysml-name shows it, sysml.UnnamedName when
// it has none.
func (c *Call) Name(id string) (string, error) { return c.e.names.name(id) }

// Evaluate evaluates expression with element subject as its subject, so a
// feature reads that element's value (sysml-value, MC-14).
func (c *Call) Evaluate(subject Element, expression string) (opensysml.Value, error) {
	return c.e.m.Evaluate(c.e.ctx, subject.QualifiedName, expression)
}

// Link is html linked to the page of element id from the page being
// converted (MC-18): unlinked when id has no page, is the page's own
// element, or html is empty. html must be escaped already.
func (c *Call) Link(id, html string) string { return c.e.linkTo(c.Page.ID, id, html) }

// Warn reports a problem without failing the macro (MC-7): the warning names
// the page and the macro, so message should not.
func (c *Call) Warn(message string) {
	c.e.warn(Warning{Page: c.Page.QualifiedName, Macro: c.Macro, Message: message})
}

// FormatValue is a value as sysml-value shows it (MC-14): a quantity as
// magnitude and unit symbol (5 kg), a string without quotes, an enumeration
// literal as written, NoValue for none and CompositeText for a composite
// value. The text is not escaped.
func FormatValue(v opensysml.Value) string {
	text, kind := formatValue(v)
	switch kind {
	case valueComposite:
		return CompositeText
	case valueNone:
		return NoValue
	}
	return text
}

// Passthrough is finished HTML as AsciiDoc that substitutions leave alone,
// for list items and table cells, whose text AsciiDoc would otherwise read
// as markup.
func Passthrough(html string) string { return passthrough(html) }
