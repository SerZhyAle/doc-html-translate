# Pointer: ICON-SET

- **Id:** `ICON-SET`
- **Version:** 0.15 (draft)
- **Home:** the shared contracts catalog, `iconography/README.md` section 2, data in `iconography/vocabulary.jsonl` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, opted in as a draft on 2026-09-25 - every edition (converted book, GUI, extension, site)
- **Wire carrier:** none - a vocabulary read by people; the drawings this product ships are vendored under `assets/glyphs/`

What this repo owes it:
- One meaning, one glyph, one name (rules 1-3): the inventory is [`../GLYPH-MAP.md`](../GLYPH-MAP.md), every row
  `conforms`, `exception` or `artwork`. Paging is `media.previous` / `media.next`, never `nav.back` / `nav.forward`.
- New development starts in the vocabulary (rule 5): a control whose meaning is missing keeps its old face under a
  dated registry exception until the catalog adds the meaning - never a private glyph.
- What the vocabulary lacked was filed as `iconography/PROPOSAL-2026-09-25-doc-html-translate.md` and decided in
  0.15 (2026-09-25): `app.theme`, `action.text-smaller` / `action.text-larger`, `view.text-layer`,
  `nav.go-to-page`, `action.convert`, `action.install`; the word "Copied" as `action.copy`'s done state;
  "Оглавление" as `nav.contents`' Russian book-reader form; the page forms of `nav.scroll-top`. Still open there:
  items 3 (RTL paging), 5, 11 and 12, and the co-signed items 2, 6-9.

**Conformance.** Rung 2 (inventoried) and rung 3 (labels agree with glyphs) for the meanings the product
ships: `tests/iconography_test.go` (vendored copies = catalog, both editions' tables, inline copies, retired
glyphs, shared names) and `internal/htmlgen/glyphs_test.go` (the paging pair and every glyph-only reader control, glyph and name,
in all thirteen languages).
