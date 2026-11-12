# Pointer: APP-STYLE

- **Id:** `APP-STYLE`
- **Version:** 0.14 draft
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
  destructive and costly buttons of the confirmation dialog use `--danger`. `warning` marks the caveats and
  `danger` the failures and refusals; static section headings take `--text-muted`, not the interaction `accent`.
- **Rule 5 - out-of-theme surfaces.** The log and the command line are declared `.console` - a dark
  field with its own text colours in every theme.
- **Section 10 (0.12) - complete live theme coverage.** Every nested child, the one modal dialog included,
  resolves the same `light-dark()` table on every paint, so a switch reaches everything already on screen;
  forced colors take over the palette (high contrast wins). The Windows layout profile beside it is
  [WINDOWS-UI.md](WINDOWS-UI.md).
- The reader theme palette of the converted pages is shared between the desktop output and the browser
  extension (derived on both sides from `internal/appearance`, tested via `TestAppearanceRolesMatchSource`).

**0.13 - the five status-fill roles are not adopted, on purpose.** They bind a product that has a
state-filled surface, and this GUI has none: state is coloured text (`.hint.*`, `.qstate`, `.res-state`) and the
only status-coloured fill is the acting face of `button.danger`, whose ink is tested at 4.5:1. Nothing is
invented to fit the text. Feasibility was measured on 2026-10-20 (one white ink at 6.27:1 or better on the
light fills, the dark ink at 8.10:1 or better), so a future state board adds the five roles first. Whether
"a word and the ICON-SET glyph per state" binds ink-only status text is asked of the owner in the
`desktop-app-ux` proposal file of 2026-10-20 (item 1); every state has a word today and none has a glyph.

**Conformance.** `cmd/doc-html-ui/contract_test.go` (roles in both themes, three themes, console
declared), `tests/appearance_parity_test.go`, `tests/parity_test.go`. Audit of 2026-10-20: 8 clauses
conform, 6 are not applicable, no gap; `go test ./cmd/doc-html-ui` exit 0.

**Warning-text difference (2026-10-02, until 2026-12-31, narrowed 2026-10-20).** The GUI keeps day
`--warning: #8a4b00` for small text because the kit's `#EF6C00` fails the 4.5:1 floor on its light surfaces
(`TestPaletteTextMeetsWCAGAA`). Colour values are product-owned (section 6, 0.13), so this is no APP-STYLE
deviation; only the glyph tone of the answer dialog differs from `ICON-RENDER`, and the catalog registry
records that one row against `ICON-RENDER`.

**Re-read 2026-10-29 (ticket E0126) at 0.14.** The 2026-10-29 amendment (items A-E) decides this product's proposal on the fills-only reading already applied: no state-filled surface, every state keeps its word, and the rail selection owes no `surface.selected`. Nothing changes.
