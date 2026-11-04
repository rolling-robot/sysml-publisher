package macros

import (
	"fmt"
	"html"
	"strings"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/model"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

// MetadataTable is the block macro sysml-metadata-table::<metadata def>[],
// replaced by a table of the attributes of that metadata on the page's
// element: each attribute's name and value, the metadata def's value where
// the element binds none (MC-13).
const MetadataTable = "sysml-metadata-table"

// MetadataTableRole is the role of the block holding the tables, for styling.
const MetadataTableRole = "sysml-metadata-table"

// NoValue is shown for an attribute that has no value.
const NoValue = "(no value)"

func registerMetadataTable(g *asciidoctor.ExtensionGroup, t *tree.Tree) {
	md := newMetadataIndex(t.Model)
	g.BlockMacro(MetadataTable, asciidoctor.BlockMacroProcessorFunc(
		func(parent asciidoctor.StructuralNode, target string, _ map[string]any) (asciidoctor.StructuralNode, error) {
			n, err := pageElement(parent, t)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", MetadataTable, err)
			}
			def, err := md.definition(target)
			if err != nil {
				return nil, fmt.Errorf("%s::%s[] on the page of %s: %w", MetadataTable, target, n.QualifiedName, err)
			}
			usages := md.usages(n.ID, def)
			if len(usages) == 0 {
				return nil, nil // the element has no such metadata; the line is dropped
			}
			attrs := md.attributes(def)
			// One table per usage; an element rarely carries the same
			// metadata twice, so they share an open block.
			block, err := asciidoctor.CreateOpenBlock(parent, nil, asciidoctor.Attributes{"role": MetadataTableRole}, nil)
			if err != nil {
				return nil, err
			}
			for _, u := range usages {
				tbl, err := md.table(block, def, u, attrs)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", MetadataTable, err)
				}
				block.Append(tbl)
			}
			return block, nil
		}), nil)
}

// metadataIndex answers questions about metadata from the model.
type metadataIndex struct {
	m *model.Model
	// supertypes are each definition's direct supertypes, from its
	// subclassifications, in declaration order.
	supertypes map[string][]string
}

func newMetadataIndex(m *model.Model) *metadataIndex {
	md := &metadataIndex{m: m, supertypes: map[string][]string{}}
	for _, id := range m.Order {
		e := m.Elements[id]
		if e.Type() == "Subclassification" {
			specific := e.Ref("specific")
			md.supertypes[specific] = append(md.supertypes[specific], e.Ref("general"))
		}
	}
	return md
}

// definition finds the metadata def a macro names: by qualified name, or by
// name when only one metadata def has it.
func (md *metadataIndex) definition(target string) (model.Element, error) {
	if target == "" {
		return nil, fmt.Errorf("no metadata def named; write %s::<metadata def>[]", MetadataTable)
	}
	var byName []model.Element
	for _, id := range md.m.Order {
		e := md.m.Elements[id]
		if e.Type() != "MetadataDefinition" {
			continue
		}
		if e.Str("qualifiedName") == target {
			return e, nil
		}
		if e.Str("declaredName") == target {
			byName = append(byName, e)
		}
	}
	switch len(byName) {
	case 0:
		return nil, fmt.Errorf("the model has no metadata def %q", target)
	case 1:
		return byName[0], nil
	}
	var names []string
	for _, e := range byName {
		names = append(names, e.Str("qualifiedName"))
	}
	return nil, fmt.Errorf("%q names several metadata defs (%s); use a qualified name", target, strings.Join(names, ", "))
}

// usages are the metadata usages the element owns that are typed by def, in
// declaration order. Both @def and the prefix form #def give one.
func (md *metadataIndex) usages(elementID string, def model.Element) []model.Element {
	var out []model.Element
	for _, id := range md.m.Elements[elementID].Refs("ownedMember") {
		e := md.m.Elements[id]
		if e.Type() != "MetadataUsage" {
			continue
		}
		for _, typ := range e.Refs("type") {
			if typ == def.ID() {
				out = append(out, e)
				break
			}
		}
	}
	return out
}

// attributes are the attributes of def: its own, then those of its
// supertypes, nearest first. One a nearer def redefines is not repeated.
func (md *metadataIndex) attributes(def model.Element) []model.Element {
	var out []model.Element
	redefined := map[string]bool{}
	visited := map[string]bool{}
	queue := []string{def.ID()}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if visited[id] {
			continue
		}
		visited[id] = true
		for _, fid := range md.m.Elements[id].Refs("ownedMember") {
			f := md.m.Elements[fid]
			if f.Type() != "AttributeUsage" && f.Type() != "ReferenceUsage" {
				continue
			}
			if redefined[fid] {
				continue
			}
			for _, r := range f.Refs("redefines") {
				redefined[r] = true
			}
			out = append(out, f)
		}
		queue = append(queue, md.supertypes[id]...)
	}
	return out
}

// value is the value of attr in usage u: the one u binds, else the one the
// metadata def gives (a default, or a fixed = value), following the
// attribute's own redefinitions; nil when there is none.
func (md *metadataIndex) value(u, attr model.Element) model.Element {
	for _, fid := range u.Refs("ownedMember") {
		f := md.m.Elements[fid]
		for _, r := range f.Refs("redefines") {
			if r == attr.ID() {
				if v, ok := md.m.Elements[f.Ref("value")]; ok {
					return v
				}
			}
		}
	}
	seen := map[string]bool{}
	for a := attr; a != nil && !seen[a.ID()]; {
		seen[a.ID()] = true
		if v, ok := md.m.Elements[a.Ref("value")]; ok {
			return v
		}
		next, ok := md.m.Elements[a.Ref("redefines")]
		if !ok {
			break
		}
		a = next
	}
	return nil
}

// valueText is how a value is shown: a string without its quotes, anything
// else as written in the model (7, true, Level::low, 1 + 2).
func valueText(v model.Element) string {
	if v.Type() == "LiteralString" {
		return v.Str("value")
	}
	if s := strings.TrimSpace(v.Str("sysx:sourceText")); s != "" {
		return s
	}
	if x, ok := v["value"]; ok && x != nil {
		return fmt.Sprint(x)
	}
	return v.Type()
}

// table is the attribute table of usage u, titled with the metadata def's
// name.
func (md *metadataIndex) table(parent asciidoctor.StructuralNode, def, u model.Element, attrs []model.Element) (*asciidoctor.Table, error) {
	tbl, err := asciidoctor.CreateTable(parent, nil)
	if err != nil {
		return nil, err
	}
	tbl.SetTitle(passthrough(html.EscapeString(def.Str("declaredName"))))
	var cols [2]*asciidoctor.Column
	for i := range cols {
		if cols[i], err = asciidoctor.CreateTableColumn(tbl, i, nil); err != nil {
			return nil, err
		}
		tbl.AddColumn(cols[i])
	}
	row := func(section asciidoctor.TableSection, name, value string) error {
		c1, err := asciidoctor.CreateTableCell(cols[0], name, nil)
		if err != nil {
			return err
		}
		c2, err := asciidoctor.CreateTableCell(cols[1], value, nil)
		if err != nil {
			return err
		}
		tbl.AddRow(section, c1, c2)
		return nil
	}
	if err := row(asciidoctor.TableHeader, "Attribute", "Value"); err != nil {
		return nil, err
	}
	for _, a := range attrs {
		value := `<span class="no-value">` + html.EscapeString(NoValue) + `</span>`
		if v := md.value(u, a); v != nil {
			value = html.EscapeString(valueText(v))
		}
		if err := row(asciidoctor.TableBody, passthrough(html.EscapeString(a.Str("declaredName"))), passthrough(value)); err != nil {
			return nil, err
		}
	}
	tbl.AssignColumnWidths()
	return tbl, nil
}
