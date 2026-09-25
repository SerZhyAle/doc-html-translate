# PDF conversion drops short lines and shows some images wrong or not at all

**Status:** Draft
**Priority:** 80
**Date:** 2026-09-26

> Filed by the pre-release audit, [ticket 34](done/34_2026-09-25_full-code-audit-pre-release.md). Finding ids
> refer to its register, [`FINDINGS.md`](done/34_2026-09-25_full-code-audit-pre-release/FINDINGS.md).

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
