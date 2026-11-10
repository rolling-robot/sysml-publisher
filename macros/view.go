package macros

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"html"
	"regexp"
	"strings"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/sysml"
)

// View is the block macro sysml-view::[<view>][], replaced by a view of the
// model as OpenSysML renders it (MC-21): a diagram as a picture, a table
// view as a table. Without a target it shows the page's element, which
// must then be a view; a target names the view as sysml-name does (MC-15).
// Views are rendered by the host's Views (Registry.SetViews).
const View = "sysml-view"

// ViewRole is the role of the block a view is put in, for styling.
const ViewRole = "sysml-view"

// AttrView is the attribute of the block a view is put in: the view's
// qualified name. html5 writes a diagram's block as an open block holding
// the SVG; a converter for another output can tell it by AttrView and
// write the picture its own way (Confluence's <ac:image>).
const AttrView = "sysml-view"

// Views renders the views sysml-view shows (MC-21).
type Views interface {
	// Render renders the view with that qualified name. Its error is the
	// macro's message (MC-19): it says what went wrong, not which macro.
	Render(ctx context.Context, qualifiedName string) (*Rendering, error)
}

// Rendering is a rendered view: a diagram or a table.
type Rendering struct {
	// SVG is a diagram, as a standalone SVG document; nil for a table. It
	// is put in the page as it is, so it must come from a trusted
	// renderer.
	SVG []byte
	// Table is a table view's cells, row by row, the header row first.
	Table [][]string
}

// errNoViews is the message of a sysml-view used where the host renders no
// views.
var errNoViews = errors.New("views are not rendered here (no view renderer)")

// SetViews makes v render the views of sysml-view (MC-21). Without it, the
// macro fails (MC-19).
func (r *Registry) SetViews(v Views) { r.e.views = v }

// viewMetaclass is the metaclass of the elements sysml-view shows.
const viewMetaclass = "ViewUsage"

func registerView(g *asciidoctor.ExtensionGroup, e *env) {
	block(g, e, View, nil, nil,
		func(c call[asciidoctor.StructuralNode]) (asciidoctor.StructuralNode, error) {
			v, err := viewTarget(c.target, c.page, e)
			if err != nil {
				return nil, err
			}
			if e.views == nil {
				return nil, errNoViews
			}
			qn, err := e.names.qualifiedName(v.ID)
			if err != nil {
				return nil, err
			}
			r, err := e.views.Render(e.ctx, qn)
			if err != nil {
				return nil, err
			}
			var b asciidoctor.StructuralNode
			if r.SVG == nil {
				b, err = viewTable(c.parent, r.Table)
			} else {
				b, err = viewDiagram(c.parent, r.SVG, qn)
			}
			if err != nil {
				return nil, err
			}
			// The parser keeps a block macro's block as made, without the
			// macro's attributes or the title written above it (.Title).
			b.AddRole(ViewRole)
			b.SetAttr(AttrView, qn, true)
			if title, ok := c.attrs["title"].(string); ok {
				b.SetTitle(title)
			}
			return b, nil
		})
}

// viewDiagram is a diagram as an open block holding the SVG.
func viewDiagram(parent asciidoctor.StructuralNode, svg []byte, qn string) (asciidoctor.StructuralNode, error) {
	b, err := asciidoctor.CreateOpenBlock(parent, nil, nil, nil)
	if err != nil {
		return nil, err
	}
	pass, err := asciidoctor.CreatePassBlock(b, inlineSVG(svg, qn), nil, nil)
	if err != nil {
		return nil, err
	}
	b.Append(pass)
	return b, nil
}

// viewTarget is the view a sysml-view target names (MC-15), or the page's
// element without a target; an element that is no view fails the macro.
func viewTarget(target string, n element, e *env) (sysml.Ref, error) {
	if target == "" {
		p, err := e.m.Props(e.ctx, n.id, "name")
		if err != nil {
			return sysml.Ref{}, err
		}
		if p.Element.Metaclass != viewMetaclass {
			return sysml.Ref{}, fmt.Errorf("the page's element %s is a %s, not a view; name a view", n.qualifiedName, p.Element.Metaclass)
		}
		return p.Element, nil
	}
	isView := func(r sysml.Ref) bool { return r.Metaclass == viewMetaclass }
	v, err := e.names.resolve(target, n, isView)
	if !errors.Is(err, errNoSuchName) {
		return v, err
	}
	// No view of that name: say what the name is, if anything.
	other, err2 := e.names.resolve(target, n, nil)
	if err2 != nil {
		return sysml.Ref{}, err
	}
	qn, err2 := e.names.qualifiedName(other.ID)
	if err2 != nil {
		return sysml.Ref{}, err2
	}
	return sysml.Ref{}, fmt.Errorf("%s is a %s, not a view", qn, other.Metaclass)
}

// viewTable is a table view as an AsciiDoc table: the first row is the
// header. Cells are model text, escaped and passed through.
func viewTable(parent asciidoctor.StructuralNode, rows [][]string) (asciidoctor.StructuralNode, error) {
	if len(rows) == 0 {
		return nil, errors.New("the table view has no columns")
	}
	tbl, err := asciidoctor.CreateTable(parent, nil)
	if err != nil {
		return nil, err
	}
	cols := make([]*asciidoctor.Column, len(rows[0]))
	for i := range cols {
		if cols[i], err = asciidoctor.CreateTableColumn(tbl, i, nil); err != nil {
			return nil, err
		}
		tbl.AddColumn(cols[i])
	}
	for i, row := range rows {
		section := asciidoctor.TableBody
		if i == 0 {
			section = asciidoctor.TableHeader
		}
		cells := make([]*asciidoctor.Cell, len(cols))
		for j, col := range cols {
			text := ""
			if j < len(row) {
				text = row[j]
			}
			if cells[j], err = asciidoctor.CreateTableCell(col, passthrough(html.EscapeString(text)), nil); err != nil {
				return nil, err
			}
		}
		tbl.AddRow(section, cells...)
	}
	tbl.AssignColumnWidths()
	return tbl, nil
}

// svgRefRx finds the ids an SVG declares and its references to them.
var svgRefRx = regexp.MustCompile(`(\sid="|[\s:]href="#|url\(#)`)

// inlineSVG is an SVG document as an element to put in HTML: without what
// comes before <svg> (the XML declaration, the doctype, comments), and with
// its ids prefixed per view, so that the ids of two views on one page
// (Graphviz numbers them node1, edge1, ... in every picture) do not clash.
func inlineSVG(svg []byte, qualifiedName string) string {
	s := string(svg)
	if i := strings.Index(s, "<svg"); i > 0 {
		s = s[i:]
	}
	h := fnv.New32a()
	h.Write([]byte(qualifiedName))
	prefix := fmt.Sprintf("sysml-view-%08x-", h.Sum32())
	return svgRefRx.ReplaceAllString(strings.TrimSpace(s), "${1}"+prefix)
}
