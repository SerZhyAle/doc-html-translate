# Pointer: ICON-SET

- **Id:** `ICON-SET`
- **Version:** 0.36 draft
- **Home:** the shared contracts catalog, `iconography/README.md` section 2, data in `iconography/vocabulary.jsonl` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, `partial` - opted in as a draft on 2026-09-25; every edition (converted book, GUI, extension, site); read at 0.36 on 2026-10-20 (ticket 120)
- **Wire carrier:** `vocabulary.version` at build time (section 5); the drawings this product ships are vendored under `assets/glyphs/`

Meanings added after 0.30 (`app.tour`, `action.paste`, `action.skip`, `action.replace`, `action.unhide`, the exchange
glyphs) are not drawn here, and `app.help` (0.34) matches both uses (the header `?` button with F1, and the
option help). No id this product draws is retired or deprecated.

What this repo owes it:
- One meaning, one glyph, one name (rules 1-3): the inventory is [`../GLYPH-MAP.md`](../GLYPH-MAP.md), every row
  `conforms`, `exception` or `artwork`. Paging is `media.previous` / `media.next`, never `nav.back` / `nav.forward`.
  The vendored files (47 on 2026-10-20) are byte-identical to the catalog export, SHA-256 in
  [`../../assets/glyphs/PROVENANCE.txt`](../../assets/glyphs/PROVENANCE.txt).
- A control's name is its meaning's name, never another meaning's word (rule 3). Fixed 2026-10-20: the Russian
  Cancel is "Отмена" (not `action.undo`'s word), the queue Clear names `action.clear-all`, the extension's Retry
  names are `action.retry`'s, the OCR bar's Stop is Cancel, and the Close answers draw `nav.close`. Held by
  `TestIconographyNamesAreNotAnotherMeaningsName`, which reads the catalog's records (it skips without
  `SZA_CONTRACTS_ROOT` and logs the vocabulary version).
- New development starts in the vocabulary (rule 5): a control whose meaning is missing keeps its old face under a
  dated registry exception until the catalog adds the meaning - never a private glyph. `action.retry` and
  `app.update` are in the catalog since 0.33 / 0.31 and vendored.
- Names the catalog lacks for this product's controls (13 used ids are named only in en, ru, uk) are offered by
  proposal, not invented (the `iconography` proposal file of 2026-10-20, item 3).

**Deviations, in the catalog registry (section 3).** Temporary, until 2026-12-31: further-language names (rule 3,
section 13 B) - 54 (id, language) cells over 19 ids, ticket 122 - and the site's theme button glyph (ticket 119).
Permanent variants, recorded in [`../UI_PROFILE.md`](../UI_PROFILE.md) section 4: the dialog affirmative "OK" drawn
with `action.confirm` (decision D6 of ticket 119), and settings destinations that show the glyph of the nearest
meaning under the destination's own topic name (decision D4). Section 2.8 (docs and site show the glyph beside the
name) was only sampled.

**Conformance.** `tests/iconography_test.go` compares all vendored SVG bytes with the catalog when
`SZA_CONTRACTS_ROOT` is set, and checks edition tables, inline copies, shared labels and the name guard above;
`internal/htmlgen/glyphs_test.go` checks the reader controls in all thirteen languages. The catalog has no
generated pixel vectors yet (section 3 rule 1).
