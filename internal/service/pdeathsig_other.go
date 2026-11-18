// Copyright 2026 Petr Mukhachev
// SPDX-License-Identifier: MIT

//go:build !linux

package service

import "os/exec"

// setPDeathsig is Linux's service reaping; elsewhere the service outlives
// this process unless its stop ran, which is acceptable for a development
// machine and reaped at reboot.
func setPDeathsig(cmd *exec.Cmd) error { return nil }
