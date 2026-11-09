package macros

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/rolling-robot/sysml-publisher/internal/model"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

// errNoSuchName matches (errors.Is) the error resolve returns when nothing
// is called that name, so a caller can try another reading of it (an
// element kind, for one).
var errNoSuchName = errors.New("no such name")

// unknownNameError is that error; its text is the message for the reader.
type unknownNameError struct{ name string }

func (e unknownNameError) Error() string {
	return fmt.Sprintf("the model has no element named %q", e.name)
}

func (e unknownNameError) Is(target error) bool { return target == errNoSuchName }

// nameIndex finds the element a macro names, the way a SysML reference is
// read: every segment of a qualified name may be a name or a short name.
type nameIndex struct {
	m *model.Model
	// tops are the top-level elements: the members of each file's root
	// namespace.
	tops []string
	// byName are the ids of the elements, library ones included, that have
	// a name or short name, in model order.
	byName map[string][]string
	// library maps a library element's qualified name to its id.
	library map[string]string
}

func newNameIndex(m *model.Model) *nameIndex {
	x := &nameIndex{m: m, byName: map[string][]string{}, library: map[string]string{}}
	for _, id := range m.Order {
		e := m.Elements[id]
		if e.Type() == "Namespace" && e.Ref("owningRelationship") == "" {
			x.tops = append(x.tops, e.Refs("ownedMember")...)
		}
		for _, name := range x.names(id) {
			x.byName[name] = append(x.byName[name], id)
		}
	}
	// Sorted, so an ambiguity is reported the same way on every run.
	ids := make([]string, 0, len(m.Library))
	for id := range m.Library {
		ids = append(ids, id)
	}
	slices.Sort(ids)
	for _, id := range ids {
		lib := m.Library[id]
		x.library[lib.QualifiedName] = id
		x.byName[lib.Name()] = append(x.byName[lib.Name()], id)
	}
	return x
}

// names are the name and the short name of an element, those its owning
// membership gives it when it declares none (a redefinition borrows them).
func (x *nameIndex) names(id string) []string {
	e := x.m.Elements[id]
	mem := x.m.Elements[e.Ref("owningMembership")]
	var out []string
	for _, pair := range [][2]string{{"declaredName", "memberName"}, {"declaredShortName", "memberShortName"}} {
		n := e.Str(pair[0])
		if n == "" {
			n = mem.Str(pair[1])
		}
		if n != "" && !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return out
}

// resolve finds the one element name refers to, seen from the page of n,
// among the elements accept allows (nil allows all):
//
//  1. as a qualified name from the top of the model (Vehicles::Wheel);
//  2. inside the page's element, then inside each element around it;
//  3. as a standard-library element's qualified name (ScalarValues::Real);
//  4. as a plain name or short name only one element has.
//
// Each segment of a qualified name may be a name or a short name, quoted or
// not. An unknown name gives an error matching errNoSuchName.
func (x *nameIndex) resolve(name string, n *tree.Node, accept func(model.Element) bool) (string, error) {
	if accept == nil {
		accept = func(model.Element) bool { return true }
	}
	segs := splitQualified(name)
	keep := func(ids []string) []string {
		var out []string
		for _, id := range ids {
			if e, ok := x.m.Elements[id]; ok && accept(e) || !ok && x.acceptLibrary(id, accept) {
				out = append(out, id)
			}
		}
		return out
	}
	tries := [][]string{keep(x.walk(x.tops, segs))}
	for scope := n.ID; scope != ""; scope = x.m.Elements[scope].Ref("owner") {
		tries = append(tries, keep(x.walk(x.m.Elements[scope].Refs("ownedMember"), segs)))
	}
	if id, ok := x.library[name]; ok {
		tries = append(tries, keep([]string{id}))
	}
	if len(segs) == 1 {
		tries = append(tries, keep(x.byName[segs[0]]))
	}
	for _, ids := range tries {
		switch {
		case len(ids) == 1:
			return ids[0], nil
		case len(ids) > 1:
			var qns []string
			for _, id := range ids {
				qns = append(qns, x.qualifiedName(id))
			}
			return "", fmt.Errorf("%q names several elements (%s); use a qualified name", name, strings.Join(qns, ", "))
		}
	}
	return "", unknownNameError{name}
}

// acceptLibrary applies accept to a library element, which has no element
// data: it is offered as an element with only its @type.
func (x *nameIndex) acceptLibrary(id string, accept func(model.Element) bool) bool {
	lib, ok := x.m.Library[id]
	return ok && accept(model.Element{"@id": id, "@type": lib.Type})
}

// walk follows segs from the elements start: the first segment names one of
// them, each later one a member of the element before.
func (x *nameIndex) walk(start, segs []string) []string {
	cur := start
	for i, seg := range segs {
		var next []string
		for _, id := range cur {
			if slices.Contains(x.names(id), seg) && !slices.Contains(next, id) {
				next = append(next, id)
			}
		}
		if i == len(segs)-1 || len(next) == 0 {
			return next
		}
		cur = nil
		for _, id := range next {
			cur = append(cur, x.m.Elements[id].Refs("ownedMember")...)
		}
	}
	return nil
}

// name is the element's name as declared, without quotes: its declared
// name, or the one its owning membership gives it; a library element's last
// segment (Real); UnnamedName when it has none.
func (x *nameIndex) name(id string) string {
	if lib, ok := x.m.Library[id]; ok {
		return lib.Name()
	}
	e := x.m.Elements[id]
	if n := e.Str("declaredName"); n != "" {
		return n
	}
	if n := x.m.Elements[e.Ref("owningMembership")].Str("memberName"); n != "" {
		return n
	}
	return tree.UnnamedName
}

func (x *nameIndex) qualifiedName(id string) string {
	if lib, ok := x.m.Library[id]; ok {
		return lib.QualifiedName
	}
	if qn := x.m.Elements[id].Str("qualifiedName"); qn != "" {
		return qn
	}
	return id
}

// splitQualified splits a qualified name at "::" outside quotes and unquotes
// each segment: Vehicles::'front wheel' gives Vehicles, front wheel.
func splitQualified(name string) []string {
	var segs []string
	var cur strings.Builder
	quoted := false
	for i := 0; i < len(name); i++ {
		switch c := name[i]; {
		case c == '\'':
			quoted = !quoted
		case c == '\\' && quoted && i+1 < len(name):
			i++
			cur.WriteByte(name[i])
		case c == ':' && !quoted && strings.HasPrefix(name[i:], "::"):
			segs = append(segs, cur.String())
			cur.Reset()
			i++
		default:
			cur.WriteByte(c)
		}
	}
	return append(segs, cur.String())
}
