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
| Plain text: source-encoding decode | [`internal/txt/extract.go`](../internal/txt/extract.go) (`decodeText`) | [`extension/src/txt.js`](../extension/src/txt.js) (`decodeText`) |
| RTF strip + cp1251 decode | [`internal/rtf/`](../internal/rtf/) | [`extension/src/rtf.js`](../extension/src/rtf.js) |
| Markdown -> HTML | [`internal/md/`](../internal/md/) (`goldmark`) | [`extension/src/md.js`](../extension/src/md.js) (vendored `marked`) |
| FB2 XML -> sections/TOC | [`internal/fb2/`](../internal/fb2/) | [`extension/src/fb2.js`](../extension/src/fb2.js) |
| HTML `<body>` extract | [`internal/htmlconv/`](../internal/htmlconv/) | [`extension/src/html.js`](../extension/src/html.js) |
| MOBI / AZW3 (KF8) | [`internal/mobi/`](../internal/mobi/) (shells out to Calibre) | [`extension/src/ebook.js`](../extension/src/ebook.js) (vendored `foliate-js`) |
| Comic archive -> page book | [`internal/comic/`](../internal/comic/) (CBZ/CBT stdlib; CBR/CB7 shell out to 7-Zip) | [`extension/src/comic.js`](../extension/src/comic.js) (CBZ/CBT only; CBR/CB7 declined) |
| Comic natural page order + entry filter | [`internal/comic/natural.go`](../internal/comic/natural.go), `extract.go` (`isPageEntry`) | [`extension/src/comic.js`](../extension/src/comic.js) (`naturalCompare`, `isPageEntry`) |
| Comic forced-OCR decision | [`internal/pipeline/pipeline.go`](../internal/pipeline/pipeline.go) (`comic.IsComic` -> `forceOCR`) | [`extension/src/viewer.js`](../extension/src/viewer.js) (`loadComicData` -> `registerImagesForOcr(.., true)`) |
| HTML sanitize -> fragment | (EPUB-only in Go: `epub.go` normalize) | [`extension/src/sanitize.js`](../extension/src/sanitize.js) |
| OCR overlay (recognize -> plates) | [`internal/ocr/overlay.go`](../internal/ocr/overlay.go), `tesseract.go` | [`extension/src/ocr-overlay.js`](../extension/src/ocr-overlay.js) + `.css` |
| OCR language manager | [`internal/ocr/tessdata.go`](../internal/ocr/tessdata.go) | [`extension/src/ocr-lang.js`](../extension/src/ocr-lang.js) |
| Reader chrome (themes, fonts, controls) | [`internal/htmlgen/navbar.go`](../internal/htmlgen/navbar.go) (`readerCSS`, `readerScript`) | [`extension/src/viewer.css`](../extension/src/viewer.css), [`viewer.js`](../extension/src/viewer.js), [`viewer.html`](../extension/src/viewer.html) |
| Source-language detection | (none - Go copies the source `<html lang>`) | [`extension/src/lang.js`](../extension/src/lang.js) |
| Settings / options surface | [`internal/config/flags.go`](../internal/config/flags.go), [`ui.html`](../cmd/doc-html-ui/ui.html) | [`popup.js`](../extension/src/popup.js), [`options.js`](../extension/src/options.js), [`background.js`](../extension/src/background.js) |

## Shared invariants (MUST stay identical on both sides)

These are duplicated across codebases with no shared source. Changing a value on one side without the
other is a bug. Each row cites the two places that must agree.

### Input format detection is by byte signature, not extension

Both editions decide what a file *is* from its leading bytes, not its name, so a mislabelled or
extensionless file still routes correctly and a binary is never fed to a text reader.

- **Extension:** `extension/src/viewer.js` `detectFormat` tests `%PDF`, `PK..` (ZIP), `{\rtf`, MOBI's
  `BOOKMOBI`, and image magics, before falling back to the filename extension. Both EPUB and CBZ are ZIP,
  so the one case the signature cannot settle - `PK..` - is broken by the filename extension: a `.cbz`
  routes to the comic reader, any other ZIP to the EPUB reader (the EPUB hot path is unchanged). Anything
  unrecognized routes to the PDF reader, which reports an unreadable file clearly. So a `.docx` or DjVu
  fails with a real error rather than rendering as garbage.
- **Go:** the CLI still dispatches known extensions by name (its readers are extension-keyed), but the
  `default:` "unknown extension" arm now sniffs the bytes via `internal/txt` `LooksBinary` before handing
  them to the text extractor. A recognized binary signature - ZIP, RAR, 7z, tar (`ustar` at offset 257),
  DjVu, and defensively PDF/MOBI/image - is refused and named; anything else with a NUL byte in the first
  4 KB is refused as "binary data". A BOM is checked first, so UTF-16 text (which is full of NUL bytes) is
  not mistaken for binary.

The two are not byte-identical by design - the extension re-renders in a live tab and leans on the PDF
reader's error path, while the Go CLI is a batch converter that must refuse with a non-zero exit and no
output directory. What must stay true on both: **detection is signature-first, and a binary never
becomes a document.** Go: `internal/txt/sniff.go`. JS: `extension/src/viewer.js` `detectFormat` /
`imageMime` / `isMobiBytes`.

### Plain-text source-encoding decode order

Both editions decide a `.txt` file's encoding from its leading bytes, in this order. The **order is the
invariant**: the same file must not read correctly on one edition and as mojibake on the other.

| # | Test | Result |
|---|---|---|
| 1 | `EF BB BF` | UTF-8; the mark is removed, never shown |
| 2 | `FF FE` | UTF-16LE |
| 3 | `FE FF` | UTF-16BE |
| 4 | the bytes are valid UTF-8 | UTF-8 as-is |
| 5 | otherwise | a legacy Cyrillic code page by detection (below), else UTF-8 |

Go: `internal/txt/extract.go` `decodeText` (`golang.org/x/text/encoding/unicode`, BOM tested explicitly).
JS: `extension/src/txt.js` `decodeText` (`TextDecoder` with an explicit `utf-16le`/`utf-16be` label).

**One difference that is not drift:** step 1 is implicit on the JS side. `TextDecoder` strips a leading
BOM by itself unless `ignoreBOM` is set, so the extension never had the UTF-8-BOM leak the Go side did.
Both arrive at the same text; only Go has to say so out loud.

Scope: this covers `.txt` only. `md.js`, `html.js` and `fb2.js` still decode as UTF-8 unconditionally,
matching their Go counterparts - a shared gap, not a divergence.

#### Step 5: legacy Cyrillic code-page detection

The candidate set, the letter-frequency table, and the confidence floor **must be identical** on both
sides, or the same DOS-era `.txt` decodes to one code page here and another there.

- **Candidates, most-likely-first:** Windows-1251, KOI8-R, CP866 (`ibm866` as a TextDecoder label),
  ISO-8859-5. This is the RU/UA audience's set, not a general code-page sweep.
- **Selection by frequency-weighted fit.** Each candidate's decoding is scored by the summed expected
  frequency of the Russian letters it contains; the highest wins. This is load-bearing: cp1251 and
  KOI8-R remap the *same* byte range, so both yield ~the same *number* of Cyrillic letters (fraction
  0.761 vs 0.760 on the corpus fixture) - only the frequency weighting separates them (16718 vs 10612).
- **Confidence by Cyrillic fraction.** Russian letters over all characters must reach **0.30**, or the
  bytes pass through as UTF-8 unchanged. Measured: the real cp1251 fixture is 0.76; French Latin-1
  mis-read as KOI8-R (which tops the *weight* score) is 0.17, so the floor rejects it. Selection needs
  weight, confidence needs fraction - neither metric alone does both.
- **Known limit, accepted:** a very short, accent-dense non-Russian string can exceed 0.30 and be
  mis-decoded. Short files carry too little signal; the floor is the agreed trade.

Go: `internal/txt/legacy.go` (`legacyCandidates`, `ruLetterFreq`, `cyrillicFit`, `minCyrillicFraction`,
`detectLegacy`). JS: `extension/src/txt.js` (`LEGACY_CANDIDATES`, `RU_LETTER_FREQ`, `cyrillicFit`,
`MIN_CYRILLIC_FRACTION`, `detectLegacy`). RTF's separate cp1251 decode (`\'XX` escapes) is unrelated and
stays as-is.

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

### PDF page-image selection

When a PDF page yields more than one raster, both editions collapse **proportional-scale duplicates** -
the same picture embedded at two resolutions - down to the largest, so a scanned page is not shown twice.
The signal is the aspect ratio: a uniform scale preserves it, so two rasters whose ratios match within
**`aspectRatioTolerance = 0.01` (1%)** are the same image and only the larger is kept. Differently-shaped
images (a composed page: an illustration beside a figure) are all kept - guessing "the page" among genuinely
distinct images would be wrong as often as right.

Go: `internal/pdf/extract.go` `selectPageImages` / `sameShapeRaster` / `aspectRatioTolerance`.
JS: `extension/src/pdf-images.js` `dedupeSameShape` / `sameShapeRaster` / `ASPECT_RATIO_TOLERANCE`.

The **thumbnail** half of the problem is handled asymmetrically by construction, not by drift (see
[Intentional divergences](#intentional-divergences-do-not-fix)): the Go extractor drops pdfcpu's `/Thumb`
image explicitly (`img.Thumb`), while the extension never sees a thumbnail at all - it mines the page's
paint operators, and a `/Thumb` is a page-dict entry the content stream never paints.

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

### Comic archive page order and entry filter

A comic archive (CBZ/CBR/CB7/CBT) is a container of page images with no text layer; the reader OCRs each
page into translatable plates (forced on, like a standalone image - opening a comic *is* the request to
read its bubbles). Two rules must match exactly on both editions, or the same archive reads with different
pages, or the same pages in a different order:

- **Page order is natural (numeric-aware) filename order.** Page order *is* archive entry order by
  filename, so this is correctness, not cosmetics: a plain lexicographic sort puts `page10.jpg` before
  `page2.jpg`. Runs of ASCII digits compare by value; equal value (`"2"` vs `"02"`) breaks toward the
  shorter raw run so the order is total and stable. Go: [`internal/comic/natural.go`](../internal/comic/natural.go)
  `naturalLess`. JS: [`extension/src/comic.js`](../extension/src/comic.js) `naturalCompare`.
- **Page-entry filter.** A page is a regular file whose extension is one of **`png jpg jpeg gif webp bmp`**
  (TIFF deliberately excluded - browsers cannot display it, and it is vanishingly rare in comics).
  Ignored: directory entries, `ComicInfo.xml`, `Thumbs.db`, hidden dotfiles (`.DS_Store`, `._*`), and
  anything under `__MACOSX/`. Go: `internal/comic/extract.go` `pageExts` / `isPageEntry`. JS:
  `extension/src/comic.js` `PAGE_EXTS` / `isPageEntry`. Guarded by `tests/parity_test.go`
  (`TestParityComicPageFilter`).

Container support differs by capability, not drift (see [Intentional divergences](#intentional-divergences-do-not-fix)):
the desktop app opens all four (CBR/CB7 by shelling out to 7-Zip, the MOBI/Calibre precedent), while the
extension opens **CBZ (ZIP) and CBT (TAR) only** - a browser has no RAR/7z decoder and cannot shell out,
so it recognizes a CBR/CB7 by signature and shows a "use the desktop app" notice.

### OCR

| Contract | Value / rule | Go | JS |
|---|---|---|---|
| Bundled language | `eng` only, provisioned at build time (not committed) | `scripts/build.ps1` -> `<exe>/tessdata/eng.traineddata` | `npm run vendor` -> `vendor/tesseract/lang/` |
| traineddata filename | `<code>.traineddata`, `code` = Tesseract name | [`tessdata.go`](../internal/ocr/tessdata.go) | [`ocr-lang.js`](../extension/src/ocr-lang.js) |
| Plate granularity | one plate per **proximity cluster of confident text lines** (not per paragraph - the engine folds imagery into text paragraphs and splits uniform prose arbitrarily). Flatten the recognition to lines, drop noise (below), then grow a plate while the next line keeps the **line pitch** - top of one line to top of the next - within `OCR_CLUSTER_PITCH_FACTOR (1.2) x` the page's reference pitch and the lines share an x-extent; a bigger step - a figure, a section break, a new column - starts a new plate. The reference is the **median pitch over the image**, taken over successive kept lines that share a column and sit no further apart than `OCR_MAX_LEADING_RATIO (3) x` the median ink height (beyond that it is a section break, not leading); a page that yields no pitch at all falls back to the ink-box gap. The factor multiplies the pitch and **never the height of the recognized ink box** - all-caps lettering boxes far shorter than its own line, and measuring against the ink split one balloon into three plates | [`tesseract.go`](../internal/ocr/tesseract.go) `clusterLines` / `medianLinePitch` | [`ocr-cluster.js`](../extension/src/ocr-cluster.js) `clusterLines` / `medianLinePitch` |
| Plate geometry | percent of natural image size; plate bbox = **union of the cluster's line boxes**; font-size in `cqw` from the cluster's median line height x `0.92` fit factor (the starting size); block-level container `display:block; width:100%; aspect-ratio:W/H; container-type:inline-size; line-height:1.1` with the image at `width:100%; margin:0; max-height:none` (a page-level `img` reset must not offset or shrink the overlay image, or the percent-positioned plates drift vertically - up above the image centre, down below it); plates **centre their text** (`align-items:center`) inside their source region (`min-height`) with `overflow:hidden` | [`overlay.go`](../internal/ocr/overlay.go), [`tesseract.go`](../internal/ocr/tesseract.go) | [`ocr-overlay.js`](../extension/src/ocr-overlay.js) |
| Plate runtime re-fit | The compile-time font size is computed from the **source** geometry and cannot know the reflowed - or later translator-swapped - text length, so a fixed size clips a third of plates. After layout each plate's font is shrunk (down to `0.5 x` the starting `cqw`) until the text fits its source-region box; if it still overflows at that floor the box is allowed to grow (`height:auto`) so **nothing is ever clipped**. Re-runs on window resize and whenever a `MutationObserver` sees the page translator swap a plate's text. Degrades safely (CSS `overflow:hidden`) if the script does not run | [`overlay.go`](../internal/ocr/overlay.go) `ocrScript` / `ensureScript` | [`ocr-overlay.js`](../extension/src/ocr-overlay.js) `fitPlate` / `scheduleFit` |
| Plate colours | adaptive, sampled from the source image (best-effort; falls back to white `#fff` / dark `#111`): background = median colour over the whole block ("paper"); text = mean of pixels standing out from bg (L1 dist > `90`) within the first line (`1.3 x` line height), else near-black/near-white; contrast floor `55` luma; `0.015`/`6`-px min-ink threshold | [`overlay.go`](../internal/ocr/overlay.go) `blockColors` | [`ocr-overlay.js`](../extension/src/ocr-overlay.js) `blockColors` |
| Noise filter | two gates. **Line confidence:** before clustering, drop a recognized line whose mean word confidence is `< OCR_MIN_LINE_CONF (50)` - real text scores ~80-97, "text" hallucinated from a drawing scores ~0-50, so this keeps plates off imagery and keeps oversized noise boxes from inflating the font. **Text (`isTranslatable`)** on the assembled plate text: drop when `< 5` letters (also kills numbers/symbols); letters but no vowels; the whole text is an address (URL/email/domain/path); or "mishmash" - among letter-bearing tokens, `< 0.5` are word-like (`>= 2` letters + a vowel), needs `>= 3` such tokens. Short CJK (`>= 2` ideographs) is kept | [`tesseract.go`](../internal/ocr/tesseract.go), [`text.go`](../internal/ocr/text.go) `isTranslatable` | [`ocr-cluster.js`](../extension/src/ocr-cluster.js), [`ocr-text.js`](../extension/src/ocr-text.js) `isTranslatable` |
| Pre-OCR resolution handling | Gate on **estimated DPI**, not raw pixel count (a page scan clears 1000 px even at ~100 DPI, so a pixel gate upscales clean renders for nothing or misses the scans that need it). Estimate DPI from the long side over an assumed `OCR_ASSUMED_PAGE_INCHES (11)`-tall page; below `OCR_UPSCALE_DPI_FLOOR (120)` enlarge `OCR_UPSCALE_FACTOR (2 x)` (high-quality) before recognition and divide recognized coordinates back; **always declare the resolution** to Tesseract (`user_defined_dpi`, clamped `>= OCR_MIN_DECLARED_DPI (70)`, doubled when upscaled) so layout analysis separates regions - adjacent balloons - it otherwise merges. Measured: a ~90-DPI newsprint scan gains hugely from the upscale, a ~150-DPI scan only needs the DPI declared (upscaling over-segments it) | [`tesseract.go`](../internal/ocr/tesseract.go) `prepareForOCR` / `estimateDPI` / `scaleDown` | [`ocr-overlay.js`](../extension/src/ocr-overlay.js) `upscaleForOcr` / `estimateDpi` |
| Page-segmentation mode | Tesseract runs in **PSM 3 (AUTO)** on both editions so layout analysis isolates real text regions on an illustrated/scanned page (a speech bubble, a caption) instead of reading the whole frame as one block. The desktop CLI's default is already PSM 3 (made explicit via `--psm`); the extension must set it because tesseract.js defaults to PSM 6 (SINGLE_BLOCK), which folds scene edges into the recognized text (stray punctuation, digits) and mis-merges separate regions into one plate | [`tesseract.go`](../internal/ocr/tesseract.go) `ocrPageSegMode` | [`ocr-overlay.js`](../extension/src/ocr-overlay.js) `OCR_PSM` |
| Grey rescue ladder | An image whose ordinary colour pass returns **no plates at all** is retried on a greyscale copy, first with the engine's own thresholder (`thresholding_method 0`) and then with Leptonica's tiled one (`1`); the first rung that finds plates wins, and an image that reads normally never enters the ladder. Tesseract's default thresholder runs Otsu **per RGB channel** and takes ink only where every channel agrees - harmless on flat paper, but on saturated artwork (a brick-red comic panel behind a white balloon, a coloured poster) the channels disagree over the lettering and the mask that reaches recognition holds no text. The second rung then changes *who* thresholds: one global cut-off cannot survive a background that varies across the image (on a sky gradient Otsu splits the gradient itself and a white caption comes out the same value as its ground), while a tiled thresholder decides locally. It is a **ladder, not a replacement** - the colour pass wins where lettering is separated by hue rather than brightness, so retrying only after an empty result leaves every image that works today unchanged | [`tesseract.go`](../internal/ocr/tesseract.go) `greyRescuePasses` / `greyRescue` / `greyRendition` | [`ocr-overlay.js`](../extension/src/ocr-overlay.js) `GREY_RESCUE_PASSES` / `greyRescue` / `greyRendition` |

**OCR download version and catalog are aligned** (2026-07-01 parity pass) and guarded by
`tests/parity_test.go`:

- **tessdata version = 4.0.0** on both sides. Go pins GitHub `tessdata_fast/raw/4.0.0` (plain,
  [`tessdata.go`](../internal/ocr/tessdata.go)); the extension loads
  `tessdata.projectnaptha.com/4.0.0_fast` (gzip, [`ocr-lang.js`](../extension/src/ocr-lang.js)) and
  bundles eng from the same 4.0.0 ([`build.mjs`](../extension/build.mjs)). Different host/format,
  identical upstream bytes -> matching recognition.
- **Language catalog = 13** on both sides (`eng rus ukr jpn jpn_vert deu fra spa ita por pol chi_sim
  kor`): `tessdata.go` `Available` == `ocr-lang.js` `LANGS`.
- **Overlay grouping constants** identical: `OCR_MIN_LINE_CONF = 50`, `OCR_CLUSTER_PITCH_FACTOR = 1.2`
  and `OCR_MAX_LEADING_RATIO = 3` (`tesseract.go` `ocrMinLineConf` / `ocrClusterPitchFactor` /
  `ocrMaxLeadingRatio` == [`ocr-cluster.js`](../extension/src/ocr-cluster.js)). The **quantity** the
  pitch factor multiplies is part of the invariant, not only its value: equal numbers over different
  quantities still group all-caps lettering differently, so `TestParityOCRClustering` also pins the
  expression that computes the bound (`pitchMax` from the reference pitch) and the one that measures
  a pitch (`y0` to `y0`) on both sides.
- **Overlay resolution constants** identical: `OCR_UPSCALE_DPI_FLOOR = 120`, `OCR_ASSUMED_PAGE_INCHES = 11`,
  `OCR_MIN_DECLARED_DPI = 70` and `OCR_UPSCALE_FACTOR = 2` (`tesseract.go` `ocrUpscaleDPIFloor` /
  `ocrAssumedPageInches` / `ocrMinDeclaredDPI` / `ocrUpscaleFactor` == `ocr-overlay.js`). Guarded by
  `TestParityOCRClustering`.
- **Page-segmentation mode** identical: PSM `3` (AUTO). `tesseract.go` `ocrPageSegMode` (passed to the
  CLI as `--psm 3`) == `ocr-overlay.js` `OCR_PSM` (applied via `setParameters({tessedit_pageseg_mode})`).
  The extension must set it explicitly because tesseract.js defaults to PSM 6 (SINGLE_BLOCK); the
  desktop CLI's own default is already 3.
- **Grey rescue ladder** identical: the retry order for an image the colour pass could not read is
  `[engine-default thresholder (0), Leptonica tiled Otsu (1)]` over a greyscale copy - `tesseract.go`
  `greyRescuePasses` == `ocr-overlay.js` `GREY_RESCUE_PASSES`, and `thresholdEngineDefault` /
  `thresholdLeptonicaOtsu` == `THRESHOLD_ENGINE_DEFAULT` / `THRESHOLD_LEPTONICA_OTSU` (guarded by
  `TestParityOCRGreyRescue`). **Intentional implementation difference:** the desktop app writes an
  8-bit grey PNG, the extension draws through a canvas `grayscale(1)` filter and stays RGBA. Both
  reach the same place - per-channel Otsu over three identical channels is one decision - and the
  browser has no cheap way to emit 8-bit grey.
- **Halftone screen rung** identical: the ladder's **last** rung, tried only after both grey rungs
  returned nothing. It measures the period of the dot screen the picture is printed with and hands
  Tesseract a copy low-passed with a Gaussian of `sigma = pitch / 4`, at the rescue confidence floor
  and the engine-default thresholder. `screen.go` `ocrScreenSigmaDivisor` / `ocrScreenTile` /
  `ocrScreenMinPitch` / `ocrScreenMaxPitch` / `ocrScreenMaxTiles` / `ocrScreenMinEnergy` /
  `ocrScreenPeakFloor` / `ocrScreenTileFrac` == `ocr-screen.js` `OCR_SCREEN_*` (guarded by
  `TestParityOCRScreenRung`, which also pins the rung's **position** after the grey ladder and the
  fact that the sigma is derived from the measured pitch on both sides). The kernel is derived
  rather than fixed because a screen's pitch depends on the press and on the scan resolution; the
  measurements behind the divisor are in
  [`DEV/research/ocr_halftone_2026-08-12.md`](../DEV/research/ocr_halftone_2026-08-12.md).
  **Intentional implementation difference:** the desktop app convolves the 8-bit grey copy itself,
  the extension applies CSS `blur(<sigma>px)` in the same canvas draw as `grayscale(1)`. CSS blur is
  a Gaussian whose length *is* the standard deviation, so both build the same kernel.
- **Additive screen sweep** identical: the same low-pass, spent on a page the ordinary pass *did*
  read. The rescue ladder cannot reach that page - it fires only for an image with no plates at all,
  and on a real comic the dialogue on clean balloons reads fine while the caption printed as a tint
  does not - so `tesseract.go` `screenSweep` == `ocr-overlay.js` `screenSweep` runs on the opposite
  branch. Three parts, all shared and all guarded by `TestParityOCRScreenRung`:
  - **Trigger:** the detector restricted to the area no plate covers (`screenPitchOutside` ==
    `screenPitch(.., covered)`), so the second recognition is spent only where there is screened area
    the reader is not served on. A tile more than `ocrScreenTileCoverMax` == `OCR_SCREEN_TILE_COVER_MAX`
    = `0.5` covered by an existing plate is dropped from the vote.
  - **Merge:** every plate the ordinary pass produced survives untouched; a sweep plate joins it only
    when the plates already accepted cover at most `ocrScreenMergeMaxOverlap` ==
    `OCR_SCREEN_MERGE_MAX_OVERLAP` = `0.2` of *its own* area, measured as the **union** of the
    overlaps. The union is what makes a candidate straddling two existing plates a duplicate; a
    per-plate rule would see two halves under the bound and let it through.
  - **Confidence:** the rescue floor, unchanged. Inherited rather than re-derived, and recorded as a
    lower bound: the local prior is the rescue prior (nothing was found *in this region*) while a
    wrong plate costs more here, landing on a page the reader is otherwise happy with.

  The pass is additive because it is not better as a replacement - measured, it gains on screened
  material (+47% and +18% confident words) and loses badly where there is no screen (16 confident
  words to 0 on one cover), see
  [`DEV/research/ocr_halftone_2026-08-12.md`](../DEV/research/ocr_halftone_2026-08-12.md) §5.
  **Intentional implementation difference:** the extension multiplies its covered rectangles back up
  by the upscale factor before the detector sees them, because `collectLines` has already divided its
  blocks by it while the prepared image has not been downscaled. The Go app downscales after the
  sweep, so there every rectangle is already in prepared-image coordinates.
- **Plate font-fit factor** identical: `0.92` - `overlay.go` `fontFitFactor` == `ocr-overlay.js`
  `FONT_FIT` (guarded by `TestParityOCRFontFit`). Plate font-size = median line height x this factor;
  below `1.0` so translated text (often longer) has room before it overflows the block.

- **Empty-result language report** identical in substance: when a pass recognized nothing at all,
  both editions name the language data that was used - code plus catalog name, `tessdata.go`
  `LangLabel` == `ocr-lang.js` `langLabel` - and say where to change it (`-ocr-lang` / the extension
  popup). "No text found" is a true sentence about the data that was loaded and reads as a verdict on
  the picture, which is the wrong lesson when an English recognizer was pointed at a Russian page.
  Guarded by `TestParityOCRLangReport`. **Intentional difference in the trigger, not the message:**
  the desktop reports once per book, at the end of a pass it knows is finished; the extension's queue
  grows as the reader scrolls and never ends, so it speaks once `OCR_EMPTY_RUN_HINT (3)` images have
  come back with nothing and none has yet carried text. There is no shared constant here - the
  desktop's boundary is "the run finished".

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

**File-type association is opt-in, off by default, on every edition** (2026-07-15). No edition makes
itself the default handler / auto-interceptor without an explicit user action; instead each always
offers a right-click "convert" entry. Desktop: the no-arg first run and GUI launch register only the
non-destructive "Convert to HTML" verb + "Open with" ([`windowsreg`](../internal/windowsreg/register_windows.go)
`RegisterContextMenu`/`RegisterOpenWith`); becoming the default handler is a separate opt-in (CLI
`-register`, GUI association toggle, one-time first-run prompt) and `-unregister` reverses it. Extension:
[`defaults.js`](../extension/src/defaults.js) `enabledByDefault` is **`false`** (no DNR interception until
the popup toggle is on); the "Convert with doc-html-translate" right-click item
([`background.js`](../extension/src/background.js)) is the always-available on-demand path. See
[Intentional divergences](#intentional-divergences-do-not-fix) for the MSIX exception.

### Product URL and feedback address

Every edition surfaces the same product page and the same feedback address. Both are duplicated string
constants - changing either is a cross-edition change; update all rows together.

| Element | Value | Sources |
|---|---|---|
| Product site | `https://serzhyale.github.io/doc-html-translate/` | CLI splash [`app.go`](../internal/app/app.go) (`internal/app/splash/*.txt`), navbar [`projectURL`](../internal/htmlgen/navbar.go#L17), GUI [`ui.html`](../cmd/doc-html-ui/ui.html) byline, extension [`popup.html`](../extension/src/popup.html) / [`viewer.html`](../extension/src/viewer.html) / [`options.html`](../extension/src/options.html) |
| Feedback | `mailto:sza@ukr.net` | CLI splash [`app.go`](../internal/app/app.go), GUI [`ui.html`](../cmd/doc-html-ui/ui.html) byline, extension [`popup.html`](../extension/src/popup.html) / [`viewer.html`](../extension/src/viewer.html) / [`options.html`](../extension/src/options.html) |

### Report field labels

Both editions hand the author a `key: value` block, one field per line, and both write it **in English
whatever the interface language is** - the author reads one format, and a summary they cannot read is
worse than none. The two blocks are produced independently, so the labels they share are pinned here:

| Element | Go app | Extension |
|---|---|---|
| Producer | [`report.Environment`](../internal/report/environment.go) -> `environment.txt` in the archive | [`reportText`](../extension/src/diagnostics.js) -> the clipboard |
| Shared labels | `edition`, `version`, `platform`, `interface language`, `ocr` | same five, same spelling |
| Rest of the block | `ocr languages`, `ocr data dir`, `ollama model` | `user agent`, `auto reflow`, `theme`, `source language`, `ocr language`, `disabled hosts`, `last format`, `last pages`, `last error`, `last run at` |

Renaming a shared label on one side only makes two reports that cannot be read the same way. Guarded by
`TestParityReportFields`.

### OCR lab evidence schema

Not a shipped surface - a **developer** contract. The OCR visual-fidelity lab
([`tools/ocrlab`](../tools/ocrlab/README.md)) grades both editions with one metrics package against one
set of annotations, so both runners must describe a run in the same terms. Only the *shape* is pinned:
strategic §6 accepts that the Tesseract CLI and tesseract.js recognize different text, and each edition
is compared with the annotations rather than with the other's output.

| Element | Go app | Extension |
|---|---|---|
| Schema | [`evidence.SchemaVersion`](../tools/ocrlab/evidence/evidence.go) | `SCHEMA_VERSION` in [`_ocrlab-evidence.mjs`](../extension/scripts/_ocrlab-evidence.mjs) |
| Plate fields | `Plate` JSON tags, in declaration order | `makePlate()` keys, same order |
| Stress cases | `StressCases` in [`stress.go`](../tools/ocrlab/runner/stress.go) | `STRESS_CASES` in [`ocrlab.mjs`](../extension/scripts/ocrlab.mjs) - same six names, texts, factors, directions |
| Viewports | `Viewports` in [`browser.go`](../tools/ocrlab/runner/browser.go) | `VIEWPORTS` in `ocrlab.mjs` - 1280x800@1, 768x1024@1, 390x844@2 |
| Geometry space | natural image pixels | natural image pixels |
| Screenshot space | natural image pixels, resampled locally by `CropToImage` in [`browser.go`](../tools/ocrlab/runner/browser.go) | natural image pixels, resampled locally by `assembleToNatural` in [`_ocrlab-image.mjs`](../extension/scripts/_ocrlab-image.mjs) |
| Clip slack | `evidence.ClipSlackPx` = 4 | `CLIP_SLACK_PX` = 4 |

Both runners capture at the viewport's own resolution and do the mapping into image space themselves.
Neither may ask the browser to render the clip at the natural size: measured on 2026-08-12, a
`Page.captureScreenshot` reply past about four megabytes of base64 is dropped and takes the DevTools
connection with it, which is why 17 of 45 scenes were recorded as extension crashes that were never
crashes at all. The extension additionally captures in bands (`CaptureBandPx`) because its clip is the
image rather than the viewport and can be arbitrarily tall; the Go runner captures the viewport and
needs no equivalent. That is an **intentional difference**, not a gap to close.

The clip slack decides what counts as a translation the reader cannot finish, which is one of the
strategic spec's hard gates, so it has to mean the same thing in both editions. It is 4 px rather than
the re-fit script's 1 px because `scrollHeight` and `clientHeight` are each rounded to an integer from a
fractional layout: measured over the whole corpus on 2026-08-12, every plate a one-pixel rule called
clipped overshot by 2 or 3 px with nothing actually hidden.

Bump the schema version on **both** sides or neither. Guarded by `TestParityOCRLabEvidenceSchema` and by
[`test/ocrlab-evidence.test.mjs`](../extension/test/ocrlab-evidence.test.mjs), which validates a run the
Go runner actually emitted.

### Interface language set, and what the interface language must never touch

Both editions ship the same 13 interface languages, in this order, `en` first:

`en ru uk de it es fr pt ar hi bn ur zh`

RTL is `ar` and `ur`. Script fonts: `Nirmala UI` for `hi`/`bn`, `Microsoft YaHei UI` for `zh`. The extension's
`_locales` directories use Chrome's own naming (`pt`, `zh_CN`); Store locale tags use `pt-br` and `zh-hans`.
Adding a language means adding it on **both** sides plus the site, the installer and the listings.

| Element | Go app | Extension |
|---|---|---|
| Code list | [`i18n.Codes`](../internal/i18n/i18n.go) | [`test/i18n.test.mjs`](../extension/test/i18n.test.mjs) `LOCALES` + `_locales/` dirs |
| Resolution order | explicit `-ui-lang` -> saved -> system ([`i18n.Resolve`](../internal/i18n/i18n.go)) | stored override -> `chrome.i18n.getUILanguage()` -> `en` ([`src/i18n.js`](../extension/src/i18n.js)) |
| RTL / fonts | `i18n.IsRTL`, `i18n.FontFamily` | `RTL_UI_LANGS` in `src/i18n.js`, `FONT_STACKS` in `cmd/doc-html-ui/i18n.js` for the GUI |

**The invariant both sides must keep:** the interface language dresses the *chrome* only. The converted
document keeps its own `<html lang>` and its own direction - the Go side sets `lang`/`dir` on the navbar
div ([`chromeDirAttr`](../internal/htmlgen/navbar.go)), the extension sets them on the toolbar and TOC
scope only ([`applyI18n`](../extension/src/i18n.js)). Carrying the UI language on `<html lang>` would stop
Chrome offering "Translate page", which is the product's entire free workflow. Guarded by
[`TestConvertedChromeLanguage`](../tests/smoke_test.go) and the RTL assertions in
[`make-screenshot.ps1`](../tools/store/make-screenshot.ps1).

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
- **How images reach the page differs by edition, but both show them.** FB2's embedded `<binary>` images:
  Go decodes each referenced binary to a **sibling file** ([`internal/fb2`](../internal/fb2/extract.go),
  `imageFileName`), the extension inlines it as a **`data:` URL** by id
  ([`fb2.js`](../extension/src/fb2.js)) - the same file-vs-inline split as EPUB above. Local images
  referenced by an HTML file: Go **copies the sibling files** from the source's directory subtree into the
  output ([`internal/htmlconv`](../internal/htmlconv/extract.go), `copyLocalImages`, confined against `../`
  traversal); the extension has **no analogue and needs none** - a URL-loaded page lets the browser resolve
  relative images against the origin, and a file picked through the picker grants no directory access to
  reach its siblings anyway. So HTML local-image copying is intentionally **Go-only**.
- **Reader features that are Go-only:** reading-position persistence + "Continue reading", page zoom
  (Ctrl+wheel, `?z=`), and the separate `index.html` TOC page / multi-file navigation. Interface
  localization is **no longer** on this list - both editions ship the same 13 languages, see the invariant
  above.
- **Reader features that are JS-only:** heuristic source-language detection ([`lang.js`](../extension/src/lang.js)),
  the collapsible sidebar TOC, the single continuous-scroll document, and the two toolbar downloads -
  "&#8595; File" (the untouched source bytes) and "&#8595; HTML" (the current on-screen view saved as a
  self-contained `.html`). The HTML export serializes the *live* `#content`, so it captures whatever the
  browser's built-in translator has swapped in - the extension's way of "keeping the translation" without
  a translation API. The Go app has no equivalent: it never translates in place and already writes HTML to
  disk, so both downloads are extension-only by design ([`viewer.js`](../extension/src/viewer.js)).
  Consequently "&#8595; HTML" on a chunk-rendered PDF (below) exports only the pages reached so far, and
  says so in the status bar rather than rendering the remainder: finishing the render would reimpose the
  freeze chunking removes *and* mix untranslated pages under translated ones.
- **Chunked PDF rendering is extension-only.** The viewer renders a PDF forward `PAGE_CHUNK = 50` pages at
  a time, building the next chunk when the reader reaches `CHUNK_LEAD = 2` pages from the edge
  ([`viewer.js`](../extension/src/viewer.js)). This exists because the extension renders *while the reader
  waits* in a live DOM that Chrome's translator is also mutating: an unbounded render loop both looks hung
  on a large book and breaks a translation requested mid-render. Go writes static HTML to disk in a batch
  job with no translator racing it, so it has no reason to chunk and no counterpart. The scanned /
  image-only banner heuristic therefore judges the **first chunk** in JS but the **whole document** in Go -
  same 30%-of-pages-with-text rule, different sample.
- **Sending logs to the author differs in shape, by decision.** The desktop editions keep a bounded
  run-log store on disk ([`internal/report`](../internal/report/store.go)), pack it with the environment
  summary and the last run's settings into one archive, and hand off to the user's mail program - the GUI
  About button and the CLI `-report` flag. The extension only **copies a text summary** to the clipboard
  ([`diagnostics.js`](../extension/src/diagnostics.js), the options page's "Copy diagnostics"), because the
  browser sandbox hosts no run-log store worth archiving and no way to reveal a file or attach one
  (ticket ADR-4). Neither edition uploads anything. The field labels the two do share are pinned in
  [Report field labels](#report-field-labels).
- **Progress bar** looks identical (3px accent bar) but means **reading progress** in Go vs
  **load/OCR progress** in the extension.
- **OCR execution:** Go OCRs eagerly at conversion time, across a pool of `tesseract` processes
  (`ocr.ocrWorkers`), because nothing is readable until the whole file is written; the extension
  both **extracts** a PDF page's rasters and OCRs them lazily on scroll (IntersectionObserver),
  on one worker, because a reader only ever needs the page in front of them and the tab is shared
  with the reading itself. Extraction and recognition deliberately ride the same trigger: doing
  either eagerly costs minutes on a scanned book and buys nothing, since the plate that carries
  the readable text arrives on scroll regardless. The extension has two entry points for OCRing a **standalone image**: the
  right-click "OCR & translate this image", and opening a bare image file (PNG/JPEG/GIF/BMP/WebP) with the
  viewer's **Open file** picker (which OCRs it unconditionally, ignoring the "Use OCR for images" toggle).
  The Go app now **also accepts a standalone image as input** ([`internal/img`](../internal/img/extract.go)):
  a bare PNG/JPG/JPEG/WebP/GIF/BMP/TIFF is wrapped in a one-page HTML doc and the OCR overlay is forced on
  (independent of the `-ocr` flag), so `doc-html-translate <image>` shows the picture with translatable
  plates - the same result as the extension's picker. Both editions share the overlay logic; the remaining
  difference is the engine (extension = tesseract.js WASM + `4.0.0_fast`; Go = the local `tesseract` CLI).
  There is deliberately **no** file-type association (DNR redirect) for image URLs on either side - unlike
  PDF/EPUB, image links are never intercepted; only the explicit picker / right-click / CLI paths OCR images.
- **Comic archives CBR/CB7: desktop-only, by capability.** All four comic containers open on the desktop
  app; the extension implements **CBZ (ZIP) and CBT (TAR) only**. CBR is RAR and CB7 is 7z, neither of
  which has a pure-Go/JS decoder at acceptable weight, so the desktop app shells out to **7-Zip** (detected
  on PATH plus known install paths, the MOBI/Calibre precedent) while the browser - which cannot shell out
  or vendor a RAR/7z decoder - recognizes the RAR/7z signature and shows a "use the desktop app" notice
  instead of a parse error ([`comic.js`](../extension/src/comic.js) `DesktopOnlyError`). This is an
  intentional divergence, not a gap: the desktop side can do what the browser cannot. The desktop app is
  therefore the second runtime-dependency format (after MOBI/Calibre): CBR/CB7 without 7-Zip fail with an
  actionable "install 7-Zip" notice, never a crash or garbage.
- **TIFF: Go transcodes it, the extension refuses it.** Chrome cannot decode TIFF, so both editions must do
  *something* other than show it raw. The extension refuses (its `imageMime` has no TIFF entry, so a `.tif`
  falls through to the PDF reader's clear "cannot read this"), because a browser tab has no decoder. The Go
  app has one (`golang.org/x/image/tiff`), so `internal/img.extractTIFF` **transcodes** each frame to PNG on
  the way in - a multi-page TIFF becomes one PNG page per frame - and honestly supports the format. This is
  an intentional divergence, not drift: the Go side can do what the browser cannot. Keep TIFF out of the
  extension's accepted-image set unless a browser-side TIFF decoder is ever vendored.
- **PDF paths not ported:** Go's pdftotext `-layout` path, its double-spaced/ZWSP paragraph merge, and
  its blank-page skip + `hrefForPDFPage` TOC remap have **no JS counterpart**. Conversely the JS reflow
  adds a **font-size dimension** (`FONT_BREAK_RATIO=0.25`, `big`/`veryBig` heading triggers) and
  geometric centering that Go does not have. (ALL-CAPS heading detection is now **aligned**: both sides
  are language-agnostic, so Cyrillic headings like "ГЛАВА ПЕРВАЯ" are detected on both.) JS image
  extraction also normalizes **mirrored image placements** - a negative CTM scale at paint time means
  the raster is stored flipped, so [`pdf-images.js`](../extension/src/pdf-images.js) mirrors the pixels
  back to their rendered orientation; Go's pdfcpu extraction writes raw streams and has no counterpart yet.
- **A vector page with no text and no usable raster is handled differently, by capability.** When a page
  has no text layer and no page-covering image (a vector chart with outlined labels), the Go app copies the
  original PDF beside the output and shows it in an `<embed>` with a "no extractable text layer" note
  ([`buildFallbackPDFHTML`](../internal/pdf/extract.go)) - it never rasterizes a page itself. The extension
  instead **rasterizes the whole page** to one image ([`rasterizePage`](../extension/src/pdf-images.js), the
  `pageChars < 20` fallback), because pdf.js renders vector content in-tab where the Go side has no renderer.
  Same goal - never a blank page or a thumbnail stamp - reached by each edition's available means.
- **Translation target:** the extension has **no target language** - it delegates to the browser's
  built-in "Translate page", so `-dst` is CLI/GUI-only.
- **Storage:** Go uses `localStorage`/`sessionStorage` string keys (`dht_*`); the extension uses
  `chrome.storage.local` objects. Reading preferences are not portable between the two.
- **MSIX default-handler / right-click:** on the Store/MSIX build the file-type association comes from
  the package manifest ([`AppxManifest.xml`](../msix/AppxManifest.xml) `windows.fileTypeAssociation`),
  which Windows never force-defaults (it always prompts) and only surfaces in "Open with" - so it is
  opt-in by OS design. The unpackaged `RegisterContextMenu` / `-register` / `-unregister` flows are a
  no-op there because MSIX **virtualizes HKCU** writes, so the GUI hides the association toggle and the
  first-run prompt when packaged ([`isPackaged`](../cmd/doc-html-ui/main.go)). A native `IExplorerCommand`
  context-menu handler (a COM component in the package) would add a dedicated "Convert to HTML" verb under
  MSIX too; it is deferred, not "missing". Do not try to write the HKCU verb from the packaged app.

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
   pin a new invariant here. These guard the **value** invariants. The extension's own DOM path (chapter
   sanitize / image / link / TOC) is covered by `extension/test/epub-dom.test.mjs` +
   `extension/test/sanitize.test.mjs` under `npm test` (a dev-only `linkedom` DOM, never bundled) - these
   assert the JS-side behaviour that mirrors `internal/epub`, complementing the value guards above.
6. **Structural drift-check (advisory):** `scripts/parity-check.ps1` (alias `a pc`, and run in the
   `scripts/check.ps1` gate) mirrors the port map above and *warns* when a Go extractor changed without its
   paired JS module, or vice versa. It never blocks - it turns silent drift into a prompt. Touching this
   file (docs/PARITY.md) in the same change set silences it, so the escape hatch for an intentional
   one-sided change is to record it here under [Intentional divergences](#intentional-divergences-do-not-fix).
   Keep the port map in the script in sync with the table at the top of this file.
