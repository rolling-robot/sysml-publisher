// Package views renders SysML v2 views, for the sysml-view macro
// (requirement MC-21): OpenSysML writes a view in process with
// sysml.Model.RenderView — DOT for a diagram, columns and rows for a table —
// and Graphviz's dot lays the DOT out as SVG. A table needs no program.
//
//	reg.SetViews(&views.InProcess{Model: m}) // reg from macros.Register
package views

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"sync"

	"github.com/Open-MBEE/OpenSysML/client/opensysml"
	"github.com/rolling-robot/sysml-publisher/macros"
	"github.com/rolling-robot/sysml-publisher/sysml"
)

// InProcess renders views with OpenSysML's RenderView, in process, once per
// view: it is a macros.Views. Renderings are kept, failures included, so a
// view shown on several pages is rendered once. Safe for concurrent use.
type InProcess struct {
	// Model is the parsed model whose views these are, open while rendering.
	Model *sysml.Model
	// Dot is Graphviz's dot command, which lays a diagram out; "" runs "dot"
	// from PATH. Only a diagram needs it: a table is written from the rows
	// the rendering gives.
	Dot string

	mu    sync.Mutex
	cache map[string]result
}

var _ macros.Views = (*InProcess)(nil)

// result is one view's rendering, kept in InProcess.cache: the failure too,
// so a view the service refuses is refused once.
type result struct {
	r   *macros.Rendering
	err error
}

// Render renders the view with that qualified name: a diagram as SVG (the
// rendering's DOT form, then dot -Tsvg), else a table, from the rows the
// rendering gives. Its errors say what went wrong, as the service or dot
// reports it.
func (v *InProcess) Render(ctx context.Context, qualifiedName string) (*macros.Rendering, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if res, ok := v.cache[qualifiedName]; ok {
		return res.r, res.err
	}
	r, err := v.render(ctx, qualifiedName)
	if ctx.Err() != nil {
		return nil, ctx.Err() // not kept: an interrupted rendering says nothing about the view
	}
	if v.cache == nil {
		v.cache = map[string]result{}
	}
	v.cache[qualifiedName] = result{r, err}
	return r, err
}

// render asks for the view as DOT, and for its data when it has no diagram
// form: a table is refused a diagram form, and only the data says that the
// view is one.
func (v *InProcess) render(ctx context.Context, qn string) (*macros.Rendering, error) {
	if v.Model == nil {
		return nil, errors.New("no model to render views from")
	}
	rv, err := v.Model.RenderView(ctx, qn, opensysml.WithForm(opensysml.ViewFormDOT))
	if err == nil {
		svg, err := v.layout(ctx, []byte(rv.Text))
		if err != nil {
			return nil, err
		}
		return &macros.Rendering{SVG: svg}, nil
	}
	data, dataErr := v.Model.RenderView(ctx, qn)
	if dataErr != nil || len(data.Columns) == 0 {
		return nil, viewMessage(err, qn) // the diagram form's refusal is the answer
	}
	return &macros.Rendering{Table: tableRows(data)}, nil
}

// tableRows is a tabular rendering as a table: its columns as the header
// row, then each row's cells. The cells are copied: the table outlives the
// rendering.
func tableRows(data *opensysml.RenderedView) [][]string {
	rows := make([][]string, 0, len(data.Rows)+1)
	rows = append(rows, append([]string(nil), data.Columns...))
	for _, row := range data.Rows {
		rows = append(rows, append([]string(nil), row.Cells...))
	}
	return rows
}

// layout lays DOT out as SVG with dot -Tsvg.
func (v *InProcess) layout(ctx context.Context, dot []byte) ([]byte, error) {
	return run(exec.CommandContext(ctx, or(v.Dot, "dot"), "-Tsvg"), "dot", dot)
}

// viewMessage is a rendering the service refused, in the macro's words: the
// service's own message, naming the view when the service's does not.
func viewMessage(err error, qualifiedName string) error {
	var status *opensysml.StatusError
	if !errors.As(err, &status) {
		return err
	}
	msg := status.Message
	if !strings.HasPrefix(msg, qualifiedName) {
		msg = qualifiedName + ": " + msg
	}
	return errors.New(msg) // the macro reports the message alone: no cause to keep
}
