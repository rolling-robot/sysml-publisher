package macros

import (
	"context"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"

	"github.com/Open-MBEE/OpenSysML/client/opensysml"
	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

// Value is the inline macro sysml-value:<attribute>[], replaced by the value
// of that attribute of the page's element, units included (MC-14).
const Value = "sysml-value"

// CompositeText is shown for an attribute whose value is composite: one
// made of attributes of its own, which has no single value to show.
const CompositeText = "[composite]"

func registerValue(ctx context.Context, g *asciidoctor.ExtensionGroup, t *tree.Tree, warn Warn) {
	g.InlineMacro(Value, asciidoctor.InlineMacroProcessorFunc(
		func(parent asciidoctor.Node, target string, _ map[string]any) (*asciidoctor.PhraseNode, error) {
			n, err := pageElement(parent, t)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", Value, err)
			}
			macro := fmt.Sprintf("%s:%s[]", Value, target)
			if target == "" {
				return nil, fmt.Errorf("%s on the page of %s: no attribute named; write %s:<attribute>[]", macro, n.QualifiedName, Value)
			}
			// The page's element is the subject, so the attribute reads its
			// value: its own, a redefinition's, or one from its types.
			v, err := t.Model.Evaluate(ctx, n.QualifiedName, target)
			if err != nil {
				return nil, fmt.Errorf("%s on the page of %s: %w", macro, n.QualifiedName, err)
			}
			text, kind := formatValue(v)
			switch kind {
			case valueComposite:
				warn(Warning{Page: n, Macro: macro, Message: fmt.Sprintf("%s is composite; shown as %s", target, CompositeText)})
				text = CompositeText
			case valueNone:
				text = NoValue
			}
			return asciidoctor.CreatePhraseNode(parent, "quoted", html.EscapeString(text), nil, nil)
		}), nil)
}

type valueKind int

const (
	valuePlain     valueKind = iota
	valueNone                // unset or undetermined
	valueComposite           // an object made of attributes of its own
)

// formatValue is how an evaluated value is shown: a quantity as magnitude
// and unit (5 kg), a string without quotes, an enumeration literal as
// written (Level::low).
func formatValue(v opensysml.Value) (string, valueKind) {
	switch x := v.(type) {
	case nil, opensysml.Unset, opensysml.Undetermined, opensysml.Null:
		return "", valueNone
	case opensysml.InstanceID:
		return "", valueComposite
	case opensysml.Quantity:
		if unit := formatUnit(x.Unit); unit != "" {
			return formatNumber(x.Magnitude) + " " + unit, valuePlain
		}
		return formatNumber(x.Magnitude), valuePlain
	case opensysml.MeasurementRef:
		return formatUnit(x.Unit), valuePlain
	case opensysml.Number:
		return formatNumber(x), valuePlain
	case opensysml.Bool:
		return strconv.FormatBool(bool(x)), valuePlain
	case opensysml.String:
		return string(x), valuePlain
	case opensysml.EnumLiteral:
		return x.Name, valuePlain
	case opensysml.Sequence:
		parts := make([]string, 0, len(x))
		for _, item := range x {
			s, kind := formatValue(item)
			if kind == valueComposite {
				return "", valueComposite
			}
			if kind == valueNone {
				s = NoValue
			}
			parts = append(parts, s)
		}
		if len(parts) == 0 {
			return "", valueNone
		}
		return strings.Join(parts, ", "), valuePlain
	}
	return fmt.Sprint(v), valuePlain
}

// formatNumber writes a magnitude as a reader expects: integers as they
// are, a rational as a decimal (10 [km/h] * 2 is 50/9 m/s, shown 5.555…).
func formatNumber(n opensysml.Number) string {
	switch x := n.(type) {
	case opensysml.Int:
		return strconv.FormatInt(int64(x), 10)
	case opensysml.Real:
		return strconv.FormatFloat(float64(x), 'g', -1, 64)
	case opensysml.Rational:
		r := x.Rat()
		if r.IsInt() {
			return r.Num().String()
		}
		f, _ := r.Float64()
		return strconv.FormatFloat(f, 'g', -1, 64)
	}
	return fmt.Sprint(n)
}

// unitQualifier is a package prefix in a unit name: the SI:: of SI::kg.
var unitQualifier = regexp.MustCompile(`(?:[A-Za-z_][A-Za-z0-9_]*::)+`)

// formatUnit writes a unit as its symbol. The engine names a unit it
// composed by its qualified, quoted name (SI::'m³⋅s⁻¹'); a reader wants
// m³⋅s⁻¹.
func formatUnit(unit string) string {
	unit = unitQualifier.ReplaceAllString(unit, "")
	return strings.ReplaceAll(unit, "'", "")
}
