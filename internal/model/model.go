// Package model loads a SysML v2 model and exposes it as the SysML v2 API
// element array (the "api-json" form), which the rest of the publisher reads.
package model

import (
	"context"
	"encoding/json"
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

// Element is one element of the SysML v2 API element array: its metamodel
// properties keyed by name, references written as {"@id": ...}.
type Element map[string]any

// ID is the element's @id.
func (e Element) ID() string { return e.Str("@id") }

// Type is the element's metaclass, its @type.
func (e Element) Type() string { return e.Str("@type") }

// Str is a string-valued property, "" when absent or null.
func (e Element) Str(key string) string {
	s, _ := e[key].(string)
	return s
}

// Ref is the id a single-valued reference property names, "" when absent.
func (e Element) Ref(key string) string { return refID(e[key]) }

// Refs are the ids a reference property names, in order. A single-valued
// reference yields one id, so callers need not know the cardinality.
func (e Element) Refs(key string) []string {
	switch v := e[key].(type) {
	case []any:
		ids := make([]string, 0, len(v))
		for _, item := range v {
			if id := refID(item); id != "" {
				ids = append(ids, id)
			}
		}
		return ids
	case map[string]any:
		if id := refID(v); id != "" {
			return []string{id}
		}
	}
	return nil
}

func refID(v any) string {
	m, _ := v.(map[string]any)
	id, _ := m["@id"].(string)
	return id
}

// Model is a loaded model.
type Model struct {
	// Files are the model files parsed, in the order given to the parser.
	Files []string
	// Elements by @id, and Order lists the ids as the element array does.
	Elements map[string]Element
	Order    []string
	// Warnings are the parser's diagnostics below error severity, each as
	// "file:line:col: severity: message".
	Warnings []string
	// ExperimentalNotice is non-empty when the element form is marked
	// experimental by the parser, as OpenSysML marks api-json today.
	ExperimentalNotice string
	// Library are the standard-library elements the model refers to, by
	// @id. They are not in Elements: the element array refers to them by
	// id only.
	Library map[string]LibraryElement

	// The engine the model was parsed by, kept open for Evaluate until Close.
	client opensysml.Client
	parsed *opensysml.Model
}

// LibraryElement is a standard-library element the model refers to.
type LibraryElement struct {
	QualifiedName string // ScalarValues::Real
	Type          string // its metaclass: DataType
}

// Name is the library element's own name, the last segment of its
// qualified name: Real for ScalarValues::Real.
func (l LibraryElement) Name() string {
	name := l.QualifiedName
	if i := strings.LastIndex(name, "::"); i >= 0 {
		name = name[i+2:]
	}
	if len(name) >= 2 && name[0] == '\'' && name[len(name)-1] == '\'' {
		name = name[1 : len(name)-1]
	}
	return name
}

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

// Load parses every model file under the folder as one model and converts it
// to the API element array.
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
	conv, err := client.Convert(ctx, parsed, opensysml.FormatAPIJSON)
	if err != nil {
		return nil, fmt.Errorf("converting the model to API JSON: %w", err)
	}
	if conv.Experimental {
		m.ExperimentalNotice = conv.ExperimentalNotice
	}
	m.Library = make(map[string]LibraryElement, len(conv.LibraryReferences))
	for _, r := range conv.LibraryReferences {
		m.Library[r.ID] = LibraryElement{QualifiedName: r.QualifiedName, Type: r.Type}
	}
	var elements []Element
	if err := json.Unmarshal([]byte(conv.Content), &elements); err != nil {
		return nil, fmt.Errorf("reading the API JSON element array: %w", err)
	}
	f.logger().DebugContext(ctx, "model converted to API JSON", "elements", len(elements), "duration", time.Since(start))
	m.Elements = make(map[string]Element, len(elements))
	m.Order = make([]string, 0, len(elements))
	for _, e := range elements {
		id := e.ID()
		if id == "" {
			continue
		}
		if _, dup := m.Elements[id]; !dup {
			m.Order = append(m.Order, id)
		}
		m.Elements[id] = e
	}
	ok = true
	return m, nil
}

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
