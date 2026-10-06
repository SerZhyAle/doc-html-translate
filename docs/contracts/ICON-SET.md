# Pointer: ICON-SET

- **Id:** `ICON-SET`
- **Version:** 0.17 draft
- **Home:** the shared contracts catalog, `iconography/README.md` section 2, data in `iconography/vocabulary.jsonl` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, opted in as a draft on 2026-09-25 - every edition (converted book, GUI, extension, site)
- **Wire carrier:** none - a vocabulary read by people; the drawings this product ships are vendored under `assets/glyphs/`

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
  "Оглавление" as `nav.contents`' Russian book-reader form; the page forms of `nav.scroll-top`. Still open there:
  items 3 (RTL paging), 5, 11 and 12, and the co-signed items 2, 6-9.

**Re-checked 2026-10-06 (ticket 95).** Two typed glyphs removed and put on the retired-glyph gate: the
single-page contents panel's `×` close (now the localized word "Close", like the search panel) and the
extension popup's `→` in an untranslated sentence. Names corrected to their records: the extension's site-list
"Remove" read ru "Удалить" / uk "Видалити" (`action.delete`'s words; now "Убрать" / "Прибрати"), the button
that abandons a running export preparation read "Stop" (`media.stop`; now `action.cancel`'s "Cancel"), the
popup's link to the settings page read "Options" (now `app.settings`' "Settings"), and the night theme read
"Ночь" / "Ніч" against the `app.theme` note ("Ночная" / "Нічна"), in both editions - the four theme words are
now compared across editions by `TestEditionsNameSharedControlsAlike`. The seven 0.15 records this product
shipped in `v26.0930.1107` are still `proposed` in the vocabulary; turning them `active` (rule 6) is asked of
the owner in `iconography/PROPOSAL-2026-10-06-doc-html-translate-shipped-records.md`.

**Conformance.** Rung 2 (inventoried) and rung 3 (labels agree with glyphs) for the meanings the product
ships: `tests/iconography_test.go` (vendored copies = catalog, both editions' tables, inline copies, retired
glyphs, shared names) and `internal/htmlgen/glyphs_test.go` (the paging pair and every glyph-only reader control, glyph and name,
in all thirteen languages).
