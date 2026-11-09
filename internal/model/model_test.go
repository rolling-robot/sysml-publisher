package model

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func load(t *testing.T, dir string) (*Model, error) {
	t.Helper()
	return Folder{Dir: filepath.Join("..", "..", "testdata", dir)}.Load(context.Background())
}

// A model of two files is read as one: the usage in top.sysml is typed by
// the definition in lib.sysml. Load counts the elements and finds the
// top-level ones in file order.
func TestLoadCrossFileModel(t *testing.T) {
	m, err := load(t, "crossfile")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	if len(m.Files) != 2 {
		t.Fatalf("files = %v, want 2", m.Files)
	}
	types, err := m.Types(t.Context(), "Top::x")
	if err != nil {
		t.Fatal(err)
	}
	if len(types) != 1 || types[0].ID != "Lib__pd" {
		t.Errorf("Top::x types = %+v, want [Lib::pd]", types)
	}
	var tops []string
	for _, r := range m.Tops {
		tops = append(tops, r.ID)
	}
	if got := tops; !slices.Equal(got, []string{"Lib", "Top"}) {
		t.Errorf("tops = %v, want [Lib Top]", got)
	}
	// What a query returns: Lib, Lib::pd, its representation, Top and
	// Top::x; relationships (the import, the memberships, the typing) and
	// the implicit multiplicity are not among them.
	if m.ElementCount != 5 {
		t.Errorf("element count = %d, want 5", m.ElementCount)
	}
	if !m.Contains("Lib__pd") {
		t.Error("Contains(Lib__pd) = false, want true")
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

// Hidden files and folders are skipped: an Emacs lock file is a dangling
// link named .#<file>.sysml, and a hidden folder (.git) is not the model.
// The model folder itself may be "." and is still read.
func TestLoadSkipsHiddenFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "m.sysml"), []byte("package P;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("user@host.1234:5678", filepath.Join(dir, ".#m.sysml")); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "broken.sysml"), []byte("not sysml {"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ name, dir string }{{"absolute", dir}, {"dot", "."}} {
		t.Run(c.name, func(t *testing.T) {
			if c.dir == "." {
				t.Chdir(dir)
			}
			m, err := Folder{Dir: c.dir}.Load(t.Context())
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { m.Close() })
			if len(m.Files) != 1 || filepath.Base(m.Files[0]) != "m.sysml" {
				t.Errorf("files = %v, want only m.sysml", m.Files)
			}
		})
	}
}
