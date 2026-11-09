package macros

import (
	"context"
	"errors"
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

func registerMetadataTable(g *asciidoctor.ExtensionGroup, e *env) {
	md := e.md
	g.BlockMacro(MetadataTable, asciidoctor.BlockMacroProcessorFunc(
		func(parent asciidoctor.StructuralNode, target string, _ map[string]any) (asciidoctor.StructuralNode, error) {
			n, err := pageElement(parent, e.t)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", MetadataTable, err)
			}
			fail := func(err error) error {
				return fmt.Errorf("%s::%s[] on the page of %s: %w", MetadataTable, target, n.QualifiedName, err)
			}
			def, err := md.definition(target, n)
			if err != nil {
				return nil, fail(err)
			}
			usages, err := md.usages(n.ID, def.ID)
			if err != nil {
				return nil, fail(err)
			}
			if len(usages) == 0 {
				return nil, nil // the element has no such metadata; the line is dropped
			}
			attrs, err := md.attributes(def.ID)
			if err != nil {
				return nil, fail(err)
			}
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

// metadataIndex answers questions about metadata, reading the model through
// Props (DS-4).
type metadataIndex struct {
	ctx   context.Context
	m     *model.Model
	names *nameIndex
}

// definition finds the metadata def a macro names, the way every macro reads
// a name (nameIndex.resolve), among metadata defs only: Info, Plant::Info,
// or a short name.
func (md *metadataIndex) definition(target string, n *tree.Node) (model.Ref, error) {
	if target == "" {
		return model.Ref{}, fmt.Errorf("no metadata def named; write %s::<metadata def>[]", MetadataTable)
	}
	def, err := md.names.resolve(target, n, func(r model.Ref) bool { return r.Metaclass == "MetadataDefinition" })
	if errors.Is(err, errNoSuchName) {
		return model.Ref{}, fmt.Errorf("the model has no metadata def %q", target)
	}
	return def, err
}

// usages are the metadata usages the element owns that are typed by def, in
// declaration order. Both @def and the prefix form #def give one.
func (md *metadataIndex) usages(elementID, def string) ([]model.Ref, error) {
	members, err := md.m.Members(md.ctx, elementID)
	if err != nil {
		return nil, err
	}
	var out []model.Ref
	for _, r := range members {
		if r.Metaclass != "MetadataUsage" {
			continue
		}
		p, err := md.m.Props(md.ctx, r.ID, "metadataDefinition")
		if err != nil {
			return nil, err
		}
		if p.Ref("metadataDefinition").ID == def {
			out = append(out, r)
		}
	}
	return out, nil
}

// attributes are the attributes of def: its own, then those of its
// supertypes, nearest first. One a nearer def redefines is not repeated.
func (md *metadataIndex) attributes(def string) ([]model.Ref, error) {
	var out []model.Ref
	redefined := map[string]bool{}
	visited := map[string]bool{}
	queue := []string{def}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if visited[id] {
			continue
		}
		visited[id] = true
		members, err := md.m.Members(md.ctx, id)
		if err != nil {
			return nil, err
		}
		for _, f := range members {
			if f.Metaclass != "AttributeUsage" && f.Metaclass != "ReferenceUsage" || redefined[f.ID] {
				continue
			}
			rs, err := md.m.Redefined(md.ctx, f.ID)
			if err != nil {
				return nil, err
			}
			for _, r := range rs {
				redefined[r] = true
			}
			out = append(out, f)
		}
		supers, err := md.m.Generals(md.ctx, id)
		if err != nil {
			return nil, err
		}
		queue = append(queue, supers...)
	}
	return out, nil
}

// value is the expression giving attr's value in usage u: the one u binds,
// else the one the metadata def gives (a default, or a fixed = value),
// following the attribute's own redefinitions; "" when there is none.
func (md *metadataIndex) value(u, attr string) (string, error) {
	members, err := md.m.Members(md.ctx, u)
	if err != nil {
		return "", err
	}
	for _, f := range members {
		rs, err := md.m.Redefined(md.ctx, f.ID)
		if err != nil {
			return "", err
		}
		for _, r := range rs {
			if r != attr {
				continue
			}
			if v, err := md.valueOf(f.ID); v != "" || err != nil {
				return v, err
			}
		}
	}
	seen := map[string]bool{}
	for a := attr; a != "" && !seen[a]; {
		seen[a] = true
		if v, err := md.valueOf(a); v != "" || err != nil {
			return v, err
		}
		rs, err := md.m.Redefined(md.ctx, a)
		if err != nil || len(rs) == 0 {
			return "", err
		}
		a = rs[0]
	}
	return "", nil
}

// valueOf is the id of the expression a feature's value is, "" when none.
func (md *metadataIndex) valueOf(feature string) (string, error) {
	p, err := md.m.Props(md.ctx, feature, "value")
	if err != nil {
		return "", err
	}
	return p.Ref("value").ID, nil
}

// valueText is how a value is shown: a string without its quotes, anything
// else as the model writes it (7, true, Level::low, 1 + 2). The model gives
// no source text for an expression (AGENTS.md §13), so it is written back
// from the expression's parts: literals, references and operators; any
// other expression shows as its metaclass.
func (md *metadataIndex) valueText(expr string) (string, error) {
	return md.exprText(expr, true)
}

func (md *metadataIndex) exprText(expr string, top bool) (string, error) {
	p, err := md.m.Props(md.ctx, expr, "value", "operator", "argument", "referent")
	if err != nil {
		return "", err
	}
	switch mc := p.Element.Metaclass; mc {
	case "LiteralString":
		if top {
			return p.Str("value"), nil
		}
		return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(p.Str("value")) + `"`, nil
	case "LiteralInteger", "LiteralRational", "LiteralBoolean":
		return p.Str("value"), nil
	case "LiteralInfinity":
		return "*", nil
	case "NullExpression":
		return "null", nil
	case "FeatureReferenceExpression":
		return md.referenceText(p.Ref("referent"))
	case "OperatorExpression":
		args := p.Refs("argument")
		op := p.Str("operator")
		parts := make([]string, len(args))
		for i, a := range args {
			if parts[i], err = md.exprText(a.ID, false); err != nil {
				return "", err
			}
		}
		switch len(parts) {
		case 1:
			if op != "" && 'a' <= op[0] && op[0] <= 'z' {
				return op + " " + parts[0], nil // not x
			}
			return op + parts[0], nil // -x
		case 2:
			if !top {
				return "(" + parts[0] + " " + op + " " + parts[1] + ")", nil
			}
			return parts[0] + " " + op + " " + parts[1], nil
		}
		return mc, nil
	default:
		return mc, nil
	}
}

// referenceText writes a reference: an enumeration literal with its
// enumeration's name (Level::low), anything else by its name.
func (md *metadataIndex) referenceText(r model.Ref) (string, error) {
	name, err := md.names.name(r.ID)
	if err != nil {
		return "", err
	}
	owner, err := md.m.Owner(md.ctx, r.ID)
	if err != nil {
		return "", err
	}
	if owner.Metaclass == "EnumerationDefinition" {
		on, err := md.names.name(owner.ID)
		if err != nil {
			return "", err
		}
		return quoteName(on) + "::" + quoteName(name), nil
	}
	return quoteName(name), nil
}

// table is the attribute table of usage u, titled with the metadata def's
// name.
func (md *metadataIndex) table(parent asciidoctor.StructuralNode, def, u model.Ref, attrs []model.Ref) (*asciidoctor.Table, error) {
	tbl, err := asciidoctor.CreateTable(parent, nil)
	if err != nil {
		return nil, err
	}
	title, err := md.names.name(def.ID)
	if err != nil {
		return nil, err
	}
	tbl.SetTitle(passthrough(html.EscapeString(title)))
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
		expr, err := md.value(u.ID, a.ID)
		if err != nil {
			return nil, err
		}
		if expr != "" {
			text, err := md.valueText(expr)
			if err != nil {
				return nil, err
			}
			value = html.EscapeString(text)
		}
		name, err := md.names.name(a.ID)
		if err != nil {
			return nil, err
		}
		if err := row(asciidoctor.TableBody, passthrough(html.EscapeString(name)), passthrough(value)); err != nil {
			return nil, err
		}
	}
	tbl.AssignColumnWidths()
	return tbl, nil
}
