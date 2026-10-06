# Pointer: ICON-RENDER

- **Id:** `ICON-RENDER`
- **Version:** 0.16 draft
- **Home:** the shared contracts catalog, `iconography/README.md` sections 3 and 10, `iconography/palette.json` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, opted in as a draft on 2026-09-25
- **Wire carrier:** none - a picture, not a payload

What this repo owes it:
- Rule 1 and section 10 A: the catalog's own 24 x 24 filled drawings, mono look everywhere (toolbars, links,
  disclosure). The converted book draws them as inline `currentColor` SVG (`internal/htmlgen/glyphs.go`), the
  extension from `extension/src/glyphs.js`, the GUI and the site as inline or CSS-mask copies.
- Rules 2-3: colour from the theme, at least 3:1 on every reader theme (light, sepia, dark, night) - the index
  table of contents takes `--dht-link` from the shared palette (`internal/appearance`).
- Rule 7: `nav.open-external` and `nav.go-to` mirror in a right-to-left layout; the paging pair is `rtl: fixed`
  (whether document paging should follow the reading direction is proposal item 3).
- Rule 8: a glyph-only control's accessible name is the localized word, never the glyph character.
- Rule 9 (since 2026-09-25, ticket 32): `internal/iconart` draws every system surface from the product mark
  and the vendored glyphs - the ICO at 16-256 px, the MSIX `Square44x44Logo` `targetsize-*` / `altform-unplated`
  / `altform-lightunplated` set resolved through `resources.pri` (`msix/build-msix.ps1` runs `makepri`), the
  "Convert to HTML" verb in `action.convert` and the registered document type in `content.document`, both mono
  `#808080` and embedded as exe icon resources 1 and 2, and the extension's action icon as the mark on its own
  plate. `tests/icons_test.go` holds the committed files to the generator, the resource order and the 3:1 ratios.

- **Section 11 (0.16, 2026-10-05):** the navigation identity-ink permission is noted and not taken - the GUI
  has no destination navigation list, so there is nothing to ink; ordinary role colouring stands.

**Open, under dated registry exceptions:** rule 4 for the text-layer toggle (`view.text-layer` has no drawn off
form yet, so `aria-pressed` and a pressed look carry the state), rule 5's 44 px target on the dense reader bar
and the site (proposal item 8, co-signed). Widened 2026-10-06 (ticket 95) to every web glyph-only control
measured below 44 px: the desktop reader bar's and index toolbar's glyph buttons (about 33 x 26 px), the
extension viewer's toolbar buttons (about 35 x 31 px, height now pinned) and its contents disclosure toggles
(raised from 16.8 px to the product's 24 x 24 px floor). All hold that floor (`docs/PARITY.md`, ticket 57).
