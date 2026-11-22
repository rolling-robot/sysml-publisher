# Multi-file UAV Example Model Implementation Plan

## Overview

Replace the single-file `model/aircraft.sysml` with a five-file fixed-wing UAV
model that becomes the repository's public example: a whole system
(`System::uavSystem`) holding the UAV and its ground remote control, a UAV
subsetting a general UAV in another package, battery/electric propulsion, a
generic payload, two RF links, and a human operator. Regenerate `docs/example`
from it and bring `README.md`, `AGENTS.md`, `docs/sysml-publish.toml` and
`CHANGELOG.md` in step.

No production code, tests, fixtures, requirements or macro reference change.
The model is documentation: every declaration is chosen to exercise a macro or
a page rule.

## Current State Analysis

- `model/aircraft.sysml` is one package `Model` with `Definitions::AircraftPart`
  and `Project::drone`; `airConditioner` is referenced but never declared, so
  the file cannot produce the committed `docs/example` (which contains an
  `airConditioner` page). It is also modified but uncommitted.
- `docs/example/` is generated from an older model: `Model/AircraftPart`,
  `Model/aircraft/{wing,engine,fuelTank,airConditioner}`. Its landing title is
  `Model documentation` (published defaults, no canonical links).
- `README.md` embeds a shortened single-file model, says "six pages", lists the
  old pages, uses `Model::aircraft::…` names in macro examples, shows the old
  config and log, and gives a regeneration command without `-api local` (which
  CLI-9 now requires).
- `docs/sysml-publish.toml` says "Aircraft documentation" /
  `https://example.com/aircraft/`.
- `AGENTS.md` names `model/aircraft.sysml` at §3 (layout) and §8 (smoke copy);
  §8's regeneration command also lacks `-api local`.
- `internal/model/model.go:202` has an `aircraft.sysml` example in a comment;
  that file is outside this ticket's scope and stays.
- `bin/`, `site/`, `*.api.json` and `*~` are ignored; `docs/example/assets/` is
  tracked (`mermaid.min.js` is not ignored).

## Desired End State

`model/` holds five flat files, one top-level package each:

| File | Package | Contents |
|---|---|---|
| `definitions.sysml` | `UavDefinitions` | `Size`, `PartMetadata`, `MHz`, `Chemistry`, `TailType`, `UavPart` (shared template), component definitions |
| `uavkinds.sysml` | `UavKinds` | the general `uav` usage |
| `payloads.sysml` | `Payloads` | `DataPort`, `Payload`, `CameraPayload` |
| `links.sysml` | `UavLinks` | `RfPort`, `ConsolePort` (no page) |
| `system.sysml` | `System` | `uavSystem`, `uav :> UavKinds::uav`, `remoteControl`, the three interfaces, two views |

`docs/example` is regenerated with defaults (`Model documentation`, no
canonical links), contains the pages in the tree below, and the four docs
files describe it. `grep -ri aircraft` finds nothing in `model/`,
`docs/example/`, `README.md`, `AGENTS.md`, `docs/sysml-publish.toml` or
`CHANGELOG.md`.

### Key Discoveries (verified in this planning session)

- **Qualified subsetting works end to end.** `part uav :> Kinds::uav` inherits
  the general usage's representations, and `sysml-value:range[]` evaluates with
  the page element as subject: `Sys::sys::uav` rendered `6 kg` and `97200 m`.
  No fallback to `part uav : UavKinds::Uav` is needed.
- **Inherited ports are not owned ports.** `port def Payload { port data; }`
  plus `part payload : Payload` gives `sysml-fig::payload[]: has no ports`.
  The usage must declare `port :>> data : …` to own the port; verified that the
  redefinition makes the figure render. This is why the payload usage redefines
  its port while the `Payload` definition owns one.
- **A definition may override a base `default` with `=`.** `part def Battery :>
  UavPart { attribute :>> mass = 0.85 [kg]; }` loads and renders `0.85 kg`;
  what fails is overriding a base written with `=`.
- **`System`, `Payloads`, `UavDefinitions`, `UavKinds`, `UavLinks` are not
  library package names.** All five load with zero model warnings. (`Links`
  and `Connections` are taken, hence `UavLinks`.)
- **`sysml-view` draws only connectors the exposed element owns.** Interfaces
  in `uavSystem` drawn, interfaces in a sibling package not drawn, even with a
  second `expose`. All three interface usages are members of `uavSystem`.
- **`sysml-fig` of a portless element fails** with `has no ports; a figure
  needs at least one`. With `sysml-fig::[]` in the shared template (the chosen
  option) 28 pages will show a failure placeholder and warn; this is accepted
  and must be reflected in the README log example.
- **`expose uavSystem::*; render asElementTable;` is the verified table-view
  shape** (`testdata/views/views.sysml:38-43`).

## Decisions

1. **`sysml-fig` on all documented pages** (user choice): it goes in the shared
   `UavPart` representation and in every full representation that does not
   inherit it. Portless pages show placeholders and warnings; no `sysml-fig`
   omissions to keep the example clean.
2. **Accept ~43 pages**: every component definition is documented and gets a
   page. No multi-typing trick to hide definitions.
3. **Document `operatorLink`**: it gets a page and a figure (frame:
   `remoteControl`).
4. **Semantic filenames**: `definitions.sysml`, `uavkinds.sysml`,
   `payloads.sysml`, `links.sysml`, `system.sysml`. Visible tops:
   `UavDefinitions`, `Payloads`, `System`, `UavKinds` (`UavLinks` has no page).

## What We're NOT Doing

- No changes to `testdata/` (not even `testdata/smoke`), production code,
  `requirements.org`, `docs/macros.adoc`, or `model/smoke.sysml~`.
- No new macros, requirements, tests, state machines, actions, flows or
  requirement elements in the model.
- No short names on every element; only `UavPart`, `wing`, `uav` (both),
  `uavSystem`, `Payload`, `CameraPayload`, `remoteControl` and `left`/`right`
  get one, so the `(…)` in titles demonstrates both the present and the empty
  case (as the current README already explains).
- No fix for unrelated stale README claims, except the one that the new
  example would directly contradict: "A failing macro stops the whole run".
- No commit; the change is left staged/working for review.

## Implementation Approach

The model is a rendering fixture with three layers of representation:

1. **One shared template** in `UavPart`: title, doc, metadata table, mass/power,
   members, interfaces, figure. Every definition `:> UavPart` and every usage
   typed by one inherits it.
2. **Data-only own representations** (no `= Title`) on definitions and airframe
   usages, holding the values specific to that element (`== Technical data`,
   `== Flight data`, …). Because they have no title, the inherited template
   supplies the page title and no duplicate heading is produced. The macros in
   them speak about the page element, so one `Battery` representation serves the
   definition page and every battery usage page with the usage's own values.
3. **Full own representations** on elements that inherit nothing: `uavSystem`,
   `remoteControl`, `left`, `right`, `Payload`, and the three interfaces.

Links are owned by `uavSystem` (view constraint). Ports are declared on the
**usages** that need them (antennas, receivers, operator, `remoteControl`),
except `Payload`, which owns `data` and whose usage redefines it.

## Phase 1: Write the five model files

### Overview

Delete `model/aircraft.sysml`; add the five files with declarations,
representations, metadata, links and views. Values are exactly the validated
set in `thoughts/research/2026-11-22_uav_attributes.md` §5–§7, with the
corrections below (battery energy, safe attribute names).

### Changes Required

#### 1.1 Remove the old model

**File**: `model/aircraft.sysml` — `git rm` it (it is uncommitted-modified; the
deletion is intended). Do not touch `model/smoke.sysml~`.

#### 1.2 `model/definitions.sysml` — `package UavDefinitions`

Imports at package level:

```sysml
private import ScalarValues::*;
private import SI::*;
private import MeasurementReferences::*;
private import SIPrefixes::*;
```

Contents, in order:

- `attribute <MHz> megahertz : FrequencyUnit { :>> unitConversion: ConversionByPrefix { :>> prefix = mega; :>> referenceUnit = Hz; } }`
  (`ConversionByPrefix` from `MeasurementReferences`, `mega` from
  `SIPrefixes`; validated across packages).
- `attribute def Size { attribute width : ISQ::LengthValue; attribute height : ISQ::LengthValue; }`
- `metadata def PartMetadata` exactly as today, **including the fixed**
  `attribute process : String = "TBD";` (never bind `process`).
- `enum def Chemistry { LiPo; LiIon; }`
- `enum def TailType { conventional; vTail; flyingWing; }`
- `part def <up> UavPart` with `mass : ISQ::MassValue default 0.1 [kg]` and
  `power : ISQ::PowerValue default 0 [W]`, and the shared representation
  (verbatim, including the figure):

```asciidoc
= sysml-name:[] (sysml-short-name:[])

sysml-doc::[]

sysml-metadata-table::PartMetadata[]

== Properties

[cols="1,1"]
|===
|mass | sysml-value:mass[]
|power | sysml-value:power[]
|===

== Members

sysml-members-list::PartUsage[]

== Interfaces

sysml-ifaces::[]

== Figure

sysml-fig::[]
```

- Component definitions, each `part def <…> :> UavPart`, each with a `doc` and
  a **data-only** own representation (section heading, `[cols="1,1"]` table,
  no `= Title`):
  - `Battery` — capacity 8 `A*h`, voltage 22.2 `V`, energy 177.6 `W*h`,
    maxCurrent 60 `A`, specificEnergy 209 `W*h/kg`, cells 6,
    chemistry `Chemistry::LiPo`, rechargeable true; mass 0.85 kg.
  - `Motor` — power 0.6 kW, torque 0.72 `N*m`, `kv` (not `kV`) 480,
    efficiency 0.85; mass 0.28 kg.
  - `Propeller` — diameter 330 `mm`, pitch 203 `mm`, staticThrust 12 `N`,
    blades 2; mass 0.06 kg.
  - `FlightController` — power 1.5 W, updateRate 400 `Hz`, processor
    "STM32H7", firmware **no value** (`(no value)`); mass 0.04 kg.
  - `Imu` — power 0.05 W, sampleRate 1000 `Hz`, gyroRange 2000, calibrated
    false; mass 0.01 kg.
  - `Gps` — power 0.3 W, updateRate 10 `Hz`, accuracy 2.5 m,
    constellations 4; mass 0.03 kg.
  - `Antenna` — frequency 2400 `MHz`, gain 3, connector "SMA"; mass 0.02 kg.
    **No port on the definition** (ports are owned by the usages so their
    figures render).
  - `CommandReceiver` — power 0.5 W, frequency 2400 `MHz`, sensitivity -100;
    mass 0.05 kg. No port.
  - `DataReceiver` — power 1.2 W, frequency 5800 `MHz`, sensitivity -95,
    dataRate 250000 `B/s`; mass 0.06 kg. No port.
  - `Operator` — `role` (not `name`) "pilot", certified true, flightHours
    120 h; mass 80 kg. No port.
- Overrides use `attribute :>> mass = …` / `attribute :>> power = …` (base is
  `default`, so `=` is accepted). No attribute named `name`, `type`, `owner`
  or another metamodel property (`role`, `configuration`, `kv` are used
  instead).

#### 1.3 `model/uavkinds.sysml` — `package UavKinds`

Imports: `ScalarValues`, `SI`, `UavDefinitions`.

- `part <U> uav : UavDefinitions::UavPart` with:
  - `doc /* …general fixed-wing UAV… */`
  - data-only own representation `== Flight data` with `cruiseSpeed`,
    `endurance`, `range`, `emptyMass`, `payloadCapacity` (not `mass`: the
    shared template already shows it).
  - `attribute :>> mass = 6 [kg];`
  - `cruiseSpeed : ISQ::SpeedValue default 18 [m/s]`
  - `endurance : ISQ::DurationValue default 90 [min]`
  - `range : ISQ::LengthValue = endurance * cruiseSpeed` → `97200 m`
  - `emptyMass : ISQ::MassValue default 4.2 [kg]`
  - `payloadCapacity : ISQ::MassValue default 1 [kg]`
  - no children.

#### 1.4 `model/payloads.sysml` — `package Payloads`

Imports: `ScalarValues`, `SI`, `UavDefinitions` (for `PartMetadata`).

- `port def DataPort;`
- `part def <PL> Payload` with `doc`, `port data : DataPort;`, mass 0.3 kg,
  power 5 W, dataRate 5000 `B/s`, and a **full own representation** (title
  `= sysml-name:[] (sysml-short-name:[])`, doc, metadata table, technical data
  table, `sysml-fig::[]`). Its figure renders on the definition page (owned
  port) and on usage pages that own a redefined port.
- `part def <CP> CameraPayload :> Payload` with `doc`, data-only own
  representation `== Camera data` (resolution 12, fieldOfView 84 `['°']`), and
  `attribute :>> mass = 0.45 [kg];`, `attribute :>> power = 7 [W];`. It inherits
  `Payload`'s representation and figure; its figure fails because the port is
  inherited, not owned (accepted).

#### 1.5 `model/links.sysml` — `package UavLinks`

- `port def RfPort;`
- `port def ConsolePort;`
- No representations, docs or attributes → no page, no model warnings.
- File name stays `links.sysml`; the package is `UavLinks` because `Links` is a
  Kernel Semantic Library package name (load warning).

#### 1.6 `model/system.sysml` — `package System`

Imports: `ScalarValues`, `SI`, `UavDefinitions`, `UavKinds`, `Payloads`,
`UavLinks`, `Views`, `StandardViewDefinitions`.

`part <us> uavSystem` (bare, no type):

- `doc /* …the UAV with its ground remote control… */`
- `@PartMetadata { owner = "pete"; revision = 4; reviewed = true; note = "flight-ready demonstrator"; }`
- attributes: `missionRange : ISQ::LengthValue = uav.endurance * uav.cruiseSpeed`
  (→ `97200 m`), `operatorCount : Integer default 1`, `deployment : String
  default "field"`, `flightReady : Boolean default true`.
- **Full own representation** (title, doc, metadata table, values table,
  `sysml-members-list::InterfaceUsage[]`, the two views, `sysml-fig::[]`).
  The figure fails (no owned ports) — accepted, and documented in the README.
- Children:
  - `part <U1> uav :> UavKinds::uav` with `doc`, data-only own representation
    `== Air vehicle` (`serial` "UAV-001", `autonomous` true), and
    `attribute serial : String default "UAV-001";`,
    `attribute autonomous : Boolean default true;`. Its page shows its own rep,
    then the inherited `UavKinds::uav` rep, then the inherited `UavPart` rep.
  - The 16 UAV children, each with a `doc`; those typed by `UavPart` directly
    get data-only own reps:
    - `fuselage : UavDefinitions::UavPart` — length 1.15 m, material
      "carbon fiber", watertight false; mass 1.1 kg.
    - `<W> wing : UavDefinitions::UavPart` — mass 0.95 kg, span 2.4 m,
      area 0.45 `m^2`, `meanChord = area / span`, `aspectRatio = span * span /
      area` (Real), airfoil "NACA 2412", and **`attribute size : Size;`** (no
      value) shown by `sysml-value:size[]` in the data rep → `[composite]` and
      the warning. Children:
      - `part <LW> left { … }` and `part <RW> right { … }`: bare parts, each
        with `doc`, `mass = 0.45 [kg]`, `tipDevice = "none"`, and a **full own
        representation** (title, doc, values table, `sysml-fig::[]`). Their
        figures fail (no ports) — accepted.
    - `tail : UavPart` — `configuration` `TailType::conventional`, area
      0.12 `m^2`, incidence 0 `['°']`; mass 0.35 kg.
    - `landingGear : UavPart` — wheels 3, retractable false, track 0.6 m;
      mass 0.25 kg.
    - `payloadBay : UavPart` — volume 2 `L`, maxPayload 1.2 kg, open false;
      mass 0.2 kg; owns
      `part payload : Payloads::Payload { doc; @PartMetadata { owner = "camera team"; } port :>> data : Payloads::DataPort; }`.
    - Typed by definitions, no own rep (inherit their definition's):
      `motor : Motor` (`@PartMetadata { revision = 1; }`),
      `propeller : Propeller`, `battery : Battery`
      (`@PartMetadata { owner = "pete"; revision = 2; reviewed = true; }`),
      `flightController : FlightController`, `imu : Imu`, `gps : Gps`.
    - `commandAntenna : Antenna` — `port rf : UavLinks::RfPort;`, frequency
      2400 MHz, gain 3, mass 0.02 kg, connector "SMA",
      `@PartMetadata { note = "spare"; }`.
    - `dataAntenna : Antenna` — `port rf : UavLinks::RfPort;`, frequency
      5800 MHz, gain 5, mass 0.03 kg.
  - `part <RC> remoteControl` (bare) with `doc`, `port console :
    UavLinks::ConsolePort;`, attributes mass 1.2 kg, channels 16, batteryLife
    8 h, range 10 km, screen true, and a **full own representation** (title,
    doc, values table, `sysml-members-list::PartUsage[]`, `sysml-ifaces::[]`,
    `sysml-fig::[]` — renders because it owns `console`). Children:
    - `commandReceiver : CommandReceiver` with `doc`, `port rf : RfPort;`
    - `dataReceiver : DataReceiver` with `doc`, `port rf : RfPort;`
    - `operator : Operator` with `doc`, `port console : ConsolePort;`
  - The three interface usages, owned by `uavSystem`:
    - `interface commandLink connect (uav.commandAntenna.rf, remoteControl.commandReceiver.rf)`
    - `interface dataLink connect (uav.dataAntenna.rf, remoteControl.dataReceiver.rf)`
    - `interface operatorLink connect (remoteControl.operator.console, remoteControl.console)`
    - Each has a **full own representation**: `= sysml-name:[]`, a sentence,
      `== Ends` + `sysml-iface-ends::[]`, `== Figure` + `sysml-fig::[]`
      (all three render: common ancestors `uavSystem`, `uavSystem`,
      `remoteControl`).
- Views inside `uavSystem`, after the interfaces:

```sysml
view systemInterfaces {
    expose uavSystem;
    render asInterconnectionDiagram;
}

view systemParts {
    expose uavSystem::*;
    render asElementTable;
}
```

`expose uavSystem.uav` (a feature chain) is not used.

### Representation matrix (summary)

| Element | Own representation | Inherited |
|---|---|---|
| `UavPart` | full (shared template) | — |
| component definitions (10) | data-only | `UavPart` |
| `uav` (Kinds) | data-only | `UavPart` |
| `Payload` | full | — |
| `CameraPayload` | data-only | `Payload` |
| `fuselage`, `wing`, `tail`, `landingGear`, `payloadBay` | data-only | `UavPart` |
| `left`, `right` | full | — |
| `uav` (System) | data-only | `UavKinds::uav`, `UavPart` |
| `uavSystem` | full | — |
| `remoteControl` | full | — |
| `commandLink`, `dataLink`, `operatorLink` | full | — |
| component usages (`battery`, `motor`, …), `payload`, antennas, receivers, `operator` | none (doc/metadata/ports only) | their definition |

### Success Criteria

#### Automated Verification:

- [x] `git status --porcelain model/` shows `aircraft.sysml` deleted and the
      five new files.
- [x] `bin/sysml-publish -api local -model model -out /tmp/opencode/uav-site`
      exits 0.
- [x] The scratch site contains every expected page:
      `UavDefinitions/{index,UavPart,Battery,Motor,Propeller,FlightController,Imu,Gps,Antenna,CommandReceiver,DataReceiver,Operator}`,
      `UavKinds/{index,uav}`, `Payloads/{index,Payload,CameraPayload}`,
      `System/index`, `System/uavSystem/index`,
      `System/uavSystem/uav/{index,fuselage,wing/{index,left,right},tail,landingGear,motor,propeller,battery,flightController,imu,gps,payloadBay/{index,payload},commandAntenna,dataAntenna}`,
      `System/uavSystem/remoteControl/{index,commandReceiver,dataReceiver,operator}`,
      `System/uavSystem/{commandLink,dataLink,operatorLink}/index.html`.
- [x] `UavLinks` has no page.
- [x] The run's warnings are exactly the expected set: one
      `sysml-fig::[]: has no ports; a figure needs at least one` per portless
      page (28 expected; confirm the count against the page list) and one
      `sysml-value:size[]: size is composite` on `wing`; no other warnings.
- [x] `system.sysml` loads with no model warnings (no `Unresolved`, no library
      name warning).

#### Manual Verification:

- [x] Sidebar tops read `UavDefinitions, Payloads, System, UavKinds`.
- [x] `System/uavSystem/uav/index.html` shows the page title once, then its own
      `== Air vehicle` section, then `== Flight data`, then the `UavPart`
      template; values are `6 kg` and `97200 m`.
- [x] `System/uavSystem/uav/wing/index.html` shows `[composite]` and the
      composite warning; aspect ratio `12.8`, mean chord `0.1875 m`.
- [x] `System/uavSystem/index.html` shows the metadata table, `97200 m`, both
      views, and the three interfaces.
- [x] `Battery` and `battery` pages show the technical data plus the inherited
      mass/power template; `motor` shows its metadata bindings; `flightController`
      shows `(no value)` for firmware.
- [x] `commandAntenna`, `commandReceiver`, `dataAntenna`, `dataReceiver`,
      `operator`, `remoteControl`, `payload`, `Payload` and the three interface
      pages render figures; portless pages show the failure placeholder.

---

## Phase 2: Structural verification and corrections (scratch output)

### Overview

Render to `/tmp/opencode/uav-site`, inspect the generated HTML for the claims
whose failure would otherwise be discovered after regenerating the committed
site. Fix the model, not the tool.

### Changes Required

No source changes unless a check fails. Decision rules:

- If `sysml-value:missionRange[]` fails on `uavSystem`, use the literal
  `97200 [m]` (same displayed value) and note the change.
- If `port :>> data` is rejected, declare `port data : Payloads::DataPort;` on
  the usage (new name, e.g. `dataPort`) and keep it out of any connection; the
  figure then shows one node.
- If the `asElementTable` view fails, drop `systemParts` and its macro; keep
  `systemInterfaces` (the fixture proves the renderer, so this should not
  happen).
- If `sysml-iface-ends::operatorLink[]` lists ends in an unexpected order,
  keep the declaration order (`remoteControl.operator.console` first).

### Success Criteria

#### Automated Verification:

- [x] A scripted page existence check over the list in Phase 1 passes.
- [x] `grep -c 'class="sysml-failed"'` on each expected portless page is 1; on
      each expected renderable page it is 0.
- [x] The scratch site has `assets/mermaid.min.js`; a page with a figure has a
      `<pre class="mermaid">` block; a page without one has no mermaid asset
      reference (the asset is written when any page shows a figure).
- [x] `uavSystem` page contains `<img src="data:image/svg+xml;base64,` for the
      diagram and a table for the parts view.
- [x] Warning output has no `Evaluate: evaluation failed`, no
      `unresolved reference`, and no `asciidoc:` records.

#### Manual Verification:

- [x] Read `commandLink`, `dataLink`, `operatorLink` pages: the figure frames
      are `uavSystem`, `uavSystem`, `remoteControl`; ends are source first.
- [x] Read the `uav` page's representation order and the `wing` page's
      composite placeholder.
- [x] Confirm the only placeholders are on pages whose element owns no ports
      (list agreed).

---

## Phase 3: Regenerate `docs/example`

### Changes Required

#### 3.1 Generate the committed site

```sh
bin/sysml-publish -api local -model model -out docs/example
```

The marked folder is replaced; stale `Model/` pages disappear. Use defaults:
no `-config`, so the title stays `Model documentation` and no canonical links
are written.

#### 3.2 Verify the output

- `docs/example/index.html` `<title>Model documentation</title>`.
- No `rel="canonical"` in any page.
- `docs/example/assets/{asciidoctor.css,layout.css,sidebar.js,sort.js}` plus
  `mermaid.min.js` (new, because figures exist).
- No absolute paths (`grep -r '/home/' docs/example` empty).
- Re-running the command leaves `git diff --stat docs/example` empty.

### Success Criteria

#### Automated Verification:

- [x] Generation command exits 0.
- [x] `find docs/example -name index.html | wc -l` equals the Phase 1 page
      count plus the landing page (expected 44).
- [x] No `Model/` or `aircraft` path remains under `docs/example/`.
- [x] `git status --porcelain docs/example` shows only the expected
      deletions/additions.

#### Manual Verification:

- [x] Open `docs/example/index.html` from disk; the sidebar shows the new
      tree, the landing title is `Model documentation`.
- [x] Open the `uavSystem`, `uav`, `wing` and `commandLink` pages: figures,
      views, values and warnings render as in the scratch site.

---

## Phase 4: Rewrite `README.md`

### Overview

The README is the public explanation of the example. Update only the sections
that name or describe the model, plus the two corrections the example forces.

### Changes Required

#### 4.1 Quick start (lines 44–151)

- Replace "This one is a shortened `model/aircraft.sysml`" with a short
  explanation of the five files (one package each) and **three** snippets:
  1. `model/definitions.sysml`: imports, the `UavPart` definition with a
     shortened form of its representation (keep title, `sysml-doc`,
     properties table and one figure line), and `Battery` with its data.
  2. `model/uavkinds.sysml`: `UavKinds::uav` with mass/speed/endurance/range.
  3. `model/system.sysml`: `uavSystem` with `uav :> UavKinds::uav`,
     `remoteControl`, `commandLink`/`dataLink`/`operatorLink`, and the
     `systemInterfaces` view.
- State that the remaining files are `payloads.sysml` (generic and camera
  payload) and `links.sysml` (port definitions), and link to `model/`.
- Replace "You get six pages" with the real story: ~43 pages, one per
  documented element plus empty package ancestors; list representative pages
  (`UavPart`, `UavKinds::uav`, `uavSystem`, `uav`, `wing`, `battery`,
  `commandLink`). Explain that `uav` shows three representations (own,
  `UavKinds`, `UavPart`) and that a portless part's figure is a warning and a
  placeholder, not a failed run.
- Keep the `-api` instructions.

#### 4.2 Example output (lines 153–176)

- Regenerate links to: landing page, `UavDefinitions/UavPart`,
  `System/uavSystem`, `System/uavSystem/uav`,
  `System/uavSystem/uav/wing`, `System/uavSystem/uav/battery`,
  `System/uavSystem/commandLink`, `Payloads/CameraPayload`.
- Update the regeneration command to
  `go run ./cmd/sysml-publish -api local -model model -out docs/example`.

#### 4.3 The generated site (lines 271–300)

Replace the `Model/` tree with an abbreviated new tree (`UavDefinitions/`,
`UavKinds/`, `Payloads/`, `System/uavSystem/…`, `assets/` including
`mermaid.min.js`). Keep the path rules and the marker paragraph.

#### 4.4 Macros (lines 302–360)

- Update qualified-name examples to `System::uavSystem::uav::wing`.
- Update the `sysml-value` example to `System::uavSystem::uav::battery` and a
  feature chain `battery.voltage`.
- Update the target examples to `sysml-name:battery[]`,
  `sysml-doc::UavDefinitions::UavPart[]`, and the short-name example to
  `UavDefinitions::up`.
- Add a `sysml-fig::[<element>][]` row: a Mermaid figure of an interface (its
  ends' common ancestor as the frame) or of a part (its owned ports and the
  connections they are on); a part without ports fails with a placeholder.

#### 4.5 Configuration example (lines 414–423)

`title = "UAV docs"`, `baseURL = "https://example.com/uav/"` (the sample
values, matching `docs/sysml-publish.toml`).

#### 4.6 Log example (lines 455–464)

Capture real output from Phase 3 and show:

- `msg="model loaded" folder=./model files=5 elements=<actual> warnings=0`;
- one `page written` for `System::uavSystem::uav::wing`;
- the `sysml-value:size[]` composite warning with the real file:line;
- one representative `sysml-fig::[]: has no ports` warning with the real
  file:line;
- `msg="site published" pages=<actual> ...`.

Add one sentence: a portless element's figure leaves a placeholder and warns;
the run still succeeds.

#### 4.7 Limitations (line 511)

Replace "A failing macro stops the whole run; whether it should instead leave a
placeholder and warn is undecided." with "A failing macro leaves a placeholder
and a warning; the run carries on." (This is the existing MC-19 behaviour and
the new example relies on it.)

### Success Criteria

#### Automated Verification:

- [x] `grep -ri aircraft README.md` returns nothing.
- [x] Every inline link to `docs/example/…` resolves to an existing file.
- [x] The quick-start snippets are valid SysML excerpts: paste each into a
      temporary model with the other files and run the publisher (or check the
      declarations against `model/` mechanically).
- [x] README's page count, log line and config values match the generated site.

#### Manual Verification:

- [x] Read the quick start end to end: a newcomer can see the multi-file
      structure and the page result without opening `model/`.
- [x] The macros section's examples use names that exist in the new model.
- [x] No stale "six pages"/`Model::`/`aircraft` wording remains.

---

## Phase 5: `AGENTS.md`, `docs/sysml-publish.toml`, `CHANGELOG.md`

### Changes Required

#### 5.1 `AGENTS.md`

- §3 line 127: `docs/example/` — "generated site of the multi-file UAV model
  in `model/`, linked from README.md".
- §3 `model/` line stays "the user's model (default input)".
- §8 line 633–638: change "`model/aircraft.sysml`" to "the multi-file UAV model
  in `model/`"; update the smoke-copy sentence ("then `model/smoke.sysml`, now
  the multi-file UAV model").
- §8 regeneration command: add `-api local`:
  `bin/sysml-publish -api local -model model -out docs/example`.
- Do not touch §13's `model/` timing note or the Go-comment example in
  `internal/model`.

#### 5.2 `docs/sysml-publish.toml`

- `title = "UAV documentation"`.
- `baseURL = "https://example.com/uav/"`.
- Keep the comments and the six keys unchanged.

#### 5.3 `CHANGELOG.md`

Under `## Unreleased` → `### Changed`, add one bullet:

> The example model in `model/` is now a multi-file fixed-wing UAV: the whole
> system (`System::uavSystem`) contains the UAV and its ground remote control,
> the UAV subsets a general UAV (`UavKinds::uav`), and the model demonstrates
> inherited representations, interface figures, views and values across files.
> `docs/example/` is regenerated from it.

### Success Criteria

#### Automated Verification:

- [x] `grep -ri aircraft AGENTS.md docs/sysml-publish.toml CHANGELOG.md`
      returns nothing.
- [x] `AGENTS.md` and README name the same regeneration command.
- [x] `docs/sysml-publish.toml` still parses (`bin/sysml-publish -config
      docs/sysml-publish.toml -api local -model model -out /tmp/opencode/uav-config-site`
      exits 0; the title/baseURL take effect).

#### Manual Verification:

- [x] AGENTS §3 and §8 describe the multi-file model accurately.
- [x] The CHANGELOG bullet is in the Unreleased section and reads as a change,
      not a feature.

---

## Phase 6: Final verification

### Automated Verification:

- [x] `go build ./...` and `go vet ./...` pass.
- [x] `go test ./...` passes unchanged (no fixture edits); if no service can be
      obtained, confirm the skips are the only reason.
- [x] `bin/sysml-publish -api local -model model -out docs/example` exits 0 and
      is idempotent (`git diff --stat docs/example` empty after a second run).
- [x] `grep -ri aircraft` over `model/ docs/example/ README.md AGENTS.md
      docs/sysml-publish.toml CHANGELOG.md` returns nothing.
- [x] `git status --porcelain` shows only: deleted `model/aircraft.sysml`, five
      new model files, regenerated `docs/example/`, and the four edited docs
      files (plus the untracked `thoughts/`).
- [x] No `*.api.json`, `*~`, `site/` or scratch files are staged.

### Manual Verification:

- [x] The landing page and sidebar read as a UAV example, tops in the chosen
      order.
- [x] `uavSystem` → `uav` → parts and `remoteControl` branches are complete;
      every named part has a page.
- [x] Values displayed in `docs/example` match the research baseline:
      6 kg, 4.2 kg empty, 18 m/s, 90 min, 97200 m, 2.4 m span, 0.45 m²,
      12.8 aspect ratio, 177.6 W·h, 0.6 kW, 330 mm.
- [x] All three interface figures and the system diagram render; the antenna,
      receiver, operator, `remoteControl` and `payload` figures render.
- [x] The accepted placeholders are present only on portless pages and are
      explained in the README.

---

## Testing Strategy

There are no new automated tests: the model is documentation, and the existing
suite does not read `model/`. The regression risk is documentation drift, so
the verification is mostly scripted page existence and rendered-output checks:

- **Page existence**: a shell loop over the page list in Phase 1, run against
  the scratch site and again against `docs/example`.
- **Warning inventory**: capture stderr once, compare the set of
  `(page, macro)` warnings with the expected portless-page list plus `size`.
- **Idempotence**: regenerate twice; the second run produces no diff.
- **Docs consistency**: `grep -ri aircraft` in the six paths; README links
  checked with `test -f`; the two regeneration commands compared.
- **Model validity**: the scratch run is the integration test (real
  `sysml-grpc`, real rendering); a `sysml -validate model` run is optional but
  cheap.

Manual checks focus on what automation cannot judge: representation order,
figure frames, sidebar readability, and whether the README explains the model
truthfully.

## Performance Considerations

- The site grows from 7 pages to ~43 and 11 figures; rendering takes longer
  (the model loads once, each page converts its representations, and each
  figure queries connections). Expect a few seconds with a local service and
  `dot`; record the real duration in the README log only if it is stable.
- `mermaid.min.js` is written once and shared. No page caches converted HTML
  (MC-3), so the figure placeholders cost nothing beyond a model read each.
- No production code changes, so runtime and memory of the tool are unaffected
  apart from the larger model.

## Migration Notes

- `model/aircraft.sysml` is deleted, not kept; external readers following
  README links to the old pages get the new paths after regeneration.
- `docs/example` is replaced wholesale (the marker allows it); old `Model/`
  pages disappear from git.
- `testdata/smoke` remains the frozen copy; no fixture changes.
- No API, config-key or CLI changes.

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| A representation macro names an attribute the page element lacks | Put `range`/specific values only in the owning element's data rep; the shared template uses only `mass` and `power`, which every `UavPart` descendant has (verified: a shared `range` failed on `UavPart`). |
| Usage figures fail because ports are inherited | Declare ports on usages; only `Payload` owns a definition port, and its usage redefines it (verified). |
| The view misses connectors | Own all three interfaces in `uavSystem` (verified). |
| 28 figure placeholders look like breakage | Document the accepted placeholders in the README and the log example; the run still exits 0. |
| A package name collides with a library | `UavLinks`, not `Links`; the other four verified warning-free. |
| README/AGENTS drift | Scripted `grep -ri aircraft`, link existence and idempotence checks in Phase 6. |

## References

- Ticket: `thoughts/tickets/feature_uav_model.md`
- Research: `thoughts/research/2026-11-22_uav_attributes.md` (§5 baseline,
  §6 attributes, §7 metadata, §8 template, §9 structural constraints, §10
  views, §11 page tree, §12 ticket corrections)
- Patterns: `testdata/crossfile`, `testdata/inherit`, `testdata/figs`,
  `testdata/ifaces`, `testdata/views`, `testdata/metadata-table`,
  `testdata/values`
- Macro behaviour: `macros/ifaces.go:122-136` (owned ports),
  `macros/ifaces.go:216-241` (partner owner), `macros/fig.go:268-301`
  (part figure), `macros/fig.go:159-187` (interface frame)
- Docs to edit: `README.md`, `AGENTS.md` §3/§8,
  `docs/sysml-publish.toml`, `CHANGELOG.md`


## Deviations from Plan

### Phase 1: Model files
- **Original Plan**: `system.sysml` imports `UavKinds`, `Payloads` and `UavLinks`.
- **Actual Implementation**: only `ScalarValues`, `SI`, `UavDefinitions`, `Views` and
  `StandardViewDefinitions` are imported; `UavKinds::uav`, `Payloads::Payload` /
  `Payloads::DataPort` and `UavLinks::RfPort` / `ConsolePort` are referenced by
  qualified name.
- **Reason for Deviation**: importing `UavKinds::*` brings in a member named `uav`,
  which the local `part uav` shadows; the qualified references need no import and
  avoid the ambiguity.
- **Impact Assessment**: none; all references resolve and the site renders as planned.
- **Date/Time**: 2026-11-22.

### Phase 1 and 4: Avoid the literal word "aircraft"
- **Original Plan**: prose was free to use "aircraft" (the example is an aircraft).
- **Actual Implementation**: model docs and README use "UAV"/"air vehicle" throughout.
- **Reason for Deviation**: the ticket's and plan's automated criterion is
  `grep -ri aircraft` over `model/`, `docs/example/`, `README.md`, `AGENTS.md`,
  `docs/sysml-publish.toml` and `CHANGELOG.md`; any prose use would fail it.
- **Impact Assessment**: docs/example was regenerated after the wording change; the
  grep is now clean.
- **Date/Time**: 2026-11-22.

### Phase 1: Attribute names that avoid metamodel properties
- **Original Plan**: `role`/`configuration`/`kv` were already the plan's names.
- **Actual Implementation**: used `role`, `configuration`, `kv` (not `name`, `type`,
  `kV`) exactly as planned.
- **Reason for Deviation**: none from the plan; recorded because the research tables
  had used `name`/`type`/`kV`, and the implementation deliberately differs from them.
- **Impact Assessment**: the values render and no derived property is shadowed.
- **Date/Time**: 2026-11-22.

### Phase 2 and 4: Interface ends show short names
- **Original Plan**: illustrative ends were written as `remoteControl.operator.console`.
- **Actual Implementation**: `sysml-iface-ends::[]` renders `U1 . commandAntenna .rf`
  and `RC . commandReceiver .rf`, because `uav` and `remoteControl` carry short names
  and the macro's default is `name=short`.
- **Reason for Deviation**: macro behaviour, not a model change; the default is
  correct and demonstrates short names.
- **Impact Assessment**: none; ends are still source first.
- **Date/Time**: 2026-11-22.

### Phase 3 and 4: Log example uses the docs/example run
- **Original Plan**: the README log example would show `site/...` paths.
- **Actual Implementation**: it shows the real `docs/example` regeneration run
  (`folder=model`, `path=docs/example/...`, real durations), because that is the
  command the README documents and the site it commits.
- **Reason for Deviation**: truthfulness; the numbers and paths now match the
  committed output and the documented command.
- **Impact Assessment**: none; the model-loaded and published values are real.
- **Date/Time**: 2026-11-22.

### Phase 2: The parts table includes attributes
- **Original Plan**: `systemParts` was described as a parts table.
- **Actual Implementation**: `expose uavSystem::*` exposes every owned member, so the
  table also lists `serial` and `autonomous` (attributes) before the parts.
- **Reason for Deviation**: `asElementTable` renders all exposed members; the plan's
  shape was verified in `testdata/views` but its content was not enumerated.
- **Impact Assessment**: none; the table is still the verified table view and is
  labelled "Parts" on the page.
- **Date/Time**: 2026-11-22.

## Follow-up (2026-11-22): 50 kg UAV with CAN bus and power line

After the first implementation, the model was scaled up and extended without
changing the tool:

- **Scale**: 50 kg total mass, 42 kg empty, 8 kg payload, 5 m span, 1.5 m² wing,
  0.3 m mean chord, 25 m/s cruise, 90 min endurance, 135000 m range; a 12S 22 Ah
  pack, a 4 kW motor and a 610 mm propeller.
- **New definitions**: `PayloadRadio` (payload data, RF, CAN, power), `CanBus`,
  `PowerDistribution`, `Bec` and `MotorController`; ports on the existing
  definitions.
- **New links**: 16 interfaces owned by `uav` — nine power, five CAN, the payload
  data link and the payload RF link. The CAN bus is a hub part with five ports,
  because SysML `connect` is binary.
- **New view**: `avionicsInterfaces` inside `uav`, shown by
  `sysml-view::avionicsInterfaces[]` on the UAV page.
- **Result**: 53 pages, 15 warnings (14 figure placeholders plus the composite
  `size`), `docs/example` regenerated and idempotent. `README.md` and
  `CHANGELOG.md` updated; `AGENTS.md` and `docs/sysml-publish.toml` needed no
  further change.
