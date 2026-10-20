# Pointer: UI-SIZING

- **Id:** `UI-SIZING`
- **Version:** 1.1 (the catalog's current text read on 2026-10-20)
- **Home:** the shared contracts catalog, `ui-sizing/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, `partial` - the desktop launcher (`cmd/doc-html-ui`) implements and measures it (ticket 119, 2026-10-20); the browser extension, the generated reader chrome and the official site are planned slices dated in [`docs/UI_PROFILE.md`](../UI_PROFILE.md)
- **Wire carrier:** none - component size, input profile and scale composition

One token resolver (`--ui` interface scale, `--tx` text-only enlargement, the input profile `data-input` chosen once at opening) serves every covered component of the launcher. Range compact 0.875 .. huge 2; declared smallest targets 28 (pointer) and 42 (touch) CSS px; ordinary 32 and 48; the Interface page exposes the size, the text size and the Controls profile. `docs/UI_PROFILE.md` section 2 is the size-profile record.

**Where it is proven.** Static pins: `cmd/doc-html-ui/*_test.go`. Measured on the real build: `node tools/uicheck/run.mjs` (see [`tools/uicheck/README.md`](../../tools/uicheck/README.md)). The product's size profile, permanent variants and what is still owed: [`docs/UI_PROFILE.md`](../UI_PROFILE.md). Nothing is claimed for a surface that has not been measured.
