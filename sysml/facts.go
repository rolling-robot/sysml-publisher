package sysml

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// The element facts below are read with Props, so they work for any element
// of the model, with a page or not, and are cached with it.

// Name is the element's name: its declared name, or the one it gets
// otherwise (a redefining feature borrows the redefined feature's); "" when
// it has none. A library element's is the last segment of its qualified
// name.
func (m *Model) Name(ctx context.Context, id string) (string, error) {
	p, err := m.Props(ctx, id, "name")
	if err != nil {
		return "", err
	}
	if n := p.Str("name"); n != "" {
		return n, nil
	}
	return p.Element.Name(), nil
}

// ShortName is the element's short name, "" when it has none.
func (m *Model) ShortName(ctx context.Context, id string) (string, error) {
	p, err := m.Props(ctx, id, "shortName")
	if err != nil {
		return "", err
	}
	return p.Str("shortName"), nil
}

// QualifiedName is how the element is identified in messages and to the
// evaluator: its qualified name, or for an element without one its
// positional name, its owner's name followed by @ and its position among
// the owner's members (M::@2, counted from 0), as OpenSysML writes it. A
// root namespace, which is no element of the model's text, has "".
func (m *Model) QualifiedName(ctx context.Context, id string) (string, error) {
	p, err := m.Props(ctx, id, "owner")
	if err != nil {
		return "", err
	}
	if p.Element.QualifiedName != "" {
		return p.Element.QualifiedName, nil
	}
	owner := p.Ref("owner")
	if owner.ID == "" {
		return "", nil
	}
	members, err := m.Members(ctx, owner.ID)
	if err != nil {
		return "", err
	}
	i := slices.IndexFunc(members, func(r Ref) bool { return r.ID == p.Element.ID })
	if i < 0 {
		return "", fmt.Errorf("%s is not among the members of its owner %s", p.Element.ID, owner.ID)
	}
	prefix, err := m.QualifiedName(ctx, owner.ID)
	if err != nil {
		return "", err
	}
	if prefix == "" {
		return fmt.Sprintf("@%d", i), nil
	}
	return fmt.Sprintf("%s::@%d", prefix, i), nil
}

// Owner is the element that owns the element; the zero Ref for a root
// namespace.
func (m *Model) Owner(ctx context.Context, id string) (Ref, error) {
	p, err := m.Props(ctx, id, "owner")
	if err != nil {
		return Ref{}, err
	}
	return p.Ref("owner"), nil
}

// Members are the element's owned members, in declaration order.
func (m *Model) Members(ctx context.Context, id string) ([]Ref, error) {
	p, err := m.Props(ctx, id, "ownedMember")
	if err != nil {
		return nil, err
	}
	return p.Refs("ownedMember"), nil
}

// Types are the element's types, in declaration order, those a usage gets
// through what it subsets or redefines included; none for an element that
// is not a feature.
func (m *Model) Types(ctx context.Context, id string) ([]Ref, error) {
	p, err := m.Props(ctx, id, "type")
	if err != nil {
		return nil, err
	}
	return p.Refs("type"), nil
}

// Docs are the bodies of the element's doc comments, trimmed, in
// declaration order.
func (m *Model) Docs(ctx context.Context, id string) ([]string, error) {
	p, err := m.Props(ctx, id, "documentation")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, d := range p.Refs("documentation") {
		dp, err := m.Props(ctx, d.ID, "body")
		if err != nil {
			return nil, err
		}
		out = append(out, strings.TrimSpace(dp.Str("body")))
	}
	return out, nil
}

// Representation is a textual representation, in any language.
type Representation struct {
	ID       string
	Owner    string // id of the element that owns it
	Language string // as written
	Body     string // exactly as written, untrimmed
}

// Representation reads the textual representation id names.
func (m *Model) Representation(ctx context.Context, id string) (Representation, error) {
	p, err := m.Props(ctx, id, "owner", "language", "body")
	if err != nil {
		return Representation{}, err
	}
	return Representation{ID: p.Element.ID, Owner: p.Ref("owner").ID, Language: p.Str("language"), Body: p.Str("body")}, nil
}

// Representations are the textual representations the element owns, in
// any language, in declaration order.
func (m *Model) Representations(ctx context.Context, id string) ([]Representation, error) {
	p, err := m.Props(ctx, id, "textualRepresentation")
	if err != nil {
		return nil, err
	}
	var out []Representation
	for _, r := range p.Refs("textualRepresentation") {
		rep, err := m.Representation(ctx, r.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, rep)
	}
	return out, nil
}

// Generals are the ids of the elements the element specializes directly,
// through any specialization it owns (typing, subclassification,
// subsetting, redefinition, reference subsetting), in declaration order.
// A standard-library element's are not read, and are none: reading any
// library element makes OpenSysML build its library document's element
// form, seconds per document (AGENTS.md §13), and the library has no pages.
func (m *Model) Generals(ctx context.Context, id string) ([]string, error) {
	if !m.Contains(id) {
		return nil, nil
	}
	p, err := m.Props(ctx, id, "ownedSpecialization")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, s := range p.Refs("ownedSpecialization") {
		sp, err := m.Props(ctx, s.ID, "general")
		if errors.Is(err, ErrNoSuchElement) {
			// A standard-library element can list relationships that cannot
			// be read (AGENTS.md §13, 12); the library has no pages.
			continue
		}
		if err != nil {
			return nil, err
		}
		if g := sp.Ref("general").ID; g != "" && g != p.Element.ID {
			out = append(out, g)
		}
	}
	return out, nil
}

// Redefined are the ids of the features the feature redefines, in
// declaration order.
func (m *Model) Redefined(ctx context.Context, id string) ([]string, error) {
	p, err := m.Props(ctx, id, "ownedRedefinition")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, r := range p.Refs("ownedRedefinition") {
		rp, err := m.Props(ctx, r.ID, "redefinedFeature")
		if errors.Is(err, ErrNoSuchElement) {
			continue // as in Generals
		}
		if err != nil {
			return nil, err
		}
		if f := rp.Ref("redefinedFeature").ID; f != "" {
			out = append(out, f)
		}
	}
	return out, nil
}
