// Command sysml-publish builds a static HTML page tree from the AsciiDoc
// textual representations of a SysML v2 model.
//
//	sysml-publish [-model DIR] [-out DIR]
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/rolling-robot/sysml-publisher/internal/model"
	"github.com/rolling-robot/sysml-publisher/internal/site"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("sysml-publish", flag.ContinueOnError)
	fs.SetOutput(stderr)
	modelDir := fs.String("model", "./model", "folder holding the SysML model files (*.sysml, *.kerml), searched recursively")
	outDir := fs.String("out", "./site", "folder to write the pages to; replaced on every run")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage: sysml-publish [-model DIR] [-out DIR]")
		fmt.Fprintln(stderr, "\nBuilds a static HTML page tree from the AsciiDoc textual representations of a SysML v2 model.")
		fmt.Fprintln(stderr)
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(stderr, "sysml-publish: unexpected argument %q\n", fs.Arg(0))
		fs.Usage()
		return 2
	}
	if err := publish(*modelDir, *outDir, stdout, stderr); err != nil {
		fmt.Fprintln(stderr, "sysml-publish:", err)
		return 1
	}
	return 0
}

func publish(modelDir, outDir string, stdout, stderr io.Writer) error {
	m, err := model.Folder{Dir: modelDir}.Load(context.Background())
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "model: %d file(s), %d elements, %d warning(s)\n", len(m.Files), len(m.Elements), len(m.Warnings))
	if m.ExperimentalNotice != "" {
		fmt.Fprintln(stderr, "note: the SysML v2 API JSON conversion is experimental in OpenSysML")
	}
	for _, w := range m.Warnings {
		fmt.Fprintln(stderr, "warning:", w)
	}

	t := tree.Build(m)
	for _, w := range t.Warnings {
		fmt.Fprintln(stderr, "warning:", w)
	}
	documented := 0
	for _, n := range t.Pages {
		if n.Documented() {
			documented++
		}
	}
	if documented == 0 {
		fmt.Fprintln(stderr, "warning: no element has an AsciiDoc representation; writing the landing page only")
	}

	s, err := site.Render(t)
	if err != nil {
		return err
	}
	if err := s.Write(outDir); err != nil {
		return err
	}
	for _, n := range t.Pages {
		label := n.Name
		if !n.Documented() {
			label += " (empty)"
		}
		fmt.Fprintf(stdout, "page: %s → %s\n", label, filepath.Join(outDir, filepath.FromSlash(n.Dir), "index.html"))
	}
	fmt.Fprintf(stdout, "%d page(s) and the landing page written to %s\n", len(t.Pages), outDir)
	return nil
}
