# Phase 01 — Foundation: generic book dispatch + shared sanitize

**Strategic spec:** [`../2026-07-01_extension-format-parity.md`](../2026-07-01_extension-format-parity.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done (code + `npm test`; manual PDF/EPUB browser smoke pending user)
**Depends on:** none — foundation phase
**Steps done:** 5 / 5

## Objective
Turn the PDF/EPUB-specific viewer dispatch into a format-agnostic seam: a `detectFormat`
classifier, a generic `renderBook` that renders any book-shaped object, a `loadBook` helper
that wraps parse+error+render, and a shared `src/sanitize.js` that later parsers use to build
sanitized fragments. No new user-facing format is advertised in this phase.

## Prerequisites
- [ ] Working tree clean or on a feature branch.
- [ ] Read the "Architecture seam" section in [`INDEX.md`](INDEX.md).

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `extension/src/sanitize.js` | New | ≤ 130 |
| `extension/src/viewer.js` | Modified (~770 lines; surgical additions/renames) | ≤ 120 delta |
| `extension/manifest.json` | Modified | ≤ 5 delta |

> `viewer.js` is already ~770 lines. Edits here are surgical (one new function block, one
> rename, one signature change). Do not restructure the file.

## Steps

### Step 01.1 — Add the shared sanitize helper
**Files:** `extension/src/sanitize.js` (New)
**Depends on:** - start of phase

**Prompt for developer:**
> Create `extension/src/sanitize.js` exporting `sanitizeToFragment(html, index)`. It parses an
> HTML string with `DOMParser` (`text/html`), imports the `<body>` children into a detached
> `<div>`, removes the same unsafe/irrelevant nodes epub.js drops (reuse its `DROP_TAGS`
> list: `script,style,link,base,meta,title,noscript,iframe,object,embed,form`), strips every
> `on*` and `style` attribute, and namespaces every element `id` as `d<index>-<id>` (so
> multiple sections can share one document, exactly like epub.js `renderChapter`). It returns
> `{ frag, label }` where `frag` is a `DocumentFragment` of the cleaned nodes and `label` is
> the first `h1..h4` text (collapsed whitespace, sliced to 140 chars) or `""`. Model the
> structure on `renderChapter` in [`epub.js`](../../../extension/src/epub.js) but keep it
> standalone (no zip/blob/anchor params).

**Verification:**
- File `extension/src/sanitize.js` exists.
- `export function sanitizeToFragment` matches exactly once.
- The string `d${index}-` (id namespacing) is present.

**Status:** `[x]` done

---

### Step 01.2 — Add `detectFormat` classifier
**Files:** `extension/src/viewer.js`
**Depends on:** - start of phase

**Prompt for developer:**
> In [`viewer.js`](../../../extension/src/viewer.js), add a `detectFormat(data, name)` helper
> **next to** the existing `sniffType(data)` (do not remove `sniffType` yet - Step 01.4 rewires
> its caller and deletes it, so the file stays runnable between steps). `detectFormat` returns a
> format id string. Detection order: magic bytes first (authoritative), then filename
> extension. Magic bytes: `%PDF` -> `"pdf"`, `PK\x03\x04` -> `"epub"` (keep current behaviour).
> Extension fallback from `name` (the original filename or URL basename, lowercased):
> `.pdf`->`pdf`, `.epub`->`epub`. Return `"unknown"` otherwise. Add a module-level
> `const FORMAT_EXT = { pdf: "pdf", epub: "epub" }` map used for the extension lookup; later
> phases extend this map and the magic checks.

**Verification:**
- `function detectFormat` matches exactly once.
- `const FORMAT_EXT` is present.

**Status:** `[x]` done

---

### Step 01.3 — Generalize `renderEpubDocument` to `renderBook`
**Files:** `extension/src/viewer.js`
**Depends on:** Step 01.2

**Prompt for developer:**
> Rename `renderEpubDocument(book, fallbackTitle)` to `renderBook(book, fallbackTitle)` and
> update its single caller. Move the `revokeCurrent = book.revoke` assignment (currently in
> `loadEpubData`) into `renderBook`, guarded by `if (book.revoke)`. Keep all existing render
> behaviour (title, `applyLang`, sections loop, TOC, empty-text banner). Leave the PDF-only
> "Original" button logic where it is for now (it is toggled by the load wrappers).

**Verification:**
- `function renderBook` matches exactly once.
- `renderEpubDocument` no longer appears (grep returns zero).
- `if (book.revoke)` is present inside `renderBook`.

**Status:** `[x]` done

---

### Step 01.4 — Route `loadFromData` through `detectFormat` and add `loadBook`
**Files:** `extension/src/viewer.js`
**Depends on:** Step 01.2, Step 01.3

**Prompt for developer:**
> Change `loadFromData(data, title)` to `loadFromData(data, title, name)` and dispatch on
> `detectFormat(data, name)`: `"pdf"` -> `loadPdfData`, `"epub"` -> `loadEpubData`, anything
> else keeps today's fallback (PDF path, whose error handling reports an unreadable file).
> Thread the source name through both callers: `loadUrl` passes the URL, `openFilePicker`
> passes `f.name`. Add a generic helper
> `async function loadBook(data, title, parseFn, statusLabel)` that: sets the status to
> `statusLabel`, hides the PDF-only `#btn-original`, `await`s `parseFn(data)` inside a
> try/catch that shows a "Couldn't open this file" notice with a file-picker button on
> failure, then calls `renderBook(book, title)`. This is the entry point every later phase's
> parser reuses. Do not remove `loadEpubData`/`loadPdfData`. Now that `detectFormat` is the
> sole classifier, delete the old `sniffType` helper.

**Verification:**
- `function loadFromData(data, title, name)` matches exactly once.
- `async function loadBook` matches exactly once.
- `detectFormat(data, name)` is referenced inside `loadFromData`.
- `loadFromData(` call sites pass a third argument (grep shows the URL/`f.name` argument).
- `sniffType` no longer appears (grep returns zero).

**Status:** `[x]` done

---

### Step 01.5 — Expose `sanitize.js` as a web-accessible resource
**Files:** `extension/manifest.json`
**Depends on:** Step 01.1

**Prompt for developer:**
> Add `"src/sanitize.js"` to the `web_accessible_resources[0].resources` array in
> [`manifest.json`](../../../extension/manifest.json), next to the other `src/*.js` entries.

**Verification:**
- `src/sanitize.js` appears in `manifest.json`.

**Status:** `[x]` done

## Phase done criteria
- [ ] Every `Step 01.*` is `[x] done`.
- [ ] `cd extension && npm test` passes (existing pure-function tests still green after the
      rename/refactor). Name the command run here.
- [ ] Load the unpacked extension and open a PDF and an EPUB (file picker) - both still render
      (manual smoke; the refactor must not regress the two working formats).
- [ ] Grep for `TODO(phase-01)` returns zero hits.
- [ ] `DEV/CHANGELOG.md` entry added for `viewer.js`, `sanitize.js`, `manifest.json`.

## Handoff notes
- The seam is live: later phases only (a) add a parser module returning the book shape, (b)
  extend `FORMAT_EXT` (and magic checks if the format has a signature), (c) add a dispatch
  case in `loadFromData` calling `loadBook(data, title, parseXxx, "Reading ..")`, (d) extend
  the file-picker `accept` and, for interceptable formats, the DNR regex.
- `epub.js` `renderChapter` is intentionally NOT refactored to use `sanitize.js` (regression
  risk); the mild duplication is recorded in Phase 05's PARITY update.

## Rollback plan
Revert the phase commit(s). Low-risk - additive helpers plus one rename with a single caller.
