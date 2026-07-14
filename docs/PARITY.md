# Cross-edition parity

The single source of truth for what must stay identical across the project's editions, what is
intentionally different, and how to keep them from drifting. **If you add or change a user-facing
feature, read this file first and update it.**

Why this file exists: the app ships as two independent codebases that do **not** share code - a Go
desktop app (CLI / GUI / MSIX Store) and a JavaScript browser extension. Logic is ported from Go to JS
by hand, so shared constants and heuristics drift silently unless they are pinned here. See the
[Editions](../README.md#editions) section for the user-facing framing.

> Convention: this file is the *reference* (the invariants and the map). The actionable backlog of
> open gaps lives in the parity ticket [`DEV/plan/2026-07-01_cross-edition-parity.md`](../DEV/plan/2026-07-01_cross-edition-parity.md).
> When a gap is closed, update both.

## Editions and codebases

| Edition | Codebase | Language | Entry point |
|---|---|---|---|
| CLI (`doc-html-translate.exe`) | shared Go | Go | [`cmd/doc-html-translate/main.go`](../cmd/doc-html-translate/main.go) |
| GUI (`doc-html-ui.exe`) | shared Go (shells out to CLI) | Go + HTML | [`cmd/doc-html-ui/main.go`](../cmd/doc-html-ui/main.go), [`ui.html`](../cmd/doc-html-ui/ui.html) |
| Microsoft Store app (MSIX) | same Go GUI + CLI, packaged | Go | [`msix/README.md`](../msix/README.md) |
| Browser extension | independent JS | JavaScript (MV3) | [`extension/src/`](../extension/src/) |
| Website / docs | static | HTML | GitHub Pages |

The CLI, GUI and MSIX app are one codebase (the GUI drives the CLI), so parity work is really
**Go (all three) vs the JS extension**, plus **CLI vs GUI** for the settings surface.

## The port map (Go <-> JS)

Each JS module re-implements the named Go code. A change to one side is a change to the other.

| Capability | Go | JS (extension) |
|---|---|---|
| PDF paragraph/heading reflow | [`internal/pdf/extract.go`](../internal/pdf/extract.go) (`rowsToText`, `classifyBlock`, `isLigaturesArtifact`) | [`extension/src/reflow.js`](../extension/src/reflow.js) |
| PDF outline -> TOC | [`internal/pdf/toc.go`](../internal/pdf/toc.go) | [`extension/src/toc.js`](../extension/src/toc.js) |
| EPUB unzip + OPF/spine + sanitize + TOC | [`internal/epub/`](../internal/epub/) (`epub.go`, `toc.go`) | [`extension/src/epub.js`](../extension/src/epub.js) |
| Plain text -> paragraphs/pages | [`internal/txt/`](../internal/txt/) | [`extension/src/txt.js`](../extension/src/txt.js) |
| RTF strip + cp1251 decode | [`internal/rtf/`](../internal/rtf/) | [`extension/src/rtf.js`](../extension/src/rtf.js) |
| Markdown -> HTML | [`internal/md/`](../internal/md/) (`goldmark`) | [`extension/src/md.js`](../extension/src/md.js) (vendored `marked`) |
| FB2 XML -> sections/TOC | [`internal/fb2/`](../internal/fb2/) | [`extension/src/fb2.js`](../extension/src/fb2.js) |
| HTML `<body>` extract | [`internal/htmlconv/`](../internal/htmlconv/) | [`extension/src/html.js`](../extension/src/html.js) |
| MOBI / AZW3 (KF8) | [`internal/mobi/`](../internal/mobi/) (shells out to Calibre) | [`extension/src/ebook.js`](../extension/src/ebook.js) (vendored `foliate-js`) |
| HTML sanitize -> fragment | (EPUB-only in Go: `epub.go` normalize) | [`extension/src/sanitize.js`](../extension/src/sanitize.js) |
| OCR overlay (recognize -> plates) | [`internal/ocr/overlay.go`](../internal/ocr/overlay.go), `tesseract.go` | [`extension/src/ocr-overlay.js`](../extension/src/ocr-overlay.js) + `.css` |
| OCR language manager | [`internal/ocr/tessdata.go`](../internal/ocr/tessdata.go) | [`extension/src/ocr-lang.js`](../extension/src/ocr-lang.js) |
| Reader chrome (themes, fonts, controls) | [`internal/htmlgen/navbar.go`](../internal/htmlgen/navbar.go) (`readerCSS`, `readerScript`) | [`extension/src/viewer.css`](../extension/src/viewer.css), [`viewer.js`](../extension/src/viewer.js), [`viewer.html`](../extension/src/viewer.html) |
| Source-language detection | (none - Go copies the source `<html lang>`) | [`extension/src/lang.js`](../extension/src/lang.js) |
| Settings / options surface | [`internal/config/flags.go`](../internal/config/flags.go), [`ui.html`](../cmd/doc-html-ui/ui.html) | [`popup.js`](../extension/src/popup.js), [`options.js`](../extension/src/options.js), [`background.js`](../extension/src/background.js) |

## Shared invariants (MUST stay identical on both sides)

These are duplicated across codebases with no shared source. Changing a value on one side without the
other is a bug. Each row cites the two places that must agree.

### Reader theme palette

Exactly four themes, in this order: **`light`, `sepia`, `dark`, `night`**. Eight CSS variables per
theme. The **values must be identical**; only the variable *names* and the `<html>` attribute differ
(see [Intentional divergences](#intentional-divergences-do-not-fix)).

| Theme | bg | fg | muted | bar-bg | bar-fg | border | accent | link |
|---|---|---|---|---|---|---|---|---|
| light (`:root`) | `#faf9f7` | `#1b1b1b` | `#6b6b6b` | `#ffffff` | `#222222` | `#e2e0db` | `#2563eb` | `#1a4fb4` |
| sepia | `#f4ecd8` | `#4a3f2f` | `#7a6c54` | `#efe6cf` | `#4a3f2f` | `#ddd0b0` | `#8a5a2b` | `#7a4a1b` |
| dark | `#1a1a1c` | `#e6e4df` | `#9a9893` | `#232327` | `#e6e4df` | `#36363b` | `#5b8dff` | `#8fb4ff` |
| night | `#0a0a0b` | `#9a9a9a` | `#6a6a6a` | `#131315` | `#b8b8b8` | `#262629` | `#5599d6` | `#6aa8e0` |

Sources: [`navbar.go:353-373`](../internal/htmlgen/navbar.go#L353-L373) (`--dht-*`),
[`viewer.css:5-51`](../extension/src/viewer.css#L5-L51) (`--*`). Guarded by `tests/parity_test.go`
(`TestParityThemePalette`), which compares the per-theme colour sequences (3- and 6-digit hex normalized).

### Reader fonts

Serif / sans / mono families, identical strings both sides:
`serif` = `Georgia,"Times New Roman",serif` · `sans` = `"Segoe UI",system-ui,Arial,sans-serif` ·
`mono` = `"Cascadia Code",Consolas,monospace`. Sources:
[`navbar.go:404-408`](../internal/htmlgen/navbar.go#L404-L408),
[`viewer.js:91-95`](../extension/src/viewer.js#L91-L95).

### PDF reflow heuristics

| Constant | Value | Go | JS |
|---|---|---|---|
| Paragraph Y-gap factor | `1.5` x median line spacing | `extract.go` (`medianGap*1.5`) | `PARA_GAP_FACTOR` [`reflow.js:19`](../extension/src/reflow.js#L19) |
| First-line indent threshold | `8` pt | `indentThreshold` | `INDENT_THRESHOLD` [`reflow.js:17`](../extension/src/reflow.js#L17) |
| Left-margin baseline | 25th percentile of first-word X | `extract.go` | `reflow.js` |
| Median line-spacing fallback | `12` | `extract.go` | `reflow.js` |
| Ligature-artifact filter | avg word length `< 3.0` over `>= 4` words | `isLigaturesArtifact` | [`reflow.js:48-53`](../extension/src/reflow.js#L48-L53) |
| Heading word caps | "short" `<= 8`, "medium" `<= 14` | `classifyBlock` | [`reflow.js:36,41`](../extension/src/reflow.js#L36-L41) |

Both sides now name these constants (Go: a documented `const` block in `extract.go`; JS: the
`*_FACTOR`/`*_THRESHOLD` consts in `reflow.js`) and `tests/parity_test.go` asserts the values match. See
the JS-only additions under [Intentional divergences](#intentional-divergences-do-not-fix).

### EPUB TOC parsing

| Rule | Both sides |
|---|---|
| TOC source priority | EPUB3 `nav.xhtml` (`properties="nav"`) preferred; EPUB2 `toc.ncx` used **only** if nav yields 0 entries |
| NCX item lookup | `<spine toc="..">` idref first, then media-type `application/x-dtbncx+xml` |
| `<nav>` selection order | `epub:type="toc"` -> `role="doc-toc"` -> first `<nav>` |
| Entry survival | drop an entry whose target is unresolvable **unless** it has surviving children (then it becomes a label-only node) |
| Label | first `<a>` (or `<span>`) not inside a nested `ol`/`ul` |

Sources: [`internal/epub/toc.go`](../internal/epub/toc.go), [`extension/src/epub.js`](../extension/src/epub.js).
Title whitespace normalization (Go NCX now uses `collapseWS`, matching the nav path and the extension)
and the external-href definition (extension `isExternalHref` now mirrors `toc.go`: any `://` scheme, or
`mailto:`/`tel:`/`data:`) were aligned in the 2026-07-01 parity pass. The remaining intentional
difference - Go keeps external TOC entries, the extension drops them (single in-memory DOM) - is listed
under [Intentional divergences](#intentional-divergences-do-not-fix).

### OCR

| Contract | Value / rule | Go | JS |
|---|---|---|---|
| Bundled language | `eng` only, provisioned at build time (not committed) | `scripts/build.ps1` -> `<exe>/tessdata/eng.traineddata` | `npm run vendor` -> `vendor/tesseract/lang/` |
| traineddata filename | `<code>.traineddata`, `code` = Tesseract name | [`tessdata.go`](../internal/ocr/tessdata.go) | [`ocr-lang.js`](../extension/src/ocr-lang.js) |
| Plate granularity | one plate per **proximity cluster of confident text lines** (not per paragraph - the engine folds imagery into text paragraphs and splits uniform prose arbitrarily). Flatten the recognition to lines, drop noise (below), then grow a plate while the vertical gap to the next line stays within `OCR_CLUSTER_GAP_FACTOR (1.2) x` the median line height and the lines share an x-extent; a bigger gap - a figure, a section break, a new column - starts a new plate | [`tesseract.go`](../internal/ocr/tesseract.go) `clusterLines` | [`ocr-overlay.js`](../extension/src/ocr-overlay.js) `clusterLines` |
| Plate geometry | percent of natural image size; plate bbox = **union of the cluster's line boxes**; font-size in `cqw` from the cluster's median line height x `0.92` fit factor (so text fits its block, not overflow into the next plate); block-level container `display:block; width:100%; aspect-ratio:W/H; container-type:inline-size; line-height:1.1` with the image at `width:100%; margin:0; max-height:none` (a page-level `img` reset must not offset or shrink the overlay image, or the percent-positioned plates drift vertically - up above the image centre, down below it); plates top-align their text (`align-items:flex-start`) and grow down via `min-height` | [`overlay.go`](../internal/ocr/overlay.go), [`tesseract.go`](../internal/ocr/tesseract.go) | [`ocr-overlay.js`](../extension/src/ocr-overlay.js) |
| Plate colours | adaptive, sampled from the source image (best-effort; falls back to white `#fff` / dark `#111`): background = median colour over the whole block ("paper"); text = mean of pixels standing out from bg (L1 dist > `90`) within the first line (`1.3 x` line height), else near-black/near-white; contrast floor `55` luma; `0.015`/`6`-px min-ink threshold | [`overlay.go`](../internal/ocr/overlay.go) `blockColors` | [`ocr-overlay.js`](../extension/src/ocr-overlay.js) `blockColors` |
| Noise filter | two gates. **Line confidence:** before clustering, drop a recognized line whose mean word confidence is `< OCR_MIN_LINE_CONF (50)` - real text scores ~80-97, "text" hallucinated from a drawing scores ~0-50, so this keeps plates off imagery and keeps oversized noise boxes from inflating the font. **Text (`isTranslatable`)** on the assembled plate text: drop when `< 5` letters (also kills numbers/symbols); letters but no vowels; the whole text is an address (URL/email/domain/path); or "mishmash" - among letter-bearing tokens, `< 0.5` are word-like (`>= 2` letters + a vowel), needs `>= 3` such tokens. Short CJK (`>= 2` ideographs) is kept | [`tesseract.go`](../internal/ocr/tesseract.go), [`text.go`](../internal/ocr/text.go) `isTranslatable` | [`ocr-overlay.js`](../extension/src/ocr-overlay.js), [`ocr-text.js`](../extension/src/ocr-text.js) `isTranslatable` |
| Pre-OCR upscale | an image whose long side is `< 1000 px` is enlarged `2 x` (high-quality) before recognition so Tesseract reads low-res scans/thumbnails better; recognized coordinates (and dimensions) are divided back by the factor so plates map onto the original picture. Larger images are recognized as-is | [`tesseract.go`](../internal/ocr/tesseract.go) `upscaleForOCR` / `scaleDown` | [`ocr-overlay.js`](../extension/src/ocr-overlay.js) `upscaleForOcr` |
| Page-segmentation mode | Tesseract runs in **PSM 3 (AUTO)** on both editions so layout analysis isolates real text regions on an illustrated/scanned page (a speech bubble, a caption) instead of reading the whole frame as one block. The desktop CLI's default is already PSM 3 (made explicit via `--psm`); the extension must set it because tesseract.js defaults to PSM 6 (SINGLE_BLOCK), which folds scene edges into the recognized text (stray punctuation, digits) and mis-merges separate regions into one plate | [`tesseract.go`](../internal/ocr/tesseract.go) `ocrPageSegMode` | [`ocr-overlay.js`](../extension/src/ocr-overlay.js) `OCR_PSM` |

**OCR download version and catalog are aligned** (2026-07-01 parity pass) and guarded by
`tests/parity_test.go`:

- **tessdata version = 4.0.0** on both sides. Go pins GitHub `tessdata_fast/raw/4.0.0` (plain,
  [`tessdata.go`](../internal/ocr/tessdata.go)); the extension loads
  `tessdata.projectnaptha.com/4.0.0_fast` (gzip, [`ocr-lang.js`](../extension/src/ocr-lang.js)) and
  bundles eng from the same 4.0.0 ([`build.mjs`](../extension/build.mjs)). Different host/format,
  identical upstream bytes -> matching recognition.
- **Language catalog = 13** on both sides (`eng rus ukr jpn jpn_vert deu fra spa ita por pol chi_sim
  kor`): `tessdata.go` `Available` == `ocr-lang.js` `LANGS`.
- **Overlay grouping constants** identical: `OCR_MIN_LINE_CONF = 50` and `OCR_CLUSTER_GAP_FACTOR = 1.2`
  (`tesseract.go` `ocrMinLineConf` / `ocrClusterGapFactor` == `ocr-overlay.js`).
- **Overlay upscaling constants** identical: `OCR_UPSCALE_BELOW = 1000` and `OCR_UPSCALE_FACTOR = 2`
  (`tesseract.go` `ocrUpscaleBelow` / `ocrUpscaleFactor` == `ocr-overlay.js`).
- **Page-segmentation mode** identical: PSM `3` (AUTO). `tesseract.go` `ocrPageSegMode` (passed to the
  CLI as `--psm 3`) == `ocr-overlay.js` `OCR_PSM` (applied via `setParameters({tessedit_pageseg_mode})`).
  The extension must set it explicitly because tesseract.js defaults to PSM 6 (SINGLE_BLOCK); the
  desktop CLI's own default is already 3.
- **Plate font-fit factor** identical: `0.92` - `overlay.go` `fontFitFactor` == `ocr-overlay.js`
  `FONT_FIT` (guarded by `TestParityOCRFontFit`). Plate font-size = median line height x this factor;
  below `1.0` so translated text (often longer) has room before it overflows the block.

**Still divergent (tracked in the parity ticket):**

- **CSS class names** differ: Go `.ocr-fig` / `.ocr-box`; JS `.ocr-overlay` / `.ocr-plate` /
  `.ocr-overlay-img` / `.ocr-badge`. Cosmetic; deferred.
- **Default OCR language rule**: Go derives from `-src` (`TessLang`, else `eng`); the extension uses a
  fixed persisted `eng` (it has no translation source language). Intentional for now.

### Settings defaults

Canonical defaults (from [`flags.go`](../internal/config/flags.go)): `-split 5000`, `-toc-depth 0`
(unlimited), `-src en`, `-dst ru`, `-max-cost 0` (no limit), `-ocr false`, `-ocr-lang ""` (falls back
to `-src`, else `eng`), `-ollama-model gemma3:12b`, `-ollama-parallel 1`, `-ollama-ctx 8192`.

Invariant: **the GUI must expose every CLI flag** (see [`ui-cli-parity`](../CLAUDE.md) memory). Known
default mismatches (GUI split=0, extension source-lang=auto, extension OCR-lang fixed `eng`) are tracked
in the parity ticket.

### Product URL and feedback address

Every edition surfaces the same product page and the same feedback address. Both are duplicated string
constants - changing either is a cross-edition change; update all rows together.

| Element | Value | Sources |
|---|---|---|
| Product site | `https://serzhyale.github.io/doc-html-translate/` | CLI splash [`app.go`](../internal/app/app.go) (`printSplashEN`/`printSplashRU`), navbar [`projectURL`](../internal/htmlgen/navbar.go#L17), GUI [`ui.html`](../cmd/doc-html-ui/ui.html) byline, extension [`popup.html`](../extension/src/popup.html) / [`viewer.html`](../extension/src/viewer.html) / [`options.html`](../extension/src/options.html) |
| Feedback | `mailto:sza@ukr.net` | CLI splash [`app.go`](../internal/app/app.go), GUI [`ui.html`](../cmd/doc-html-ui/ui.html) byline, extension [`popup.html`](../extension/src/popup.html) / [`viewer.html`](../extension/src/viewer.html) / [`options.html`](../extension/src/options.html) |

## Intentional divergences (do NOT "fix")

These are by design. Do not "sync" them without a decision - document changes here instead.

- **EPUB output model.** Go extracts a **multi-file book to disk** and does **not** sanitize chapter
  HTML (it opens local files in the user's own browser). The extension merges the whole spine into **one
  in-memory DOM** and therefore sanitizes (drops `script/style/inline styles/on*`), rewrites `<img>` to
  `blob:` URLs, and namespaces ids/anchors. So sanitize, image-blobbing and anchor remapping exist
  **only in the extension** by design.
- **New-format parsing stacks differ by design.** The extension reimplements the pure-Go extractors
  in JS feeding a shared renderer: TXT/RTF/FB2/HTML are hand ports, but Markdown uses the vendored
  **`marked`** (Go uses `goldmark`) and MOBI/AZW3 use the vendored **`foliate-js`** (Go shells out to
  **Calibre**, which the browser can't). New formats build their fragment via [`sanitize.js`](../extension/src/sanitize.js);
  EPUB keeps its own `renderChapter`. Behavioural parity ("opens and reads correctly") is the bar, not
  byte-identical HTML.
- **Reader features that are Go-only:** reading-position persistence + "Continue reading", page zoom
  (Ctrl+wheel, `?z=`), Russian localization (`syslocale.IsRussian`), and the separate `index.html` TOC
  page / multi-file navigation.
- **Reader features that are JS-only:** heuristic source-language detection ([`lang.js`](../extension/src/lang.js)),
  the collapsible sidebar TOC, the single continuous-scroll document, and the two toolbar downloads -
  "&#8595; File" (the untouched source bytes) and "&#8595; HTML" (the current on-screen view saved as a
  self-contained `.html`). The HTML export serializes the *live* `#content`, so it captures whatever the
  browser's built-in translator has swapped in - the extension's way of "keeping the translation" without
  a translation API. The Go app has no equivalent: it never translates in place and already writes HTML to
  disk, so both downloads are extension-only by design ([`viewer.js`](../extension/src/viewer.js)).
- **Progress bar** looks identical (3px accent bar) but means **reading progress** in Go vs
  **load/OCR progress** in the extension.
- **OCR execution:** Go OCRs eagerly at conversion time; the extension OCRs lazily on scroll
  (IntersectionObserver). The extension has two entry points for OCRing a **standalone image**: the
  right-click "OCR & translate this image", and opening a bare image file (PNG/JPEG/GIF/BMP/WebP) with the
  viewer's **Open file** picker (which OCRs it unconditionally, ignoring the "Use OCR for images" toggle).
  The Go app now **also accepts a standalone image as input** ([`internal/img`](../internal/img/extract.go)):
  a bare PNG/JPG/JPEG/WebP/GIF/BMP/TIFF is wrapped in a one-page HTML doc and the OCR overlay is forced on
  (independent of the `-ocr` flag), so `doc-html-translate <image>` shows the picture with translatable
  plates - the same result as the extension's picker. Both editions share the overlay logic; the remaining
  difference is the engine (extension = tesseract.js WASM + `4.0.0_fast`; Go = the local `tesseract` CLI).
  There is deliberately **no** file-type association (DNR redirect) for image URLs on either side - unlike
  PDF/EPUB, image links are never intercepted; only the explicit picker / right-click / CLI paths OCR images.
- **PDF paths not ported:** Go's pdftotext `-layout` path, its double-spaced/ZWSP paragraph merge, and
  its blank-page skip + `hrefForPDFPage` TOC remap have **no JS counterpart**. Conversely the JS reflow
  adds a **font-size dimension** (`FONT_BREAK_RATIO=0.25`, `big`/`veryBig` heading triggers) and
  geometric centering that Go does not have. (ALL-CAPS heading detection is now **aligned**: both sides
  are language-agnostic, so Cyrillic headings like "ГЛАВА ПЕРВАЯ" are detected on both.) JS image
  extraction also normalizes **mirrored image placements** - a negative CTM scale at paint time means
  the raster is stored flipped, so [`pdf-images.js`](../extension/src/pdf-images.js) mirrors the pixels
  back to their rendered orientation; Go's pdfcpu extraction writes raw streams and has no counterpart yet.
- **Translation target:** the extension has **no target language** - it delegates to the browser's
  built-in "Translate page", so `-dst` is CLI/GUI-only.
- **Storage:** Go uses `localStorage`/`sessionStorage` string keys (`dht_*`); the extension uses
  `chrome.storage.local` objects. Reading preferences are not portable between the two.

## Process: keeping editions in sync

1. **One cross-edition ticket per feature.** A user-facing feature gets a single ticket in `DEV/plan/`
   that covers all affected editions, using the template
   [`DEV/plan/_TEMPLATE_cross-edition.md`](../DEV/plan/_TEMPLATE_cross-edition.md). Do **not** open a
   separate ticket per edition (the OCR feature was done as two tickets - that is the anti-pattern this
   replaces).
2. **Parity checklist.** Every such ticket answers, for CLI, GUI, MSIX and Extension: implemented, or
   **intentionally declined with a one-line rationale**. "Not done" is only acceptable as an explicit
   "declined" with a reason recorded here under [Intentional divergences](#intentional-divergences-do-not-fix).
3. **Update this file** whenever you touch a shared invariant (a palette value, a heuristic constant, the
   OCR host/catalog, a default). The invariant tables above are the source of truth; the code must match
   them, not the other way around.
4. **Prefer a single source of truth in code** when practical (e.g. a generated palette both front-ends
   read) over manual copy + a "values match" comment.
5. **Guard tests** parse both codebases and fail on drift: `tests/parity_test.go` (theme palette, OCR
   tessdata version, OCR language catalog, PDF reflow constants) and `tests/ui_cli_parity_test.go` (the
   GUI exposes every CLI flag). They run in the normal `scripts/test.ps1` gate - extend them whenever you
   pin a new invariant here.
