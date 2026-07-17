# OCR page-segmentation mode parity (extension junk-text fix)

**Status:** Implemented
**Priority:** 1
**Date:** 2026-07-02

## What / why

OCR-translating a full-page illustration in the browser extension produced junk: on a page whose only
real text is one clean speech bubble ("Ugh.. can't school be optional just this once"), the overlay
read "< = Ugh.. can't school be | 4 optional just this once" plus a duplicate plate, stamping garbled,
opaque plates over the artwork. The desktop app reads the same page cleanly. Root cause: the two
editions ran Tesseract in **different page-segmentation modes**. The desktop CLI uses its default
PSM 3 (AUTO), which runs layout analysis and isolates the bubble; the extension inherited
`tesseract.js`'s default **PSM 6 (SINGLE_BLOCK)**, which forces the whole 8-megapixel render to be read
as one uniform text block, so scene edges become stray glyphs woven into the dialogue and regions
mis-merge. The line-confidence gate can't recover it because the junk rides *inside* otherwise-confident
lines. Fix: pin **PSM 3** on both editions so recognition matches; the extension now sets it explicitly,
the desktop makes its existing default explicit. This closes a hidden parity gap - the earlier
line-clustering and upscale tickets were tuned assuming both engines recognize identically, which was
false while the PSM differed.

Repro: `C:\TEMP\Becoming A Milf Chapter 1.pdf`, page 1 (one 3840x2100 DCTDecode render, no text layer).
Verified with the extension's own bundled `tessdata_fast 4.0.0` eng model via the Tesseract 5.4.0 CLI:
`--psm 6` reproduces the exact junk from the browser; `--psm 3` returns the two clean bubble lines and
nothing else.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` | `internal/ocr/tesseract.go`: new `ocrPageSegMode = 3`, passed as `--psm 3` (was implicit CLI default; now explicit + guarded). Behaviour unchanged. |
| GUI (`doc-html-ui`) | `[x]` | inherits the pipeline; no flag change |
| MSIX Store app | `[x]` | inherits the GUI; no packaging impact |
| Browser extension | `[x]` | `extension/src/ocr-overlay.js`: new `OCR_PSM = "3"`, applied via `worker.setParameters({ tessedit_pageseg_mode })` (was tesseract.js default PSM 6). This is the actual behaviour fix. |
| Website / docs | `[ ]` | Declined: behaviour-only quality fix, no user-facing copy change |

## Shared invariants touched

- **New shared invariant** (added to `docs/PARITY.md` OCR table + the aligned list, guarded by
  `tests/parity_test.go` `TestParityOCRClustering`): page-segmentation mode = **PSM 3 (AUTO)** on both
  editions (`ocrPageSegMode` == `OCR_PSM`).
- No change to the existing OCR constants (confidence gate, cluster gap, upscale) or to any default.

## Cross-references

- Go: [`internal/ocr/tesseract.go`](../../../internal/ocr/tesseract.go) `ocrPageSegMode`, `Recognize`
  (`--psm`).
- JS: [`extension/src/ocr-overlay.js`](../../../extension/src/ocr-overlay.js) `OCR_PSM`, `getWorker`
  (`setParameters`).
- Guard: [`tests/parity_test.go`](../../../tests/parity_test.go) `TestParityOCRClustering` ("page-seg mode").

## Done criteria

- [x] Every edition row above is `Done` or `Declined (reason)`.
- [x] `docs/PARITY.md` updated (new OCR row + aligned-list bullet) and guard extended.
- [x] Go green (`go test ./internal/ocr ./tests`, incl. `TestParityOCRClustering`); `npm test` green;
  `go vet` + `go build ./...` OK.
- [x] Changelog entry in `DEV/CHANGELOG.md`.
- [x] Engine-level verification on the repro image with the extension's bundled model: `--psm 3` clean,
  `--psm 6` reproduces the junk. Final browser visual check (Reload unpacked, reopen the PDF) is the
  reader's acceptance step.
