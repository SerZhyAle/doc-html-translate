# Phase 07 — PDF image integration

**Strategic spec:** [`../2026-07-01_ocr-image-overlay.md`](../2026-07-01_ocr-image-overlay.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 06
**Steps done:** 3 / 3

## Objective
Extract raster images from PDF pages (with a scanned-page fallback) and feed them to the same lazy
OCR scheduler so image-borne text in PDFs becomes translatable alongside the reflowed text.

## Prerequisites
- [ ] Phase 06 ✅ Done (lazy OCR scheduler exists and is proven on EPUB).
- [ ] Research `research/01__ocr-stack-decisions.md` Q1 (extraction approach) reviewed.

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `extension/src/pdf-images.js` | New | ≤ 200 |
| `extension/src/viewer.js` | Modified | ≤ 780 |

## Steps

### Step 7.1 — Per-page image extraction
**Files:** `extension/src/pdf-images.js`
**Depends on:** - start of phase

**Prompt for developer:**
> Create `extension/src/pdf-images.js` exporting `extractPageImages(page)` that walks the page operator
> list for `OPS.paintImageXObject` / `OPS.paintInlineImageXObject`, resolves each via `page.objs`/
> `page.commonObjs`, and returns an array of `{ blob_or_bitmap, width, height }`. If the page yields no
> usable images but is image-dominant, rasterize the page (existing pdf.js render) as one fallback
> image. Keep it defensive: skip images below a small size threshold.

**Verification:**
- File `extension/src/pdf-images.js` exists.
- `export async function extractPageImages` matches exactly once.
- Grep shows `OPS.paintImageXObject` handling and a rasterization fallback path.

**Status:** `[x]` done

---

### Step 7.2 — Feed PDF images into the OCR scheduler
**Files:** `extension/src/viewer.js`
**Depends on:** Step 7.1, Step 6.1

**Prompt for developer:**
> In the PDF render path (`renderDocument`), when `options.ocrImages` is on, call `extractPageImages`
> per page, wrap each extracted image as an `<img>` (or overlay mount) appended into that page's section,
> and register it with the lazy OCR scheduler from Phase 06. Do not run extraction or OCR when
> `options.ocrImages` is off — text reflow behavior stays exactly as today.

**Verification:**
- Grep in `viewer.js` shows `extractPageImages` imported and called inside the PDF render path under an
  `options.ocrImages` guard.
- Extracted images are registered with the same scheduler used for EPUB (grep shows the shared call).

**Status:** `[x]` done

---

### Step 7.3 — Progress accounting across pages
**Files:** `extension/src/viewer.js`
**Depends on:** Step 7.2

**Prompt for developer:**
> Ensure the overall "OCR: N/M" counter includes PDF-extracted images and completes when the last
> visible page's images finish, so progress is coherent across a multi-page scanned PDF.

**Verification:**
- Grep shows the OCR total counter incremented by the count of PDF-extracted images.

**Status:** `[x]` done

## Phase done criteria
- [ ] Every `Step 7.*` is `[x] done`.
- [ ] `node --check` passes for `pdf-images.js` and `viewer.js`.
- [ ] Manual smoke (record in Handoff): open (a) a PDF with an embedded text image and (b) a scanned
      PDF, OCR on -> overlays appear with progress; "Translate page" translates the image text.
- [ ] With OCR off, both PDFs reflow exactly as today (Done criterion §11.8).
- [ ] Grep for `TODO(phase-07)` returns zero hits.
- [ ] Changelog entry added for every file in "Files touched".

## Handoff notes
Entry C works. All three strategic entry points now share one engine, scheduler, and progress model.

## Rollback plan
Revert phase commit(s); delete `pdf-images.js`. OCR path is gated behind `options.ocrImages`.
