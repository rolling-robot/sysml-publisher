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

Put a model in `./model`. This one is a shortened `model/aircraft.sysml`, the
example model in this repository:

```sysml
package Model {
    private import SI::*;

    part def <ap> AircraftPart {
        rep language "asciidoc"
            /*
            = sysml-name:[] (sysml-short-name:[])

            sysml-doc::[]

            == Properties

            [cols="1,1"]
            |===
            |weight | sysml-value:mass[]
            |power | sysml-value:power[]
            |===
            */
        attribute mass : ISQ::MassValue default 12 [kg];
        attribute power : ISQ::PowerValue default 1.5 [kW];
    }
    part aircraft : ap {
        rep language "asciidoc"
        /*
        == Purpose

        sysml-doc::[]

        == Owned parts:

        sysml-members-list::PartUsage[]

        == Interfaces:

        sysml-members-list::InterfaceUsage[]
        */

        doc /*
        The purpose of the aircraft is to transport people
        */
        part <a> wing : ap {
            doc /*
            Documentation for part sysml-name:[]

            Owned elements:

            sysml-members-list::[]
            */
            attribute :>> mass = 9.5 [kg];
            attribute :>> power = 750 [W];
            port 'de-icing fluid';
        }
        part engine : ap {
            doc /*
            Engine provides thrust and compressed air for life-support
            system and other systems
            */
            in port fuel;
            out port air;
            attribute :>> mass = 400 [kg];
        }
        part fuelTank : ap {
            out fuel;
            attribute :>> mass = 100 [kg];
        }
        interface fuelLine connect (fuelTank.fuel, engine.fuel);
    }
}
```

Run it and open the result:

```sh
sysml-publish -api localhost:50051    # reads ./model, writes ./site
xdg-open site/index.html              # or open it in any browser
```

(Or run `sysml-publish -api local` and skip starting the service yourself:
the tool starts one for the run and stops it after.)

You get six pages, one per documented element plus the package that holds
them:

- `site/Model/index.html`: the package. It has no documentation of its own,
  but it contains documented elements, so it gets a page with just its name.
- `site/Model/AircraftPart/index.html`, titled "AircraftPart (ap)", with a
  Properties table: weight 12 kg, power 1.5 kW, the definition's defaults.
- `site/Model/aircraft/wing/index.html`, titled "wing (a)". `wing` is typed by
  `AircraftPart` (through its short name `ap`), so it shows that
  representation, but the macros speak about `wing`: its own name and short
  name, its own doc (which itself lists wing's members: mass, power,
  de-icing fluid), and its own values, 9.5 kg and 750 W.
- `site/Model/aircraft/engine/index.html` and `.../fuelTank/index.html`, the
  same way: engine's doc and 400 kg; fuelTank's 100 kg.
- `site/Model/aircraft/index.html`: aircraft's own representation first
  (its purpose, then its parts wing, engine and fuelTank, then its interface
  fuelLine, each list picked by element kind), then the inherited
  AircraftPart one. Its own representation has no `= Title` line, so the
  page takes its title from the inherited one: "aircraft ()", empty
  parentheses because `aircraft` has no short name. The purpose shows twice,
  because both representations include `sysml-doc::[]`.

The full model adds a metadata table (`sysml-metadata-table`) to every part
page; see the example below.

### Example output

[`docs/example/`](docs/example/) holds the site generated from the full
[`model/aircraft.sysml`](model/aircraft.sysml), committed so you can look
before building anything:

- [landing page](docs/example/index.html)
- [AircraftPart](docs/example/Model/AircraftPart/index.html): the definition
  whose representation every part inherits
- [aircraft](docs/example/Model/aircraft/index.html): parts and interfaces
  listed by element kind
- [wing](docs/example/Model/aircraft/wing/index.html): the inherited
  representation filled in for the wing, with its metadata table
- [engine](docs/example/Model/aircraft/engine/index.html),
  [fuelTank](docs/example/Model/aircraft/fuelTank/index.html),
  [airConditioner](docs/example/Model/aircraft/airConditioner/index.html)

Open them from a clone in a browser (`xdg-open docs/example/index.html`): a
repository browser such as GitHub shows HTML files as source, not as pages.
After changing the model or the tool, regenerate them with:

```sh
go run ./cmd/sysml-publish -model model -out docs/example   # in a clone
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
  (`part wing : AircraftPart`), a definition those of its supertypes
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

```
site/
├── index.html                  landing page: the whole tree
├── Model/
│   ├── index.html              page of Model
│   ├── AircraftPart/index.html page of Model::AircraftPart
│   └── aircraft/
│       ├── index.html          page of Model::aircraft
│       ├── wing/index.html     page of Model::aircraft::wing
│       └── ...
├── assets/                     asciidoctor.css, layout.css
└── .sysml-publish              marker: this folder was written by the tool
```

- The page of `A::B::C` is `site/A/B/C/index.html`. A page keeps its path when
  unrelated parts of the model change (except the numbered `unnamed-N`
  folders).
- Every page has the whole tree as a sidebar on the left, with the current
  page marked.
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
| `sysml-qualified-name:[<element>]` | inline | its qualified name, `Model::aircraft::wing` |
| `sysml-short-name:[<element>]` | inline | its short name, the `<W>` in `part def <W> Wheel` |
| `sysml-rep:[<element>][<language>]` | inline | the body of the element's representation in another language, as plain text |
| `sysml-value:<attribute>[]` | inline | the attribute's value with its unit, `9 kg`; also another element's attribute (`Model::aircraft::engine::mass`) by a qualified or relative name, or a feature chain (`engine.mass`); a metadata attribute: `@m::owner` |
| `sysml-doc::[<element>][]` | block | the element's `doc` comments |
| `sysml-members-list::[]` | block | a list of its members with their types |
| `sysml-members-list::<element>[]` | block | its members typed by, subclassing or subsetting `<element>` |
| `sysml-members-list::<kind>[]` | block | its members of one kind: `partUsage`, `interfaceUsage`, ... |
| `sysml-iface-ends::[<interface>][depth=<n>\|all][name=short\|long]` | block | the ends of an interface or connection, source first: `engine.air`; `depth` shows the last n segments of the port's qualified path (`engine::air`), `all` all of it; `name=long` names each segment by its name (short names are the default) |
| `sysml-ifaces::[<element>][name=short\|long]` | block | the interfaces and connections the element's ports are on: `i1 -> y`, or `i3 -> y, z` for the other parts |
| `sysml-metadata-table::<metadata def>[]` | block | a table of that metadata's attributes on the element |
| `sysml-view::[<view>][]` | block | the view as OpenSysML renders it: a diagram as an SVG picture, a table view as a table; a diagram needs Graphviz's `dot` |

Inline macros go anywhere in text; block macros go on a line of their own.
Without a target, `sysml-name`, `sysml-qualified-name`, `sysml-short-name`,
`sysml-doc` and `sysml-rep` describe the page's element; give them an element
(`sysml-name:engine[]`, `sysml-doc::Model::AircraftPart[]`) to describe that
one instead.
Where a macro takes an element, name it as SysML would: by qualified name,
by a name relative to the page's element, or by a name only one element has;
short names work in every part of a name (`Model::ap` for `Model::AircraftPart`).

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
title = "Aircraft docs"      # the site's title: landing page, sidebar, browser tabs
baseURL = "https://example.com/aircraft/"  # the URL the site is published under
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

Everything the tool reports goes to stderr as log records (here for
`model/aircraft.sysml`, with a warning as it would appear for a composite
value):

```
level=INFO msg="model loaded" folder=./model files=1 elements=294 warnings=0 duration=251ms
level=INFO msg="page written" element=Model::aircraft::wing empty=false path=site/Model/aircraft/wing/index.html
level=WARN msg="Model::aircraft::wing: sysml-value:size[]: size is composite; shown as [composite]"
level=INFO msg="site published" pages=7 landing=site/index.html duration=943ms
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
- a model with no documented elements at all: only the landing page is
  written, and the run still succeeds.

## Limitations

This is a first version.

- Models are read from files only; reading from a SysML v2 API server is
  planned.
- Views (`sysml-view`) are rendered by OpenSysML in process; a diagram is
  laid out by Graphviz, which must be installed where the site is built.
- A failing macro stops the whole run; whether it should instead leave a
  placeholder and warn is undecided.
- The tool reads models through OpenSysML's SysML v2 API queries and element
  reads, which are not in an OpenSysML release yet: it is built against a
  development commit of OpenSysML.

## Further reading

- [docs/macros.adoc](docs/macros.adoc): the macro reference.
- [requirements.org](requirements.org): the requirements, decisions and open
  questions, with the IDs the code and tests refer to.
- [AGENTS.md](AGENTS.md): how the program is built, for anyone changing it.
