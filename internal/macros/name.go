package macros

import (
	"fmt"
	"slices"
	"strings"

	asciidoctor "github.com/rolling-robot/asciidoctorg"
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
// [<element>::]@<metadata>[::<attribute>] (MC-8). The element defaults to
// the page's; the metadata is a metadata def's name or short name; the
// attribute is one of the def's attributes, its supertypes' included.
func nameTarget(target string, n *tree.Node, md *metadataIndex) (string, error) {
	segs := splitQualified(target)
	at := slices.IndexFunc(segs, func(s string) bool { return strings.HasPrefix(s, "@") })
	if at < 0 {
		return md.names.resolve(target, n, nil)
	}
	meta := strings.TrimPrefix(segs[at], "@")
	if meta == "" || len(segs) > at+2 {
		return "", fmt.Errorf("write [<element>::]@<metadata>[::<attribute>]")
	}
	element, elementName := n.ID, n.QualifiedName
	if at > 0 {
		quoted := make([]string, at)
		for i, seg := range segs[:at] {
			quoted[i] = "'" + strings.ReplaceAll(seg, "'", `\'`) + "'"
		}
		id, err := md.names.resolve(strings.Join(quoted, "::"), n, nil)
		if err != nil {
			return "", err
		}
		element, elementName = id, md.names.qualifiedName(id)
	}
	def, err := md.definition(meta, n)
	if err != nil {
		return "", err
	}
	if len(md.usages(element, def)) == 0 {
		return "", fmt.Errorf("%s has no @%s metadata", elementName, md.names.name(def.ID()))
	}
	if at == len(segs)-1 {
		return def.ID(), nil
	}
	attr := segs[at+1]
	for _, a := range md.attributes(def) {
		if slices.Contains(md.names.names(a.ID()), attr) {
			return a.ID(), nil
		}
	}
	return "", fmt.Errorf("metadata def %s has no attribute %q", md.names.name(def.ID()), attr)
}
