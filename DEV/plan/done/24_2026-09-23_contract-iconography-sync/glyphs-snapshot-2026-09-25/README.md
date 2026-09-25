# Glyph snapshot 2026-09-25

**Source:** the shared contracts catalog, `iconography/glyphs/`, copied byte for byte on 2026-09-25
(SHA-256 of every copy compared with its source: 29 of 29 identical). Catalog versions at copy time:
`ICON-SET` 0.13, `ICON-RENDER` 0.11, `ICON-EXTERNAL` 0.9 (all `draft`).

**What it is:** a working copy of the 29 glyphs that ticket
[`24_2026-09-23_contract-iconography-sync`](../../24_2026-09-23_contract-iconography-sync.md) maps or
contrasts, so its rendering question (open question 3) and its Direction A steps can be worked from this
repository alone. It is not a second source: the catalog stays authoritative, a file here is never edited,
and **this folder is deleted together with the ticket** when the ticket moves to `done/`. Code that ships a
glyph copies it from the catalog at that time (or from this folder only after checking the catalog has not
moved), and records its own source and licence (`ICON-EXTERNAL` rule 5, ticket step A9).

Every file is 24 x 24, painted with `currentColor` only (colour comes from the theme, `ICON-RENDER` rule 2).
Two files use SVG features a partial renderer might miss: `action.extract-text.svg` (`fill-rule="evenodd"`)
and `action.download.svg` (nested `<g transform>`). `media.play.svg` and `action.run.svg` are identical by
design (declared share, `sharedWith`).

| File | Meaning | Shape (vocabulary) |
| --- | --- | --- |
| `nav.back.svg` | Back | arrow pointing left |
| `nav.forward.svg` | Forward | arrow pointing right |
| `media.previous.svg` | Previous | triangle pointing left against a bar |
| `media.next.svg` | Next | triangle pointing right against a bar |
| `media.play.svg` | Play | solid triangle pointing right |
| `action.run.svg` | Run | solid triangle pointing right (shared with `media.play`) |
| `nav.contents.svg` | Table of contents | lines with bullets |
| `feature.continue-reading.svg` | Continue reading | open book |
| `nav.scroll-top.svg` | Scroll to top | arrow up to a bar |
| `nav.scroll-bottom.svg` | Scroll to bottom | arrow down to a bar |
| `action.save.svg` | Save | floppy disk |
| `action.export.svg` | Export | arrow rising from a tray |
| `action.copy.svg` | Copy | two overlapping sheets |
| `action.confirm.svg` | Apply | check mark |
| `nav.go-to.svg` | Go to | chevron pointing right |
| `nav.open-external.svg` | Open in new window | arrow leaving a square at its top corner |
| `nav.expand.svg` | Expand | chevron pointing down |
| `nav.collapse.svg` | Collapse | chevron pointing up |
| `nav.dropdown.svg` | Choose from list | small solid triangle pointing down |
| `action.download.svg` | Download | cloud with a downward arrow |
| `action.move-up.svg` | Move up | arrow pointing up |
| `action.move-down.svg` | Move down | arrow pointing down |
| `action.swap.svg` | Swap | two arrows pointing opposite ways |
| `action.extract-text.svg` | Extract text | the letters OCR |
| `media.zoom-in.svg` | Zoom in | magnifier with a plus |
| `media.zoom-out.svg` | Zoom out | magnifier with a minus |
| `weather.clear.svg` | Clear | sun |
| `app.night-mode.svg` | Night mode | crescent moon |
| `camera.record-video.svg` | Record video | solid dot |

## Licence note

The catalog carries no licence file. `ICON-EXTERNAL` rule 5, verbatim: "**The source of every glyph is on
record.** A glyph exported from a product names its drawable in `ref`; a drawn one names its origin in
`glyph.source` (Material Icons, Apache-2.0, or "drawn for this contract"). A product that copies a glyph
copies its licence obligation with it."

All 29 records above are `ref` records - exported from FastMediaSorter Android drawables (for example
`media.previous` -> `ic_skip_previous`, `nav.contents` -> `ic_toc`); none carries a `glyph.source`, so the
catalog does not state their licence. `ICON-RENDER` rule 1 names Material Icons (filled) as "the base of
the vocabulary", and the `ICON-SET` document log (0.9) says only the 17 then-proposed records were drawn
from Material Icons (Apache-2.0) or for the contract. Before any of these files ships, ticket step A9 has
to settle the licence of each copied drawing (with the owner if needed) and record it in a third-party
notices file.
