# Phase 03 — Overlay core

**Strategic spec:** [`../2026-07-01_ocr-image-overlay.md`](../2026-07-01_ocr-image-overlay.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 02
**Steps done:** 4 / 4

## Objective
Build the reusable OCR-overlay unit: lazy shared worker, recognize an image, group results into
block plates, render opaque plates with fitted real HTML text over the image, and emit progress.

## Prerequisites
- [ ] Phase 02 ✅ Done (`ocr-lang.js` provides `resolveLangPath`).
- [ ] Research `research/01__ocr-stack-decisions.md` Q3/Q4 reviewed.

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `extension/src/ocr-overlay.js` | New | ≤ 260 |
| `extension/src/ocr-overlay.css` | New | ≤ 120 |

## Steps

### Step 3.1 — Lazy shared worker + FIFO queue
**Files:** `extension/src/ocr-overlay.js`
**Depends on:** - start of phase

**Prompt for developer:**
> Create `extension/src/ocr-overlay.js`. Import from `./ocr-lang.js`. Implement a lazily-created shared
> Tesseract worker (created from the vendored `vendor/tesseract/` worker + core, `langPath` via
> `resolveLangPath`) and a single-flight FIFO queue so only one image is recognized at a time. Export
> `recognize(imageSource, { lang, onProgress })` returning `{ blocks, width, height }` where each block
> has `{ text, bbox:{x0,y0,x1,y1} }`. Route Tesseract progress into `onProgress`.

**Verification:**
- File `extension/src/ocr-overlay.js` exists and imports `./ocr-lang.js`.
- `export async function recognize` matches exactly once.
- Worker creation is lazy (grep shows a null-check/singleton guard, not top-level `await createWorker`).

**Status:** `[x]` done

---

### Step 3.2 — Image source normalization (cross-origin safe)
**Files:** `extension/src/ocr-overlay.js`
**Depends on:** Step 3.1

**Prompt for developer:**
> Add a helper that accepts an image URL, a Blob, or an `HTMLImageElement` and yields a bitmap the
> engine can read. For a URL, fetch the bytes (the extension has host access, avoiding tainted-canvas
> problems) and build a Blob/ImageBitmap; capture natural width/height. Reject non-http/https/file/blob/
> data sources.

**Verification:**
- A source-normalizer function is defined and referenced by `recognize`.
- Grep shows a `fetch(` of the image URL (not reading pixels off a cross-origin `<img>` via canvas).
- A guard rejects unexpected URL schemes.

**Status:** `[x]` done

---

### Step 3.3 — Overlay renderer (opaque plates + fitted text)
**Files:** `extension/src/ocr-overlay.js`, `extension/src/ocr-overlay.css`
**Depends on:** Step 3.2

**Prompt for developer:**
> Export `buildOverlay({ imageEl_or_src, blocks, width, height })` returning a container element: the
> image as a base layer plus one absolutely-positioned plate per block, positioned/sized in percent of
> the image dimensions, with an opaque background covering the source text and the recognized text as
> real DOM text auto-fitted to the box (font-size clamp; height auto so longer translations wrap).
> Style the plate, container, and per-image progress badge in `ocr-overlay.css`. Skip empty-text blocks.

**Verification:**
- `export function buildOverlay` matches exactly once.
- File `extension/src/ocr-overlay.css` exists and defines the plate + container + progress-badge classes
  referenced from `ocr-overlay.js`.
- Plate positions use percentage units (grep shows `%` in the position/size assignment).

**Status:** `[x]` done

---

### Step 3.4 — One-call convenience + language tag helper
**Files:** `extension/src/ocr-overlay.js`
**Depends on:** Step 3.3

**Prompt for developer:**
> Export `overlayImage(imageEl_or_src, { lang, onProgress })` that runs `recognize` then `buildOverlay`
> and resolves to the container. Export `ocrLangToHtmlLang(code)` mapping a Tesseract code to a BCP-47
> tag (e.g. `jpn`->`ja`, `rus`->`ru`, `ukr`->`uk`, `eng`->`en`) for callers that set `<html lang>`.

**Verification:**
- `export async function overlayImage` matches exactly once.
- `export function ocrLangToHtmlLang` matches exactly once and maps `jpn` to `ja`.

**Status:** `[x]` done

## Phase done criteria
- [ ] Every `Step 3.*` is `[x] done`.
- [ ] `node --check extension/src/ocr-overlay.js` passes.
- [ ] Grep for `TODO(phase-03)` returns zero hits.
- [ ] Changelog entry added for every file in "Files touched".

## Handoff notes
`overlayImage()` is the single entry the context-menu page and the viewer integrations call. Progress
flows through `onProgress`. The queue guarantees one-at-a-time OCR.

## Rollback plan
Revert phase commit(s); delete `ocr-overlay.js` / `ocr-overlay.css`.
