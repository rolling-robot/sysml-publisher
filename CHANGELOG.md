# Changelog

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
