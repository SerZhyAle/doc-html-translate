# PDF / EPUB -> translatable HTML (browser extension)

A Chromium MV3 extension that intercepts PDF **and EPUB** opens and re-renders them as clean,
semantic HTML so the browser's built-in **Translate page** works on them for free (no API key, no invoice, no catch).
Spin-off of [doc-html-translate](../README.md); it ports that app's proven PDF reflow and EPUB
extraction heuristics to run in-browser (PDF.js for PDF, a native unzip + DOM pipeline for EPUB).

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
  popup.html/.js       toolbar: global + per-site toggle
  options.html/.js     defaults: on/off, theme, source-language hint
vendor/                pdfjs-dist (pdf.mjs, pdf.worker.mjs, cmaps, standard_fonts) - generated, git-ignored
icons/                 16/32/48/128
test/                  node:test unit tests for the pure modules (reflow/toc/lang + epub unzip/path)
build.mjs              `vendor` (sync pdfjs) and `zip` (store package)
```

## Develop

```sh
npm install        # pulls pdfjs-dist (dev dependency)
npm run vendor     # copies the pieces of pdfjs we ship into vendor/  (run once after install)
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
- The viewer is **text-extraction only** (`getTextContent()`); it never rasterizes pages. The bundled
  PDF.js worker contains a WASM JPEG2000 decoder that the default MV3 CSP would block, but that path is
  never reached. If you ever add canvas rendering / image decoding, add
  `"content_security_policy": { "extension_pages": "script-src 'self' 'wasm-unsafe-eval'; object-src 'self'" }`
  to the manifest (and keep `minimum_chrome_version` >= 103).

## Store packaging (Steps 6-7)

`npm run zip` produces a minimal-permission, store-ready ZIP. Publishing (Chrome Web Store $5
one-time registration, Edge Add-ons free, listing assets, privacy policy) is done from the developer
consoles and is out of scope for this repo drop. Selling point for the listing: everything runs
locally; only the browser's own translate touches the network, and even that waits until you ask.
