# Pointer: ICON-EXTERNAL

- **Id:** `ICON-EXTERNAL`
- **Version:** 0.12 draft
- **Home:** the shared contracts catalog, `iconography/README.md` section 4 (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, opted in as a draft on 2026-09-25; re-read at 0.12 on 2026-10-20 (ticket 120), unchanged
- **Wire carrier:** none

What this repo owes it:
- Rule 1 holds by absence: no third-party mark is drawn anywhere - the store and GitHub channels are text links.
- Rules 2-4: the product shows no other app's icon and downloads no picture of its own. Section 13 F declares
  embedded document images user content; a missing or broken one should show `content.image` with the original
  alt text. That replacement does not exist yet (no error handler in `internal/htmlgen` or `extension/src`, and
  the cover wrapper writes an empty alt) - a dated registry exception until 2026-12-31, remediation ticket 122.
- Rule 5: every copied glyph's source is on record - in the catalog (`ref.source` or `glyph.source`),
  in `assets/glyphs/PROVENANCE.txt` (SHA-256 per file) and the Material Icons (Apache-2.0) licence in
  `THIRD-PARTY-NOTICES.txt`, shipped beside the desktop app and inside the extension package
  (`extension/src/THIRD-PARTY-NOTICES.txt`); `TestGlyphNoticesShipWithTheGlyphs` holds it.
- Rule 6 (0.11): every language this product lets the user choose or see is its endonym and no flag stands for
  one. The GUI's interface-language select (`ENDONYMS`) and translation selects (`TRANSLANG_ENDONYMS`, both in
  `cmd/doc-html-ui/i18n.js`, pinned by `TestUITranslationLangNamesUseEndonyms`), the OCR language and download
  lists of both editions (`internal/ocr/tessdata.go`, `extension/src/ocr-lang.js`) and the extension's interface
  and source-language selects (`extension/src/options.js`, `options.html`). A scan for regional-indicator
  characters over html, js, css, go, json and md finds none; the site's switcher is held by `PAGE-STYLE`
  section 4.2.

No catalog vectors exist for this contract.
