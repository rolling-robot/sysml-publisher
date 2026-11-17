// Copyright 2026 Petr Mukhachev | SPDX-License-Identifier: MIT
//
// Sortable tables (SB-4): clicking a header cell sorts the table's rows by
// that column. Progressive enhancement: the pages read exactly as before
// when this script does not run. Sorts each tbody separately, so a table
// with footer rows keeps them; numbers compare as numbers, everything else
// as text, case-insensitive; a cell with no text sorts last either way.
(function () {
  "use strict";

  // key is what a row's cell in column col sorts by: its text, lowercased,
  // or null when the row has no cell there or the cell has no text.
  function key(row, col) {
    var cell = row.cells[col];
    if (!cell) return null;
    var t = cell.textContent.trim();
    return t === "" ? null : t.toLowerCase();
  }

  // numeric reports whether column col of rows holds numbers only: every
  // non-empty value is one, and at least one is.
  function numeric(rows, col) {
    var some = false;
    for (var i = 0; i < rows.length; i++) {
      var k = key(rows[i], col);
      if (k === null) continue;
      if (!/^[+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?$/.test(k)) return false;
      some = true;
    }
    return some;
  }

  function sortTable(table, col, dir) {
    for (var b = 0; b < table.tBodies.length; b++) {
      var body = table.tBodies[b];
      var rows = [].slice.call(body.rows);
      var num = numeric(rows, col);
      rows.sort(function (r, s) {
        var a = key(r, col), c = key(s, col);
        if (a === null && c === null) return 0;
        if (a === null) return 1; // no value sorts last, either direction
        if (c === null) return -1;
        if (num) {
          var x = parseFloat(a), y = parseFloat(c);
          return x === y ? 0 : (x < y ? -1 : 1) * dir;
        }
        return a < c ? -dir : a > c ? dir : 0;
      });
      for (var i = 0; i < rows.length; i++) body.appendChild(rows[i]);
    }
    var head = table.tHead;
    for (var j = 0; j < head.rows.length; j++) {
      var cells = head.rows[j].cells;
      for (var c = 0; c < cells.length; c++) {
        if (c === col) cells[c].setAttribute("aria-sort", dir < 0 ? "descending" : "ascending");
        else cells[c].removeAttribute("aria-sort");
      }
    }
  }

  var tables = [];
  document.querySelectorAll("table").forEach(function (table) {
    if (table.tHead && table.tHead.rows.length > 0 && table.tBodies.length > 0) {
      tables.push(table);
    }
  });

  tables.forEach(function (table) {
    table.classList.add("sortable");
    var th = table.tHead.rows[table.tHead.rows.length - 1];
    [].forEach.call(th.cells, function (cell, col) {
      cell.addEventListener("click", function () {
        var cur = cell.getAttribute("aria-sort");
        var dir = cur === "ascending" ? -1 : 1; // same column again: flip
        sortTable(table, col, dir);
      });
    });
  });
})();
