# Phase 05 — Controls & progress UI

**Strategic spec:** [`../2026-07-01_ocr-image-overlay.md`](../2026-07-01_ocr-image-overlay.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 02
**Steps done:** 4 / 4

## Objective
Give the user control: an "Use OCR for images" toggle next to the file-open control with nested
language-download controls, an options-page language manager, and the settings fields these write.

## Prerequisites
- [ ] Phase 02 ✅ Done (`ocr-lang.js` exposes `downloadLang`/`getInstalledLangs`/`LANGS`).

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `extension/src/popup.html` | Modified | ≤ 110 |
| `extension/src/popup.js` | Modified | ≤ 160 |
| `extension/src/options.html` | Modified | ≤ 130 |
| `extension/src/options.js` | Modified | ≤ 170 |

## Steps

### Step 5.1 — Settings fields (additive)
**Files:** `extension/src/popup.js`, `extension/src/options.js`
**Depends on:** - start of phase

**Prompt for developer:**
> Extend the shared `DEFAULT_OPTIONS` in both `popup.js` and `options.js` with `ocrImages: false` and
> `ocrLang: "eng"` (additive; older stored settings without them fall back to these defaults). Keep the
> two default objects in sync (they are duplicated today).

**Verification:**
- Grep in both `popup.js` and `options.js` shows `ocrImages` and `ocrLang` in `DEFAULT_OPTIONS`.

**Status:** `[x]` done

---

### Step 5.2 — Popup: OCR toggle next to file-open + nested language download
**Files:** `extension/src/popup.html`, `extension/src/popup.js`
**Depends on:** Step 5.1

**Prompt for developer:**
> Under the "Open a PDF or EPUB file" button in `popup.html`, add a checkbox "Use OCR for images"
> (bound to `options.ocrImages`) and a nested, indented block (disabled/hidden when the checkbox is off)
> listing OCR languages from `LANGS`: each installed language selectable as `ocrLang`; each not-yet-
> installed language shows a "Download" button. Wire `popup.js` to persist the toggle and selected
> language, call `downloadLang(code, onProgress)` on click, show inline download progress, and refresh
> the list on success. Use i18n message keys for all labels.

**Verification:**
- `popup.html` contains a checkbox with id bound to OCR images and a nested language container.
- Grep in `popup.js` shows `downloadLang` called from a button handler and `ocrImages`/`ocrLang`
  written to storage.
- The nested block's enabled/disabled state is tied to the checkbox (grep shows the handler).

**Status:** `[x]` done

---

### Step 5.3 — Options: language manager section
**Files:** `extension/src/options.html`, `extension/src/options.js`
**Depends on:** Step 5.1

**Prompt for developer:**
> Add an "Image OCR" section to `options.html`: the "Use OCR for images" toggle, the default OCR
> language selector, and an installed/available language list with Download buttons (same data as the
> popup, more room). Wire `options.js` with `getInstalledLangs`/`downloadLang`, progress feedback, and
> persistence. Reuse the existing `.field`/`.hint`/`.saved` styles.

**Verification:**
- `options.html` contains an OCR section with the toggle, language select, and language list container.
- Grep in `options.js` shows imports/usage of `getInstalledLangs` and `downloadLang`.

**Status:** `[x]` done

---

### Step 5.4 — Localized control strings
**Files:** `extension/_locales/en/messages.json`, `extension/_locales/ru/messages.json`, `extension/_locales/uk/messages.json`
**Depends on:** Step 5.2, Step 5.3

**Prompt for developer:**
> Add message keys for the OCR controls (`ocrUseImages`, `ocrDefaultLang`, `ocrDownload`,
> `ocrDownloading`, `ocrInstalled`, `ocrLangsHint`) to all three locale catalogs with translations,
> following the repo typography rules. Reference them from `popup.js`/`options.js` (or via
> `__MSG__`/`data-i18n` if that pattern is used).

**Verification:**
- Each of the three `messages.json` files contains `ocrUseImages` and `ocrDownload`.
- Grep in `popup.js` or `options.js` shows the new keys used.

**Status:** `[x]` done

## Phase done criteria
- [ ] Every `Step 5.*` is `[x] done`.
- [ ] `node --check` passes for `popup.js` and `options.js`.
- [ ] Manual smoke (record in Handoff): toggle appears next to the open button; downloading a language
      shows progress and then it becomes selectable.
- [ ] Grep for `TODO(phase-05)` returns zero hits.
- [ ] Changelog entry added for every file in "Files touched".

## Handoff notes
`options.ocrImages` and `options.ocrLang` now exist and are user-editable. Viewer integration
(Phases 06-07) reads them to decide whether/how to OCR document images.

## Rollback plan
Revert phase commit(s); the settings fields are additive and safe to drop.
