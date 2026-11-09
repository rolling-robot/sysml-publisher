// Package model loads a SysML v2 model and reads it the way the SysML v2
// API does (DS-4): element queries (Query), reads of one element's
// metamodel properties (Props, cached) and expression evaluation
// (Evaluate). Nothing exports the whole model.
package model

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Open-MBEE/OpenSysML/client/opensysml"
)

// Model is a loaded model.
type Model struct {
	// Files are the model files parsed, in the order given to the parser.
	Files []string
	// ElementCount is the number of elements in the model.
	ElementCount int
	// Tops are the top-level elements, in file order: the members of each
	// file's root namespace.
	Tops []Ref
	// Warnings are the parser's diagnostics below error severity, each as
	// "file:line:col: severity: message".
	Warnings []string

	// The engine the model was parsed by, kept open for Evaluate until Close.
	client opensysml.Client
	parsed *opensysml.Model
	// cache holds the properties Props has read, by element id and by every
	// other name an element was asked by.
	cache map[string]*elementCache
	// own are the ids of the model's own elements, by readIndex.
	own   map[string]bool
	stats Stats
}

// Stats count how the model was read: the cost of reading it through
// queries and element reads rather than one export (DS-4).
type Stats struct {
	Queries int // Query calls
	Reads   int // ElementProperties calls: elements, or properties, not cached yet
	Hits    int // Props calls answered from the cache
}

// Stats are the reads made so far.
func (m *Model) Stats() Stats { return m.stats }

// Evaluate evaluates a SysML expression with the element named subject (by
// qualified name) as its subject, so a feature reads that element's value:
// its own, a redefinition's, or one inherited from its types.
func (m *Model) Evaluate(ctx context.Context, subject, expression string) (opensysml.Value, error) {
	if m.client == nil {
		return nil, errors.New("this model cannot evaluate expressions")
	}
	return m.client.Evaluate(ctx, m.parsed, expression, opensysml.WithSubject(subject))
}

// Close releases the engine. The model's elements stay readable.
func (m *Model) Close() error {
	if m.client == nil {
		return nil
	}
	err := m.client.Close()
	m.client, m.parsed = nil, nil
	return err
}

// Source is where a model comes from. The MVP reads a folder of files; a
// SysML v2 API service is a later Source (requirements DS-1, DS-3).
type Source interface {
	Load(ctx context.Context) (*Model, error)
}

// ErrNoModelFiles is returned when the model folder holds no model files (ER-3).
var ErrNoModelFiles = errors.New("no model files")

// DiagnosticsError reports a model with errors (ER-2).
type DiagnosticsError struct {
	// Errors are the error diagnostics, each as "file:line:col: error: message".
	Errors []string
}

func (e *DiagnosticsError) Error() string {
	return fmt.Sprintf("model has %d error(s):\n%s", len(e.Errors), strings.Join(e.Errors, "\n"))
}

// Folder is a folder of *.sysml and *.kerml files, searched recursively and
// parsed in-process as one model.
type Folder struct {
	Dir string
	// Logger receives debug records (stage durations); nil uses slog.Default.
	Logger *slog.Logger
}

func (f Folder) logger() *slog.Logger {
	if f.Logger != nil {
		return f.Logger
	}
	return slog.Default()
}

// Load parses every model file under the folder as one model and queries it
// once for its elements (readIndex).
func (f Folder) Load(ctx context.Context) (*Model, error) {
	files, err := modelFiles(f.Dir)
	if err != nil {
		return nil, err
	}
	client, err := opensysml.New()
	if err != nil {
		return nil, fmt.Errorf("starting the SysML engine: %w", err)
	}
	// The engine stays open for Evaluate when loading succeeds; the caller
	// closes it with Model.Close.
	ok := false
	defer func() {
		if !ok {
			client.Close()
		}
	}()

	start := time.Now()
	parsed, err := client.ParseFiles(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", f.Dir, err)
	}
	f.logger().DebugContext(ctx, "model parsed", "files", len(files), "diagnostics", len(parsed.Diagnostics), "duration", time.Since(start))
	m := &Model{Files: files, client: client, parsed: parsed}
	var errs []string
	for _, d := range parsed.Diagnostics {
		if d.Severity == opensysml.SeverityError {
			errs = append(errs, d.String())
		} else {
			m.Warnings = append(m.Warnings, d.String())
		}
	}
	if len(errs) > 0 {
		return nil, &DiagnosticsError{Errors: errs}
	}

	start = time.Now()
	if err := m.readIndex(ctx); err != nil {
		return nil, err
	}
	f.logger().DebugContext(ctx, "model queried", "elements", m.ElementCount, "top-level", len(m.Tops), "duration", time.Since(start))

	ok = true
	return m, nil
}

// readIndex queries every element once: the count, the ids of the model's
// own elements, and those without an owner, which are the top-level ones,
// in file order. The query names elements by qualified or positional name,
// so each is read once (its name, a cheap property) for its id.
func (m *Model) readIndex(ctx context.Context) error {
	all, err := m.Query(ctx, opensysml.Query{Select: []string{"owner"}})
	if err != nil {
		return err
	}
	m.ElementCount = len(all)
	m.own = make(map[string]bool, len(all))
	for _, e := range all {
		p, err := m.Props(ctx, e.ID, "name")
		if err != nil {
			return err
		}
		m.own[p.Element.ID] = true
		if e.Properties["owner"] == "" {
			m.Tops = append(m.Tops, p.Element)
		}
	}
	return nil
}

// Contains reports whether id is an element of the model itself, not of
// the standard library it refers to.
func (m *Model) Contains(id string) bool { return m.own[id] }

// modelFiles lists the model files under dir, sorted so every run parses them
// in the same order.
func modelFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Hidden files and folders are not the model: editor lock files
		// (Emacs' .#aircraft.sysml, a dangling link), .git, and the like.
		if path != dir && strings.HasPrefix(d.Name(), ".") {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		switch filepath.Ext(path) {
		case ".sysml", ".kerml":
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("reading model folder: %w", err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("%w in %s", ErrNoModelFiles, dir)
	}
	sort.Strings(files)
	return files, nil
}
