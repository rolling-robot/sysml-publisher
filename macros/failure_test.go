package macros

import (
	"context"
	"html"
	"strings"
	"testing"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
)

// recorded holds the warnings of each asciidoctorg instance set up by setup
// or setupFixture, newest last. Tests do not run in parallel (AGENTS.md §8).
var recorded = map[*asciidoctor.Asciidoctor]*[]Warning{}

// record is the Warn of ad: it keeps the warnings in recorded.
func record(t *testing.T, ad *asciidoctor.Asciidoctor) Warn {
	ws := &[]Warning{}
	recorded[ad] = ws
	t.Cleanup(func() { delete(recorded, ad) })
	return func(w Warning) { *ws = append(*ws, w) }
}

// mustFail converts src, in which one macro fails (MC-19): the conversion
// succeeds, exactly one warning is reported, its line (without a location:
// the renderer adds it) is want, and the placeholder in the output shows
// the macro and the problem. It returns the output.
func mustFail(t *testing.T, ad *asciidoctor.Asciidoctor, src string, attrs asciidoctor.Attributes, want string) string {
	t.Helper()
	ws := recorded[ad]
	*ws = nil
	out, err := ad.Convert(src, &asciidoctor.Options{Attributes: attrs})
	if err != nil {
		t.Fatalf("conversion failed: %v; want a placeholder and a warning", err)
	}
	if len(*ws) != 1 {
		t.Fatalf("warnings %q, want one", *ws)
	}
	w := (*ws)[0]
	if got := w.String(); got != want {
		t.Errorf("warning\n got %q\nwant %q", got, want)
	}
	if strings.Contains(w.Message, w.Macro) || strings.Contains(w.Message, " on the page of ") {
		t.Errorf("message %q repeats the macro or the page", w.Message)
	}
	text := html.EscapeString(w.Macro + ": " + w.Message)
	inline := `<span class="` + FailedRole + `">` + text + `</span>`
	block := `<div class="` + FailedRole + `"><p>` + text + `</p></div>`
	if !strings.Contains(out, inline) && !strings.Contains(out, block) {
		t.Errorf("output %q lacks the placeholder %q or %q", out, inline, block)
	}
	return out
}

// A failing macro leaves a placeholder where it was written, inline in its
// sentence or as a block of its own, and the rest of the representation is
// converted (MC-19).
func TestFailurePlaceholder(t *testing.T) {
	tr, ad := setup(t)
	page := Attributes(pageAt(t, tr, "M/pd"))
	t.Run("inline", func(t *testing.T) {
		out := mustFail(t, ad, "Before sysml-name:Nothing[] after.", page,
			`M::pd: sysml-name:Nothing[]: the model has no element named "Nothing"`)
		want := `<p>Before <span class="sysml-failed">sysml-name:Nothing[]: the model has no element named &#34;Nothing&#34;</span> after.</p>`
		if !strings.Contains(out, want) {
			t.Errorf("output %q lacks %q", out, want)
		}
	})
	t.Run("block", func(t *testing.T) {
		out := mustFail(t, ad, "Before.\n\nsysml-doc::Nothing[]\n\nAfter.", page,
			`M::pd: sysml-doc::Nothing[]: the model has no element named "Nothing"`)
		for _, want := range []string{
			"<p>Before.</p>",
			`<div class="sysml-failed"><p>sysml-doc::Nothing[]: the model has no element named &#34;Nothing&#34;</p></div>`,
			"<p>After.</p>",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("output %q lacks %q", out, want)
			}
		}
	})
	t.Run("escaped", func(t *testing.T) {
		out := mustFail(t, ad, "sysml-name:M::'x<y'[]", page,
			`M::pd: sysml-name:M::'x<y'[]: the model has no element named "M::'x<y'"`)
		if strings.Contains(out, "x<y") {
			t.Errorf("output %q holds the target unescaped", out)
		}
	})
}

// Every failing macro is reported, not just the first (MC-19). Asciidoctor
// runs block macros while parsing and inline macros afterwards, so the
// block's warning comes first.
func TestEveryFailureReported(t *testing.T) {
	tr, ad := setup(t)
	ws := recorded[ad]
	src := "sysml-name:Nothing[] and sysml-short-name:Nobody[].\n\nsysml-doc::Nowhere[]"
	if _, err := ad.Convert(src, &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, "M/pd"))}); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, w := range *ws {
		got = append(got, w.String())
	}
	want := []string{
		`M::pd: sysml-doc::Nowhere[]: the model has no element named "Nowhere"`,
		`M::pd: sysml-name:Nothing[]: the model has no element named "Nothing"`,
		`M::pd: sysml-short-name:Nobody[]: the model has no element named "Nobody"`,
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("warnings\n got %q\nwant %q", got, want)
	}
}

// A cancelled rendering is not a macro failure: the conversion stops with
// an error instead of leaving a placeholder.
func TestCancelledIsNoFailure(t *testing.T) {
	tr, _ := setup(t)
	ad, err := asciidoctor.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { ad.Close() })
	ctx, cancel := context.WithCancel(t.Context())
	if err := Register(ctx, ad, tr, record(t, ad)); err != nil {
		t.Fatal(err)
	}
	cancel()
	_, err = ad.Convert("sysml-value:mass[]", &asciidoctor.Options{Attributes: Attributes(pageAt(t, tr, "M/pd"))})
	if err == nil {
		t.Error("conversion succeeded after cancellation")
	}
	if ws := *recorded[ad]; len(ws) != 0 {
		t.Errorf("cancellation reported as warnings: %q", ws)
	}
}
