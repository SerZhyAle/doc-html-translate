# Pointer: APP-STYLE

- **Id:** `APP-STYLE`
- **Version:** 0.10 draft
- **Home:** the shared contracts catalog, `desktop-app-ux/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - desktop GUI styling (`cmd/doc-html-ui`) and the reader theme palette
- **Wire carrier:** none - theme palette tokens and CSS variables

What the GUI holds:

- **Rule 2 - three themes.** System, light and dark, defaulting to system, switched at once and saved
  with the other GUI settings. System follows the OS while the window runs through
  `prefers-color-scheme` (the browser-hosted form of the live OS signal - ticket 23, B2).
- **Rule 3 - one palette table, dynamic references.** One `:root` table in `ui.html` where every role is a
  `light-dark()` pair, resolved on every paint against `color-scheme`.
- **Rule 4 - role vocabulary.** Custom properties are the vocabulary's names (`--surface-window`,
  `--text-muted`, `--accent-ink`, ..), plus `--accent-hover`, `--success`, `--danger`, `--warning`; the
  destructive and costly buttons of the confirmation dialog use `--danger`.
- **Rule 5 - out-of-theme surfaces.** The log and the command line are declared `.console` - a dark
  field with its own text colours in every theme.
- The reader theme palette of the converted pages is shared between the desktop output and the browser
  extension (derived on both sides from `internal/appearance`, tested via `TestAppearanceRolesMatchSource`).

**Conformance.** `cmd/doc-html-ui/contract_test.go` (roles in both themes, three themes, console
declared), `tests/appearance_parity_test.go`, `tests/parity_test.go`.
