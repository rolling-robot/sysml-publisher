package site

import (
	"os"
	"strconv"
	"strings"

	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

// sources finds where representations and the macros in them are written in
// the model files, for warnings (MC-19). The model gives no source positions
// (AGENTS.md §13, 15), so a representation is found by its text: the first
// place in the files, in parse order, where its lines appear in sequence.
// OpenSysML strips a body's indentation, so lines are compared without the
// whitespace around them.
type sources struct {
	files []string
	lines map[string][]string // a file's lines, trimmed; nil when unreadable
	start map[string]location // by representation id; the zero location when not found
}

type location struct {
	file string
	line int // 1-based; 0 when unknown
}

func newSources(files []string) *sources {
	return &sources{files: files, lines: map[string][]string{}, start: map[string]location{}}
}

// locate is where the macro written as macro is in rep, file:line: the line
// of its first use in the body, or the body's first line when the macro is
// not in it (a macro in a doc that sysml-doc inserted). It is empty when the
// representation is not found.
func (s *sources) locate(rep tree.Rep, macro string) string {
	start, ok := s.start[rep.ID]
	if !ok {
		start = s.find(rep.Body)
		s.start[rep.ID] = start
	}
	if start.line == 0 {
		return ""
	}
	line := start.line
	// The macro as written up to its attributes: its name and target.
	prefix := macro
	if i := strings.LastIndex(prefix, "["); i >= 0 {
		prefix = prefix[:i+1]
	}
	for i, l := range bodyLines(rep.Body) {
		if strings.Contains(l, prefix) {
			line += i
			break
		}
	}
	return start.file + ":" + strconv.Itoa(line)
}

// find is where body starts in the model files.
func (s *sources) find(body string) location {
	want := bodyLines(body)
	if len(want) == 0 {
		return location{}
	}
	for _, f := range s.files {
		lines, ok := s.lines[f]
		if !ok {
			if data, err := os.ReadFile(f); err == nil {
				lines = strings.Split(string(data), "\n")
				for i := range lines {
					lines[i] = strings.TrimSpace(lines[i])
				}
			}
			s.lines[f] = lines
		}
		for i := range lines {
			if matches(lines, i, want) {
				return location{file: f, line: i + 1}
			}
		}
	}
	return location{}
}

// bodyLines are the lines of a representation body, trimmed, from its first
// non-blank line.
func bodyLines(body string) []string {
	lines := strings.Split(strings.TrimRight(body, " \t\r\n"), "\n")
	for len(lines) > 0 && strings.TrimSpace(lines[0]) == "" {
		lines = lines[1:]
	}
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	return lines
}

// matches reports whether the body lines are the comment body written at
// src[i:]: the body opens right after a /* (on its first line, or ending the
// last non-blank line before it), closes right before a */ (on its last
// line, or starting the next non-blank line), and every line between is the
// body's, whitespace aside. Text elsewhere, such as a // note quoting a
// macro, is not a body.
func matches(src []string, i int, body []string) bool {
	if i+len(body) > len(src) {
		return false
	}
	last := len(body) - 1
	for j, want := range body {
		line := src[i+j]
		if j == 0 {
			if k := strings.Index(line, "/*"); k >= 0 {
				line = strings.TrimSpace(line[k+2:])
			} else if !strings.HasSuffix(nonBlank(src, i, -1), "/*") {
				return false
			}
		}
		if j == last {
			if k := strings.LastIndex(line, "*/"); k >= 0 {
				line = strings.TrimSpace(line[:k])
			} else if !strings.HasPrefix(nonBlank(src, i+j, +1), "*/") {
				return false
			}
		}
		if line != want {
			return false
		}
	}
	return true
}

// nonBlank is the first non-blank line of src from i in direction step
// (-1 before, +1 after), not counting src[i]; empty when there is none.
func nonBlank(src []string, i, step int) string {
	for k := i + step; k >= 0 && k < len(src); k += step {
		if src[k] != "" {
			return src[k]
		}
	}
	return ""
}
