# Pointer: BUTTONS

- **Id:** `BUTTONS`
- **Version:** 2.4 (the catalog's current text read on 2026-11-16, ticket E0150)
- **Home:** the shared contracts catalog, `buttons/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, `partial` - the desktop launcher (`cmd/doc-html-ui`), browser extension, generated reader chrome and official site all implement route D1 touch captions (`::after` `attr(title)` / `""`: `@media (pointer: coarse)` on web surfaces, `html[data-input="touch"]` in the launcher per section 11 item F) and `#toTop` `title` (ticket E0150); full browser measurement on all surfaces waits for the next release run, as dated in [`docs/UI_PROFILE.md`](../UI_PROFILE.md)
- **Wire carrier:** none - observable button appearance, layout and interaction

Rectangular, content-width buttons with the glyph before the caption on a neutral face; ranks ordinary / secondary / primary / destructive; wrapping flex rows with equal visual-row heights; stateful buttons reserve their widest state; dialog answers read Cancel first and the affirmative last with the safe answer focused. The drop zone is a drop target, not a command button (recorded variant).

**Where it is proven.** Static pins: `cmd/doc-html-ui/*_test.go`. Measured on the real build: `node tools/uicheck/run.mjs` (see [`tools/uicheck/README.md`](../../tools/uicheck/README.md)). The product's size profile, permanent variants and what is still owed: [`docs/UI_PROFILE.md`](../UI_PROFILE.md). Nothing is claimed for a surface that has not been measured.

Release re-verification: 2026-11-27: read 2.4 (items H-L are allowances). GUI unchanged since v26.1117.1450; extension re-measured, uicheck PASS 5121, hidden-button defect fixed. Partial: physical pass.
