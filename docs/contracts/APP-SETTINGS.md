# Pointer: APP-SETTINGS

- **Id:** `APP-SETTINGS`
- **Version:** 0.3 draft
- **Home:** the shared contracts catalog, `desktop-app-ux/APP-SETTINGS.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - the GUI launcher's settings surface (`cmd/doc-html-ui`), adopted 2026-10-05, ticket 93
- **Wire carrier:** none - a shipped user-facing surface (the GUI's saved blob is this product's own store, wire: none)

The GUI's whole window is its settings surface, in the no-Commit shape APP-BEHAVIOUR rule 12 names: every
value control applies on touch and can be touched back; anything irreversible (delete result, clear logs,
clear the recent-documents history, rebuild over a changed result) keeps its own button and its own
confirmation with the safe answer focused. The two Windows registrations are reversible toggles, not
operations: switching one off unregisters it and restores the handler it replaced from its backup, and the
first write is the user's own yes (APP-BEHAVIOUR rule 4, 0.12). What the twelve rules plus section 8 find
here:

- **Rules 1-2:** one surface; the pane opens where the window opens and remembers its page context (groups,
  viewport) across sessions. A second launch of the exe is a documented dated exception (a second instance
  runs; no activation IPC is built). The language and theme selectors sit in the header - there is no
  first page and no page list, because there is one pane; About is last.
- **Rules 3-4:** every setting of the pane carries its caption, control and muted hint - ticket 93 added five,
  ticket 95 (2026-10-06) the five it missed (output folder, the language pair, the three Ollama values) and
  the one-line purpose under the title; the two header switchers (theme, language) carry no visible caption
  or hint and are a dated registry exception (their caption is the localized accessible name; their options
  name themselves). The commit model is the declared autosave shape, above.
- **Rules 5-6:** the language selector offers all 13 endonyms and applies live, including layout direction;
  the theme selector offers system/light/dark (default system, applying without a restart).
- **Rules 7-9:** logical metrics and the browser's per-monitor DPI hold 100-200% (emulated in the driven
  run; physical mixed-monitor is open); the surface fits its content at open, remembers the user's rectangle
  and clamps to the work area; Escape in a dialog is the no-action exit (the window itself is the app's
  main surface, not a companion).
- **Rules 10-12:** keyboard-first operation with visible focus (`:focus-visible` - since ticket 95 on the text
  fields too, which had suppressed it), accessible names on every glyph-only control, select and field
  (`TestEveryFieldIsNamedAndEveryLabelNamesOne`: every caption names its control by `for=`); About carries the version, the licence (MIT), the support bundle
  ("Send logs to the author", written only on a click) and the privacy policy link (`TestAboutCarriesLicenceAndPrivacy`).
- **Section 8:** the Windows layout detail is [WINDOWS-UI](WINDOWS-UI.md); the GUI implements its row,
  editor, checkbox, numeric and remembered-context conventions as that pointer records.

**Open:** rule 11's inventory-vs-documentation alignment is informal (the GUI's settings are the CLI flags
README.md documents; a mechanical alignment check is future work), recorded in the catalog registry.

**Conformance.** The same gates as [WINDOWS-UI](WINDOWS-UI.md) (`cmd/doc-html-ui/contract_test.go`, ticket 93;
`cmd/doc-html-ui/conformance_test.go`, ticket 95),
plus the standing dictionary and markup gates (`i18n_test.go`, `main_test.go`, `session_test.go`).

**Review 2026-10-07 (ticket 108).** Section 9 permits dedicated, confirmed operations on this touch-commit surface. Clearing logs re-reads the count and reports an empty store without asking; clearing recent history is hidden when empty. No value control deletes data. Existing rule 11 and physical acceptance exceptions remain.
