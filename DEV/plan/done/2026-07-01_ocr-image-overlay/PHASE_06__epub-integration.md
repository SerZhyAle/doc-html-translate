# Phase 06 — EPUB image integration

**Strategic spec:** [`../2026-07-01_ocr-image-overlay.md`](../2026-07-01_ocr-image-overlay.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 03, Phase 05
**Steps done:** 3 / 3

## Objective
When OCR is enabled, images inside an opened EPUB are recognized lazily (on scroll into view) and
replaced with translatable overlays, with per-image and overall progress, without blocking reading.

## Prerequisites
- [ ] Phase 03 ✅ Done (`overlayImage`), Phase 05 ✅ Done (`options.ocrImages`/`ocrLang`).
- [ ] Research `research/01__ocr-stack-decisions.md` Q3 (visibility-based trigger) reviewed.

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `extension/src/viewer.js` | Modified | ≤ 720 |
| `extension/src/viewer.css` | Modified | ≤ 400 |

## Steps

### Step 6.1 — Lazy OCR scheduler in the viewer
**Files:** `extension/src/viewer.js`
**Depends on:** - start of phase

**Prompt for developer:**
> In `viewer.js`, add a lazy OCR scheduler: an `IntersectionObserver` that, when `options.ocrImages`
> is on, queues each observed `<img>` for `overlayImage(imgEl, { lang: options.ocrLang, onProgress })`
> and replaces the image node with the returned overlay container on completion. Maintain an overall
> counter and show it in the toolbar status ("OCR: N/M"). Do nothing when `options.ocrImages` is off.

**Verification:**
- Grep in `viewer.js` shows `new IntersectionObserver` used for OCR and a guard on `options.ocrImages`.
- Grep shows `overlayImage(` called with `options.ocrLang`.
- Grep shows an overall OCR counter written into the status text.

**Status:** `[x]` done

---

### Step 6.2 — Observe EPUB chapter images
**Files:** `extension/src/viewer.js`
**Depends on:** Step 6.1

**Prompt for developer:**
> In the EPUB render path (`renderEpubDocument`), after each chapter section is appended, register its
> `<img>` elements with the lazy OCR scheduler from Step 6.1 (only when `options.ocrImages` is on). Do
> not change behavior when OCR is off (chapters render exactly as today with blob: images).

**Verification:**
- Grep in `viewer.js` shows the scheduler being fed `section.querySelectorAll("img")` (or equivalent)
  within the EPUB render path.
- The `options.ocrImages`-off path leaves the existing EPUB rendering unchanged (no unconditional OCR).

**Status:** `[x]` done

---

### Step 6.3 — Per-image progress badge styling
**Files:** `extension/src/viewer.css`
**Depends on:** Step 6.1

**Prompt for developer:**
> Add styles in `viewer.css` for the per-image OCR progress badge and the overlay container as they
> appear inside the reading column (spacing, max-width, badge position), reusing `ocr-overlay.css`
> classes where possible and only adding viewer-context overrides here.

**Verification:**
- Grep in `viewer.css` shows rules targeting the OCR overlay/badge classes.

**Status:** `[x]` done

## Phase done criteria
- [ ] Every `Step 6.*` is `[x] done`.
- [ ] `node --check extension/src/viewer.js` passes.
- [ ] Manual smoke (record in Handoff): open an EPUB with a text-bearing image, OCR on -> scrolling to
      the image shows progress then an overlay; "Translate page" translates image text with the body.
- [ ] With OCR off, the same EPUB opens and reads unchanged (Done criterion §11.8).
- [ ] Grep for `TODO(phase-06)` returns zero hits.
- [ ] Changelog entry added for every file in "Files touched".

## Handoff notes
Entry B works. The scheduler is format-agnostic; the PDF phase feeds it extracted images the same way.

## Rollback plan
Revert phase commit(s); the OCR scheduler is gated behind `options.ocrImages` and inert when off.
