# PDF conversion drops short lines and shows some images wrong or not at all

**Status:** BlockNeedUserTest - implemented 2026-09-26; left: `go test ./internal/pdf/` on Windows with the vendored pdftotext
**Priority:** 80
**Date:** 2026-09-26

> Filed by the pre-release audit, [ticket 34](34_2026-09-25_full-code-audit-pre-release.md). Finding ids
> refer to its register, [`FINDINGS.md`](34_2026-09-25_full-code-audit-pre-release/FINDINGS.md).

## 1. Problem

- **X25 (high)** - the ligature-artifact filter (4 or more words averaging under 3 letters) silently
  drops real short text on the pdftotext path: "Text of page 2", "Is it so? I do.", short verse and
  dialogue. It also makes `TestExtract_Volume3ImagesOnTheirPages` fail wherever the vendored
  pdftotext exists, so `scripts/test.ps1` is red on the owner's machine. The pdflib path has no such
  filter; the extension filters per row.
- **X26** - Flate CMYK and Indexed-CMYK rasters are written as `.tif`, which Chrome cannot display, and
  are flipped on disk and again by CSS; OCR reads the flipped file.
- **X27** - the desktop prefers the raster without a stencil `/Mask` in a same-shape group (the MRC fix);
  the extension keeps the largest, and docs/PARITY.md does not record the rule.

## 2. Goals

1. The artifact filter keeps real short lines on every path, in both editions alike.
2. Every extracted image is in a format Chrome shows, stored the right way up, flipped at most once.
3. The `/Mask` preference is ported, or recorded as a divergence with its consequence.

## 3. Constraints

- Parity pair (`internal/pdf` with `reflow.js` and `pdf-images.js`): one ticket, both editions.
- Output-format change: covered by the completion record's rebuild rule.

## 4. Acceptance

- `go test ./internal/pdf/` passes with the vendored pdftotext present, output cited.
- A CMYK fixture renders in headless Chrome, upright.

## Implementation (2026-09-26)

Done in a Linux cloud container. The pdftotext evidence comes from the vendored binary itself: the
`internal/bundledtools/pdftotext/` set (Xpdf 4.00, Git for Windows build) was run under wine 9.0 through a
`pdftotext` wrapper put first on `PATH`. The owner confirms the same run natively on Windows (step at the end).

### X25 - ligature filter keeps real short lines (both editions)

- New signature, the same in both editions: a row is an artifact only when it has `>= 4` tokens, every
  token is a letters-only fragment of `<= 2` letters, and the distinct fragments are `<= 0.5` x the
  tokens ("if lf if if if if if"). Short average word length alone is gone. Accepted residue: a bare run of
  repeated one- or two-letter words with no punctuation ("no no no no") still reads as an artifact.
- Go: `internal/pdf/extract.go` - `isLigaturesArtifact` + `isLigatureFragment`, constants
  `ligatureMinWords` / `ligatureFragmentMaxLen` / `ligatureMaxDistinctRatio` (the old
  `ligatureMaxAvgWordLen` is removed). The filter now also runs per row on the pure-Go path
  (`rowsToText`), as it does in the extension, so all three text paths apply one rule.
- JS: `extension/src/reflow.js` - `isLigaturesArtifact` with `LIGATURE_MIN_WORDS` /
  `LIGATURE_FRAGMENT_MAX_LEN` / `LIGATURE_MAX_DISTINCT_RATIO` (code points, `\p{L}`, so it counts as Go does).
- Shared case table `tests/testdata/ligature_artifact_cases.json` (17 cases) drives
  `internal/pdf/ligature_test.go` `TestIsLigaturesArtifactSharedCases` and
  `extension/test/reflow.test.mjs` "isLigaturesArtifact: shared Go/JS fixture".
  `tests/parity_test.go` `TestParityReflowConstants` pins the three constant pairs and fails if either
  unit test stops reading the table. `docs/PARITY.md` "PDF reflow heuristics" records the rule.
- The pdftotext-path filter is unit-testable without pdftotext: `TestParsePDFLayoutPage_KeepsShortRealLines`
  (a `-layout` page string) and `TestExtractWithPDFToText_KeepsShortTextPages` (stub pdftotext emitting
  Xpdf-shaped CRLF output). `TestRowsToText_DropsOnlyLigatureRows` covers the pure-Go path; the JS test
  "reflowPage: short dialogue rows survive, fragment rows do not" covers the extension.
- `TestExtract_Volume3ImagesOnTheirPages`: its fixture had been reworded to "Paragraph text on page N of
  the document." (6c48a90), which hid the bug. It is back to "Text of page N", the text the audit saw fail.
- Evidence with the vendored Xpdf 4.00 (under wine) first on `PATH`:
  before the fix `--- FAIL: TestExtract_Volume3ImagesOnTheirPages` `images_test.go:131: spine has 4 pages,
  want 10` (the audit's exact failure), plus the new tests failing; after the fix
  `--- PASS: TestExtract_Volume3ImagesOnTheirPages`, `ok doc-html-translate/internal/pdf`.
  Extension before the fix: `not ok 8 - isLigaturesArtifact: shared Go/JS fixture`,
  `not ok 9 - reflowPage: short dialogue rows survive..`; after: pass.

### X26 - CMYK rasters are PNG, upright, never flipped

- Orientation was established, not assumed: a fixture PDF with a Flate DeviceCMYK raster (red band first in
  the stream, blue second) renders red over blue in Poppler's `pdftoppm`, so PDF image samples are stored top
  row first, and pdfcpu writes them in that order. The raster needs no flip at all; the old on-disk flip plus
  CSS `scaleY(-1)` was wrong on both counts. (For plain CMYK the on-disk flip never even ran:
  `golang.org/x/image/tiff` rejects the CMYK colour model - "tiff: unsupported feature: color model" - so
  the file stayed a `.tif` Chrome cannot show; a soft-masked one decoded as NRGBA and was flipped.)
- `internal/pdf/images.go`: new `tiffAsPNG` re-encodes a pdfcpu TIFF raster (DeviceCMYK, Indexed over
  DeviceCMYK, with or without soft mask) as PNG in memory before the file is written, so it lands on disk as
  `.png` and is named by the same collision-safe `imageFileName`. It decodes with pdfcpu's own TIFF package
  `github.com/hhrutter/tiff` (already linked through pdfcpu; `go.mod` now lists it as a direct requirement),
  after the pixel-budget check `limits.CheckTIFF`. On failure the `.tif` is written as before, with a warning.
  `flipImageFileVertically` / `flipRowsInPlace` are removed.
- `internal/pdf/extract.go`: `imageHTMLClassAttr` and both `.pdf-flip-y` CSS rules are removed, so nothing
  flips an image in the page any more. OCR reads the same upright PNG the page shows.
- Tests: `internal/pdf/cmyk_test.go` `TestExtractImages_CMYKRasterIsUprightPNG` (DeviceCMYK and IndexedCMYK:
  a `.png`, red over blue, no `.tif` left, no flip in the page; failed before with `image written as
  "pdf_images/DeviceCMYK_1_Im0.tif", want a .png Chrome can show`) and `TestTIFFAsPNGRefusesOverBudget`
  (replaces the old flip budget test). `flip_test.go`, `TestFlipImageFileVertically` and
  `TestImageHTMLClassAttr` are deleted with the code they tested.
- Headless Chrome (`chromium-1194`): both fixtures converted with the CLI (`-noopen -force`) give
  `pdf_images/<name>_1_Im0.png`, `<img src="pdf_images/cmyk_1_Im0.png" loading="lazy"/>`, no `flip` in the
  page, and the screenshots show red over blue, the same as `pdftoppm`.
- `docs/PARITY.md` "Input limits": a PDF CMYK raster over the budget stays a `.tif`.
- Rebuild: there is no output-format version constant; the completion record does not compare the tool
  version, so an existing output with `.tif` images is reused until the source or options change or
  `-force` is used. No precedent bumps anything for an extractor fix, so nothing is bumped here.

### X27 - `/Mask` preference: recorded as a divergence, not ported

- Not portable with what the extension has: pdf.js 6.2.108 marks an image with `/SMask` and one with `/Mask`
  identically in the operator list (`addImageOps`, `hasMask = SMask || Mask`) and hands over only pixels,
  while the Go rule must not fire on `/SMask`. It would need a raw PDF object reader beside pdf.js.
- Consequence (recorded in `docs/PARITY.md` "PDF page-image selection"): pdf.js applies the stencil as alpha
  (`PDFImage.fillOpacity`), so the extension keeps the MRC foreground layer, not a smear but lettering on
  transparency; the background layer is missing, lettering sits on the theme page colour, and OCR runs on
  that raster. Reasoned from the vendored pdf.js source, not measured on an MRC file in the extension.
- The PARITY port map and prose now place `selectPageImages` / `sameShapeRaster` / `betterPageRaster` in
  `internal/pdf/images.go`; the `pdf-images.js` comments point there and name the unported rule.

### Checks

- `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...`: clean. `gofmt -l` on touched Go files: clean.
- `go test ./...`: every package `ok` (Poppler 24.02 `pdftotext` also on `PATH` in the container).
- `cd extension && npm test`: `# tests 273`, `# pass 273`, `# fail 0` (baseline 271 + 2 new).

### For the owner on Windows

- Run `./scripts/test.ps1` (or `go test ./internal/pdf/`) with the vendored pdftotext present and confirm
  `TestExtract_Volume3ImagesOnTheirPages` passes natively; the stub-based pdftotext tests skip on Windows.
- Optional: convert a real CMYK PDF and look at the page in Chrome.
