# Research 01 — OCR stack technical decisions

Resolves strategic §6 open questions so the tactical plan can start. Grounded in the known
behavior of Tesseract.js v5/v6 and pdf.js v4 (the vendored engine).

## Q2 — Language-pack source and integrity (RESOLVED)

- **Engine core:** `tesseract.js` (JS glue) + `tesseract.js-core` (WASM). Both vendored into
  `vendor/tesseract/` at build time; nothing fetched at runtime for the engine itself.
- **English data:** ship `eng.traineddata` (from `tessdata_fast`) inside `vendor/tesseract/lang/`.
  The worker is created with `langPath` pointing at that local directory, so English OCR is fully
  offline — no network, satisfying the privacy promise.
- **Other languages:** downloaded on explicit user action from the canonical Tesseract.js CDN
  `https://tessdata.projectnaptha.com/4.0.0_fast` (this is tesseract.js's own default host for
  `tessdata_fast`; `.traineddata.gz`). Sizes are modest for `fast` (~1-10 MB/lang; `jpn` largest).
- **Caching + integrity:** rely on Tesseract.js's built-in IndexedDB cache (`cacheMethod: "refresh"`
  on first fetch, `"none"`/read for reuse). A language counts as "installed" only after a successful
  worker init with it; on failure the UI shows the language as available (not installed) and the
  error. Downloaded data persists across sessions and is cleared on uninstall (IndexedDB is
  per-extension-origin).
- **Privacy consequence:** the only new outbound request is the user-triggered language download to
  the CDN above. Disclose it in PRIVACY.md and store copy.

## Q1 — PDF image extraction fidelity (RESOLVED for MVP)

- **Primary path:** per page, call `page.getOperatorList()` and walk for `OPS.paintImageXObject` /
  `OPS.paintInlineImageXObject`, resolving the image object via `page.objs`/`page.commonObjs`. Each
  yields pixels (RGBA) + width/height; the transform on the op stack gives on-page placement/scale.
- **Scanned-page fallback:** if a page has effectively one full-page image (or extraction yields
  nothing but the page is image-dominant), rasterize the page region with the existing pdf.js render
  path and OCR that bitmap.
- **MVP scope:** OCR the extracted raster images and present each overlay inline with the reflowed
  text for that page (placement precision on the original page is not required — the reflow view is
  already a re-layout, not a pixel-faithful copy). Sequenced as the last implementation phase.

## Q3 — Deferred-OCR trigger for documents (RESOLVED)

- **Trigger:** visibility-based lazy OCR via `IntersectionObserver`. An image is queued for OCR only
  when it scrolls into view, and only if the user's "Use OCR for images" setting is on.
- **Concurrency:** a single shared worker processes a FIFO queue (one image at a time) so an image-
  heavy document never spawns parallel OCR or blocks reading of already-reflowed text.
- **Reassurance:** each pending image shows a per-image progress badge; the toolbar status shows an
  overall "OCR: N/M images" counter while any are queued/running.

## Q4 — Overlay text fitting and readability (RESOLVED for MVP)

- **Plate:** one absolutely-positioned element per recognized block, sized to the block bbox
  (percent of image dimensions so it survives responsive scaling), opaque background sampled toward
  the image's local background (default near-white/near-black by luminance), covering the source text.
- **Text fitting:** binary-search / clamp the font-size so the source text fits the box; allow the
  box to grow downward (min-height = bbox, height auto) so post-translation text (often longer) wraps
  instead of clipping. Real DOM text so the browser translator replaces it in place.
- **Language tag:** set `document.documentElement.lang` to the OCR source language so Chrome offers
  "Translate page". In the viewer, the existing lang detection already sets it; OCR must not override
  a correct document lang with a per-image guess unless the document had none.
