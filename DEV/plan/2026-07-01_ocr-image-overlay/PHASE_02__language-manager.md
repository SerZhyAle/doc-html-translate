# Phase 02 — Language manager

**Strategic spec:** [`../2026-07-01_ocr-image-overlay.md`](../2026-07-01_ocr-image-overlay.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 01
**Steps done:** 3 / 3

## Objective
Provide a single module that owns OCR languages: the offline English path, on-demand download +
IndexedDB caching of other languages from the CDN, and an "installed vs. available" view.

## Prerequisites
- [ ] Phase 01 ✅ Done (engine + English data vendored).
- [ ] Research `research/01__ocr-stack-decisions.md` Q2 reviewed.

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `extension/src/ocr-lang.js` | New | ≤ 200 |

## Steps

### Step 2.1 — Define the language catalog and paths
**Files:** `extension/src/ocr-lang.js`
**Depends on:** - start of phase

**Prompt for developer:**
> Create `extension/src/ocr-lang.js`. Export a `LANGS` catalog of supported OCR languages (at least
> `eng`, `jpn`, `jpn_vert`, `rus`, `ukr`) each with a Tesseract code and a display name. Export
> `BUNDLED = ["eng"]`. Export `localLangPath()` returning `chrome.runtime.getURL("vendor/tesseract/lang/")`
> (the offline English directory) and `CDN_LANG_PATH` = the `tessdata_fast` CDN base from the research
> artifact. Do not initialize any worker here.

**Verification:**
- File `extension/src/ocr-lang.js` exists.
- `export const LANGS` matches exactly once and includes `eng` and `jpn`.
- `export const BUNDLED` matches exactly once.
- `CDN_LANG_PATH` string is present and points at the host named in the research artifact.

**Status:** `[x]` done

**Step Log:**
- 2026-07-01 — Created `src/ocr-lang.js` with `LANGS` (eng, rus, ukr, jpn, jpn_vert), `BUNDLED=["eng"]`,
  `CDN_LANG_PATH` = tessdata_fast host, `localLangPath()`, plus `workerAssets()`/`workerOptions()`
  helpers shared with Phase 03. `node --check` PASS.

---

### Step 2.2 — Installed-language tracking
**Files:** `extension/src/ocr-lang.js`
**Depends on:** Step 2.1

**Prompt for developer:**
> Add `getInstalledLangs()` and `markInstalled(code)` backed by the shared settings/storage
> (`chrome.storage.local`), returning at least the `BUNDLED` set unioned with any downloaded codes.
> Add `isInstalled(code)`. English must always report installed without any network or storage entry.

**Verification:**
- `export async function getInstalledLangs` matches exactly once.
- `export function isInstalled` (or async) matches exactly once.
- Code path returns `eng` as installed when storage is empty (grep shows `BUNDLED` unioned into the result).

**Status:** `[x]` done

**Step Log:**
- 2026-07-01 — Added `getInstalledLangs()` (unions `BUNDLED` with stored `ocrInstalledLangs`),
  `isInstalled(code)` (async), and `markInstalled(code)` (skips bundled). English always installed with
  no storage entry. PASS.

---

### Step 2.3 — On-demand download + cache
**Files:** `extension/src/ocr-lang.js`
**Depends on:** Step 2.2

**Prompt for developer:**
> Add `downloadLang(code, onProgress)` that ensures a Tesseract worker can load `code` from
> `CDN_LANG_PATH`, letting Tesseract.js cache the `.traineddata` in IndexedDB, then calls
> `markInstalled(code)` on success. Report progress via `onProgress({status, progress})`. On failure,
> throw with a message and do not mark installed. Add `resolveLangPath(code)` returning the local path
> for `eng` and the CDN path for others — this is the seam the overlay engine consumes.

**Verification:**
- `export async function downloadLang` matches exactly once.
- `export function resolveLangPath` matches exactly once and branches on `BUNDLED`/`eng`.
- `downloadLang` calls `markInstalled` only after a successful init (grep shows `markInstalled` inside a
  post-success path, not before the await).

**Status:** `[x]` done

**Step Log:**
- 2026-07-01 — Added `downloadLang(code, onProgress)` (throwaway worker prewarms + caches to IndexedDB,
  then `markInstalled` after `await`), and `resolveLangPath(code)` branching on `isBundled`. PASS.

## Phase done criteria
- [ ] Every `Step 2.*` is `[x] done`.
- [ ] `node --check extension/src/ocr-lang.js` passes (syntax gate).
- [ ] Grep for `TODO(phase-02)` returns zero hits.
- [ ] Changelog entry added for `extension/src/ocr-lang.js`.

## Handoff notes
`ocr-lang.js` is the single authority for language availability and paths. Overlay core (Phase 03)
gets the langPath via `resolveLangPath`; the controls UI (Phase 05) drives `downloadLang` /
`getInstalledLangs`.

## Rollback plan
Revert phase commit(s); delete `extension/src/ocr-lang.js`.
