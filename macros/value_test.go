package macros

import (
	"context"
	"math/big"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/Open-MBEE/OpenSysML/client/opensysml"
	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/internal/model"
	"github.com/rolling-robot/sysml-publisher/internal/tree"
)

var paragraphRx = regexp.MustCompile(`<p>([^<]*)</p>`)

// The expectations are the // notes in testdata/values/values.sysml.
func TestValue(t *testing.T) {
	m, err := model.Folder{Dir: filepath.Join("..", "testdata", "values")}.Load(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	tr, err := tree.Build(t.Context(), m)
	if err != nil {
		t.Fatal(err)
	}
	ad, err := asciidoctor.New()
	if err != nil {
		t.Fatal(err)
	}
	defer ad.Close()
	var warnings []string
	if err := Register(context.Background(), ad, tr, func(w Warning) { warnings = append(warnings, w.String()) }); err != nil {
		t.Fatal(err)
	}

	common := []string{
		"volume flow: 2.5 m³⋅s⁻¹",
		"speed: 90 km/h",
		"top speed: 5.555555555555555 m/s",
		"ratio: 0.25",
		"ok: true",
		"position: [composite]",
		"note: (no value)",
	}
	cases := map[string][]string{
		"Values/typical": append([]string{"mass: 5 kg", "count: 3", "label: P-1", "level: Level::low", "motor power: 750 W"}, common...),
		"Values/custom":  append([]string{"mass: 7.5 kg", "count: 4", "label: P-2", "level: Level::high", "motor power: 1.1 kW"}, common...),
	}
	for dir, want := range cases {
		n := pageAt(t, tr, dir)
		if len(n.Inherited) != 1 {
			t.Fatalf("%s: want Pump's representation inherited", dir)
		}
		out, err := ad.Convert(n.Inherited[0].Body, &asciidoctor.Options{Attributes: Attributes(n)})
		if err != nil {
			t.Fatalf("%s: %v", dir, err)
		}
		got := map[string]bool{}
		for _, p := range paragraphRx.FindAllStringSubmatch(out, -1) {
			got[p[1]] = true
		}
		for _, w := range want {
			if !got[w] {
				t.Errorf("%s: missing %q in %v", dir, w, got)
			}
		}
	}
	want := []string{
		"Values::typical: sysml-value:position[]: position is composite; shown as [composite]",
		"Values::custom: sysml-value:position[]: position is composite; shown as [composite]",
	}
	for _, w := range want {
		found := false
		for _, g := range warnings {
			found = found || g == w
		}
		if !found {
			t.Errorf("warning %q not reported; got %v", w, warnings)
		}
	}
}

// The target names a feature, with the page's element as subject: its
// attributes, qualified or relative names of other elements' attributes,
// feature chains and quoted names with spaces (MC-14).
func TestValueTargets(t *testing.T) {
	tr, ad := setupFixture(t, "values")
	page := Attributes(pageAt(t, tr, "Values/typical"))
	cases := []struct{ name, target, want string }{
		{"own attribute", "count", "3"},
		{"quoted name with a space", "'rated power'", "2 kW"},
		{"qualified name, another element", "Values::custom::count", "4"},
		{"relative name, another element", "custom::count", "4"},
		{"feature chain, another element", "custom.count", "4"},
		{"quoted segment in a qualified name", "Values::Pump::'rated power'", "2 kW"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := ad.Convert("V: sysml-value:"+c.target+"[].", &asciidoctor.Options{Attributes: page})
			if err != nil {
				t.Fatal(err)
			}
			if want := "<p>V: " + c.want + ".</p>"; !strings.Contains(out, want) {
				t.Errorf("output %q does not contain %q", out, want)
			}
		})
	}
}

// The target is a feature, not an expression: operators, literals and
// calls are refused with our error, before OpenSysML evaluates anything
// (MC-14). count<5 also checks that the escaped < is read back as <.
func TestValueRejectsExpressions(t *testing.T) {
	tr, ad := setupFixture(t, "values")
	page := Attributes(pageAt(t, tr, "Values/typical"))
	cases := []struct{ name, target string }{
		{"arithmetic", "count*2"},
		{"comparison with <, escaped by Asciidoctor before the macro", "count<5"},
		{"sum of two features", "count+custom::count"},
		{"literal", "5"},
		{"call", "size(count)"},
		{"trailing separator", "custom::"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ad.Convert("V: sysml-value:"+c.target+"[].", &asciidoctor.Options{Attributes: page})
			want := "sysml-value:" + c.target + "[] on the page of Values::typical: \"" + c.target + "\" is not a feature name"
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("err = %v, want one containing %q", err, want)
			}
		})
	}
}

func TestValueErrors(t *testing.T) {
	m, err := model.Folder{Dir: filepath.Join("..", "testdata", "values")}.Load(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	tr, err := tree.Build(t.Context(), m)
	if err != nil {
		t.Fatal(err)
	}
	ad, err := asciidoctor.New()
	if err != nil {
		t.Fatal(err)
	}
	defer ad.Close()
	if err := Register(context.Background(), ad, tr, nil); err != nil {
		t.Fatal(err)
	}
	page := Attributes(pageAt(t, tr, "Values/typical"))
	for src, want := range map[string]string{
		"sysml-value:missing[]": "sysml-value:missing[] on the page of Values::typical",
		"sysml-value:[]":        "",
	} {
		out, err := ad.Convert(src, &asciidoctor.Options{Attributes: page})
		if want == "" {
			// Without a target the text is not the macro at all (its
			// syntax needs one), so it stays as written.
			if err != nil || !strings.Contains(out, "sysml-value:[]") {
				t.Errorf("%s: out %q err %v, want the text unchanged", src, out, err)
			}
			continue
		}
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v, want one containing %q", src, err, want)
		}
	}
}

func TestFormatValue(t *testing.T) {
	type res struct {
		text string
		kind valueKind
	}
	for _, c := range []struct {
		v    opensysml.Value
		want res
	}{
		{opensysml.Quantity{Magnitude: opensysml.Int(5), Unit: "kg"}, res{"5 kg", valuePlain}},
		{opensysml.Quantity{Magnitude: opensysml.Real(2.5), Unit: "SI::'m³⋅s⁻¹'"}, res{"2.5 m³⋅s⁻¹", valuePlain}},
		{opensysml.Quantity{Magnitude: opensysml.NewRational(big.NewRat(6, 3)), Unit: "SI::'m/s'"}, res{"2 m/s", valuePlain}},
		{opensysml.Quantity{Magnitude: opensysml.Int(3)}, res{"3", valuePlain}},
		{opensysml.MeasurementRef{Unit: "SI::kg"}, res{"kg", valuePlain}},
		{opensysml.Real(0.1), res{"0.1", valuePlain}},
		{opensysml.Bool(false), res{"false", valuePlain}},
		{opensysml.String("a<b"), res{"a<b", valuePlain}},
		{opensysml.EnumLiteral{Name: "Level::low"}, res{"Level::low", valuePlain}},
		{opensysml.Sequence{opensysml.Int(1), opensysml.Int(2)}, res{"1, 2", valuePlain}},
		{opensysml.Sequence{}, res{"", valueNone}},
		{opensysml.Unset{}, res{"", valueNone}},
		{opensysml.Undetermined{}, res{"", valueNone}},
		{opensysml.InstanceID(2), res{"", valueComposite}},
		{opensysml.Sequence{opensysml.InstanceID(2)}, res{"", valueComposite}},
	} {
		text, kind := formatValue(c.v)
		if got := (res{text, kind}); !reflect.DeepEqual(got, c.want) {
			t.Errorf("formatValue(%#v) = %v, want %v", c.v, got, c.want)
		}
	}
}

// A target [<element>::]@<metadata>::<attribute> gives the value of a
// metadata attribute as sysml-metadata-table shows it: bound, else the
// def's, as written in the model (MC-14). The expected values are the //
// notes in testdata/metadata-table.
func TestValueMetadata(t *testing.T) {
	tr, ad := setupFixture(t, "metadata-table")
	page := Attributes(pageAt(t, tr, "Meta/bound"))
	cases := []struct{ name, target, want string }{
		{"bound string, without quotes", "@Info::owner", "pete"},
		{"bound integer", "@Info::count", "7"},
		{"fixed value of the def", "@Info::ratio", "0.5"},
		{"default enumeration literal", "@Info::level", "Level::low"},
		{"default expression, as written", "@Info::sum", "1 + 2"},
		{"no value", "@Info::note", NoValue},
		{"another element, relative", "overridden::@Info::owner", "ann"},
		{"another element, qualified, prefix form #Info", "Meta::defaults::@Info::owner", "nobody"},
		{"short name of the def, supertype's attribute", "specialized::@drv::origin", "here"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out, err := ad.Convert("V: sysml-value:"+c.target+"[].", &asciidoctor.Options{Attributes: page})
			if err != nil {
				t.Fatal(err)
			}
			if want := "<p>V: " + c.want + ".</p>"; !strings.Contains(out, want) {
				t.Errorf("output %q does not contain %q", out, want)
			}
		})
	}
}

func TestValueMetadataErrors(t *testing.T) {
	tr, ad := setupFixture(t, "metadata-table")
	page := Attributes(pageAt(t, tr, "Meta/bound"))
	cases := []struct{ name, target, want string }{
		{"no attribute", "@Info", "no attribute named; write sysml-value:[<element>::]@<metadata>::<attribute>[]"},
		{"unknown attribute", "@Info::missing", `metadata def Info has no attribute "missing"`},
		{"element without the metadata", "plain::@Info::owner", "Meta::plain has no @Info metadata"},
		{"unknown metadata def", "@Nope::owner", `the model has no metadata def "Nope"`},
		{"too many segments", "@Info::owner::x", "write [<element>::]@<metadata>[::<attribute>]"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ad.Convert("V: sysml-value:"+c.target+"[].", &asciidoctor.Options{Attributes: page})
			want := "sysml-value:" + c.target + "[] on the page of Meta::bound: " + c.want
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("err = %v, want one containing %q", err, want)
			}
		})
	}
}
