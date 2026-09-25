# Pointer: ICON-RENDER

- **Id:** `ICON-RENDER`
- **Version:** 0.13 (draft)
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

**Open, under dated registry exceptions:** rule 4 for the text-layer toggle (`view.text-layer` has no drawn off
form yet, so `aria-pressed` and a pressed look carry the state), rule 5's 44 px target on the dense reader bar
and the site (proposal item 8, co-signed), rule 9's system surfaces - the ICO, the MSIX tile set, the document
type, the "Convert to HTML" verb (`action.convert`, mono `#808080`) and, since 0.13, the extension's action
icon - which ticket 32 carries.
