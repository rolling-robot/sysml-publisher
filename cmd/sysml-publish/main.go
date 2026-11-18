// Copyright 2026 Petr Mukhachev
// SPDX-License-Identifier: MIT

// Command sysml-publish builds a static HTML page tree from the AsciiDoc
// textual representations of a SysML v2 model, which a sysml-grpc service
// parses and holds (DS-1).
//
//	sysml-publish -api ADDR [-config FILE] [-model DIR] [-out DIR] [-v] [-log-format text|json] [-dot CMD]
//
// An optional TOML config file (CFG-1..CFG-6) names the model folder, the
// service's address and the site's folder, title and base URL; command-line
// flags win over it.
//
// Everything it reports goes to stderr as structured log records (log/slog):
// human-readable text by default, JSON with -log-format json.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/rolling-robot/sysml-publisher/internal/model"
	"github.com/rolling-robot/sysml-publisher/internal/service"
	"github.com/rolling-robot/sysml-publisher/internal/site"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
	"github.com/rolling-robot/sysml-publisher/views"
)

// Exit codes.
const (
	exitOK          = 0
	exitFailure     = 1
	exitUsage       = 2
	exitInterrupted = 130 // 128 + SIGINT, as shells report it
)

func main() {
	// Ctrl-C or SIGTERM cancels the run; nothing is written (CLI-7).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Stderr)
	stop()
	os.Exit(code)
}

func run(ctx context.Context, args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("sysml-publish", flag.ContinueOnError)
	fs.SetOutput(stderr)
	apiAddr := fs.String("api", "", "address of the SysML v2 service (sysml-grpc) that parses the model and answers the reads (CLI-9); for example localhost:50051, or local to start one for this run and stop it after")
	modelDir := fs.String("model", "./model", "folder holding the SysML model files (*.sysml, *.kerml), searched recursively")
	outDir := fs.String("out", "./site", "folder to write the pages to; replaced on every run")
	configPath := fs.String("config", "", "configuration file (TOML); "+DefaultConfigFile+" in the working directory when unset (CFG-1)")
	verbose := fs.Bool("v", false, "also log debug records: the duration of every stage and page")
	logFormat := fs.String("log-format", "text", "format of the log on stderr: text or json")
	dotCmd := fs.String("dot", "dot", "Graphviz's dot command, which lays out the diagrams sysml-view shows")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: sysml-publish -api ADDR [-config FILE] [-model DIR] [-out DIR] [-v] [-log-format text|json] [-dot CMD]")
		fmt.Fprintln(stderr, "\nBuilds a static HTML page tree from the AsciiDoc textual representations of a SysML v2 model, parsed by a sysml-grpc service.")
		fmt.Fprintln(stderr)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitUsage
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "sysml-publish: unexpected argument %q\n", fs.Arg(0))
		fs.Usage()
		return exitUsage
	}
	logger, err := newLogger(stderr, *logFormat, *verbose)
	if err != nil {
		fmt.Fprintln(stderr, "sysml-publish:", err)
		fs.Usage()
		return exitUsage
	}
	// Libraries that log through the default logger use the same one.
	slog.SetDefault(logger)

	// The config file (CFG-1..CFG-3): an explicit -config must exist; the
	// default name may not. A flag the user gave wins over the file, the
	// file over the flag's default.
	given := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { given[f.Name] = true })
	path := *configPath
	if path == "" {
		path = DefaultConfigFile
	}
	cfg, err := loadConfig(path, given["config"])
	if err != nil {
		fmt.Fprintln(stderr, "sysml-publish:", err)
		return exitUsage
	}
	modelPath, sitePath := *modelDir, *outDir
	if !given["model"] && cfg.Source.Dir != "" {
		modelPath = cfg.Source.Dir
	}
	if !given["out"] && cfg.Site.Dir != "" {
		sitePath = cfg.Site.Dir
	}
	api := *apiAddr
	if !given["api"] && cfg.Source.API != "" {
		api = cfg.Source.API
	}
	if api == "" {
		fmt.Fprintln(stderr, "sysml-publish: no SysML service address (CLI-9): start a sysml-grpc service and name it with -api ADDR, or [source] api in "+DefaultConfigFile)
		return exitUsage
	}
	// "local" names no service: it says the tool starts one of its own for
	// this run and stops it after (CLI-9). A service that cannot be
	// started is a failure, named why (ER-1).
	if api == "local" {
		addr, stop, err := service.Start()
		if err != nil {
			fmt.Fprintln(stderr, "sysml-publish:", err)
			return exitFailure
		}
		defer stop()
		api = addr
		logger.Info("started a local SysML service", "address", addr)
	}

	err = publish(ctx, logger, modelPath, api, sitePath, *dotCmd, site.Options{Title: cfg.Site.Title, BaseURL: cfg.Site.BaseURL})
	switch {
	case err == nil:
		return exitOK
	case ctx.Err() != nil:
		// The Service reports a cancelled call with its own error, which
		// does not wrap context.Canceled, so ask the context.
		logger.Error("interrupted; nothing written")
		return exitInterrupted
	}
	// Each model error is its own record, so JSON stays one record per
	// problem; the error itself is logged once, here (never also deeper).
	var diags *model.DiagnosticsError
	if errors.As(err, &diags) {
		for _, d := range diags.Errors {
			logger.Error("model error", "diagnostic", d)
		}
		logger.Error("publishing failed: the model has errors; nothing written", "errors", len(diags.Errors))
		return exitFailure
	}
	logger.Error("publishing failed; nothing written", "error", err)
	return exitFailure
}

// newLogger is the logger for format ("text" or "json"), at Debug level when
// verbose and Info otherwise. Text records leave out the time, which a person
// watching the run does not need; JSON records keep it for log tooling.
func newLogger(w io.Writer, format string, verbose bool) (*slog.Logger, error) {
	opts := &slog.HandlerOptions{Level: slog.LevelInfo}
	if verbose {
		opts.Level = slog.LevelDebug
	}
	switch format {
	case "text":
		opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			if len(groups) == 0 && a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		}
		return slog.New(slog.NewTextHandler(w, opts)), nil
	case "json":
		return slog.New(slog.NewJSONHandler(w, opts)), nil
	}
	return nil, fmt.Errorf("unknown -log-format %q: want text or json", format)
}

// publish builds the site of the model in modelDir into outDir; the model is
// parsed by the SysML service at api (DS-1, CLI-9), dotCmd lays
// out the diagrams sysml-view shows (views.InProcess), and siteOpts carry
// the config file's site title and base URL (CFG-4, CFG-5).
func publish(ctx context.Context, logger *slog.Logger, modelDir, api, outDir, dotCmd string, siteOpts site.Options) error {
	start := time.Now()

	stage := time.Now()
	m, err := model.Folder{Dir: modelDir, API: api, Logger: logger}.Load(ctx)
	if err != nil {
		return err
	}
	defer m.Close()
	logger.InfoContext(ctx, "model loaded",
		"folder", modelDir, "files", len(m.Files), "elements", m.ElementCount,
		"warnings", len(m.Warnings), "duration", time.Since(stage))
	for _, w := range m.Warnings {
		logger.WarnContext(ctx, "model warning", "diagnostic", w)
	}

	stage = time.Now()
	t, err := tree.Build(ctx, m)
	if err != nil {
		return err
	}
	documented := 0
	for _, n := range t.Pages {
		if n.Documented() {
			documented++
		}
	}
	logger.DebugContext(ctx, "page tree built", "pages", len(t.Pages), "documented", documented,
		"queries", m.Stats().Queries, "element_reads", m.Stats().Reads, "cache_hits", m.Stats().Hits,
		"duration", time.Since(stage))
	for _, w := range t.Warnings {
		logger.WarnContext(ctx, w)
	}
	if documented == 0 {
		logger.WarnContext(ctx, "no element has an AsciiDoc representation; writing the landing page only")
	}

	stage = time.Now()
	s, err := site.Render(ctx, t, logger, &views.InProcess{Model: m.Model, Dot: dotCmd}, siteOpts)
	if err != nil {
		return err
	}
	logger.DebugContext(ctx, "pages rendered", "pages", len(t.Pages)+1, "duration", time.Since(stage))
	for _, w := range s.Warnings {
		logger.WarnContext(ctx, w)
	}

	if err := ctx.Err(); err != nil {
		return err // interrupted after rendering: still write nothing
	}
	stage = time.Now()
	if err := s.Write(outDir); err != nil {
		return err
	}
	logger.DebugContext(ctx, "site written", "files", len(s.Files), "duration", time.Since(stage))
	for _, n := range t.Pages {
		logger.InfoContext(ctx, "page written",
			"element", n.QualifiedName, "empty", !n.Documented(),
			"path", filepath.Join(outDir, filepath.FromSlash(n.Dir), "index.html"))
	}
	logger.InfoContext(ctx, "site published",
		"pages", len(t.Pages), "landing", filepath.Join(outDir, "index.html"), "duration", time.Since(start))
	return nil
}
