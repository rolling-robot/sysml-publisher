// Package sysml reads a SysML v2 model the way the SysML v2 API serves it
// (requirement DS-4): element queries (Query), reads of one element's
// metamodel properties (Props, cached), facts built on them (names, docs,
// representations, specializations) and expression evaluation (Evaluate).
// Nothing exports the whole model. It works on a model parsed by
// OpenSysML's in-process engine (github.com/Open-MBEE/OpenSysML).
package sysml

import (
	"context"
	"errors"

	"github.com/Open-MBEE/OpenSysML/client/opensysml"
)

// UnnamedName stands in for the name of an element without one, wherever
// this tool shows its name (requirement PG-5).
const UnnamedName = "Unnamed"

// Model reads one parsed model. It is not safe for concurrent use: reads
// are cached.
type Model struct {
	// ElementCount is the number of elements in the model.
	ElementCount int
	// Tops are the top-level elements, in file order: the members of each
	// file's root namespace.
	Tops []Ref

	// The engine the model was parsed by, open until Close.
	client opensysml.Client
	parsed *opensysml.Model
	// cache holds the properties Props has read, by element id and by every
	// other name an element was asked by.
	cache map[string]*elementCache
	// own are the ids of the model's own elements, by readIndex.
	own   map[string]bool
	stats Stats
}

// Open reads the model parsed by client: it queries every element once, for
// ElementCount, Tops and the model's own ids. Open takes the client over
// when it succeeds: Close closes it. On error the client stays the
// caller's.
func Open(ctx context.Context, client opensysml.Client, parsed *opensysml.Model) (*Model, error) {
	m := &Model{client: client, parsed: parsed}
	if err := m.readIndex(ctx); err != nil {
		return nil, err
	}
	return m, nil
}

// Stats count how the model was read: the cost of reading it through
// queries and element reads rather than one export (DS-4).
type Stats struct {
	Queries int // Query calls
	Reads   int // ElementProperties calls: elements, or properties, not cached yet
	Hits    int // Props calls answered from the cache
}

// Stats are the reads made so far.
func (m *Model) Stats() Stats { return m.stats }

// Evaluate evaluates a SysML expression with the element named subject (by
// qualified name) as its subject, so a feature reads that element's value:
// its own, a redefinition's, or one inherited from its types.
func (m *Model) Evaluate(ctx context.Context, subject, expression string) (opensysml.Value, error) {
	if m.client == nil {
		return nil, errors.New("sysml: the model is closed")
	}
	return m.client.Evaluate(ctx, m.parsed, expression, opensysml.WithSubject(subject))
}

// Close releases the engine. What was read stays cached, but nothing more
// can be read.
func (m *Model) Close() error {
	if m.client == nil {
		return nil
	}
	err := m.client.Close()
	m.client, m.parsed = nil, nil
	return err
}

// readIndex queries every element once: the count, the ids of the model's
// own elements, and those without an owner, which are the top-level ones,
// in file order. The query names elements by qualified or positional name,
// so each is read once (its name, a cheap property) for its id.
func (m *Model) readIndex(ctx context.Context) error {
	all, err := m.Query(ctx, opensysml.Query{Select: []string{"owner"}})
	if err != nil {
		return err
	}
	m.ElementCount = len(all)
	m.own = make(map[string]bool, len(all))
	for _, e := range all {
		p, err := m.Props(ctx, e.ID, "name")
		if err != nil {
			return err
		}
		m.own[p.Element.ID] = true
		if e.Properties["owner"] == "" {
			m.Tops = append(m.Tops, p.Element)
		}
	}
	return nil
}

// Contains reports whether id is an element of the model itself, not of
// the standard library it refers to.
func (m *Model) Contains(id string) bool { return m.own[id] }
