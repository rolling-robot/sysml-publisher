package macros

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"html"
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

// AttrView is the attribute of the block a view is put in (MC-21): the
// view's qualified name. As links are anchor nodes (AttrLink), a view is an
// ordinary node the document's converter writes: a diagram an image block,
// whose target is the SVG as a data: URI, a table view a table. html5
// writes the diagram as <img src="data:...">; a converter for another
// output can tell the block by AttrView and write the picture its own way
// (Confluence's <ac:image> with an attachment), taking the SVG from
// ViewSVG.
const AttrView = "sysml-view"

// Views renders the views sysml-view shows (MC-21).
type Views interface {
	// Render renders the view with that qualified name. Its error is the
	// macro's message (MC-19): it says what went wrong, not which macro.
	Render(ctx context.Context, qualifiedName string) (*Rendering, error)
}

// Rendering is a rendered view: a diagram or a table.
type Rendering struct {
	// SVG is a diagram, as a standalone SVG document; nil for a table.
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
				var name string
				if name, err = e.names.name(v.ID); err == nil {
					b, err = viewDiagram(c.parent, r.SVG, name)
				}
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

// svgDataURI is how a diagram's SVG is put in its image block's target.
const svgDataURI = "data:image/svg+xml;base64,"

// viewDiagram is a diagram as an image block: its target the SVG as a
// data: URI, so the picture needs no file, its alternative text the view's
// name.
func viewDiagram(parent asciidoctor.StructuralNode, svg []byte, name string) (asciidoctor.StructuralNode, error) {
	return asciidoctor.CreateImageBlock(parent, asciidoctor.Attributes{
		"target": svgDataURI + base64.StdEncoding.EncodeToString(svg),
		"alt":    name,
	}, nil)
}

// ViewSVG is the SVG of a diagram sysml-view put in, for a converter that
// writes it elsewhere than in the page (an attachment): n is the image
// block, carrying AttrView. ok is false for any other node.
func ViewSVG(n asciidoctor.Node) (svg []byte, ok bool) {
	if _, isView := n.Attr(AttrView).(string); !isView {
		return nil, false
	}
	target, _ := n.Attr("target").(string)
	data, isSVG := strings.CutPrefix(target, svgDataURI)
	if !isSVG {
		return nil, false
	}
	svg, err := base64.StdEncoding.DecodeString(data)
	return svg, err == nil
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
