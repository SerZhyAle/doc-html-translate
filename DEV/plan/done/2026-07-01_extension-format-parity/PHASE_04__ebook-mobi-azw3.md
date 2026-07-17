# Phase 04 — MOBI and AZW3 via vendored foliate-js

**Strategic spec:** [`../2026-07-01_extension-format-parity.md`](../2026-07-01_extension-format-parity.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done (code + `npm run vendor` + `npm test` + import-smoke; manual acceptance on real .mobi/.azw3 pending user)
**Depends on:** Phase 01
**Steps done:** 5 / 5

## Objective
Add the binary Kindle ebook formats (MOBI and KF8/AZW3) by vendoring the minimal foliate-js
module set and adapting its book interface to the extension's book shape. This replaces the
desktop's Calibre dependency with client-side parsing.

## Prerequisites
- [ ] Phase 01 is ✅ Done.
- [ ] Read the strategic §6 open item (minimal module set) and §9 ADR-2 - it is resolved by
      Step 04.1 (following the import graph is the real work, not a separate spike).
- [ ] Have at least one real `.mobi` and one real `.azw3` file for acceptance.

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `extension/build.mjs` | Modified | ≤ 45 delta |
| `extension/package.json` | Modified | ≤ 3 delta |
| `extension/src/ebook.js` | New | ≤ 180 |
| `extension/src/viewer.js` | Modified | ≤ 20 delta |
| `extension/src/viewer.html` | Modified | ≤ 3 delta |
| `extension/manifest.json` | Modified | ≤ 4 delta |
| `extension/src/background.js` | Modified | ≤ 2 delta |

## Steps

### Step 04.1 — Vendor the minimal foliate-js module set
**Files:** `extension/build.mjs`, `extension/package.json`
**Depends on:** - start of phase

**Prompt for developer:**
> Add foliate-js as a pinned dependency: prefer `foliate-js` on npm pinned to an exact version
> if it is johnfactotum's package; otherwise vendor from a pinned GitHub commit. Add
> `async function vendorFoliate()` to [`build.mjs`](../../../../extension/build.mjs) that copies
> into `vendor/foliate/` the MOBI/KF8 entry module (`mobi.js`) plus its transitive **local**
> imports only - follow each `import` in `mobi.js` and copy the referenced module, repeating
> until the set is closed (expected: `mobi.js` and a small number of helpers; `fflate` for KF8
> font decompression). Vendor `fflate` (its ESM build) into `vendor/` as well and copy the
> foliate MIT `LICENSE` to `vendor/LICENSE.foliate`. Call `vendorFoliate()` from the `vendor`
> command branch. Record the exact vendored file list in this phase's Handoff notes.

**Verification:**
- `foliate-js` (or the pinned source) is declared in `package.json`.
- `async function vendorFoliate` matches once in `build.mjs` and is called in the `vendor` branch.
- After `npm install && npm run vendor`, `vendor/foliate/mobi.js` exists.

**Status:** `[x]` done

---

### Step 04.2 — Ebook adapter: sections + TOC
**Files:** `extension/src/ebook.js` (New)
**Depends on:** Step 04.1, Phase 01 Step 01.1 (`sanitize.js`)

**Prompt for developer:**
> Create `extension/src/ebook.js` exporting `async function parseEbook(data)` and a pure
> `export function isMobiBytes(bytes)` (true when the PDB type+creator at offset 60 is
> `BOOKMOBI`/`TPZ3`, covering both MOBI and KF8/AZW3). `parseEbook` wraps the byte buffer in a
> `Blob`/`File`, opens it with the vendored foliate MOBI module (inspect the vendored module's
> exports for the open function), and maps the returned foliate book to the extension book
> shape: for each foliate section call `createDocument()` to get its `Document`, pass its
> `<body>` innerHTML through `sanitizeToFragment(html, i)` to build `frag`, and push
> `{ id:"ebook-sec-<i>", label:<section heading or foliate label>, frag }`. Map foliate `.toc`
> (`.label`/`.href`/`.subitems`) to the extension `{ title, anchor, children }` tree, resolving
> each href to the matching `ebook-sec-<i>` anchor. Set `title`/`lang` from foliate
> `.metadata`. Leave image resolution to Step 04.3.

**Verification:**
- `export async function parseEbook` and `export function isMobiBytes` each match once.
- `sanitizeToFragment` is imported.
- `ebook-sec-` id scheme present.

**Status:** `[x]` done

---

### Step 04.3 — Ebook adapter: image resources + revoke
**Files:** `extension/src/ebook.js`
**Depends on:** Step 04.2

**Prompt for developer:**
> Extend `parseEbook` to resolve in-book images. For each section `Document`, rewrite every
> `<img>` whose source points at an in-archive resource to a `blob:` URL built from the bytes
> the foliate book exposes for that resource (inspect the vendored module for its resource
> accessor). Cache one blob URL per resource, drop `<img>` elements whose resource cannot be
> resolved, and return a `revoke()` that revokes every minted blob URL (mirroring
> [`epub.js`](../../../../extension/src/epub.js) `blobFor`/`revoke`). `renderBook` already stores
> and calls `book.revoke` on teardown.

**Verification:**
- `URL.createObjectURL` and `URL.revokeObjectURL` both appear in `ebook.js`.
- `parseEbook` returns an object whose `revoke` is a function (grep for `revoke`).

**Status:** `[x]` done

---

### Step 04.4 — Wire dispatch, detection, and file picker
**Files:** `extension/src/viewer.js`, `extension/src/viewer.html`
**Depends on:** Step 04.3

**Prompt for developer:**
> In [`viewer.js`](../../../../extension/src/viewer.js): import `parseEbook` and `isMobiBytes`.
> Extend `FORMAT_EXT` with `mobi:"mobi", azw3:"mobi"` (both route to the one adapter; foliate
> distinguishes MOBI vs KF8 internally). Add a magic-byte check to `detectFormat` using
> `isMobiBytes(data)` -> `"mobi"`. Add the dispatch case `"mobi"` ->
> `loadBook(data, title, parseEbook, "Reading e-book..")`. In
> [`viewer.html`](../../../../extension/src/viewer.html), extend `#file-input` `accept` to add
> `.mobi,.azw3`. Now that the full set is wired, update the user-facing copy in `viewer.js`
> that still says "PDF or EPUB" - the empty-state notice in `main()`, the
> `filePickerButton`/`openFilePicker` default labels, and the `#btn-open` title - to name the
> wider document set (e.g. "Open a document (PDF, EPUB, MOBI, FB2, RTF, TXT, MD, HTML)").

**Verification:**
- `parseEbook` and `isMobiBytes` are imported in `viewer.js`.
- `FORMAT_EXT` contains `mobi` and `azw3`.
- `loadFromData` routes `"mobi"` to `parseEbook`.
- `viewer.html` `accept` contains `.mobi` and `.azw3`.
- `viewer.js` no longer contains the literal "PDF or EPUB" (copy generalized).

**Status:** `[x]` done

---

### Step 04.5 — Package the module and intercept MOBI/AZW3 URLs
**Files:** `extension/manifest.json`, `extension/src/background.js`
**Depends on:** Step 04.4

**Prompt for developer:**
> Add `"src/ebook.js"` to `web_accessible_resources[0].resources` in
> [`manifest.json`](../../../../extension/manifest.json) (the vendored foliate files are covered
> by the existing `vendor/*` entry). In [`background.js`](../../../../extension/src/background.js),
> add `mobi` and `azw3` to the DNR `regexFilter` alternation in both rules
> (`(?:pdf|epub|rtf|fb2|mobi|azw3)`).

**Verification:**
- `manifest.json` contains `src/ebook.js`.
- `background.js` DNR `regexFilter` contains `mobi` and `azw3`.

**Status:** `[x]` done

## Phase done criteria
- [ ] Every `Step 04.*` is `[x] done`.
- [ ] `cd extension && npm install && npm run vendor` produces `vendor/foliate/mobi.js`. Name
      the command here.
- [ ] `cd extension && npm test` passes (includes any `isMobiBytes` test if added).
- [ ] Manual acceptance: open a real `.mobi` and a real `.azw3` via the file picker - text
      renders reflowed, TOC works, and images appear (or degrade cleanly when unresolved).
      Confirm a DRM-protected file fails with a clear notice (out of scope, must not hang).
- [ ] Grep for `TODO(phase-04)` returns zero hits.
- [ ] `DEV/CHANGELOG.md` entry added for every file in "Files touched".

## Handoff notes
- **Vendored file list (resolves strategic §6 Q1):** `vendor/foliate/mobi.js` (self-contained -
  no local foliate imports; receives `unzlib` via its constructor) + `vendor/foliate/LICENSE.foliate`,
  and `vendor/fflate.js` (fflate ESM `esm/browser.js`, for KF8 font decompression) +
  `vendor/fflate.LICENSE`. No other foliate modules are needed. foliate-js@1.0.1, fflate@0.8.3
  pinned in `package.json` devDependencies.
- The adapter uses `section.load()` (uniform for MOBI6 + KF8: resolves images to blob: URLs and
  injects styles), fetches the blob, and sanitizes the body. `revoke()` frees the collected image
  blob: URLs and calls `book.destroy()` when present (KF8).
- foliate-js is upstream-tagged "not stable": the vendored version is pinned; re-vendoring is a
  deliberate, re-tested action, never an automatic bump.
- MOBI/AZW3 both flow through `parseEbook`; there is intentionally no separate AZW3 module.

## Rollback plan
Revert the phase commit(s). Remove `vendor/foliate/` and the vendored `fflate` if regenerating
vendor output. Simple formats (Phases 02-03) are unaffected.
