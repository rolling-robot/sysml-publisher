package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func fixture(name string) string { return filepath.Join("..", "..", "testdata", name) }

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

func TestUsageErrors(t *testing.T) {
	for _, args := range [][]string{
		{"-log-format", "xml"},
		{"extra-argument"},
		{"-no-such-flag"},
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
