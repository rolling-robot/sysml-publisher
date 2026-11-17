// Copyright 2026 Petr Mukhachev
// SPDX-License-Identifier: MIT

// This file runs the one program a rendering needs: Graphviz's dot, which
// lays a diagram out. A table is written from the rendering's rows, with no
// program at all.

package views

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// run runs cmd with stdin and returns its stdout. A program that cannot be
// started is an error naming it; one that fails is an error of the last line
// it wrote to stderr.
func run(cmd *exec.Cmd, program string, stdin []byte) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdin = bytes.NewReader(stdin)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit):
		return nil, fmt.Errorf("%s: %s", program, lastLine(stderr.String(), program, exit))
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
