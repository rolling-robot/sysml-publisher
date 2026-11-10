package site

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

// Warnings name where a macro is written (MC-19). The model gives no source
// positions, so representations are found by their text between /* and */.
func TestLocate(t *testing.T) {
	dir := t.TempDir()
	note := filepath.Join(dir, "a.sysml")
	src := `// A note quoting sysml-name:X[] is no body.
package P {
    part a {
        rep language "adoc" /* sysml-name:X[] */
    }
    part b {
        rep language "adoc"
            /*

            First line.

            Then sysml-value:m[] and sysml-value:m[].
            */
    }
}
`
	if err := os.WriteFile(note, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	s := newSources([]string{filepath.Join(dir, "missing.sysml"), note})
	multi := "First line.\n\nThen sysml-value:m[] and sysml-value:m[].\n"
	cases := []struct{ name, body, macro, want string }{
		{"one-line body after rep, not the // note", "sysml-name:X[] ", "sysml-name:X[]", note + ":4"},
		{"macro on a later line of the body", multi, "sysml-value:m[]", note + ":12"},
		{"macro not in the body: the body's first line", multi, "sysml-doc::D[]", note + ":10"},
		{"attributes written differently: found by name and target", "sysml-name:X[] ", "sysml-name:X[other]", note + ":4"},
		{"body not in the files", "Nowhere.", "sysml-name:X[]", ""},
	}
	for i, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rep := tree.Rep{ID: string(rune('a' + i)), Body: c.body}
			if got := s.locate(rep, c.macro); got != c.want {
				t.Errorf("locate = %q, want %q", got, c.want)
			}
		})
	}
}
