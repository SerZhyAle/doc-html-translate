# Specification: PDF / EPUB to translatable HTML browser extension

**Status:** Implemented (extension v0.1.0, source in [`extension/`](../../extension/)); store-ready ZIP builds
via `npm run zip`; pending the first manual store submission. See [extension/README.md](../../extension/README.md)
and [extension/PUBLISHING.md](../../extension/PUBLISHING.md) for the shipped detail; this file records the
strategic *what / why*.

Spin-off ("субпродукт") of doc-html-translate. A Chromium MV3 extension that intercepts **PDF and EPUB** opens
and re-renders them as clean, semantic HTML so the browser's built-in "Translate page" works on them for free
(no API key). Reuses this project's PDF reflow and EPUB extraction competency, but in-browser.

## 1. Core design decision (make-or-break)

For PDF: reflow to flowing `<p>` / `<h2>` text, **not** the PDF.js canvas + text-layer overlay. The overlay
keeps the original glyphs on a canvas underneath transform-scaled text spans, so native translate produces
doubled / garbled text. Clean reflow loses exact visual layout but gives a DOM the browser translates
perfectly - the same trade-off the desktop app makes in its "free workflow". This is the whole reason it works.

EPUB needs no reflow heuristic - its content is already semantic XHTML. The work is unzipping the archive,
following the OPF spine in reading order, sanitizing each chapter, rewriting images to `blob:` URLs and
internal links to in-page anchors, reading the authored TOC (EPUB3 `nav.xhtml` / EPUB2 `toc.ncx`), and
combining all chapters into one scrollable document so native translate sees the whole book at once.

## 2. Scope

- In: text-based PDFs and EPUBs, local (`file://`) and web (`https`), translate-friendly DOM, nav + TOC.
- Out: OCR for scanned PDFs (graceful fallback only), pixel-perfect layout, DRM EPUB, ZIP64, Firefox, mobile.

## 3. Decisions (locked)

1. Location: `extension/` in this repo. Shipped here, not a separate repo.
2. Branding: shares the doc-html-translate name (the extension is the desktop app's in-browser companion).
3. Stores: Chrome Web Store + Edge Add-ons (same MV3 ZIP; Edge aligns with the Microsoft / SZA publisher
   identity). Firefox out of scope.
4. Source-language handling: auto-detected and written to `<html lang>` so the browser offers the right translation.

## 4. Components (as built)

- `manifest.json` (MV3): permissions `declarativeNetRequest` + `storage`, `host_permissions <all_urls>`, web-accessible viewer.
- `src/background.js`: `declarativeNetRequest` interception of `*.pdf` / `*.epub` + on/off toggle + "open original".
- `src/viewer.html/.js/.css`: redirect target; sniffs PDF vs EPUB and routes to the right reader.
- `src/reflow.js`: PDF paragraph / heading heuristics ported from `internal/pdf/extract.go`.
- `src/toc.js`: PDF outline -> nested TOC, ported from `internal/pdf/toc.go`.
- `src/epub.js`: EPUB unzip + OPF/spine + nav/NCX TOC + chapter sanitize, ported from `internal/epub`.
- `src/lang.js`: source-language detection -> `<html lang>`.
- `src/popup.html/.js`, `src/options.html/.js`: toolbar (global + per-site toggle) and defaults.
- `vendor/`: pdfjs-dist (text extraction only; never rasterizes pages), generated and git-ignored.

## 5. Acceptance gates (manual, in a real browser)

Pure modules (`reflow` / `toc` / `lang` / `epub`) are unit-tested via `npm test`. End-to-end gates are manual:

1. **PoC gate (the make-or-break).** Open `viewer.html?file=<pdf-url>` on several text PDFs and run
   "Translate page": the whole document - including off-screen text - translates with no doubled, overlapping,
   or scaled text. Settled first; de-risks everything else.
2. **Interception.** Any web or local `*.pdf` / `*.epub` auto-loads the viewer; the toggle restores native
   behavior; "Original PDF" opens the untouched PDF (hidden for EPUB, which has no native viewer).
3. **Reflow quality.** Paragraphs, headings, and the TOC read comparably to the desktop app on the same PDFs.
4. **Reading UX.** Font size / family / theme, page jump, progress on large PDFs.
5. **Hardening.** Large, corrupt, password-protected, and scanned PDFs each give a correct result or a clear
   fallback (no crashes / hangs); scanned PDFs show a "little or no text" notice pointing to the desktop OCR flow.

## 6. Reuse map (Go -> JS)

| Need | Desktop (Go) | Extension (JS) |
| ---- | ------------ | -------------- |
| PDF text source | `pdftotext` / `ledongthuc/pdf` | PDF.js `getTextContent()` (vendored) |
| PDF paragraph detection | `rowsToText` (internal/pdf/extract.go) | `src/reflow.js` |
| PDF heading detection | `classifyBlock` (internal/pdf/extract.go) | `src/reflow.js` |
| PDF TOC | `internal/pdf/toc.go` | `src/toc.js` (PDF.js `getOutline()`) |
| EPUB unzip + spine + TOC + sanitize | `internal/epub` | `src/epub.js` (native `DecompressionStream`) |
| Translation | Chrome native (free flow) | Chrome native (same) |

## 7. Publishing

Store packaging and release are scripted and documented in [extension/PUBLISHING.md](../../extension/PUBLISHING.md):
`npm run zip` builds the store ZIP; first release on each store is manual (the stores' APIs only *update* an
existing item), then `npm run release:cws` / `release:edge` (or a CI tag) automate subsequent releases. Listing
copy and permission justifications live in `extension/store/LISTING.md`; the privacy policy is the rendered
`extension-privacy.html` (not the desktop app's `privacy.html`).

## 8. Risks

- Primary: cleanliness of native translate on the reflowed DOM - settled by the PoC gate.
- PDF.js text quality below Poppler (ligatures / non-standard font maps).
- Interception matches the request URL (`*.pdf` / `*.epub`); files served without a matching extension in the
  URL must be opened via "Open file" in the viewer (DNR can't see response Content-Type pre-request).
- `host_permissions <all_urls>` + `declarativeNetRequest` draws stricter / slower store review.
- Scanned PDFs out of scope (OCR); DRM EPUB and ZIP64 unsupported.
