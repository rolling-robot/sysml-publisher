// Copyright 2026 Petr Mukhachev
// SPDX-License-Identifier: MIT

package views

import (
	"os/exec"
	"testing"
)

// These tests run sh, which any Unix has, to make a program fail the way dot
// does: a rendering this tool writes is valid DOT, so dot's own failures are
// not something a test can bring about.
func shFailing(t *testing.T, script string) error {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skipf("sh is not installed: %v", err)
	}
	_, err := run(exec.Command("sh", "-c", script), "dot", nil)
	return err
}

func TestRunReportsAFailingProgram(t *testing.T) {
	cases := []struct{ name, script, want string }{
		{"the program's last line, without its name", "echo one >&2; echo 'dot: boom' >&2; exit 3", "dot: boom"},
		{"its exit status when it wrote nothing", "exit 3", "dot: exit status 3"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if err := shFailing(t, c.script); err == nil || err.Error() != c.want {
				t.Errorf("error %v, want %s", err, c.want)
			}
		})
	}
}
