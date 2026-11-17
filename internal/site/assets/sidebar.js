// Copyright 2026 Petr Mukhachev | SPDX-License-Identifier: MIT
//
// Resizing the sidebar (SB-5): dragging the .sidebar-resize strip changes the
// sidebar's width for the page; the width is remembered in the browser and
// applied on every page of the site. Progressive enhancement like the table
// sorting (SB-4): a page whose script does not run keeps the default width,
// and the strip, inert, changes nothing.
(function () {
  "use strict";

  var MIN = 160, MAX = 640, STEP = 16; // px; keyboard steps, Shift for four
  var KEY = "sysml-publish:sidebar-width";

  var root = document.documentElement;
  var handle = document.querySelector(".sidebar-resize");
  var sidebar = document.querySelector(".sidebar");
  if (!handle || !sidebar) return;

  function clamp(px) {
    return Math.min(MAX, Math.max(MIN, px));
  }

  function setWidth(px) {
    root.style.setProperty("--sidebar-width", px + "px");
  }

  function store() {
    try { localStorage.setItem(KEY, String(width)); } catch (e) {}
  }

  function forget() {
    try { localStorage.removeItem(KEY); } catch (e) {}
  }

  // The default is the width the stylesheet gives before any change.
  var def = sidebar.getBoundingClientRect().width;

  // Restore the width the user chose on an earlier page of this site.
  var saved = NaN;
  try { saved = parseFloat(localStorage.getItem(KEY)); } catch (e) {}
  if (!isNaN(saved)) setWidth(clamp(saved));

  var width = clamp(isNaN(saved) ? def : saved);

  // Dragging: from the strip, the width follows the pointer until release.
  var startX = null, startW = 0;
  handle.addEventListener("pointerdown", function (e) {
    if (e.pointerType === "mouse" && e.button !== 0) return;
    handle.setPointerCapture(e.pointerId);
    startX = e.clientX;
    startW = width;
    root.classList.add("sidebar-resizing");
    e.preventDefault();
  });
  handle.addEventListener("pointermove", function (e) {
    if (startX === null) return;
    width = clamp(startW + e.clientX - startX);
    setWidth(width);
  });
  function endDrag() {
    if (startX === null) return;
    startX = null;
    root.classList.remove("sidebar-resizing");
    store();
  }
  handle.addEventListener("pointerup", endDrag);
  handle.addEventListener("pointercancel", endDrag);

  // Double-click restores the stylesheet's default.
  handle.addEventListener("dblclick", function () {
    root.style.removeProperty("--sidebar-width");
    width = def;
    forget();
  });

  // Keyboard: the strip is a separator; the arrows widen and narrow it.
  handle.addEventListener("keydown", function (e) {
    var d = e.shiftKey ? 4 * STEP : STEP;
    if (e.key === "ArrowLeft") width = clamp(width - d);
    else if (e.key === "ArrowRight") width = clamp(width + d);
    else return;
    setWidth(width);
    store();
    e.preventDefault();
  });
})();
