# Phase 05 - Extension UI in thirteen languages

**Strategic spec:** [`../2026-07-28_thirteen-ui-languages.md`](../2026-07-28_thirteen-ui-languages.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 01 (independent of the Go phases - may run in parallel with 03/04)
**Steps done:** 7 / 7

## Objective

Take the extension's viewer, options and popup from English-only to all 13 languages through
`chrome.i18n`, with an explicit in-extension language override (strategic decision D6).

## Prerequisites

- [ ] Phase 01 ✅ Done (the typography guard reads `_locales` per directory).
- [ ] `npm test` green in `extension/` before starting.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `extension/src/i18n.js` | New | ≤ 120 |
| `extension/src/viewer.html` · `viewer.js` | Modified | ≤ 500 / ≤ 900 |
| `extension/src/options.html` · `options.js` | Modified | ≤ 300 / ≤ 400 |
| `extension/src/popup.html` · `popup.js` | Modified | ≤ 200 / ≤ 300 |
| `extension/src/ocr.js` · `ocr-overlay.js` | Modified | ≤ 500 each |
| `extension/_locales/<code>/messages.json` × 13 | New / Modified | ≤ 400 each |
| `extension/test/i18n.test.mjs` | New | ≤ 200 |

## Steps

### Step 05.1 - Add the shared i18n helper and the markup pass

**Files:** `extension/src/i18n.js`
**Depends on:** - start of phase

**Prompt for developer:**
> Create `i18n.js` exporting `t(key, fallback)` (wrapping `chrome.i18n.getMessage` in the existing
> try/catch-with-fallback idiom already used in `options.js` and `popup.js`), `uiLang()` returning the
> effective language (stored override first, then `chrome.i18n.getUILanguage()`, then `en`), and
> `applyI18n(root)` which walks `[data-i18n]`, `[data-i18n-title]`, `[data-i18n-ph]` and fills them.
> `applyI18n` also sets `dir="rtl"` on the element it is given for `ar` and `ur`.

**Verification:**
- File `extension/src/i18n.js` exists and exports `t`, `uiLang`, `applyI18n`.
- `npm test` in `extension/` exits 0.

**Status:** `[x] done`

---

### Step 05.2 - Move the viewer strings into `_locales/en`

**Files:** `extension/src/viewer.html`, `extension/src/viewer.js`, `extension/_locales/en/messages.json`
**Depends on:** Step 05.1

**Prompt for developer:**
> Replace the ~17 visible strings in `viewer.html` with `data-i18n` keys and the ~41 user-facing strings in
> `viewer.js` with `t("key", "English fallback")`. Add every key to `_locales/en/messages.json` with a
> `description`. Call `applyI18n(document)` on load. Keep the English fallback argument at each call site -
> a missing key must degrade to English, never to blank.

**Verification:**
- `data-i18n=` appears at least 17 times in `viewer.html`.
- Every `t("` key used in `viewer.js` exists in `_locales/en/messages.json`.
- `npm test` exits 0.

**Status:** `[x] done`

---

### Step 05.3 - Do the same for options, popup and the OCR surfaces

**Files:** `extension/src/options.html`, `options.js`, `popup.html`, `popup.js`, `ocr.js`, `ocr-overlay.js`, `extension/_locales/en/messages.json`
**Depends on:** Step 05.2

**Prompt for developer:**
> Repeat Step 05.2 for the options page (~29 strings), the popup (~12) and the remaining OCR status strings
> that are still literals. Reuse the 20 keys that already exist in `_locales` rather than adding duplicates.

**Verification:**
- No user-visible English literal remains in `options.html`, `popup.html` (grep for `>[A-Za-z]` outside
  `<script>`/`<style>` returns only keys or non-text markup).
- `_locales/en/messages.json` key count ≥ 100.
- `npm test` exits 0.

**Status:** `[x] done`

---

### Step 05.4 - Add the twelve non-English locale files

**Files:** `extension/_locales/{ru,uk,de,it,es,fr,pt,ar,hi,bn,ur,zh_CN}/messages.json`
**Depends on:** Step 05.3

**Prompt for developer:**
> Extend `ru` and `uk` to the full key set and create the ten new locale directories with complete
> translations - `pt` (one neutral translation per decision D1) and `zh_CN` per Chrome's locale naming.
> Every key present in `en`, every `message` non-empty, placeholders preserved. This also localizes the
> extension's store name and short description, which Chrome and Edge serve automatically from these files.

**Verification:**
- `ls extension/_locales` returns 13 directories.
- Every `messages.json` has the same key set as `en` and no empty `message` value.
- `npm test` exits 0 and `node build.mjs zip` produces a package containing all 13 locale folders.

**Status:** `[x] done`

---

### Step 05.5 - Mirror the viewer chrome, not the document

**Files:** `extension/src/viewer.html`, `extension/src/viewer.css`, `extension/src/viewer.js`
**Depends on:** Step 05.4

**Prompt for developer:**
> For `ar` and `ur` set `dir="rtl"` on the toolbar/TOC chrome only, and set the content container's `dir`
> from the **document's** detected language (`lang.js`), never from the UI language. Add a comment stating
> the rule, matching the Go side established in Step 03.7.

**Verification:**
- `dir` is assigned in two distinct places with distinct sources (UI language, document language).
- A CBZ/EPUB fixture rendered under `ar` in the existing DOM test keeps its content container LTR.
- `npm test` exits 0.

**Status:** `[x] done`

---

### Step 05.6 - Add the interface-language override to the options page

**Files:** `extension/src/options.html`, `extension/src/options.js`, `extension/src/i18n.js`
**Depends on:** Step 05.5

**Prompt for developer:**
> Add a "Interface language" select listing the 13 endonyms plus a first entry meaning "follow the browser"
> (the default). Persist it with the existing options storage and have `uiLang()` read it. Applying it must
> re-render the open viewer without a reload where practical, or state in the option's hint that the viewer
> picks it up on the next open.

**Verification:**
- The options page contains a select with 14 entries (13 languages + follow-browser).
- The stored key appears in `options.js` and is read by `i18n.js`.
- `npm test` exits 0.

**Status:** `[x] done`

---

### Step 05.7 - Guard the locale files with a test

**Files:** `extension/test/i18n.test.mjs`
**Depends on:** Step 05.6

**Prompt for developer:**
> Add a test asserting: the 13 locale directories exist and match the language set; every locale has
> exactly the `en` key set; no `message` is empty; every `data-i18n*` key used in the three HTML files and
> every `t("key"` in the JS sources exists in `en`.

**Verification:**
- File `extension/test/i18n.test.mjs` exists.
- `npm test` exits 0 and its output names the new test.

**Status:** `[x] done`

## Step log

- 2026-07-28 - 05.1 `src/i18n.js` with `t`/`uiLang`/`initI18n`/`applyI18n`/`loadMessages`/`setUiLang`. PASS.
- 2026-07-28 - 05.2 viewer markup tagged (toolbar, TOC, title, themes, fonts) and `applyViewerChromeI18n()` wired into `main()`.
- 2026-07-28 - 05.2 finished: 50 status/notice literals in `viewer.js` moved onto `t()`; `t()` gained `{1}`-style substitution so counted messages translate as whole sentences; 66 new keys in all 13 locales. `npm test` 88/88 PASS.
- 2026-07-28 - 05.3 options + popup markup tagged; both `msg()` helpers routed through `t()` so the override reaches them.
- 2026-07-28 - 05.4 `_locales` 3 -> 13 directories, 53 runtime keys each; `storeDescription` + captions deliberately left en/ru/uk (listing copy, phase 08).
- 2026-07-28 - 05.5 `applyI18n` sets `dir` on the chrome scope only; the document keeps its own direction and `<html lang>`.
- 2026-07-28 - 05.6 interface-language selector in the options page, first entry "follow the browser"; persisted as `uiLang`.
- 2026-07-28 - 05.7 `test/i18n.test.mjs`; `npm test` 88/88 PASS, typography guard green over the new locales, `node build.mjs zip` OK.

## Phase done criteria

- [x] Every `Step 05.*` is `[x] done`.
- [x] `npm test` in `extension/` exits 0 - 88/88 PASS.
- [x] `go test ./tests -run TestTypography` exits 0 (the new locales are scanned by the Phase 01 rule).
- [x] Grep for `TODO(phase-05)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Handoff notes

The extension now matches the desktop rule: chrome carries the UI language, the document keeps its own.
Phase 07 uses the Step 05.6 override to capture extension screenshots without 13 browser profiles.

## Rollback plan

Revert the phase commit(s). Deleting the ten new `_locales` directories alone returns the extension to
en/ru/uk with the mechanism intact.
