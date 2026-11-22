---
type: feature
priority: medium
created: 2026-11-21T00:00:00Z
created_by: OpenAgent
status: implemented
tags: [model, example, docs, uav, multi-file, sysml]
keywords: [model/aircraft.sysml, model/uav.sysml, docs/example, README.md, AGENTS.md, docs/sysml-publish.toml, CHANGELOG.md, Folder.Load, ParseSources, Generals, subsetting, sysml-fig, sysml-ifaces, sysml-view, uavSystem, UavPart]
patterns: [cross-file imports, cross-package interface figure, subsetting and inherited representations, view definitions, metadata tables, docs/example regeneration, page tree]
---

# FEATURE-001: Replace the aircraft example model with a multi-file fixed-wing UAV model

## Description

Replace the repository's single-file example model, `model/aircraft.sysml`, with a
multi-file model of a fixed-wing UAV that carries a generic payload and includes its
remote control. The whole system is modelled as one part (`uavSystem`) containing the
UAV and the ground-side remote control, so the command/control and data-link
interfaces have a common ancestor and their `sysml-fig` figures render. The UAV part
subsets a more general UAV defined in another package in another file. The model keeps
the example's role as the public demonstration of the tool's macros, and `docs/example`
is regenerated from it.

## Context

`model/aircraft.sysml` is the only model file in `model/` (plus an ignored
`smoke.sysml~`). It is the user's model, **not a test input**: `testdata/smoke` is a
frozen copy of an older version and must not change. The model is embedded (shortened)
in `README.md`, its generated site is committed in `docs/example/`, and `AGENTS.md`
refers to it.

The committed output is stale: `docs/example/` has an `airConditioner` page and README
mentions a `size` attribute, but the current `model/aircraft.sysml` declares neither
(`interface airDuct connect (engine.air, airConditioner.air)` references an undeclared
part). The current model therefore cannot be the source of the committed site. The
replacement must parse cleanly and the example must be regenerated.

The tool already reads a folder of `*.sysml`/`*.kerml` files as one `ParseSources`
(DS-2, `testdata/crossfile`), but the public example is single-file and does not show
cross-file references. The new model is multi-file and flat in `model/` (file layout
does not affect page paths; page paths follow element containment, PT-1).

## Requirements

### Functional Requirements

1. **Remove `model/aircraft.sysml`** and add a flat multi-file model in `model/`.
   Suggested files (names may be adjusted for sidebar order; tops follow sorted
   filenames and declaration order):
   - `model/definitions.sysml` — `package UavDefinitions`: `Size`, `PartMetadata`,
     `part def <up> UavPart` with the inherited AsciiDoc representation (title line,
     `sysml-doc`, `sysml-metadata-table`, mass/power `sysml-value` rows), and the
     part definitions used below (`Battery`, `Motor`, `Propeller`, `FlightController`,
     `Imu`, `Gps`, `Antenna`, `CommandReceiver`, `DataReceiver`, `Operator`,
     `Payload`, `CameraPayload :> Payload`).
   - `model/uavkinds.sysml` — `package UavKinds`: the **more general UAV**, a
     documented `part uav : UavPart` with its own representation (title, doc,
     metadata table, values). This is the element the system's UAV specializes.
   - `model/payloads.sysml` — `package Payloads`: the generic `Payload` definition
     (mass, power, data port, documented) and `CameraPayload :> Payload` (documented,
     own page, shows specialization).
   - `model/links.sysml` — `package Links`: `interface commandLink` and
     `interface dataLink`, each documented (title, `sysml-iface-ends`,
     `sysml-fig`).
   - `model/system.sysml` — `package System`: the whole system as **one part**
     `part uavSystem`, its view(s), and the two nested parts:
     - `part uav :> UavKinds::uav` (literal subsetting; see risk 1 for the fallback)
     - `part remoteControl` (bare part, no definition)
2. **UAV structure** (`part uav`): `fuselage`, `wing`, `tail`, `landingGear`,
   `motor`, `propeller`, `battery`, `flightController`, `imu`, `gps`,
   `payloadBay` (owning `payload : Payload`), `commandAntenna` (with an `rf` port),
   `dataAntenna` (with an `rf` port). Every named part gets at least a `doc` so it has
   a page.
3. **Wing**: `part wing : UavPart` with two **bare** child parts, `left` and `right`
   (documented), and `attribute size : Size` plus its own representation containing
   `sysml-value:size[]`, so the page shows `[composite]` and the composite warning
   (keeps the README log example true).
4. **Energy**: a battery (no fuel tank). Attributes shown via `sysml-value`, e.g.
   `capacity = 5000 [W*h]` and `voltage = 22.2 [V]`; adjust the unit spelling if
   OpenSysML rejects it (see risk 4). Electric motor + propeller.
5. **Remote control**: `part remoteControl` containing `commandReceiver`,
   `dataReceiver` (each with an `rf` port) and `operator : Operator` (with a `console`
   port), plus an operator-to-station interface (e.g.
   `interface operatorLink connect (operator.console, remoteControl.console)`), so the
   operator appears in `sysml-ifaces`/`sysml-fig`.
6. **Two links, two antennas**: `Links::commandLink` connects
   `uav.commandAntenna.rf` to `remoteControl.commandReceiver.rf`;
   `Links::dataLink` connects `uav.dataAntenna.rf` to `remoteControl.dataReceiver.rf`.
   Use qualified names in the `connect` expressions. The ends are both under
   `uavSystem`, so `sysml-fig` on the interface pages has a common ancestor
   (`uavSystem`) and must render.
7. **Views**: at least one `sysml-view` of the system (`view … { expose uavSystem;
   render asInterconnectionDiagram; }`), plus a second view for the UAV or the remote
   control if `expose` accepts a feature chain (risk 5). Views need the `Views::*`
   and `StandardViewDefinitions::*` imports.
8. **Macro coverage**: keep the existing demonstrations (inherited representations,
   metadata table, `sysml-members-list` by kind, `sysml-value` with units,
   `sysml-doc`, `sysml-rep`, short and qualified names, `sysml-view`) and add
   `sysml-ifaces` and `sysml-fig` **on all documented pages**, as requested. On
   definition pages the macros fail as accepted: `sysml-ifaces` silently drops its
   line for a part with no ports, and `sysml-fig` leaves a failure placeholder and a
   warning (not the original macro text) — see risk 2.
9. **Documentation updates**:
   - `README.md`: quick start (several file snippets, not one), page list rewritten
     for the new tree (`UavDefinitions`, `UavKinds`, `Payloads`, `Links`, `System`),
     example links, generated-site tree, macro examples' qualified names, config
     example title/baseURL, and the log example (`model/` with 5 files, new page
     paths, the `size` warning).
   - `AGENTS.md` §3 layout and §8: `model/` is now multi-file; update the
     `docs/example` description and the regeneration command.
   - `docs/sysml-publish.toml`: `title = "UAV documentation"`,
     `baseURL = "https://example.com/uav/"`.
   - `CHANGELOG.md`: a `Changed` entry under `Unreleased` for the example model.
   - Regenerate `docs/example/` with defaults (site title `Model documentation`, no
     canonical links), as now.

### Non-Functional Requirements

- The model must parse on the OpenSysML version pinned in `go.mod` (capability
  preflight and `ParseSources` unchanged); a service is needed to verify.
- No production code changes, no new macros, no test-fixture changes.
- Deterministic output: `docs/example` must be reproducible; no absolute paths in
  the committed site.
- Keep the example readable: the sidebar must not become a maze of undocumented
  empty pages.

## Current State

- `model/aircraft.sysml`: one package `Model` with `Definitions::AircraftPart` and
  `Project::drone`, four parts (wing, engine, propeller, fuelTank), two interfaces,
  a view, metadata and values. `airConditioner` is referenced but not declared.
- `docs/example/`: generated from an older/different model (has `airConditioner`,
  not consistent with the current file); `README.md` describes that stale output.
- `README.md` embeds a shortened single-file model, lists six pages, and uses
  `aircraft` names in macro, tree and log examples.
- `docs/sysml-publish.toml` says "Aircraft documentation" and
  `https://example.com/aircraft/`.
- `AGENTS.md` line 127 and §8 name `model/aircraft.sysml`.

## Desired State

- `model/` holds five flat `.sysml` files forming one model: definitions, the general
  UAV, payloads, links, and the system part.
- `part uavSystem` contains `uav` (subsetting `UavKinds::uav`) and `remoteControl`;
  the two link interfaces cross between them inside `uavSystem`.
- The UAV has the fixed-wing structure, battery, electric propulsion, avionics,
  payload bay with a generic payload, and two antennas; the ground side has two
  receivers and an operator.
- `docs/example/` is regenerated and matches the model; README, AGENTS.md,
  `docs/sysml-publish.toml` and CHANGELOG describe the UAV model.
- No `aircraft` references remain outside the allowed files (see Success Criteria).

## Research Context

### Keywords to Search

- `model/aircraft.sysml` — the file being replaced; the only model input.
- `docs/example` — committed generated site; must be regenerated.
- `Folder.Load`, `ParseSources`, `DS-2` — how multiple files become one model;
  cross-file references (`testdata/crossfile`).
- `Generals`, `PG-7` — inheritance walk that must find the general UAV through
  subsetting (`part uav :> UavKinds::uav`).
- `sysml-fig`, `sysml-ifaces`, `sysml-iface-ends` — macro behaviour and the
  common-ancestor constraint (`macros/fig.go`, `owners` in `macros/iface_ends.go`).
- `sysml-view`, `Views::*`, `StandardViewDefinitions::*` — view rendering, `dot`.
- `UavPart`, `PartMetadata`, `Size` — shared definition, metadata and the composite
  value demo.
- `uavSystem` — the common owner that makes interface figures work.

### Patterns to Investigate

- **Cross-file imports**: `testdata/crossfile` (`Lib` + `Top`), the only fixture
  that reads two files as one model.
- **Cross-package interface figures**: `testdata/figs` `ixy` (parts in `X` and `Y`
  share `F`); the new model needs the same shared owner (`uavSystem`).
- **Subsetting/inheritance**: `testdata/inherit`, and the `part front :> wheels`
  behaviour in AGENTS.md §9.
- **Views**: `testdata/views` and the current `aircraftInternals` view.
- **Metadata and values**: `testdata/metadata-table`, `testdata/values`.
- **Page tree and paths**: `internal/tree` tests, PT-1/PT-6; file layout must not
  change page paths.
- **Regeneration**: AGENTS.md §8 (`bin/sysml-publish -model model -out docs/example`).

### Key Decisions Made

- **Replace** `model/aircraft.sysml`; do not keep the old file.
- **Multi-file**, flat in `model/`, one top-level package per file (SysML has no
  partial packages).
- **Whole system as one part** `uavSystem` containing `uav` and `remoteControl`;
  this gives the link interfaces a common ancestor so `sysml-fig` works.
- **The system's `uav` subsets a more general `uav`** in `UavKinds` (another package
  and file); literal `:>` is the primary form (fallback in risk 1).
- **Two antennas, two interfaces** (command/control and data), ground station with
  **two separate receivers** and a **human operator**.
- **Battery** for energy, **electric motor + propeller**.
- **One wing part** with bare `left`/`right` parts; **single landing gear**.
- **Generic payload**: `part payload : Payload` in a payload bay, plus a documented
  `CameraPayload :> Payload` definition.
- **Links live in a top-level `Links` package**; interface ownership does not affect
  the figure's common ancestor.
- **`sysml-ifaces` and `sysml-fig` on all documented pages**; failures on definition
  pages are accepted for now (see risk 2).
- **`docs/example` generated with defaults**, not with `docs/sysml-publish.toml`.
- **Out of scope**: `testdata/` fixtures, new requirements, `docs/macros.adoc`
  examples, state machines/actions/flows/requirements in the model, and
  `model/smoke.sysml~`.

## Success Criteria

### Automated Verification

- [ ] `go build ./...` and `go vet ./...` pass.
- [ ] `go test ./...` passes unchanged (no fixture edits; `testdata/smoke` untouched).
- [ ] `bin/sysml-publish -api local -model model -out docs/example` (or
      `go run ./cmd/sysml-publish …`) exits 0.
- [ ] `docs/example/` is regenerated and contains pages for `UavDefinitions`,
      `UavKinds`, `Payloads`, `Links`, `System/uavSystem`, `uav` and its parts,
      `remoteControl`, and the two interfaces.
- [ ] `grep -ri aircraft` returns nothing in `model/`, `docs/example/`, `README.md`,
      `AGENTS.md`, `docs/sysml-publish.toml`, `CHANGELOG.md` (allowed: `testdata/`,
      `docs/macros.adoc`, `requirements.org`, `.opencode/`).
- [ ] `git status` shows no unintended files (fixtures, scratch dumps, `*~`).

### Manual Verification

- [ ] The sidebar shows the expected tops and the `System > uavSystem > uav/remoteControl`
      branches; every named part has a page.
- [ ] `uav`'s page shows its own representation first and the inherited
      `UavKinds::uav` representation after it (PG-7).
- [ ] The wing page shows `sysml-value:size[]` as `[composite]` and logs the
      composite warning.
- [ ] `sysml-ifaces::uav[]` and `sysml-ifaces::remoteControl[]` list the two links;
      `sysml-fig::commandLink[]` and `sysml-fig::dataLink[]` render (common ancestor
      `uavSystem`); `sysml-fig::uav[]` and `sysml-fig::remoteControl[]` render.
- [ ] The system view renders with `dot`; a table view (if any) needs no program.
- [ ] Definition pages show the accepted `sysml-fig` failure placeholders (and
      `sysml-ifaces` lines dropped), with warnings; no other warnings besides the
      composite `size` one.
- [ ] README's quick start, page list, example links, site tree and log example match
      the regenerated site.

## Related Information

- Requirements: PG-1, PG-7, PG-8, PT-1, PT-6, MC-3, MC-18, MC-19, MC-21, MC-25,
  DS-1, DS-2, DS-4.
- Fixtures: `testdata/crossfile`, `testdata/figs`, `testdata/inherit`,
  `testdata/views`, `testdata/metadata-table`, `testdata/ifaces`.
- AGENTS.md §3 (layout), §5.2 (`Generals`/subsetting), §5.4 (macro behaviour),
  §8 (example regeneration).
- `docs/macros.adoc` (macro reference; unchanged).

## Notes

Risks and open points for research/planning:

1. **Qualified subsetting syntax.** `part uav :> UavKinds::uav` is the literal
   reading of the request. If OpenSysML rejects a qualified target in `:>`, fall
   back to `part uav : UavKinds::Uav` with `part def Uav` in `UavKinds`; either way
   the `uav` page must show the inherited representation. Verify with the pinned
   service before writing the whole model.
2. **Definition-page macros.** `sysml-fig` on a `PartDefinition` with no ports fails
   with a placeholder and a warning (not the original macro text); `sysml-ifaces`
   drops its line silently. The user accepted this "for now", but the regenerated
   `docs/example` will contain the placeholders. If they look unacceptable, the
   fallback is to keep the two macros only on pages with ports/connections — a scope
   change to confirm before implementing.
3. **`sysml-fig::uavSystem[]`** fails (the system part owns no ports). On the system
   page use `sysml-fig::commandLink[]` / `sysml-fig::dataLink[]` (or the view)
   instead; do not put a failing figure on the main page.
4. **Units.** OpenSysML has rejected some unit spellings (`m^3/h`, §13). Check
   `[W*h]` and `[V]`; if rejected, use an equivalent spelling or SI units.
5. **View `expose`.** `expose uavSystem.uav` (a feature chain) may not parse; the
   current example only exposes a simple name. If it fails, use one system view
   (`expose uavSystem`) and rely on `sysml-fig` for the UAV and ground pages.
6. **Sidebar order** follows sorted filenames. Choose filenames (or prefixes) so the
   tops read sensibly; document the chosen order in README.
7. **`docs/example` size.** The model grows from ~7 to ~25 pages; keep the committed
   example intentional and check that no page is an empty stub.

## Follow-up (2026-11-22)

The model was scaled to a 50 kg UAV (42 kg empty, 8 kg payload) and extended with
a payload radio, a CAN bus linking the platform's digital systems, and a power
line with a BEC and a motor controller. The model now has 16 interfaces inside
`uav`, an `avionicsInterfaces` view, and 53 pages; `docs/example` is regenerated
and idempotent. See the plan's "Follow-up" section.
