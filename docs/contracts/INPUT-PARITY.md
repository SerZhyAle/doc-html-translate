# Pointer: INPUT-PARITY

- **Id:** `INPUT-PARITY`
- **Version:** 0.5 draft
- **Home:** the shared contracts catalog, `input-controls/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, keyboard and mouse (adopted 2026-09-29, ticket 69; read at 0.4 on 2026-10-20, ticket 120)
- **Owner:** shared (the keyboard and mouse columns are CyrFlip's; amendments through the catalog page)

What it binds: the shared action vocabulary (`confirm`, `cancel`, `move` and its siblings), the rule that no
action exists on one kind of device only, the per-device binding tables of its section 4 and, since 0.4, the
default table `VIEWER-KEYBOARD.md` for "every applicable hardware-keyboard viewer".

**Boundary (2026-10-20).** `MEDIA-VIEWER` names only the FastMediaSorter family as consumers, and this product has
no folder browsing, random or jump, destination slots, slideshow, transport, thumbnails, frame step, zoom or
rotate. The GUI is a converter window and the reader and the extension viewer are document readers (a comic is
a book of page images), so those `VIEWER-KEYBOARD` rows are not applicable here; the rows that apply are
page, next and previous, search, cancel and help.

**Where this product holds it**

- The GUI's shortcuts are one table (`cmd/doc-html-ui/ui.html`, `SHORTCUTS`): `Ctrl+Enter` confirms (an
  addition beside the table's `Enter`, which rule 5 allows), `Esc` steps back one layer and cancels a run only
  while one is active - it never discards unconfirmed work (rule 4) - and `Ctrl+Shift+F` focuses the file box;
  the rail and the two lists walk by arrow keys (`move`). **F1 is not in that table:** bare `F1` is owned by
  one capture-phase handler in `cmd/doc-html-ui/help.js` with one resolver (ticket 118; `help_test.go`,
  `help_resolver_test.mjs`).
- Both readers bind `search` to `Ctrl+F` (`Cmd+F` on macOS, also on a non-Latin layout by the physical key); it
  opens the whole-book search with the field focused, and pressed inside the field it is left to the browser
  (`internal/htmlgen/search.go`, `extension/src/reader-search.js`, a test on each side). Paging at the scroll
  edge uses the Web default keys and a Shift-modified key is the browser's (2026-10-20,
  `internal/htmlgen/navbar.go`). The extension's partial-export dialog takes Escape as its Cancel.
- Every control is a native HTML button or input, so nothing exists on the pointer only (rule 1).
- The extension manifest ships two browser-scope default commands, `Alt+Shift+O` and `Alt+Shift+R`, remappable
  only in the browser; neither is global, and section 7 of `INPUT-CHORD` leaves default chords to the product.
- Hover-only text (the reader bar's truncated titles, two GUI shortcut tooltips, a blocked queue row's reason)
  is a dated registry exception to rule 1, until 2026-12-31.

**`INPUT-CHORD` is a scoped decline (re-checked at 0.4 on 2026-10-20).** The product stores, captures and binds
no chord - no remapping setting, no global hook, no chord token parsed or written. It does *show* chords in
tooltips (`Ctrl+Enter`, `Esc`, `Ctrl+Shift+F`, `Ctrl+F`) in the canonical Latin names. Since 0.5 (item D) a displayed
chord binds rule 5 and nothing else; the registry row reads 0.5 for that rule (see the re-read below).

**Conformance.** No vectors are consumed (the chord vectors are not read: no chord token exists here). The
shortcut table and the arrow-key walkers are pinned by the GUI's own tests; focus visibility by
`TestFocusRingIsNeverSuppressed`.

**Re-read 2026-10-29 (ticket E0126) at 0.5, with `INPUT-CHORD` 0.5.** `INPUT-PARITY` items F-H: nothing owed - the readers keep page, next/previous, search and cancel, `help` = F1 is the GUI's and the readers have no help surface. `INPUT-CHORD` item D answers the question above: a chord that is only shown follows rule 5 and owes nothing else, so the decline narrows to everything but display. The French cancel tooltip now reads `Esc`, and on macOS the reader and extension search tooltips show `⌘+F`.
