# Pointer: APP-STYLE

- **Id:** `APP-STYLE`
- **Version:** 0.12 draft
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
- **Rule 4, re-checked 2026-10-06 (ticket 95):** `warning` marks the caveats (a key saved but unverified, a
  partial or blocked registration) and `danger` the failures and refusals - before, every `.warn` hint took
  `danger` and `--warning` was declared but unused. Static section headings take `--text-muted`, not the
  interaction `accent`.
- **Rule 5 - out-of-theme surfaces.** The log and the command line are declared `.console` - a dark
  field with its own text colours in every theme.
- **Section 10 (0.12) - complete live theme coverage.** Every nested child, the one modal dialog included,
  resolves the same `light-dark()` table on every paint, so a switch reaches everything already on screen;
  the driven run of 2026-10-05 (ticket 93) cycles Light -> Dark -> Light over sampled controls and the
  window, and the forced-colors emulation takes over the palette (high contrast wins). The Windows layout
  profile beside it is [WINDOWS-UI.md](WINDOWS-UI.md).
- The reader theme palette of the converted pages is shared between the desktop output and the browser
  extension (derived on both sides from `internal/appearance`, tested via `TestAppearanceRolesMatchSource`).

**Conformance.** `cmd/doc-html-ui/contract_test.go` (roles in both themes, three themes, console
declared), `tests/appearance_parity_test.go`, `tests/parity_test.go`.

**Warning text exception (2026-10-02, until 2026-12-31).** The GUI retains day `--warning: #8a4b00`
for small text: the kit's `#EF6C00` fails the 4.5:1 text floor on its light surfaces. This is an
accessibility reason, measured by `TestPaletteTextMeetsWCAGAA`, not a new glyph palette. The vendored
kit stays byte-identical. The catalog registry records this distinction.
