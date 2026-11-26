# Pointer: SETTINGS-NAVIGATION

- **Id:** `SETTINGS-NAVIGATION`
- **Version:** 2.6 (the catalog's current text read on 2026-11-15)
- **Home:** the shared contracts catalog, `settings-navigation/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, `partial` - the desktop launcher (`cmd/doc-html-ui`) implements and measures it (ticket 119, 2026-10-20); the browser extension, the generated reader chrome and the official site are planned slices dated in [`docs/UI_PROFILE.md`](../UI_PROFILE.md)
- **Wire carrier:** none - settings navigation, page composition and presentation behavior

A start-side rail and a content pane, seven pages (convert, ollama, ocr, windows, interface, advanced, about), one level, monochrome glyphs, a narrow-window drawer, About last with the whole resource set, the page and viewport remembered, and complete-frame page switching measured.

**Where it is proven.** Static pins: `cmd/doc-html-ui/*_test.go`. Measured on the real build: `node tools/uicheck/run.mjs` (see [`tools/uicheck/README.md`](../../tools/uicheck/README.md)). The product's size profile, permanent variants and what is still owed: [`docs/UI_PROFILE.md`](../UI_PROFILE.md). Nothing is claimed for a surface that has not been measured.

Release re-verification: 2026-11-27: read 2.4-2.6 (items G-M are allowances; no rail headings, action rows or title glyph here). GUI unchanged since v26.1117.1450; tests pass. Partial: theme/language page row (section 3).
