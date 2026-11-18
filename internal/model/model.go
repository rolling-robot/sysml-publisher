// Copyright 2026 Petr Mukhachev
// SPDX-License-Identifier: MIT

// Package model hands a folder of SysML v2 files to a SysML v2 service and
// opens the parsed model for reading through package sysml (DS-1, DS-2,
// DS-4). The tool does not parse: the Service parses the model, holds it,
// and answers the tool's reads.
package model

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
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

// Model is a loaded model: the files it came from, the Service's warnings,
// and the model read through package sysml, whose methods it has. Close
// releases the client; the service keeps running.
type Model struct {
	// Files are the model files read, in the order they were sent to the
	// service.
	Files []string
	// Warnings are the parser's diagnostics below error severity, each as
	// "file:line:col: severity: message".
	Warnings []string
	*sysml.Model
}

// Source is where a model comes from. The Service parses; a Source speaks
// one service kind (DS-3). Today Folder reads a folder of files and speaks
// sysml-grpc; a SysML v2 REST service (Flexo) is a later Source.
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
// handed to the SysML service as the documents of one model.
type Folder struct {
	Dir string
	// API is the address of the SysML v2 service that parses the model and
	// answers the tool's reads (DS-1, DS-3): sysml-grpc, spelled as
	// opensysml.Dial takes it ("localhost:50051", "https://sysml.example.com").
	API string
	// Logger receives debug records (stage durations); nil uses slog.Default.
	Logger *slog.Logger
}

// requiredCapabilities are what the tool needs of the Service (ER-1):
// parse_sources to send the model as documents, query and
// element_properties for the reads the tree and macros make, render_view
// and render_view_form for sysml-view, and evaluate_subject because a
// service without it would silently evaluate sysml-value against declared
// defaults instead of the page's element.
var requiredCapabilities = []string{
	opensysml.CapabilityParseSources,
	opensysml.CapabilityQuery,
	opensysml.CapabilityElementProperties,
	opensysml.CapabilityRenderView,
	opensysml.CapabilityRenderViewForm,
	opensysml.CapabilityEvaluateSubject,
}

func (f Folder) logger() *slog.Logger {
	if f.Logger != nil {
		return f.Logger
	}
	return slog.Default()
}

// Load dials the service at API, checks it serves what the tool needs, sends
// every model file as one document of a ParseSources model (DS-2), and opens
// the result for reading (sysml.Open). The client stays open until
// Model.Close; the service keeps the model (DS-1): the tool holds the files'
// paths, the diagnostics, and nothing else of the model here.
func (f Folder) Load(ctx context.Context) (*Model, error) {
	if f.API == "" {
		return nil, errors.New("no SysML service address: give -api ADDR (or [source] api) of a sysml-grpc service")
	}
	files, err := modelFiles(f.Dir)
	if err != nil {
		return nil, err
	}
	client, err := opensysml.Dial(f.API)
	if err != nil {
		return nil, err
	}
	// The client stays open for Evaluate and RenderView when loading
	// succeeds; the caller closes it with Model.Close.
	ok := false
	defer func() {
		if !ok {
			client.Close()
		}
	}()

	// Dial connects lazily; this is the call that reaches the service, so an
	// unreachable one fails here, naming the address (ER-1).
	info, err := client.ServerInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("reaching the SysML service at %s: %w", f.API, err)
	}
	var missing []string
	for _, c := range requiredCapabilities {
		if !info.Has(c) {
			missing = append(missing, c)
		}
	}
	if missing != nil {
		return nil, fmt.Errorf("the SysML service at %s does not serve %s", f.API, strings.Join(missing, ", "))
	}

	// The files are read and sent as the documents of one model (DS-2); the
	// tool parses nothing and holds no parsed model (DS-1). Inline documents
	// are named by their path, which is what diagnostics report, and .kerml
	// ones say their language, which content alone cannot carry.
	start := time.Now()
	docs := make([]opensysml.Document, 0, len(files))
	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", path, err)
		}
		doc := opensysml.Source(path, string(content))
		if strings.HasSuffix(path, ".kerml") {
			doc.Language = opensysml.LanguageKerML
		}
		docs = append(docs, doc)
	}
	parsed, err := client.ParseDocuments(ctx, docs)
	if err != nil {
		return nil, fmt.Errorf("the SysML service at %s refused the model: %w", f.API, err)
	}
	f.logger().DebugContext(ctx, "model parsed",
		"files", len(files), "diagnostics", len(parsed.Diagnostics), "duration", time.Since(start))

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

// modelFiles lists the model files under dir, sorted so every run sends them
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
