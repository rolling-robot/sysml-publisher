package macros

import (
	"context"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"testing"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/model"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

func setupFixture(t *testing.T, fixture string) (*tree.Tree, *asciidoctor.Asciidoctor) {
	t.Helper()
	m, err := model.Folder{Dir: filepath.Join("..", "testdata", fixture)}.Load(t.Context())
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
	if err := Register(context.Background(), ad, tr, record(t, ad)); err != nil {
		t.Fatal(err)
	}
	return tr, ad
}

var (
	captionRx = regexp.MustCompile(`<caption class="title">(.*?)</caption>`)
	cellRx    = regexp.MustCompile(`<td [^>]*><p class="tableblock">(.*?)</p></td>`)
)

// tables reads the converted tables as caption → "attribute = value" rows.
func tables(out string) map[string][]string {
	got := map[string][]string{}
	parts := captionRx.FindAllStringSubmatchIndex(out, -1)
	for i, p := range parts {
		end := len(out)
		if i+1 < len(parts) {
			end = parts[i+1][0]
		}
		caption := out[p[2]:p[3]]
		cells := cellRx.FindAllStringSubmatch(out[p[1]:end], -1)
		rows := []string{}
		for j := 0; j+1 < len(cells); j += 2 {
			rows = append(rows, cells[j][1]+" = "+cells[j+1][1])
		}
		got[caption] = rows
	}
	return got
}

const noValueHTML = `<span class="no-value">(no value)</span>`

// The expectations are the // notes in testdata/metadata-table/metadata-table.sysml.
func TestMetadataTable(t *testing.T) {
	tr, ad := setupFixture(t, "metadata-table")
	src := "sysml-metadata-table::Info[]\n\nsysml-metadata-table::Derived[]"
	cases := map[string]map[string][]string{
		"Meta/bound": {"Info": {
			"owner = pete", "count = 7", "ratio = 0.5", "ok = true",
			"level = Level::low", "sum = 1 &#43; 2", "note = " + noValueHTML,
		}},
		"Meta/overridden": {"Info": {
			"owner = ann", "count = 0", "ratio = 0.5", "ok = false",
			"level = Level::high", "sum = 10", "note = checked",
		}},
		"Meta/defaults": {"Info": {
			"owner = nobody", "count = 3", "ratio = 0.5", "ok = true",
			"level = Level::low", "sum = 1 &#43; 2", "note = " + noValueHTML,
		}},
		"Meta/specialized": {"Derived": {"extra = x", "origin = here"}},
		"Meta/plain":       {},
	}
	for dir, want := range cases {
		out, err := ad.Convert(src, &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, dir))})
		if err != nil {
			t.Fatalf("%s: %v", dir, err)
		}
		if got := tables(out); !reflect.DeepEqual(got, want) {
			t.Errorf("%s:\n got %v\nwant %v", dir, got, want)
		}
	}
}

func TestMetadataTableByQualifiedName(t *testing.T) {
	tr, ad := setupFixture(t, "metadata-table")
	out, err := ad.Convert("sysml-metadata-table::Meta::Derived[]", &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, "Meta/specialized"))})
	if err != nil {
		t.Fatal(err)
	}
	if got := tables(out)["Derived"]; len(got) != 2 {
		t.Errorf("rows = %v, want 2", got)
	}
}

// A metadata def is found by its short name too (<drv> Derived).
func TestMetadataTableByShortName(t *testing.T) {
	tr, ad := setupFixture(t, "metadata-table")
	out, err := ad.Convert("sysml-metadata-table::drv[]", &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, "Meta/specialized"))})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := tables(out)["Derived"], []string{"extra = x", "origin = here"}; !slices.Equal(got, want) {
		t.Errorf("rows = %q, want %q", got, want)
	}
}

func TestMetadataTableErrors(t *testing.T) {
	tr, ad := setupFixture(t, "metadata-table")
	page := Attributes(pageAt(t, tr, "Meta/bound"))
	cases := []struct{ name, src, want string }{
		{"unknown metadata def", "sysml-metadata-table::Missing[]",
			`Meta::bound: sysml-metadata-table::Missing[]: the model has no metadata def "Missing"`},
		{"no metadata def named", "sysml-metadata-table::[]",
			"Meta::bound: sysml-metadata-table::[]: no metadata def named; write sysml-metadata-table::<metadata def>[]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			mustFail(t, ad, c.src, page, c.want)
		})
	}
}
