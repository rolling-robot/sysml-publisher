package macros

import (
	"fmt"
	"slices"
	"strings"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/model"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

// Name is the inline macro sysml-name:[<element>], replaced by an element's
// name (MC-8): the page's element without a target, else the element the
// target names (MC-15), such as sysml-name:Model::AircraftPart[] or
// sysml-name:ap[]. A target may also name metadata applied to an element and
// its attributes: sysml-name:@m[] gives the metadata def's name (m),
// sysml-name:@m::owner[] the attribute's (owner), for the page's element or,
// with a path before the @, another one (wing::@m::owner).
const Name = "sysml-name"

func registerName(g *asciidoctor.ExtensionGroup, t *tree.Tree) {
	registerText(g, t, Name,
		func(n *tree.Node) string { return n.Name },
		func(md *metadataIndex, id string) string { return md.names.name(id) })
}

// nameTarget is the element a sysml-name target names: an element (MC-15),
// or metadata applied to an element and one of its attributes, written
// [<element>::]@<metadata>[::<attribute>] (MC-8): the metadata def, or the
// attribute.
func nameTarget(target string, n *tree.Node, md *metadataIndex) (string, error) {
	ref, err := parseMetadataRef(target, n, md)
	switch {
	case err != nil:
		return "", err
	case ref == nil:
		return md.names.resolve(target, n, nil)
	case ref.attr != nil:
		return ref.attr.ID(), nil
	}
	return ref.def.ID(), nil
}

// metadataRef is what a target [<element>::]@<metadata>[::<attribute>]
// names: metadata applied to an element, and one of its attributes.
type metadataRef struct {
	element string          // the element carrying the metadata
	def     model.Element   // the metadata def
	usages  []model.Element // the element's usages of def, at least one
	attr    model.Element   // nil when the target names no attribute
}

// parseMetadataRef reads a target of the form
// [<element>::]@<metadata>[::<attribute>]; it is nil, without error, for a
// target with no @ segment. The element defaults to the page's; the metadata
// is a metadata def's name or short name; the attribute is one of the def's
// attributes, its supertypes' included, by name or short name.
func parseMetadataRef(target string, n *tree.Node, md *metadataIndex) (*metadataRef, error) {
	segs := splitQualified(target)
	at := slices.IndexFunc(segs, func(s string) bool { return strings.HasPrefix(s, "@") })
	if at < 0 {
		return nil, nil
	}
	meta := strings.TrimPrefix(segs[at], "@")
	if meta == "" || len(segs) > at+2 {
		return nil, fmt.Errorf("write [<element>::]@<metadata>[::<attribute>]")
	}
	element, elementName := n.ID, n.QualifiedName
	if at > 0 {
		quoted := make([]string, at)
		for i, seg := range segs[:at] {
			quoted[i] = "'" + strings.ReplaceAll(seg, "'", `\'`) + "'"
		}
		id, err := md.names.resolve(strings.Join(quoted, "::"), n, nil)
		if err != nil {
			return nil, err
		}
		element, elementName = id, md.names.qualifiedName(id)
	}
	def, err := md.definition(meta, n)
	if err != nil {
		return nil, err
	}
	ref := &metadataRef{element: element, def: def, usages: md.usages(element, def)}
	if len(ref.usages) == 0 {
		return nil, fmt.Errorf("%s has no @%s metadata", elementName, md.names.name(def.ID()))
	}
	if at == len(segs)-1 {
		return ref, nil
	}
	attr := segs[at+1]
	for _, a := range md.attributes(def) {
		if slices.Contains(md.names.names(a.ID()), attr) {
			ref.attr = a
			return ref, nil
		}
	}
	return nil, fmt.Errorf("metadata def %s has no attribute %q", md.names.name(def.ID()), attr)
}
