// Copyright 2026 Petr Mukhachev
// SPDX-License-Identifier: MIT

// Package servicetest gives the tests the address of a real SysML v2
// service (DS-1): the suite runs the real stack, and the stack now holds a
// sysml-grpc. The address comes from $SYSML_PUBLISH_TEST_API (a service
// someone else started), else a sysml-grpc found on PATH, else one built
// from the OpenSysML checkout and run here with `-port 0 -report-address
// -health-port 0`. One service per test process, shared by its tests; on
// Linux the service is killed when the test process dies, so no cleanup
// can stop it while a later test of the same package still needs it. A
// test that cannot get a service skips, and the skip says why.
package servicetest

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Open-MBEE/OpenSysML/client/opensysml"
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
		a, _, e := start()
		addr, startErr = a, e
	})
	if startErr != nil {
		t.Skipf("no SysML service: %v", startErr)
	}
	return addr
}

// Service is API without the test plumbing: the address a runnable Example
// — which has no *testing.T to skip with — gets, and how to stop a service
// this call started (a no-op for the environment's). Examples cannot skip,
// so they fail rather than skip when no service can be obtained.
func Service() (string, func(), error) {
	a, stop, err := start()
	if err != nil {
		return "", nil, err
	}
	return a, stop, nil
}

// start obtains a service: the environment's, or one on PATH, or one built
// from the checkout.
func start() (string, func(), error) {
	if env := os.Getenv(envAPI); env != "" {
		return env, func() {}, nil
	}
	bin, err := serviceBinary()
	if err != nil {
		return "", nil, err
	}
	address, stop, err := serve(bin)
	if err != nil {
		return "", nil, err
	}
	// The service answers before the caller uses it, so a build that starts
	// but cannot serve fails here, once, with its stderr beside the message.
	if err := probe(address); err != nil {
		stop()
		return "", nil, err
	}
	return address, stop, nil
}

// serviceBinary finds a sysml-grpc on PATH, or builds one from the
// OpenSysML checkout. The build is Go-cached: the first test process pays
// for it, the rest only for a cache check.
func serviceBinary() (string, error) {
	if p, err := exec.LookPath("sysml-grpc"); err == nil {
		return p, nil
	}
	dir := checkout()
	tmp, err := os.MkdirTemp("", "sysml-grpc-servicetest-")
	if err != nil {
		return "", err
	}
	bin := filepath.Join(tmp, "sysml-grpc")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/sysml-grpc")
	cmd.Dir = dir
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("building sysml-grpc from %s: %w: %s", dir, err, oneLine(out.String()))
	}
	return bin, nil
}

// checkout is where OpenSysML lives: $OPENSYSML_DIR, else ../sysmlv2/OpenSysML
// beside this module, found from this file's own path, not the working
// directory a test may have changed.
func checkout() string {
	if d := os.Getenv("OPENSYSML_DIR"); d != "" {
		return d
	}
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "../sysmlv2/OpenSysML"
	}
	repo := filepath.Dir(filepath.Dir(filepath.Dir(thisFile)))
	return filepath.Join(filepath.Dir(repo), "sysmlv2", "OpenSysML")
}

// serve runs bin with a kernel-chosen port and returns the address it
// reports on stdout. On Linux the service is killed when the test process
// dies, which is what keeps it alive across the package's sequential tests
// and keeps a crashed run from leaving one behind.
func serve(bin string) (string, func(), error) {
	cmd := exec.Command(bin, "-port", "0", "-report-address", "-health-port", "0")
	if err := setPDeathsig(cmd); err != nil {
		return "", nil, err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", nil, err
	}
	if err := spawn(cmd); err != nil {
		return "", nil, fmt.Errorf("starting %s: %w", bin, err)
	}
	line := make(chan string, 1)
	go func() {
		l, _ := bufio.NewReader(stdout).ReadString('\n')
		line <- strings.TrimSpace(l)
	}()
	runDown := func() {
		cmd.Process.Kill()
		cmd.Wait()
	}
	select {
	case address := <-line:
		if address == "" {
			runDown()
			return "", nil, fmt.Errorf("%s printed no address; stderr: %s", bin, oneLine(stderr.String()))
		}
		return address, runDown, nil
	case <-time.After(2 * time.Minute):
		runDown()
		return "", nil, fmt.Errorf("%s printed no address within 2m; stderr: %s", bin, oneLine(stderr.String()))
	}
}

// spawn starts cmd on a thread locked for its lifetime: Linux's parent-death
// signal fires when the starting thread dies, and the runtime may retire it
// otherwise. The goroutine parks forever after the start, which is one
// parked thread per test process.
func spawn(cmd *exec.Cmd) error {
	errc := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		errc <- cmd.Start()
		<-errc // park with the thread, until the process is over
	}()
	return <-errc
}

// probe dials the service and asks what it serves, so the suite's first
// failure reads as "the service is broken", not as every test failing on a
// refused call.
func probe(address string) error {
	client, err := opensysml.Dial(address)
	if err != nil {
		return err
	}
	defer client.Close()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	if _, err := client.ServerInfo(ctx); err != nil {
		return fmt.Errorf("the service at %s answers nothing: %w", address, err)
	}
	return nil
}

// oneLine flattens a log for an error message.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
