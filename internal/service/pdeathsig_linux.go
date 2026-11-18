// Copyright 2026 Petr Mukhachev
// SPDX-License-Identifier: MIT

//go:build linux

package service

import (
	"os/exec"
	"syscall"
)

// setPDeathsig has Linux kill the service when this process dies, so a
// crashed run leaves no service behind.
func setPDeathsig(cmd *exec.Cmd) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGKILL}
	return nil
}
