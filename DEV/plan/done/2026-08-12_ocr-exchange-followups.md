# The overlay is graded on stability, not on being in the right place

**Status:** Implemented
**Priority:** 48
**Date:** 2026-08-12

> Cross-edition feature ticket. One feature = one ticket covering every edition.
> Read [`docs/PARITY.md`](../../../docs/PARITY.md) before starting; update it when a shared invariant moves.

## What / why

Answering the FastMediaSorter Lite positioning exchange
(`FastMediaSorter_Lite/docs/specifications/SPECIFICATION_DOC2HTML_OCR_POSITIONING_EXCHANGE.md` §13)
against the shipped code turned up more than answers. Two defects were found and fixed while
writing the reply; what is left here is the reason they were able to ship at all, plus the gaps the
reply had to record as "not done".

Evidence and the full answers:
[`DEV/research/ocr_positioning_exchange_2026-08-12.md`](../../research/ocr_positioning_exchange_2026-08-12.md).

**The root item is the first one.** The acceptance gate grades the position dimension on *drift* -
how far a plate moves between viewports - and on nothing else. The navbar defect fixed on
2026-08-12 put every plate on a picture 1.9x the size it belonged on, and drifted **0 px** at every
viewport, because it was equally wrong at all of them. A stability measure cannot see a systematic
offset, and the lab already computes the measure that can (`metrics.IoU`, `metrics.Edges`) without
grading anything on it.

## Items

| # | What | Where |
|---|---|---|
| 1 | **Done.** Gate absolute position, not only drift: `DimPosition` in `gate.go` and `DEV/ocrlab/thresholds.json` evaluates an IoU floor (`min: 0.77` overall, baseline 0.7756; category bounds `comic: 0.75`, `texture: 0.74`). Verified by `TestGateFailsPreFixNavbarDefect` showing systematic offset causes gate failure while drift=0 passed. | [`DEV/ocrlab/thresholds.json`](../../ocrlab/thresholds.json), [`tools/ocrlab/report/gate.go`](../../../tools/ocrlab/report/gate.go) |
| 2 | **Done 2026-08-15, and the premise above it was wrong.** `print-color-adjust:exact` is on `.ocr-box` and `.ocr-plate`, pinned by `TestParityOCRPrintPlate`. Measured through `Page.printToPDF(printBackground:false)` - the print dialog's own path, backgrounds unchecked - on `img-png_Nyoka-comic-page`: the plate does **not** print transparent over legible source lettering. Chromium repaints it **white** and darkens its text, so the sheet stays readable and stops matching the artwork: **20 of 20 sampled plate papers forced to white** and 14 ink colours darkened without the declaration, 0 with it. Extension, same instrument on a harness carrying the shipped stylesheet: 3 of 3 forced, 0 with it. Chrome's `--print-to-pdf` switch cannot see the difference - it never prints backgrounds and ignores the opt-in | `overlay.go: ocrCSS`, `extension/src/ocr-overlay.css` |
| 3 | **Decided & Documented.** Vertical writing is unsupported by design in both editions: `jpn_vert` recognizes text, but the plate is a horizontal flex container and the clustering assumption (vertical pitch between horizontally overlapping lines) is wrong by construction for vertical text. Documented in `docs/PARITY.md`. | `tesseract.go: clusterLines`, `ocr-cluster.js`, [`docs/PARITY.md`](../../../docs/PARITY.md) |
| 4 | **Documented.** No CJK scene and no RTL scene with human ground truth in the corpus; `synth-rtl-layout` is a direction proxy and says so. Short CJK runs bypass minimum-length/vowels in `isTranslatable`; RTL plates inherit the document's DOM `dir` without script re-sorting and are verified not to clip/drift under `rtl-arabic` stress. Documented in `docs/PARITY.md`. | [`DEV/ocrlab/corpus.json`](../../ocrlab/corpus.json), `DEV/ocrlab/annotations/`, [`docs/PARITY.md`](../../../docs/PARITY.md) |
| 5 | **Done 2026-09-12.** `synth-two-columns` word-gap split (`ocrMaxWordGapRatio = 3.5` / `OCR_MAX_WORD_GAP_RATIO = 3.5`) and column regrouping (`orderColumns`) in `tesseract.go` and `ocr-cluster.js` fixed the merge in both editions (1 plate crossing gutter -> 2 plates, one per column). Pinned by `TestParityOCRClustering` and ticket `done/2026-09-12_ocr-line-stitched-across-the-picture.md`. | `tesseract.go: clusterLines`, `ocr-cluster.js`, [`docs/PARITY.md`](../../../docs/PARITY.md) |
| 6 | **Done 2026-08-15.** Every `createImageBitmap` in `ocr-overlay.js` now takes `BITMAP_OPTS = { imageOrientation: "from-image" }` - the recognizer's bitmap, the colour sample and both grey rungs, because one bare call is enough to put the plates in a different space from the picture. `TestParityOCRExifOrientation` pins the constant and fails on any bare call; `docs/PARITY.md` "Display space is the only coordinate space" carries the reason | `extension/src/ocr-overlay.js` |
| 7 | **Documented.** EXIF is read from JPEG only in Go (`internal/ocr/exif.go`); PNG `eXIf` and WebP can carry the same tag, but camera photos are JPEG in practice. The extension uses browser decoding (`createImageBitmap`). Documented gap in `docs/PARITY.md`. | [`internal/ocr/exif.go`](../../../internal/ocr/exif.go), [`docs/PARITY.md`](../../../docs/PARITY.md) |

## Done when

- The gate fails a run whose plates are systematically off their annotated groups, and that failure
  is demonstrated by re-scoring the pre-fix evidence of the navbar defect (item 1).
- Items 2-7 are each either implemented with a measurement, or recorded in
  [`docs/PARITY.md`](../../../docs/PARITY.md) as an intentional difference with the reason.

## Not in scope

Re-deriving any existing bound. The thresholds file rests on 11 annotated dev scenes with no
holdout and says so; that is the visual-fidelity lab ticket's Phase 07, not this one.
