# Strategic spec: 2026-07-01_ocr-image-overlay — OCR text overlay on images for in-browser translation

**Ticket:** 2026-07-01_ocr-image-overlay
**Status:** BlockNeedUserTest
**Priority:** 50

> **BlockNeedUserTest (2026-07-01):** All 8 tactical phases implemented; static checks (node --check,
> JSON validity, `npm run vendor`/`npm run zip`) pass. Needs hands-on verification in a real Chromium
> browser (WASM OCR cannot run here). Verify the §11 done criteria: (1) right-click a web image ->
> "OCR & translate this image" -> overlay + Translate page offered; (2) EPUB with image text, OCR on ->
> lazy overlay + Translate page; (3) PDF (embedded-image and scanned) with OCR on; (4) English works
> offline; (5) download another language in the popup/options, then reuse it in a later session; (6)
> progress counter advances for single image and multi-image documents; (7) toggle sits next to Open
> file with reachable language downloads; (8) with OCR off, documents open/read unchanged. Note: literal
> verification tags were not inserted (the kit's tag format is not available in this session); this note
> is the manual-test checklist instead.
**Date:** 2026-07-01
**Tier:** Strategic
**Tactical plan:** `DEV/plan/2026-07-01_ocr-image-overlay/` (created by /spec-tech)

> **Scope:** STRATEGIC. Goals, constraints, open questions. No class names, paths, line
> budgets, schema versions, or framework module details.

---

## 1. Problem

The extension makes the *text* of PDFs and EPUBs translatable by re-rendering it as clean HTML,
but any text that lives *inside an image* (scanned pages, comics/manga, screenshots, diagrams,
memes) is invisible to the browser's "Translate page" feature and stays untranslated. Users who
open image-heavy documents, or who simply find a picture with foreign text on any web page, have
no free, on-device way to read it in their language. There is currently no path from "image with
text" to "translatable HTML".

## 2. Goals

1. From a right-click on any image on any web page, the user can open that image in a new tab where
   the text baked into it is recognized and shown as real, selectable HTML text placed over the
   original, so the browser offers "Translate page" and translates it in place.
2. When reading a PDF or EPUB in the viewer, text inside the document's images is recognized and
   overlaid the same way, so a single "Translate page" covers both the reflowed text and the images.
3. Recognition works fully on-device with no API key for the bundled language (English), preserving
   the extension's "100% local" promise for the default case.
4. The user can add more recognition languages on demand (e.g. Japanese, Russian, Ukrainian) via an
   explicit control, and those languages persist for reuse without re-downloading.
5. During recognition the user always sees clear progress, so long-running OCR never looks frozen.
6. When picking a document via the extension's file-open UI, the user can turn image OCR on or off
   before opening, and reach the language-download controls from the same place.

**Non-goals:**
- True computer-vision speech-bubble / panel segmentation. First iteration groups text by the OCR
  engine's own block/paragraph boxes, not by detected bubbles.
- Editing, re-typesetting, or exporting the recognized text; the overlay exists only to enable
  in-browser translation and reading.
- Doing the translation ourselves. Translation remains the browser's built-in feature, invoked by
  the user, as today.
- Recognizing text in video, canvas-rendered page content, or vector text that is already real text.
- Automatic OCR of every image with no way to opt out (must be user-controllable for performance).

## 3. Wishes and constraints

### 3.1 Owner wishes

- Progress feedback should feel reassuring and specific (per-image and overall), not a spinner alone.
- Overlay should look clean — opaque plates covering the source text — matching the manga-translator
  convention the owner referenced.
- Keep the packaged extension as small as is reasonable; only English recognition data ships in the box.

### 3.2 Hard constraints

- **Platform / versions:** Chromium MV3, existing `minimum_chrome_version`. Service worker + web-
  accessible pages only; no persistent background.
- **Performance:** OCR is heavy. Image-heavy documents must not freeze the tab; recognition is
  deferred/queued and must not block reading of the already-reflowed text. OCR must not run on an
  image until the user has opted in (per §2.6) or the image is actually being viewed.
- **Data compatibility:** The shared settings object is extended additively; older settings without
  the new fields must keep working with sensible defaults. Downloaded language data persists locally
  across sessions and is removed on uninstall.
- **Privacy:** The bundled-English path stays fully on-device and downloads no code at runtime.
  Optional language packs are *data* fetched from a public source only after an explicit user action;
  this is a new outbound network access that must be disclosed in the privacy policy and store copy.
- **Localization:** New user-facing strings (context-menu item, OCR toggle, language-download UI,
  progress labels) follow the existing message-catalog approach for the supported locales.
- **Accessibility:** Overlaid text is real, selectable DOM text with adequate contrast; controls are
  keyboard-reachable and labeled.

### 3.3 Owner inputs (Approval gate)

- **OCR engine:** Tesseract.js, bundled and run on-device (owner-confirmed).
- **Bundled vs. downloaded languages:** English ships in the package; all other languages are
  downloaded on demand from a public CDN and cached locally (owner-confirmed).
- **Overlay style:** Opaque plates over the original text (owner-confirmed).
- **Copy/tone policy:** New strings match the existing plain, reassuring listing voice; typography
  rules of the repo apply (short hyphens, "..").
- **Data compatibility:** Additive settings; no migration of existing stored data.
- **Localization:** en, ru, uk (the locales the extension already ships).
- **Validation level:** Standard — automated checks for pure logic plus hands-on verification of the
  three entry points in a real browser.
- **Owner sign-off:** Required before release (privacy-policy and store-copy change).
- **Related tickets:** Builds on `2026-06-25_pdf-translate-extension` (the viewer/pipeline this
  extends). No blocking dependencies.

## 4. Current architecture context

The extension has two runtime surfaces: a background service worker that owns request interception
and the on/off model, and a set of web-accessible pages (the reflow viewer, popup, options) that do
all rendering locally in the tab. Documents flow through the viewer, which reads bytes in the browser,
extracts text, sets the document language so the browser offers translation, and streams clean HTML
into the DOM. Third-party runtime code (the PDF engine) is vendored into the package and never fetched
at runtime; settings live in a single shared object in extension storage.

The problem cannot be solved as-is because the whole pipeline is text-only: images are passed through
(EPUB) or dropped in favor of extracted text (PDF), and there is no recognition capability, no image-
entry point from arbitrary web pages (no context menu), and no place in the UI to control OCR or
manage recognition languages.

## 5. Proposed approach

Introduce an **image-OCR overlay capability** as a self-contained, reusable unit and wire it into the
existing surfaces at three points. The unit takes an image, recognizes text on-device, and produces a
positioned overlay (the original image plus opaque text plates carrying real HTML text), and it sets
the document language so the browser's translator engages. It reports progress throughout.

- **Entry A — any web image:** the background worker gains a right-click context-menu action on images
  that opens a dedicated overlay page in a new tab for the chosen image.
- **Entry B — EPUB images:** the viewer runs each chapter's images through the overlay unit and swaps
  the image for the overlaid result, so one "Translate page" covers text and images together.
- **Entry C — PDF images:** the viewer extracts raster images from pages and runs them through the same
  overlay unit alongside the reflowed text. (Highest-risk integration; sequenced last.)

The recognition engine is vendored for the default (English) language and initialized lazily. Extra
languages are fetched on explicit request and cached locally for reuse. All three entries share one
engine lifecycle, one overlay renderer, and one progress model.

### 5.1 Pillars / modules

- **OCR overlay unit (shared core).** Goal: turn an image into an overlaid, translatable container.
  Requirements: lazy engine init; recognize with a selected language; group results into block-level
  plates with bounding boxes; render opaque plates with fitted real text over the source image; emit
  progress events; degrade gracefully when nothing is recognized.
- **Language manager.** Goal: make English work offline and let the user add/reuse other languages.
  Requirements: bundled-English path with no network; on-demand download of other language data from a
  public source; local persistent cache; a clear "installed vs. available" view; disclosure of the
  network action.
- **Context-menu entry + overlay page.** Goal: OCR any image from the open web. Requirements: image-
  context menu action; a web-accessible page that loads the chosen image (including cross-origin, via
  the extension's own fetch rights) and shows the overlay with progress.
- **Viewer integration (EPUB, then PDF).** Goal: OCR images inside opened documents. Requirements:
  discover images per format; run them through the shared unit without blocking text reading; defer
  work until an image is in view or the user opts in; keep the single-"Translate page" experience.
- **Controls & progress UI.** Goal: user control and reassurance. Requirements: an "Use OCR for
  images" toggle next to the file-open control with nested language-download controls; per-image and
  overall progress indication during recognition; an options-page home for language management.

### 5.2 Data & event flows

- Web image: user right-click -> background opens overlay page for the image URL -> page fetches image
  bytes -> shared unit recognizes (progress shown) -> overlay rendered -> document language set ->
  user invokes browser translate.
- Document image: viewer builds page/chapter -> discovers images -> (deferred until visible/opt-in)
  shared unit recognizes (progress shown per image) -> image replaced by overlay in the DOM -> user
  invokes browser translate once for the whole document.
- Language add: user requests a language in the UI -> language manager fetches + caches data locally ->
  language becomes selectable for all future recognitions.
- Settings: OCR-on/off and preferred recognition language(s) live in the shared settings object read
  by the viewer, popup, options, and overlay page.

### 5.3 Extension points

- The recognition engine choice sits behind the overlay unit's boundary so it can be swapped or
  augmented later without touching the entry points.
- The image-discovery step is per-format, so new document formats can add discovery without changing
  the overlay unit.
- Overlay grouping is pluggable, leaving room to replace block-based grouping with true bubble/panel
  detection later without changing callers.

## 6. Open questions / research items

1. **PDF image extraction fidelity**
   - **Question:** What is the most reliable way to obtain per-page raster images (position + pixels)
     from the PDF engine for OCR, including scanned pages that are effectively one full-page image?
   - **Options:** Extract embedded image objects with their placement; or rasterize the page region.
   - **Status:** Resolved
   - **Artifact:** `DEV/plan/2026-07-01_ocr-image-overlay/research/01__ocr-stack-decisions.md`

2. **Language-pack source and integrity**
   - **Question:** Which public source hosts the recognition language data, and how do we handle
     failed/partial downloads and offline reuse?
   - **Options:** A well-known CDN with local caching and retry; verify presence before enabling a
     language.
   - **Status:** Resolved
   - **Artifact:** `DEV/plan/2026-07-01_ocr-image-overlay/research/01__ocr-stack-decisions.md`

3. **Deferred-OCR trigger for documents**
   - **Question:** What triggers OCR of a document image — visibility, an explicit per-image action,
     or a document-level "OCR images now"?
   - **Options:** Visibility-based lazy run; explicit per-image button; hybrid.
   - **Status:** Resolved
   - **Artifact:** `DEV/plan/2026-07-01_ocr-image-overlay/research/01__ocr-stack-decisions.md`

4. **Overlay text fitting and readability**
   - **Question:** How to size/wrap recognized text to its plate so the pre-translation and post-
     translation text both stay readable (translation length differs from source).
   - **Options:** Auto-fit font to box with wrapping; allow overflow with scroll/expand.
   - **Status:** Resolved
   - **Artifact:** `DEV/plan/2026-07-01_ocr-image-overlay/research/01__ocr-stack-decisions.md`

## 7. Risks

| Risk | Likelihood | Impact | Mitigation |
|------|:----------:|--------|-----------|
| OCR freezes/slows image-heavy documents | High | Tab unresponsive, bad UX | Defer/queue OCR, run off the main reading path, opt-in + visibility-based triggering, visible progress |
| Package size grows too large for the store | Med | Review friction, slow install | Bundle only English data; download other languages on demand |
| Privacy-promise regression via language downloads | Med | Trust/store-policy issue | Keep default fully local; disclose optional download in policy + store copy; download only on explicit action |
| Cross-origin image pixels unreadable | Med | OCR fails on some web images | Fetch bytes via the extension's own host access rather than reading a tainted canvas |
| PDF image extraction unreliable across producers | High | Missed/garbled images | Sequence PDF last; validate on embedded-image and scanned PDFs; fall back to page rasterization |
| Poor recognition accuracy (stylized/vertical text) | Med | Weak translations | Set expectations; allow language selection; leave grouping pluggable for later bubble detection |
| Overlaid text mispositioned after translation length change | Med | Overlap/clipping | Auto-fit and allow expansion; opaque plates hide the original underneath |

## 8. User impact (docs)

New capability the user perceives: "Recognize and translate text inside images — right-click any web
image, or turn on image OCR for PDFs and EPUBs — with optional downloadable language packs." The
feature docs and store listing gain a short section, and the privacy policy gains a line about the
optional, user-triggered language-pack download.

## 9. Architecture decisions (ADR)

**ADR-1: On-device OCR via a bundled engine, English in the box, other languages on demand.**
Decision: recognize locally with a vendored engine; ship only English data; fetch other language data
on explicit user request and cache it. Alternatives: cloud OCR (more accurate, smaller package) or
bundling many languages (fully offline for all). Why: preserves the "100% local, no API key" identity
for the default case and keeps the package small, while still supporting more languages for those who
opt in.

**ADR-2: One shared overlay unit behind three entry points.** Decision: a single recognition+overlay
core reused by the context-menu page and by EPUB and PDF viewer integration. Alternatives: separate
implementations per surface. Why: one progress model, one engine lifecycle, and consistent output;
avoids drift between surfaces.

**ADR-3: Block-based overlay now, bubble detection later.** Decision: group recognized text by the
engine's block/paragraph boxes for the first iteration. Alternatives: build CV speech-bubble
segmentation upfront. Why: delivers translatable overlays without a hard CV problem; grouping is kept
pluggable so bubble detection can replace it later.

## 10. Links to other specs

- `DEV/plan/2026-06-25_pdf-translate-extension.md` — the viewer/interception base this extends.

## 11. Done criteria (strategic)

1. Right-clicking an image on a web page offers an OCR action that opens a new tab where the image's
   text appears as real HTML over the image, and the browser offers "Translate page" for it.
2. Opening an EPUB with image-borne text and invoking "Translate page" translates that image text in
   place, alongside the reflowed body text.
3. Opening a PDF with image-borne text (including a scanned page) and invoking "Translate page"
   translates that image text in place.
4. English recognition works with no network access after install.
5. The user can download at least one additional language from the UI, and after downloading it is
   usable for recognition and remains available in a later session without re-downloading.
6. During recognition the user sees progress that advances and completes for both single images and
   multi-image documents.
7. The file-open UI shows an "Use OCR for images" toggle with reachable language-download controls,
   and the toggle governs whether document images are OCR'd.
8. With OCR off (or before opt-in), documents open and read as fast as they do today.

## 12. Next step

`/spec-tech 2026-07-01_ocr-image-overlay` — creates the phased tactical plan.
