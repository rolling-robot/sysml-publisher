package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeConfig writes toml text to a file in dir and returns its path.
func writeConfig(t *testing.T, dir, text string) string {
	t.Helper()
	path := filepath.Join(dir, "sysml-publish.toml")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// The file's keys are the five of CFG-2; anything else, and a baseURL that
// is not absolute, is an error (CFG-5). OpenSysML wording is not pinned
// here; BurntSushi's own type-error text is not either.
func TestLoadConfig(t *testing.T) {
	cases := []struct {
		name    string
		toml    string
		want    config
		wantErr string // empty: no error; a prefix: its beginning
	}{
		{
			name: "sections",
			toml: "[source]\ndir = \"m\"\n[site]\ndir = \"s\"\ntitle = \"T\"\nbaseURL = \"https://example.com/docs/\"",
			want: config{Source: sourceConfig{Dir: "m"}, Site: siteConfig{Dir: "s", Title: "T", BaseURL: "https://example.com/docs/"}},
		},
		{
			name: "dotted keys",
			toml: "source.dir = \"m\"\nsite.title = \"T\"\nsite.baseURL = \"https://example.com\"",
			want: config{Source: sourceConfig{Dir: "m"}, Site: siteConfig{Title: "T", BaseURL: "https://example.com"}},
		},
		{
			name: "empty file",
		},
		{
			name:    "unknown key",
			toml:    "[site]\nbaseurl = \"https://example.com\"",
			wantErr: `unknown key "site.baseurl"`,
		},
		{
			name:    "unknown section",
			toml:    "[bogus]\ndir = \"x\"",
			wantErr: `unknown key "bogus"`,
		},
		{
			name:    "section is not a table",
			toml:    "source = \"m\"",
			wantErr: "source is not a table of settings",
		},
		{
			name:    "value of the wrong type",
			toml:    "[site]\ntitle = 3",
			wantErr: "site.title",
		},
		{
			name:    "relative baseURL",
			toml:    "site.baseURL = \"example.com/docs\"",
			wantErr: `site.baseURL "example.com/docs" is not an absolute URL`,
		},
		{
			name:    "empty baseURL",
			toml:    "site.baseURL = \"\"",
			wantErr: `site.baseURL "" is not an absolute URL`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := writeConfig(t, t.TempDir(), c.toml)
			got, err := loadConfig(path, true)
			if c.wantErr != "" {
				if err == nil {
					t.Fatalf("loadConfig = %+v, want an error starting %q", got, c.wantErr)
				}
				if !strings.HasPrefix(err.Error(), c.wantErr) && !strings.Contains(err.Error(), c.wantErr) {
					t.Errorf("error = %q, want it to contain %q", err, c.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("loadConfig: %v", err)
			}
			if got.Source.Dir != c.want.Source.Dir || got.Site != c.want.Site {
				t.Errorf("loadConfig = %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestLoadConfigMissing(t *testing.T) {
	absent := filepath.Join(t.TempDir(), "absent.toml")
	// The default file may be absent (CFG-1): an empty config results.
	c, err := loadConfig(absent, false)
	if err != nil {
		t.Fatalf("loadConfig(missing default): %v", err)
	}
	if c.Source.Dir != "" || c.Site.Dir != "" {
		t.Errorf("missing default file gave %+v, want an empty config", c)
	}
	// An explicit -config must exist.
	if _, err := loadConfig(absent, true); err == nil || !strings.Contains(err.Error(), "reading the config file") {
		t.Errorf("loadConfig(missing explicit) = %v, want a reading error", err)
	}
}
