# OCR: upscale small images before recognition

**Status:** Implemented
**Priority:** 3
**Date:** 2026-07-01

## What / why

Tesseract reads small, low-resolution images (thumbnails, small scans) poorly - it misses or garbles
text that would be legible at a larger size. Before running OCR, enlarge a small image so the engine
has more pixels to work with, then map the recognized coordinates back so the translatable-text overlay
still lands on the original picture. This is a best-effort quality lift for the image-OCR feature; it
never changes what the user sees except that more real text gets recognized on low-res inputs.

Measured on a 640x359 thumbnail: the desktop/CLI engine went from recognizing nothing usable to reading
the headline word cleanly (confidence 97) once the image was enlarged 2x. (Very large "poster" display
type remains a Tesseract limitation that upscaling does not fix.)

## Approach

Shared, identical on both editions: if the image's long side is below `OCR_UPSCALE_BELOW (1000 px)`,
enlarge it `OCR_UPSCALE_FACTOR (2x)` with a high-quality resample, OCR the enlarged copy, then divide all
recognized coordinates (and the reported dimensions) back by the factor. Larger images are recognized
as-is. Best-effort: any decode/scale failure falls back to the original image. The scale-back keeps the
rest of the pipeline (clustering, colour sampling, percent-positioned plates) untouched - it all runs in
original pixel space.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` | `internal/ocr/tesseract.go` `upscaleForOCR` (temp PNG via `x/image/draw`) + `scaleDown` |
| GUI (`doc-html-ui`) | `[x]` | inherits the pipeline |
| MSIX Store app | `[x]` | inherits the GUI; temp file uses the OS temp dir |
| Browser extension | `[x]` | `extension/src/ocr-overlay.js` `upscaleForOcr` (canvas 2x) + coord scale-back in `collectLines` |
| Website / docs | `[ ]` | Declined: behaviour-only quality lift, no user-facing copy |

## Shared invariants touched

- **New shared constants** (added to docs/PARITY.md + guarded by `tests/parity_test.go`
  `TestParityOCRClustering`): `OCR_UPSCALE_BELOW = 1000`, `OCR_UPSCALE_FACTOR = 2`.
- New "Pre-OCR upscale" row in the PARITY.md OCR table.

## Cross-references

- Go: [`tesseract.go`](../../internal/ocr/tesseract.go) `Recognize`, `upscaleForOCR`, `scaleDown`,
  `ocrUpscaleBelow` / `ocrUpscaleFactor`.
- JS: [`ocr-overlay.js`](../../extension/src/ocr-overlay.js) `recognize`, `upscaleForOcr`, `collectLines`
  (scale arg), `OCR_UPSCALE_BELOW` / `OCR_UPSCALE_FACTOR`.

## Done criteria

- [x] Every edition row above is `Done` or `Declined (reason)`.
- [x] `docs/PARITY.md` updated (new row + constants) and guard extended.
- [x] Go green (`go test ./internal/ocr ./tests`, new `upscale_test.go` + guard); `npm test` green; vet/gofmt/lint clean; `go build ./...` OK.
- [x] Verified via the real tesseract.js: image1 (563 px long side) upscales with an identical 2-plate result (no regression); low-res inputs get more pixels for recognition.
- [x] Changelog entry in `DEV/CHANGELOG.md`.
