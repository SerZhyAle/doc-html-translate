# PDF raster extraction takes whatever it finds: thumbnails, duplicates, or nothing

**Status:** Implemented
**Priority:** 8
**Date:** 2026-07-17

## What / why

Converting a PDF with no text layer, the reader is shown *the embedded rasters* of each page - on the
assumption that a page's raster is the page. For a plain scan that holds. For anything else it does
not, and the failures are silent and user-visible. Desktop-only in origin (`internal/pdf` extraction),
but the extension renders PDF pages itself and should be checked against the same three fixtures
before this is called done.

Found converting the new single-page fixtures (2026-07-17). Three shapes, three fixtures, all in
`test_doc`:

### 1. A page thumbnail is mistaken for the page

`pdf-1page-tiny_NASA-Quaoar-sky-chart.pdf` (50 KB, vector chart, labels as outlines, no text layer)
extracts exactly one raster: `.._1_thumb.png`, **128x104**. PDF pages may carry a `/Thumb` entry - a
preview thumbnail - and we take it as content. The reader gets:

```html
<div class="pdf-images pdf-page-scan" style="width:min(128px,96vw,113.2vh)">
  <img src="pdf_images/pdf-1page-tiny_NASA-Quaoar-sky-chart_1_thumb.png"/>
```

A 128-pixel postage stamp where the chart should be, plus zero text. Note the sizing code is doing
its job correctly here - `pageScanBox` clamps the box to the image's own pixel width so a raster is
never upscaled into mush. It faithfully renders a thumbnail at thumbnail size. The bug is upstream,
in *what was extracted*.

### 2. One page, two rasters, page shown twice

`pdf-1page-blackletter_Plague-Proclamation-1625.pdf` is one page and logs `Images: 2 extracted across
1 pages`: `.._1_I1.jpg` (1455x2065) and `.._1_Im001.jpg` (4363x6193) - the same scan at two
resolutions. Both are emitted, so the reader scrolls the same proclamation twice. Nothing in the log
suggests anything is wrong; `2 extracted across 1 pages` reads like success.

### 3. A vector page with no text layer yields nothing to read

The general case behind (1): when a page's content is vector drawing and its text is outlined, there
is no page raster to extract and no text to pull, so the output is an empty page - or a thumbnail, if
one happens to be embedded. We never rasterize the page ourselves. Whether that is worth doing is a
product call; today the failure is silent either way, which is not.

## Where to look

- Extraction: [`internal/pdf/extract.go`](../../../internal/pdf/extract.go) `writePDFImages` -
  `pdfcpulib.ExtractPageImages(ctx, pageNum, false)` returns every image associated with the page.
  There is a `singleImgPerPage := len(imgs) == 1` branch already, so the multi-image case is known to
  exist; it affects naming, not selection.
- Page assembly: `buildPDFPageHTML` / `buildPageHTML` emit one `<img>` per extracted raster and pick
  the `pdf-page-scan` box only when `len(images) == 1`.

Selection needs a rule. Worth weighing in the tactical pass: drop `/Thumb` images outright; when a
page yields several rasters, prefer the one whose dimensions plausibly cover the page rather than
emitting all of them; and decide explicitly what to do when a page yields no usable raster (say so,
or rasterize).

## Interaction

Item 1 and 2 both change *which* image OCR runs on, so they sit upstream of
[`2026-07-17_ocr-upscale-threshold-misses-page-scans`](../2026-07-17_ocr-upscale-threshold-misses-page-scans.md).
A 128x104 thumbnail is under the upscale gate and *would* be enlarged 2x - to 256x208, still
unreadable. Fixing the gate does not fix this; fixing this changes what the gate sees.

## Done criteria

- [x] A `/Thumb` preview is never emitted as page content. `selectPageImages` drops `img.Thumb`
      ([`internal/pdf/extract.go`](../../../internal/pdf/extract.go)).
- [x] A page that yields several rasters produces one page in the output, not one per raster.
      Proportional-scale duplicates (aspect ratios equal within `aspectRatioTolerance` 1%) collapse to
      the largest; a NOTE reports the drop so it is not silent.
- [x] A page with nothing readable says so rather than emitting an empty page or a stamp. With the
      thumbnail dropped the NASA chart yields no image, so the existing no-text fallback embeds the
      original PDF with its "no extractable text layer" note - the reader gets the real vector chart.
- [x] All three fixtures above convert to something a reader would accept - checked by looking at the
      output, not at the log. Chrome-verified: NASA -> `<embed application/pdf>` + `original.pdf` on disk;
      Plague -> `total=1 render=1 broken=0`.
- [x] Extension checked against the same three fixtures. It mines paint operators, so `/Thumb` never
      reaches it and a vector page is rasterized whole; the duplicate case *could* show both XObjects, so
      the same `dedupeSameShape` is ported to [`pdf-images.js`](../../../extension/src/pdf-images.js).
- [x] Tests / gate green (`scripts/test.ps1`) - all packages + the 177s integration suite; lint green;
      extension `npm test` 50/50.
- [x] Changelog entry in `DEV/CHANGELOG.md` (2026-07-17 21:56).

## Implementation notes

- The dimensions were the trap: pdfcpu's real (non-stub) `ExtractPageImages` leaves `Width/Height` at
  zero (only the stub pass reads them from the image dict), so the dedup silently never fired until a
  cheap second stub pass enriched each image's dimensions before `selectPageImages`.
- Composed pages (differently-shaped images) are deliberately left intact; controls confirmed no
  regression on a single scan, a 6-page comic (one image per page), and a 319-page book.
- New shared invariant "PDF page-image selection" in [`docs/PARITY.md`](../../../docs/PARITY.md); the
  thumbnail and vector-page halves are recorded there as intentional-by-capability asymmetries.
