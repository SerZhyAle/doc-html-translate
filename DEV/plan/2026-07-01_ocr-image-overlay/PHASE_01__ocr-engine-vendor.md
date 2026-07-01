# Phase 01 — OCR engine vendoring

**Strategic spec:** [`../2026-07-01_ocr-image-overlay.md`](../2026-07-01_ocr-image-overlay.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** none — foundation phase
**Steps done:** 4 / 4

## Objective
Vendor the Tesseract.js engine (JS glue + WASM core) and the bundled English language data into the
extension package, and open the manifest surface (permissions + web-accessible resources) the OCR
feature needs.

## Prerequisites
- [ ] Working tree clean or on a feature branch.
- [ ] `npm install` runnable in `extension/`.

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `extension/package.json` | Modified | ≤ 40 |
| `extension/build.mjs` | Modified | ≤ 180 |
| `extension/manifest.json` | Modified | ≤ 90 |
| `extension/vendor/tesseract/**` | New (generated) | n/a |

## Steps

### Step 1.1 — Add Tesseract dev dependencies
**Files:** `extension/package.json`
**Depends on:** - start of phase

**Prompt for developer:**
> Add `tesseract.js` and `tesseract.js-core` to `devDependencies` in `extension/package.json` (they are
> vendored at build time, like `pdfjs-dist`). Run `npm install` in `extension/` so `node_modules` has
> both. Do not add any runtime `dependencies`.

**Verification:**
- `extension/package.json` `devDependencies` contains `tesseract.js` and `tesseract.js-core`.
- `extension/node_modules/tesseract.js` and `extension/node_modules/tesseract.js-core` exist.

**Status:** `[x]` done

**Step Log:**
- 2026-07-01 — Added `tesseract.js@^5.1.1` + `tesseract.js-core@^5.1.1` to devDependencies; `npm install` added 14 packages; both present in node_modules (v5.1.1). PASS.

---

### Step 1.2 — Vendor engine + English data via build.mjs
**Files:** `extension/build.mjs`
**Depends on:** Step 1.1

**Prompt for developer:**
> Extend the `vendor` command in `extension/build.mjs` to copy the Tesseract runtime into
> `extension/vendor/tesseract/`: the worker script, the WASM core (`tesseract-core*.wasm` /
> its JS loader) from `tesseract.js-core`, and `eng.traineddata` placed under
> `vendor/tesseract/lang/eng.traineddata` (decompress the `.gz` if the package ships it gzipped, or
> fetch the `tessdata_fast` `eng.traineddata` once and cache it into the repo). Carry the upstream
> LICENSE next to it as `LICENSE.tesseract`. Print a one-line summary like the pdfjs step.

**Verification:**
- Running `npm run vendor` in `extension/` exits 0.
- `extension/vendor/tesseract/lang/eng.traineddata` exists and is > 1 MB.
- A Tesseract worker script (`worker.min.js`) and the self-contained WASM core
  (`tesseract-core-simd-lstm.wasm.js`, wasm embedded) exist under `extension/vendor/tesseract/`.
- `extension/vendor/tesseract/LICENSE.tesseract` exists.

**Status:** `[x]` done

**Step Log:**
- 2026-07-01 — Added `vendorTesseract()` to build.mjs (runs after pdfjs vendor). Copies
  `tesseract.esm.min.js`, `worker.min.js`, `tesseract-core-simd-lstm.wasm.js`, fetches
  `eng.traineddata.gz` from tessdata_fast and stores it decompressed (3.92 MB). SIMD-only /
  self-contained core chosen (min_chrome 105 guarantees SIMD) — ~8 MB total vs ~30 MB for all
  variants. `npm run vendor` exits 0; all files present. PASS.

---

### Step 1.3 — Declare contextMenus permission
**Files:** `extension/manifest.json`
**Depends on:** - start of phase

**Prompt for developer:**
> Add `"contextMenus"` to the `permissions` array in `extension/manifest.json`. Do not change
> `host_permissions`.

**Verification:**
- `extension/manifest.json` `permissions` array contains `"contextMenus"`.

**Status:** `[x]` done

**Step Log:**
- 2026-07-01 — Added `"contextMenus"` to manifest permissions. PASS.

---

### Step 1.4 — Expose OCR resources as web-accessible
**Files:** `extension/manifest.json`
**Depends on:** Step 1.2

**Prompt for developer:**
> Add the OCR page and shared modules to the existing `web_accessible_resources` resources list in
> `extension/manifest.json`: `src/ocr.html`, `src/ocr.js`, `src/ocr-overlay.js`, `src/ocr-overlay.css`,
> `src/ocr-lang.js`. The existing `vendor/*` glob already covers `vendor/tesseract/**`; confirm it does
> and widen to `vendor/**` only if the current glob is non-recursive.

**Verification:**
- `extension/manifest.json` `web_accessible_resources[0].resources` includes `src/ocr.html`,
  `src/ocr-overlay.js`, and `src/ocr-lang.js`.
- The resources list still includes the existing viewer entries (no regression).

**Status:** `[x]` done

**Step Log:**
- 2026-07-01 — Added `src/ocr.html`, `src/ocr.js`, `src/ocr-overlay.js`, `src/ocr-overlay.css`,
  `src/ocr-lang.js`, `src/pdf-images.js` to web_accessible_resources; viewer entries and `vendor/*`
  (recursive, already covers `vendor/tesseract/**` like cmaps) retained. PASS.

## Phase done criteria
- [ ] Every `Step 1.*` is `[x] done`.
- [ ] `npm run vendor` in `extension/` passes and `npm run zip` still succeeds (packaging invariant).
- [ ] Grep for `TODO(phase-01)` returns zero hits.
- [ ] Changelog entry added for every file in "Files touched".

## Handoff notes
`extension/vendor/tesseract/` now holds the offline engine + English data; the manifest permits the
context menu and web-accessible OCR pages. Later phases consume these assets; the English `langPath`
is `vendor/tesseract/lang/`.

## Rollback plan
Revert phase commit(s); delete `extension/vendor/tesseract/`.
