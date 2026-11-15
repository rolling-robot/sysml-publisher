package views_test

import (
	"context"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/Open-MBEE/OpenSysML/client/opensysml"
	"github.com/rolling-robot/sysml-publisher/sysml"
	"github.com/rolling-robot/sysml-publisher/views"
)

// These tests render testdata/views in process; only the diagram runs
// Graphviz's dot, and a test that lays one out is skipped where it is not
// installed.
func needDot(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("dot"); err != nil {
		t.Skipf("dot is not installed: %v", err)
	}
}

// loadViews parses the views fixture and opens it: the model sysml-view
// renders its views from.
func loadViews(t *testing.T) *sysml.Model {
	t.Helper()
	client, err := opensysml.New()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := client.ParseFiles(t.Context(), []string{"../testdata/views/views.sysml"})
	if err != nil {
		client.Close()
		t.Fatal(err)
	}
	m, err := sysml.Open(t.Context(), client, parsed)
	if err != nil {
		client.Close() // Open hands the client back when it fails
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	return m
}

func TestInProcessDiagram(t *testing.T) {
	needDot(t)
	r, err := (&views.InProcess{Model: loadViews(t)}).Render(t.Context(), "V::sys::interfaces")
	if err != nil {
		t.Fatal(err)
	}
	svg := string(r.SVG)
	if !strings.Contains(svg, "<svg") || !strings.Contains(svg, "<title>V::sys::interfaces</title>") {
		t.Errorf("not the view's SVG: %.300q", svg)
	}
	if r.Table != nil {
		t.Errorf("a diagram with a table %q", r.Table)
	}
}

// A table view's rows: its header, then one row per member of sys.
func TestInProcessTable(t *testing.T) {
	r, err := (&views.InProcess{Model: loadViews(t)}).Render(t.Context(), "V::partsTable")
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"Element", "Kind", "Type", "Declared in"},
		{"V::sys::x", "part", "Box", ""},
		{"V::sys::y", "part", "Box", ""},
		{"V::sys::i", "interface", "", ""},
		{"V::sys::interfaces", "view", "", ""},
		{"V::sys::unrendered", "view", "", ""},
	}
	if !reflect.DeepEqual(r.Table, want) {
		t.Errorf("table\n got %q\nwant %q", r.Table, want)
	}
	if r.SVG != nil {
		t.Errorf("a table with an SVG")
	}
}

// A view the service refuses fails with its message, naming the view.
func TestInProcessRefused(t *testing.T) {
	_, err := (&views.InProcess{Model: loadViews(t)}).Render(t.Context(), "V::sys")
	if err == nil || err.Error() != "V::sys: not a view" {
		t.Errorf("error %v, want V::sys: not a view", err)
	}
}

// Only a diagram needs a program: with no dot to run, a table still renders
// and a diagram is named as failing to run.
func TestInProcessNeedsNoProgramForATable(t *testing.T) {
	v := &views.InProcess{Model: loadViews(t), Dot: "/nonexistent/dot"}
	r, err := v.Render(t.Context(), "V::partsTable")
	if err != nil {
		t.Fatalf("a table needs no program: %v", err)
	}
	if len(r.Table) != 6 {
		t.Errorf("table with %d rows, want 6: %q", len(r.Table), r.Table)
	}
	_, err = v.Render(t.Context(), "V::sys::interfaces")
	if err == nil || !strings.HasPrefix(err.Error(), "cannot run dot: ") {
		t.Errorf("error %v, want cannot run dot: ...", err)
	}
}

// A view is rendered once, however often it is asked for; a failure too.
func TestInProcessRenderedOnce(t *testing.T) {
	m := loadViews(t)
	v := &views.InProcess{Model: m}
	first, err := v.Render(t.Context(), "V::partsTable")
	if err != nil {
		t.Fatal(err)
	}
	if err := m.Close(); err != nil { // would fail if rendered again
		t.Fatal(err)
	}
	again, err := v.Render(t.Context(), "V::partsTable")
	if err != nil || again != first {
		t.Errorf("second rendering %p, %v; want the first, %p", again, err, first)
	}
}

// An interrupted rendering returns the context's error and is not kept.
func TestInProcessCancelled(t *testing.T) {
	v := &views.InProcess{Model: loadViews(t)}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := v.Render(ctx, "V::partsTable"); err != context.Canceled {
		t.Fatalf("error %v, want context.Canceled", err)
	}
	if _, err := v.Render(t.Context(), "V::partsTable"); err != nil {
		t.Errorf("after the interruption: %v", err)
	}
}

// Without a model there is nothing to render: it is the macro's warning, not
// a panic.
func TestInProcessWithoutAModel(t *testing.T) {
	_, err := (&views.InProcess{}).Render(t.Context(), "V::sys")
	if err == nil || err.Error() != "no model to render views from" {
		t.Errorf("error %v, want no model to render views from", err)
	}
}
