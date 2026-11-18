// Copyright 2026 Petr Mukhachev
// SPDX-License-Identifier: MIT

// Package servicetest gives the tests the address of a real SysML v2
// service (DS-1): the suite runs the real stack, and the stack now holds a
// sysml-grpc. The address comes from $SYSML_PUBLISH_TEST_API (a service
// someone else started), else from internal/service — a sysml-grpc found
// on PATH, or one built from the OpenSysML checkout and run here. One
// service per test process, shared by its tests; on Linux the service is
// killed when the test process dies, so no cleanup can stop it while a
// later test of the same package still needs it. A test that cannot get a
// service skips, and the skip says why.
package servicetest

import (
	"os"
	"sync"
	"testing"

	"github.com/rolling-robot/sysml-publisher/internal/service"
)

// envAPI names a service a caller of the suite already runs.
const envAPI = "SYSML_PUBLISH_TEST_API"

var (
	startOnce sync.Once
	addr      string
	startErr  error
)

// API returns the address the test dials (as opensysml.Dial and the -api
// flag take it). Every test of the package may call it; they share the one
// service, which lives until the test process does.
func API(t *testing.T) string {
	t.Helper()
	startOnce.Do(func() {
		a, _, e := Service()
		addr, startErr = a, e
	})
	if startErr != nil {
		t.Skipf("no SysML service: %v", startErr)
	}
	return addr
}

// Service is the service the tests and examples use: its address, and how
// to stop it when it was started here (a no-op for the environment's).
// It is Service, not API, that a runnable Example — which has no
// *testing.T to skip with — calls.
func Service() (string, func(), error) {
	if env := os.Getenv(envAPI); env != "" {
		return env, func() {}, nil
	}
	return service.Start()
}
