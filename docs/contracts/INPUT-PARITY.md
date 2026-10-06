# Pointer: INPUT-PARITY

- **Id:** `INPUT-PARITY`
- **Version:** 0.3 draft
- **Home:** the shared contracts catalog, `input-controls/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, keyboard and mouse (adopted 2026-09-29, ticket 69)
- **Owner:** shared (the keyboard and mouse columns are CyrFlip's; amendments through the catalog page)

What it binds: the shared action vocabulary (`confirm`, `cancel`, `move` and its siblings) and the rule that no action exists
on one kind of device only, with the per-device binding tables of its section 4.

**Where this product holds it**

- The GUI's whole shortcut set is one table (`cmd/doc-html-ui/ui.html`, `SHORTCUTS`): `Ctrl+Enter`
  confirms (an addition beside the table's `Enter`, which rule 5 allows), `Esc` cancels and fires only
  while a run is active - it never discards unconfirmed work (rule 4) - and `Ctrl+Shift+F` focuses the
  file box. The queue and recent-documents lists walk by arrow keys (`move`).
- Every control is a native HTML button or input in the window's web view, so nothing exists on the
  pointer only (rule 1); the measured floor behind it is ticket 57's accessibility pass.
- The extension viewer gained keyboard reach with ticket 60: arrow / PageUp / PageDown paging and
  tab-reachable toolbar buttons.

- **Since 2026-10-06 (ticket 95)** the converted book's reader is named in the registry row too. Both
  readers bind `search` to `Ctrl+F` (`Cmd+F` on macOS, also on a non-Latin layout by the physical key):
  it opens the whole-book search with the field focused and selected; pressed inside the field it is not
  intercepted, so the browser's own find bar is one more press away (`internal/htmlgen/search.go`,
  `extension/src/reader-search.js`, a test on each side). The extension's partial-export dialog now takes
  Escape as its Cancel and returns the focus to the export button. The GUI intercepts Escape only while a
  run is active and never while its dialog is open. Three hover-only texts (the reader bar's truncated
  titles, two GUI shortcut tooltips, a blocked queue row's reason) are a dated registry exception to rule 1.

`INPUT-CHORD` is deliberately not adopted: this product stores, captures and globally binds no chord, and
per-application in-window shortcuts that die with the window are out of that contract's own scope.

**Conformance.** No vectors yet (the contract's section 6 names the ladder). The shortcut table and the
two arrow-key walkers are pinned by the GUI's own tests; focus visibility rides the web view's default
focus rings.
