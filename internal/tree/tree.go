// Package tree decides which elements of a model get pages, what each page
// shows, and where it is written (requirements PG-1..7, PT-1..6). It finds
// them by querying the model for its textual representations, then grows
// the tree from their owners: the elements that inherit them, and the
// owners of both up to the top of the model (DS-4).
package tree

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/Open-MBEE/OpenSysML/client/opensysml"
	"github.com/rolling-robot/sysml-publisher/internal/model"
)

// UnnamedName stands in for the name of an element without one, wherever
// its name is shown, page title included (PG-5).
const UnnamedName = "Unnamed"

// Rep is an AsciiDoc textual representation.
type Rep struct {
	ID    string
	Owner string // id of the element that owns it
	Body  string
}

// Node is one page of the site, or the site root.
type Node struct {
	ID string // element id; empty for the root
	// Name is the element's name, UnnamedName, or empty for the root. It is
	// not the page title, which is decided when the page is rendered (PG-8).
	Name string
	// Unnamed is set when the element has no name (PG-5, PT-6).
	Unnamed bool
	// QualifiedName identifies the element in messages and to the
	// evaluator: its qualified name, or its positional name (M::@2) when it
	// has none.
	QualifiedName string
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
	// annotations and multiplicities (MC-12).
	Members  []Member
	Parent   *Node
	Children []*Node
}

// Documented reports whether the element has representations of its own or
// inherited ones; other nodes are empty ancestor pages (PG-6).
func (n *Node) Documented() bool { return len(n.Own)+len(n.Inherited) > 0 }

// Member is an owned member of a page's element.
type Member struct {
	ID        string
	Name      string // its name, or UnnamedName
	Metaclass string // PartUsage, AttributeUsage, ...
	// Types are the names of the member's types (part w : Wheel gives
	// Wheel), in declaration order, including those a usage gets through
	// what it subsets or redefines (part front :> wheels gives Wheel); none
	// for a definition or an untyped usage.
	Types []string
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

// Specializes reports whether element specific is typed by, subclasses or
// subsets element general, directly or through a chain of such relationships
// (part spare : Sport with Sport :> Wheel specializes Wheel; MC-12). No
// element specializes itself.
func (t *Tree) Specializes(ctx context.Context, specific, general string) (bool, error) {
	seen := map[string]bool{specific: true}
	queue, err := t.Model.Generals(ctx, specific)
	if err != nil {
		return false, err
	}
	for len(queue) > 0 {
		g := queue[0]
		queue = queue[1:]
		if g == general {
			return true, nil
		}
		if seen[g] {
			continue
		}
		seen[g] = true
		next, err := t.Model.Generals(ctx, g)
		if err != nil {
			return false, err
		}
		queue = append(queue, next...)
	}
	return false, nil
}

// Build computes the page tree of m:
//
//  1. query the textual representations; the owners of the AsciiDoc ones
//     are documented (others are skipped with a warning, PG-4);
//  2. query the usages and definitions, and keep those that inherit an
//     AsciiDoc representation (PG-7);
//  3. keep every owner of a documented element up to the top of the model
//     (PG-6), and nothing else (PT-4);
//  4. lay the kept elements out as the model's containment tree, in
//     declaration order, and give each a folder (PT-1, PT-6).
func Build(ctx context.Context, m *model.Model) (*Tree, error) {
	b := &builder{m: m, t: &Tree{Model: m, ByID: map[string]*Node{}}, ctx: ctx,
		reps: map[string][]Rep{}, inheritedOf: map[string][]Rep{}, keep: map[string]bool{}}
	for _, step := range []func() error{b.collectReps, b.collectInherited, b.collectAncestors} {
		if err := step(); err != nil {
			return nil, err
		}
	}
	b.t.Root = &Node{}
	for _, top := range m.Tops {
		if err := b.add(b.t.Root, top); err != nil {
			return nil, err
		}
	}
	b.walk(b.t.Root)
	return b.t, nil
}

type builder struct {
	m   *model.Model
	t   *Tree
	ctx context.Context
	// reps are the AsciiDoc representations by owner id; inheritedOf the
	// representations each usage or definition inherits.
	reps        map[string][]Rep
	inheritedOf map[string][]Rep
	// keep are the elements that get a page.
	keep map[string]bool
}

// isAsciiDoc reports whether a representation's language is AsciiDoc.
func isAsciiDoc(language string) bool {
	switch strings.ToLower(language) {
	case "adoc", "asciidoc":
		return true
	}
	return false
}

// id is the element id of an element a query returned, which names it by
// qualified or positional name.
func (b *builder) id(e opensysml.QueryElement) (string, error) {
	p, err := b.m.Props(b.ctx, e.ID, "owner")
	if err != nil {
		return "", err
	}
	return p.Element.ID, nil
}

func (b *builder) collectReps() error {
	reps, err := b.m.Query(b.ctx, opensysml.Query{Where: opensysml.InstanceOf("TextualRepresentation")})
	if err != nil {
		return err
	}
	for _, e := range reps {
		r, err := b.m.Representation(b.ctx, e.ID)
		if err != nil {
			return err
		}
		if !isAsciiDoc(r.Language) {
			owner, err := b.m.QualifiedName(b.ctx, r.Owner)
			if err != nil {
				return err
			}
			b.t.Warnings = append(b.t.Warnings, fmt.Sprintf(
				"%s: skipped representation in language %q", owner, r.Language))
			continue
		}
		b.reps[r.Owner] = append(b.reps[r.Owner], Rep{ID: r.ID, Owner: r.Owner, Body: r.Body})
		b.keep[r.Owner] = true
	}
	return nil
}

// collectInherited finds the usages and definitions that inherit AsciiDoc
// representations: no query asks "what specializes X" across subsetting
// and redefinition (AGENTS.md §13), so each one's generals are followed.
func (b *builder) collectInherited() error {
	types, err := b.m.Query(b.ctx, opensysml.Query{Where: opensysml.InstanceOf("Usage", "Definition")})
	if err != nil {
		return err
	}
	for _, e := range types {
		id, err := b.id(e)
		if err != nil {
			return err
		}
		reps, err := b.inherited(id)
		if err != nil {
			return err
		}
		if len(reps) > 0 {
			b.inheritedOf[id] = reps
			b.keep[id] = true
		}
	}
	return nil
}

// inherited are the representations owned by the elements id specializes,
// directly or transitively, nearest first, each once.
func (b *builder) inherited(id string) ([]Rep, error) {
	var out []Rep
	seenRep := map[string]bool{}
	visited := map[string]bool{id: true}
	queue, err := b.m.Generals(b.ctx, id)
	if err != nil {
		return nil, err
	}
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
		next, err := b.m.Generals(b.ctx, g)
		if err != nil {
			return nil, err
		}
		queue = append(queue, next...)
	}
	return out, nil
}

// collectAncestors keeps the owners of every documented element, up to the
// root namespace, which is not a page.
func (b *builder) collectAncestors() error {
	documented := make([]string, 0, len(b.keep))
	for id := range b.keep {
		documented = append(documented, id)
	}
	for _, id := range documented {
		for {
			owner, err := b.m.Owner(b.ctx, id)
			if err != nil {
				return err
			}
			if owner.ID == "" || owner.Metaclass == "Namespace" && b.isRoot(owner.ID) {
				break
			}
			if b.keep[owner.ID] {
				break // its owners are kept already, or will be
			}
			b.keep[owner.ID] = true
			id = owner.ID
		}
	}
	return nil
}

// isRoot reports whether a namespace is a file's root namespace: one with no
// owner.
func (b *builder) isRoot(id string) bool {
	owner, err := b.m.Owner(b.ctx, id)
	return err == nil && owner.ID == ""
}

// add adds a node for the kept element e under parent, and below it nodes
// for its kept members, in declaration order.
func (b *builder) add(parent *Node, e model.Ref) error {
	if !b.keep[e.ID] {
		return nil
	}
	name, err := b.m.Name(b.ctx, e.ID)
	if err != nil {
		return err
	}
	qn, err := b.m.QualifiedName(b.ctx, e.ID)
	if err != nil {
		return err
	}
	docs, err := b.m.Docs(b.ctx, e.ID)
	if err != nil {
		return err
	}
	short, err := b.m.ShortName(b.ctx, e.ID)
	if err != nil {
		return err
	}
	members, err := b.members(e.ID)
	if err != nil {
		return err
	}
	n := &Node{
		ID:            e.ID,
		Name:          name,
		Unnamed:       name == "",
		QualifiedName: qn,
		Own:           b.reps[e.ID],
		Inherited:     b.inheritedOf[e.ID],
		Docs:          docs,
		ShortName:     short,
		Members:       members,
		Parent:        parent,
	}
	if n.Unnamed {
		n.Name = UnnamedName
	}
	parent.Children = append(parent.Children, n)
	owned, err := b.m.Members(b.ctx, e.ID)
	if err != nil {
		return err
	}
	for _, child := range owned {
		if err := b.add(n, child); err != nil {
			return err
		}
	}
	return nil
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
				"%s: element has no name; its page is titled %q", n.QualifiedName, UnnamedName))
		} else {
			seg = segment(n.Name)
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

// notMembers are the metaclasses of owned members that sysml-members-list
// leaves out (MC-12): annotations, which annotate the owner rather than make
// it up, and multiplicities, which are part of the owner's declaration
// (OpenSysML gives every usage one, its implicit [1..1] included).
var notMembers = map[string]bool{
	"Comment": true, "Documentation": true, "TextualRepresentation": true,
	"MetadataUsage": true, "MetadataFeature": true,
	"Multiplicity": true, "MultiplicityRange": true,
}

// members are id's owned members other than annotations and multiplicities.
func (b *builder) members(id string) ([]Member, error) {
	owned, err := b.m.Members(b.ctx, id)
	if err != nil {
		return nil, err
	}
	var out []Member
	for _, r := range owned {
		if notMembers[r.Metaclass] {
			continue
		}
		name, err := b.m.Name(b.ctx, r.ID)
		if err != nil {
			return nil, err
		}
		if name == "" {
			name = UnnamedName
		}
		types, err := b.m.Types(b.ctx, r.ID)
		if err != nil {
			return nil, err
		}
		var typeNames []string
		for _, typ := range types {
			tn := typ.Name() // from the qualified name: library types need no read
			if tn == "" {
				if tn, err = b.m.Name(b.ctx, typ.ID); err != nil {
					return nil, err
				}
			}
			if tn != "" {
				typeNames = append(typeNames, tn)
			}
		}
		out = append(out, Member{ID: r.ID, Name: name, Metaclass: r.Metaclass, Types: typeNames})
	}
	return out, nil
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
