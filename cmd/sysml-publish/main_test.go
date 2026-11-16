package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func fixture(name string) string { return filepath.Join("..", "..", "testdata", name) }

// fixtureAbs is a fixture path that stays right when the test changes the
// working directory.
func fixtureAbs(t *testing.T, name string) string {
	t.Helper()
	p, err := filepath.Abs(fixture(name))
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestPublishSmoke(t *testing.T) {
	out := filepath.Join(t.TempDir(), "site")
	var log bytes.Buffer
	if code := run(context.Background(), []string{"-model", fixture("smoke"), "-out", out}, &log); code != exitOK {
		t.Fatalf("exit %d, log:\n%s", code, log.String())
	}
	for _, f := range []string{"index.html", "Model/pd/index.html", "Model/aaaa/index.html", ".sysml-publish"} {
		if _, err := os.Stat(filepath.Join(out, f)); err != nil {
			t.Errorf("%s not written: %v", f, err)
		}
	}
	text := log.String()
	for _, want := range []string{
		`level=INFO msg="model loaded"`,
		`msg="page written" element=Model::pd`,
		`msg="site published" pages=4`,
	} {
		if !strings.Contains(text, want) {
			t.Errorf("log lacks %q:\n%s", want, text)
		}
	}
	// Text records carry no time; debug records need -v.
	if strings.Contains(text, "time=") || strings.Contains(text, "level=DEBUG") {
		t.Errorf("unexpected time or debug records:\n%s", text)
	}
}

func TestJSONLogWithTimings(t *testing.T) {
	out := filepath.Join(t.TempDir(), "site")
	var log bytes.Buffer
	args := []string{"-model", fixture("smoke"), "-out", out, "-v", "-log-format", "json"}
	if code := run(context.Background(), args, &log); code != exitOK {
		t.Fatalf("exit %d, log:\n%s", code, log.String())
	}
	msgs := map[string]map[string]any{}
	sc := bufio.NewScanner(&log)
	for sc.Scan() {
		var rec map[string]any
		if err := json.Unmarshal(sc.Bytes(), &rec); err != nil {
			t.Fatalf("not a JSON record: %q: %v", sc.Text(), err)
		}
		if _, ok := rec["time"]; !ok {
			t.Errorf("JSON record without time: %q", sc.Text())
		}
		msgs[rec["msg"].(string)] = rec
	}
	// The debug records time every stage.
	for _, msg := range []string{"model parsed", "model queried", "page tree built", "page converted", "pages rendered", "site written", "site published"} {
		rec, ok := msgs[msg]
		if !ok {
			t.Errorf("no %q record", msg)
			continue
		}
		if _, ok := rec["duration"]; !ok {
			t.Errorf("%q record has no duration", msg)
		}
	}
	// The page tree record counts how the model was read (DS-4).
	for _, key := range []string{"queries", "element_reads", "cache_hits"} {
		if n, ok := msgs["page tree built"][key].(float64); !ok || n <= 0 {
			t.Errorf("page tree built: %s = %v, want a positive count", key, msgs["page tree built"][key])
		}
	}
}

func TestModelErrorsAreLoggedEachAndNothingWritten(t *testing.T) {
	out := filepath.Join(t.TempDir(), "site")
	var log bytes.Buffer
	if code := run(context.Background(), []string{"-model", fixture("syntaxerror"), "-out", out}, &log); code != exitFailure {
		t.Fatalf("exit %d, want %d; log:\n%s", code, exitFailure, log.String())
	}
	text := log.String()
	if !strings.Contains(text, `level=ERROR msg="model error" diagnostic=`) || !strings.Contains(text, "bad.sysml:2:") {
		t.Errorf("model error not logged with its location:\n%s", text)
	}
	if _, err := os.Stat(out); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("site folder created after a failed run: %v", err)
	}
}

// ER-4: a model without documented elements is not an error: the run warns,
// exits 0 and writes the landing page only (with its assets and marker).
func TestModelWithoutRepresentationsWritesLandingPageOnly(t *testing.T) {
	out := filepath.Join(t.TempDir(), "site")
	var log bytes.Buffer
	if code := run(t.Context(), []string{"-model", fixture("undocumented"), "-out", out}, &log); code != exitOK {
		t.Fatalf("exit %d, want %d; log:\n%s", code, exitOK, log.String())
	}
	text := log.String()
	if !strings.Contains(text, `level=WARN msg="no element has an AsciiDoc representation; writing the landing page only"`) {
		t.Errorf("log lacks the ER-4 warning:\n%s", text)
	}
	if strings.Contains(text, `msg="page written"`) {
		t.Errorf("pages reported written:\n%s", text)
	}
	if !strings.Contains(text, `msg="site published" pages=0`) {
		t.Errorf("summary does not report zero pages:\n%s", text)
	}

	var got []string
	err := filepath.WalkDir(out, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(out, path)
		got = append(got, filepath.ToSlash(rel))
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{".sysml-publish", "assets/asciidoctor.css", "assets/layout.css", "index.html"}
	if !slices.Equal(got, want) {
		t.Errorf("files written = %v, want %v", got, want)
	}
}

// Failing macros do not fail the run (MC-19): every failing page is
// reported as a warning with the file and line of the macro, and the site is
// written with placeholders. The cases are the // notes in
// testdata/macroerror/macroerror.sysml.
func TestMacroFailuresAreWarnings(t *testing.T) {
	out := filepath.Join(t.TempDir(), "site")
	var log bytes.Buffer
	if code := run(t.Context(), []string{"-model", fixture("macroerror"), "-out", out}, &log); code != exitOK {
		t.Fatalf("exit %d, want %d; log:\n%s", code, exitOK, log.String())
	}
	file := filepath.Join(fixture("macroerror"), "macroerror.sysml")
	var got []string
	for _, line := range strings.Split(log.String(), "\n") {
		if strings.HasPrefix(line, "level=WARN ") {
			got = append(got, line)
		}
	}
	prefixes := []string{
		`level=WARN msg="` + file + `:21: M::a: sysml-value:mas[]: `,
		`level=WARN msg="` + file + `:21: M::pd: sysml-value:mas[]: `,
		`level=WARN msg="` + file + `:26: M::b: sysml-doc::Nothing[]: `,
	}
	if len(got) != len(prefixes) {
		t.Fatalf("warnings:\n%s\nwant %d", strings.Join(got, "\n"), len(prefixes))
	}
	for i, p := range prefixes {
		if !strings.HasPrefix(got[i], p) {
			t.Errorf("warning %d:\n got %s\nwant %s...", i, got[i], p)
		}
	}
	page, err := os.ReadFile(filepath.Join(out, "M", "b", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(page), `<div class="sysml-failed">`) {
		t.Errorf("M/b/index.html has no placeholder:\n%s", page)
	}
}

// The config file names the folders, the title and the base URL (CFG-1,
// CFG-2, CFG-4, CFG-5): the run takes them all from -config, and every page
// shows the configured title and its canonical URL.
func TestConfigFile(t *testing.T) {
	model, out := fixtureAbs(t, "smoke"), filepath.Join(t.TempDir(), "site")
	cfg := writeConfig(t, t.TempDir(), fmt.Sprintf(
		"[source]\ndir = %q\n[site]\ndir = %q\ntitle = \"Model docs\"\nbaseURL = \"https://example.com/docs/\"\n", model, out))
	var log bytes.Buffer
	if code := run(t.Context(), []string{"-config", cfg}, &log); code != exitOK {
		t.Fatalf("exit %d, want %d; log:\n%s", code, exitOK, log.String())
	}
	landing, err := os.ReadFile(filepath.Join(out, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	page, err := os.ReadFile(filepath.Join(out, "Model", "pd", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ file, html string }{
		{"index.html", "<title>Model docs</title>"},
		{"index.html", `<a class="home" href="index.html" aria-current="page">Model docs</a>`},
		{"index.html", `<link rel="canonical" href="https://example.com/docs/">`},
		{"Model/pd/index.html", "<title>Element: pd — Model docs</title>"},
		{"Model/pd/index.html", `<link rel="canonical" href="https://example.com/docs/Model/pd/">`},
	} {
		if c.file == "index.html" && !bytes.Contains(landing, []byte(c.html)) {
			t.Errorf("landing lacks %q:\n%s", c.html, landing)
		}
		if c.file != "index.html" && !bytes.Contains(page, []byte(c.html)) {
			t.Errorf("pd's page lacks %q:\n%s", c.html, page)
		}
	}
}

// The sysml-publish.toml of the working directory is read without -config
// (CFG-1), and a flag the user gave wins over the file (CFG-3).
func TestConfigDefaultNameAndFlagsWin(t *testing.T) {
	model := fixtureAbs(t, "smoke")
	dir := t.TempDir()
	cfgOut := filepath.Join(dir, "fromconfig")
	flagOut := filepath.Join(dir, "fromflag")
	// Both folders in the file are wrong: the run only works when the
	// flags won.
	writeConfig(t, dir, fmt.Sprintf(
		"[source]\ndir = %q\n[site]\ndir = %q\n", "no-such-model", cfgOut))
	t.Chdir(dir)
	var log bytes.Buffer
	args := []string{"-model", model, "-out", flagOut}
	if code := run(t.Context(), args, &log); code != exitOK {
		t.Fatalf("exit %d, want %d; log:\n%s", code, exitOK, log.String())
	}
	if _, err := os.Stat(filepath.Join(flagOut, "index.html")); err != nil {
		t.Errorf("the -out folder was not used: %v", err)
	}
	if _, err := os.Stat(cfgOut); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the config's site folder was written despite -out: %v", err)
	}
}

// A -config file that does not exist is a usage error (CFG-1); with no
// -config and no default file, the run goes on.
func TestConfigMissing(t *testing.T) {
	absent := filepath.Join(t.TempDir(), "absent.toml")
	var log bytes.Buffer
	if code := run(context.Background(), []string{"-config", absent}, &log); code != exitUsage {
		t.Errorf("exit %d, want %d; log:\n%s", code, exitUsage, log.String())
	}
	if !strings.Contains(log.String(), "reading the config file") {
		t.Errorf("log lacks the missing file:\n%s", log.String())
	}
	// A broken default file is an error too, named as a config problem.
	dir := t.TempDir()
	writeConfig(t, dir, "[site]\nbogus = 1\n")
	t.Chdir(dir)
	log.Reset()
	if code := run(context.Background(), nil, &log); code != exitUsage {
		t.Errorf("exit %d, want %d; log:\n%s", code, exitUsage, log.String())
	}
	if !strings.Contains(log.String(), DefaultConfigFile) {
		t.Errorf("log does not name %s:\n%s", DefaultConfigFile, log.String())
	}
}

func TestUsageErrors(t *testing.T) {
	for _, args := range [][]string{
		{"-log-format", "xml"},
		{"extra-argument"},
		{"-no-such-flag"},
		// Views are rendered in process: the sysml command's flag is gone,
		// not accepted and ignored (MC-21).
		{"-sysml", "/usr/bin/sysml"},
	} {
		var log bytes.Buffer
		if code := run(context.Background(), args, &log); code != exitUsage {
			t.Errorf("%v: exit %d, want %d", args, code, exitUsage)
		}
	}
}

func TestInterruptedRunWritesNothing(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out := filepath.Join(t.TempDir(), "site")
	var log bytes.Buffer
	if code := run(ctx, []string{"-model", fixture("smoke"), "-out", out}, &log); code != exitInterrupted {
		t.Fatalf("exit %d, want %d; log:\n%s", code, exitInterrupted, log.String())
	}
	if !strings.Contains(log.String(), "interrupted; nothing written") {
		t.Errorf("log lacks the interruption:\n%s", log.String())
	}
	if _, err := os.Stat(out); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("site folder created after an interrupted run: %v", err)
	}
}

// -dot names the program that lays a diagram out (CLI-8); a view that needs
// no program renders without one. A program that cannot be run fails the
// macro, not the run (MC-19). The fixture's tables are written from the
// rendering's rows, so they render with no program at all.
func TestViewPrograms(t *testing.T) {
	out := filepath.Join(t.TempDir(), "site")
	var log bytes.Buffer
	args := []string{"-model", fixture("views"), "-out", out, "-dot", "/nonexistent/dot"}
	if code := run(t.Context(), args, &log); code != exitOK {
		t.Fatalf("exit %d, want %d; log:\n%s", code, exitOK, log.String())
	}
	file := filepath.Join(fixture("views"), "views.sysml")
	// The rest of a missing program's message is Go's (os/exec).
	missing := `cannot run dot: `
	want := []string{
		`level=WARN msg="` + file + `:14: V::sys: sysml-view::interfaces[]: ` + missing,
		`level=WARN msg="` + file + `:48: V::notAView: sysml-view::sys[]: V::sys is a PartUsage, not a view"`,
	}
	var got []string
	for _, line := range strings.Split(log.String(), "\n") {
		if strings.HasPrefix(line, "level=WARN ") {
			got = append(got, line)
		}
	}
	ok := len(got) == len(want)
	for i := 0; ok && i < len(want); i++ {
		ok = got[i] == want[i] || strings.HasSuffix(want[i], missing) && strings.HasPrefix(got[i], want[i])
	}
	if !ok {
		t.Errorf("warnings:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	for _, page := range []string{"V/sys/index.html", "V/partsTable/index.html"} {
		b, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(page)))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(b, []byte("<table")) {
			t.Errorf("%s shows no table", page)
		}
	}
}
