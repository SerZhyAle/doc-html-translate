# Pointer: CAPTURE-OUTPUT

- **Id:** `CAPTURE-OUTPUT`
- **Version:** 0.2 draft
- **Home:** the shared contracts catalog, `capture-output/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - a reader of the `documents` kinds (`text`, `ocr_text`, `translation`) through the TXT input path; produces no file of any kind in rule 1's table

0.2 (2026-09-29) added the `stream_video` / `stream_audio` kinds, the grow-in-place rule 12 and the
source-container rule 13; nothing of it touches a producer that writes no capture, so the obligations
below are unchanged. Re-read with ticket 69.

What this repo owes, and where it is held:
- **Rule 14, second sentence** (a reader accepts CRLF and a byte-order mark on input): the Go reader strips the UTF-8 and both UTF-16 marks and normalizes CR / CRLF (`internal/txt/extract.go`, `decodeText` and `parseParagraphs`); the extension does the same (`extension/src/txt.js`, `decodeText` and `splitParagraphs`). Both editions decode by the one ladder [`../PARITY.md`](../PARITY.md) pins.
- **Section 3** (a reader never parses a kind out of a name it did not write): the `<prefix>_<yyMMdd>_<HHmmss>` stem becomes the page title verbatim (`txtTitle`); nothing dispatches on the prefix.
- **Rule 15** (the `=== TRANSLATION ===` / `=== ORIGINAL ===` layout): no reader obligation is written. Such a file converts as ordinary paragraphs with the marker lines kept; rendering the two halves as two sections would be an opt-in supplement, not a debt.

Out of the contract's scope by its own section 5, so not measured here: the converted document set (`index.html`, pages, assets), the extension's "Save as HTML" and "Download original", the diagnostic archive (`DIAGNOSTIC-REPORT`), and the run logs and OCR diagnostics under the app's log store, which are not user-facing captures. Rule 14's first sentence (the writer side) is not exercised: the product writes no `.txt`.

**Conformance.** No vectors in the catalog yet (section 4 of the contract). Held by `internal/txt` (`TestExtractStripsUTF8BOM`, `TestExtractDecodesUTF16`, `TestExtract_CRLFLineEndings`, `TestExtract_OldMacCRLineEndings`) and by `extension/test/txt.test.mjs` with `legacy-text.test.mjs`, all run by `scripts/test.ps1`.
