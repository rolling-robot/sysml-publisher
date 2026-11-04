# sysml-publish

`sysml-publish` turns a SysML v2 textual model into a static HTML website. You
write the documentation inside the model, as AsciiDoc textual representations
(`rep language "asciidoc"`), and the tool gives every documented element a page,
arranged as the model's containment tree, with that tree as a navigation
sidebar on every page. Small `sysml-*` macros pull live data from the model into
the text: names, `doc` comments, member lists, metadata tables and evaluated
attribute values with their units.

The model is parsed in-process by [OpenSysML](https://github.com/Open-MBEE/OpenSysML)
and the AsciiDoc is rendered by [asciidoctorg](https://github.com/rolling-robot/asciidoctorg).
No server and no Ruby installation are needed; the output is plain HTML and CSS
that opens straight from disk.

## Quick start

Build the tool (see [Requirements and building](#requirements-and-building)):

```sh
go build -o bin/sysml-publish ./cmd/sysml-publish
```

Put a model in `./model`, for example `model/vehicle.sysml`:

```sysml
package Vehicles {
    private import SI::*;

    part def Wheel {
        doc /* A road wheel with its tyre. */
        attribute mass = 9 [kg];
        rep language "asciidoc"
            /*
            = Wheel: sysml-name:[]

            sysml-doc::[]

            Mass:: sysml-value:mass[]
            */
    }
    part frontLeft : Wheel;
}
```

Run it and open the result:

```sh
bin/sysml-publish            # reads ./model, writes ./site
xdg-open site/index.html     # or open it in any browser
```

You get three pages:

- `site/Vehicles/index.html`: the package. It has no documentation of its own,
  but it holds documented elements, so it gets a page with just its name.
- `site/Vehicles/Wheel/index.html`, titled "Wheel: Wheel", with the doc comment
  and "Mass: 9 kg".
- `site/Vehicles/frontLeft/index.html`, titled "Wheel: frontLeft". `frontLeft`
  is typed by `Wheel`, so it shows Wheel's representation, but the macros
  speak about `frontLeft`: its own name, its own (absent) doc, its mass.

## Requirements and building

- Go 1.27.2 or newer.
- Local checkouts of the two libraries. `go.mod` points at them with `replace`
  directives:

  ```
  replace github.com/Open-MBEE/OpenSysML => /home/pete/projects/sysmlv2/OpenSysML
  replace github.com/rolling-robot/asciidoctorg => /home/pete/projects/asciidoctor-go/asciidoctor-go
  ```

  Clone both and change these paths to where they are on your machine.
  asciidoctorg cannot be fetched as an ordinary module version yet.

Then:

```sh
go build -o bin/sysml-publish ./cmd/sysml-publish
go test ./...        # optional: runs the test suite on the fixture models
```

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
  (`part frontLeft : Wheel`), a definition those of its supertypes
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
├── Vehicles/
│   ├── index.html              page of Vehicles
│   ├── Wheel/index.html        page of Vehicles::Wheel
│   └── frontLeft/index.html    page of Vehicles::frontLeft
├── assets/                     asciidoctor.css, layout.css
└── .sysml-publish              marker: this folder was written by the tool
```

- The page of `A::B::C` is `site/A/B/C/index.html`. A page keeps its path when
  unrelated parts of the model change (except the numbered `unnamed-N`
  folders).
- Every page has the whole tree as a sidebar on the right, with the current
  page marked.
- All links are relative, so the site works from `file://`, from any web
  server, and from any sub-path.
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
| `sysml-name:[]` | inline | the element's name |
| `sysml-qualified-name:[]` | inline | its qualified name, `Vehicles::Wheel` |
| `sysml-short-name:[]` | inline | its short name, the `<W>` in `part def <W> Wheel` |
| `sysml-value:<attribute>[]` | inline | the attribute's value with its unit, `9 kg` |
| `sysml-doc::[]` | block | the element's `doc` comments |
| `sysml-members-list::[]` | block | a list of its members with their types |
| `sysml-members-list::<element>[]` | block | its members that subclass or subset `<element>` |
| `sysml-metadata-table::<metadata def>[]` | block | a table of that metadata's attributes on the element |

Inline macros go anywhere in text; block macros go on a line of their own.

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

## Command line

```
sysml-publish [-model DIR] [-out DIR] [-v] [-log-format text|json]
```

| Flag | Default | Meaning |
|---|---|---|
| `-model DIR` | `./model` | folder with the model; every `*.sysml` and `*.kerml` file in it, searched recursively, is read as one model, so references across files work |
| `-out DIR` | `./site` | folder to write the site to; replaced on every run |
| `-v` | off | also report how long each stage and each page took |
| `-log-format` | `text` | `text` for people, `json` for one JSON object per line, for CI and log tools |

Everything the tool reports goes to stderr as log records:

```
level=INFO msg="model loaded" folder=./model files=1 elements=16 warnings=0 duration=157ms
level=INFO msg="page written" element=Vehicles::Wheel empty=false path=site/Vehicles/Wheel/index.html
level=WARN msg="Vehicles::Pump: sysml-value:position[]: position is composite; shown as [composite]"
level=INFO msg="site published" pages=3 landing=site/index.html duration=365ms
```

Exit status:

| Status | Meaning |
|---|---|
| 0 | the site was written (possibly with warnings) |
| 1 | the run failed; nothing was written |
| 2 | wrong command-line usage |
| 130 | interrupted with Ctrl-C or SIGTERM; nothing was written |

## Errors and warnings

**Stop the run, write nothing, exit 1:**

- errors in the model, each reported with its location
  (`msg="model error" diagnostic="model/bad.sysml:2:14: ..."`);
- a model folder with no model files;
- a macro that cannot do what it was asked: an attribute or metadata def the
  model does not have, an element name that is unknown or names several
  elements;
- an output folder that is not empty and was not written by the tool.

**Warn and carry on:**

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
- There is no macro yet for diagrams of SysML views (`sysml-view` is planned).
- A failing macro stops the whole run; whether it should instead leave a
  placeholder and warn is undecided.
- The tool relies on OpenSysML's SysML v2 API JSON export, which OpenSysML
  marks as experimental (the tool says so on every run).
- In member lists, a member that gets its type only through what it subsets or
  redefines (`part front :> wheels;`) is shown without a type.

## Further reading

- [docs/macros.adoc](docs/macros.adoc): the macro reference.
- [requirements.org](requirements.org): the requirements, decisions and open
  questions, with the IDs the code and tests refer to.
- [AGENTS.md](AGENTS.md): how the program is built, for anyone changing it.
