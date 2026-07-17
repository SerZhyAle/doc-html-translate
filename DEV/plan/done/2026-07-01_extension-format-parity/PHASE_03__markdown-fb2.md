# Phase 03 — Markdown (vendored) and FB2

**Strategic spec:** [`../2026-07-01_extension-format-parity.md`](../2026-07-01_extension-format-parity.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done (code + `npm run vendor` + `npm test`; manual browser smoke pending user)
**Depends on:** Phase 01
**Steps done:** 5 / 5

## Objective
Add Markdown (via a vendored `marked` converter) and FB2 (a DOMParser port of the pure-Go FB2
extractor, with embedded images). Both are markup-to-HTML formats that carry a heading-derived
table of contents.

## Prerequisites
- [ ] Phase 01 is ✅ Done.
- [ ] `npm install` works in `extension/` (the vendor step needs dev dependencies).

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `extension/build.mjs` | Modified | ≤ 40 delta |
| `extension/package.json` | Modified | ≤ 2 delta |
| `extension/src/md.js` | New | ≤ 110 |
| `extension/src/fb2.js` | New | ≤ 170 |
| `extension/src/viewer.js` | Modified | ≤ 20 delta |
| `extension/src/viewer.html` | Modified | ≤ 3 delta |
| `extension/manifest.json` | Modified | ≤ 4 delta |
| `extension/src/background.js` | Modified | ≤ 2 delta |

## Steps

### Step 03.1 — Vendor the `marked` Markdown converter
**Files:** `extension/build.mjs`, `extension/package.json`
**Depends on:** - start of phase

**Prompt for developer:**
> Add `"marked"` (pin a current major, e.g. `^12`) to `devDependencies` in
> [`package.json`](../../../../extension/package.json). In
> [`build.mjs`](../../../../extension/build.mjs), add `async function vendorMarked()` modelled on
> `vendorPdfjs`: copy `node_modules/marked/lib/marked.esm.js` to `vendor/marked.esm.js` and
> `node_modules/marked/LICENSE.md` to `vendor/LICENSE.marked`, and log the vendored version.
> Call `vendorMarked()` from the `vendor` command branch (after `vendorTesseract()`). The `zip`
> `PACKAGE` allow-list already includes `vendor`, so no change there.

**Verification:**
- `marked` appears in `package.json` `devDependencies`.
- `async function vendorMarked` matches once in `build.mjs`.
- `vendorMarked()` is called in the `vendor` command branch.

**Status:** `[x]` done

---

### Step 03.2 — Markdown parser
**Files:** `extension/src/md.js` (New)
**Depends on:** Step 03.1, Phase 01 Step 01.1 (`sanitize.js`)

**Prompt for developer:**
> Create `extension/src/md.js` exporting `async function parseMarkdown(data)`. Decode bytes as
> UTF-8, convert to HTML with `marked` (`import { marked } from "../vendor/marked.esm.js"`),
> run the HTML through `sanitizeToFragment` (index 0), then split the resulting fragment into
> sections at each `H1`/`H2` boundary (mirror the heading split in
> [`internal/md/extract.go`](../../../../internal/md/extract.go) `splitBySections`): accumulate
> nodes into the current section, starting a new section whenever an `H1`/`H2` is encountered.
> Give each section `id:"md-sec-<i>"` and a `label` from its leading heading, and build a flat
> `toc` of `{ title:<heading>, anchor:"md-sec-<i>", children:[] }`. Return the book shape
> (`title:""`, `lang:""`, `sampleText`, `sections`, `toc`, `revoke:()=>{}`).

**Verification:**
- `export async function parseMarkdown` matches once.
- `../vendor/marked.esm.js` is imported.
- `sanitizeToFragment` is imported.

**Status:** `[x]` done

---

### Step 03.3 — FB2 parser (port the Go XML extractor)
**Files:** `extension/src/fb2.js` (New)
**Depends on:** - start of phase

**Prompt for developer:**
> Create `extension/src/fb2.js` exporting `async function parseFb2(data)`. Decode as UTF-8,
> parse with `DOMParser` (`application/xml`). Port [`internal/fb2/extract.go`](../../../../internal/fb2/extract.go):
> read `<description><title-info><book-title>` (title) and `<lang>` (lang); walk `<body>`
> `<section>` recursively; each `<section>` becomes one output section, its `<title>` text the
> label, its `<p>` elements `<p>` nodes (set via `textContent`). Handle images: collect every
> `<binary id content-type>` (base64) into a map, and rewrite each `<image>` whose
> `l:href`/`xlink:href` is `#<id>` to an `<img src="data:<type>;base64,<data>">`. Build a
> nested `toc` from the section titles (`anchor:"fb2-sec-<i>"`). Return the book shape. Because
> images are inlined as `data:` URLs, `revoke` is a no-op.

**Verification:**
- `export async function parseFb2` matches once.
- The string `data:` (image inlining) is present in `fb2.js`.
- `fb2-sec-` (section id scheme) is present.

**Status:** `[x]` done

---

### Step 03.4 — Wire dispatch, detection, and file picker
**Files:** `extension/src/viewer.js`, `extension/src/viewer.html`
**Depends on:** Step 03.2, Step 03.3

**Prompt for developer:**
> In [`viewer.js`](../../../../extension/src/viewer.js): import `parseMarkdown` and `parseFb2`.
> Extend `FORMAT_EXT` with `md:"md", markdown:"md", fb2:"fb2"`. Add dispatch cases in
> `loadFromData`: `"md"` -> `loadBook(data, title, parseMarkdown, "Reading Markdown..")`,
> `"fb2"` -> `loadBook(data, title, parseFb2, "Reading FB2..")`. In
> [`viewer.html`](../../../../extension/src/viewer.html), extend `#file-input` `accept` to add
> `.md,.markdown,.fb2` (the strip regex was already widened in Phase 02).

**Verification:**
- `parseMarkdown` and `parseFb2` are imported in `viewer.js`.
- `FORMAT_EXT` contains `md` and `fb2`.
- `loadFromData` routes to `parseMarkdown` and `parseFb2`.
- `viewer.html` `accept` contains `.md` and `.fb2`.

**Status:** `[x]` done

---

### Step 03.5 — Package the modules and intercept FB2 URLs
**Files:** `extension/manifest.json`, `extension/src/background.js`
**Depends on:** Step 03.4

**Prompt for developer:**
> Add `"src/md.js"` and `"src/fb2.js"` to `web_accessible_resources[0].resources` in
> [`manifest.json`](../../../../extension/manifest.json). In
> [`background.js`](../../../../extension/src/background.js), add `fb2` to the DNR `regexFilter`
> alternation in both rules (`(?:pdf|epub|rtf|fb2)`). Do NOT add `md`/`markdown` (rarely served
> as a document; reachable via the file picker).

**Verification:**
- `manifest.json` contains `src/md.js` and `src/fb2.js`.
- `background.js` DNR `regexFilter` contains `fb2`.

**Status:** `[x]` done

## Phase done criteria
- [ ] Every `Step 03.*` is `[x] done`.
- [ ] `cd extension && npm install && npm run vendor` produces `vendor/marked.esm.js`. Name the
      command here.
- [ ] `cd extension && npm test` passes (no regression).
- [ ] Manual smoke: open a `.md` and a `.fb2` (one with an embedded image) via the file picker
      - both render reflowed with a working TOC.
- [ ] Grep for `TODO(phase-03)` returns zero hits.
- [ ] `DEV/CHANGELOG.md` entry added for every file in "Files touched".

## Handoff notes
- `marked` is the second vendored runtime library after PDF.js/Tesseract; the vendor step now
  produces it. FB2 uses no external library (pure DOMParser), matching the pure-Go extractor.
- FB2/MD render paths are DOM-based, so they have no `node --test` coverage (same constraint as
  epub.js DOM code); covered by manual smoke.

## Rollback plan
Revert the phase commit(s). Remove `vendor/marked.esm.js` if regenerating vendor output.
