# Phase 06 - Extension diagnostics

**Strategic spec:** [`../2026-07-29_send-logs-to-author.md`](../2026-07-29_send-logs-to-author.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** none - separate codebase, may run alongside 01-05
**Steps done:** 5 / 5

## Objective

The extension's options page carries an About block whose button copies a short English report -
version, browser, format, page count, options, last error - to the clipboard, with no new
permission and no document content.

## Prerequisites

- [ ] `npm install` done in `extension/` and `npm test` green before starting.
- [ ] `extension/src/viewer.js` (1525) is committed before editing.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `extension/src/diagnostics.js` | New | ≤ 140 |
| `extension/src/viewer.js` | Modified (1525) | ≤ 1580 |
| `extension/src/options.html` | Modified (96) | ≤ 140 |
| `extension/src/options.js` | Modified (208) | ≤ 280 |
| `extension/_locales/<code>/messages.json` × 13 | Modified | ≤ +12 keys each |
| `extension/test/diagnostics.test.mjs` | New | ≤ 160 |

## Steps

### Step 06.1 - Record the last run

**Files:** `extension/src/diagnostics.js`
**Depends on:** - start of phase

**Prompt for developer:**
> Create `extension/src/diagnostics.js` exporting `recordRun({format, pages, error})` and
> `readRun()`, persisting a single `lastRun` object in `chrome.storage.local` with `format`,
> `pages`, `error` (message text only, truncated to 200 characters) and `at` (ISO string).
> Store **no** file name, no document text and no URL. Both functions swallow storage errors -
> diagnostics must never break a render. Follow the module style of `defaults.js` (ES module,
> named exports).

**Verification:**
- File `extension/src/diagnostics.js` exists and exports `recordRun` and `readRun`.
- `"lastRun"` appears exactly once as the storage key.
- Neither `name` nor `url` is written into the stored object (read the object literal).

**Status:** `[x]` done - 2026-08-11. `"lastRun"` is the single storage key; the stored object is
exactly `{format, pages, error, at}` - pinned by `recordRun keeps no document identity and
truncates the error`, which asserts the key set, not just the absence of a name.

---

### Step 06.2 - Feed it from the viewer

**Files:** `extension/src/viewer.js`
**Depends on:** Step 06.1

**Prompt for developer:**
> Import `recordRun` from `./diagnostics.js`. Call it once per document after the format is decided
> and the page total is known - at the `detectFormat` switch site near line 808 and wherever
> `page-total` is set - with the format id and the page count. Call it with `{error}` from the
> viewer's existing failure notice path so the last error is recorded. Do not add a new
> try/catch layer; `recordRun` already swallows.

**Verification:**
- `recordRun(` appears at least twice in `extension/src/viewer.js`.
- `import { recordRun }` (or an equivalent named import) appears exactly once.
- `npm test` in `extension/` exits 0.

**Status:** `[x]` done - 2026-08-11. `recordRun(` appears three times behind one named import:
the `detectFormat` site, the failure path, and `setPageTotal`. The four `page-total` assignments
were funnelled into that one helper rather than getting a `recordRun` each - a reader added later
cannot then leave the record without a page count. The failure call sits in `showNotice`, the
single funnel all fifteen failure notices already go through. `npm test` exit 0.

---

### Step 06.3 - Compose the report text

**Files:** `extension/src/diagnostics.js`
**Depends on:** Step 06.1

**Prompt for developer:**
> Add `export async function reportText(manifestVersion, options)` returning a plain English
> `key: value` block, one field per line: extension version, browser user agent, platform,
> interface language (from `uiLang()`), the option set (enabled-by-default, theme, source language,
> OCR on/off, OCR language, count of disabled hosts - the **count**, not the hosts), then the
> `lastRun` fields. English only, regardless of interface language, so it matches the desktop
> archive's `environment.txt`.

**Verification:**
- `export async function reportText(` matches exactly once.
- `disabledHosts.length` (or an equivalent count) appears, and no code path joins the host list
  into the text.

**Status:** `[x]` done - 2026-08-11. `reportText` matches once and emits only
`(o.disabledHosts || []).length`; the host names never enter the text, which the test asserts by
feeding a host and demanding it be absent. The shared labels with the desktop
`environment.txt` are `version`, `platform`, `interface language` and `ocr`.

---

### Step 06.4 - Add the About block and the button

**Files:** `extension/src/options.html`, `extension/src/options.js`
**Depends on:** Step 06.3

**Prompt for developer:**
> In `options.html`, turn the existing `<footer>` into an About block: keep the version span and
> the two links, and add above them a `<label data-i18n="optAbout">` heading plus a `<button
> id="copy-diag" data-i18n="btnCopyDiag">` and a `<span class="saved" id="saved-diag"
> data-i18n="optCopied">`, reusing the `.field` / `.saved` classes the page already uses for its
> save confirmations. In `options.js`, import `reportText`, and on click write it with
> `navigator.clipboard.writeText`, falling back to a hidden textarea plus `document.execCommand`
> when the clipboard call rejects; show the confirmation span the way the theme field already does.
> Add no permission to `manifest.json` - a user-gesture clipboard write needs none.

**Verification:**
- `id="copy-diag"` and `data-i18n="btnCopyDiag"` each appear exactly once in `options.html`.
- `reportText(` and `writeText(` each appear in `options.js`.
- `git diff --stat extension/manifest.json` shows no change.

**Status:** `[x]` done - 2026-08-11. Both attributes match once, `reportText(` and `writeText(`
are both in `options.js`, and `git diff --stat extension/manifest.json` is empty - the clipboard
write happens on the user's own click, which needs no permission.

---

### Step 06.5 - Localize and guard

**Files:** `extension/_locales/*/messages.json`, `extension/test/diagnostics.test.mjs`
**Depends on:** Step 06.4

**Prompt for developer:**
> Add `optAbout`, `btnCopyDiag`, `optCopied` and `diagHint` ("Copies a short technical summary -
> no document text - so you can paste it into a mail to the author.") to **all thirteen**
> `_locales/<code>/messages.json` files, matching `en`'s key set exactly (`zh_CN` is Chrome's name
> for the Chinese directory). Then write `extension/test/diagnostics.test.mjs` asserting
> `reportText` output contains the version and the format field, contains no `http` URL and no
> host name, and that all four new keys exist in every locale with a non-empty message.

**Verification:**
- `grep -l btnCopyDiag extension/_locales/*/messages.json` lists 13 files.
- `npm test` in `extension/` exits 0.
- `go test ./tests -run TestTypography` exits 0 (it scans `_locales` and keys the rule on the
  directory name).

**Status:** `[x]` done - 2026-08-11. All 13 `messages.json` files carry the four keys (13 files
match `btnCopyDiag`), added by a textual insert rather than a `JSON.stringify` round-trip so the
diff is 24 lines per file instead of a whole-file reformat. `npm test` exit 0 (92 tests) and the
typography gate exit 0. The test needed `Object.defineProperty` for `navigator`: Node 24 ships a
read-only one of its own, and a plain assignment throws.

## Phase done criteria

- [x] Every `Step 06.*` is `[x] done`.
- [x] `npm test` in `extension/` exits 0 - 92 tests, 0 failures.
- [x] `node build.mjs zip` in `extension/` exits 0 - wrote a 6.2 MB package.
- [x] Grep for `TODO(phase-06)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Handoff notes

Established: `diagnostics.js` as the extension's only diagnostics surface, the `lastRun` storage
key, and the rule that the report text is English and carries no document content or host list.
The shape difference from the desktop archive is intentional (strategic ADR-4) and phase 07 records
it in `docs/PARITY.md`.

## Rollback plan

Revert phase commit(s). The `lastRun` key is left behind in users' `chrome.storage.local` and is
harmless; no migration is needed either way.
