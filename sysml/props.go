package sysml

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Open-MBEE/OpenSysML/client/opensysml"
)

// Ref identifies an element: one of the model, or a standard-library element
// the model refers to.
type Ref struct {
	// ID is the element's id in the SysML v2 API element form ("M__pd"),
	// which Props accepts back.
	ID string
	// QualifiedName is "" for an unnamed element; a library element's is its
	// library path (ScalarValues::Real).
	QualifiedName string
	// Metaclass is the element's metaclass: "PartUsage", "Package", ...
	Metaclass string
}

// Name is the last segment of the qualified name, unquoted: Real for
// ScalarValues::Real, top speed for M::'top speed'; "" for an unnamed
// element.
func (r Ref) Name() string {
	qn := r.QualifiedName
	if i := lastSeparator(qn); i >= 0 {
		qn = qn[i+2:]
	}
	if len(qn) >= 2 && qn[0] == '\'' && qn[len(qn)-1] == '\'' {
		qn = strings.NewReplacer(`\'`, `'`, `\\`, `\`).Replace(qn[1 : len(qn)-1])
	}
	return qn
}

// lastSeparator is the index of the last "::" outside quotes, -1 if none.
func lastSeparator(qn string) int {
	last, quoted := -1, false
	for i := 0; i < len(qn); i++ {
		switch {
		case qn[i] == '\\' && quoted:
			i++
		case qn[i] == '\'':
			quoted = !quoted
		case !quoted && strings.HasPrefix(qn[i:], "::"):
			last = i
			i++
		}
	}
	return last
}

// Props are metamodel properties of one element, as Model.Props read them.
type Props struct {
	Element Ref
	values  map[string][]opensysml.PropertyValue
}

// Refs are the elements a property refers to, in order; none when the
// property is empty, absent, or holds no elements.
func (p *Props) Refs(name string) []Ref {
	var out []Ref
	for _, v := range p.values[name] {
		if v.Element != nil {
			out = append(out, Ref{ID: v.Element.ID, QualifiedName: v.Element.QualifiedName, Metaclass: v.Element.Metaclass})
		}
	}
	return out
}

// Ref is the first element a property refers to, the zero Ref when none.
func (p *Props) Ref(name string) Ref {
	if refs := p.Refs(name); len(refs) > 0 {
		return refs[0]
	}
	return Ref{}
}

// Str is the first value of a property as text ("" when it has none): a
// string as it is, a number or boolean as written.
func (p *Props) Str(name string) string {
	for _, v := range p.values[name] {
		if v.Element == nil {
			return v.Text()
		}
	}
	return ""
}

// Strs are the property's non-element values as text, in order.
func (p *Props) Strs(name string) []string {
	var out []string
	for _, v := range p.values[name] {
		if v.Element == nil {
			out = append(out, v.Text())
		}
	}
	return out
}

// Bool is a boolean property; false when absent.
func (p *Props) Bool(name string) bool {
	for _, v := range p.values[name] {
		if v.Bool != nil {
			return *v.Bool
		}
	}
	return false
}

// ErrNoSuchElement matches (errors.Is) the error Props returns for an id the
// model does not know.
var ErrNoSuchElement = errors.New("model: no such element")

// elementCache holds what Props has read: per element, its ref and the
// properties read so far (a property its metaclass does not declare is kept
// as nil, so it is not asked again).
type elementCache struct {
	ref    Ref
	values map[string][]opensysml.PropertyValue
}

// Props reads the named properties of the element id names: an element form
// id (M__pd), a qualified name (M::pd) or a positional id (M::@2). Values
// are cached, so asking again costs nothing. A property the element's
// metaclass does not declare reads as empty. Not safe for concurrent use.
func (m *Model) Props(ctx context.Context, id string, names ...string) (*Props, error) {
	if m.client == nil {
		return nil, errors.New("the model is closed")
	}
	if m.cache == nil {
		m.cache = map[string]*elementCache{}
	}
	c := m.cache[id]
	var missing []string
	for _, name := range names {
		if c == nil {
			missing = names
			break
		}
		if _, ok := c.values[name]; !ok {
			missing = append(missing, name)
		}
	}
	if c != nil && len(missing) == 0 {
		m.stats.Hits++
	}
	if c == nil || len(missing) > 0 {
		m.stats.Reads++
		ep, err := m.client.ElementProperties(ctx, m.parsed, id, missing...)
		if err != nil {
			var fe *opensysml.FailureError
			if errors.As(err, &fe) && strings.Contains(fe.Message, "not found") {
				return nil, fmt.Errorf("%w: %s", ErrNoSuchElement, id)
			}
			return nil, fmt.Errorf("reading %s: %w", id, err)
		}
		ref := Ref{ID: ep.Element.ID, QualifiedName: ep.Element.QualifiedName, Metaclass: ep.Element.Metaclass}
		if ref.ID == "" {
			return nil, fmt.Errorf("%w: %s", ErrNoSuchElement, id)
		}
		// One entry per element, whatever it was asked by: its form id is
		// the key, and the name asked by points at the same entry.
		if c = m.cache[ref.ID]; c == nil {
			c = &elementCache{ref: ref, values: map[string][]opensysml.PropertyValue{}}
			m.cache[ref.ID] = c
		}
		m.cache[id] = c
		for _, p := range ep.Properties {
			c.values[p.Name] = p.Values
		}
		for _, name := range missing {
			if _, ok := c.values[name]; !ok {
				c.values[name] = nil
			}
		}
	}
	return &Props{Element: c.ref, values: c.values}, nil
}

// Query selects the model's elements a query matches, in declaration order.
// A property it selects gives one value per element (see AGENTS.md §13):
// read multi-valued properties with Props.
func (m *Model) Query(ctx context.Context, q opensysml.Query) ([]opensysml.QueryElement, error) {
	if m.client == nil {
		return nil, errors.New("the model is closed")
	}
	m.stats.Queries++
	els, err := m.client.Query(ctx, m.parsed, q)
	if err != nil {
		return nil, fmt.Errorf("querying the model: %w", err)
	}
	return els, nil
}
