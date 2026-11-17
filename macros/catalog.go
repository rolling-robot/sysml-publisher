// Copyright 2026 Petr Mukhachev
// SPDX-License-Identifier: MIT

package macros

import (
	"context"
	"errors"
	"fmt"
	"html"
	"sort"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/sysml"
)

// Catalog is the block macro sysml-catalog::[], replaced, on a view's page,
// by a table of that view's exposed elements: one row per element of the
// exposed tree that carries the view's filter metadata, with a Name column
// and one column per metadata def attribute (MC-24). The macro takes no
// target or attributes: the exposed root and the filter come solely from the
// page's view element, so the user never re-states them.
const Catalog = "sysml-catalog"

// CatalogRole is the role of the block holding the catalog table, for
// styling.
const CatalogRole = "sysml-catalog"

func registerCatalog(g *asciidoctor.ExtensionGroup, e *env) {
	block(g, e, Catalog, nil, nil,
		func(c call[asciidoctor.StructuralNode]) (asciidoctor.StructuralNode, error) {
			return e.catalogTable(c)
		})
}

// exposure is one view's expose: the root the view exposes and whether it
// is recursive (a ** expose, the whole subtree) or direct (a * expose, the
// root's members only).
type exposure struct {
	root      sysml.Ref
	recursive bool
}

// viewExposure reads a view element's expose: its NamespaceExpose (direct)
// or MembershipExpose (recursive) relationship and the root it exposes.
func viewExposure(ctx context.Context, m *sysml.Model, viewID string) (exposure, error) {
	p, err := m.Props(ctx, viewID, "ownedRelationship")
	if err != nil {
		return exposure{}, err
	}
	var ex exposure
	found := false
	for _, r := range p.Refs("ownedRelationship") {
		switch r.Metaclass {
		case "NamespaceExpose":
			ep, err := m.Props(ctx, r.ID, "memberNamespace", "importedNamespace")
			if err != nil {
				return exposure{}, err
			}
			if root := ep.Ref("memberNamespace"); root.ID != "" {
				ex.root = root
			} else if root := ep.Ref("importedNamespace"); root.ID != "" {
				ex.root = root
			}
			found = true
		case "MembershipExpose":
			ep, err := m.Props(ctx, r.ID, "importedMembership")
			if err != nil {
				return exposure{}, err
			}
			if member := ep.Ref("importedMembership"); member.ID != "" {
				mp, err := m.Props(ctx, member.ID, "memberElement")
				if err != nil {
					return exposure{}, err
				}
				ex.root = mp.Ref("memberElement")
				ex.recursive = true
			}
			found = true
		}
	}
	if !found || ex.root.ID == "" {
		return exposure{}, errors.New("this view exposes nothing")
	}
	return ex, nil
}

// viewFilter reads a view element's filter: the metadata def its
// ElementFilterMembership condition requires (the @<def> the condition's
// input feature is typed by).
func viewFilter(ctx context.Context, m *sysml.Model, viewID string) (sysml.Ref, error) {
	p, err := m.Props(ctx, viewID, "ownedRelationship")
	if err != nil {
		return sysml.Ref{}, err
	}
	for _, r := range p.Refs("ownedRelationship") {
		if r.Metaclass != "ElementFilterMembership" {
			continue
		}
		fp, err := m.Props(ctx, r.ID, "condition")
		if err != nil {
			return sysml.Ref{}, err
		}
		cond := fp.Ref("condition")
		if cond.ID == "" {
			continue
		}
		cp, err := m.Props(ctx, cond.ID, "feature")
		if err != nil {
			return sysml.Ref{}, err
		}
		for _, pin := range cp.Refs("feature") {
			pp, err := m.Props(ctx, pin.ID, "type")
			if err != nil {
				return sysml.Ref{}, err
			}
			for _, ty := range pp.Refs("type") {
				if ty.Metaclass == "MetadataDefinition" {
					return ty, nil
				}
			}
		}
	}
	return sysml.Ref{}, errors.New("this view has no element filter")
}

// catalogTable is the catalog of the page's view: its exposed elements that
// carry the view's filter metadata, as a table with a Name column and one
// column per filter-def attribute.
func (e *env) catalogTable(c call[asciidoctor.StructuralNode]) (asciidoctor.StructuralNode, error) {
	p, err := e.m.Props(e.ctx, c.page.id, "name")
	if err != nil {
		return nil, err
	}
	if p.Element.Metaclass != "ViewUsage" {
		return nil, fmt.Errorf("the page's element %s is a %s, not a view; a catalog is a view's exposed elements", c.page.qualifiedName, p.Element.Metaclass)
	}
	ex, err := viewExposure(e.ctx, e.m, c.page.id)
	if err != nil {
		return nil, err
	}
	filter, err := viewFilter(e.ctx, e.m, c.page.id)
	if err != nil {
		return nil, err
	}
	exposed, err := e.exposedElements(e.ctx, ex)
	if err != nil {
		return nil, err
	}
	var rows []sysml.Ref
	for _, el := range exposed {
		us, err := e.md.usages(el.ID, filter.ID)
		if err != nil {
			return nil, err
		}
		if len(us) > 0 {
			rows = append(rows, el)
		}
	}
	if len(rows) == 0 {
		return nil, nil // nothing to show; the line is dropped (MC-12)
	}
	cols, err := e.catalogColumns(filter.ID)
	if err != nil {
		return nil, err
	}
	return e.catalogTableOf(c.parent, c.page.id, filter.ID, rows, cols)
}

// exposedElements are the elements of the view's exposed tree, in order: the
// direct members of the root for a * expose, the whole subtree (BFS, member
// declaration order) for a ** expose, the root itself first in both.
func (e *env) exposedElements(ctx context.Context, ex exposure) ([]sysml.Ref, error) {
	var out []sysml.Ref
	seen := map[string]bool{}
	if ex.root.ID != "" && e.m.Contains(ex.root.ID) {
		out = append(out, ex.root)
		seen[ex.root.ID] = true
	}
	members, err := e.m.Members(ctx, ex.root.ID)
	if err != nil {
		return nil, err
	}
	if !ex.recursive {
		for _, m := range members {
			if e.m.Contains(m.ID) && !seen[m.ID] {
				seen[m.ID] = true
				out = append(out, m)
			}
		}
		return out, nil
	}
	queue := members
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if !e.m.Contains(cur.ID) || seen[cur.ID] {
			continue
		}
		seen[cur.ID] = true
		out = append(out, cur)
		next, err := e.m.Members(ctx, cur.ID)
		if err != nil {
			return nil, err
		}
		queue = append(queue, next...)
	}
	return out, nil
}

// catalogColumn is one metadata def attribute shown as a catalog column, with
// its presentation label and order.
type catalogColumn struct {
	attr  sysml.Ref
	label string
	order int
}

// catalogColumns are the columns of a catalog: the filter def's attributes,
// each with its presentation label (the redefined attribute's rep body, else
// its name) and order, ordered by order ascending.
func (e *env) catalogColumns(filterID string) ([]catalogColumn, error) {
	attrs, err := e.md.attributes(filterID)
	if err != nil {
		return nil, err
	}
	var cols []catalogColumn
	for _, a := range attrs {
		label, err := e.md.label(a.ID)
		if err != nil {
			return nil, err
		}
		if label == "" {
			if label, err = e.names.name(a.ID); err != nil {
				return nil, err
			}
		}
		order, err := e.md.order(a.ID)
		if err != nil {
			return nil, err
		}
		cols = append(cols, catalogColumn{attr: a, label: label, order: order})
	}
	sort.SliceStable(cols, func(i, j int) bool { return cols[i].order < cols[j].order })
	return cols, nil
}

// catalogTableOf is the catalog table: a Name column (each cell the element's
// name, linked to its page), then one column per metadata attribute, its
// header the presentation label, its cells the element's value for the
// attribute — bound, else the def's default/fixed, else "(no value)".
func (e *env) catalogTableOf(parent asciidoctor.StructuralNode, from, filterID string, rows []sysml.Ref, cols []catalogColumn) (*asciidoctor.Table, error) {
	tbl, err := asciidoctor.CreateTable(parent, nil)
	if err != nil {
		return nil, err
	}
	colsFor := make([]*asciidoctor.Column, len(cols)+1)
	for i := range colsFor {
		if colsFor[i], err = asciidoctor.CreateTableColumn(tbl, i, nil); err != nil {
			return nil, err
		}
		tbl.AddColumn(colsFor[i])
	}
	cell := func(col *asciidoctor.Column, text string) (*asciidoctor.Cell, error) {
		return asciidoctor.CreateTableCell(col, text, nil)
	}
	noValue := `<span class="no-value">` + html.EscapeString(NoValue) + `</span>`
	// valueOf is the element's value for a metadata attribute: bound on its
	// filter usage, else the filter def's (default or fixed), else "(no value)".
	valueOf := func(u, attr string) string {
		if u == "" {
			return noValue
		}
		expr, err := e.md.value(u, attr)
		if err != nil || expr == "" {
			return noValue
		}
		text, err := e.md.valueText(expr)
		if err != nil || text == "" {
			return noValue
		}
		return html.EscapeString(text)
	}
	// Header: Name, then the attributes' labels.
	header := make([]*asciidoctor.Cell, len(colsFor))
	if header[0], err = cell(colsFor[0], passthrough("Name")); err != nil {
		return nil, err
	}
	for i, c := range cols {
		if header[i+1], err = cell(colsFor[i+1], passthrough(html.EscapeString(c.label))); err != nil {
			return nil, err
		}
	}
	tbl.AddRow(asciidoctor.TableHeader, header...)
	// Body: one row per exposed element carrying the filter metadata.
	for _, row := range rows {
		us, err := e.md.usages(row.ID, filterID)
		if err != nil {
			return nil, err
		}
		usage := ""
		if len(us) > 0 {
			usage = us[0].ID
		}
		name, err := e.names.name(row.ID)
		if err != nil {
			name = row.Name() // a read hiccup: use the raw name
		}
		cells := make([]*asciidoctor.Cell, len(colsFor))
		link, err := e.linkTo(parent, from, row.ID, html.EscapeString(name))
		if err != nil {
			link = html.EscapeString(name)
		}
		if cells[0], err = cell(colsFor[0], passthrough(link)); err != nil {
			return nil, err
		}
		for i, c := range cols {
			if cells[i+1], err = cell(colsFor[i+1], passthrough(valueOf(usage, c.attr.ID))); err != nil {
				return nil, err
			}
		}
		tbl.AddRow(asciidoctor.TableBody, cells...)
	}
	tbl.AddRole(CatalogRole)
	tbl.AssignColumnWidths()
	return tbl, nil
}
