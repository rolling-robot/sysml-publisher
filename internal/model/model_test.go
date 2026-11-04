package model

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func load(t *testing.T, dir string) (*Model, error) {
	t.Helper()
	return Folder{Dir: filepath.Join("..", "..", "testdata", dir)}.Load(context.Background())
}

// A model of two files converts as one: the usage in top.sysml is typed by
// the definition in lib.sysml, and the definition's representation is there.
func TestLoadCrossFileModel(t *testing.T) {
	m, err := load(t, "crossfile")
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Files) != 2 {
		t.Fatalf("files = %v, want 2", m.Files)
	}
	x, ok := m.Elements["Top__x"]
	if !ok {
		t.Fatal("Top::x missing from the element array")
	}
	if got := x.Refs("type"); len(got) != 1 || got[0] != "Lib__pd" {
		t.Errorf("Top::x type = %v, want [Lib__pd]", got)
	}
	var rep Element
	for _, id := range m.Order {
		if e := m.Elements[id]; e.Type() == "TextualRepresentation" {
			rep = e
		}
	}
	if rep == nil {
		t.Fatal("no TextualRepresentation in the element array")
	}
	if rep.Ref("owner") != "Lib__pd" || rep.Str("language") != "asciidoc" || strings.TrimSpace(rep.Str("body")) != "From lib" {
		t.Errorf("representation = owner %q language %q body %q", rep.Ref("owner"), rep.Str("language"), rep.Str("body"))
	}
	if m.ExperimentalNotice == "" {
		t.Error("expected the experimental notice for api-json")
	}
}

func TestLoadReportsErrorsWithLocation(t *testing.T) {
	_, err := load(t, "syntaxerror")
	var de *DiagnosticsError
	if !errors.As(err, &de) {
		t.Fatalf("err = %v, want *DiagnosticsError", err)
	}
	if !strings.Contains(de.Errors[0], "bad.sysml:2:") {
		t.Errorf("first error %q does not locate bad.sysml line 2", de.Errors[0])
	}
}

func TestLoadEmptyFolder(t *testing.T) {
	_, err := load(t, "empty")
	if !errors.Is(err, ErrNoModelFiles) {
		t.Fatalf("err = %v, want ErrNoModelFiles", err)
	}
}
