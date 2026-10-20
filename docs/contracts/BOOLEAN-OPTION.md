# Pointer: BOOLEAN-OPTION

- **Id:** `BOOLEAN-OPTION`
- **Version:** 2.0 (the catalog's current text read on 2026-10-20)
- **Home:** the shared contracts catalog, `boolean-option/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, `partial` - the desktop launcher (`cmd/doc-html-ui`) implements and measures it (ticket 119, 2026-10-20); the browser extension, the generated reader chrome and the official site are planned slices dated in [`docs/UI_PROFILE.md`](../UI_PROFILE.md)
- **Wire carrier:** none - observable boolean option layout and interaction

Every boolean is an `.opt` row: a leading vector checkbox (20 / 24 visible, 32 / 48 hit), a heading label that is the activation region, the explanation beneath as the checkbox's description, and optional local help (the OCR overlay and the default-handler option) whose short summary and full text are one dictionary string.

**Where it is proven.** Static pins: `cmd/doc-html-ui/*_test.go`. Measured on the real build: `node tools/uicheck/run.mjs` (see [`tools/uicheck/README.md`](../../tools/uicheck/README.md)). The product's size profile, permanent variants and what is still owed: [`docs/UI_PROFILE.md`](../UI_PROFILE.md). Nothing is claimed for a surface that has not been measured.
