# OCR overlay: line-confidence gate + proximity clustering

**Status:** Implemented
**Priority:** 2
**Date:** 2026-07-01

## What / why

When OCR-translating an image that mixes a heading, a picture and a body of text (a typical
illustrated page), the overlay currently misbehaves in three ways: the picture gets buried under an
opaque text plate, one visually-uniform block of text is chopped into several plates with visible
seams, and the heading font sometimes balloons. All three come from the same cause - the recognizer's
raw block/paragraph boxes are turned into plates as-is, and that segmentation is unstable: the engine
groups a picture into a text paragraph (so the plate spans it), splits continuous prose into several
paragraphs, and hallucinates tall low-confidence "text lines" out of the drawing (which inflate the
font). This ticket makes the overlay group **confident text lines by proximity** instead of trusting
raw paragraphs, so a plate covers a coherent column of real text and never a figure or a blank gap.

Repro image (355x563): heading "A Farmer and His Wife", a central illustration, then a story. The
extension's `tesseract.js` merges heading + illustration + upper story into one paragraph whose opaque
plate blankets the whole drawing; the six illustration "lines" score confidence 0-48 while every real
line scores 82-97 - a clean separation the fix exploits.

## Approach

Shared, identical logic on both editions:

1. **Flatten to lines.** Take the recognized hierarchy down to text lines; each line carries its bbox,
   concatenated word text, and mean word confidence.
2. **Confidence gate.** Drop any line whose mean word confidence is `< OCR_MIN_LINE_CONF (50)` or whose
   text is blank. This removes the noise "text" hallucinated from imagery before it can cover a figure
   or inflate a font.
3. **Proximity clustering.** Walk the surviving lines in reading order and grow one plate while the
   vertical gap to the next line stays within `OCR_CLUSTER_GAP_FACTOR (1.2) x` the median line height
   and the lines horizontally overlap (same column); a larger gap (a figure, a section break, a new
   column) starts a new plate. A plate's bbox is the **union of its line boxes**; its font-size stays
   median line height `x 0.85` (`fontFitFactor` / `FONT_FIT`, unchanged).
4. **Text filter unchanged.** `isTranslatable` still runs on the assembled plate text as the final
   guard.

Net effect on the repro: heading plate + one body plate + fully-visible illustration.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` | `internal/ocr/tesseract.go` `parseTSV` rewritten to line clustering |
| GUI (`doc-html-ui`) | `[x]` | inherits the pipeline; no flag change |
| MSIX Store app | `[x]` | inherits the GUI; no packaging impact |
| Browser extension | `[x]` | `extension/src/ocr-overlay.js` `recognize` ported to identical logic |
| Website / docs | `[ ]` | Declined: no user-facing copy change (behaviour-only quality fix) |

## Shared invariants touched

- **Plate granularity** (docs/PARITY.md): was "one plate per paragraph"; now "one plate per proximity
  cluster of confident text lines". Updated in PARITY.md.
- **Plate geometry**: plate bbox is now the union of member line boxes (was the paragraph bbox); font
  metric and `0.85` fit factor unchanged.
- **Noise filter**: adds a line-level confidence gate alongside `isTranslatable`.
- **New shared constants** (added to PARITY.md + guarded by `tests/parity_test.go`):
  `OCR_MIN_LINE_CONF = 50`, `OCR_CLUSTER_GAP_FACTOR = 1.2`.

## Cross-references

- Go: [`internal/ocr/tesseract.go`](../../internal/ocr/tesseract.go) `parseTSV`, `clusterLines`,
  `ocrMinLineConf`, `ocrClusterGapFactor`.
- JS: [`extension/src/ocr-overlay.js`](../../extension/src/ocr-overlay.js) `recognize`, `collectLines`,
  `clusterLines`, `OCR_MIN_LINE_CONF`, `OCR_CLUSTER_GAP_FACTOR`.
- Guard: [`tests/parity_test.go`](../../tests/parity_test.go) `TestParityOCRClustering`.

## Done criteria

- [x] Every edition row above is `Done` or `Declined (reason)`.
- [x] `docs/PARITY.md` updated (granularity, geometry, noise filter, new constants).
- [x] Go green (`go test ./internal/ocr ./tests`, incl. new `TestParityOCRClustering`); `npm test` green (33/33); `go vet` + `gofmt` + lint clean; `go build ./...` OK.
- [x] Changelog entry in `DEV/CHANGELOG.md`.
- [x] Verified on the repro image via the real `tesseract.js`: 2 plates (heading + one body), illustration fully visible.
