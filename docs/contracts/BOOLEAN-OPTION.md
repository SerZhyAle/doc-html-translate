# Pointer: BOOLEAN-OPTION

- **Id:** `BOOLEAN-OPTION`
- **Version:** 2.3 (the catalog's current text read on 2026-11-16, ticket E0150)
- **Home:** the shared contracts catalog, `boolean-option/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, `partial` - launcher, extension options and popup implement the rules (ticket E0150 removed the lone `view.text-layer` leading glyph on the OCR option heading across all three surfaces, pinned by `TestOptionHeadingGlyphUniformity`; full `tools/uicheck` verification on the built window and extension runs at release, per [`docs/UI_PROFILE.md`](../UI_PROFILE.md))
- **Wire carrier:** none - observable boolean option layout and interaction

Every boolean is an `.opt` row: a leading vector checkbox (20 / 24 visible, 32 / 48 hit), a heading label that is the activation region, the explanation beneath as the checkbox's description, and optional local help (the OCR overlay and the default-handler option) whose short summary and full text are one dictionary string.

**Where it is proven.** Static pins: `cmd/doc-html-ui/*_test.go`. Measured on the real build: `node tools/uicheck/run.mjs` (see [`tools/uicheck/README.md`](../../tools/uicheck/README.md)). The product's size profile, permanent variants and what is still owed: [`docs/UI_PROFILE.md`](../UI_PROFILE.md). Nothing is claimed for a surface that has not been measured.

Release re-verification: 2026-11-27: read 2.3 (compact-range allowance, nothing owed). No option row added or changed since v26.1117.1450, whose uicheck passed; GUI tests pass. Partial: physical pass owed.
