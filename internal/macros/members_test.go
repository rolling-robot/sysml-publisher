package macros

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/model"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

func TestMembers(t *testing.T) {
	m, err := model.Folder{Dir: filepath.Join("..", "..", "testdata", "members")}.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	tr := tree.Build(m)
	ad, err := asciidoctor.New()
	if err != nil {
		t.Fatal(err)
	}
	defer ad.Close()
	if err := Register(context.Background(), ad, tr, nil); err != nil {
		t.Fatal(err)
	}
	convert := func(dir string) string {
		t.Helper()
		out, err := ad.Convert("sysml-members::[]\n\nAfter.", &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, dir))})
		if err != nil {
			t.Fatalf("%s: %v", dir, err)
		}
		return out
	}

	out := convert("M/pd")
	// In declaration order; docs and representations left out; a member
	// with a page is linked; a name keeps its markup characters.
	items := []string{
		"<p>mass</p>",
		"<p>a*b*&#43;&#43;&#43;c</p>",
		`<p><a href="sub/index.html">sub</a></p>`,
		"<p>Unnamed</p>",
	}
	last := -1
	for _, item := range items {
		i := strings.Index(out, item)
		if i < 0 {
			t.Errorf("M/pd: output %q lacks item %q", out, item)
			continue
		}
		if i < last {
			t.Errorf("M/pd: item %q out of order", item)
		}
		last = i
	}
	if !strings.Contains(out, `<div class="ulist sysml-members">`) {
		t.Errorf("M/pd: list lacks the sysml-members role: %q", out)
	}
	if strings.Contains(out, "documentation") || strings.Contains(out, "Members:") {
		t.Errorf("M/pd: annotations listed: %q", out)
	}

	// The page's element (MC-2): leaf's own member q, not pd's; the
	// metadata usage @tag is left out.
	out = convert("M/leaf")
	if !strings.Contains(out, "<p>q</p>") || strings.Contains(out, "mass") || strings.Contains(out, "tag") {
		t.Errorf("M/leaf: want only q, got %q", out)
	}

	// sub owns only a representation: nothing to list, the line is dropped.
	out = convert("M/pd/sub")
	if strings.Contains(out, "ulist") || !strings.Contains(out, "<p>After.</p>") {
		t.Errorf("M/pd/sub: want no list, got %q", out)
	}
}
