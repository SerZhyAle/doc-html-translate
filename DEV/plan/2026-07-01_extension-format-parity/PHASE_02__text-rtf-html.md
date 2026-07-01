# Phase 02 — Plain text, RTF, and local HTML

**Strategic spec:** [`../2026-07-01_extension-format-parity.md`](../2026-07-01_extension-format-parity.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done (code + `npm test`; manual browser smoke pending user)
**Depends on:** Phase 01
**Steps done:** 5 / 5

## Objective
Add the three dependency-free formats: TXT, RTF, and local HTML. Each ships a parser that
returns the book shape and is wired into the Phase 01 dispatch, file picker, and (RTF only)
URL interception.

## Prerequisites
- [ ] Phase 01 is ✅ Done (`renderBook`, `loadBook`, `detectFormat`, `FORMAT_EXT`,
      `src/sanitize.js` exist).
- [ ] Working tree clean or on a feature branch.

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `extension/src/txt.js` | New | ≤ 120 |
| `extension/src/rtf.js` | New | ≤ 200 |
| `extension/src/html.js` | New | ≤ 90 |
| `extension/test/txt.test.mjs` | New | ≤ 80 |
| `extension/test/rtf.test.mjs` | New | ≤ 80 |
| `extension/src/viewer.js` | Modified | ≤ 30 delta |
| `extension/src/viewer.html` | Modified | ≤ 3 delta |
| `extension/manifest.json` | Modified | ≤ 5 delta |
| `extension/src/background.js` | Modified | ≤ 4 delta |

## Steps

### Step 02.1 — TXT parser + test
**Files:** `extension/src/txt.js` (New), `extension/test/txt.test.mjs` (New)
**Depends on:** - start of phase

**Prompt for developer:**
> Create `extension/src/txt.js`. Export a pure `splitParagraphs(text)` that mirrors
> [`internal/txt/extract.go`](../../../internal/txt/extract.go): normalize CRLF/CR to LF; if
> the text has blank lines, split into paragraphs on blank-line boundaries; otherwise treat
> each non-empty line as its own paragraph. Also export `async function parseText(data)` that
> decodes the bytes as UTF-8, calls `splitParagraphs`, chunks paragraphs into sections of 30
> (`const PARAS_PER_SECTION = 30`, matching the Go pagination), builds each section's `frag`
> as `<p>` elements set via `textContent` (no HTML injection), and returns the book shape
> `{ title:"", lang:"", sampleText:<first ~8000 chars>, sections:[{id:`txt-sec-<i>`,label:"",frag}], toc:[], revoke:()=>{} }`.
> Add `extension/test/txt.test.mjs` using `node --test` that asserts `splitParagraphs` on
> (a) blank-line-separated text and (b) line-per-paragraph text.

**Verification:**
- `export function splitParagraphs` and `export async function parseText` each match once.
- `PARAS_PER_SECTION = 30` present.
- `extension/test/txt.test.mjs` exists and imports from `../src/txt.js`.

**Status:** `[x]` done

---

### Step 02.2 — RTF parser (port the Go stripper) + test
**Files:** `extension/src/rtf.js` (New), `extension/test/rtf.test.mjs` (New)
**Depends on:** - start of phase

**Prompt for developer:**
> Create `extension/src/rtf.js`. Port [`internal/rtf/extract.go`](../../../internal/rtf/extract.go)
> to a pure `stripRtf(bytes)` returning plain text: strip control words (`\word` + optional
> numeric arg), handle `{`/`}` groups, decode `\'XX` hex escapes, and decode Windows-1251
> bytes with `new TextDecoder("windows-1251")` (available in Chrome and node). Preserve the Go
> paragraph rule (split on `\par`/double newline). Also export `async function parseRtf(data)`
> that runs `stripRtf`, splits into paragraphs, and returns the book shape with sections of
> `<p>` nodes (reuse the chunking approach from `txt.js`; `id:`rtf-sec-<i>``). Add
> `extension/test/rtf.test.mjs` (`node --test`) asserting `stripRtf` on a small RTF sample
> including a `\'e0`-style Cyrillic escape.

**Verification:**
- `export function stripRtf` and `export async function parseRtf` each match once.
- `windows-1251` appears in `rtf.js`.
- `extension/test/rtf.test.mjs` exists and imports from `../src/rtf.js`.

**Status:** `[x]` done

---

### Step 02.3 — Local HTML parser
**Files:** `extension/src/html.js` (New)
**Depends on:** Phase 01 Step 01.1 (`sanitize.js`)

**Prompt for developer:**
> Create `extension/src/html.js` exporting `async function parseHtml(data)`. Decode the bytes
> as UTF-8, parse with `DOMParser` to read `<title>` (title) and the `<html lang>` attribute
> (lang), then hand the `<body>` inner HTML to `sanitizeToFragment` from
> [`sanitize.js`](../../../extension/src/sanitize.js) (index 0) to get one section frag.
> Return the book shape: `{ title, lang, sampleText:<frag text, ~8000 chars>, sections:[{id:"html-sec-0",label,frag}], toc:[], revoke:()=>{} }`.
> Mirror [`internal/htmlconv/extract.go`](../../../internal/htmlconv/extract.go) for the
> title/body extraction intent.

**Verification:**
- `export async function parseHtml` matches once.
- `sanitizeToFragment` is imported in `html.js`.

**Status:** `[x]` done

---

### Step 02.4 — Wire dispatch, detection, and file picker
**Files:** `extension/src/viewer.js`, `extension/src/viewer.html`
**Depends on:** Step 02.1, Step 02.2, Step 02.3

**Prompt for developer:**
> In [`viewer.js`](../../../extension/src/viewer.js): import `parseText`, `parseRtf`,
> `parseHtml`. Extend `FORMAT_EXT` with `txt:"txt", rtf:"rtf", htm:"html", html:"html"`. Add
> an RTF magic-byte check to `detectFormat` (`{\rtf` -> `"rtf"`). Add dispatch cases in
> `loadFromData`: `"txt"` -> `loadBook(data, title, parseText, "Reading text..")`, `"rtf"` ->
> `loadBook(data, title, parseRtf, "Reading RTF..")`, `"html"` ->
> `loadBook(data, title, parseHtml, "Reading HTML..")`. Extend the extension-stripping regex
> used by `fileTitle` and `openFilePicker` to include `txt|rtf|html?|md|fb2|mobi|azw3` (cover
> the whole planned set now to avoid churn). In [`viewer.html`](../../../extension/src/viewer.html),
> extend the `#file-input` `accept` attribute to add `.txt,.rtf,.html,.htm`.

**Verification:**
- `parseText`, `parseRtf`, `parseHtml` are imported in `viewer.js`.
- `FORMAT_EXT` contains `txt`, `rtf`, `html`.
- `loadFromData` contains a case routing to each of `parseText`, `parseRtf`, `parseHtml`.
- `viewer.html` `accept` contains `.txt` and `.rtf` and `.html`.

**Status:** `[x]` done

---

### Step 02.5 — Package the modules and intercept RTF URLs
**Files:** `extension/manifest.json`, `extension/src/background.js`
**Depends on:** Step 02.4

**Prompt for developer:**
> Add `"src/txt.js"`, `"src/rtf.js"`, `"src/html.js"` to `web_accessible_resources[0].resources`
> in [`manifest.json`](../../../extension/manifest.json). In
> [`background.js`](../../../extension/src/background.js), add `rtf` to the DNR `regexFilter`
> alternation in both `httpsRule` and `fileRule` (`(?:pdf|epub|rtf)`). Do NOT add `txt`, `html`,
> or `htm` to the DNR rules - those are normal browser-viewable resources and must not be
> intercepted; they are reachable via the file picker only.

**Verification:**
- `manifest.json` contains `src/txt.js`, `src/rtf.js`, `src/html.js`.
- `background.js` DNR `regexFilter` contains `rtf`.
- `background.js` DNR `regexFilter` does NOT contain `txt` or `html`.

**Status:** `[x]` done

## Phase done criteria
- [ ] Every `Step 02.*` is `[x] done`.
- [ ] `cd extension && npm test` passes (new `txt`/`rtf` tests green). Name the command here.
- [ ] Manual smoke: open a `.txt`, a `.rtf` (with Cyrillic), and a local `.html` via the file
      picker - each renders reflowed and Chrome offers "Translate page".
- [ ] Grep for `TODO(phase-02)` returns zero hits.
- [ ] `DEV/CHANGELOG.md` entry added for every file in "Files touched".

## Handoff notes
- TXT/RTF section chunking (`PARAS_PER_SECTION = 30`) is the shared pagination convention; the
  ebook phase reuses natural section breaks instead.
- RTF fidelity is a stripper, not a full renderer (matches the Go side) - known limitation.

## Rollback plan
Revert the phase commit(s). New parser modules are isolated; the viewer/manifest/background
deltas are additive.
