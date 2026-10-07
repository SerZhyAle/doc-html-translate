# Pointer: ICON-RENDER

- **Id:** `ICON-RENDER`
- **Version:** 0.18 draft
- **Home:** the shared contracts catalog, `iconography/README.md` sections 3 and 10, `iconography/palette.json` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, opted in as a draft on 2026-09-25
- **Wire carrier:** none - a picture, not a payload

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
  / `altform-lightunplated` set resolved through `resources.pri` (`msix/build-msix.ps1` runs `makepri`), the
  "Convert to HTML" verb in `action.convert` and the registered document type in `content.document`, both mono
  `#808080` and embedded as exe icon resources 1 and 2, and the extension's action icon as the mark on its own
  plate. `tests/icons_test.go` holds the committed files to the generator, the resource order and the 3:1 ratios.

- **Section 11 (0.16, 2026-10-05):** the navigation identity-ink permission is noted and not taken - the GUI
  has no destination navigation list, so there is nothing to ink; ordinary role colouring stands.

**0.18 review (2026-10-07, ticket 108).** Section 13 C mirrors only the document paging
group in the RTL desktop reader, including disabled endpoints; the extension has no paging
glyph pair. Section 13 E is implemented in both reader editions: 28 px for a fine pointer,
44 px for a coarse pointer. The site keeps 44 px under every pointer (PAGE-STYLE section 5).
No new glyph drawing is introduced. The text-layer off form is still absent from the catalog;
its dated state exception remains. The catalog pixel vector is still owed by the exporter.

**Conformance.** The catalog SVG files are compared by the product's existing iconography
suite; the reader's target floors and RTL paging are checked in ticket 108's browser probe.
