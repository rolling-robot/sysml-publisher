// The config file (CFG-1..CFG-5): an optional TOML file naming the model and
// site folders and the site's title and base URL. A value the user gave on
// the command line wins over the file; the file wins over the defaults
// (CFG-3, CLI-2, CLI-3).
package main

import (
	"fmt"
	"maps"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
)

// DefaultConfigFile is read when -config is not given; it may be absent
// (CFG-1).
const DefaultConfigFile = "sysml-publish.toml"

// config is what the config file may say; every field is optional ("" when
// unset). The keys are strict (CFG-2): anything else is an error, so a
// misspelled key is reported instead of silently ignored.
type config struct {
	Source sourceConfig
	Site   siteConfig
}

type sourceConfig struct {
	Dir string // the model folder (CLI-2)
}

type siteConfig struct {
	Dir     string // the site folder (CLI-3)
	Title   string // the site's title (CFG-4)
	BaseURL string // the URL the site is published under (CFG-5)
}

// allowed keys of the config file, by section (CFG-2).
var allowedKeys = map[string][]string{
	"source": {"dir"},
	"site":   {"dir", "title", "baseURL"},
}

// loadConfig reads the TOML config file at path. An explicit path (-config)
// must exist (mustExist); the default file may be absent, which is no config
// at all. Paths in the file are relative to the working directory, like the
// flags' values are.
func loadConfig(path string, mustExist bool) (*config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && !mustExist {
			return &config{}, nil
		}
		return nil, fmt.Errorf("reading the config file: %w", err)
	}
	c := &config{}
	// Decoded twice: into a map to check the keys strictly (the decoder
	// also matches them ignoring case), then into the struct for the
	// values, whose errors carry the file position.
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for _, top := range slices.Sorted(maps.Keys(raw)) {
		want, known := allowedKeys[top]
		if !known {
			return nil, fmt.Errorf("%s: unknown key %q: want a [source] or [site] section", path, top)
		}
		section, ok := raw[top].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s: %s is not a table of settings", path, top)
		}
		for _, key := range slices.Sorted(maps.Keys(section)) {
			if !slices.Contains(want, key) {
				return nil, fmt.Errorf("%s: unknown key %q: want one of %s",
					path, top+"."+key, strings.Join(want, ", "))
			}
		}
	}
	if err := toml.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	// baseURL that is set must be absolute (CFG-5); set to "", it is not.
	baseURLSet := false
	if site, ok := raw["site"].(map[string]any); ok {
		_, baseURLSet = site["baseURL"]
	}
	if baseURLSet {
		u, err := url.Parse(c.Site.BaseURL)
		if err != nil || !u.IsAbs() {
			return nil, fmt.Errorf("%s: site.baseURL %q is not an absolute URL", path, c.Site.BaseURL)
		}
	}
	return c, nil
}
