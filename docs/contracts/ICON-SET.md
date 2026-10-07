# Pointer: ICON-SET

- **Id:** `ICON-SET`
- **Version:** 0.28 draft
- **Home:** the shared contracts catalog, `iconography/README.md` section 2, data in `iconography/vocabulary.jsonl` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, opted in as a draft on 2026-09-25 - every edition (converted book, GUI, extension, site)
- **Wire carrier:** `vocabulary.version` at build time (section 5); the drawings this product ships are vendored under `assets/glyphs/`

0.16 (2026-09-27) added one meaning, `content.disk-container` (a `.fdd` container, FileDO's surface);
no surface of this product draws a `.fdd` file, so nothing here changes. Re-read with ticket 69.

What this repo owes it:
- One meaning, one glyph, one name (rules 1-3): the inventory is [`../GLYPH-MAP.md`](../GLYPH-MAP.md), every row
  `conforms`, `exception` or `artwork`. Paging is `media.previous` / `media.next`, never `nav.back` / `nav.forward`.
- New development starts in the vocabulary (rule 5): a control whose meaning is missing keeps its old face under a
  dated registry exception until the catalog adds the meaning - never a private glyph.
- What the vocabulary lacked was filed as `iconography/PROPOSAL-2026-09-25-doc-html-translate.md` and decided in
  0.15 (2026-09-25): `app.theme`, `action.text-smaller` / `action.text-larger`, `view.text-layer`,
  `nav.go-to-page`, `action.convert`, `action.install`; the word "Copied" as `action.copy`'s done state;
  "Оглавление" as `nav.contents`' Russian book-reader form; the page forms of `nav.scroll-top`.

**0.28 review (2026-10-07, ticket 108).** The seven records reported shipped in the 2026-10-06
proposal are active. Section 13 C permits mirrored document paging without splitting the
transport pair; the desktop reader implements it, the extension has no such glyph pair.
The brush/drawing split affects no surface here. Newly imported further-language names
(section 13 B) require a broader string/document audit and remain a dated exception.

**Conformance.** `tests/iconography_test.go` compares all vendored SVG bytes with the catalog
when `SZA_CONTRACTS_ROOT` is set, and checks edition tables, inline copies and shared labels;
`internal/htmlgen/glyphs_test.go` checks the reader controls in all thirteen languages.
The catalog has no generated pixel vectors yet (section 3 rule 1).
