# Strategic spec: 2026-09-24_bugfix-pdf-extraction-accuracy - PDF images on the right page, no pages lost

**Ticket:** 2026-09-24_bugfix-pdf-extraction-accuracy
**Status:** Draft
**Priority:** 65
**Date:** 2026-09-24
**Tier:** Easy
**Tactical plan:** `DEV/plan/2026-09-24_bugfix-pdf-extraction-accuracy/` (created by /spec-tech)
**Findings:** X6 X8 Q6 Q7 (see `../README.md`)

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
No open questions.

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
`/spec-tech 2026-09-24_bugfix-pdf-extraction-accuracy`
