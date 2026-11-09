package macros

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/Open-MBEE/OpenSysML/client/opensysml"
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
// It reads the model through Props and Query (DS-4).
type nameIndex struct {
	ctx context.Context
	m   *model.Model
	// tops are the top-level elements: the members of each file's root
	// namespace.
	tops []model.Ref
	// byName caches the elements Query found with a name or short name.
	byName map[string][]model.Ref
}

func newNameIndex(ctx context.Context, t *tree.Tree) *nameIndex {
	return &nameIndex{ctx: ctx, m: t.Model, tops: t.Model.Tops, byName: map[string][]model.Ref{}}
}

// names are the name and the short name of an element (a redefinition
// borrows the redefined feature's name).
func (x *nameIndex) names(id string) ([]string, error) {
	p, err := x.m.Props(x.ctx, id, "name", "shortName")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, n := range []string{p.Str("name"), p.Element.Name(), p.Str("shortName")} {
		if n != "" && !slices.Contains(out, n) {
			out = append(out, n)
		}
	}
	return out, nil
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
func (x *nameIndex) resolve(name string, n *tree.Node, accept func(model.Ref) bool) (model.Ref, error) {
	if accept == nil {
		accept = func(model.Ref) bool { return true }
	}
	segs := splitQualified(name)
	keep := func(refs []model.Ref) []model.Ref {
		var out []model.Ref
		for _, r := range refs {
			if accept(r) && !slices.Contains(out, r) {
				out = append(out, r)
			}
		}
		return out
	}
	tries := func(yield func([]model.Ref, error) bool) {
		if !yield(x.walk(x.tops, segs)) {
			return
		}
		for scope := n.ID; scope != ""; {
			members, err := x.m.Members(x.ctx, scope)
			if err != nil {
				yield(nil, err)
				return
			}
			if !yield(x.walk(members, segs)) {
				return
			}
			owner, err := x.m.Owner(x.ctx, scope)
			if err != nil {
				yield(nil, err)
				return
			}
			scope = owner.ID
		}
		if !yield(x.library(name)) {
			return
		}
		if len(segs) == 1 {
			yield(x.named(segs[0]))
		}
	}
	for refs, err := range tries {
		if err != nil {
			return model.Ref{}, err
		}
		refs = keep(refs)
		switch {
		case len(refs) == 1:
			return refs[0], nil
		case len(refs) > 1:
			var qns []string
			for _, r := range refs {
				qn, err := x.qualifiedName(r.ID)
				if err != nil {
					return model.Ref{}, err
				}
				qns = append(qns, qn)
			}
			return model.Ref{}, fmt.Errorf("%q names several elements (%s); use a qualified name", name, strings.Join(qns, ", "))
		}
	}
	return model.Ref{}, unknownNameError{name}
}

// walk follows segs from the elements start: the first segment names one of
// them, each later one a member of the element before.
func (x *nameIndex) walk(start []model.Ref, segs []string) ([]model.Ref, error) {
	cur := start
	for i, seg := range segs {
		var next []model.Ref
		for _, r := range cur {
			names, err := x.names(r.ID)
			if err != nil {
				return nil, err
			}
			if slices.Contains(names, seg) && !slices.Contains(next, r) {
				next = append(next, r)
			}
		}
		if i == len(segs)-1 || len(next) == 0 {
			return next, nil
		}
		cur = nil
		for _, r := range next {
			members, err := x.m.Members(x.ctx, r.ID)
			if err != nil {
				return nil, err
			}
			cur = append(cur, members...)
		}
	}
	return nil, nil
}

// library is the standard-library element with that qualified name, if
// any: the model reads library elements by qualified name.
func (x *nameIndex) library(name string) ([]model.Ref, error) {
	if !strings.Contains(name, "::") {
		return nil, nil
	}
	p, err := x.m.Props(x.ctx, name, "owner")
	if errors.Is(err, model.ErrNoSuchElement) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if p.Element.QualifiedName != name {
		return nil, nil // read by another kind of id, not a qualified name
	}
	return []model.Ref{p.Element}, nil
}

// named are the model's elements whose name or short name is name, in
// model order.
func (x *nameIndex) named(name string) ([]model.Ref, error) {
	if refs, ok := x.byName[name]; ok {
		return refs, nil
	}
	els, err := x.m.Query(x.ctx, opensysml.Query{Where: opensysml.Any(
		opensysml.Equals("name", name), opensysml.Equals("shortName", name))})
	if err != nil {
		return nil, err
	}
	var refs []model.Ref
	for _, e := range els {
		p, err := x.m.Props(x.ctx, e.ID, "owner")
		if err != nil {
			return nil, err
		}
		refs = append(refs, p.Element)
	}
	x.byName[name] = refs
	return refs, nil
}

// name is the element's name, as declared, without quotes; UnnamedName
// when it has none.
func (x *nameIndex) name(id string) (string, error) {
	n, err := x.m.Name(x.ctx, id)
	if n == "" && err == nil {
		n = tree.UnnamedName
	}
	return n, err
}

// qualifiedName is the element's qualified name, or its positional name.
func (x *nameIndex) qualifiedName(id string) (string, error) {
	return x.m.QualifiedName(x.ctx, id)
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
