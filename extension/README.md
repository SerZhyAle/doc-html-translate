# PDF / EPUB -> translatable HTML (browser extension)

A Chromium MV3 extension that intercepts PDF **and EPUB** opens and re-renders them as clean,
semantic HTML so the browser's built-in **Translate page** works on them for free (no API key, no invoice, no catch).
Spin-off of [doc-html-translate](../README.md); it ports that app's proven PDF reflow and EPUB
extraction heuristics to run in-browser (PDF.js for PDF, a native unzip + DOM pipeline for EPUB).

**Available now on the [Chrome Web Store](https://chromewebstore.google.com/detail/nmcckamdocainafmmompkbmelkpbnmic)** (Chrome and Edge / Chromium). An Edge Add-ons listing is still planned; you can also load it unpacked from this folder (see below). This extension is one of several forms of the same project - the desktop CLI/GUI, the Microsoft Store app, and this extension; see [Editions](../README.md#editions).

The make-or-break decision for PDF (see [the spec](../DEV/research/pdf_translate_extension_spec.md)):
reflow to flowing `<p>` / `<h2>` text, **not** the PDF.js canvas + text-layer overlay. The overlay
leaves the original glyphs under transform-scaled spans, which native translate doubles/garbles. Clean
reflow loses exact layout but yields a DOM the browser translates perfectly.

EPUB is simpler, for once: its content is *already* semantic XHTML, so there is no reflow heuristic. The work is
unzipping the archive (native `DecompressionStream`, no dependency), following the OPF spine in reading
order, sanitizing each chapter (drop scripts/styles, strip inline styles and event handlers), rewriting
its images to `blob:` URLs and its internal links to in-page anchors, and reading the authored TOC
(EPUB3 `nav.xhtml` or EPUB2 `toc.ncx`). All chapters are combined into one scrollable document so native
translate sees the whole book at once.

## Layout

```text
manifest.json          MV3 manifest
src/
  background.js        service worker: declarativeNetRequest interception + on/off toggle + "open original"
  viewer.html/.js/.css viewer (the redirect target); sniffs PDF vs EPUB and routes to the right reader
  reflow.js            PDF paragraph/heading heuristics ported from internal/pdf/extract.go
  toc.js               PDF outline -> nested TOC, ported from internal/pdf/toc.go
  epub.js              EPUB: unzip + OPF/spine + nav/NCX TOC + chapter sanitize, ported from internal/epub
  lang.js              source-language detection -> <html lang>
  popup.html/.js       toolbar: global + per-site toggle + "Use OCR for images" + language downloads
  options.html/.js     defaults: on/off, theme, source-language hint, image OCR + language manager
  ocr-lang.js          OCR languages: bundled English, on-demand download + IndexedDB cache
  ocr-overlay.js/.css  shared OCR unit: recognize -> opaque translatable plates over the image
  ocr.html/.js         standalone page for the right-click "OCR & translate this image" action
  pdf-images.js        pull raster images out of a PDF page (embedded XObjects + scanned-page raster)
vendor/                pdfjs-dist + tesseract/ (engine + eng.traineddata) - generated, git-ignored
icons/                 16/32/48/128
test/                  node:test unit tests for the pure modules (reflow/toc/lang + epub unzip/path)
build.mjs              `vendor` (sync pdfjs) and `zip` (store package)
```

## Develop

```sh
npm install        # pulls pdfjs-dist + tesseract.js / tesseract.js-core (dev dependencies)
npm run vendor     # copies pdfjs + the Tesseract engine and eng.traineddata into vendor/ (run once after install)
npm test           # unit-tests reflow.js / toc.js / lang.js (no browser needed)
npm run zip        # builds dist/doc-html-translate-extension.zip (store-ready)
```

`vendor/` is generated and git-ignored; run `npm run vendor` before loading the extension.

## Load unpacked (Chrome / Edge)

1. Go to `chrome://extensions` (or `edge://extensions`), enable **Developer mode**.
2. **Load unpacked** -> select this `extension/` folder.
3. For local PDFs / EPUBs you have two options:
   - **Easiest / most robust:** open `chrome-extension://<id>/src/viewer.html` and click **Open file**
     to pick a PDF or EPUB from the OS dialog. This needs no toggle and no host access, and - because the
     file path never appears in any URL - it is immune to URL-based content blockers (a common cause of an
     `ERR_BLOCKED_BY_CLIENT` on the viewer page).
   - **Auto-intercept:** open the extension's **Details** and turn on **Allow access to file URLs**, then
     open a `file://...pdf` or `file://...epub` directly and it redirects to the viewer.

## Manual acceptance (the spec's gates - need a real browser)

The pure heuristics are covered by `npm test`. The end-to-end gates are manual:

- **Step 1 (PoC gate).** Open `chrome-extension://<id>/src/viewer.html?file=<pdf-url>` directly on a few
  text PDFs. Run **Translate page**. The whole document - including text that was off-screen at load -
  must translate with no doubled, overlapping, or scaled text. This de-risks the whole project.
- **Step 2 (interception).** Open any web or local `*.pdf` or `*.epub`; the viewer should auto-load.
  Toggle the extension off in the popup and confirm the native PDF viewer / EPUB download returns. Use
  **Original PDF** in the toolbar to open the untouched PDF in the native viewer without turning the
  extension off (EPUB has no native viewer, so that button is hidden for EPUBs).
- **Step 3 (reflow quality).** Paragraphs, headings, and the TOC should read comparably to the desktop
  app on the same PDFs.
- **Step 4 (reading UX).** Change font size/family/theme, jump pages, watch progress on a large PDF.
- **Step 5 (hardening).** Large, corrupt, password-protected, and scanned PDFs each give a correct
  result or a clear fallback (no crashes/hangs). Scanned PDFs show a "little or no text" notice that
  points back to the desktop OCR flow.

## Translate text in images (OCR)

Text baked into images (scanned pages, comics, screenshots) is invisible to the browser translator. The
extension can OCR it **on-device** (Tesseract, bundled) and lay the recognized text over the image as
real, translatable HTML, so one **Translate page** covers pictures too. Three ways in:

- **Any web image:** right-click -> **OCR & translate this image**. Opens a new tab, OCRs the image,
  overlays the text, and sets `<html lang>` so the browser offers Translate page.
- **PDFs / EPUBs:** turn on **Use OCR for images** (in the toolbar popup, next to *Open file*, or in
  Options). Images are OCR'd lazily as they scroll into view; an "OCR: done/total" counter shows progress
  so a picture-heavy document never looks frozen.

**English is bundled** and works fully offline. Other languages (Russian, Ukrainian, Japanese, ..) are
optional: they download on demand from the tessdata_fast host and cache locally for reuse - the only new
network access, and only when you click **Download**. See [`store/PRIVACY.md`](store/PRIVACY.md).

## Known limitations

- Interception matches the **request URL** (`*.pdf` / `*.epub`), so files served without a matching
  extension in the URL aren't auto-intercepted (open them via **Open file** in the viewer). DNR can't
  see the response Content-Type before the request.
- PDF.js text extraction is weaker than Poppler on ligatures / non-standard font maps (same caveat the
  desktop app notes for its pure-Go reader - exotic fonts remain a humbling experience).
- Scanned/image-only PDFs are out of scope (OCR); the viewer detects them and offers the original.
- EPUB: the viewer applies its own clean reading CSS and drops author stylesheets/inline styles, so
  heavily designed books lose their exact layout (by design - that is what makes them translate cleanly).
  ZIP64 archives are not supported (vanishingly rare for EPUB). DRM-protected EPUBs cannot be read.
- Firefox and mobile are out of scope (different PDF + interception story).
- With OCR **off**, the viewer is text-extraction only (`getTextContent()`) and never rasterizes pages.
  OCR (Tesseract) uses WebAssembly, and scanned-PDF OCR rasterizes the page, so the manifest sets
  `"content_security_policy": { "extension_pages": "script-src 'self' 'wasm-unsafe-eval'; object-src 'self'" }`
  (`minimum_chrome_version` is 105, comfortably >= 103).
- OCR accuracy depends on image quality and the chosen language; stylized/vertical text may recognize
  imperfectly. Text is grouped by the engine's blocks, not by detected speech bubbles.

## Store packaging & publishing

`npm run zip` produces a minimal-permission, store-ready ZIP. The extension is **published on the
[Chrome Web Store](https://chromewebstore.google.com/detail/nmcckamdocainafmmompkbmelkpbnmic)**; an Edge
Add-ons listing is still planned. Listing copy and review-form answers live in
[`store/LISTING.md`](store/LISTING.md), the hosted privacy policy in [`store/PRIVACY.md`](store/PRIVACY.md)
(served as `extension-privacy.html`), and the release/automation steps in [`PUBLISHING.md`](PUBLISHING.md).
Selling point for the listing: everything runs locally; only the browser's own translate touches the
network, and even that waits until you ask.
