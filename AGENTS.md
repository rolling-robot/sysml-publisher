# AGENTS.md — sysml-publisher

Guide for anyone (human or agent) changing this code: what the program does,
how it is built, the rules the code relies on, and an assessment of the
architecture as of November 2026.

The requirements live in `requirements.org` (IDs such as PG-7, MC-10, TBD-9
below refer to it). The user-facing macro reference is `docs/macros.adoc`.
Keep both in step with the code: every behaviour change updates the
requirement it implements, and every macro change updates the reference.

---

## 1. What the program does

`sysml-publish` turns a folder of SysML v2 textual models into a static HTML
site: one page per element that has an AsciiDoc textual representation
(`rep language "asciidoc" /* ... */`), arranged as the model's containment
tree, with the tree as a sidebar on every page. **It does not parse the model
and does not hold one in memory** (DS-1): it reads the files' text and sends
it to a **sysml-grpc** service (OpenSysML), which parses the model, holds it,
and answers the tool's reads over the SysML v2 API surface it serves — Query,
ElementProperties, Evaluate, RenderView. Representation bodies are
AsciiDoc, rendered by asciidoctorg, and may use `sysml-*` macros that pull data
from the model (names, docs, members, metadata tables, evaluated attribute
values).

```
sysml-publish [-api ADDR] [-config FILE] [-model DIR] [-out DIR] [-v] [-log-format text|json] [-dot CMD]
# -api is required: the address of a sysml-grpc service (CLI-9); defaults:
# ./model, ./site, dot from PATH; config file sysml-publish.toml (CFG-1..CFG-6)
```

## 2. Building, running, testing

```sh
go build -o bin/sysml-publish ./cmd/sysml-publish
bin/sysml-publish -api localhost:50051   # ./model -> ./site, parsed by the service
bin/sysml-publish -api local             # or: the tool starts and stops its own
go test ./...                            # spawns a sysml-grpc, runs the fixtures through it
go vet ./...
```

- **A sysml-grpc service is needed to run and to test.** `-api local`
  starts one of the run's own — from `PATH` or built from the OpenSysML
  checkout (`../sysmlv2/OpenSysML`) — and stops it after; `internal/service`
  is that logic, shared with the tests. Or start one yourself
  (`go run ./cmd/sysml-grpc` serves gRPC, gRPC-Web and Connect on
  `localhost:50051`, or `sysml-grpc -port 0 -report-address -health-port 0`
  to let the kernel choose a port and print it) and name it with `-api`.
  The tests get theirs from `internal/servicetest` and skip when it cannot
  be obtained (§8). A service that cannot be reached, started, or lacks a
  capability the tool needs, is a runtime failure (ER-1), never a usage
  error.
- **Dependencies are versions anyone can download** in `go.mod`: OpenSysML
  at a `develop` commit (`v0.12.2-0.20261111062923-56febd38bcad`, for
  `Client.Dial` to sysml-grpc, `ParseSources` documents as inline content,
  the query properties `body`, `language` and `textualRepresentation`,
  `Client.ElementProperties`, and `Client.RenderView`'s written forms
  (`WithForm`: DOT, CSV, …), which no tag has yet; move to the next tag when
  it is released) and asciidoctorg
  `v0.2.1`; `go install …/cmd/sysml-publish@latest` must work, so `go.mod`
  never carries `replace` directives. OpenSysML pseudo-versions work: it
  has no submodules.
- **asciidoctorg is installable only from a release tag.** Its `master` keeps
  the Asciidoctor and CodeRay Ruby sources as git submodules, which Go module
  downloads leave out, so `//go:embed` in `rubysrc/rubysrc.go` finds no files
  and the build fails; its tags (`v0.1.0`, `v0.2.0`, `v0.2.1`) commit the sources as plain
  folders. Never require an asciidoctorg pseudo-version (`@master`, a commit).
  A newer asciidoctorg API needs a new asciidoctorg tag first.
- **Local development against checkouts** uses an untracked `go.work`
  (ignored by `.gitignore`) with `use .` and two `replace` lines pointing at
  the checkouts:

  ```
  go 1.27.2

  use .

  replace github.com/Open-MBEE/OpenSysML => ../sysmlv2/OpenSysML
  replace github.com/rolling-robot/asciidoctorg => ../asciidoctorg
  ```

  Use `replace`, not
  `use`, for them: `use` makes them main modules, which needs every
  dependency's `go.mod` in the cache, including versions MVS does not pick.
  Check the published versions with `GOWORK=off go build ./...` before a
  release.
- **Offline builds** work once the module cache is filled (`go mod download`
  with network, once). In a sandbox without DNS, run Go with `GOPROXY=off`;
  the Go tool resolves names itself, so a sandbox that lets `git` and `curl`
  through can still block it. Building is offline, but running and testing
  additionally need a sysml-grpc to dial.
- **Toolchain**: Go 1.27.2+ (asciidoctorg's minimum).
- `bin/`, `site/`, the root `sysml-publish` binary, `*.api.json` dumps and
  `*~` backups are build output or scratch, ignored by `.gitignore`.

## 3. Layout

```
cmd/sysml-publish/main.go   CLI: flags, wiring, messages, exit codes
cmd/sysml-publish/config.go the config file: TOML, strict keys, flag precedence (CFG-1..CFG-6)
CHANGELOG.md                changes per release
internal/model/             reads the files' text, dials the Service, splits diagnostics; embeds *sysml.Model
internal/service/           starts a local sysml-grpc (the CLI's -api local; the tests via servicetest)
internal/servicetest/       the tests' address of a SysML service; skips when it cannot get one
internal/tree/              which elements get pages, inheritance, names, paths (reads the model, writes nothing)
internal/render/            convert representations per page (asciidoctorg), titles;
                            source.go (where a macro is written, for warnings)
internal/site/              lay a render result out as HTML files (html/template), write
internal/site/assets/       asciidoctor.css (Asciidoctor default, MIT), layout.css,
                            sort.js (table sorting in the browser, SB-4),
                            sidebar.js (sidebar resizing, SB-5; collapsible
                            branches remembered, SB-6),
                            mermaid.min.js (Mermaid 12, MIT; MC-25; written with the site only when a
                            page shows a figure)
internal/site/page.html.tmpl
sysml/                      public: reads the model the Service parsed (Query, Props, element facts, Evaluate,
                            RenderView)
macros/                     public: the sysml-* AsciiDoc macros, one file per macro;
                            names.go (element lookup), kinds.go (metaclass names);
                            example_test.go (use from another module)
views/                      public: renders views for sysml-view through the Service, with Graphviz's dot
model/                      the user's model (default input)
testdata/<case>/            fixture models, one folder per test case
docs/macros.adoc            user reference for the macros
docs/example/               generated site of the multi-file UAV model in model/, linked from README.md
requirements.org            requirements, decisions, open questions
```

## 4. Pipeline

```
 model/*.sysml, *.kerml ── read as text ──▶ sysml-grpc  (the Service, DS-1)
        │  model.Folder.Load: Dial + ParseSources (documents named by path;
        │                                     .kerml documents carry LanguageKerML)
        ▼                            the model is parsed and held by the service
        │  capabilities checked once: parse_sources, query, element_properties,
        │  render_view, render_view_form, evaluate_subject (ER-1 when missing)
        │  diagnostics: errors stop the run (ER-2); Query(all elements)
        │  -> ElementCount, Tops, own ids              (the client kept open)
        ▼
 model.Model { Files, Warnings, Tops; Query, Props, Evaluate }      (DS-4)
        │  tree.Build(ctx, m)
        │    Query TextualRepresentation -> AsciiDoc ones, their owners documented
        │    Query Usage/Definition -> generals walk -> inherited representations
        │    owner chains up to the tops -> ancestors; ownedMember order
        ▼
 tree.Tree { Root, Pages (pre-order), ByID, Model, Warnings }
        │  render.Tree(ctx, t, Options{Logger, Pages, Views})
        ▼
 asciidoctorg instance + macros.Register(ad, model, Pages (default: the tree), warn)
   pass 1: convert every page's representations (per page — MC-3)
   pass 2: assign page titles per sibling group (PG-8, PG-9)
        ▼
 render.Result { Tree, Pages map[node]*Page{Title, Own, Inherited}, Warnings }
        │  site.New              (site.Render = render.Tree + site.New)
        ▼
   execute page.html.tmpl for every page + landing page, add assets + marker
        ▼
 site.Site { Files map[path][]byte, Warnings }
        │  Site.Write            replaces the out folder only if it carries .sysml-publish
        ▼
 site/…/index.html, site/assets/*.css, site/.sysml-publish
```

Everything is in memory until `Site.Write`; a failing run writes nothing.

## 5. Packages

### 5.1 `internal/model` and `sysml`

`internal/model` reads the files' text, hands the model to the Service, and
splits the diagnostics; the public package `sysml` reads the parsed model
(`sysml.Open(ctx, client, parsed)`), for this tool and for other modules.
`model.Model` embeds `*sysml.Model`, so its read methods are the model's.

- `Folder{Dir, API}.Load` walks the folder for `*.sysml`/`*.kerml` (sorted, so
  runs are deterministic; hidden files and folders skipped, DS-2), reads
  them, and sends them to the service at `API` as the documents of one
  `ParseSources`, each named by its path (`.kerml` documents carry
  `LanguageKerML`), so cross-file references resolve (DS-2). It checks the
  service's capabilities once before parsing (`parse_sources`, `query`,
  `element_properties`, `render_view`, `render_view_form`,
  `evaluate_subject` — a service without the last two would silently change
  `sysml-value`'s and `sysml-view`'s answers), fails ER-1 with the address
  when it cannot reach the service or a capability is missing, splits
  diagnostics into errors (returned as `*DiagnosticsError`) and warnings,
  and queries every element once (`readIndex`): `ElementCount`, `Tops`
  (top-level elements, file order) and the model's own ids (`Contains`).
  The tool holds no parsed model: the service does (DS-1); what the tool
  keeps is the hash, the diagnostics, and, later, each rendering's working
  set (representations, read properties, the macros' indexes).
- **Reading (DS-4)**: no whole-model export. `Query` selects elements
  (OpenSysML's SysML v2 API query; one value per property, so never read a
  multi-valued property through it — §13). `Props(ctx, id, names...)` reads
  one element's metamodel properties through `ElementProperties` (the API's
  get-element), by form id (`M__pd`), qualified name or positional id
  (`M::@2`); values are cached per element and property, the entry shared
  by every name it was asked by. `Ref{ID, QualifiedName, Metaclass}` is an
  element reference; `Ref.Name()` is the unquoted last segment.
- **Element facts** (`facts.go`), for any element, page or not: `Name`
  (derived: redefinitions borrow names), `ShortName`, `QualifiedName`
  (positional name `M::@2` for an unnamed element, counted in the owner's
  `ownedMember`), `Owner`, `Members`, `Types`, `Docs`,
  `Representation(s)`, `Generals` (all owned specializations),
  `Specializes` (transitive typing, subclassification or subsetting over
  `Generals`; `sysml-members-list`), `Redefined`. `Generals` of a library element are none and unread (§13,
  13).
- `Stats()` counts queries, element reads and cache hits; the CLI logs them
  with the page tree record.
- The SysML client stays open after a successful load for `Props`,
  `Query`, `Model.Evaluate(ctx, subjectQualifiedName, expression)` and
  `Model.RenderView`; the caller must `Close()` it (a dialed service keeps
  running: it was not the client's to stop). Not safe for concurrent use
  (the cache).
- `Source` is the seam between the tool and a service kind (DS-3): `Folder`
  speaks sysml-grpc through `opensysml.Dial` today; a Flexo/REST Source is
  to be added later without touching the rest of the tool. `Evaluate` is
  not part of the interface.

### 5.2 `internal/tree`

`Build(ctx, m)` reads the model and writes nothing; this is where the
requirements about *which* pages exist live. It starts from a query (DS-4)
and reads only the elements that get pages and their surroundings.

- **Representations**: one Query for `TextualRepresentation`; those whose
  `language` is `adoc`/`asciidoc` (case-insensitive) are grouped by owner,
  and their owners are documented. Others → warning (PG-4).
- **Inheritance** (PG-7): one Query for `Usage`/`Definition`; for each, a
  BFS over `Model.Generals` (every owned specialization: typing,
  subclassification, subsetting, redefinition, reference subsetting)
  collects the representations of reached elements, nearest first, each
  once, cycle-safe (OpenSysML rejects specialization cycles anyway). No
  query asks "what specializes X" across subsetting (§13, 2), hence the
  walk.
- **Pages**: documented elements and their owners up to the tops are kept
  (PG-1, PG-6); the tree is laid out from `Model.Tops` over `ownedMember`,
  in declaration order; anything not kept is pruned (PT-4).
- **Names**: `Model.Name` (derived `name`: redefinitions borrow names),
  else `Unnamed` with the `Unnamed` flag set.
- **Paths**: `Dir` is the slash path of folder segments (PT-1). Segments are
  made filesystem-safe; unnamed siblings become `unnamed-N` (PT-6); clashes
  (case-insensitive) get `-2`, `-3`.
- Nodes carry what pages need (name, qualified name, folder,
  representations), nothing for macros: macros read the model themselves.
- `Href(from, to)` is the one relative-link function (shortest path,
  segments URL-escaped, `+` as `%2B`). `Tree.Href(fromID, toID)` is the
  same by element id, `ok` false when either has no page: it makes the tree
  the macros' `macros.Pages`.

Three names, kept apart: `Node.Name` is the element's **name** (`Unnamed`,
the constant `UnnamedName`, when it has none); `Node.QualifiedName` is the
**qualified name**, or the positional name `M::@2` for an unnamed element, and
is what messages and `Evaluate` use; the **page title** is decided later in
`site` (PG-8) and is not stored on the node.

### 5.3 `internal/render` and `internal/site`

Rendering is split from output so another output (Confluence, say) can
take the same converted pages: `internal/render` converts and titles,
`internal/site` writes HTML files.

`render.Tree(ctx, t, Options{Logger, Pages, Views})`:

- creates **one** asciidoctorg instance per call and registers the
  macros on it, with `Options.Pages` deciding their links between pages
  (MC-18; nil: the tree's relative links, `tree.Tree.Href`). An output
  whose pages are not the site's folders passes its own. `Options.Views`
  renders `sysml-view`'s views (`Registry.SetViews`; nil: every
  `sysml-view` fails).
- **Pass 1** (`renderer.body`): each page converts each of its
  representations with `ConvertDocumentContext`, passing the page's element id as
  the document attribute `sysml-element` (`macros.Attributes`). The first
  representation (page order: own, then inherited) with an AsciiDoc header
  (`= Title`) supplies the page title; later ones are converted with
  `showtitle` so their titles stay visible (PG-10). Only a header title
  counts — Asciidoctor's `Doctitle` would fall back to the first section.
- **Pass 2** (`assignTitles`): per sibling group — document title, else
  `untitled-N`; empty ancestor pages use the element name; duplicates get
  `-N` (PG-8, PG-9). A `Title` has an HTML form (heading, keeps inline
  markup) and a text form (tab and sidebar).
- returns `Result{Tree, Pages, Warnings}`: a `Page{Title, Own,
  Inherited}` (HTML fragments) for every page of `Tree.Pages`, none for
  the root; `Warnings` as `macros.Warning`s, location filled.

`site.New(result, Options{Title, BaseURL})` lays the result out: `page.html.tmpl`
for every page and the landing page, with the sidebar built from the tree, the
stylesheets and the marker. `Options.Title` is the site's title (CFG-4;
default `LandingTitle`), `Options.BaseURL` gives every page a canonical link
(CFG-5; the pages' links stay relative). Asciidoctor output is trusted as
`template.HTML` (secure safe mode); everything else is escaped by
`html/template`. Its links are the tree's, so the result must be rendered
with `Pages` nil. `site.Render(ctx, t, logger, views, o)` is `render.Tree`
then `New`, and is what the CLI calls.

- `Site.Write` refuses to replace a non-empty folder without the
  `.sysml-publish` marker, then `RemoveAll`s and writes (CLI-4). Not atomic:
  a crash mid-write leaves a partial site (still marked, so the next run
  replaces it).
- Macro warnings arrive as `macros.Warning{Page, Macro, Message}` and are
  kept once per (page, macro) in `Result.Warnings` (MC-7); `Site.Warnings`
  are their lines, `<file>:<line>: <page>: <macro>: <message>`. Deduplication is by those
  fields, never by the text, so rewording a message cannot change it
  (`TestWarningsOncePerPageAndMacro`).
- **Asciidoctor's messages** (`asciidoclog.go`, CLI-6): without a log
  handler, Asciidoctor writes raw lines to stderr (`asciidoctor: WARNING:
  <stdin>: line 3: ...`), outside slog and the JSON format. `render.Tree`
  registers `asciidocLog`, asciidoctorg's `SlogHandler` adapted: the record
  goes to `Options.Logger` at the message's severity, as
  `msg="asciidoc: <message>"` with `element` (the page's element) and
  `location` (`sources.lineOf`: the body line Asciidoctor reports, turned
  into the model's file:line; Asciidoctor counts the body's leading blank
  lines, which OpenSysML keeps). An inherited representation's message is
  logged on each page that shows it. They are not `Result.Warnings`.
- **Source locations** (`source.go`, MC-19): the model gives no source
  positions (§13, 15), so `render.Tree` fills `Warning.Location` itself. Macros
  warn while a representation is converted (`renderer.current`); `sources`
  finds that representation's body in the model files by its text (lines
  compared without surrounding whitespace, since OpenSysML strips the
  indentation, and only between `/*` and `*/`, so a `//` note quoting a
  macro is not taken for it), then the first body line holding the macro's
  name and target. A macro not in the body (one in a doc `sysml-doc`
  inserted) gets the body's first line; a body not found, no location.
  Identical bodies in two places are located at the first.

### 5.4 `macros`

**Usable from other modules.** The package imports no `internal/`
package (`TestNoInternalImports` keeps it so) and its API names no
internal type:

- `Register(ctx, ad, m *sysml.Model, pages Pages, warn Warn)
  (*Registry, error)` adds every macro in one `ExtensionGroup` named
  `sysml` (MC-4).
- **Custom macros (MC-20, `custom.go`)**: `Registry.Inline(name, fn,
  positional...)` and `Registry.Block(...)` add the caller's own macros,
  each in its own group (asciidoctorg replaces a group registered again),
  through the same `inline`/`block` path as the built-ins, so failures
  become placeholders and warnings. `fn` gets a `*Call`: `Page`, `Target`,
  `Macro`, `Attr`, `Element()` (`nameTarget`), `Resolve` (`nameIndex`),
  `Name`, `Evaluate`, `Link`, `Warn`, `Model`, `Context`, `Parent`;
  `FormatValue` and `Passthrough` are the built-ins' formatting. Names must
  be macro names and not start with `sysml-` (`checkName`). This is public
  API: changing `Call` or these functions breaks other modules' macros.
- `Views` is the host's say on views (MC-21): `Render(ctx, qualifiedName)
  (*Rendering, error)`, a `Rendering` being an SVG or a table's rows;
  `Registry.SetViews` sets it after `Register`, so `Register`'s signature
  did not change. `views.InProcess` (package `views`) is this tool's: the
  model's `RenderView` gives a diagram as DOT, laid out by Graphviz's `dot`,
  or a table as its rows, which needs no program. A view's
  rendering does not depend on the page, so caching it per view does not
  break MC-3.
- `Pages` is the host's say on links: `Href(fromID, toID) (href, ok)`.
  `*tree.Tree` implements it for this tool; nil links nothing.
- `Attributes(elementID)` are the document attributes to convert a
  representation with for an element.
- `Warning{Page, Macro, Message, Location}`: `Page` is the page element's
  qualified (or positional) name.

`Example` and `ExampleRegistry_Inline` (`example_test.go`) and
`custom_test.go`, all in package `macros_test`, are the outside caller's
view: it parses a model, `sysml.Open`s it, registers with a map as
`Pages` and converts, importing only public packages. Shared pieces in
`macros.go`:

- `pageElement(parent, e)`: reads `sysml-element` from
  `parent.Document()` and the element's qualified name from the model, as
  an `element{id, qualifiedName}`. This is how every macro knows the page's
  element (MC-2, MC-5); it need not have a page. A missing attribute or an
  id the model lacks is a renderer bug and returns an error.
- `env` (`macros.go`): what the macros of one rendering share — the
  rendering's context, the model, the host's `Pages`, `nameIndex`,
  `metadataIndex` and the lazily built `kindIndex` and `connIndex` (the
  whole-model reverse index of ports to connections, `sysml-ifaces`,
  MC-23). `Register` makes one
  and passes it to every `register<Name>(g, e)`. Macros read the model only
  through `sysml.Model` (`Props`, `Query`, the element facts, `Specializes`,
  `Evaluate`), never a raw element map, and never through the page tree.
- **Links (MC-18)**: `e.href(from, to)` asks `Pages`; `e.linkTo(parent,
  from, to, text)` links text unless the element has no page, is the page
  itself, or the text is empty. Every link goes through `link(parent, to,
  href, text)`: an anchor node of type `link` (`CreateAnchor`) carrying
  `AttrLink` (`sysml-link`, the target element's id), converted at once
  by the document's converter (`PhraseNode.Convert`). html5 writes it as
  `<a href>`, exactly as before; a converter for another output
  recognises `AttrLink` on `inline_anchor` and writes its own link
  (Confluence's `<ac:link>`). Converting at once, rather than returning the
  node, lets the same path serve list items and table cells, whose text is
  AsciiDoc embedding finished markup (`passthrough`). `sysml-name` & co.,
  `sysml-members-list` and `Call.Link` use `linkTo`; `sysml-iface-ends` and
  `sysml-ifaces`
  call `e.href` and `link` themselves, so they also link a segment (or the
  connection and partner parts) that is the
  page's own element. Never write `<a href>` in a macro's converted output
  (`TestLinksAreWrittenByTheConverter`, `TestIfaceEndsLinks`); the exception
  is a figure's Mermaid source, whose labels carry the link to the element's
  page as `<a href>` for mermaid's browser rendering to make clickable
  (`mermaidLabel`; mermaid takes no click on a subgraph, and the figure's
  frontmatter states `securityLevel: loose` so mermaid keeps the label's
  HTML).
  `sysml-view` follows the same rule for pictures (MC-21): an image block
  carrying `AttrView`, never SVG or `<img>` markup in a pass block
  (`TestViewsAreWrittenByTheConverter`).
- **Failing macros (MC-19)**: every macro is registered through
  `inline(g, e, name, opts, written, body)` or `block(...)`. They read the
  page's element, unescape inline targets, write the macro as it appears in
  warnings (`written`; default `name:target[]` / `name::target[]`), and
  call `body(call)`. An error from the body is the macro failing: `e.fail`
  reports `Warning{Page, Macro, Message: err.Error()}` and a placeholder
  takes the macro's place (`<span class="sysml-failed">` inline, a pass
  block `<div class="sysml-failed">` for a block macro), showing the macro
  and the message, escaped. So a body returns plain errors that say what is
  wrong, never `"<macro> on the page of …"`. Only a cancelled context (and
  `pageElement`'s renderer bugs) still return an error from the processor,
  which stops the conversion. Block macros run while Asciidoctor parses,
  inline ones afterwards, so on one page block warnings come first.
- `registerText(g, e, name, ofPage, ofElement)`: an inline macro
  `name:[<element>]` returning escaped text: `ofPage(node)` without a
  target, `ofElement(id)` for the element `nameTarget` resolves.
  `sysml-name`, `sysml-qualified-name` and `sysml-short-name` are a line
  each on top of it; the element facts of `model.Model` answer for
  any element, page or not. The text links to the element's page when it
  has one and is not the page itself (`linkTo`, MC-18).
- **Inline targets** (`macros.go`, `value.go`): Asciidoctor's inline forms
  take a target always (long) or never (short), and the target stops at a
  space. `optionalTargetRx(name)` (used by `registerText` and `sysml-rep`)
  and `valueMacroRx` set `MacroOptions.Regexp`: an optional or required
  target that may contain quoted segments with spaces. Targets arrive
  HTML-escaped (`<` as `&lt;`), so they are `html.UnescapeString`ed first.
  Block macro targets (`sysml-doc`) arrive as written.
- `nameIndex.resolve(name, page, accept)` (`names.go`): the one element
  lookup (MC-15), used by `sysml-members-list` and `sysml-metadata-table`.
  It walks names segment by segment over `ownedMember` (`Props`), so every
  segment may be a name or a short name: from the top-level elements, then
  from the page's element and each owner outward, then library qualified
  names (`Props` by qualified name), then a plain name or short name only
  one model element has (`Query`, cached per name). `accept` narrows the candidates (metadata defs
  only); an unknown name wraps `errNoSuchName`, so a caller can try another
  reading (an element kind). Relative lookup sees owned members only, not
  inherited or imported ones.
- `kindIndex` (`kinds.go`): metaclass names copied from OpenSysML's
  internal export tables, plus every `@type` a query over the whole model
  returns; built on first use.
- `metadataIndex` (`metadata_table.go`): metadata defs (by `nameIndex`),
  an element's usages of one (`metadataDefinition`), a def's attributes
  (own, then supertypes'; redefined ones once) and an attribute's value
  expression (bound, else the def's, following redefinitions).
  `valueText` writes an expression back from its parts (literals,
  enumeration literals as `Level::low`, operators, parenthesised inside),
  since the model gives no source text (§13); anything else shows as its
  metaclass.
- `passthrough(html)`: wraps finished HTML in `+++…+++` for places where
  AsciiDoc substitutions would otherwise run over model text (list items,
  table cells).
- `Warning{Page, Macro, Message, Location}` and `Warn`: how a macro reports
  a problem while still producing output, a failure included. The macro
  fills the first three and never formats the page or macro into
  `Message`; `Location` is left to the renderer (§5.3). `Warning.String()`
  builds the reported line, and `render.Tree` deduplicates on (page, macro)
  (MC-7).

| Macro | Form | Reads | Notes |
|---|---|---|---|
| `sysml-name:[<element>]` | inline | `nameIndex`, `metadataIndex` | escaped text, linked to the element's page if it has one and it is not this page (MC-18); no target: page's element; target resolved by MC-15; `[<element>::]@<metadata>[::<attribute>]` gives the metadata def's or attribute's name (`nameTarget`) |
| `sysml-rep:[<element>][<language>]` | inline | `Model.Representations`, `nameTarget` | body of the element's own representation in that language (case-insensitive, several joined), escaped plain text; target as `sysml-name` |
| `sysml-short-name:[<element>]` | inline | `Model.ShortName` | empty when none; linked as `sysml-name`; target as `sysml-name` |
| `sysml-qualified-name:[<element>]` | inline | `Model.QualifiedName` | positional name for unnamed; linked as `sysml-name`; target as `sysml-name` |
| `sysml-doc::[<element>][]` | block | `Model.Docs` | each doc parsed as AsciiDoc (`ParseContent`) into an open block; target as `sysml-name`; macros in the docs still see the page's element |
| `sysml-members-list::[<element or kind>][]` | block | `members` (`Model.Members`, `Model.Types`), `Model.Specializes` | ulist; owned members minus annotations and multiplicities (`notMembers`); links to members that have pages; no target: all members with `(types)`; element: members typed by, subclassing or subsetting it; else kind: members of exactly that metaclass (`kindIndex`, case-insensitive) |
| `sysml-iface-ends::[<element>][]` | block | `Props` (`connectorEnds`: `ownedFeature`, `isEnd`, `ownedReferenceSubsetting`, `chainingFeature`), `nameTarget` | ulist of owned ends (`isEnd`): the connected feature from its `ReferenceSubsetting` (a `chainingFeature` path), segments linked; the end named `source` first; no ends → failure; `depth` (named or positional): last n segments of owners(chain[0]) + chain, ` :: `-joined, single quotes stripped, `all` the whole path; none: chain as written with `.`; `name` (default `short`): each segment by short name when it has one, else by name; `long` always by name; an end that connects nothing shows its name |
| `sysml-ifaces::[<element>][]` | block | `connIndex` (query `ConnectionUsage`/`InterfaceUsage`, then `connectorEnds`), `nameTarget`, `Model.Members` | ulist of the connections the target's owned `PortUsage` members are on: one item per connection, its name linked, ` -> `, then the distinct other ends' parts (owners of the leaf feature), linked, joined `, `; self-loop: connection alone (`i -> `); no ports or none referenced: line dropped; a target that is not a part (`PartUsage`/`PartDefinition`) fails; `name` (default `short`): each by short name when it has one, else by name; `long` always by name |
| `sysml-fig::[<element>][]` | block | `connIndex` (`connectorEnds`), `owners`, `nameTarget`, `Model.Name`, `Pages` (`Href`) | a figure of the target as Mermaid (MC-25), drawn in the browser by the site's mermaid.js (12, ELK layout) in OpenSysML's Mermaid interconnection style. An `InterfaceUsage`: the ends' deepest common ancestor as the figure's frame, drawn whatever it is (a package included; packages further down a chain not drawn, what they own hoisted), the end parts and the parts between nested as the model nests them, each box's title the keyword line above the name with the contents drawn below all the title's lines (ELK reserves the space; the Pilot colors pinned by classDefs), a stadium node per end feature, the edges one per pair of ends for a two-ended interface, one per end into the interface's own node (labelled `«interface»` and its name, drawn where the model owns it: inside its nearest drawn box ancestor, the frame included, else at the top of the figure) for one with more. A part (`PartUsage`/`PartDefinition`): the part's box with its owned ports as stadium nodes, and, per connection its ports are on, each end — an end on one of the part's ports anchored at the port's node, another end's leaf feature inside a box for its immediate owner, the owner beside the part's box or inside it when the model owns it there — the edges one per pair of the connection's ends for a two-ended connection, one per end into the connection's own node (labelled by its metaclass, `«interface»` or `«connection»`, and its name, drawn where the model owns it: inside its nearest drawn box ancestor, the part's box included, else at the top of the figure) for one with more; ports with no connections: ports and no edges. A drawn element with a page, not the page the figure is on, links to it in its label's HTML, styled as a link (mermaid takes no click on a subgraph; the frontmatter states `securityLevel: loose`). The source is written by the macro (`AttrFig`, `AttrMermaid`, `MermaidText`), no view and no Graphviz; no target: the page's element; a resolvable target that is neither interface nor part echoes the macro text, no warning; no element, an interface with fewer than two ends / an end that connects nothing / no common ancestor, a part with no ports: failure |
| `sysml-metadata-table::<def>[]` | block | `metadataIndex` (`Props`) | table per MetadataUsage; bound value, else def value, else "(no value)"; supertypes' attributes after own |
| `sysml-view::[<view>][]` | block | `nameIndex` (`ViewUsage`), `Views` | the host's `Views` renders the view by qualified name (`Registry.SetViews`; none: failure); diagram → image block (`CreateImageBlock`), target the SVG as a `data:image/svg+xml;base64,` URI, alt the view's name; table → table, header row first; both with role and `AttrView` (the qualified name), written by the output's converter as links are (MC-18; `ViewSVG` gives another converter the SVG) and the `.Title` written above set on the block itself, since the parser drops a block macro's attributes; no target: the page's element, must be a view; a name that is no view → "X is a PartUsage, not a view" |
| `sysml-value:<feature>[]` | inline | `Model.Evaluate` | target must be a feature reference (`featureRefRx`: names, `::`, `.`), not an expression, or `[<element>::]@<metadata>::<attribute>` (`parseMetadataRef`, value read as `sysml-metadata-table` does, not evaluated); OpenSysML evaluates it with the page element as subject; units formatted; composite → `[composite]` + warning |

Failure policy (MC-19): a macro that cannot do what the user wrote leaves
a placeholder and a warning, and the run succeeds; there is no flag to turn
warnings into a failing exit code. `TestMacroFailureLeavesPlaceholder`
(site), `TestMacroFailuresAreWarnings` (CLI) and `TestFailurePlaceholder`,
`TestEveryFailureReported`, `TestCancelledIsNoFailure` (macros) pin it.
Errors that still stop the run (cancellation) keep their Go chain through
asciidoctorg's Ruby: `render`'s `convert` wraps an `*asciidoctor.Error` whose
`Cause` is the macro's error; identify them by type or `ctx.Err()`, not by
message.

### 5.5 `cmd/sysml-publish`

`run(ctx, args, stderr) int`, tested in `main_test.go` against fixtures into
`t.TempDir()`.

- **Config file (CFG-1..CFG-6, `config.go`)**: an optional TOML file,
  `sysml-publish.toml` in the working directory or the `-config` path (which
  must exist). Keys are strict: `[source] dir` (the model folder, CLI-2) and
  `[source] api` (the Service's address, CLI-9), and `[site] dir` (the site
  folder, CLI-3), `title` (CFG-4) and `baseURL`
  (CFG-5); anything else is a usage error naming the key, so the decoder
  runs twice — into a map for the exact key check (the decoder also matches
  keys ignoring case) and into the struct for the values. `baseURL` must be
  absolute; set to `""` it is still set, and still an error. A flag the user
  gave (`fs.Visit`) wins over the file, the file over the flag defaults.
  A config problem is a usage error (exit 2), logged once, before anything
  loads.

- **Service address (CLI-9)**: `-api ADDR`, config `[source] api`; the flag
  wins over the file (CFG-3). Neither given → usage error (exit 2) that
  says to start a `sysml-grpc` and name it. The address `local` names no
  service: the run starts one of its own (`internal/service.Start`, from
  `PATH` or built from the checkout), uses it and stops it after; starting
  one that cannot be started is a failure (ER-1) saying why.
  `publish` passes the address to
  `model.Folder.API`. A service that cannot be reached, that answers
  nothing, or that lacks a needed capability fails the run (ER-1) like any
  error, logged once in `run`; nothing is written.

- **Logging (CLI-6)**: every message is a `log/slog` record on stderr: text
  without timestamps by default, JSON with `-log-format json`; `-v` enables
  Debug records with a `duration` for each stage and each page. Model
  errors are logged one record each (`msg="model error" diagnostic=...`).
  Every error is logged exactly once, in `run`; the packages return errors
  and never log them. The exception is what is not an error:
  Asciidoctor's own messages go to the rendering's logger as they happen
  (`asciidocLog`, §5.3).
- **Loggers are passed explicitly** (`model.Folder.Logger`, `site.Render`'s
  `logger`), not taken from `slog.Default`; see the asciidoctorg quirk in §9.
- **Views (MC-21, CLI-8)**: `publish` gives `site.Render` a `views.InProcess`
  with the loaded model and the `-dot` command. A view is rendered in
  process; a missing program is no usage error, and only a diagram's
  `sysml-view` macro fails without `dot` (a table needs no program).
- **Cancellation (CLI-7)**: `main` uses `signal.NotifyContext`; the context
  flows into `Load`, `Render` (checked between pages, and passed to
  `ConvertDocumentContext`, which stops a conversion at asciidoctorg's next
  checkpoint: before each block is parsed or converted, and at each macro),
  every model read and `sysml-value`'s evaluations. An interrupted run writes nothing and exits 130. Detect it
  with `ctx.Err()`: OpenSysML's cancellation error does not wrap
  `context.Canceled`.
- Exit codes: 0 ok, 1 failure, 2 usage error, 130 interrupted.

## 6. Invariants — do not break these

1. **Per-page rendering (MC-3).** A representation is converted separately on
   every page that shows it; never cache converted HTML across pages unless
   the cache key includes the page.
2. **Page context only through document attributes (MC-5).** Macros must not
   hold "current page" state; read `sysml-element` via `pageElement`.
3. **Escape everything that comes from the model.** Inline macros return text
   that Asciidoctor writes verbatim (macros run after the special-characters
   substitution): use `html.EscapeString`. List/table text is substituted:
   use `passthrough(html.EscapeString(...))`.
4. **Model text is never read as AsciiDoc markup, except `doc` bodies**,
   which `sysml-doc` deliberately parses (MC-10).
5. **Folders follow element names, titles do not** (PT-1, PG-8). Changing a
   representation's title must never move a page.
6. **Nothing is written on failure**, and only a marked folder is replaced.
7. **A failing macro never stops the run** (MC-19): it leaves a placeholder
   and a warning. Only cancellation and renderer bugs return errors from a
   macro processor.
8. **Every macro name starts with `sysml-`** (MC-1) and is documented in
   `docs/macros.adoc` (MC-6) and in `requirements.org`.

## 7. Adding a macro

1. Add a requirement (`MC-n`) to `requirements.org`.
2. Add a fixture model in `testdata/<case>/` whose `//` notes state the
   expected output (see `testdata/metadata/` and `testdata/values/`). Check it
   parses: `bin/sysml-publish -model testdata/<case> -out bin/x`.
3. Create `macros/<name>.go` with a `const` for the name and a
   `register<Name>(g, e)` taking the shared `env`; call it from `Register`.
   - Register through `inline` or `block` (§5.4), never `g.InlineMacro` /
     `g.BlockMacro` directly: they give the page's element, the unescaped
     target and the macro as written, and turn the body's errors into a
     placeholder and a warning (MC-19). Return an error saying only what is
     wrong; never add the macro or the page to it.
   - Warn without failing with
     `e.warn(Warning{Page: c.page, Macro: c.macro, Message: ...})`;
     `Message` must not repeat the page or the macro.
   - Inline text from a node field: use `registerText`.
   - Block content: return a block (`CreateOpenBlock` + `ParseContent`,
     `CreateList`, `CreateTable`), or `nil` to drop the macro line.
   - Needs elements without pages: use the element facts or `e.m.Props`
     with the properties you need (cached); a whole-model `e.m.Query` only
     when nothing else answers (DS-4). Needs evaluation: use
     `e.m.Evaluate(e.ctx, n.QualifiedName, ...)`.
   - Never read a standard-library element you only need the name of: its
     `Ref` carries the qualified name, and a read is slow (§13, 13).
4. Test in `macros/<name>_test.go`: convert a snippet with
   `ad.Convert(src, &asciidoctor.Options{Attributes: Attributes(node.ID)})`.
   Cover: the page's own element, an inherited representation (the usage's
   page, not the owner's), an unnamed element, the "nothing to show" case,
   escaping, and failures: `mustFail(t, ad, src, attrs, "<page>: <macro>:
   <message>")` checks the exact warning and the placeholder (`setup` and
   `setupFixture` record warnings).
5. Document it in `docs/macros.adoc` with an example, and verify the example
   end to end with the CLI.

## 8. Testing conventions

What the suite is:

- Tests go through the real stack: fixtures are parsed by a real
  sysml-grpc, rendered by asciidoctorg. No mocks, no build tags. The model
  leaves the test process — to the Service (DS-1) — and `dot` still runs as
  a program: the `views` tests run `dot` and skip when it is not on `PATH`,
  and the `sysml-view` tests in `macros` render with a map (`fakeViews`), as
  the `Pages` of the examples is one.
- **`internal/servicetest.API(t)` gives a test the address of a SysML
  service**: `$SYSML_PUBLISH_TEST_API` when set (a service someone else
  started), else `internal/service.Start` — a `sysml-grpc` found on `PATH`,
  or one built from the OpenSysML checkout (`$OPENSYSML_DIR`, default
  `../sysmlv2/OpenSysML` relative to this module) and run with `-port 0
  -report-address -health-port 0` — the logic the CLI's `-api local`
  runs. One service per test process, shared by its tests; every
  load and every `run(...)` takes its
  `-api` from it. Nothing to obtain → the test **skips**, and the skip
  names the reason, so a CI run without a service is visible, not silent.
  The service lives until the test process does (Linux kills it with the
  process); a runnable Example, which cannot skip, calls `Service()`
  instead and fails when nothing can be obtained.
- The suite is slower over the wire: each test package starts its own
  service (a cache-hot spawn is well under a second), every model read is a
  round trip, and each package process builds the service once. `go test
  ./...` is on the order of a minute, `-race` a few; keep an eye on it
  rather than assuming the old seconds.
- One folder per fixture under `testdata/`; expected results live as `//`
  notes in the fixture or as tables in the test. Fixtures must be committed
  with the tests that use them.
- `tree` tests assert page paths and representation lists; `site` tests
  assert HTML fragments (`mustContain`) or exact `Site.Warnings`; `macros`
  tests parse rendered HTML with small regexps; `cmd/sysml-publish` tests run
  `run(...)` and assert log records, exit codes and files written.
- `docs/example/` is generated output, committed for the README. After a
  change that alters rendered pages (templates, CSS, macros, the model),
  regenerate it: `bin/sysml-publish -api local -model model -out docs/example`.
- `testdata/smoke` is a frozen copy of an older version of the user's model
  (then `model/smoke.sysml`, now the multi-file UAV model in `model/`); "smoke"
  names the smoke tests that use it. The user's `model/` is not a test input
  and has since diverged.
- `go test -race -shuffle=on ./...` passes; keep it that way.

Rules for new tests:

- Use `t.Context()`, not `context.Background()`, and close what you open
  (`t.Cleanup(func() { m.Close() })` for a loaded model).
- Every load takes its service from `servicetest.API(t)` (a fixture load or
  a `run(...)` `-api`); never hard-code an address.
- Table-driven tests use a slice of named cases and `t.Run(name, ...)`, never
  a map (random order) or a bare loop (the first `Fatal` hides the rest).
- Assert exact values where they are known (`reflect.DeepEqual` /
  `slices.Equal` on lists, full warning lines), not counts or substrings that
  also match something else. Negative checks must be able to fail: never
  depend on Asciidoctor's whitespace.
- Do not pin the wording of OpenSysML or asciidoctorg messages; pin ours and
  check theirs by type (`errors.As`).
- After writing a test for a rule, break the rule once in the code (drop the
  `%w`, remove the dedup) and see the test fail.
- `TestTreeKeepsTheDefaultLogger` (render) changes the global logger: it must not
  run in parallel with anything. Do not add `t.Parallel()` until separate
  asciidoctorg instances are known to be safe to use concurrently.

Known gaps (from the test review, November 2026):

1. **Untested behaviour**: the between-pages check in `Render` and the
   check before `Write` in `publish` (the CLI test cancels before parsing;
   `TestConversionStopsWhenCancelled` covers a cancelled conversion); `Site.Write` into an existing empty folder or onto
   a file; the refused replacement at CLI level; cycle safety of inheritance;
   `-h` and the usage-error text; the examples in `docs/macros.adoc`.
2. **Weak assertions**: the whitespace-dependent `"<h1>Second</h1>\n<div"`
   check in `TestPageTitles`; `Contains(out, "tag")` in `TestMembers`;
   count-only checks in `TestUnnamedElements` and
   `TestMetadataByQualifiedName`; `de.Errors[0]` (panics if empty) in
   `TestLoadReportsErrorsWithLocation`; the `""` sentinel in
   `TestValueErrors`.
3. **No subtests**: about ten table tests loop without `t.Run`
   (`TestInheritance`, `TestName`, `TestQualifiedName`, `TestShortName`,
   `TestPageTitles`, `TestMetadata`, `TestSegment`, `TestUsageErrors`, ...).
4. **`tree` is pure but tested only through the parser.** Hand-built
   models (a fake behind `Props`/`Query`) would cover missing owners and
   single- vs multi-valued `Refs` in microseconds. `TestHrefEscapesSegments`
   lives in `site_test.go`, so `tree.Href` shows 0% coverage in `tree`, and
   its shortest-path cases are only checked through sidebar HTML.
5. **Duplicated setup**: fixture loading is written about seven times
   (`build`, `load`, `render`, `setup`, `setupFixture`, inline in
   `TestMembers`/`TestValue`/`TestValueErrors`; `page` and `pageAt` are the
   same function), and most loads never close the client. A shared helper
   package (`internal/testfixture`: `Load(t, name)` with cleanup) fixes both.
6. **Not yet written**: a golden-file test of the rendered smoke site
   (`-update` flag), fuzz targets for `segment`/`unique`, `formatUnit`,
   `Href`.

## 9. Dependencies and their quirks

**OpenSysML** (`client/opensysml`, dialing sysml-grpc over Connect):
- Pin the OpenSysML version: `Dial`, `ParseSources` with inline documents,
  `Query` and `ElementProperties` are on
  `develop`, not in a tag yet.
- **`Dial` connects lazily.** An unreachable service shows on the first
  call as a `CodeUnavailable` `*StatusError`, not at dial time; ER-1's
  message names the address around it. The capability preflight in
  `Folder.Load` makes that first call, so a dead service fails the run
  before any model is sent.
- **`ParseSources` reads `file_path` documents itself**, so a remote tool
  sends every document as inline `content`, named by its path, `.kerml`
  with `Language: LanguageKerML` (inline content has no extension to infer
  from). Diagnostics then name documents by those paths, which keeps
  `file:line:col` diagnostics and the macro-warning locations working
  unchanged.
- **A service that lacks `evaluate_subject` drops `WithSubject` silently**
  (the client does not gate it), and one without `render_view_form` drops
  `WithForm` (it refuses that call) — either quietly changes what
  `sysml-value` or `sysml-view` show. The preflight requires both, with
  `parse_sources`, `query`, `element_properties` and `render_view`.
- Comment bodies keep a trailing space before `*/` (trimmed for docs, not for
  representations — harmless for AsciiDoc).
- **Ids**: `ElementProperties` returns the element form's ids (`M__pd`,
  `U___401`); `Query` names elements by qualified or positional name
  (`U::@1`); `Props` accepts all three. Library elements have uuid ids and
  carry their library qualified name in the reference.
- **Query** returns only the model's own elements, not relationships or
  implicit multiplicities, in declaration order; a top-level element has no
  `owner` in it (root namespaces are not returned).
- A usage's `type` includes the types it gets through subsetting or
  redefinition (`part front :> wheels;` gives `Wheel`; v0.12.1 left them
  out).
- Every usage owns a `Multiplicity` member, its implicit `[1..1]` included
  (unnamed, through an `OwningMembership`); `tree` leaves multiplicities out
  of the members `sysml-members-list` lists.
- Derived properties are answered by `ElementProperties` (`name`, `type`,
  `documentation`, `textualRepresentation`, `owner`, `ownedMember`, ...);
  one the metaclass does not declare reads as empty (`Unsupported`).
- `Evaluate` returns typed values; derived units come back qualified and
  quoted (`SI::'m³⋅s⁻¹'`), arithmetic is done in SI base units
  (`10 [km/h] * 2` → `50/9 m/s`), composite attribute values come back as
  `InstanceID`.
- `RenderView` writes a view on the service with `WithForm` (TBD-9, §13 9):
  `dot` for a diagram, `csv` or `text` for a table. `views.InProcess` asks
  for DOT, lays it out with `dot -Tsvg`, and asks for the data when the
  view has no diagram form — a table's `Columns` and `Rows`. There is no
  SVG form, so Graphviz stays. A view needs `private import Views::*` and
  `StandardViewDefinitions::*` in the model for `render as…`.
- `Query` supports a closed set of properties (on develop: `body`,
  `language`, `textualRepresentation` added) and one value each (§13, 1).

**asciidoctorg** (Asciidoctor in an embedded Ruby, no cgo):
- Extensions are registered per instance; there is one instance per run.
- Inline macros run after special-character and quote substitutions; their
  output is not escaped.
- A line of the form `[...]` on its own is a block attribute line, not text
  (easy to trip over in tests).
- An inline macro in long form needs a target: `sysml-value:[]` is left as
  plain text, not an error. `MacroOptions.Regexp` replaces the pattern (first
  group: target, second: attributes) for optional targets or targets with
  spaces.
- **Inline macros run after the special-characters substitution**, so their
  target arrives HTML-escaped: `'a<b'` as `'a&lt;b'`. Unescape it.
- An error returned by an extension comes back as `*asciidoctor.Error`:
  message `asciidoctor: <Go error text> (RuntimeError)`, with the Go error in
  `Cause` (reachable with `errors.Is`/`As`).
- **Custom output formats**: `RegisterConverter` adds a Go converter for a
  backend name; it can hand the nodes it does not handle to
  `BuiltinConverter("xhtml5")` (since `v0.2.1`; `"html5"` leaves void
  elements unclosed, which XML formats such as Confluence's reject).
- **`asciidoctor.New()` replaces the process-wide default slog logger**
  (still in `v0.2.1`) with
  a `*log.ConfdHandler` from its `confd` dependency, which drops Debug
  records. `render.newAsciidoctor` restores the previous default
  (`TestTreeKeepsTheDefaultLogger` guards it); never rely on
  `slog.Default()` inside rendering code.

**SysML gotchas met so far**: `derived`, `standard`, `flow` are keywords (quote
them as names); `import` needs `public`/`private`; a `=` value is fixed and
cannot be redefined (use `default`); degrees Celsius is a scale, not a unit.

## 10. Architecture review

### Strengths

- **Clear, one-directional pipeline** with pure core logic: `tree` has no
  I/O and decides every page, path and inheritance question, which made the
  requirements testable one by one.
- **One way to read the model** (DS-4): tree and macros read it through
  SysML v2 API operations — Query, element reads, evaluation — and the tool
  now delegates those operations to the Service (DS-1): the model is parsed
  and held there, and these reads are its API. Nothing above the read
  layer changed for the transport swap.
- **The macro mechanism is small and uniform**: page context via a document
  attribute, one registration point, shared helpers for text, links and
  passthrough. Adding the seven macros needed no change to the renderer
  beyond the attribute and the warnings channel.
- **Safety by default**: nothing written on failure, marked-folder rule,
  HTML escaping at every model→HTML boundary, Asciidoctor in secure mode.
- **Tests run the real stack** on small fixture models that document their
  own expected output.
- **Requirements traceability**: code comments cite requirement IDs; the
  requirements file records decisions and open questions.

### Weaknesses and risks

1. **Reliance on an untagged OpenSysML API, now on both ends.**
   `ElementProperties` and the query properties used are on `develop` only;
   element ids are OpenSysML's element-form ids, and several behaviours
   (trailing spaces, unit spellings, derived `type`, implicit
   multiplicities) are observed, not specified. The client and the service
   it dials are two builds of that develop tree that must agree; the
   capability preflight catches a service missing what the client assumes,
   but wire semantics (diagnostic naming, capability gating inside the
   client) are observed, not specified. An OpenSysML upgrade can change
   pages silently (§13 lists what has already).
2. **Rendering cost scales with pages × inherited representations ×
   macros.** Every page re-converts every representation it shows (MC-3),
   and `sysml-value` calls the evaluator per macro per page. Model reads are
   cached, so repeated lookups are cheap; the inheritance walk reads every
   usage's and definition's specializations once (`-v` logs the counts).
   Fine for small models; a definition shared by hundreds of usages
   multiplies the conversions.
3. **Rendering needs the live service.** Macros read the model through
   `sysml.Model`; reads, evaluation and view rendering are RPCs to
   sysml-grpc now, and every uncached read is a round trip. The caches
   (the per-element property cache, the name, metadata, kind and
   connection indexes) carry the repeats, and a slow service slows
   rendering proportionally. Rendering stays tied to the client's
   lifetime, for this tool and for any module using the macros.
4. **Source locations are found by text.** The model gives no source
   positions (§13, 15), so a failing macro's file and line come from
   matching the representation's body against the model files (§5.3): two
   identical bodies are both located at the first, and a body OpenSysML
   changes beyond its indentation is not found (the warning then has no
   location). Failures pass silently in CI: the run succeeds, by decision
   (MC-19).
5. **Test gaps** (§8): the main paths are covered (normal run, JSON +
   timings, model errors, ER-4, macro failures, warnings, usage,
   interruption), but several stated rules are not (cancellation during
   rendering, `Write` edge cases, inheritance cycles), table tests have no
   subtests, some assertions cannot fail, and the docs' macro examples are
   checked by hand.
6. **Value formatting heuristics** (`formatUnit` strips package prefixes and
   quotes; rationals become floats) are reasonable but unspecified for units
   with spaces or for values beyond float precision.
7. **`Site.Write` is not atomic** (remove, then write).
8. **Pinned to an untagged OpenSysML**: `go.mod` uses an OpenSysML
   `develop` commit, whose behaviour can still change before it is tagged,
   and the sysml-grpc a user runs is a build of that tree of their own; the
   preflight catches only capability gaps the client knows about, not
   behavioural drift. asciidoctorg is at its latest tag, `v0.2.1`; later
   fixes on its `master` reach users only with a new asciidoctorg tag.
9. **Repository hygiene**: commits have missed files (a commit referenced
   `registerMetadata` without `metadata.go`; tests were committed for a
   while before their `testdata/` fixtures). Before committing, check
   `git status` for untracked files the change needs, and run
   `go build ./... && go test ./...` on the staged tree.

### Recommendations (in priority order)

1. **Close the test gaps of §8**, cheapest first: subtests and weak
   assertions, then a shared fixture helper, then the untested rules.
2. **Move model reading behind the tree**: precompute what macros need
   (metadata defs by name, metadata usages per element) in `tree.Build` or a
   small index built once, so macros never scan the model. Over the network
   this matters more: a cached read saves a round trip, so the fewer reads
   the tree leaves to per-macro per-page scanning, the better the wire
   behaves.
3. **Cache by (page, representation) only where it matters**, or skip
   per-page conversion for representations that contain no `sysml-` macro
   (a cheap substring check), which restores sharing for the common case.
4. **Make `Write` atomic**: render into a sibling temp folder, then swap.
5. **Tag asciidoctorg releases** whenever this project needs its newer code
   (and push its local commits), then bump `go.mod`.

## 11. Open decisions

- **AM-2** — `xref:` links between pages (not scheduled).

## 12. Upstream issues (to report or fix in the dependencies)

- **asciidoctorg**: `asciidoctor.New()` replaces the process-wide default
  slog logger (via `confd`). Worked around in `render.newAsciidoctor`; a
  library should not change a program's global logger.
- **OpenSysML**: see §13.

## 13. OpenSysML issues

Problems met in OpenSysML while working on this tool: one entry each, with
the version, what was seen, how to reproduce it, and the workaround used
here. Add an entry whenever a new one is found; mark it **fixed in <version>**
rather than deleting it, and drop the workaround once `go.mod` has the fix.
"develop" means the commit `go.mod` pins (`6f9622b8a9`, 2026-11-10). The
entries were found at `e074d3f40c` (2026-11-09) or earlier and have not been
re-checked since; the workarounds still pass the tests.

1. **Query keeps one value per property** (v0.12.1, develop). The query
   evaluator stores only `values[0]` of each selected property
   (`internal/semantic/query/query.go`), and `QueryElement.Properties` is a
   `map[string]string`, so a list cannot be returned at all. Seen:
   `documentation` and `textualRepresentation` of an element with two docs or
   two reps give the first one only. Workaround: query the `Documentation` /
   `TextualRepresentation` elements by `owner`, or read the element with
   `ElementProperties`, whose values are lists.
2. **Query `general` is typing and `:>` only** (v0.12.1, develop). For a
   usage it is its typings, for a definition its subclassifications; a
   usage's subsetting and redefinition (`part y :> x`) are absent, so
   "what specializes X" cannot be asked with one query. Workaround: build the
   generals graph from `ElementProperties(id, "ownedSpecialization")` and
   each relationship's `general`.
3. **No representation properties in Query** (v0.12.1; **fixed in develop**,
   PR #1926). `body`, `language`, `textualRepresentation` and `locale` were
   unknown query properties (`unknown query property "language"`).
4. **Query reports no `type` for a `MetadataUsage`** (v0.12.1; not yet
   checked on develop): which metadata definition a usage has cannot be
   queried. Workaround: `ElementProperties(id, "metadataDefinition")`.
5. **Reflective reads of unnamed elements fail without a cast** (v0.12.1).
   `(X meta SysML::Element).textualRepresentation` and `.documentation`
   evaluate to `null: unsupported: metaobject of an element with no
   qualified name`; adding `as SysML::TextualRepresentation` (or
   `SysML::Documentation`) and reading `.body` works. Not used here:
   `ElementProperties` replaces it.
6. **Positional names do not resolve in expressions** (v0.12.1):
   `Evaluate` of `(M::D::'@0' meta SysML::TextualRepresentation).body` fails
   with `unresolved reference: M::D::@0`, although Query and
   `ElementProperties` accept and return positional ids. An unnamed element
   cannot be an expression's subject.
7. **The cancellation error does not wrap `context.Canceled`** (v0.12.1):
   detect cancellation with `ctx.Err()` (§5.5).
8. **The unit `m^3/h` is rejected** with "Must be a valid feature" (v0.12.1;
   cause not investigated).
9. **View rendering is not public API** (v0.12.1; **fixed in develop**
   `56febd38bcad`, 2026-11-11): the DOT and Mermaid writers are in
   `internal/ir/view`; `RenderView` gave only the view's data, and
   `RenderViewRequest` had no form. `WithForm` now writes the view (DOT,
   CSV, Mermaid, …), and `views.InProcess` uses it; there is no SVG form,
   so Graphviz still lays a diagram out (TBD-9, MC-21).
10. **Export changes between v0.12.1 and develop** (not bugs, but they
    changed this tool's output): every usage owns an unnamed `Multiplicity`
    member, its implicit `[1..1]` included (left out of the members `sysml-members-list` lists); a
    usage's `type` includes the types it gets through subsetting or
    redefinition.
11. **`develop` needs goldmark v1.8.7** while v0.12.1 used v1.8.6; a module
    cache filled for v0.12.1 cannot build develop with `GOPROXY=off`.
12. **`ElementProperties` lists library relationships it cannot read**
    (develop). `ScalarValues::Real`'s `ownedSpecialization` holds a
    `Subclassification` whose id (`bead6e69-…`) `ElementProperties` answers
    with "element not found". Reproduce: a model with `attribute mass :
    Real;`, read `ownedSpecialization` of the type of `mass`, then `general`
    of the relationship. Workaround: `model.Generals` skips relationships it
    cannot read, so specialization chains stop at such library elements.
13. **Reading a library element is slow** (develop). The first
    `ElementProperties` of any standard-library element builds the element
    form of its whole library document (`LibraryElementForm` →
    `modelToRDF`): about 3 s to build the tree of `model/` when the
    inheritance walk followed generals into `ScalarValues` and the base
    part definitions. Workaround: `model.Contains` (the model's own ids,
    from the all-elements query in `Load`) stops `Generals` at library
    elements without reading them; type names come from the reference's
    qualified name, also without a read.
14. **No source text for expressions** (develop). `ElementProperties` of an
    expression gives its parts (`operator`, `argument`, `referent`, a
    literal's `value`) but not the text as written, which the API JSON
    export carried as `sysx:sourceText`. Seen: the metadata default
    `1 + 2` is an `OperatorExpression` with two `LiteralInteger`
    arguments. Workaround: `metadataIndex.valueText` writes literals,
    references and operators back; any other expression shows as its
    metaclass name.
15. **No source positions for elements** (develop). Neither `Query` nor
    `ElementProperties` gives the file and line an element is declared at;
    only diagnostics carry a `Span`. A comment body is also not its text as
    written: OpenSysML strips the common indentation and the line break
    after `/*`. Seen: the representation
    `/*⏎    At sysml-value:position[], …⏎    */` reads back as
    `"At sysml-value:position[], …\n"`. Workaround: `render.sources` matches
    the body's lines, whitespace aside, between `/*` and `*/` in the model
    files to place macro warnings (MC-19, §5.3).
16. **`Feature::type` is the derived feature type** (seen on the service at
    `34f6f6ce49`, 2026-11-15; OpenSysML #2198, between the pinned develop
    and that build). `type` now answers a feature's written typings and the
    types it inherits through subsetting and redefinition, with a written
    type another written one specializes folded away — KerML 1.1
    §8.3.3.3.4 `deriveFeatureType`: `Sport :> Wheel` with
    `part both : Wheel, Sport` answers `[Sport]`, where the pinned develop
    answered `[Wheel, Sport]`. `sysml-members-list` follows the API's
    answer (DS-4); the written typings are in api-json's
    `sysx:mappingValues` only, which `GetElementProperties` does not
    expose. Updated the fixture notes, the test and MC-12's example.
