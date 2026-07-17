# PDF / EPUB -> translatable HTML browser extension

**Status:** Verified

> **Closed 2026-07-17.** Steps 0-5 (the code drop this ticket tracks) are done and verified below.
> Steps 6-7 were scoped out as "needs accounts" and have since happened in reality: the extension is
> published on the Chrome Web Store and Edge Add-ons and has shipped through several releases. Later
> work on the same codebase moved to its own tickets - `2026-07-01_extension-format-parity` (the other
> seven formats) and `2026-07-01_ocr-image-overlay` (OCR).

Tactical plan for the spin-off browser extension specified in
[DEV/research/pdf_translate_extension_spec.md](../../research/pdf_translate_extension_spec.md).
This ticket tracks the implementation of Steps 0-5 (a loadable, store-shaped MV3 extension);
Steps 6-7 (store submission, $5 registration, listing assets) need accounts and are out of scope
for the code drop and are only scaffolded.

## Goal

A Chromium MV3 extension that intercepts PDF opens (`https` + `file://`) and re-renders them as
clean, reflowed semantic HTML (`<p>` / `<h2>` / `<h3>`) so Chrome's built-in "Translate page" works
for free. Reflow, not the PDF.js canvas + text-layer overlay (the make-or-break decision, spec sec 1).

## Architecture (everything under `extension/`)

```
extension/
  manifest.json          MV3: permissions, web_accessible_resources, background sw
  src/
    background.js        service worker - dynamic declarativeNetRequest rules + on/off toggle
    viewer.html          reflow viewer shell
    viewer.js            orchestrator: ?file= -> pdfjs getDocument -> reflow -> render -> TOC -> toolbar
    viewer.css           reading styles (mirrors internal/pdf buildPDFPageHTML CSS)
    reflow.js            PURE port of rowsToText + classifyBlock + isDoubleSpacedLayout
    toc.js               PURE port of buildPDFTOC (PDF.js getOutline -> nested entries)
    lang.js              source-language detection -> <html lang> (drives Chrome translate offer)
    options.html/js      default on/off, source-language hint, theme
    popup.html/js        toolbar: global + per-site toggle, open options
  vendor/                copied from pdfjs-dist build (pdf.mjs, pdf.worker.mjs, cmaps, standard_fonts)
  icons/                 16/32/48/128
  test/reflow.test.mjs   node:test for the heuristic port (runnable without Chrome)
  build.mjs              vendor-sync + zip packager
  package.json           dev dep: pdfjs-dist; scripts: vendor, test, zip
```

### Reuse map (Go -> JS)
- `rowsToText` (Y-gap > 1.5x median + first-word X-indent) -> `reflow.detectParagraphs`
- `classifyBlock` (ALL-CAPS / centered / short -> h2/h3) -> `reflow.classifyBlock`
- `isDoubleSpacedLayout` + `isLigaturesArtifact` -> `reflow` helpers
- `buildPDFTOC` (outline -> nested) -> `toc.outlineToEntries` (PDF.js `getOutline()` is already nested)

PDF.js `getTextContent()` yields positioned items (`transform`, `width`, `str`, `fontName`), which
maps to the `rowsToText` model (positioned words) far better than the pdftotext `-layout` model, so
centering is computed from real X coords, not leading-space counts.

## Interception design
- Dynamic DNR rules built in `background.js` from `chrome.runtime.getURL('src/viewer.html')`, so no
  hardcoded extension id. One rule for `^https?://.*\.pdf(\?.*)?$`, one for `file://.*\.pdf`, both
  `main_frame`, action redirect to `viewer.html?file=\0` via `regexSubstitution`.
- Limitation (noted, spec sec 4/7): DNR matches the request URL, so PDFs served as
  `application/pdf` without a `.pdf` URL are not intercepted. `.pdf` URLs + `file://` are.
- Toggle: storage-backed global on/off + per-host disable list; service worker rebuilds the rule set
  on change. `file://` requires the user "Allow access to file URLs" toggle.

## Build steps (gates from the spec)
Code for Steps 0-5 is implemented and the pure modules are unit-tested; the end-to-end
acceptance gates (marked GATE) are in-Chrome manual and remain for the user to run.
- [x] Step 0 Scaffold - manifest, vendored pdfjs, icons, viewer files (load-unpacked ready)
- [~] Step 1 PoC GATE - getTextContent over all pages -> reflow + `<html lang>` (code done; **manual gate**)
- [~] Step 2 Interception - dynamic DNR redirect + global/per-site toggle + open-original bypass (code done)
- [~] Step 3 Reflow quality - rowsToText/classifyBlock/isLigaturesArtifact + outline TOC ported (code done)
- [~] Step 4 Reading UX - toolbar (font/family/theme/page-jump/original), progress, options page (code done)
- [~] Step 5 Hardening - password/corrupt/scanned fallbacks, incremental render, all text in DOM (code done)

## Review pass (2026-06-25)
Ran a 5-dimension adversarial static review (MV3/DNR, PDF.js integration, reflow fidelity, viewer
security, store risk); each high-severity finding independently re-verified. No confirmed blockers.
Applied fixes: per-tab DNR bypass id allocator (no cross-tab collision); reworked reflow centering
(symmetric inset + centering-change paragraph breaks so wrapped centered headings group and indented
body openers aren't mis-tagged; font-size triggers gated on non-full-width); viewer scheme allow-list +
refuse-to-run-framed (SSRF hardening) + density-based scanned detection; allow-list zip packaging with a
vendor-present check; `minimum_chrome_version` 105; accurate permission justification. Accepted as
low-risk/documented: broad `<all_urls>` (needed for cross-origin PDF fetch), top-level-nav SSRF surface
(result not attacker-readable), bundled WASM JPEG2000 path (never reached by the text-only viewer).

## Verification
- `reflow.js`/`toc.js`/`lang.js` are pure and tested via `node --test` (18/18 pass, no browser).
- **Engine proven on real PDFs (2026-06-26):** ran the actual pdfjs + reflow modules over two real
  PDFs (a 33-page story, a 4-page business letter). Output was clean and correctly structured
  (433 p / 16 h2 / 1 h3 and 72 p / 1 h3), language auto-detected `en`, headings sensible, and the
  adjacent-duplicate-word ratio was **0.0000** - i.e. no doubled/garbled text, the exact failure mode
  the Step 1 gate guards against. A standalone reflow preview renders cleanly and is translate-ready.
- **In-Chrome auto-load blocked by environment, not code:** Chrome 149 disables command-line
  `--load-extension` (and the `DisableLoadExtensionCommandLineSwitch` escape hatch); the CDP
  `Extensions.loadUnpacked` path installs it but its pages return `ERR_BLOCKED_BY_CLIENT` and the MV3
  SW does not run under automation here. Manifest + DNR rule format are textbook-correct (and passed the
  adversarial review). The supported path - `chrome://extensions` -> Developer mode -> **Load unpacked**
  (a GUI action) - is unaffected and is how Steps 1-2/3-5 are accepted in a real browser.

## EPUB support (2026-06-26)

Extended the extension to give the same free-translate effect to EPUB, mirroring the desktop
`internal/epub` package but rendering into the DOM instead of writing files.

- `src/epub.js` (new): a dependency-free, central-directory ZIP reader (stored + deflate via the
  platform `DecompressionStream`); `parseContainer`/`parseOpf` (namespace-agnostic via
  `getElementsByTagNameNS("*", …)`); EPUB3 `nav.xhtml` + EPUB2 `toc.ncx` TOC parsing; and a
  per-chapter sanitize/rewrite pass (drop scripts/styles/handlers/inline-styles, images -> `blob:`
  URLs, single-image SVG cover -> `<img>`, internal links + ids namespaced `d<i>-…` and remapped to
  in-page anchors so the whole spine is one translatable document). `unzip`/`resolvePath` are pure and
  unit-tested; the DOM pieces are browser-only.
- `viewer.js`: `loadFromData` now sniffs the byte signature (`%PDF` vs `PK\x03\x04`) and routes to the
  PDF or EPUB reader; shared `applyLang`; TOC entries gained an `anchor` form; page-jump scrolls by
  `data-page` so the same nav UI drives EPUB chapters. The "Original PDF" button is hidden for EPUB.
- `background.js`: DNR regex widened to `\.(?:pdf|epub)` for both the https and file rules.
- `manifest.json`: `src/epub.js` added to `web_accessible_resources`; description/title mention EPUB.
- Reading CSS extended for richer EPUB markup (img/figure/blockquote/lists/tables/h1,h4-h6/pre).
- popup/options/README/store copy updated to say "PDF & EPUB".

**Verified (2026-06-26):** `unzip` (stored + deflate round-trip) and `resolvePath` unit-tested
(22/22 pass). Full render checked in real headless Chrome 149 against a crafted EPUB exercising the hard
cases: title + language read from the OPF; 3-section spine; nested TOC resolved to wrapper + namespaced
fragment anchors; both images served as `blob:` URLs; single-image SVG cover converted to `<img>`;
cross-chapter and same-document fragment links rewritten to resolvable in-page anchors (including a
`<body id>` re-exposed as a marker anchor - a bug found and fixed during this check); scripts, styles,
inline styles and `onclick` all stripped.

## Non-goals (spec sec 2)
OCR for scanned PDFs (fallback only), pixel-perfect layout, Firefox, mobile. For EPUB: preserving the
book's original CSS/design (replaced by the viewer's clean reading style by design), ZIP64, DRM.
