# AGENTS.md — sysml-publisher

Guide for anyone (human or agent) changing this code: what the program does,
how it is built, the rules the code relies on, and an assessment of the
architecture as of November 2026.

The requirements live in `requirements.org` (IDs such as PG-7, MC-10, TBD-8
below refer to it). The user-facing macro reference is `docs/macros.adoc`.
Keep both in step with the code: every behaviour change updates the
requirement it implements, and every macro change updates the reference.

---

## 1. What the program does

`sysml-publish` reads a folder of SysML v2 textual models and writes a static
HTML site: one page per element that has an AsciiDoc textual representation
(`rep language "asciidoc" /* ... */`), arranged as the model's containment
tree, with the tree as a sidebar on every page. Representation bodies are
AsciiDoc, rendered by asciidoctorg, and may use `sysml-*` macros that pull data
from the model (names, docs, members, metadata tables, evaluated attribute
values).

```
sysml-publish [-model DIR] [-out DIR] [-v] [-log-format text|json]   # defaults: ./model, ./site
```

## 2. Building, running, testing

```sh
go build -o bin/sysml-publish ./cmd/sysml-publish
bin/sysml-publish                       # ./model -> ./site
go test ./...                           # parses and renders the fixture models
go vet ./...
```

- **Dependencies are local checkouts**, wired with `replace` in `go.mod`:
  OpenSysML at `/home/pete/projects/sysmlv2/OpenSysML` and asciidoctorg at
  `/home/pete/projects/asciidoctor-go/asciidoctor-go`. asciidoctorg's `master`
  cannot be fetched as a module (Ruby sources are git submodules), so do not
  replace the `replace` with a pseudo-version.
- **Offline builds** work once the module cache is filled (`go mod tidy` with
  network, once). In a sandbox without DNS, run Go with `GOPROXY=off`.
- **Toolchain**: Go 1.27.2+ (asciidoctorg's minimum).
- `bin/`, `site/` and the root `sysml-publish` binary are build output and
  should not be committed (there is no `.gitignore` yet).

## 3. Layout

```
cmd/sysml-publish/main.go   CLI: flags, wiring, messages, exit codes
internal/model/             load + parse (OpenSysML in-process) -> API JSON elements; Evaluate
internal/tree/              which elements get pages, inheritance, names, paths  (pure)
internal/site/              render pages (asciidoctorg + html/template), titles, write
internal/site/assets/       asciidoctor.css (Asciidoctor default, MIT) + layout.css
internal/site/page.html.tmpl
internal/macros/            the sysml-* AsciiDoc macros, one file per macro
model/                      the user's model (default input)
testdata/<case>/            fixture models, one folder per test case
docs/macros.adoc            user reference for the macros
requirements.org            requirements, decisions, open questions
```

## 4. Pipeline

```
 model/*.sysml, *.kerml
        │  model.Folder.Load
        ▼
 opensysml.New() ─ ParseFiles (one model, all files) ─ diagnostics: errors stop the run (ER-2)
        │  Convert(api-json)                          (engine kept open for Evaluate)
        ▼
 model.Model { Elements map[@id]Element, Order, Files, Warnings }
        │  tree.Build                                  pure, no I/O
        ▼
 tree.Tree { Root, Pages (pre-order), ByID, Model, Warnings }
        │  site.Render
        ▼
 asciidoctorg instance + macros.Register(ad, tree, warn)
   pass 1: convert every page's representations (per page — MC-3)
   pass 2: assign page titles per sibling group (PG-8, PG-9)
   pass 3: execute page.html.tmpl for every page + landing page
        ▼
 site.Site { Files map[path][]byte, Warnings }
        │  Site.Write            replaces the out folder only if it carries .sysml-publish
        ▼
 site/…/index.html, site/assets/*.css, site/.sysml-publish
```

Everything is in memory until `Site.Write`; a failing run writes nothing.

## 5. Packages

### 5.1 `internal/model`

- `Folder{Dir}.Load` walks the folder for `*.sysml`/`*.kerml` (sorted, so runs
  are deterministic), parses them **as one model** (`ParseFiles`, so
  cross-file references resolve — DS-2), splits diagnostics into errors
  (returned as `*DiagnosticsError`) and warnings, and converts the model to the
  SysML v2 API JSON element array (`Convert(..., FormatAPIJSON)`).
- `Element` is `map[string]any` with helpers `ID`, `Type`, `Str`, `Ref`, `Refs`
  (`Refs` accepts single or multi-valued references, so callers need not know
  the cardinality).
- The OpenSysML client stays open after a successful load for
  `Model.Evaluate(ctx, subjectQualifiedName, expression)`; the caller must
  `Close()` it (the CLI defers it; tests mostly do not).
- `Source` is an interface so a post-MVP SysML v2 API service source (DS-1,
  DS-3) can be added; today `Folder` is its only implementation, and
  `Evaluate` is not part of the interface.

### 5.2 `internal/tree`

Pure functions over `model.Model`; this is where the requirements about *which*
pages exist live.

- **Representations**: `TextualRepresentation` elements whose `language` is
  `adoc`/`asciidoc` (case-insensitive), grouped by `owner`. Others → warning
  (PG-4).
- **Inheritance** (PG-7): a generals graph from every element carrying both
  `specific` and `general` (FeatureTyping, Subclassification, Subsetting,
  Redefinition, ReferenceSubsetting). For usages and definitions, a BFS over
  it collects the representations of reached elements, nearest first, each
  once, cycle-safe.
- **Pages**: an element is *documented* if it has own or inherited
  representations; it gets a node if it is documented or has a documented
  descendant (PG-1, PG-6); other branches are pruned (PT-4).
- **Names**: `declaredName`, else the owning membership's `memberName`
  (redefinitions borrow names), else `Unnamed` with the `Unnamed` flag set.
- **Paths**: `Dir` is the slash path of folder segments (PT-1). Segments are
  made filesystem-safe; unnamed siblings become `unnamed-N` (PT-6); clashes
  (case-insensitive) get `-2`, `-3`.
- **Per-node data for macros**: `Docs` (own `doc` bodies, trimmed),
  `ShortName`, `Members` (owned members minus annotations).
- `Href(from, to)` is the one relative-link function (shortest path,
  segments URL-escaped, `+` as `%2B`).

Naming wart: `Node.Title` holds the element's **name** and `Node.Name` holds
the **qualified name** (or positional name `M::@2` for unnamed elements). The
page title is decided later in `site` and is not stored on the node.

### 5.3 `internal/site`

- `Render` creates **one** asciidoctorg instance per run and registers the
  macros on it.
- **Pass 1** (`renderer.body`): each page converts each of its
  representations with `ConvertDocument`, passing the page's element id as
  the document attribute `sysml-element` (`macros.Attributes`). The first
  representation (page order: own, then inherited) with an AsciiDoc header
  (`= Title`) supplies the page title; later ones are converted with
  `showtitle` so their titles stay visible (PG-10). Only a header title
  counts — Asciidoctor's `Doctitle` would fall back to the first section.
- **Pass 2** (`assignTitles`): per sibling group — document title, else
  `untitled-N`; empty ancestor pages use the element name; duplicates get
  `-N` (PG-8, PG-9). A title has an HTML form (heading, keeps inline markup)
  and a text form (tab and sidebar).
- **Pass 3**: `page.html.tmpl` with the sidebar built from the tree.
  Asciidoctor output is trusted as `template.HTML` (secure safe mode);
  everything else is escaped by `html/template`.
- `Site.Write` refuses to replace a non-empty folder without the
  `.sysml-publish` marker, then `RemoveAll`s and writes (CLI-4). Not atomic:
  a crash mid-write leaves a partial site (still marked, so the next run
  replaces it).
- Macro warnings arrive as `macros.Warning{Page, Macro, Message}` and are
  kept once per (page, macro) in `Site.Warnings` (MC-7), as
  `<page>: <macro>: <message>`. Deduplication is by those fields, never by
  the text, so rewording a message cannot change it
  (`TestWarningsOncePerPageAndMacro`).

### 5.4 `internal/macros`

`Register(ad, tree, warn)` adds every macro in one `ExtensionGroup` named
`sysml` (MC-4). Shared pieces in `macros.go`:

- `pageElement(parent, tree)`: reads `sysml-element` from
  `parent.Document()` and looks it up in `tree.ByID`. This is how every macro
  knows the page's element (MC-2, MC-5). A missing or unknown id is a
  renderer bug and returns an error.
- `registerText(g, t, name, func(*tree.Node) string)`: an inline short-form
  macro (`name:[]`) returning escaped text. `sysml-name`, `sysml-short-name`
  and `sysml-qualified-name` are one line each on top of it.
- `passthrough(html)`: wraps finished HTML in `+++…+++` for places where
  AsciiDoc substitutions would otherwise run over model text (list items,
  table cells).

| Macro | Form | Reads | Notes |
|---|---|---|---|
| `sysml-name:[]` | inline | `Node.Title` | escaped text |
| `sysml-short-name:[]` | inline | `Node.ShortName` | empty when none |
| `sysml-qualified-name:[]` | inline | `Node.Name` | positional name for unnamed |
| `sysml-doc::[]` | block | `Node.Docs` | each doc parsed as AsciiDoc (`ParseContent`) into an open block |
| `sysml-members::[]` | block | `Node.Members` | ulist; links to members that have pages |
| `sysml-metadata::<def>[]` | block | `Tree.Model` | table per MetadataUsage; bound value, else def value, else "(no value)"; supertypes' attributes after own |
| `sysml-value:<expr>[]` | inline | `Model.Evaluate` | OpenSysML evaluates with the page element as subject; units formatted; composite → `[composite]` + warning |

Error policy today: a macro that cannot resolve what the user wrote (unknown
metadata def, unknown attribute) returns an error, which fails the whole run
(TBD-8 is still open on whether to degrade to a placeholder + warning).

### 5.5 `cmd/sysml-publish`

`run(ctx, args, stderr) int`, tested in `main_test.go` against fixtures into
`t.TempDir()`.

- **Logging (CLI-6)**: every message is a `log/slog` record on stderr: text
  without timestamps by default, JSON with `-log-format json`; `-v` enables
  Debug records with a `duration` for each stage and each page. Model
  errors are logged one record each (`msg="model error" diagnostic=...`),
  and every error is logged exactly once, in `run`; the packages return
  errors and never log them.
- **Loggers are passed explicitly** (`model.Folder.Logger`, `site.Render`'s
  `logger`), not taken from `slog.Default`; see the asciidoctorg quirk in §9.
- **Cancellation (CLI-7)**: `main` uses `signal.NotifyContext`; the context
  flows into `Load`, `Render` (checked between pages) and `sysml-value`'s
  evaluations. An interrupted run writes nothing and exits 130. Detect it
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
7. **Every macro name starts with `sysml-`** (MC-1) and is documented in
   `docs/macros.adoc` (MC-6) and in `requirements.org`.

## 7. Adding a macro

1. Add a requirement (`MC-n`) to `requirements.org`.
2. Add a fixture model in `testdata/<case>/` whose `//` notes state the
   expected output (see `testdata/metadata/` and `testdata/values/`). Check it
   parses: `bin/sysml-publish -model testdata/<case> -out bin/x`.
3. Create `internal/macros/<name>.go` with a `const` for the name and a
   `register<Name>(g, t[, warn])`; call it from `Register`.
   - Inline text from a node field: use `registerText`.
   - Block content: return a block (`CreateOpenBlock` + `ParseContent`,
     `CreateList`, `CreateTable`), or `nil` to drop the macro line.
   - Needs elements without pages: use `t.Model`; needs evaluation: use
     `t.Model.Evaluate` with `n.Name` as subject.
   - If the macro needs new per-element data, prefer adding a field to
     `tree.Node` filled in `tree.Build` over scanning the model per call.
4. Test in `internal/macros/<name>_test.go`: convert a snippet with
   `ad.Convert(src, &asciidoctor.Options{Attributes: Attributes(node)})`.
   Cover: the page's own element, an inherited representation (the usage's
   page, not the owner's), an unnamed element, the "nothing to show" case,
   escaping, and errors.
5. Document it in `docs/macros.adoc` with an example, and verify the example
   end to end with the CLI.

## 8. Testing conventions

- Tests go through the real stack: fixtures are parsed by OpenSysML in
  process, rendered by asciidoctorg. No mocks.
- One folder per fixture under `testdata/`; expected results live as `//`
  notes in the fixture or as tables in the test.
- `tree` tests assert page paths and representation lists; `site` tests
  assert HTML fragments (`mustContain`); `macros` tests parse rendered HTML
  with small regexps.
- `testdata/smoke` is a copy of `model/smoke.sysml` at some point; the user's
  `model/` is not a test input.
- Test helpers create a new OpenSysML client and asciidoctorg instance per
  test and rarely close the client; fine for tests, not a pattern to copy.

## 9. Dependencies and their quirks

**OpenSysML** (`client/opensysml`, in-process):
- The API JSON form is flagged experimental by OpenSysML; the CLI prints a
  note. Pin the OpenSysML version.
- Comment bodies keep a trailing space before `*/` (trimmed for docs, not for
  representations — harmless for AsciiDoc).
- `Convert`'s doc comment says "a model of one document"; multi-document
  conversion works in practice (tested in `internal/model`).
- Derived properties (`documentation`, `textualRepresentation`,
  `representedElement`, `inheritedMember`) are absent from the API JSON; use
  `owner` and the specialization relationships instead.
- `Evaluate` returns typed values; derived units come back qualified and
  quoted (`SI::'m³⋅s⁻¹'`), arithmetic is done in SI base units
  (`10 [km/h] * 2` → `50/9 m/s`), composite attribute values come back as
  `InstanceID`.
- View rendering to DOT/Mermaid exists only in `internal/ir/view` and the
  `sysml` CLI (TBD-9).
- The standard `Query` RPC supports only a closed set of properties (no
  `body`, `language`); this tool does not use it.

**asciidoctorg** (Asciidoctor in an embedded Ruby, no cgo):
- Extensions are registered per instance; there is one instance per run.
- Inline macros run after special-character and quote substitutions; their
  output is not escaped.
- A line of the form `[...]` on its own is a block attribute line, not text
  (easy to trip over in tests).
- An inline macro in long form needs a target: `sysml-value:[]` is left as
  plain text, not an error.
- **`asciidoctor.New()` replaces the process-wide default slog logger** with
  a `*log.ConfdHandler` from its `confd` dependency, which drops Debug
  records. `site.newAsciidoctor` restores the previous default
  (`TestRenderKeepsTheDefaultLogger` guards it); never rely on
  `slog.Default()` inside rendering code.

**SysML gotchas met so far**: `derived`, `standard`, `flow` are keywords (quote
them as names); `import` needs `public`/`private`; a `=` value is fixed and
cannot be redefined (use `default`); degrees Celsius is a scale, not a unit.

## 10. Architecture review

### Strengths

- **Clear, one-directional pipeline** with pure core logic: `tree` has no
  I/O and decides every page, path and inheritance question, which made the
  requirements testable one by one.
- **Single source of model truth**: one parse, one API JSON conversion, one
  element map; macros and tree read the same data.
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

1. **Reliance on an experimental, OpenSysML-specific export.** The API JSON
   form is marked experimental, element ids are positional for unnamed
   elements, and several behaviours (trailing spaces, unit spellings) are
   observed, not specified. An OpenSysML upgrade can change pages silently.
2. **Rendering cost scales with pages × inherited representations ×
   macros.** Every page re-converts every representation it shows (MC-3),
   and `sysml-value` calls the evaluator per macro per page; `sysml-metadata`
   scans the whole model to resolve a def name on every call. Fine for small
   models; a definition shared by hundreds of usages multiplies all of it.
3. **Macros reach into two data sources.** Most read `tree.Node` fields;
   `sysml-metadata` reads raw `model.Element`s and `sysml-value` calls the
   engine. The tree is no longer the single view model, and `Tree.Model` plus
   an open engine handle couple rendering to the parser's lifetime.
4. **Error policy is inconsistent and harsh.** User mistakes in macros fail
   the whole run, the messages arrive wrapped in Asciidoctor's
   `FAILED: <stdin>: Failed to load AsciiDoc document -` prefix, and the
   failing file/line in the model is not reported. TBD-8 is open.
5. **Confusing names**: `Node.Title` (element name) vs `Node.Name` (qualified
   name) vs the rendered page title. Easy to pick the wrong one.
6. **CLI tests cover the main paths** (normal, JSON + timings, model errors,
   usage, interruption), but macro examples in the docs are still checked
   by hand.
7. **Value formatting heuristics** (`formatUnit` strips package prefixes and
   quotes; rationals become floats) are reasonable but unspecified for units
   with spaces or for values beyond float precision.
8. **`Site.Write` is not atomic** (remove, then write).
9. **Hard-coded environment**: `replace` directives point at absolute paths
   on one machine; the project cannot be built elsewhere without editing
   `go.mod`.
10. **Repository hygiene**: no `.gitignore`; commits so far have mixed or
    missed files (a commit referenced `registerMetadata` without including
    `metadata.go`). Run `go build ./... && go test ./...` on the staged tree
    before committing.

### Recommendations (in priority order)

1. **Decide TBD-8** and implement one macro failure path: placeholder on the
   page + warning with page, macro and source location, optional
   `-strict` flag to turn warnings into a failing exit code for CI. Strip
   Asciidoctor's wrapper from error messages.
2. **Add a `.gitignore`** (`bin/`, `site/`, `/sysml-publish`).
3. **Rename tree fields**: `Title` → `Name`, `Name` → `QualifiedName`
   (mechanical; touches tree, site, macros).
4. **Move model reading behind the tree**: precompute what macros need
   (metadata defs by name, metadata usages per element) in `tree.Build` or a
   small index built once, so macros never scan the model.
5. **Cache by (page, representation) only where it matters**, or skip
   per-page conversion for representations that contain no `sysml-` macro
   (a cheap substring check), which restores sharing for the common case.
6. **Make `Write` atomic**: render into a sibling temp folder, then swap.
7. **Make dependencies portable**: a `go.work` file (untracked) for local
   checkouts instead of absolute `replace` paths in `go.mod`, once
   asciidoctorg tags a release with the needed API.
8. **Resolve TBD-9** (view rendering) — the recommended path (sysml CLI → DOT
   → Graphviz SVG) introduces the first external programs; isolate it behind
   an interface so option 3 (public OpenSysML writer) can replace it.

## 11. Open decisions

- **TBD-8** — what a failing macro leaves on the page; whether it fails the
  run.
- **TBD-9 / AM-3** — how `sysml-view::<view>[]` turns a view into a picture.
- **AM-2** — `xref:` links between pages (not scheduled).
