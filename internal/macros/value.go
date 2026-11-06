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

// Value is the inline macro sysml-value:<feature>[], replaced by the value
// of that feature, units included (MC-14). The target names a feature, not
// an expression: a feature of the page's element (mass, motor.power), a
// qualified name (Values::custom::count) or a name relative to the page's
// element (custom::count). It is evaluated with the page's element as its
// subject.
const Value = "sysml-value"

// featureRefRx is what a sysml-value target may be: names, plain or quoted,
// joined by :: or by the feature chain dot. Anything else (operators,
// literals, calls) is an expression and is refused.
var featureRefRx = regexp.MustCompile(`^(?:[A-Za-z_][A-Za-z0-9_]*|'(?:[^'\\]|\\.)*')(?:(?:::|\.)(?:[A-Za-z_][A-Za-z0-9_]*|'(?:[^'\\]|\\.)*'))*$`)

// valueMacroRx is Asciidoctor's long inline form with a target that may
// hold quoted names with spaces: sysml-value:Pump::'top speed'[]. The target
// still ends at the first "[" or space outside quotes.
const valueMacroRx = `\\?sysml-value:((?:'(?:[^'\\]|\\.)*'|[^\s\['])+)\[(|.*?[^\\])\]`

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
			// Inline macros run after special characters are escaped, so
			// 'a<b' arrives as 'a&lt;b'.
			target = html.UnescapeString(target)
			macro := fmt.Sprintf("%s:%s[]", Value, target)
			if target == "" {
				return nil, fmt.Errorf("%s on the page of %s: no feature named; write %s:<feature>[]", macro, n.QualifiedName, Value)
			}
			if !featureRefRx.MatchString(target) {
				return nil, fmt.Errorf("%s on the page of %s: %q is not a feature name; write a name, a qualified name (A::b) or a feature chain (a.b), not an expression", macro, n.QualifiedName, target)
			}
			// The page's element is the subject, so the feature reads its
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
		}), &asciidoctor.MacroOptions{Regexp: valueMacroRx})
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
