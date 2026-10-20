# Pointer: COLLAPSIBLE-GROUP

- **Id:** `COLLAPSIBLE-GROUP`
- **Version:** 1.1 (the catalog's current text read on 2026-10-20)
- **Home:** the shared contracts catalog, `collapsible-group/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, `partial` - the desktop launcher (`cmd/doc-html-ui`) implements and measures it (ticket 119, 2026-10-20); the browser extension, the generated reader chrome and the official site are planned slices dated in [`docs/UI_PROFILE.md`](../UI_PROFILE.md)
- **Wire carrier:** none - observable group structure, disclosure state and flow

Groups are used where a page has several topics - the About page (Updates, Support and diagnostics, Resources, Related products); a vector state triangle, a distinct header, the whole header toggles, the body opens in flow across the pane. Single-topic pages show their controls directly; no packing.

**Where it is proven.** Static pins: `cmd/doc-html-ui/*_test.go`. Measured on the real build: `node tools/uicheck/run.mjs` (see [`tools/uicheck/README.md`](../../tools/uicheck/README.md)). The product's size profile, permanent variants and what is still owed: [`docs/UI_PROFILE.md`](../UI_PROFILE.md). Nothing is claimed for a surface that has not been measured.
