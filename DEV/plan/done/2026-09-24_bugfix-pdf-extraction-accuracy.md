# Strategic spec: 14_2026-09-24_bugfix-pdf-extraction-accuracy - PDF images on the right page, no pages lost

**Ticket:** 14_2026-09-24_bugfix-pdf-extraction-accuracy
**Status:** Implemented
**Priority:** 65
**Date:** 2026-09-24
**Tier:** Easy
**Tactical plan:** `DEV/plan/14_2026-09-24_bugfix-pdf-extraction-accuracy/` (created by /spec-tech)
**Findings:** X6 X8 Q6 Q7 (see the [findings register](../../research/audit_2026-09-24/README.md))

> **Scope:** STRATEGIC.

---

## 1. Problem
- **Images on the wrong page:** PDF images are assigned to pages by parsing the file names the image extractor produces. The parser takes the first number it finds, so a PDF named `Volume_3.pdf` puts every image on page 3.
- **Missing pages:** trailing image-only pages, such as a back cover or scanned appendix plates, are dropped, because the page count is taken from the text extractor's trimmed output.
- **Non-Windows builds:** the PDF path tries to run a `.exe` and suggests a Windows-only installer. One PDF test hard-codes a Windows path and fails on Linux, which keeps the test suite red outside Windows.

## 2. Goals
1. Every extracted image lands on the page it came from, whatever the PDF's file name.
2. The HTML output has exactly as many pages as the PDF, including image-only pages at the end.
3. On non-Windows builds the PDF path uses a helper appropriate to the platform (or the pure-Go fallback), with platform-appropriate advice.
4. The PDF test suite passes on every OS the tests run on.

**Non-goals:**
- Text-layout reflow quality.

## 3. Wishes and constraints
### 3.2 Hard constraints
- **Platform / versions:** Windows is primary; the test suite should be green on Linux for CI and cloud sessions.
- **Data compatibility:** n/a.

### 3.3 Owner inputs (Approval gate)
- **Related tickets:** `bugfix-external-process-bounds` (helper discovery).
- **Platform constraints:** check the extension's PDF image mapping for the same name-parsing assumption; it likely uses page indices directly, so it is exempt.
- **Validation level:** fixtures named `Volume_3.pdf` and a PDF with two trailing scan pages.

## 4. Current architecture context
Images are dumped to disk by a PDF library whose names embed the source file name and the page
number, and are then parsed back. The page total passed to the image pass comes from the text
pass. Helper discovery is written for Windows only.

## 5. Proposed approach
### 5.1 Pillars / modules
- **Authoritative page mapping:** the page number comes from the library's own page iteration or from a name parser anchored at the known prefix, never from the first number found.
- **Authoritative page count:** the count comes from the PDF itself, and pages with no text but with images are kept.
- **Platform-aware helper lookup:** the executable name and install advice match the OS.
- **Portable tests:** path-shaped test inputs use the platform's separator, or are skipped when not on that platform.

## 6. Open questions / research items
No open questions. Owner decisions applied:

- **Decided: page mapping.** The page number of an extracted image comes from the library's own page iteration, recorded when the file is written; nothing parses a file name back.
- **Decided: page count.** The PDF's own page count (pdfcpu) is used only to extend the text extractor's page list, so trailing image-only pages are kept; it never shrinks the list.
- **Decided: non-Windows helper.** The helper is `pdftotext` looked up on PATH (plus the usual install locations); nothing is bundled or installed there, the advice names Poppler / `poppler-utils`, and the pure-Go fallback is kept.
- **Decided: Q6.** `TestPdfTitle` is portable (already fixed on this branch by `3fa32e0`, which makes the title split on either separator).

## 7. Risks
- **The page count from the library disagrees with pdftotext on malformed PDFs.** Likelihood: low. Impact: an empty page at the end. Mitigation: use the library count only to extend the page list, never to shrink it.

## 8. User impact (docs)
No changes to user docs.

## 9. Architecture decisions (ADR)
No ADRs. The decision follows established project patterns.

## 10. Links to other specs
`bugfix-external-process-bounds`.

## 11. Done criteria (strategic)
1. `Volume_3.pdf` with images on pages 1, 5 and 9 shows them on pages 1, 5 and 9.
2. A PDF with two trailing scanned pages has those pages in the output.
3. `go test ./internal/pdf/...` passes on Linux.

## 12. Next step
`/spec-tech 14_2026-09-24_bugfix-pdf-extraction-accuracy`

## Implementation

- **X6 - images on the right page.** `internal/pdf/images.go` (the image pass, moved out of `extract.go`, which was over the size budget): `writePDFImages` walks pdfcpu's pages and records each written file under the page it came from; `parseImagePageNum` is gone. File names keep pdfcpu's shape (`{source}_{page}_{resource}.{type}`, same sanitizer), with the object number appended only on a name collision. `.jpx` to `.jpg` conversion updates the recorded name.
- **X8 - trailing image-only pages.** `extractImages` now returns the pdfcpu page count with the page map; `extractWithPDFToText` pads the trimmed pdftotext page list up to that count and never below its own length.
- **Q7 - non-Windows helper.** `internal/bundledtools` is split: the embedded Windows binary is built only into the Windows build, and `PDFToTextPath` returns `ErrNotBundled` elsewhere. `internal/pdf/pdftotext*.go` holds the lookup (PATH, then per-OS install locations) and the per-OS advice; on non-Windows a missing pdftotext logs a localized "install Poppler (poppler-utils)" note and never reaches the Windows installer branch.
- **Q6.** Already green on Linux (`3fa32e0`).
- Also: a per-page `recover` around pdfcpu image extraction and one around the whole image pass, so a pdfcpu panic there costs pictures, not the book.

Tests (`go test ./internal/pdf/...` passes on Linux):

- Done 1: `TestExtractImages_PageComesFromThePageNotTheFileName` and `TestExtract_Volume3ImagesOnTheirPages` - a generated `Volume_3.pdf` with images on pages 1, 5 and 9 maps them to pages 1, 5 and 9.
- Done 2: `TestExtractWithPDFToText_KeepsTrailingImageOnlyPages` - five pages, two trailing image-only plates, a stub pdftotext that emits text for three; the output has five pages with the plates. `TestExtractWithPDFToText_UnreadableCountNeverShrinks` covers the "never shrink" rule.
- Done 3: the package suite on Linux.
