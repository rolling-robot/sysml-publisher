# Render views in process with OpenSysML's public `RenderView`

Replace the shell-out to OpenSysML's `sysml` command in `views.CLI` with the
public Go API's written forms (`client.RenderView` + `WithForm`), so that
`sysml-view` (MC-21) no longer needs the `sysml` program installed. Graphviz's
`dot` still lays a diagram out as SVG; a table view needs no program at all.

Status: implemented, in four commits (70f6fb0 the pin, db61712
`views.InProcess` beside the old renderer, 7263190 the switch and the docs,
36ff25f the old renderer's deletion). The rename became an add-then-delete,
so that no commit mixes structure with behaviour, and `docs/example` is
byte-identical.

## What the API gives (verified)

`github.com/Open-MBEE/OpenSysML/client/opensysml`, `render_view.go`:

```go
rv, err := client.RenderView(ctx, parsed, "V::sys::interfaces",
    opensysml.WithForm(opensysml.ViewFormDOT))
rv.Text              // DOT, ready for `dot -Tsvg`
rv.Kind              // "interconnection", "table", "matrix", ...
rv.Columns, rv.Rows  // a table view's cells, with no form at all
```

- The in-process client serves it (`CapabilityRenderViewForm`).
- Introduced on `develop` at `56febd38bc` (2026-11-11), i.e. **after** the pin
  `v0.12.2-0.20261110074652-6f9622b8a9d5`, so the pin must move.
- The forms are `text, mermaid, markdown, dot, plantuml, d2, drawio, csv, tsv`;
  **no SVG**. `dot` stays.
- Probed with `testdata/views/views.sysml`:
  - `V::sys::interfaces` + `WithForm(ViewFormDOT)` → `Text` beginning
    `// view: …`, `// kind: interconnection`, then `digraph …`; `Kind` is
    `interconnection`, `Columns`/`Rows` empty.
  - `V::partsTable` + `WithForm(ViewFormDOT)` → refused:
    `opensysml: INVALID_ARGUMENT: V::partsTable: a table rendering is not written as dot; ask for text, markdown, csv or tsv`.
  - `V::partsTable` without a form → `Kind="table"`, `Columns =
    ["Element" "Kind" "Type" "Declared in"]`, `Rows` exactly the cells the old
    CSV parse produced (same table).
  - `V::sys` (not a view) → `opensysml: INVALID_ARGUMENT: not a view` (the
    service's message does not name the view).
  - An empty table still has `Columns` (the 4 headings), so `len(Columns) > 0`
    identifies a tabular view (table or matrix).

## What does not change

- `macros.Views` (`Render(ctx, qualifiedName) (*Rendering, error)`),
  `macros.Rendering`, `Registry.SetViews`, `ViewSVG`, the `sysml-view` macro
  itself, `AttrView`, per-view caching (MC-21: rendered once per run).
- Graphviz's `dot` for the layout of a diagram, and the `-dot` flag (CLI-8).
- `internal/site`, `internal/render`, the page tree, `docs/example` structure.

## Decisions

1. **Renderer name**: rename `views.CLI` to `views.InProcess` (it no longer
   runs a `sysml` CLI; it still runs `dot`). The package's public/doc name is
   `views.InProcess`. (Alternative: keep `views.CLI` to avoid an API rename,
   but the name would be misleading.)
2. **`-sysml` flag**: remove it. CLI-8 becomes `-dot` only; a missing `dot` is
   still no usage error (only the diagram macros fail).
3. **Table detection**: ask for DOT first (one call for the common case). On
   failure, ask for the data with no form; `len(data.Columns) > 0` means
   tabular, and `Columns + Rows[i].Cells` is the table (no CSV parsing).
4. **Error wording**: unwrap `*opensysml.StatusError` to `status.Message`, and
   prefix the view's qualified name when the message does not already start
   with it, so warnings read as before (`V::sys: not a view`, not
   `opensysml: INVALID_ARGUMENT: not a view`). Cancellation is returned as
   `ctx.Err()` and not cached, as today.

## Changes, file by file

### 1. `go.mod` + `go.sum`

Bump OpenSysML to the first commit with `WithForm`:

```
github.com/Open-MBEE/OpenSysML v0.12.2-0.20261111062923-56febd38bcad
```

`go.sum`:

```
github.com/Open-MBEE/OpenSysML v0.12.2-0.20261111062923-56febd38bcad h1:SkFcNoKKExcZxoBGLFMpegiD9G17yCcaUX1JocJaimA=
github.com/Open-MBEE/OpenSysML v0.12.2-0.20261111062923-56febd38bcad/go.mod h1:M29NNn9HhUK2BLV7SaPiB3yg1PUa4JBbfpEwIZOQXvM=
```

Verified: the sums were generated from the commit (the method reproduces the
current pin's known hashes exactly) and `GOWORK=off go build ./...` /
`go test ./...` pass against that version with these sums. Its `go.mod`
differs from the old pin only in the `toolchain` line, so no other dependency
moves.

### 2. `sysml/model.go`

One new read operation, in line with DS-4 (the SysML v2 API's render-view
operation):

```go
// RenderView renders the view named by its qualified name, in the form
// WithForm asks for, and as data (MC-21).
func (m *Model) RenderView(ctx context.Context, qualifiedName string,
    opts ...opensysml.RenderViewOption) (*opensysml.RenderedView, error) {
    if m.client == nil {
        return nil, errors.New("sysml: the model is closed")
    }
    return m.client.RenderView(ctx, m.parsed, qualifiedName, opts...)
}
```

`sysml` already returns `opensysml.Value` from `Evaluate`, so no new kind of
public dependency.

### 3. `views/views.go`

Replace `views.CLI` (`Files`, `Sysml` fields) with an in-process renderer:

- `Model *sysml.Model` (the parsed model; no file list), `Dot string` (kept),
  the same mutex + per-view cache.
- `render`:

```go
func (r *InProcess) render(ctx context.Context, qn string) (*macros.Rendering, error) {
    dot, err := r.Model.RenderView(ctx, qn, opensysml.WithForm(opensysml.ViewFormDOT))
    if err == nil {
        svg, err := r.layout(ctx, []byte(dot.Text))
        if err != nil {
            return nil, err
        }
        return &macros.Rendering{SVG: svg}, nil
    }
    // A table view has no diagram form; its data says whether it is one.
    data, dataErr := r.Model.RenderView(ctx, qn)
    if dataErr != nil || len(data.Columns) == 0 {
        return nil, viewMessage(err, qn)
    }
    return &macros.Rendering{Table: tableRows(data.Columns, data.Rows, qn)}, nil
}
```

- `tableRows(columns, rows, qn)`: a fresh `[][]string` with `columns` as the
  header row and each `rows[i].Cells` after it (copy, do not alias).
- `viewMessage(err, qn)`: `*opensysml.StatusError` → `errors.New(status.Message)`;
  prefix `qn + ": "` unless the message already starts with `qn`; other errors
  pass through.
- `layout` and `run`/`lastLine`/`refusal`/`or` stay as they are (`dot -Tsvg`).
- Delete `sysml` execution (`c.sysml`), `encoding/csv`, `OPENSYSML_DOT`, the
  `Files`/`Sysml` fields, and the `refusal` retry-for-CSV comment.
- Update the package doc: views are rendered by OpenSysML in process; Graphviz
  lays a diagram out; example `reg.SetViews(&views.InProcess{Model: m})`.
- Keep the mutex held across a rendering, so the type stays safe for
  concurrent use (`sysml.Model` is not).

### 4. `cmd/sysml-publish/main.go`

- Delete the `-sysml` flag and its help/usage line.
- `publish(ctx, logger, modelDir, outDir, dotCmd)`.
- `site.Render(ctx, t, logger, &views.InProcess{Model: m.Model, Dot: dotCmd})`.
- Update the `publish` doc comment (`views.InProcess`).

### 5. Tests

`views/views_test.go`:

- Load `testdata/views/views.sysml` in process (`opensysml.New` +
  `ParseFiles` + `sysml.Open`, close in `t.Cleanup`); skip only on a missing
  `dot` (`needDot`).
- `TestDiagram`: unchanged assertions (`<svg`, `<title>V::sys::interfaces</title>`).
- `TestTable`: same exact rows as today (the `Columns`/`Rows` path gives the
  same cells).
- `TestRefused`: expect `V::sys: not a view` (no `sysml: ` prefix).
- `TestMissingProgram`: only the `dot` case.
- `TestRenderedOnce`: prove the cache by `m.Close()` after the first render;
  a second `Render` still returns the first result.
- `TestCancelled`: unchanged (`ctx.Err()` checked before caching).
- New `TestTableNeedsNoDot`: with `Dot: "/nonexistent/dot"`, a table view
  renders and a diagram fails with `cannot run dot: …`.

`cmd/sysml-publish/main_test.go` `TestViewPrograms`:

- Drop `-sysml`; run with `-dot /nonexistent/dot`.
- With the `views` fixture that leaves exactly two warnings: the diagram macro
  on `V::sys`'s page fails (`cannot run dot: …`), the macro-level
  "V::sys is a PartUsage, not a view"; the two table views render (no
  warning), which is the evidence that tables need no program.

`macros/view_test.go` is untouched (`fakeViews`, no programs).

### 6. Requirements, docs, README, AGENTS

- `requirements.org`:
  - MC-21: "rendered by external programs (TBD-9): OpenSysML's `sysml`
    command, and Graphviz's `dot` for the layout" → rendered in process by
    OpenSysML's public `RenderView` written form; Graphviz's `dot` still lays
    a diagram out; a table view is built from the rows `RenderView` gives.
  - CLI-8: drop `-sysml`, keep `-dot`; needed only when a representation uses
    a diagram `sysml-view`.
  - TBD-9: mark decided — the public API writes the forms (develop
    `56febd38bc`); only "a picture without Graphviz" stays open.
- `AGENTS.md`:
  - §1 synopsis: `[-dot CMD]`.
  - §2 pin rationale: add "and for `RenderView`'s written forms
    (`WithForm`)".
  - §3 `views/` line: "renders views for sysml-view in process, with
    Graphviz's dot".
  - §5.4 `Views`: `views.CLI` → `views.InProcess`.
  - §5.5 Views (MC-21, CLI-8): `publish` passes the model and `-dot`; a
    missing `dot` fails only diagram macros.
  - §9 OpenSysML bullet: `RenderView` with `WithForm` gives DOT in process;
    `views.InProcess` lays it out with `dot`.
  - §13 item 9 ("View rendering is not public API"): mark fixed at develop
    `56febd38bc`, workaround dropped (Graphviz remains).
  - §10 recommendation 6: done for DOT.
- `README.md`: command synopsis; drop the `-sysml` row; reword `-dot` (only
  diagrams need it).
- `docs/macros.adoc`, `== sysml-view`: the view is rendered inside
  `sysml-publish`; only Graphviz's `dot` must be installed (for diagrams),
  `-dot` names another; a table view needs no program.
- `docs/example`: `model/aircraft.sysml` uses `sysml-view::wiring[]`. The DOT
  writer and `dot` are the same, so the SVG should be byte-identical: diff
  `docs/example/Model/aircraft/index.html` after the change; regenerate only
  if it differs.

## Verification

1. `gofmt -l .` clean.
2. `go build ./...`, `go vet ./...`.
3. `go test ./...`, then `go test -race -shuffle=on ./...`.
4. `GOWORK=off go build ./...` (published pin + go.sum) — the module cache
   must hold `v0.12.2-0.20261111062923-56febd38bcad`; with no network, use the
   local module proxy used to generate the sums.
5. `bin/sysml-publish -model testdata/views -out /tmp/x` and
   `bin/sysml-publish -model model -out docs/example`; compare
   `docs/example` with `git diff`.
6. `git status` for untracked files the change needs.

## Risks / notes

- `views.InProcess` renames a documented public type; the module is pre-1.0,
  but any external user of `views.CLI` breaks.
- Removing `-sysml` is a CLI change; scripts passing it get a usage error
  (exit 2).
- The pin is still an untagged `develop` commit (AGENTS §10-8); moving to the
  next tag later is desirable.
- Warnings lose the `sysml: ` prefix (they gain the service's own wording,
  with an `opensysml: <code>: ` prefix stripped).
- Per diagram, one `RenderView`; a table view costs two only when the DOT
  attempt fails. The cache keeps it once per run (MC-21).
- The `sysml` program stays useful for the user's own model work; it is just
  no longer a build dependency of this tool.

## Out of scope

- Rendering SVG in process (no SVG form; would need Graphviz or a layout
  engine, TBD-9 option 4).
- Mermaid/JavaScript renderings (pages stay static HTML).
- A remote SysML v2 API source (DS-1, DS-3); it would implement
  `macros.Views` through the same `RenderView` operation.
- The Confluence backend and `sysml-yogi` (separate ports).
