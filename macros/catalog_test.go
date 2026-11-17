// Copyright 2026 Petr Mukhachev
// SPDX-License-Identifier: MIT

package macros

import (
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/model"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

// setupCatalog builds the page tree of the catalog fixture and an
// asciidoctorg instance with the macros registered.
func setupCatalog(t *testing.T) (*tree.Tree, *asciidoctor.Asciidoctor) {
	t.Helper()
	m, err := model.Folder{Dir: filepath.Join("..", "testdata", "catalog")}.Load(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { m.Close() })
	tr, err := tree.Build(t.Context(), m)
	if err != nil {
		t.Fatal(err)
	}
	ad, err := asciidoctor.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ad.Close() })
	if _, err := Register(t.Context(), ad, tr.Model.Model, tr, record(t, ad)); err != nil {
		t.Fatal(err)
	}
	return tr, ad
}

var (
	thRx = regexp.MustCompile(`<th class="tableblock[^"]*">(.*?)</th>`)
	tdRx = regexp.MustCompile(`<td class="tableblock[^"]*">(.*?)</td>`)
)

// catalogOf converts the catalog macro on the page at dir and returns the
// table as a header row and data rows; nil when the macro line was dropped.
func catalogOf(t *testing.T, ad *asciidoctor.Asciidoctor, tr *tree.Tree, dir string) (header []string, rows [][]string) {
	t.Helper()
	out, err := ad.Convert("sysml-catalog::[]", &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, dir).ID)})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `<table class="tableblock`) {
		return nil, nil
	}
	thMatches := thRx.FindAllStringSubmatch(out, -1)
	h := make([]string, len(thMatches))
	for i, m := range thMatches {
		h[i] = strings.TrimSpace(cellText(m[1]))
	}
	cells := tdRx.FindAllStringSubmatch(out, -1)
	cols := len(h)
	rows = nil
	for i := 0; i+cols <= len(cells); i += cols {
		row := make([]string, cols)
		for j := 0; j < cols; j++ {
			row[j] = strings.TrimSpace(cellText(cells[i+j][1]))
		}
		rows = append(rows, row)
	}
	return h, rows
}

// cellText strips the inner <p> and <span> wrappers and link markup to the
// visible text of a table cell.
func cellText(s string) string {
	s = regispp.ReplaceAllString(s, "")
	s = spanRx.ReplaceAllString(s, "$2")
	s = linkRx.ReplaceAllString(s, "$1")
	return strings.ReplaceAll(s, "</p>", "")
}

var (
	regispp = regexp.MustCompile(`<p class="tableblock">`)
	spanRx  = regexp.MustCompile(`(<span class="no-value">)(.*?)(</span>)`)
	linkRx  = regexp.MustCompile(`<a href="[^"]*">(.*?)</a>`)
)

// The expected catalogs are the // notes in testdata/catalog/catalog.sysml.
func TestCatalog(t *testing.T) {
	tr, ad := setupCatalog(t)
	cases := map[string]struct {
		header []string
		rows   [][]string
	}{
		// protocolCatalog: expose Protocols::*, filter @LogicalProtocolMetadata.
		"Catalog/protocolCatalog": {
			header: []string{"Name", "priority", "Description"},
			rows: [][]string{
				{"tcp", "1", "alpha"},
				{"udp", "(no value)", "none"},
			},
		},
		// protocolCatalogDeep: expose Protocols::** reaches inner.
		"Catalog/protocolCatalogDeep": {
			header: []string{"Name", "priority", "Description"},
			rows: [][]string{
				{"tcp", "1", "alpha"},
				{"udp", "(no value)", "none"},
				{"inner", "(no value)", "beta"},
			},
		},
		// emptyCatalog: nothing exposed carries the metadata; line dropped.
		"Catalog/emptyCatalog": {header: nil, rows: nil},
	}
	for dir, want := range cases {
		h, rows := catalogOf(t, ad, tr, dir)
		if !reflect.DeepEqual(h, want.header) {
			t.Errorf("%s: header %v, want %v", dir, h, want.header)
		}
		if !reflect.DeepEqual(rows, want.rows) {
			t.Errorf("%s: rows %v, want %v", dir, rows, want.rows)
		}
	}
}

// The Name cell of an element with a page links to it; one without a page
// does not (MC-18).
func TestCatalogNameLinks(t *testing.T) {
	tr, ad := setupCatalog(t)
	dir := "Catalog/protocolCatalog"
	out, err := ad.Convert("sysml-catalog::[]", &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, dir).ID)})
	if err != nil {
		t.Fatal(err)
	}
	// tcp has a page (documented), udp does not.
	if !strings.Contains(out, `<a href="../Protocols/tcp/index.html">tcp</a>`) {
		t.Errorf("output lacks a link to tcp's page: %s", out)
	}
	if strings.Contains(out, `<a href`) && strings.Contains(cellsContaining(out, "udp"), "<a ") {
		t.Errorf("udp has no page but its Name cell links: %s", cellsContaining(out, "udp"))
	}
}

// cellsContaining is the <td> source whose text contains want, "" if none.
func cellsContaining(out, want string) string {
	for _, m := range tdRx.FindAllStringSubmatch(out, -1) {
		if strings.Contains(cellText(m[1]), want) {
			return m[1]
		}
	}
	return ""
}

// The failure cases (MC-19): a page that is not a view, a view with no
// expose and a view with no filter each leave a placeholder and a warning.
func TestCatalogErrors(t *testing.T) {
	tr, ad := setupCatalog(t)
	cases := []struct{ name, dir, want string }{
		{"page is not a view", "Catalog/notAView",
			"Catalog::notAView: sysml-catalog::[]: the page's element Catalog::notAView is a PartUsage, not a view; a catalog is a view's exposed elements"},
		{"no expose", "Catalog/noExpose",
			"Catalog::noExpose: sysml-catalog::[]: this view exposes nothing"},
		{"no filter", "Catalog/noFilter",
			"Catalog::noFilter: sysml-catalog::[]: this view has no element filter"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mustFail(t, ad, "sysml-catalog::[]", Attributes(pageAt(t, tr, c.dir).ID), c.want)
		})
	}
}
