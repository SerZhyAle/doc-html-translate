# Pointer: APP-STYLE

- **Id:** `APP-STYLE`
- **Version:** 0.9 draft
- **Home:** the shared contracts catalog, `desktop-app-ux/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - desktop GUI and reader styling
- **Wire carrier:** none - theme palette tokens and CSS variables

Visual styling and theme mechanisms:
- Theme support: dark, light, and system modes.
- Reader theme palette shared across desktop HTML output and browser extension (tested via `TestParityThemePalette`).
- Clear visual hierarchy with accessible contrast.

**Conformance.** `tests/parity_test.go` and `cmd/doc-html-ui` tests.
