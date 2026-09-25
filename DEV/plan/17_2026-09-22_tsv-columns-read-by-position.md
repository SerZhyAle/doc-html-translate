# The engine's TSV is read by column number, and the header that names them is thrown away

**Status:** Draft
**Priority:** 52
**Date:** 2026-09-22

> Go edition only. The extension reads tesseract.js's structured result, not a TSV.

## What / why

`parseTSV` skips the header row and then reads fixed indices - `cols[0]` level, `cols[6..9]` the box,
`cols[10]` confidence, `cols[11]` the text ([`internal/ocr/tesseract.go:1108-1148`](../../internal/ocr/tesseract.go#L1108-L1148)).
The header row it skips is exactly the map that would make the read safe.

Nothing is wrong today: Tesseract's TSV layout has been stable for years and a short row is skipped
(`len(cols) < 12`), so a truncated line degrades rather than crashes. The failure mode is the one the
portfolio's compatibility law exists to prevent - **match by name, never by position**
(`VERSIONING.md` section 4 rule 1): an engine build that inserts a column shifts every field at once, and
the result is not an error but plates in the wrong place carrying the wrong confidences. A user who
installed a newer Tesseract would see it; we would see a bug report about geometry.

Found by the contract alignment run of 2026-09-22. It is not a catalog contract boundary - Tesseract is a
third party, not a portfolio product - so it is filed here rather than as an exception in the registry.

## Done criteria

- [ ] The header row builds an index map (`level`, `left`, `top`, `width`, `height`, `conf`, `text`), and
      the parse reads through it.
- [ ] A TSV with no header, or one whose header lacks a needed column, falls back to today's fixed indices
      rather than failing - a recognizer that stops working is worse than one reading a stable layout.
- [ ] A test feeds a TSV with an extra column inserted before `left` and asserts the boxes are still right.

## Notes

Cheap and self-contained; no behaviour changes for any Tesseract build in use today.
