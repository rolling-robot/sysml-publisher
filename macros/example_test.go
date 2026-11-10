// Package macros_test uses only public packages, as another module would.
package macros_test

import (
	"context"
	"fmt"
	"go/build"
	"html"
	"log"
	"strings"
	"testing"

	"github.com/Open-MBEE/OpenSysML/client/opensysml"
	asciidoctor "github.com/rolling-robot/asciidoctorg"
	"github.com/rolling-robot/sysml-publisher/macros"
	"github.com/rolling-robot/sysml-publisher/sysml"
)

// pages is a host's pages: the file of each element that has one, by id.
type pages map[string]string

func (p pages) Href(from, to string) (string, bool) {
	href, ok := p[to]
	return href, ok
}

// Using the macros from another module: parse a model with OpenSysML, open
// it with sysml.Open, register the macros with the host's pages, and
// convert AsciiDoc for an element. A failing macro leaves a placeholder and
// a warning.
func Example() {
	ctx := context.Background()
	client, err := opensysml.New()
	if err != nil {
		log.Fatal(err)
	}
	parsed, err := client.ParseFiles(ctx, []string{"../testdata/external/vehicles.sysml"})
	if err != nil {
		log.Fatal(err)
	}
	m, err := sysml.Open(ctx, client, parsed)
	if err != nil {
		log.Fatal(err)
	}
	defer m.Close()

	// The host decides which elements have pages, by id.
	car, err := m.Props(ctx, "Vehicles::Car", "name")
	if err != nil {
		log.Fatal(err)
	}
	wheel, err := m.Props(ctx, "Vehicles::Wheel", "name")
	if err != nil {
		log.Fatal(err)
	}
	site := pages{car.Element.ID: "car.html", wheel.Element.ID: "wheel.html"}

	ad, err := asciidoctor.New()
	if err != nil {
		log.Fatal(err)
	}
	defer ad.Close()
	warn := func(w macros.Warning) { fmt.Println("warning:", w) }
	if _, err := macros.Register(ctx, ad, m, site, warn); err != nil {
		log.Fatal(err)
	}

	src := "sysml-name:[] weighs sysml-value:mass[], on sysml-name:Wheel[]s.\n\n" +
		"sysml-members-list::Wheel[]\n\n" +
		"Missing: sysml-name:Boat[]."
	out, err := ad.Convert(src, &asciidoctor.Options{Attributes: macros.Attributes(car.Element.ID)})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// warning: Vehicles::Car: sysml-name:Boat[]: the model has no element named "Boat"
	// <div class="paragraph">
	// <p>Car weighs 1200, on <a href="wheel.html">Wheel</a>s.</p>
	// </div>
	// <div class="ulist sysml-members-list">
	// <ul>
	// <li>
	// <p>front</p>
	// </li>
	// <li>
	// <p>rear</p>
	// </li>
	// </ul>
	// </div>
	// <div class="paragraph">
	// <p>Missing: <span class="sysml-failed">sysml-name:Boat[]: the model has no element named &#34;Boat&#34;</span>.</p>
	// </div>
}

// A macro of your own works like the sysml-* ones: Element reads the target
// as sysml-name does (here the short name W), Link links to the host's
// pages, FormatValue shows values as sysml-value does, and an error leaves
// a placeholder and a warning instead of stopping the conversion.
func ExampleRegistry_Inline() {
	ctx := context.Background()
	client, err := opensysml.New()
	if err != nil {
		log.Fatal(err)
	}
	parsed, err := client.ParseFiles(ctx, []string{"../testdata/external/vehicles.sysml"})
	if err != nil {
		log.Fatal(err)
	}
	m, err := sysml.Open(ctx, client, parsed)
	if err != nil {
		log.Fatal(err)
	}
	defer m.Close()
	car, err := m.Props(ctx, "Vehicles::Car", "name")
	if err != nil {
		log.Fatal(err)
	}
	wheel, err := m.Props(ctx, "Vehicles::Wheel", "name")
	if err != nil {
		log.Fatal(err)
	}
	site := pages{car.Element.ID: "car.html", wheel.Element.ID: "wheel.html"}

	ad, err := asciidoctor.New()
	if err != nil {
		log.Fatal(err)
	}
	defer ad.Close()
	r, err := macros.Register(ctx, ad, m, site, func(w macros.Warning) { fmt.Println("warning:", w) })
	if err != nil {
		log.Fatal(err)
	}

	// my-mass:[<element>]: the element's mass, its name linked to its page.
	err = r.Inline("my-mass", func(c *macros.Call) (string, error) {
		el, err := c.Element()
		if err != nil {
			return "", err
		}
		v, err := c.Evaluate(el, "mass")
		if err != nil {
			return "", err
		}
		name, err := c.Name(el.ID)
		if err != nil {
			return "", err
		}
		return c.Link(el.ID, html.EscapeString(name)) + " weighs " + html.EscapeString(macros.FormatValue(v)), nil
	})
	if err != nil {
		log.Fatal(err)
	}

	// On the page of Wheel, so Car's name links to its page.
	out, err := ad.Convert("my-mass:Car[].\n\nmy-mass:Boat[].", &asciidoctor.Options{Attributes: macros.Attributes(wheel.Element.ID)})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(out)
	// Output:
	// warning: Vehicles::Wheel: my-mass:Boat[]: the model has no element named "Boat"
	// <div class="paragraph">
	// <p><a href="car.html">Car</a> weighs 1200.</p>
	// </div>
	// <div class="paragraph">
	// <p><span class="sysml-failed">my-mass:Boat[]: the model has no element named &#34;Boat&#34;</span>.</p>
	// </div>
}

// Other modules cannot import internal packages, so neither may macros:
// the example above compiles inside this module either way, and this test
// is what keeps the package usable from outside.
func TestNoInternalImports(t *testing.T) {
	pkg, err := build.ImportDir(".", 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range pkg.Imports {
		if strings.Contains(imp, "/internal/") || strings.HasSuffix(imp, "/internal") {
			t.Errorf("macros imports %s, which other modules cannot import", imp)
		}
	}
}
