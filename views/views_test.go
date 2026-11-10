package views_test

import (
	"context"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/rolling-robot/sysml-publisher/views"
)

// These tests run OpenSysML's sysml and Graphviz's dot: they are skipped
// where either is not installed.
func needPrograms(t *testing.T) {
	t.Helper()
	for _, p := range []string{"sysml", "dot"} {
		if _, err := exec.LookPath(p); err != nil {
			t.Skipf("%s is not installed: %v", p, err)
		}
	}
}

var fixture = []string{"../testdata/views/views.sysml"}

func TestDiagram(t *testing.T) {
	needPrograms(t)
	r, err := (&views.CLI{Files: fixture}).Render(t.Context(), "V::sys::interfaces")
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

func TestTable(t *testing.T) {
	needPrograms(t)
	r, err := (&views.CLI{Files: fixture}).Render(t.Context(), "V::partsTable")
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

// A view sysml cannot render fails with sysml's message, for a diagram's
// DOT, not the CSV tried after it.
func TestRefused(t *testing.T) {
	needPrograms(t)
	_, err := (&views.CLI{Files: fixture}).Render(t.Context(), "V::sys")
	if err == nil || err.Error() != "sysml: V::sys: not a view" {
		t.Errorf("error %v, want sysml: V::sys: not a view", err)
	}
}

// A program that cannot be run is named.
func TestMissingProgram(t *testing.T) {
	needPrograms(t)
	cases := []struct {
		name, sysml, dot string
		wantPrefix       string
	}{
		{"sysml", "/nonexistent/sysml", "", "cannot run sysml: "},
		{"dot", "", "/nonexistent/dot", "cannot run dot: "},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cli := &views.CLI{Files: fixture, Sysml: c.sysml, Dot: c.dot}
			_, err := cli.Render(t.Context(), "V::sys::interfaces")
			if err == nil || !strings.HasPrefix(err.Error(), c.wantPrefix) {
				t.Errorf("error %v, want %s...", err, c.wantPrefix)
			}
		})
	}
}

// A view is rendered once, however often it is asked for; a failure too.
func TestRenderedOnce(t *testing.T) {
	needPrograms(t)
	cli := &views.CLI{Files: fixture}
	first, err := cli.Render(t.Context(), "V::partsTable")
	if err != nil {
		t.Fatal(err)
	}
	cli.Sysml = "/nonexistent/sysml" // would fail if run again
	again, err := cli.Render(t.Context(), "V::partsTable")
	if err != nil || again != first {
		t.Errorf("second rendering %p, %v; want the first, %p", again, err, first)
	}
}

// An interrupted rendering returns the context's error and is not kept.
func TestCancelled(t *testing.T) {
	needPrograms(t)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	cli := &views.CLI{Files: fixture}
	if _, err := cli.Render(ctx, "V::partsTable"); err != context.Canceled {
		t.Fatalf("error %v, want context.Canceled", err)
	}
	if _, err := cli.Render(t.Context(), "V::partsTable"); err != nil {
		t.Errorf("after the interruption: %v", err)
	}
}
