// Package model loads a SysML v2 model from a folder of files: it parses
// them in-process as one model, reports the parser's diagnostics, and opens
// the result for reading through package sysml (DS-2, DS-4).
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
	"github.com/rolling-robot/sysml-publisher/sysml"
)

// The read layer moved to package sysml; these names stay while callers
// move to it.
type (
	Ref            = sysml.Ref
	Props          = sysml.Props
	Representation = sysml.Representation
	Stats          = sysml.Stats
)

// ErrNoSuchElement is sysml.ErrNoSuchElement.
var ErrNoSuchElement = sysml.ErrNoSuchElement

// Model is a loaded model: the files it came from, the parser's warnings,
// and the model read through package sysml, whose methods it has. Close
// releases the engine.
type Model struct {
	// Files are the model files parsed, in the order given to the parser.
	Files []string
	// Warnings are the parser's diagnostics below error severity, each as
	// "file:line:col: severity: message".
	Warnings []string
	*sysml.Model
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

// Load parses every model file under the folder as one model and opens it
// for reading (sysml.Open). The engine stays open until Model.Close.
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
	m := &Model{Files: files}
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
	if m.Model, err = sysml.Open(ctx, client, parsed); err != nil {
		return nil, err
	}
	f.logger().DebugContext(ctx, "model queried", "elements", m.ElementCount, "top-level", len(m.Tops), "duration", time.Since(start))
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
