// Package views renders SysML v2 views with external programs, for the
// sysml-view macro (requirement MC-21): OpenSysML's sysml command writes a
// view as DOT, and Graphviz's dot lays the DOT out as SVG; a table view,
// which has no DOT form, is written as CSV.
//
//	reg.SetViews(&views.CLI{Files: files}) // reg from macros.Register
package views

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/rolling-robot/sysml-publisher/macros"
)

// CLI renders views by running the sysml and dot commands, once per view:
// it is a macros.Views. Renderings are kept, failures included, so a view
// shown on several pages is rendered once. Safe for concurrent use.
type CLI struct {
	// Files are the model's files, all loaded by sysml as one model: the
	// files the model was parsed from.
	Files []string
	// Sysml is OpenSysML's sysml command; "" runs "sysml" from PATH.
	Sysml string
	// Dot is Graphviz's dot command; "" runs "dot" from PATH. sysml is
	// given it too (OPENSYSML_DOT), for the layouts it runs itself.
	Dot string

	mu    sync.Mutex
	cache map[string]result
}

type result struct {
	r   *macros.Rendering
	err error
}

var _ macros.Views = (*CLI)(nil)

// Render renders the view with that qualified name: a diagram as SVG (sysml
// -render-form dot, then dot -Tsvg), else a table (-render-form csv). Its
// errors say what went wrong as the programs report it.
func (c *CLI) Render(ctx context.Context, qualifiedName string) (*macros.Rendering, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if res, ok := c.cache[qualifiedName]; ok {
		return res.r, res.err
	}
	r, err := c.render(ctx, qualifiedName)
	if ctx.Err() != nil {
		return nil, ctx.Err() // not kept: an interrupted rendering says nothing about the view
	}
	if c.cache == nil {
		c.cache = map[string]result{}
	}
	c.cache[qualifiedName] = result{r, err}
	return r, err
}

func (c *CLI) render(ctx context.Context, qn string) (*macros.Rendering, error) {
	dot, err := c.sysml(ctx, qn, "dot")
	if err == nil {
		svg, err := c.layout(ctx, dot)
		if err != nil {
			return nil, err
		}
		return &macros.Rendering{SVG: svg}, nil
	}
	var refused *refusal
	if !errors.As(err, &refused) {
		return nil, err // sysml did not run
	}
	// A table view has no DOT form and a diagram no CSV form, so a view
	// sysml writes as CSV is a table; otherwise DOT's refusal is the
	// answer.
	text, err2 := c.sysml(ctx, qn, "csv")
	if err2 != nil {
		return nil, err
	}
	rows, err2 := csv.NewReader(bytes.NewReader(text)).ReadAll()
	if err2 != nil {
		return nil, fmt.Errorf("reading the table sysml wrote: %w", err2)
	}
	return &macros.Rendering{Table: rows}, nil
}

// refusal is sysml or dot ending with a failure: Message is the last line
// it wrote to stderr.
type refusal struct {
	Program, Message string
}

func (r *refusal) Error() string { return r.Program + ": " + r.Message }

// sysml runs sysml -render qn -render-form form over the model's files.
func (c *CLI) sysml(ctx context.Context, qn, form string) ([]byte, error) {
	args := append(append([]string{}, c.Files...), "-render", qn, "-render-form", form)
	cmd := exec.CommandContext(ctx, or(c.Sysml, "sysml"), args...)
	if c.Dot != "" {
		cmd.Env = append(os.Environ(), "OPENSYSML_DOT="+c.Dot)
	}
	return run(cmd, "sysml", nil)
}

// layout lays DOT out as SVG with dot -Tsvg.
func (c *CLI) layout(ctx context.Context, dot []byte) ([]byte, error) {
	return run(exec.CommandContext(ctx, or(c.Dot, "dot"), "-Tsvg"), "dot", dot)
}

// run runs cmd with stdin and returns its stdout. A program that cannot be
// started is an error naming it; one that fails is a *refusal.
func run(cmd *exec.Cmd, program string, stdin []byte) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit):
		return nil, &refusal{Program: program, Message: lastLine(stderr.String(), program, exit)}
	case err != nil:
		return nil, fmt.Errorf("cannot run %s: %w", program, err)
	}
	return stdout.Bytes(), nil
}

// lastLine is the last line a failing program wrote to stderr, without
// its "program: " prefix, or its exit status when it wrote nothing.
func lastLine(stderr, program string, exit *exec.ExitError) string {
	lines := strings.Split(strings.TrimSpace(stderr), "\n")
	last := strings.TrimSpace(lines[len(lines)-1])
	if last == "" {
		return exit.String()
	}
	return strings.TrimPrefix(last, program+": ")
}

func or(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
