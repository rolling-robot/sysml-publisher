---
date: 2026-11-22T00:18:02+03:00
git_commit: ba892c2a4a82bb30d995246f2a1c475ef1a149ac
branch: master
repository: sysml-publisher
topic: "Essential attributes and approximate values for the multi-file UAV example model"
tags: [research, codebase, model, uav, attributes, units, sysml-value, sysml-metadata-table, sysml-ifaces, sysml-fig, sysml-view]
last_updated: 2026-11-22
---

## Ticket Synopsis

`thoughts/tickets/feature_uav_model.md` (FEATURE-001, `status: created`) replaces
`model/aircraft.sysml` with a flat multi-file fixed-wing UAV model: one top-level
package per file, a whole system part `uavSystem` containing the UAV and its ground
remote control, the UAV subsetting a more general UAV in another package, a generic
payload plus a documented `CameraPayload`, two antennas and two RF interfaces, a human
operator, battery/electric propulsion, and `sysml-ifaces`/`sysml-fig` demos on the
documented pages. It also updates `README.md`, `AGENTS.md`, `docs/sysml-publish.toml`,
`CHANGELOG.md`, and regenerates `docs/example/`.

This research answers the open part of the ticket: **which attributes each part should
carry, what types and unit spellings they should use, and what approximate values are
credible** for a small fixed-wing UAV. Being wrong by a factor of five is acceptable;
the model exists to showcase the tool on a realistic structure, not to fly.

## Summary

The model should be built around one shared definition, `UavDefinitions::UavPart`,
carrying `mass` and `power` and the inherited AsciiDoc representation. Every part and
component definition specializes it, so the template (name, doc, metadata table,
mass/power rows, members list, interfaces list) appears on every page automatically,
and `sysml-value` is demonstrated on values of every shape the macro supports.

The research produced a **validated attribute set** (all declarations below were loaded
with the pinned `sysml` CLI and evaluated), a **validated unit vocabulary** (what
OpenSysML accepts and what it rejects), and **four structural findings that contradict
the ticket**:

1. `Links` is the name of a Kernel Semantic Library package; a root namespace named
   `Links` produces a load warning. Rename it (`UavLinks`).
2. A `sysml-view` interconnection diagram draws a connector only when the interface is
   **owned by the exposed element** (or inside its subtree). Interfaces in a separate
   `UavLinks` package are invisible to the system view, even with a second `expose`.
   The two RF link usages must be members of `uavSystem`.
3. `sysml-ifaces` lists only the target's **owned** ports, and names the **immediate
   owner** of the other end's leaf feature. `sysml-ifaces::uav[]` and
   `::remoteControl[]` therefore list nothing unless those parts own ports themselves;
   the lists belong on the antenna/receiver/operator pages.
4. `sysml-fig` of a part fails with `has no ports; a figure needs at least one`.
   `sysml-fig::uav[]` cannot render. `::remoteControl[]` renders only if
   `remoteControl` owns a `console` port (which the operator link needs anyway).

A coherent design baseline (6 kg MTOW, 2.4 m span, 18 m/s cruise, 90 min endurance,
6S 8 Ah battery, 600 W motor) makes the computed values land on round, plausible
numbers: range 97.2 km, wing aspect ratio 12.8, mean chord 187.5 mm, battery specific
energy 209 W·h/kg.

## Detailed Findings

### 1. How an attribute becomes page text

`sysml-value:<feature>[]` evaluates the named feature with the page's element as the
subject ([`macros/value.go:17-25`](../../macros/value.go)) and formats the result:

- a quantity keeps the **literal unit as written** (`0.75 kW`, `22.2 V`, `5 A*h`,
  `111 W*h`, `45 min`, `65 km/h`, `84 °`, `5000 B/s`);
- a **computed** quantity is evaluated and converted to SI base (`97200 m` for
  `endurance * cruiseSpeed`);
- a derived unit from the library is written with the package and quotes stripped by
  `formatUnit` (`SI::'m²'` → `m²`, `SI::'N⋅m'` → `N⋅m`, `SI::'kg⋅m⁻³'` → `kg⋅m⁻³`);
- a plain `Real`, `Integer`, `Boolean` or `String` is written as is (`12.8`, `480`,
  `true`, `carbon fiber`);
- an enumeration literal is written qualified (`Chemistry::LiPo`);
- a composite value (an attribute with sub-attributes) is shown as `[composite]` and
  the macro warns: `sysml-value:size[]: size is composite; shown as [composite]`;
- a feature with no value is shown as `(no value)` (`macros.NoValue`,
  [`macros/metadata_table.go:29`](../../macros/metadata_table.go)).

`sysml-metadata-table::PartMetadata[]` shows one row per attribute of the metadata def
and its supertypes, with the element's bound value, else the def's default or fixed
value, else `(no value)` ([`requirements.org:279-299`](../../requirements.org)).
A **fixed** value (`attribute process : String = "TBD";`) cannot be bound by a usage
and is always shown as written.

### 2. Declaration rules that were validated

All of the following were loaded with `sysml -validate` and evaluated with
`sysml -e` against the pinned checkout (`../sysmlv2/OpenSysML`).

| Rule | Evidence |
|---|---|
| `part uav :> Kinds::uav` (qualified subsetting of a **usage**) parses and inherits attributes, owned parts and representations | `uav.mass` = `6 kg`, `uav.wing.mass` = `0.7 kg` |
| A redefinition needs the base written with `default`; `=` is fixed for every redefinition | `cannot override the binding value of Defs::Uav2::mass: a value written with '=' is fixed …` |
| Every file needs its own `private import ScalarValues::*;` and `private import SI::*;` for `[kg]` and friends | `unresolved reference: kg — did you mean SI::kg?` |
| A part whose name is a unit symbol shadows the unit and breaks evaluation | `part m` made `[m/s]` read as the part `m`: `not a measurement unit: m resolves to the partUsage m … write SI::m` |
| A model-defined unit works across packages when it is public | `attribute <MHz> …` in `Defs`, used in `Use` → `2400 MHz` |
| A computed quantity attribute is fine | `attribute range : ISQ::LengthValue = endurance * cruiseSpeed;` → `97200 m` |
| A computed dimensionless `Real` is fine | `attribute aspectRatio : Real = span * span / area;` → `12.8` |
| Division of quantities is fine | `attribute meanChord : ISQ::LengthValue = area / span;` → `0.1875 m` |
| A model-defined unit written as `= 1000000 * Hz` does **not** evaluate | `not a measurement unit expression: *ast.LiteralInteger`; use `ConversionByPrefix` |

**Naming rule for the model:** never name a part `m`, `s`, `A`, `h`, `W`, `V`, `N`,
`J`, `K`, `g`, `L`, `B`, `Hz` or `min`. `motor` is fine; `m` is not. The ticket's
part names (`motor`, `wing`, `battery`, …) are all safe.

### 3. Unit vocabulary

Loaded in one model; the accepted ones were then evaluated through `sysml-value`.

| Accepted | Shown as | Rejected | Note |
|---|---|---|---|
| `[kg]` | `5 kg` | `[Wh]`, `[Ah]` | write `[W*h]`, `[A*h]` |
| `[W]`, `[kW]` | `0.75 kW` | `[ms]` | write `[s]` or a fraction of `[h]` |
| `[V]` | `22.2 V` | `[MHz]`, `[kHz]`, `[GHz]` | see the local-unit recipe below |
| `[A*h]` | `5 A*h` | `[MB/s]`, `[kB/s]` | write `[B/s]` or `[bit/s]` |
| `[W*h]` | `111 W*h` | `[dBi]`, `[dBm]` | `[dB]` exists; gains as plain `Real` |
| `[min]`, `[h]` | `45 min` | `[%]` | not defined, even quoted `['%']` |
| `[m]`, `[mm]`, `[km]` | `2.4 m`, `330 mm` | `[rpm]` | motor speed as plain `Integer` |
| `[m^2]` | `0.42 m²` | | computed/derived form |
| `[m/s]`, `[km/h]` | `65 km/h` | | |
| `[N]`, `[N*m]` | `12 N`, `2.5 N⋅m` | | |
| `[Hz]` | `2400000000 Hz` | | or a local `MHz` |
| `[B/s]`, `[bit/s]` | `5000 B/s` | | `TransferRateValue` |
| `[dB]` | | | plain `Real` is simpler |
| `['°']` | `84 °` | `[°]` | must be quoted |
| `[L]` | `2 L` | | |
| `[kg/m^3]` | `1.2 kg⋅m⁻³` | | |
| `[m/s^2]`, `[C]`, `[J]`, `[kJ]` | | | |
| `[W/kg]`, `[W*h/kg]` | `209 W*h/kg` | | |

**Local megahertz** (validated across packages): put this in `definitions.sysml`, and
every `[MHz]` in the model works and shows as `2400 MHz`.

```sysml
private import MeasurementReferences::*;
private import SIPrefixes::*;

attribute <MHz> megahertz : FrequencyUnit {
    :>> unitConversion: ConversionByPrefix {
        :>> prefix = mega;
        :>> referenceUnit = Hz;
    }
}
```

Fallback if the extra imports are unwanted: `attribute frequency : ISQ::FrequencyValue
default 2400000000 [Hz];` shows `2400000000 Hz` (correct, ugly).

### 4. Value-shape coverage

The model should demonstrate every shape `sysml-value` and `sysml-metadata-table`
handle. Recommended coverage:

| Shape | Where |
|---|---|
| literal quantity | `mass`, `voltage`, `capacity`, `energy`, `span`, `area`, `diameter`, `volume` |
| computed quantity | `range` (uav), `missionRange` (system), `meanChord` (wing) |
| derived unit formatted | `area` (`m²`), `torque` (`N⋅m`), `specificEnergy` (`W*h/kg`), `density` (optional) |
| dimensionless `Real` | `aspectRatio`, `efficiency`, `gain`, `sensitivity` |
| `Integer` | `cells`, `kV`, `wheels`, `blades`, `constellations`, `channels`, `resolution` |
| `Boolean` | `rechargeable`, `retractable`, `calibrated`, `open`, `flightReady`, `autonomous` |
| `String` | `material`, `airfoil`, `processor`, `serial`, `deployment` |
| `String` with no value | `firmware` → `(no value)` |
| enumeration | `chemistry` (`Chemistry::LiPo`), `type` (`TailType::conventional`) |
| composite | `size` on the wing → `[composite]` + the warning |
| fixed metadata value | `PartMetadata::process` → always `TBD`, never bound |
| metadata bound/default/absent | `uavSystem`, `battery`, `motor`, `payload`, `commandAntenna`, and the parts without metadata |

### 5. Design baseline

One coherent set of numbers; the parts below add up to it.

| Quantity | Value | Sanity |
|---|---|---|
| MTOW (`uav.mass`) | 6.0 kg | small fixed-wing sUAS class |
| empty mass | 4.2 kg | sum of the listed parts ≈ 4.2 kg |
| payload capacity | 1.0 kg | 17 % of MTOW |
| actual payload | 0.4 kg | generic `Payload` usage |
| span / area | 2.4 m / 0.45 m² | AR 12.8, mean chord 187.5 mm, wing loading 13.3 kg/m² |
| cruise speed | 18 m/s (64.8 km/h) | |
| endurance | 90 min | |
| range | 97.2 km | `90 min × 18 m/s`, computed |
| battery | 6S LiPo, 8 A·h, 22.2 V, 177.6 W·h, 0.85 kg | 209 W·h/kg |
| motor | 600 W, 0.72 N·m, kV 480, 85 % | 600 W at ~8000 rpm |
| propeller | 330 × 203 mm (13×8 in), 2 blades | 12 N static thrust |
| control / data | 2.4 GHz / 5.8 GHz | gains 3 / 5 |
| ground station | 16 channels, 8 h, 10 km | |
| operator | 80 kg, 120 h | |

Mass budget: fuselage 1.10 + wing 0.95 + tail 0.35 + landing gear 0.25 + motor 0.28 +
propeller 0.06 + battery 0.85 + flight controller 0.04 + IMU 0.01 + GPS 0.03 +
payload bay 0.20 + antennas 0.05 = **4.17 kg** empty; + payload 0.40 = 4.57 kg flying,
1.43 kg below MTOW. Coherent.

### 6. Recommended attributes per part

Types are `ISQ::…Value` from `SI::*`; every file repeats
`private import ScalarValues::*; private import SI::*;`.

#### `UavDefinitions::UavPart` (shared base, documented)

| Attribute | Type | Value | `sysml-value` |
|---|---|---|---|
| `mass` | `ISQ::MassValue` | `default 0.1 [kg]` | `0.1 kg` |
| `power` | `ISQ::PowerValue` | `default 0 [W]` | `0 W` |

Its representation is the template every part inherits (see §8).

#### `UavKinds::uav` (general UAV, `part uav : UavDefinitions::UavPart`)

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `mass` | `ISQ::MassValue` | `default 6 [kg]` | `6 kg` |
| `cruiseSpeed` | `ISQ::SpeedValue` | `default 18 [m/s]` | `18 m/s` |
| `endurance` | `ISQ::DurationValue` | `default 90 [min]` | `90 min` |
| `range` | `ISQ::LengthValue` | `= endurance * cruiseSpeed` | `97200 m` |
| `emptyMass` | `ISQ::MassValue` | `default 4.2 [kg]` | `4.2 kg` |
| `payloadCapacity` | `ISQ::MassValue` | `default 1 [kg]` | `1 kg` |

#### `System::uavSystem`

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `missionRange` | `ISQ::LengthValue` | `= uav.endurance * uav.cruiseSpeed` | `97200 m` |
| `operatorCount` | `Integer` | `default 1` | `1` |
| `deployment` | `String` | `default "field"` | `field` |
| `flightReady` | `Boolean` | `default true` | `true` |

#### `System::uavSystem::uav` (`part uav :> UavKinds::uav`)

Inherits all of `UavKinds::uav`; adds:

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `serial` | `String` | `default "UAV-001"` | `UAV-001` |
| `autonomous` | `Boolean` | `default true` | `true` |

#### `fuselage` (`part fuselage : UavPart`)

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `mass` | `ISQ::MassValue` | `default 1.1 [kg]` | `1.1 kg` |
| `length` | `ISQ::LengthValue` | `default 1.15 [m]` | `1.15 m` |
| `material` | `String` | `default "carbon fiber"` | `carbon fiber` |
| `watertight` | `Boolean` | `default false` | `false` |

#### `wing` (`part wing : UavPart`)

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `mass` | `ISQ::MassValue` | `default 0.95 [kg]` | `0.95 kg` |
| `span` | `ISQ::LengthValue` | `default 2.4 [m]` | `2.4 m` |
| `area` | `ISQ::AreaValue` | `default 0.45 [m^2]` | `0.45 m²` |
| `meanChord` | `ISQ::LengthValue` | `= area / span` | `0.1875 m` |
| `aspectRatio` | `Real` | `= span * span / area` | `12.8` |
| `airfoil` | `String` | `default "NACA 2412"` | `NACA 2412` |
| `size` | `Size` | no value | `[composite]` + warning |

`wing.left` / `wing.right` (bare parts, documented):

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `mass` | `ISQ::MassValue` | `default 0.45 [kg]` | `0.45 kg` |
| `tipDevice` | `String` | `default "none"` | `none` |

#### `tail` (`part tail : UavPart`)

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `mass` | `ISQ::MassValue` | `default 0.35 [kg]` | `0.35 kg` |
| `type` | `TailType` | `default TailType::conventional` | `TailType::conventional` |
| `area` | `ISQ::AreaValue` | `default 0.12 [m^2]` | `0.12 m²` |
| `incidence` | `ISQ::AngularMeasureValue` | `default 0 ['°']` | `0 °` |

#### `landingGear` (`part landingGear : UavPart`)

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `mass` | `ISQ::MassValue` | `default 0.25 [kg]` | `0.25 kg` |
| `wheels` | `Integer` | `default 3` | `3` |
| `retractable` | `Boolean` | `default false` | `false` |
| `track` | `ISQ::LengthValue` | `default 0.6 [m]` | `0.6 m` |

#### `motor` (`part motor : Motor`)

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `mass` | `ISQ::MassValue` | `default 0.28 [kg]` | `0.28 kg` |
| `power` | `ISQ::PowerValue` | `default 0.6 [kW]` | `0.6 kW` |
| `torque` | `ISQ::TorqueValue` | `default 0.72 [N*m]` | `0.72 N⋅m` |
| `kV` | `Integer` | `default 480` | `480` |
| `efficiency` | `Real` | `default 0.85` | `0.85` |

#### `propeller` (`part propeller : Propeller`)

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `mass` | `ISQ::MassValue` | `default 0.06 [kg]` | `0.06 kg` |
| `diameter` | `ISQ::LengthValue` | `default 330 [mm]` | `330 mm` |
| `pitch` | `ISQ::LengthValue` | `default 203 [mm]` | `203 mm` |
| `staticThrust` | `ISQ::ForceValue` | `default 12 [N]` | `12 N` |
| `blades` | `Integer` | `default 2` | `2` |

#### `battery` (`part battery : Battery`)

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `mass` | `ISQ::MassValue` | `default 0.85 [kg]` | `0.85 kg` |
| `capacity` | `ISQ::ElectricChargeValue` | `default 8 [A*h]` | `8 A*h` |
| `voltage` | `ISQ::ElectricPotentialValue` | `default 22.2 [V]` | `22.2 V` |
| `energy` | `ISQ::EnergyValue` | `default 177.6 [W*h]` | `177.6 W*h` |
| `maxCurrent` | `ISQ::ElectricCurrentValue` | `default 60 [A]` | `60 A` |
| `specificEnergy` | `ISQ::SpecificEnergyValue` | `default 209 [W*h/kg]` | `209 W*h/kg` |
| `cells` | `Integer` | `default 6` | `6` |
| `chemistry` | `Chemistry` | `default Chemistry::LiPo` | `Chemistry::LiPo` |
| `rechargeable` | `Boolean` | `default true` | `true` |

#### `flightController` (`part flightController : FlightController`)

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `mass` | `ISQ::MassValue` | `default 0.04 [kg]` | `0.04 kg` |
| `power` | `ISQ::PowerValue` | `default 1.5 [W]` | `1.5 W` |
| `updateRate` | `ISQ::FrequencyValue` | `default 400 [Hz]` | `400 Hz` |
| `processor` | `String` | `default "STM32H7"` | `STM32H7` |
| `firmware` | `String` | no value | `(no value)` |

#### `imu` (`part imu : Imu`)

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `mass` | `ISQ::MassValue` | `default 0.01 [kg]` | `0.01 kg` |
| `power` | `ISQ::PowerValue` | `default 0.05 [W]` | `0.05 W` |
| `sampleRate` | `ISQ::FrequencyValue` | `default 1000 [Hz]` | `1000 Hz` |
| `gyroRange` | `Real` | `default 2000` | `2000` |
| `calibrated` | `Boolean` | `default false` | `false` |

#### `gps` (`part gps : Gps`)

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `mass` | `ISQ::MassValue` | `default 0.03 [kg]` | `0.03 kg` |
| `power` | `ISQ::PowerValue` | `default 0.3 [W]` | `0.3 W` |
| `updateRate` | `ISQ::FrequencyValue` | `default 10 [Hz]` | `10 Hz` |
| `accuracy` | `ISQ::LengthValue` | `default 2.5 [m]` | `2.5 m` |
| `constellations` | `Integer` | `default 4` | `4` |

#### `payloadBay` (`part payloadBay : UavPart`)

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `mass` | `ISQ::MassValue` | `default 0.2 [kg]` | `0.2 kg` |
| `volume` | `ISQ::VolumeValue` | `default 2 [L]` | `2 L` |
| `maxPayload` | `ISQ::MassValue` | `default 1.2 [kg]` | `1.2 kg` |
| `open` | `Boolean` | `default false` | `false` |

Owns `part payload : Payloads::Payload` with `port data : Payloads::DataPort;`.

#### `Payloads::Payload` (documented def)

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `mass` | `ISQ::MassValue` | `default 0.3 [kg]` | `0.3 kg` |
| `power` | `ISQ::PowerValue` | `default 5 [W]` | `5 W` |
| `dataRate` | `ISQ::TransferRateValue` | `default 5000 [B/s]` | `5000 B/s` |
| `port data` | `DataPort` | | (figure only) |

#### `Payloads::CameraPayload :> Payload` (documented def)

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `mass` | `ISQ::MassValue` | `default 0.45 [kg]` | `0.45 kg` (override) |
| `power` | `ISQ::PowerValue` | `default 7 [W]` | `7 W` (override) |
| `resolution` | `Integer` | `default 12` | `12` |
| `fieldOfView` | `ISQ::AngularMeasureValue` | `default 84 ['°']` | `84 °` |

#### `commandAntenna` / `dataAntenna` (`: Antenna`)

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `mass` | `ISQ::MassValue` | `0.02` / `0.03 [kg]` | `0.02 kg` / `0.03 kg` |
| `frequency` | `ISQ::FrequencyValue` | `2400` / `5800 [MHz]` | `2400 MHz` / `5800 MHz` |
| `gain` | `Real` | `3` / `5` | `3` / `5` |
| `connector` | `String` | `"SMA"` | `SMA` |
| `port rf` | `RfPort` | | (figure only) |

`dataAntenna` may instead carry `polarization : String default "linear";`.

#### `remoteControl` (bare part, `port console : ConsolePort`)

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `mass` | `ISQ::MassValue` | `default 1.2 [kg]` | `1.2 kg` |
| `channels` | `Integer` | `default 16` | `16` |
| `batteryLife` | `ISQ::DurationValue` | `default 8 [h]` | `8 h` |
| `range` | `ISQ::LengthValue` | `default 10 [km]` | `10 km` |
| `screen` | `Boolean` | `default true` | `true` |

#### `commandReceiver` / `dataReceiver` (`: CommandReceiver` / `: DataReceiver`)

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `mass` | `ISQ::MassValue` | `0.05` / `0.06 [kg]` | `0.05 kg` / `0.06 kg` |
| `power` | `ISQ::PowerValue` | `0.5` / `1.2 [W]` | `0.5 W` / `1.2 W` |
| `frequency` | `ISQ::FrequencyValue` | `2400` / `5800 [MHz]` | `2400 MHz` / `5800 MHz` |
| `sensitivity` | `Real` | `-100` | `-100` |
| `dataRate` | `ISQ::TransferRateValue` | `250000 [B/s]` (data) | `250000 B/s` |
| `port rf` | `RfPort` | | (figure only) |

#### `operator` (`: Operator`, `port console : ConsolePort`)

| Attribute | Type | Value | Shows |
|---|---|---|---|
| `mass` | `ISQ::MassValue` | `default 80 [kg]` | `80 kg` |
| `name` | `String` | `default "pilot"` | `pilot` |
| `certified` | `Boolean` | `default true` | `true` |
| `flightHours` | `ISQ::DurationValue` | `default 120 [h]` | `120 h` |

#### Enumerations (`UavDefinitions`)

```sysml
enum def Chemistry { LiPo; LiIon; }
enum def TailType { conventional; vTail; flyingWing; }
```

### 7. Metadata

Keep `PartMetadata` exactly as it is, including the **fixed** `process` value:

```sysml
metadata def PartMetadata {
    attribute owner : String default "nobody";
    attribute revision : Integer default 0;
    attribute reviewed : Boolean default false;
    attribute process : String = "TBD";   // fixed: never bind it
    attribute note : String;
}
```

Recommended bindings, chosen to show bound / default / absent / no-value:

| Element | Binding |
|---|---|
| `uavSystem` | `@PartMetadata { owner = "pete"; revision = 4; reviewed = true; note = "flight-ready demonstrator"; }` |
| `battery` | `@PartMetadata { owner = "pete"; revision = 2; reviewed = true; }` |
| `motor` | `@PartMetadata { revision = 1; }` |
| `payload` | `@PartMetadata { owner = "camera team"; }` |
| `commandAntenna` | `@PartMetadata { note = "spare"; }` |
| most other parts | no metadata → `sysml-metadata-table` shows nothing |

`process` always reads `TBD`; binding it would be a redefinition of a fixed value and
would fail validation.

### 8. The inherited representation

Put the template in `UavPart`'s representation so every page gets it (PG-7):

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
```

`sysml-ifaces::[]` is safe everywhere: with no owned ports it drops its line silently
([`macros/ifaces.go:60-70`](../../macros/ifaces.go)). `sysml-metadata-table` shows
nothing without metadata. `sysml-members-list` shows nothing on a leaf.

**Do not put `sysml-fig::[]` in the shared template.** A part figure fails on any
element with no owned ports (`macros/fig.go:295-301`), which is most of the model
(fuselage, wing, battery, all definitions). The shared template would turn ~22 pages
into failure placeholders and add ~22 warnings. Instead put `sysml-fig::[]` in the
**own** representation of the elements that can render one:

| Page | Own representation |
|---|---|
| `commandAntenna`, `dataAntenna` | `/* sysml-fig::[] */` |
| `commandReceiver`, `dataReceiver` | `/* sysml-fig::[] */` |
| `operator`, `remoteControl` | `/* sysml-fig::[] */` |
| `payload` | `/* sysml-fig::[] */` (one port, no edges) |
| `commandLink`, `dataLink`, `operatorLink` | `/* = sysml-name:[] … sysml-iface-ends::[] … sysml-fig::[] */` |

An own representation without a header (`/* sysml-fig::[] */`) leaves the page title to
the inherited representation, so the title is not duplicated.

If the "figure on every page" reading of the ticket is preferred over a clean example,
put `sysml-fig::[]` in the shared template and accept the placeholders and warnings;
the trade-off is quantified above.

### 9. Structural constraints that change the ticket's layout

#### 9.1 `Links` is a library package name

Loading a root namespace `Links` warns:

```
warning: Root namespace `Links` has the name of a standard library package,
so qualified names starting with it may resolve to the library
```

`Links.kerml` is in the Kernel Semantic Library; `Connections` is also taken. Rename
the package to `UavLinks` (or `Comms`).

#### 9.2 The system view only draws owned connectors

Measured on the same model with the interface in different places (SVG path count on
the `uavSystem` page):

| Interface owner | Paths | Connector drawn |
|---|---|---|
| `UavLinks` package | 2 | no (only the two port stubs) |
| `UavLinks`, view exposes both packages | 2 | no |
| `uavSystem` (member) | 3 | **yes** (a long path between the parts) |

The rendered interface figure confirms the same structure works for `sysml-fig`:
`sysml-fig::commandLink[]` produced a Mermaid figure framed at `uavSystem` with
`uav > commandAntenna > rf` and `remoteControl > commandReceiver > rf`, edge `n3 === n6`.

**Recommendation:** declare the three interface usages as members of `uavSystem`;
keep `UavLinks` for the port definitions (`RfPort`, `ConsolePort`) and any shared
connectivity docs. This also makes `sysml-members-list::InterfaceUsage[]` on
`uavSystem`'s page non-empty (with the links in a separate package it is empty, because
they are not owned members).

#### 9.3 `sysml-ifaces` and `sysml-fig` targets

- `sysml-ifaces` uses the target's **owned** ports only
  ([`macros/ifaces.go:122-136`](../../macros/ifaces.go)), and names the **immediate
  owner** of the other end's leaf feature ([`macros/ifaces.go:216-241`](../../macros/ifaces.go)).
  So:
  - `sysml-ifaces::[]` on `commandAntenna` → `commandLink -> commandReceiver`
    (not `remoteControl`);
  - on `commandReceiver` → `commandLink -> commandAntenna`;
  - on `operator` → `operatorLink -> remoteControl`;
  - on `uav` or `remoteControl` → nothing (they own no ports, unless `remoteControl`
    owns `console`).
- `sysml-fig` of a part needs at least one owned port
  ([`macros/fig.go:295-301`](../../macros/fig.go)); of an interface, at least two
  ends and a common ancestor ([`macros/fig.go:159-187`](../../macros/fig.go)).
  So:
  - `sysml-fig::uav[]` **fails**: `has no ports; a figure needs at least one`
    (confirmed on a real run);
  - `sysml-fig::remoteControl[]` renders once `remoteControl` owns `console`;
  - `sysml-fig::commandLink[]` / `::dataLink[]` render (frame `uavSystem`);
  - `sysml-fig::operatorLink[]` renders (frame `remoteControl`).

The ticket's manual check "`sysml-fig::uav[]` and `::remoteControl[]` render" is only
half true and must be corrected.

### 10. Views

Use the verified shapes:

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

`asElementTable` is the table renderer used by `testdata/views` (a table needs no
`dot`). The system page shows `sysml-view::systemInterfaces[]` and, optionally,
`.Parts` + `sysml-view::systemParts[]`. Do not `expose uavSystem.uav` (a feature
chain); the ticket's risk 5 stands, and the figures cover the UAV page.

### 11. Page tree and sidebar order

Tops follow **sorted filenames**, then declaration order. With the semantic names the
order is `UavDefinitions`, `UavLinks`, `Payloads`, `System`, `UavKinds`. `UavLinks`
gets no page (it owns only port definitions), so the visible tops are
`UavDefinitions`, `Payloads`, `System`, `UavKinds`. If a different order is wanted,
prefix the filenames (`10-definitions.sysml`, `20-uavkinds.sysml`,
`30-payloads.sysml`, `40-links.sysml`, `50-system.sysml`) — page paths follow
containment, not filenames (PT-1), so prefixes are safe.

Expected site tree (≈42 pages, including empty package ancestors):

```
UavDefinitions
  UavPart
  Battery  Motor  Propeller  FlightController  Imu  Gps
  Antenna  CommandReceiver  DataReceiver  Operator
UavKinds
  uav
Payloads
  Payload
  CameraPayload
System
  uavSystem
    uav
      fuselage  wing (left, right)  tail  landingGear
      motor  propeller  battery  flightController  imu  gps
      payloadBay (payload)
      commandAntenna  dataAntenna
    remoteControl
      commandReceiver  dataReceiver  operator
    commandLink  dataLink  operatorLink
```

Every component definition is a page because it specializes `UavPart` and inherits its
representation. That is the cost of the shared template; the pages are small and
non-empty, and the tree is shallow.

### 12. Ticket corrections

| Ticket says | Research finds |
|---|---|
| `capacity = 5000 [W*h]` | 5 kW·h is implausible for this class; use `8 [A*h]` + `177.6 [W*h]` |
| `Links` package | collides with the Kernel library; rename `UavLinks` |
| links in `Links`, "ownership does not affect the figure" | true for `sysml-fig`, **false** for `sysml-view`; own the interfaces in `uavSystem` |
| `sysml-ifaces::uav[]` / `::remoteControl[]` list the two links | they list nothing (no owned ports) |
| `sysml-fig::uav[]` renders | it fails (`has no ports`) |
| `sysml-members-list::InterfaceUsage[]` on `uav` | empty unless `uav` owns the links; put it on `uavSystem` |
| `Payload`/`CameraPayload` in both `definitions.sysml` and `payloads.sysml` | one definition only, in `payloads.sysml` |
| sidebar order as listed | sorted filenames; prefix if order matters |
| "no warnings besides `size`" with fig on all pages | fig failures add one warning per portless page |
| ~25 pages | ~42 with every component definition documented |

## Code References

- [`macros/value.go:17-25`](../../macros/value.go) — `sysml-value` target rules and evaluation subject.
- [`macros/value.go`](../../macros/value.go) — `formatValue`/`formatUnit`, `[composite]`, `(no value)`.
- [`macros/metadata_table.go:29`](../../macros/metadata_table.go) — `NoValue = "(no value)"`.
- [`macros/metadata_table.go`](../../macros/metadata_table.go) — metadata def/attribute/value resolution.
- [`macros/ifaces.go:60-70`](../../macros/ifaces.go) — no connections → macro line dropped.
- [`macros/ifaces.go:122-136`](../../macros/ifaces.go) — `portsOf`: owned `PortUsage` members only.
- [`macros/ifaces.go:216-241`](../../macros/ifaces.go) — `partnersOf`: immediate owner of the other end's leaf.
- [`macros/fig.go:159-187`](../../macros/fig.go) — interface figure, ends' deepest common ancestor.
- [`macros/fig.go:268-301`](../../macros/fig.go) — `partFigure`, `has no ports; a figure needs at least one`.
- [`testdata/values/values.sysml`](../../testdata/values/values.sysml) — canonical value shapes.
- [`testdata/metadata-table/metadata-table.sysml`](../../testdata/metadata-table/metadata-table.sysml) — metadata shapes.
- [`testdata/views/views.sysml:38-43`](../../testdata/views/views.sysml) — `render asElementTable`, `expose sys::*`.
- [`testdata/ifaces/ifaces.sysml`](../../testdata/ifaces/ifaces.sysml) — `sysml-ifaces` semantics.
- [`testdata/figs/figs.sysml`](../../testdata/figs/figs.sysml) — `sysml-fig` semantics.
- [`requirements.org:279-306`](../../requirements.org) — MC-13, MC-14.
- [`AGENTS.md`](../../AGENTS.md) §5.4 — `nameIndex` resolution order; §9 — OpenSysML quirks.

## Architecture Insights

- **The model is a rendering fixture.** Every choice should exercise a macro or a page
  rule. Attributes are not there for engineering accuracy; they are there to produce
  every value shape, every link form, and every failure mode the tool documents.
- **One shared definition carries the template.** `UavPart`'s representation is the
  only place the page layout is written; per-part attributes are data. This is the
  same pattern the current `AircraftPart` uses, extended to ~20 parts.
- **Ownership decides what a view sees; the macro index does not.** `sysml-fig` and
  `sysml-ifaces` query the whole model (`connIndex`), so a link may live anywhere.
  `sysml-view` walks the exposed element's structure, so a link must be owned inside
  it. The model must be shaped for the view, not only for the macros.
- **Literal units are preserved, computed units are SI.** This is the single most
  visible formatting rule; choose literals for readability (`0.6 kW`, `330 mm`,
  `2400 MHz`) and accept SI base for computed values (`97200 m`, `0.1875 m`).
- **Names are part of the model's correctness.** A part named like a unit silently
  changes evaluation; a package named like a library package warns. Keep part names
  word-like and package names project-specific.

## Historical Context (from `thoughts/`)

`thoughts/` currently holds only the ticket; there is no prior research or plan
directory.

- `thoughts/tickets/feature_uav_model.md` — FEATURE-001, the source ticket. Its
  "Research Context" and "Notes" list the risks this document resolves (qualified
  subsetting, unit spellings, definition-page macros, view `expose`, sidebar order).

## Related Research

None yet. This is the first document under `thoughts/research/`.

## Open Questions

1. **Figure on every page or only where it renders?** The ticket asks for
   `sysml-fig` on all documented pages and accepts definition-page failures; this
   research recommends the 10 pages that can render one, to keep `docs/example` and
   the README log clean. Confirm before implementation.
2. **Page count.** ~42 pages with every component definition documented. If that is
   too many for the README example, the alternative is to type usages by two
   definitions (`part battery : Battery, UavPart;`) so component defs get no page —
   at the cost of duplicating `mass`/`power` in each def. Not recommended, but
   possible.
3. **`operatorLink` page.** Documenting it adds a third interface page and a third
   figure; leaving it undocumented still lists it on the operator and remote-control
   pages (unlinked). Decide whether the third page is wanted.
4. **`asElementTable` on `uavSystem::*`** is inferred from `testdata/views`
   (`expose sys::*`), not rendered here; verify with `dot`-less output.
5. **The local `MHz` unit** is validated and recommended; if the extra
   `MeasurementReferences::*`/`SIPrefixes::*` imports are unwanted, fall back to
   `[Hz]` with an integer.

## Verification Commands

```sh
# declarations and values
sysml -validate /tmp/opencode/uav-final
sysml -e "Final::w.meanChord" -e "Final::w.aspectRatio" -e "Final::u.range" \
      -e "Final::b.specificEnergy" -e "Final::an.frequency" /tmp/opencode/uav-final

# end-to-end rendering (values, composite warning)
./bin/sysml-publish -api local -model /tmp/opencode/uav-preview/model \
    -out /tmp/opencode/uav-preview/site -v

# structural checks (view ownership, interface figure, no-ports failure)
./bin/sysml-publish -api local -model /tmp/opencode/viewtest4/model \
    -out /tmp/opencode/viewtest4/site
```

The temporary models live under `/tmp/opencode/`; no repository file was changed by
this research.
