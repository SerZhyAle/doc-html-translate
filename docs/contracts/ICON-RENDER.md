# Pointer: ICON-RENDER

- **Id:** `ICON-RENDER`
- **Version:** 0.24 draft
- **Home:** the shared contracts catalog, `iconography/README.md` sections 3 and 10, `iconography/palette.json` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, opted in as a draft on 2026-09-25; read at 0.24 on 2026-10-20 (ticket 120)
- **Wire carrier:** none - a picture, not a payload

Since 0.20 every button surface rule lives in [`BUTTONS`](BUTTONS.md), and sizes follow
[`UI-SIZING`](UI-SIZING.md) with its precedence over fixed floors (0.23, 0.24); the 28 / 44 targets and the tinted
answer faces this pointer once described are superseded and not claimed. The product's profile is
[`../UI_PROFILE.md`](../UI_PROFILE.md).

What this repo owes it:
- Rule 1 and section 10 A: the catalog's own 24 x 24 filled drawings, mono look everywhere (toolbars, links,
  disclosure). The converted book draws them as inline `currentColor` SVG (`internal/htmlgen/glyphs.go`), the
  extension from `extension/src/glyphs.js`, the GUI and the site as inline or CSS-mask copies.
- Rules 2-3: colour from the theme, at least 3:1 on every reader theme (light, sepia, dark, night) - the index
  table of contents takes `--dht-link` from the shared palette (`internal/appearance`).
- Rule 7 and section 13 C: `nav.open-external` and `nav.go-to` mirror in a right-to-left layout;
  document paging follows the reading direction. Transport previous/next retain their fixed direction.
- Rule 8: a glyph-only control's accessible name is the localized word, never the glyph character.
- Rule 9 (since 2026-09-25, ticket 32): `internal/iconart` draws every system surface from the product mark
  and the vendored glyphs - the ICO at 16-256 px, the MSIX `Square44x44Logo` `targetsize-*` / `altform-unplated`
  / `altform-lightunplated` set resolved through `resources.pri`, the "Convert to HTML" verb in `action.convert` and
  the registered document type in `content.document`, both mono `#808080`, and the extension's action icon as the
  mark on its own plate. `tests/icons_test.go` holds the committed files to the generator.
- Section 11 (0.22): one colour policy for settings destinations - the rails are monochrome and the navigation
  identity-ink permission is noted, not taken. Section 20 of 2026-10-17 (0.21): a group's state is the triangle
  of `COLLAPSIBLE-GROUP`, not an action glyph.

**Deviations, in the catalog registry.** Rule 4 (a toggle shows its state): the text-layer toggle draws
`view.text-layer` in one form because the catalog has no `view.text-layer--off` drawing; the state rides
`aria-pressed` and a pressed look - temporary, ticket 122, until 2026-12-31. The warning glyph tone of the answer
dialog uses the darker `#8a4b00` ink instead of the kit tone (measured: the kit tone is 2.72:1 on the light
surface) - temporary, until 2026-12-31; this is the half of the old warning-text row that stays.

**Conformance.** The catalog SVG files are compared by the product's iconography suite (47 of 47 identical on
2026-10-20); the system-surface, RTL-paging and target rules are checked by `tests/icons_test.go`,
`internal/htmlgen/navbar_test.go` and `tools/uicheck` (ticket 119). Not verified: the installed MSIX tile and
taskbar look, the toolbar icon in Chrome and Edge, OS high contrast.
