package macros

import (
	"context"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/model"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

func setupFixture(t *testing.T, fixture string) (*tree.Tree, *asciidoctor.Asciidoctor) {
	t.Helper()
	m, err := model.Folder{Dir: filepath.Join("..", "..", "testdata", fixture)}.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	tr := tree.Build(m)
	ad, err := asciidoctor.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ad.Close() })
	if err := Register(ad, tr, nil); err != nil {
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

// The expectations are the // notes in testdata/metadata/metadata.sysml.
func TestMetadata(t *testing.T) {
	tr, ad := setupFixture(t, "metadata")
	src := "sysml-metadata::Info[]\n\nsysml-metadata::Derived[]"
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

func TestMetadataByQualifiedName(t *testing.T) {
	tr, ad := setupFixture(t, "metadata")
	out, err := ad.Convert("sysml-metadata::Meta::Derived[]", &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, "Meta/specialized"))})
	if err != nil {
		t.Fatal(err)
	}
	if got := tables(out)["Derived"]; len(got) != 2 {
		t.Errorf("rows = %v, want 2", got)
	}
}

func TestMetadataErrors(t *testing.T) {
	tr, ad := setupFixture(t, "metadata")
	page := Attributes(pageAt(t, tr, "Meta/bound"))
	for src, want := range map[string]string{
		"sysml-metadata::Missing[]": `no metadata def "Missing"`,
		"sysml-metadata::[]":        "no metadata def named",
	} {
		_, err := ad.Convert(src, &asciidoctor.Options{Attributes: page})
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v, want one containing %q", src, err, want)
		}
	}
}
