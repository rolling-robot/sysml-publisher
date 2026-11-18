// Copyright 2026 Petr Mukhachev
// SPDX-License-Identifier: MIT

//go:build linux

package servicetest

import (
	"os/exec"
	"syscall"
)

// setPDeathsig has Linux kill the service when the test process dies, so a
// package's one service outlives its sequential tests and outlives nothing
// else.
func setPDeathsig(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
	return nil
}
