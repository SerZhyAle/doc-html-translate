# Pointer: ICON-EXTERNAL

- **Id:** `ICON-EXTERNAL`
- **Version:** 0.12 draft
- **Home:** the shared contracts catalog, `iconography/README.md` section 4 (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, opted in as a draft on 2026-09-25
- **Wire carrier:** none

What this repo owes it:
- Rule 1 holds by absence: no third-party mark is drawn anywhere - the store and GitHub channels are text links.
- Rules 2-4: the product shows no other app's icon and downloads no picture of its own. Section 13 F now declares embedded document images user content; missing-image glyph
  replacement remains a dated exception until 2026-12-31.
- Rule 5: every copied glyph's source is on record - in the catalog (`ref.source` or `glyph.source`, 0.10),
  in `assets/glyphs/PROVENANCE.txt` (SHA-256 per file) and the
  Material Icons (Apache-2.0) licence in `THIRD-PARTY-NOTICES.txt`, shipped beside the desktop app and inside
  the extension package (`extension/src/THIRD-PARTY-NOTICES.txt`).
- Rule 6 (0.11, decided 2026-10-02; checked against the tree 2026-10-06): every language this product lets
  the user choose or see is its endonym and no flag stands for one. The GUI's interface-language select
  (`ENDONYMS`) and translation source/target selects (`TRANSLANG_ENDONYMS`, both in `cmd/doc-html-ui/i18n.js`,
  pinned by `TestUITranslationLangNamesUseEndonyms`), the OCR language and download lists of both editions
  (`internal/ocr/tessdata.go`, `extension/src/ocr-lang.js` - script qualifiers such as `縦書き` / `简体`
  in the language's own script), and the extension's interface and source-language selects
  (`extension/src/options.js`, `options.html`). No flag image or regional-indicator character exists in
  either edition; the site's switcher is held by `PAGE-STYLE` section 4.2.

**0.12 review (2026-10-07, ticket 108).** Section 13 F requires `content.image` plus
the original alt text for a missing document image. The existing editions preserve source
images/alt text but have no common failure renderer; the dated exception records that gap.
No catalog vectors exist for this contract.
