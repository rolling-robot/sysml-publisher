// Package tree decides which elements of a model get pages, what each page
// shows, and where it is written (requirements PG-1..7, PT-1..6).
package tree

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/rolling-robot/sysml-publisher/internal/model"
)

// UnnamedTitle is the title of the page of an element without a name (PG-5).
const UnnamedTitle = "Unnamed"

// Rep is an AsciiDoc textual representation.
type Rep struct {
	ID    string
	Owner string // @id of the element that owns it
	Body  string
}

// Node is one page of the site, or the site root.
type Node struct {
	ID string // @id; empty for the root
	// Title is the element's name, UnnamedTitle, or empty for the root. The
	// page title shown to readers is decided when the page is rendered.
	Title string
	// Unnamed is set when the element has no name (PG-5, PT-6).
	Unnamed bool
	// Name is how the element is named in messages: its qualified name, or
	// its positional name when it has none.
	Name string
	// Dir is the page's folder relative to the site, slash-separated; empty
	// for the root. The page itself is Dir + "/index.html".
	Dir string
	// Own are the representations the element owns, in declaration order;
	// Inherited are those it inherits, nearest specialization first (PG-7).
	Own       []Rep
	Inherited []Rep
	// Docs are the bodies of the element's own doc comments, in declaration
	// order (MC-10).
	Docs []string
	// ShortName is the element's short name (the <a> of part <a> aaaa), ""
	// when it has none (MC-11).
	ShortName string
	// Members are the element's owned members, in declaration order, without
	// the annotations: docs, comments, representations, metadata (MC-12).
	Members  []Member
	Parent   *Node
	Children []*Node
}

// Documented reports whether the element has representations of its own or
// inherited ones; other nodes are empty ancestor pages (PG-6).
func (n *Node) Documented() bool { return len(n.Own)+len(n.Inherited) > 0 }

// Member is an owned member of a page's element.
type Member struct {
	ID    string
	Title string // its name, or UnnamedTitle
}

// Tree is the page tree of a model.
type Tree struct {
	// Model is the model the tree was built from, for macros that read
	// elements without pages.
	Model *model.Model
	Root  *Node
	// Pages are every node but the root, in pre-order.
	Pages []*Node
	// ByID finds a page by element id.
	ByID map[string]*Node
	// Warnings are skipped representations (PG-4) and unnamed pages (PG-5).
	Warnings []string
}

// Build computes the page tree of m.
func Build(m *model.Model) *Tree {
	b := &builder{m: m, t: &Tree{Model: m, ByID: map[string]*Node{}}}
	b.collectReps()
	b.collectDocs()
	b.collectGenerals()
	b.t.Root = &Node{}
	for _, id := range m.Order {
		e := m.Elements[id]
		if e.Type() == "Namespace" && e.Ref("owningRelationship") == "" {
			b.addChildren(b.t.Root, e)
		}
	}
	b.walk(b.t.Root)
	return b.t
}

type builder struct {
	m        *model.Model
	t        *Tree
	reps     map[string][]Rep    // owner id → its AsciiDoc representations
	docs     map[string][]string // owner id → its doc comment bodies
	generals map[string][]string // specific id → general ids, in order
}

// isAsciiDoc reports whether a representation's language is AsciiDoc.
func isAsciiDoc(language string) bool {
	switch strings.ToLower(language) {
	case "adoc", "asciidoc":
		return true
	}
	return false
}

func (b *builder) collectReps() {
	b.reps = map[string][]Rep{}
	for _, id := range b.m.Order {
		e := b.m.Elements[id]
		if e.Type() != "TextualRepresentation" {
			continue
		}
		owner := e.Ref("owner")
		if !isAsciiDoc(e.Str("language")) {
			b.t.Warnings = append(b.t.Warnings, fmt.Sprintf(
				"%s: skipped representation in language %q", b.name(owner), e.Str("language")))
			continue
		}
		b.reps[owner] = append(b.reps[owner], Rep{ID: id, Owner: owner, Body: e.Str("body")})
	}
}

func (b *builder) collectDocs() {
	b.docs = map[string][]string{}
	for _, id := range b.m.Order {
		if e := b.m.Elements[id]; e.Type() == "Documentation" {
			owner := e.Ref("owner")
			b.docs[owner] = append(b.docs[owner], strings.TrimSpace(e.Str("body")))
		}
	}
}

// collectGenerals reads every specialization relationship (typing,
// subclassification, subsetting, redefinition): each names a specific and a
// general element.
func (b *builder) collectGenerals() {
	b.generals = map[string][]string{}
	for _, id := range b.m.Order {
		e := b.m.Elements[id]
		specific, general := e.Ref("specific"), e.Ref("general")
		if specific == "" || general == "" || specific == general {
			continue
		}
		b.generals[specific] = append(b.generals[specific], general)
	}
}

// inherited are the representations owned by the elements id specializes,
// directly or transitively, nearest first, each once.
func (b *builder) inherited(id string) []Rep {
	t := b.m.Elements[id].Type()
	if !strings.HasSuffix(t, "Usage") && !strings.HasSuffix(t, "Definition") {
		return nil
	}
	var out []Rep
	seenRep := map[string]bool{}
	visited := map[string]bool{id: true}
	queue := append([]string(nil), b.generals[id]...)
	for len(queue) > 0 {
		g := queue[0]
		queue = queue[1:]
		if visited[g] {
			continue
		}
		visited[g] = true
		for _, r := range b.reps[g] {
			if !seenRep[r.ID] {
				seenRep[r.ID] = true
				out = append(out, r)
			}
		}
		queue = append(queue, b.generals[g]...)
	}
	return out
}

// addChildren adds a node for each owned member of e.
func (b *builder) addChildren(parent *Node, e model.Element) {
	for _, id := range e.Refs("ownedMember") {
		child, ok := b.m.Elements[id]
		if !ok {
			continue
		}
		title := b.declaredTitle(id)
		n := &Node{
			ID:        id,
			Title:     title,
			Unnamed:   title == "",
			Name:      b.name(id),
			Own:       b.reps[id],
			Inherited: b.inherited(id),
			Docs:      b.docs[id],
			ShortName: b.shortName(id),
			Members:   b.members(child),
			Parent:    parent,
		}
		if n.Unnamed {
			n.Title = UnnamedTitle
		}
		b.addChildren(n, child)
		if n.Documented() || len(n.Children) > 0 {
			parent.Children = append(parent.Children, n)
		}
	}
}

// walk assigns folders in pre-order and records pages and warnings.
func (b *builder) walk(parent *Node) {
	used := map[string]bool{}
	unnamed := 0
	for _, n := range parent.Children {
		var seg string
		if n.Unnamed {
			unnamed++
			seg = fmt.Sprintf("unnamed-%d", unnamed)
			b.t.Warnings = append(b.t.Warnings, fmt.Sprintf(
				"%s: element has no name; its page is titled %q", n.Name, UnnamedTitle))
		} else {
			seg = segment(n.Title)
		}
		seg = unique(seg, used)
		if parent.Dir == "" {
			n.Dir = seg
		} else {
			n.Dir = parent.Dir + "/" + seg
		}
		b.t.Pages = append(b.t.Pages, n)
		b.t.ByID[n.ID] = n
		b.walk(n)
	}
}

// declaredTitle is the element's name: its declared name, or the name its
// owning membership gives it (a redefining feature borrows one); "" when it
// has none.
func (b *builder) declaredTitle(id string) string {
	e := b.m.Elements[id]
	if n := e.Str("declaredName"); n != "" {
		return n
	}
	if mem, ok := b.m.Elements[e.Ref("owningMembership")]; ok {
		return mem.Str("memberName")
	}
	return ""
}

// annotations are the metaclasses of owned members that annotate the owner
// rather than make it up; sysml-members leaves them out (MC-12).
var annotations = map[string]bool{
	"Comment": true, "Documentation": true, "TextualRepresentation": true,
	"MetadataUsage": true, "MetadataFeature": true,
}

// members are e's owned members other than annotations.
func (b *builder) members(e model.Element) []Member {
	var out []Member
	for _, id := range e.Refs("ownedMember") {
		m, ok := b.m.Elements[id]
		if !ok || annotations[m.Type()] {
			continue
		}
		title := b.declaredTitle(id)
		if title == "" {
			title = UnnamedTitle
		}
		out = append(out, Member{ID: id, Title: title})
	}
	return out
}

// shortName is the element's declared short name, or the short name its
// owning membership gives it; "" when it has none.
func (b *builder) shortName(id string) string {
	e := b.m.Elements[id]
	if n := e.Str("declaredShortName"); n != "" {
		return n
	}
	if mem, ok := b.m.Elements[e.Ref("owningMembership")]; ok {
		return mem.Str("memberShortName")
	}
	return ""
}

// name is how an element is identified in messages.
func (b *builder) name(id string) string {
	e := b.m.Elements[id]
	for _, key := range []string{"qualifiedName", "sysx:positionalName"} {
		if s := e.Str(key); s != "" {
			return s
		}
	}
	return id
}

// segment makes a name usable as one folder name on any filesystem.
func segment(name string) string {
	var sb strings.Builder
	for _, r := range name {
		switch {
		case r < 0x20, r == 0x7f, strings.ContainsRune(`/\:*?"<>|`, r):
			sb.WriteRune('_')
		default:
			sb.WriteRune(r)
		}
	}
	s := strings.TrimSpace(sb.String())
	if s == "" || strings.HasPrefix(s, ".") {
		s = "_" + s
	}
	return s
}

// unique returns seg, or seg with a numeric suffix when a sibling already
// took it, so two pages never share a folder.
func unique(seg string, used map[string]bool) string {
	candidate := seg
	for i := 2; used[strings.ToLower(candidate)]; i++ {
		candidate = fmt.Sprintf("%s-%d", seg, i)
	}
	used[strings.ToLower(candidate)] = true
	return candidate
}

// Href is a relative link from the page at from to the page of to, each
// folder name escaped, so the site works from file:// as from a server.
func Href(from, to *Node) string {
	src, dst := segments(from.Dir), segments(to.Dir)
	common := 0
	for common < len(src) && common < len(dst) && src[common] == dst[common] {
		common++
	}
	var sb strings.Builder
	sb.WriteString(strings.Repeat("../", len(src)-common))
	for _, seg := range dst[common:] {
		sb.WriteString(strings.ReplaceAll(url.PathEscape(seg), "+", "%2B"))
		sb.WriteByte('/')
	}
	sb.WriteString("index.html")
	return sb.String()
}

func segments(dir string) []string {
	if dir == "" {
		return nil
	}
	return strings.Split(dir, "/")
}
