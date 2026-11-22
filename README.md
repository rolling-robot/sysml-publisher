# sysml-publish

`sysml-publish` turns a SysML v2 textual model into a static HTML website. You
write the documentation inside the model, as AsciiDoc textual representations
(`rep language "asciidoc"`), and the tool gives every documented element a page,
arranged as the model's containment tree, with that tree as a navigation
sidebar on every page. Small `sysml-*` macros pull live data from the model into
the text: names, `doc` comments, member lists, metadata tables and evaluated
attribute values with their units.

The tool does not parse the model itself and does not hold one in memory: it
sends the files' text to a [sysml-grpc](https://github.com/Open-MBEE/OpenSysML)
service, which parses the model, holds it, and answers the tool's reads over
the SysML v2 API surface it serves. The AsciiDoc is rendered by
[asciidoctorg](https://github.com/rolling-robot/asciidoctorg). No Ruby
installation is needed; the output is plain HTML and CSS that opens straight
from disk.

## Quick start

Install the tool (see [Requirements and building](#requirements-and-building)):

```sh
go install github.com/rolling-robot/sysml-publisher/cmd/sysml-publish@latest
```

Run it with `-api local` and the tool brings up its own sysml-grpc for the
run — found on your `PATH`, or built from the OpenSysML repository — and
stops it after:

```sh
sysml-publish -api local
```

Or start a service yourself, from the OpenSysML repository, and name it:

```sh
git clone https://github.com/Open-MBEE/OpenSysML && cd OpenSysML
go run ./cmd/sysml-grpc        # serves localhost:50051 until you stop it

sysml-publish -api localhost:50051
```

Put a model in `./model`. The example in this repository is a five-file
fixed-wing UAV model, one top-level package per file. The snippets below are
shortened: the full files are in [`model/`](model/).

`model/definitions.sysml` holds the shared part template and the component
definitions. Every part inherits `UavPart`'s representation, so every page gets
the same name, doc, metadata, value and figure sections:

```sysml
package UavDefinitions {
    private import ScalarValues::*;
    private import SI::*;

    part def <up> UavPart {
        rep language "asciidoc"
        /*
        = sysml-name:[] (sysml-short-name:[])

        sysml-doc::[]

        == Properties

        [cols="1,1"]
        |===
        |mass | sysml-value:mass[]
        |power | sysml-value:power[]
        |===

        == Figure

        sysml-fig::[]
        */
        attribute mass : ISQ::MassValue default 0.1 [kg];
        attribute power : ISQ::PowerValue default 0 [W];
    }

    part def Battery :> UavPart {
        doc /*
        The flight battery stores the energy for the electric motor and the
        avionics.
        */
        attribute :>> mass = 8 [kg];
        attribute capacity : ISQ::ElectricChargeValue default 22 [A*h];
        attribute voltage : ISQ::ElectricPotentialValue default 50.4 [V];
        port powerOut : UavLinks::PowerPort;
    }
}
```

`model/uavkinds.sysml` defines the general air vehicle the system specialises:

```sysml
package UavKinds {
    private import ScalarValues::*;
    private import SI::*;
    private import UavDefinitions::*;

    part <U> uav : UavDefinitions::UavPart {
        attribute :>> mass = 50 [kg];
        attribute cruiseSpeed : ISQ::SpeedValue default 25 [m/s];
        attribute endurance : ISQ::DurationValue default 90 [min];
        attribute range : ISQ::LengthValue = endurance * cruiseSpeed;
        attribute emptyMass : ISQ::MassValue default 42 [kg];
        attribute payloadCapacity : ISQ::MassValue default 8 [kg];
    }
}
```

`model/system.sysml` puts the UAV and its ground remote control in one part,
so the links have a common owner and the system view can draw them:

```sysml
package System {
    private import ScalarValues::*;
    private import SI::*;
    private import UavDefinitions::*;
    private import Views::*;
    private import StandardViewDefinitions::*;

    part uavSystem {
        part uav :> UavKinds::uav {
            // fuselage, wing, tail, landing gear, motor, propeller,
            // flightController, imu, gps, payloadBay ...
            part battery : Battery { port :>> powerOut : UavLinks::PowerPort; }
            part powerDistribution : PowerDistribution {
                port :>> battery : UavLinks::PowerPort;
                port :>> bec : UavLinks::PowerPort;
                port :>> motorController : UavLinks::PowerPort;
            }
            part bec : Bec {
                port :>> powerIn : UavLinks::PowerPort;
                port :>> flightController : UavLinks::PowerPort;
            }
            part motorController : MotorController {
                port :>> powerIn : UavLinks::PowerPort;
                port :>> motor : UavLinks::PowerPort;
            }
            part canBus : CanBus {
                port :>> flightController : UavLinks::CanPort;
                port :>> motorController : UavLinks::CanPort;
            }
            part payloadRadio : Payloads::PayloadRadio {
                port :>> rf : UavLinks::RfPort;
                port :>> data : Payloads::DataPort;
            }
            part commandAntenna : Antenna { port :>> rf : UavLinks::RfPort; }
            part dataAntenna : Antenna { port :>> rf : UavLinks::RfPort; }

            interface batteryPower connect (battery.powerOut, powerDistribution.battery);
            interface becPower connect (powerDistribution.bec, bec.powerIn);
            interface motorControllerPower connect (powerDistribution.motorController, motorController.powerIn);
            interface motorPower connect (motorController.motor, motor.powerIn);
            interface flightControllerCan connect (canBus.flightController, flightController.can);
            interface payloadRfLink connect (payloadRadio.rf, dataAntenna.rf);
        }
        part remoteControl {
            part commandReceiver : CommandReceiver { port :>> rf : UavLinks::RfPort; }
            part dataReceiver : DataReceiver { port :>> rf : UavLinks::RfPort; }
        }
        interface commandLink connect (uav.commandAntenna.rf, remoteControl.commandReceiver.rf);
        interface dataLink connect (uav.dataAntenna.rf, remoteControl.dataReceiver.rf);

        view systemInterfaces {
            expose uavSystem;
            render asInterconnectionDiagram;
        }
    }
}
```

The remaining files are `payloads.sysml` (a generic `Payload`, a
`CameraPayload` specialising it, and the `PayloadRadio` module) and
`links.sysml` (the `RfPort`, `ConsolePort`, `CanPort` and `PowerPort`
definitions).

Run it and open the result:

```sh
sysml-publish -api localhost:50051    # reads ./model, writes ./site
xdg-open site/index.html              # or open it in any browser
```

(Or run `sysml-publish -api local` and skip starting the service yourself:
the tool starts one for the run and stops it after.)

You get 53 pages, one per documented element plus the empty package ancestors,
arranged as the model's containment tree:

- `site/UavDefinitions/UavPart/index.html`, titled "UavPart (up)": the
  definition whose representation every part inherits.
- `site/UavDefinitions/Battery/index.html`: the battery definition with its
  electrical data, and `site/System/uavSystem/uav/battery/index.html`: the
  same representation filled in for the part, with its metadata bindings.
- `site/System/uavSystem/uav/index.html`, titled "uav (U1)": the air vehicle.
  It shows its own representation first (including the 16 internal interfaces
  and an avionics diagram), then the inherited `UavKinds::uav` flight data,
  then the inherited `UavPart` template; the values speak about the air
  vehicle: mass 50 kg, range 135000 m.
- `site/System/uavSystem/uav/wing/index.html`, titled "wing (W)": span 5 m,
  area 1.5 m², mean chord 0.3 m, aspect ratio 16.7, and the composite
  `size` value as `[composite]` with a warning.
- `site/System/uavSystem/uav/canBus/index.html`: the CAN bus and its five
  connections to the flight controller, IMU, GPS, payload radio and motor
  controller.
- `site/System/uavSystem/uav/powerDistribution/index.html`,
  `.../bec/index.html` and `.../motorController/index.html`: the power line,
  from the battery to the BEC and the motor controller.
- `site/System/uavSystem/uav/payloadRadio/index.html`: the payload radio,
  linked to the payload, the data antenna, the CAN bus and the BEC.
- `site/System/uavSystem/commandLink/index.html`: one of the three interface
  pages. Its figure frames the whole system and draws the command link from
  the UAV's antenna to the ground receiver.
- `site/System/uavSystem/index.html`, titled "uavSystem (us)": the system's
  metadata table, its three interfaces, its system diagram and its parts
  table.

`sysml-fig` needs a part with an owned port, or an interface. A part without
ports leaves a placeholder and logs a warning; the run still succeeds. The
`UavPart` template, the airframe parts, the propeller and the UAV itself are
in that group, so the example shows both the figure and the failure mode.

### Example output

[`docs/example/`](docs/example/) holds the site generated from the full
[`model/`](model/), committed so you can look before building anything:

- [landing page](docs/example/index.html)
- [UavPart](docs/example/UavDefinitions/UavPart/index.html): the definition
  whose representation every part inherits
- [uavSystem](docs/example/System/uavSystem/index.html): the whole system,
  with both views
- [uav](docs/example/System/uavSystem/uav/index.html): three representations,
  own first, with its 16 internal interfaces and avionics diagram
- [wing](docs/example/System/uavSystem/uav/wing/index.html): the geometry and
  the composite `size`
- [battery](docs/example/System/uavSystem/uav/battery/index.html): the
  inherited technical data and metadata
- [commandLink](docs/example/System/uavSystem/commandLink/index.html): the
  interface figure
- [canBus](docs/example/System/uavSystem/uav/canBus/index.html): the CAN bus
  and its five connections
- [powerDistribution](docs/example/System/uavSystem/uav/powerDistribution/index.html):
  the power line, with the BEC and the motor controller
- [payloadRadio](docs/example/System/uavSystem/uav/payloadRadio/index.html):
  the payload radio and its four links
- [CameraPayload](docs/example/Payloads/CameraPayload/index.html): a payload
  definition specialising the generic one

Open them from a clone in a browser (`xdg-open docs/example/index.html`): a
repository browser such as GitHub shows HTML files as source, not as pages.
After changing the model or the tool, regenerate them with:

```sh
go run ./cmd/sysml-publish -api local -model model -out docs/example   # in a clone
```

## Requirements and building

You need Go 1.27.2 or newer and a running sysml-grpc service (see
[Quick start](#quick-start)). Everything else, including the OpenSysML client
and asciidoctorg, is fetched by Go.

Install the command:

```sh
go install github.com/rolling-robot/sysml-publisher/cmd/sysml-publish@latest
```

This puts `sysml-publish` in `$(go env GOPATH)/bin`; make sure that folder is
on your `PATH`.

Or build it from a clone:

```sh
git clone https://github.com/rolling-robot/sysml-publisher.git
cd sysml-publisher
go build -o bin/sysml-publish ./cmd/sysml-publish
go test ./...        # optional: starts a sysml-grpc and runs the fixtures through it
```

The binary is large (about 250 MB): it embeds a Ruby interpreter with
Asciidoctor, so rendering needs nothing else at run time; the model is the
service's to parse and hold.

## Documenting the model

Documentation lives in the model, in textual representations whose language is
`asciidoc` or `adoc` (any letter case):

```sysml
part def Pump {
    rep language "asciidoc"
        /*
        = Pump

        Moves coolant through the loop. See the sizing notes below.
        */
}
```

The body between `/*` and `*/` is ordinary [AsciiDoc](https://docs.asciidoctor.org/asciidoc/latest/):
headings, lists, tables, admonitions, source blocks and so on. An element may
have several representations; they appear on its page in the order they are
declared.

A `doc` comment is not a page on its own: it is the element's description,
which a representation can include with the `sysml-doc::[]` macro. Doc comments
are AsciiDoc too.

Representations in other languages (`rep language "markdown"`, ...) are skipped
with a warning.

## Which elements get pages

- **Documented elements**: an element gets a page when it has an AsciiDoc
  representation of its own, or inherits one.
- **Inherited representations**: definitions and usages inherit the
  representations of what they specialize: a usage those of its type
  (`part wing : UavPart`), a definition those of its supertypes
  (`part def Sport :> Wheel`), a usage those of what it subsets or redefines,
  at any depth. A page shows its own representations first, then the
  inherited ones, nearest first, each once, with no label saying where they
  come from.
- **Empty pages for ancestors**: a package or other element that has no
  documentation but contains documented elements gets a page with only its
  name, so the tree has no gaps. Branches with nothing documented in them are
  left out.
- **Unnamed elements** (such as `part : Wheel;`) are called "Unnamed", their
  page folders are `unnamed-1`, `unnamed-2`, ... among their siblings, and the
  tool warns about each one.

## Page titles

A page's title is the document title (the `= Title` line at the top) of the
first of its representations that has one. Macros in it are resolved for the
page, so `= Wheel: sysml-name:[]` gives each wheel its own title.

- A page whose representations have no title is called `untitled-1`,
  `untitled-2`, ... among its siblings.
- When siblings end up with the same title, later ones get a suffix: `Same`,
  `Same-2`, `Same-3`.
- The title is shown as the page heading, in the browser tab and in the
  sidebar. The representation that supplied it does not repeat it; titles of
  further representations stay in their text.
- Empty ancestor pages are titled with the element's name.

Titles never move pages: folders follow element names (see below), so retitling
a representation keeps every link working.

## The generated site

The tree, shortened to the pages the example discusses:

```
site/
├── index.html                      landing page: the whole tree
├── UavDefinitions/
│   ├── index.html                  page of UavDefinitions
│   ├── UavPart/index.html          page of UavDefinitions::UavPart
│   ├── Battery/index.html          page of UavDefinitions::Battery
│   ├── PowerDistribution/index.html
│   ├── Bec/index.html
│   ├── MotorController/index.html
│   └── CanBus/index.html
├── UavKinds/
│   ├── index.html                  page of UavKinds
│   └── uav/index.html              page of UavKinds::uav
├── Payloads/
│   ├── Payload/index.html          page of Payloads::Payload
│   ├── CameraPayload/index.html    page of Payloads::CameraPayload
│   └── PayloadRadio/index.html     page of Payloads::PayloadRadio
├── System/
│   ├── index.html                  page of System
│   └── uavSystem/
│       ├── index.html              page of System::uavSystem
│       ├── uav/
│       │   ├── index.html          page of System::uavSystem::uav
│       │   ├── wing/index.html     page of ...::uav::wing
│       │   ├── canBus/index.html
│       │   ├── powerDistribution/index.html
│       │   ├── bec/index.html
│       │   ├── motorController/index.html
│       │   └── payloadRadio/index.html
│       ├── remoteControl/index.html
│       └── commandLink/index.html
├── assets/                         asciidoctor.css, layout.css, sidebar.js,
│                                   sort.js, and mermaid.min.js when a page
│                                   shows a figure
└── .sysml-publish                  marker: this folder was written by the tool
```

- The page of `A::B::C` is `site/A/B/C/index.html`. A page keeps its path when
  unrelated parts of the model change (except the numbered `unnamed-N`
  folders).
- Every page has the whole tree as a sidebar on the left, with the current
  page marked; items with children are collapsible, and the browser
  remembers which ones you opened.
- All links are relative, so the site works from `file://`, from any web
  server, and from any sub-path.
- With `baseURL` configured, every page also carries a canonical link naming
  its absolute URL under it (see [Configuration file](#configuration-file)).
- **The output folder is replaced on every run**, so pages of removed elements
  disappear. To protect your files, the tool only replaces a folder that is
  empty, does not exist, or carries the `.sysml-publish` marker from an earlier
  run; it refuses any other folder.

## Macros

Macros are written in the representation's AsciiDoc and are replaced by data
from the model. They always describe the **page's element**: on a page that
shows an inherited representation, they speak about that page's element, not
the element that owns the representation.

| Macro | Kind | Replaced by |
|---|---|---|
| `sysml-name:[<element>]` | inline | the element's name |
| `sysml-name:@<metadata>::<attribute>[]` | inline | the name of metadata on the element, or of its attribute |
| `sysml-qualified-name:[<element>]` | inline | its qualified name, `System::uavSystem::uav::wing` |
| `sysml-short-name:[<element>]` | inline | its short name, the `<W>` in `part def <W> Wheel` |
| `sysml-rep:[<element>][<language>]` | inline | the body of the element's representation in another language, as plain text |
| `sysml-value:<attribute>[]` | inline | the attribute's value with its unit, `8 kg`; also another element's attribute (`System::uavSystem::uav::battery`) by a qualified or relative name, or a feature chain (`battery.voltage`); a metadata attribute: `@m::owner` |
| `sysml-doc::[<element>][]` | block | the element's `doc` comments |
| `sysml-members-list::[]` | block | a list of its members with their types |
| `sysml-members-list::<element>[]` | block | its members typed by, subclassing or subsetting `<element>` |
| `sysml-members-list::<kind>[]` | block | its members of one kind: `partUsage`, `interfaceUsage`, ... |
| `sysml-iface-ends::[<interface>][depth=<n>\|all][name=short\|long]` | block | the ends of an interface or connection, source first: `engine.air`; `depth` shows the last n segments of the port's qualified path (`engine::air`), `all` all of it; `name=long` names each segment by its name (short names are the default) |
| `sysml-ifaces::[<element>][name=short\|long]` | block | the interfaces and connections the element's ports are on: `i1 -> y`, or `i3 -> y, z` for the other parts |
| `sysml-metadata-table::<metadata def>[]` | block | a table of that metadata's attributes on the element |
| `sysml-view::[<view>][]` | block | the view as OpenSysML renders it: a diagram as an SVG picture, a table view as a table; a diagram needs Graphviz's `dot` |
| `sysml-fig::[<element>][]` | block | a Mermaid figure of an interface or a part: an interface's ends' common ancestor as the frame with an edge per pair of ends, or a part with its owned ports and the connections they are on; a part without ports fails with a placeholder |

Inline macros go anywhere in text; block macros go on a line of their own.
Without a target, `sysml-name`, `sysml-qualified-name`, `sysml-short-name`,
`sysml-doc` and `sysml-rep` describe the page's element; give them an element
(`sysml-name:battery[]`, `sysml-doc::UavDefinitions::UavPart[]`) to describe that
one instead.
Where a macro takes an element, name it as SysML would: by qualified name,
by a name relative to the page's element, or by a name only one element has;
short names work in every part of a name (`UavDefinitions::up` for
`UavDefinitions::UavPart`).

```sysml
part def Pump {
    attribute flowRate = 50 [L/min];
    port inlet : FluidPort;
    port outlet : FluidPort;
    rep language "asciidoc"
        /*
        = Pump sysml-short-name:[]

        sysml-doc::[]

        Flow rate:: sysml-value:flowRate[]

        == Members

        sysml-members-list::[]
        */
}
```

On `Pump`'s page this gives the heading "Pump P1" (if `Pump` is declared
`part def <P1> Pump`), the doc comment, "Flow rate: 50 L/min", and the list
flowRate, inlet (FluidPort), outlet (FluidPort).

The full reference, with every option, rule and example, is
[docs/macros.adoc](docs/macros.adoc).

### Using the macros in another Go program

The macros are a public package, `github.com/rolling-robot/sysml-publisher/macros`,
so another tool can put model data into its own AsciiDoc. It reads the model
through the public package `github.com/rolling-robot/sysml-publisher/sysml`,
and tells the macros where its pages are, for links:

```go
client, _ := opensysml.New()                       // OpenSysML, in-process
parsed, _ := client.ParseFiles(ctx, files)
m, _ := sysml.Open(ctx, client, parsed)            // keep open while converting
defer m.Close()

ad, _ := asciidoctor.New()                         // github.com/rolling-robot/asciidoctorg
err := macros.Register(ctx, ad, m, pages, warn)    // pages: Href(fromID, toID) (string, bool)
out, err := ad.Convert(body, &asciidoctor.Options{Attributes: macros.Attributes(elementID)})
```

`pages` may be nil (no links); `warn` receives each failing macro. A
complete, runnable version is `Example` in
[macros/example_test.go](macros/example_test.go).

`Register` returns a `*macros.Registry`, which adds macros of your own
(`Inline`, `Block`). They behave like the `sysml-*` ones: a `Call` gives the
page's element, the target read as `sysml-name` reads it, links to your
pages, evaluation and warnings, and an error leaves a placeholder and a
warning instead of stopping the conversion:

```go
r, _ := macros.Register(ctx, ad, m, pages, warn)
r.Inline("my-mass", func(c *macros.Call) (string, error) {
	el, err := c.Element()             // the page's element, or the one the target names
	if err != nil {
		return "", err                 // placeholder + warning
	}
	v, err := c.Evaluate(el, "mass")
	if err != nil {
		return "", err
	}
	return c.Link(el.ID, html.EscapeString(macros.FormatValue(v))), nil
})
```

Names starting with `sysml-` are reserved for this tool's macros. See
`ExampleRegistry_Inline` in the same file.

## Configuration file

A `sysml-publish.toml` in the working directory (or the file `-config` names)
names the folders and the site's metadata. A flag you give wins over the file,
and the file over the built-in defaults:

```toml
[source]
dir = "model"                # the model folder (the -model default)
api = "localhost:50051"      # the SysML service's address (the -api flag)

[site]
dir = "site"                 # the output folder (the -out default)
title = "UAV documentation"          # the site's title: landing page, sidebar, browser tabs
baseURL = "https://example.com/uav/"  # the URL the site is published under
```

A commented example to copy is [`docs/sysml-publish.toml`](docs/sysml-publish.toml).

- Paths are relative to the working directory, like the flags' values are.
- Without the file (or without a key in it), the flags and defaults apply:
  `./model`, `./site`, the title "Model documentation". The service's
  address has no default: `-api` or `[source] api` must name one.
- `baseURL` must be an absolute URL. When it is set, every page carries
  `<link rel="canonical" href="…">` naming that page's URL under it; the
  links between pages stay relative, so the site still opens from disk.
- These six keys are the only ones; any other key is a usage error, so a
  misspelled one is reported instead of silently ignored.
- `sysml-publish.toml` found by name may be absent; a file named with
  `-config` must exist.

## Command line

```
sysml-publish -api ADDR [-config FILE] [-model DIR] [-out DIR] [-v] [-log-format text|json] [-dot CMD]
```

| Flag | Default | Meaning |
|---|---|---|
| `-api ADDR` | none — required | address of the sysml-grpc service that parses the model and answers the reads, spelled as `localhost:50051` or `https://sysml.example.com`; `local` starts one for the run (from `PATH` or built from the OpenSysML repository) and stops it after; the service's bearer token is read from `$OPENSYSML_GRPC_TOKEN` |
| `-config FILE` | `sysml-publish.toml` in the working directory, if present | the [configuration file](#configuration-file) |
| `-model DIR` | `./model` | folder with the model; every `*.sysml` and `*.kerml` file in it, searched recursively, is read as one model, so references across files work; hidden files and folders (`.git`, editor lock files) are skipped |
| `-out DIR` | `./site` | folder to write the site to; replaced on every run |
| `-v` | off | also report how long each stage and each page took |
| `-log-format` | `text` | `text` for people, `json` for one JSON object per line, for CI and log tools |
| `-dot CMD` | `dot` from `PATH` | Graphviz's `dot` command, which lays out the diagrams `sysml-view` shows; a diagram needs it, a table view needs no program |

Everything the tool reports goes to stderr as log records (here for the UAV
model, with the composite warning and one of the figure warnings):

```
level=INFO msg="model loaded" folder=model files=5 elements=395 warnings=0 duration=3.224110373s
level=INFO msg="page written" element=System::uavSystem::uav::wing empty=false path=docs/example/System/uavSystem/uav/wing/index.html
level=WARN msg="model/system.sysml:119: System::uavSystem::uav::wing: sysml-value:size[]: size is composite; shown as [composite]"
level=WARN msg="model/definitions.sysml:66: UavDefinitions::UavPart: sysml-fig::[]: has no ports; a figure needs at least one"
level=INFO msg="site published" pages=53 landing=docs/example/index.html duration=52.88812304s
```

Exit status:

| Status | Meaning |
|---|---|
| 0 | the site was written (possibly with warnings) |
| 1 | the run failed; nothing was written |
| 2 | wrong command-line usage, a problem in the config file (an unknown key, a `baseURL` that is not an absolute URL, a missing `-config` file) |
| 130 | interrupted with Ctrl-C or SIGTERM; nothing was written |

## Errors and warnings

**Stop the run, write nothing, exit 1:**

- errors in the model, each reported with its location
  (`msg="model error" diagnostic="model/bad.sysml:2:14: ..."`);
- a model folder with no model files;
- an output folder that is not empty and was not written by the tool.

**Warn and carry on:**

- a macro that cannot do what it was asked (an attribute or metadata def the
  model does not have, an element name that is unknown or names several
  elements): the page shows a placeholder with the macro and the problem,
  and the warning names the file and line where the macro is written, the
  page and the macro (`model/parts.sysml:5: Model::pd: sysml-value:mas[]:
  ...`); every failing macro on every page is reported;
- Asciidoctor's own warnings about a representation, such as a skipped
  section level, with the page's element and the model line
  (`msg="asciidoc: section title out of sequence: ..." element=Model::pd
  location=model/parts.sysml:7`);
- a representation in another language than AsciiDoc;
- an unnamed element with a page;
- a `sysml-value` whose attribute is composite (made of attributes of its
  own), shown as `[composite]`; one warning per page and macro;
- a `sysml-fig` target that is a part without an owned port, shown as a
  placeholder; one warning per page and macro;
- a model with no documented elements at all: only the landing page is
  written, and the run still succeeds.

## Limitations

This is a first version.

- Models are read from files only; reading from a SysML v2 API server is
  planned.
- Views (`sysml-view`) are rendered by OpenSysML in process; a diagram is
  laid out by Graphviz, which must be installed where the site is built.
- A failing macro leaves a placeholder and a warning; the run carries on.
- The tool reads models through OpenSysML's SysML v2 API queries and element
  reads, which are not in an OpenSysML release yet: it is built against a
  development commit of OpenSysML.

## Further reading

- [docs/macros.adoc](docs/macros.adoc): the macro reference.
- [requirements.org](requirements.org): the requirements, decisions and open
  questions, with the IDs the code and tests refer to.
- [AGENTS.md](AGENTS.md): how the program is built, for anyone changing it.
