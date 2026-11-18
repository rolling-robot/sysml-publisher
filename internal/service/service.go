// Copyright 2026 Petr Mukhachev
// SPDX-License-Identifier: MIT

// Package service starts a local SysML v2 service for a run: the CLI's
// -api local (CLI-9) and the tests' servicetest both come here. It finds a
// sysml-grpc on PATH, or builds one from the OpenSysML checkout
// ($OPENSYSML_DIR, default ../sysmlv2/OpenSysML beside this module), runs
// it with `-port 0 -report-address -health-port 0`, and answers the
// address it reports — after a probe, so a service that starts but cannot
// serve fails here, with its stderr beside the message. On Linux the
// service is killed when this process dies, so a crashed run leaves none
// behind.
package service

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
	"time"

	"github.com/Open-MBEE/OpenSysML/client/opensysml"
)

// Start starts a local sysml-grpc and answers the address to dial (as
// opensysml.Dial and the -api flag take it) and how to stop it.
func Start() (string, func(), error) {
	bin, err := serviceBinary()
	if err != nil {
		return "", nil, err
	}
	address, stop, err := serve(bin)
	if err != nil {
		return "", nil, err
	}
	if err := probe(address); err != nil {
		stop()
		return "", nil, err
	}
	return address, stop, nil
}

// serviceBinary finds a sysml-grpc on PATH, or builds one from the
// OpenSysML checkout into the cache dir, whose single binary every run
// overwrites (Go-cached: the first run pays for the build, the rest only
// for a cache check). The build writes a temporary file in the same
// directory and renames it, so two processes building at once never hand
// out a half-written binary, and the directory never grows: one binary,
// not one per run.
func serviceBinary() (string, error) {
	if p, err := exec.LookPath("sysml-grpc"); err == nil {
		return p, nil
	}
	dir := checkout()
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	bindir := filepath.Join(cache, "sysml-publish")
	if err := os.MkdirAll(bindir, 0o755); err != nil {
		return "", err
	}
	bin := filepath.Join(bindir, "sysml-grpc")
	tmp, err := os.CreateTemp(bindir, "sysml-grpc-*")
	if err != nil {
		return "", err
	}
	name := tmp.Name()
	tmp.Close()
	build := exec.Command("go", "build", "-o", name, "./cmd/sysml-grpc")
	build.Dir = dir
	var out bytes.Buffer
	build.Stdout, build.Stderr = &out, &out
	if err := build.Run(); err != nil {
		os.Remove(name)
		return "", fmt.Errorf("no sysml-grpc on PATH, and building one from %s failed: %w: %s", dir, err, oneLine(out.String()))
	}
	if err := os.Rename(name, bin); err != nil {
		os.Remove(name)
		return "", err
	}
	return bin, nil
}

// checkout is where OpenSysML lives: $OPENSYSML_DIR, else ../sysmlv2/OpenSysML
// beside this module, found from this file's own path, not the working
// directory the run may have changed.
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
// reports on stdout, and how to stop it.
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
// parked thread per process with a local service.
func spawn(cmd *exec.Cmd) error {
	errc := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		errc <- cmd.Start()
		<-errc // park with the thread, until the process is over
	}()
	return <-errc
}

// probe dials the service and asks what it serves, so the run's first
// failure reads as "the service is broken", not as every read failing on a
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
