# Changelog

## Unreleased

### Changed
- The example model in `model/` is now a multi-file fixed-wing UAV: the whole
  system (`System::uavSystem`) contains the UAV and its ground remote control,
  the UAV subsets a general UAV (`UavKinds::uav`), and the model demonstrates
  inherited representations, interface figures, views and values across files.
  The UAV is a 50 kg air vehicle with a payload radio, a CAN bus linking its
  digital systems, and a power line with a BEC and a motor controller.
  `docs/example/` is regenerated from it.
- The tool no longer parses the model itself, and holds no model in memory
  (DS-1): it sends the model files' text to a sysml-grpc service (OpenSysML),
  which parses the model, holds it, and answers the tool's reads. Every run
  names the service with `-api ADDR` or `[source] api` in the config file
  (CLI-9, CFG-6); there is no default address, and a service that cannot be
  reached, or lacks a capability the tool needs, fails the run (ER-1) with a
  message naming the address. The tests start their own sysml-grpc
  (`internal/servicetest`) and skip when they cannot get one.
- `sysml-members-list` shows the types the API's `type` answers, which since
  OpenSysML #2198 is the KerML derived feature type: a written type another
  written one specializes is folded away (`Sport :> Wheel` with
  `part both : Wheel, Sport` shows `(Sport)`, not `(Wheel, Sport)`).

### Added
- The block macro `sysml-fig::[<element>][]` (MC-25): a Mermaid figure of an
  interface or of a part, drawn in the browser by the vendored
  `mermaid.min.js` (mermaid 12, ELK layout) in OpenSysML's Mermaid
  interconnection style. An interface's figure: the ends' common ancestor as
  the frame, the parts on its ends and the parts between, each end's
  feature, and one edge per pair of ends. A part's figure: the part's box
  with its owned ports, and, for every connection they are on, the other
  ends' features and their owner parts — beside the part's box, or inside it
  when the model owns them there. A box's title carries the keyword line
  above the name, and the box's contents are drawn below all the title's
  lines. A drawn element with a page, not the page the figure is on, links
  to it in its label, styled as a link. A resolvable target that is neither an interface nor
  a part is
  written out as the macro text; a target naming no element, or a part
  without ports, fails the macro with a warning. A page loads the asset only
  when it shows a figure, and a site with no figure carries no asset.
- Running without a service of your own: `-api local` (or
  `[source] api = "local"`) starts a sysml-grpc for the run — found on
  `PATH` or built from the OpenSysML checkout — uses it, and stops it
  after. A service that cannot be started fails the run saying why (ER-1).
- A resizable sidebar (SB-5): a strip at the sidebar's right edge widens and
  narrows the page tree by dragging; the chosen width is remembered in the
  browser and applied on every page. Double-click, or the arrow keys with the
  strip focused, restores the default width. A page without JavaScript keeps
  the default width, as before.
- Collapsible sidebar items (SB-6): every item with children folds into a
  `details` whose summary carries its link, with a chevron that turns down
  when it is open. Without JavaScript the branches on the current page's
  path are open and the others closed; a small script remembers the user's
  own choices in the browser, keyed by the item's site path, and applies
  them on every page.

## 0.4.0 — 2026-11-18

### Added
- The config file `sysml-publish.toml` (CFG-1..CFG-5): an optional TOML file
  in the working directory or given with `-config`, naming the model folder
  (`[source] dir`), the site folder (`[site] dir`), the site title and the
  base URL. Keys are strict; a flag the user gave wins over the file. An
  example file is in `docs/sysml-publish.toml`.
- The block macro `sysml-catalog::[]` (MC-24): on a view's page, a table of
  the view's exposed elements that carry the view's filter metadata — one
  row per element, the Name column linked to the element's page, one column
  per attribute of the filter metadata def, labelled by the attribute's
  representation body (else its name) and ordered by its `order` literal.
- Sortable tables (SB-4): every table with a header row is sortable in the
  browser — clicking a header sorts the rows by that column, ascending
  first, descending on the second click, with the column marked
  (`aria-sort`, a triangle). The site ships a small script for it; a page
  without JavaScript renders and reads exactly as before.

### Changed
- The repository carries the MIT license (`LICENSE`) and every source file
  a short copyright header.
