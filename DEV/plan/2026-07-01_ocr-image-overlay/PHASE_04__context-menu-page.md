# Phase 04 — Context-menu entry + overlay page

**Strategic spec:** [`../2026-07-01_ocr-image-overlay.md`](../2026-07-01_ocr-image-overlay.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 03
**Steps done:** 4 / 4

## Objective
Right-click any web image -> open a dedicated tab that OCRs the image and shows the translatable
overlay with visible progress. Delivers the first end-to-end entry point.

## Prerequisites
- [ ] Phase 03 ✅ Done (`overlayImage` available).
- [ ] Phase 01 done (manifest has `contextMenus` + web-accessible OCR page).

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `extension/src/background.js` | Modified | ≤ 190 |
| `extension/src/ocr.html` | New | ≤ 60 |
| `extension/src/ocr.js` | New | ≤ 140 |
| `extension/_locales/en/messages.json` | Modified | ≤ 200 |
| `extension/_locales/ru/messages.json` | Modified | ≤ 200 |
| `extension/_locales/uk/messages.json` | Modified | ≤ 200 |

## Steps

### Step 4.1 — Register the image context menu
**Files:** `extension/src/background.js`
**Depends on:** - start of phase

**Prompt for developer:**
> In `extension/src/background.js`, on `onInstalled` create a context-menu item with
> `contexts: ["image"]` whose title comes from a new i18n message key. On click, open
> `chrome.runtime.getURL("src/ocr.html") + "?src=" + encodeURIComponent(info.srcUrl)` in a new tab via
> `chrome.tabs.create`. Guard against duplicate creation (remove-all or `try/catch` on create).

**Verification:**
- Grep in `background.js` shows `chrome.contextMenus.create` with `contexts: ["image"]`.
- Grep shows `src/ocr.html` and `info.srcUrl` used to build the new-tab URL.

**Status:** `[x]` done

---

### Step 4.2 — OCR page shell
**Files:** `extension/src/ocr.html`
**Depends on:** - start of phase

**Prompt for developer:**
> Create `extension/src/ocr.html`: a minimal page linking `ocr-overlay.css`, containing a status/
> progress region (reuse the viewer's status-bar markup pattern) and a mount point for the overlay, and
> loading `ocr.js` as a module. Set an initial `<html lang>` that will be updated by the script.

**Verification:**
- File `extension/src/ocr.html` exists.
- It links `ocr-overlay.css` and loads `ocr.js` with `type="module"`.
- It contains a progress/status element and an overlay mount element with stable ids.

**Status:** `[x]` done

---

### Step 4.3 — OCR page controller
**Files:** `extension/src/ocr.js`
**Depends on:** Step 4.2, Step 3.4

**Prompt for developer:**
> Create `extension/src/ocr.js`. Parse `?src=`, validate the scheme, read the preferred OCR language
> from settings (default `eng`), call `overlayImage(src, { lang, onProgress })` mounting the result and
> updating the status/progress region throughout, then set `document.documentElement.lang` via
> `ocrLangToHtmlLang` so Chrome offers "Translate page". Show a clear message if no text is found or the
> image cannot be loaded.

**Verification:**
- File `extension/src/ocr.js` exists and imports `overlayImage` and `ocrLangToHtmlLang` from
  `./ocr-overlay.js`.
- Grep shows it reads `?src=` and sets `document.documentElement.lang`.
- Grep shows an `onProgress` handler updating the status/progress element.

**Status:** `[x]` done

---

### Step 4.4 — Localized context-menu + page strings
**Files:** `extension/_locales/en/messages.json`, `extension/_locales/ru/messages.json`, `extension/_locales/uk/messages.json`
**Depends on:** Step 4.1

**Prompt for developer:**
> Add message keys used by the context menu and OCR page (e.g. `ocrImageMenu`, `ocrProgress`,
> `ocrNoText`, `ocrLoadError`) to all three locale catalogs, with translations. Follow the repo
> typography rules (short hyphens, ".."). Reference them via `chrome.i18n.getMessage` in code.

**Verification:**
- Each of the three `messages.json` files contains the new keys (`ocrImageMenu` at minimum).
- Grep in `background.js`/`ocr.js` shows `chrome.i18n.getMessage("ocrImageMenu")` (or equivalent).

**Status:** `[x]` done

## Phase done criteria
- [ ] Every `Step 4.*` is `[x] done`.
- [ ] `npm run zip` in `extension/` succeeds (page + assets package cleanly).
- [ ] Manual smoke (record in Handoff): right-click a web image with text -> new tab shows overlay ->
      Chrome offers "Translate page". (Set spec status BlockNeedUserTest if not yet run.)
- [ ] Grep for `TODO(phase-04)` returns zero hits.
- [ ] Changelog entry added for every file in "Files touched".

## Handoff notes
Entry A works end to end. The viewer phases reuse `overlayImage` the same way. Note any accuracy/
progress observations for tuning.

## Rollback plan
Revert phase commit(s); delete `ocr.html`/`ocr.js`; remove the context-menu block and new i18n keys.
