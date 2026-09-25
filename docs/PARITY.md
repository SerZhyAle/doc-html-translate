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
| PDF page images: select + same-shape dedupe | [`internal/pdf/extract.go`](../internal/pdf/extract.go) (`selectPageImages`, `sameShapeRaster`) | [`extension/src/pdf-images.js`](../extension/src/pdf-images.js) (`dedupeSameShape`, `sameShapeRaster`) |
| EPUB unzip + OPF/spine + sanitize + TOC | [`internal/epub/`](../internal/epub/) (`epub.go`, `toc.go`) | [`extension/src/epub.js`](../extension/src/epub.js) |
| Plain text -> paragraphs/pages | [`internal/txt/`](../internal/txt/) | [`extension/src/txt.js`](../extension/src/txt.js) |
| Plain text: source-encoding decode | [`internal/txt/extract.go`](../internal/txt/extract.go) (`decodeText`) | [`extension/src/txt.js`](../extension/src/txt.js) (`decodeText`) |
| RTF reader + code-page decode | [`internal/rtf/`](../internal/rtf/) | [`extension/src/rtf.js`](../extension/src/rtf.js) |
| Markdown -> HTML | [`internal/md/`](../internal/md/) (`goldmark`) | [`extension/src/md.js`](../extension/src/md.js) (vendored `marked`) |
| FB2 XML -> sections/TOC | [`internal/fb2/`](../internal/fb2/) | [`extension/src/fb2.js`](../extension/src/fb2.js) |
| HTML `<body>` extract | [`internal/htmlconv/`](../internal/htmlconv/) | [`extension/src/html.js`](../extension/src/html.js) |
| MOBI / AZW3 (KF8) | [`internal/mobi/`](../internal/mobi/) (shells out to Calibre) | [`extension/src/ebook.js`](../extension/src/ebook.js) (vendored `foliate-js`) |
| Comic archive -> page book | [`internal/comic/`](../internal/comic/) (CBZ/CBT stdlib; CBR/CB7 shell out to 7-Zip) | [`extension/src/comic.js`](../extension/src/comic.js) (CBZ/CBT only; CBR/CB7 declined) |
| Comic natural page order + entry filter | [`internal/comic/natural.go`](../internal/comic/natural.go), `extract.go` (`isPageEntry`) | [`extension/src/comic.js`](../extension/src/comic.js) (`naturalCompare`, `isPageEntry`) |
| Comic forced-OCR decision | [`internal/pipeline/pipeline.go`](../internal/pipeline/pipeline.go) (`comic.IsComic` -> `forceOCR`) | [`extension/src/viewer.js`](../extension/src/viewer.js) (`loadComicData` -> `registerImagesForOcr(.., true)`) |
| Input limits (archive listing budget, per-entry caps, capped inflation) | [`internal/limits/`](../internal/limits/) (+ `internal/epub` `maxEntryBytes`, `internal/comic` `maxPageBytes`) | [`extension/src/limits.js`](../extension/src/limits.js) |
| HTML sanitize -> fragment | (EPUB-only in Go: `epub.go` normalize) | [`extension/src/sanitize.js`](../extension/src/sanitize.js) |
| OCR overlay (recognize -> plates) | [`internal/ocr/overlay.go`](../internal/ocr/overlay.go), `tesseract.go` | [`extension/src/ocr-overlay.js`](../extension/src/ocr-overlay.js) (recognition) + [`ocr-plates.js`](../extension/src/ocr-plates.js) (plates) + `ocr-overlay.css` (overlay rules generated from `internal/appearance`) |
| OCR line clustering + text filter | [`internal/ocr/tesseract.go`](../internal/ocr/tesseract.go), [`text.go`](../internal/ocr/text.go) (`isTranslatable`) | [`extension/src/ocr-cluster.js`](../extension/src/ocr-cluster.js), [`ocr-text.js`](../extension/src/ocr-text.js) (`isTranslatable`) |
| Whole-page OCR on a live web page | (none - extension-only by design, see Intentional divergences) | [`extension/src/page-ocr.js`](../extension/src/page-ocr.js) (broker), [`page-agent.js`](../extension/src/page-agent.js) (in-page), [`ocr-host.js`](../extension/src/ocr-host.js) (engine host) |
| OCR language manager | [`internal/ocr/tessdata.go`](../internal/ocr/tessdata.go) | [`extension/src/ocr-lang.js`](../extension/src/ocr-lang.js) |
| Reader chrome (themes, fonts, controls) | [`internal/htmlgen/navbar.go`](../internal/htmlgen/navbar.go) (`readerCSS`, `readerScript`) | [`extension/src/viewer.css`](../extension/src/viewer.css), [`viewer.js`](../extension/src/viewer.js), [`viewer.html`](../extension/src/viewer.html) |
| Source-language detection | (none - Go copies the source `<html lang>`) | [`extension/src/lang.js`](../extension/src/lang.js) |
| Settings / options surface | [`internal/config/flags.go`](../internal/config/flags.go), [`ui.html`](../cmd/doc-html-ui/ui.html) | [`popup.js`](../extension/src/popup.js), [`options.js`](../extension/src/options.js), [`background.js`](../extension/src/background.js) |

## Shared invariants (MUST stay identical on both sides)

These are duplicated across codebases, mostly with no shared source. Changing a value on one side
without the other is a bug. Each row cites the two places that must agree.

Every section below opens with a **Guard** line: `Guarded by` names the test that fails when the two
sides part, `Prose only` means nothing but this document holds them together - being right there is
currently luck, and the line names what a future ticket would have to pin. The marks were derived from
the test files (`tests/*_test.go`, `extension/test/*.test.mjs`) on 2026-09-24; a one-sided unit test
does not count, only a check that reads both editions or a single source both derive from.

### Shared appearance (OCR overlay unit, reader theme palette)

**Guard:** Guarded by `TestAppearanceRolesMatchSource`, `TestAppearanceNoRoleDeclaredOutsideSource` and
`TestAppearanceComparatorDetectsDrift` ([`tests/appearance_parity_test.go`](../tests/appearance_parity_test.go)).

The one invariant here that is **not duplicated**: the OCR overlay's container, image and plate roles and
the four reader themes are written once, in
[`internal/appearance/appearance.json`](../internal/appearance/appearance.json), and both editions derive
their CSS from it. The desktop app builds it at run time
([`internal/appearance`](../internal/appearance/appearance.go) `OverlayCSS` / `PaletteCSS`, called by
`overlay.go` and `navbar.go`) and still inlines it into every page; the extension generates it into marked
regions of `ocr-overlay.css` and `viewer.css` ([`gen-appearance.mjs`](../extension/scripts/gen-appearance.mjs),
`npm run appearance`, checked before every package). **Neither edition's copy may be edited by hand** -
change `internal/appearance/appearance.json` and regenerate. See its [README](../internal/appearance/README.md).

The gate compares every declaration of every role and theme on both sides against the source, keyed on
the role or theme, never on a selector or custom-property name, so naming stays per-edition and outside
the comparison. A declaration one side has and the other lacks fails and names the property, the role and
the side missing it - the hairline `box-shadow` ring that shipped on the extension's plate only is the
case it was built from. A difference is legal only when the source's `divergences` list names it with a
reason; that list is what the gate reads, and an entry that licenses a declaration but matches nothing
fails too. It also fails on a rule outside the derived path - outside the generated regions, or anywhere
in the desktop Go sources - that targets a role selector or declares a palette colour.

The first full run (2026-09-24) found one equivalence case and resolved it rather than listing it: the
image role's reset guard (`margin:0; max-height:none`) sat on the role in the desktop app but in the
extension's reader stylesheet only, so the standalone OCR page relied on a browser default. It is on the
role now, on both sides; the viewer's `#content img` reset skips the overlay image instead of being
overridden. The viewer's spacing of the container in its reading column is named in `divergences` as
placement, not appearance.

### Input format detection is by byte signature, not extension

**Guard:** Prose only. The Go sniffer has its own unit tests (`internal/txt/sniff_test.go`), but nothing
compares the two signature lists; a future ticket would pin the set of magics both `detectFormat` and the
Go sniffer recognize, and the `PK..` tie-break by extension.

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
  not mistaken for binary, and BOM-less UTF-16 is recognized by its NULs sitting on one byte parity (the
  text decoder's step 4, below).

The two are not byte-identical by design - the extension re-renders in a live tab and leans on the PDF
reader's error path, while the Go CLI is a batch converter that must refuse with a non-zero exit and no
output directory. What must stay true on both: **detection is signature-first, and a binary never
becomes a document.** Go: `internal/txt/sniff.go`. JS: `extension/src/viewer.js` `detectFormat` /
`imageMime` / `isMobiBytes`.

### Plain-text source-encoding decode order

**Guard:** Guarded by the shared fixtures in [`tests/testdata/legacy-text/`](../tests/testdata/legacy-text/)
(`tests/legacy_text_test.go` `TestLegacyTextFixtures` and `extension/test/legacy-text.test.mjs` read the
same inputs and expected texts), and by `TestParityLegacyTextTables`, which compares the candidate list,
the Western label and the UTF-16 sniff window. The damage threshold and the NUL-pattern rule are pinned
by the fixtures, not compared as source.

Both editions decide a `.txt` file's encoding from its bytes, in this order. The **order is the
invariant**: the same file must not read correctly on one edition and as mojibake on the other.

| # | Test | Result |
|---|---|---|
| 1 | `EF BB BF` | UTF-8; the mark is removed, never shown |
| 2 | `FF FE` | UTF-16LE |
| 3 | `FE FF` | UTF-16BE |
| 4 | BOM-less UTF-16: in the first 4096 bytes the NULs sit on one byte parity (below) | UTF-16LE (odd offsets) or UTF-16BE (even offsets) |
| 5 | UTF-8 with damage below the threshold (below) | UTF-8; each invalid subpart becomes one U+FFFD |
| 6 | a legacy Cyrillic code page is confidently detected (below) | that code page |
| 7 | otherwise | windows-1252, the Western fallback |

- **BOM-less UTF-16 (step 4).** The high byte of every Latin letter, digit, space and punctuation mark is
  zero, and 8-bit text has no NUL at all. The dominant parity must hold at least 2 NULs and cover 5% of
  the code units; the other parity at most a tenth as many. Such a file is often valid UTF-8 (ASCII and
  Cyrillic code units are all bytes below 0x80), which is why this step comes before step 5. The Go
  binary sniff (`LooksBinary`) accepts the same pattern as text.
- **Damaged-UTF-8 threshold (step 5, owner decision 2026-09-25).** Counted with the WHATWG UTF-8
  decoder (the one `TextDecoder` runs): the bytes are UTF-8 when they hold at least one valid multi-byte
  sequence and the invalid bytes are under 1% of those sequences, or when the only invalidity is a
  sequence cut off by the end of the data. One bad byte used to send a whole Russian book to step 6.
- **Output.** Always valid UTF-8; damage is visible as U+FFFD, one per maximal invalid subpart (the
  WHATWG rule), never dropped. Go: `internal/textutil` `DecodeUTF8`, which the PDF line normalizer uses
  too (`pdftotext` output).

Go: `internal/txt/extract.go` `decodeText`, `internal/txt/decode.go` (`sniffUTF16`, `acceptAsUTF8`),
`internal/textutil` (`MeasureUTF8`, `DecodeUTF8`, `LookupCodec`).
JS: `extension/src/txt.js` (`decodeText`, `sniffUtf16`, `measureUtf8`, `acceptAsUtf8`).

**One difference that is not drift:** step 1 is implicit on the JS side. `TextDecoder` strips a leading
BOM by itself unless `ignoreBOM` is set, so the extension never had the UTF-8-BOM leak the Go side did.
Both arrive at the same text; only Go has to say so out loud.

**Code-page tables are the browser's.** Every legacy decode (TXT candidates, the Western fallback, RTF
code pages, FB2 declarations) resolves a WHATWG Encoding Standard label: `TextDecoder(label)` in the
extension, `internal/textutil` `LookupCodec` (`golang.org/x/text/encoding/htmlindex`) in Go. Where a
single-byte code page leaves a byte in 0x80-0x9F undefined (0x81 in windows-1252, 0x98 in
windows-1251), the WHATWG index maps it to the C1 control of the same value and `x/text` to U+FFFD;
`LookupCodec` follows the browser, so both editions emit the same character.

Scope: `.txt`, RTF and FB2 (next section). `md.js` and `html.js` still decode as UTF-8 unconditionally;
the HTML-input gap is recorded under "EPUB and HTML content fidelity".

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
  bytes fall through to windows-1252 (step 7). Measured: the real cp1251 fixture is 0.76; French Latin-1
  mis-read as KOI8-R (which tops the *weight* score) is 0.17, so the floor rejects it. Selection needs
  weight, confidence needs fraction - neither metric alone does both.
- **Known limit, accepted:** a very short, accent-dense non-Russian string can exceed 0.30 and be
  mis-decoded. Short files carry too little signal; the floor is the agreed trade.

Go: `internal/txt/legacy.go` (`legacyCandidates`, `ruLetterFreq`, `cyrillicFit`, `minCyrillicFraction`,
`detectLegacy`). JS: `extension/src/txt.js` (`LEGACY_CANDIDATES`, `RU_LETTER_FREQ`, `cyrillicFit`,
`MIN_CYRILLIC_FRACTION`, `detectLegacy`). RTF does not guess: its code page is declared (next section).

### RTF and FB2 text decoding

**Guard:** Guarded by the shared fixtures in [`tests/testdata/legacy-text/`](../tests/testdata/legacy-text/)
(WordPad Russian RTF with a font table, LibreOffice RTF with `\uc` and `\'XX` fallbacks, a cp1252 RTF, a
windows-1251 FB2 with a poem), read by `tests/legacy_text_test.go` and
`extension/test/legacy-text.test.mjs`, and by `TestParityLegacyTextTables`, which compares the RTF code-page,
charset, destination, symbol and break tables and the FB2 declaration window value by value. The same
RTF unit cases run on both sides (`internal/rtf/parse_test.go`, `extension/test/rtf.test.mjs`).

**RTF** is read in one forward pass with RTF's group state:

- **Destinations.** Each `{` copies the group state and `}` restores it. The font table is read for
  charsets and never shown. The known non-text destinations (`colortbl`, `stylesheet`, `info`, `pict`,
  `object`, header and footer variants, `listtable`, `fldinst`, `xe`, `tc` and the rest of
  `skippedDestinations`) and every `{\*\..}` group are skipped. Only known names are listed, so an unknown
  generator's text is never hidden. `fldrslt`, `pntext` and `listtext` are visible text and stay.
- **Unicode.** `\uN` is signed 16-bit (a negative N adds 65536); a surrogate pair of two `\u` words
  becomes one character, a lone surrogate U+FFFD. After each `\uN` the next `\ucN` characters are the
  fallback and are dropped: a text byte, a `\'XX` escape or a control word each counts as one; `{` or `}`
  ends the fallback. `\ucN` defaults to 1 and is scoped to its group.
- **Binary.** `\binN` skips N raw bytes by count, braces and backslashes included.
- **Control symbols.** `\~` U+00A0, `\_` U+2011, `\-` dropped, `\{ \} \\` literal, `\<newline>` a paragraph
  break. Symbol words (`\emdash`, `\ldblquote`, `\bullet`, ..) map to their characters; `\par`, `\line`,
  `\sect`, `\page`, `\row` break the paragraph; `\tab` and `\cell` are a tab. Raw CR/LF in the source are
  not text.
- **Code page.** `\'XX` escapes and raw high bytes decode in the current font's code page (`\fcharsetN`
  mapped to a code page, or the font's `\cpgN`), else the document's `\ansicpgN` (`\mac` = 10000), else
  1252. `\fcharset0` and `1` mean "the document's code page". A code page with no WHATWG label (437,
  850) falls back to 1252. Consecutive bytes decode together, through one decoder per code page per
  document, so a double-byte code page sees both halves of a character and nothing creates a decoder
  per byte.

Go: `internal/rtf/parse.go`, `internal/rtf/codepage.go`. JS: `extension/src/rtf.js`.

**FB2:**

- **Encoding.** A BOM wins, then the encoding the XML declaration names (in the first 1024 bytes),
  resolved as a WHATWG label, then UTF-8. An unknown label, or a UTF-16 label with no BOM, reads as
  UTF-8; damaged UTF-8 shows U+FFFD instead of failing the parse.
- **Prose elements.** Everything in a `<body>` that holds text becomes a paragraph, in document order:
  `<p>` (in sections, epigraphs, citations, annotations, titles), `<subtitle>` (class `subtitle`),
  `<text-author>` (class `text-author`), table cells `<td>`/`<th>`, and each `<stanza>` as one paragraph
  of class `stanza` with its `<v>` lines separated by `<br>`. Inline markup is flattened and whitespace
  collapses to single spaces. A section title is a paragraph in the Go pages and a heading in the
  extension, as before.

Go: `internal/fb2/content.go` (`decodingReader`, `parseFB2`). JS: `extension/src/fb2.js` (`decodeFb2`,
`renderBlock`).

### Reader theme palette

**Guard:** Guarded by `TestAppearanceRolesMatchSource` - the palette is part of the
[shared appearance](#shared-appearance-ocr-overlay-unit-reader-theme-palette), derived on both sides
from `internal/appearance/appearance.json`.

Exactly four themes, in this order: **`light`, `sepia`, `dark`, `night`**. Eight colour tokens per
theme. The **values are identical by construction**; only the custom-property names and the `<html>`
attribute differ (see [Intentional divergences](#intentional-divergences-do-not-fix)).

The table is the human-readable form of `internal/appearance/appearance.json` - generated-from, not
authoritative. Change a colour there, not here, then update this table.

| Theme | bg | fg | muted | bar-bg | bar-fg | border | accent | link |
|---|---|---|---|---|---|---|---|---|
| light (`:root`) | `#faf9f7` | `#1b1b1b` | `#6b6b6b` | `#ffffff` | `#222222` | `#e2e0db` | `#2563eb` | `#1a4fb4` |
| sepia | `#f4ecd8` | `#4a3f2f` | `#7a6c54` | `#efe6cf` | `#4a3f2f` | `#ddd0b0` | `#8a5a2b` | `#7a4a1b` |
| dark | `#1a1a1c` | `#e6e4df` | `#9a9893` | `#232327` | `#e6e4df` | `#36363b` | `#5b8dff` | `#8fb4ff` |
| night | `#0a0a0b` | `#9a9a9a` | `#6a6a6a` | `#131315` | `#b8b8b8` | `#262629` | `#5599d6` | `#6aa8e0` |

Emitted by [`navbar.go`](../internal/htmlgen/navbar.go) `readerCSS` (`--dht-*`, `data-dht-theme`) and the
generated region of [`viewer.css`](../extension/src/viewer.css) (`--*`, `data-theme`).

### Reader fonts

**Guard:** Prose only. A future ticket would pin the three family strings in `readerScript` `FAMILIES`
against `viewer.js`, whitespace- and quote-normalized - or move them into `internal/appearance`.

Serif / sans / mono families, identical strings both sides:
`serif` = `Georgia,"Times New Roman",serif` · `sans` = `"Segoe UI",system-ui,Arial,sans-serif` ·
`mono` = `"Cascadia Code",Consolas,monospace`. Sources:
[`navbar.go:404-408`](../internal/htmlgen/navbar.go#L404-L408),
[`viewer.js:91-95`](../extension/src/viewer.js#L91-L95).

### PDF reflow heuristics

**Guard:** Guarded by `TestParityReflowConstants`.

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

**Guard:** Prose only. A future ticket would pin `aspectRatioTolerance` == `ASPECT_RATIO_TOLERANCE` and
the keep-the-largest rule.

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

**Guard:** Prose only. A future ticket would pin the source priority and the `<nav>` selection order,
most cheaply by running one fixture EPUB through both parsers and comparing the trees.

| Rule | Both sides |
|---|---|
| TOC source priority | EPUB3 `nav.xhtml` (`properties="nav"`) preferred; EPUB2 `toc.ncx` used **only** if nav yields 0 entries |
| NCX item lookup | `<spine toc="..">` idref first, then media-type `application/x-dtbncx+xml` |
| `<nav>` selection order | `epub:type="toc"` -> `role="doc-toc"` -> first `<nav>` |
| Entry survival | drop an entry whose target is unresolvable **unless** it has surviving children (then it becomes a label-only node) |
| Label | first `<a>` (or `<span>`) not inside a nested `ol`/`ul` |

Sources: [`internal/epub/toc.go`](../internal/epub/toc.go), [`extension/src/epub.js`](../extension/src/epub.js).
Title whitespace normalization (Go NCX now uses `collapseWS`, matching the nav path and the extension)
was aligned in the 2026-07-01 parity pass. External TOC links (ticket
`bugfix-reader-layer-and-single-page`, 2026-09-25): Go's `ExternalHref` ([`links.go`](../internal/epub/links.go))
counts any scheme or `//` prefix as external, keeps `http`/`https`/`mailto` entries as written (never under
the base folder) and turns any other scheme (`javascript:`, `data:`, `vbscript:`, `tel:`, ..) into a
label-only entry. The extension's `isExternalHref` (any `://`, or `mailto:`/`tel:`/`data:`) drops every
external entry, and a `javascript:` value fails `resolveBookPath` (colon), so neither edition ever renders a
script link. The remaining intentional difference - Go keeps web/mail TOC entries, the extension drops them
(single in-memory DOM) - is listed under [Intentional divergences](#intentional-divergences-do-not-fix).

### EPUB href resolution

**Guard:** Guarded by the shared fixture [`tests/testdata/epub_href_cases.json`](../tests/testdata/epub_href_cases.json),
which `TestResolveBookPathSharedCases` ([`internal/epub/resolve_test.go`](../internal/epub/resolve_test.go)) and
`resolveBookPath: shared Go/JS fixture` ([`extension/test/epub.test.mjs`](../extension/test/epub.test.mjs)) both
run. A case added for one edition runs against the other.

Every name the book supplies - the container `full-path`, each manifest href, and (in the extension) each
link, image and TOC target - becomes a path through one function, `resolveBookPath`
([`resolve.go`](../internal/epub/resolve.go), [`epub.js`](../extension/src/epub.js)). Ticket
`hotfix-epub-href-containment` introduced it after a crafted spine made the desktop single-page merge read
and then delete a file next to the book.

| Step | Both sides |
|---|---|
| 1 | cut `?query` and `#fragment` off the **raw** value, so an encoded `%23` stays part of the file name |
| 2 | percent-decode once; a malformed or non-UTF-8 escape keeps the raw text |
| 3 | `\` is a separator |
| 4 | refuse a colon (drive letter, scheme, stream), a leading `//` (UNC), control characters, a segment ending in a dot or space, and DOS device names (`CON`, `NUL`, `COM1`..) |
| 5 | a leading `/` is relative to the book root; anything else resolves against the referring file's directory |
| 6 | the result must name something strictly inside the book, or the name is dropped |

A manifest item that fails is dropped with a warning (Go: localized log line, extension: console), and its
spine entries go with it; a spine item whose file is missing costs only that chapter. The Go app resolves
once, at OPF parse time, and every later stage uses the resolved `ManifestItem.Href` - a decoded file path
that generated HTML escapes through `URLPath`.

Intentional difference: case collisions. The Go app writes files, so it compares its generated names
(`index.html`) with book files ignoring case and warns when two archive entries differ only in case (NTFS
would keep one). The extension keeps entries in a `Map` keyed by exact name, where nothing is overwritten.

### EPUB and HTML content fidelity (2026-09-25)

**Guard:** Prose only. The desktop behaviour is covered by its own package tests; nothing compares it
with the extension. A future ticket would port the two extension gaps below and pin them with one
fixture read by both editions.

Ticket `06_2026-09-24_bugfix-epub-html-content-fidelity` moved the desktop EPUB normalization onto the
parsed tree and made HTML input charset-aware. Checked against the extension on the same date:

| Behaviour | Go app | Extension |
|---|---|---|
| Single-image SVG cover -> `<img>` | DOM, only an `<svg>` whose sole content is one `<image>` ([`normalize.go`](../internal/epub/normalize.go) `rewriteCoverSVGs`) | DOM, `convertSvgImage` in [`epub.js`](../extension/src/epub.js) replaces an `<svg>` holding one `<image>` - but does not check for `<text>` beside it, so a titled cover drawing loses its text |
| Link rewrites | real link attributes only, resolved per file ([`links.go`](../internal/epub/links.go) `rewriteLinks`) | `<a>` on the DOM, resolved per chapter (`rewriteAnchor`) - same model |
| XHTML self-closing tags (`<a id/>`, `<script/>`, `<title/>`) | expanded before parsing | **open gap:** `renderChapter` parses the XHTML as `text/html`, so the same Calibre markup still swallows text in the viewer |
| Chapter / HTML-input charset | BOM, then XML declaration or meta charset, then detection ([`charset.go`](../internal/epub/charset.go), [`htmlconv`](../internal/htmlconv/extract.go)) | **open gap:** `TextDecoder("utf-8")` in `epub.js` `decodeText` and `html.js` - a windows-1251 page reads as mojibake |
| Long-chapter splitting | [`htmlsplit`](../internal/htmlsplit/) splits through wrappers, by characters, keeping root attributes and retargeting links | none - the viewer renders one DOM, nothing to split (by construction) |
| HTML input images / styles | copied locally ([`internal/assets`](../internal/assets/)) | the viewer cannot reach a local page's sibling files (by construction) |

### Single-page merge and the reader layer (2026-09-25)

**Guard:** Prose only on the cross-edition side. The desktop behaviour is pinned by
`internal/htmlgen/merge_test.go` and `reader_key_test.go`; nothing runs the extension against it.

Ticket `09_2026-09-24_bugfix-reader-layer-and-single-page` fixed the desktop merge and reader script.
Checked against the extension on the same date:

| Behaviour | Go app | Extension |
|---|---|---|
| Merging chapters into one page | [`merge.go`](../internal/htmlgen/merge.go) `prepareMerge`: relative `src`/`href`/`srcset` and CSS `url()` (style attributes, `<style>` blocks) rebased from the chapter folder to the merged page's folder; only **colliding** ids renamed `cN-<id>`, so book CSS aimed at ids keeps working | `renderChapter` namespaces **every** id `d<index>-<id>` in one in-memory DOM and loads images as `blob:` URLs, so there is no folder to rebase (by construction) |
| `chapter.html#note` and bare `chapter.html` links | in-page `#<id>` / `#dht-ch-N` chapter marker; a root (`<body>`) id lands on the marker | `rewriteAnchor`: `#d<idx>-<frag>` / `#epub-sec-<idx>`; root ids re-exposed as marker anchors - same model |
| Reading-position key | `epub.Book.ReaderKey` from source name + size + original title + page count, set once before translation | **n/a** - the viewer persists no reading position (Go-only feature, see Intentional divergences), so it has neither the old key drift nor a key to align |
| Restore vs URL fragment | restore only when `location.hash` is empty | **n/a** - no restore; a TOC click scrolls to the anchor directly (`scrollToAnchor`) |
| Script literals / hrefs | `jsString` (JSON) for script values, `epub.URLPath` for every generated path | links stay DOM attributes set through `setAttribute` - nothing is spliced into script text |

### Comic archive page order and entry filter

**Guard:** Guarded by `TestParityComicPageFilter` for the page-extension set. Page order (`naturalLess` /
`naturalCompare`) is prose only; a future ticket would run one list of names through both comparators.

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
- **TAR entry names follow Go's `archive/tar`** (2026-09-25). A GNU `L` record names the next entry, a
  PAX `x` record's `path=` / `size=` override the next header, a `GNU.sparse.*` key marks the entry sparse
  (skipped - its stored bytes are not the page), the ustar prefix field is joined only in USTAR/PAX headers
  (a GNU header keeps other data there), and a `\0` typeflag counts as a regular file unless the name ends
  in `/`. The `L`/`K`/`x` records are consumed and not counted as entries, as in Go, so the entry budget
  matches too. Go: [`readers_tar.go`](../internal/comic/readers_tar.go) (`tar.Reader`). JS: `comic.js`
  `tarEntries`, pinned by `test/comic.test.mjs`. Before this the extension read only the 100-byte name
  field and the prefix, so long page names and PAX archives listed different pages than the desktop.

Container support differs by capability, not drift (see [Intentional divergences](#intentional-divergences-do-not-fix)):
the desktop app opens all four (CBR/CB7 by shelling out to 7-Zip, the MOBI/Calibre precedent), while the
extension opens **CBZ (ZIP) and CBT (TAR) only** - a browser has no RAR/7z decoder and cannot shell out,
so it recognizes a CBR/CB7 by signature and shows a "use the desktop app" notice.

### Input limits

**Guard:** Guarded by `TestParityInputLimits` ([`tests/limits_parity_test.go`](../tests/limits_parity_test.go)),
which compares the Go and JS values and pins the published numbers. Ticket
`12_2026-09-24_bugfix-resource-budgets`.

One hostile or merely huge file must be turned into a message before it is allocated: the desktop app
ships a 32-bit build with a 2 GB address space, and a browser tab has less. Both editions probe first (an
archive listing, an image header) and refuse or degrade from the probe. The same archive must be refused
by both editions, so these numbers are one invariant:

| Limit | Value | Go | JS |
|---|---|---|---|
| Archive entry count (EPUB, CBZ, CBT; CBR/CB7 desktop only) | `20000` - also the comic page cap | `limits.MaxArchiveEntries` | `ARCHIVE_MAX_ENTRIES` |
| Archive unpacked total, over the entries that will be unpacked | `4 GB` (`4 << 30`) | `limits.MaxArchiveTotalBytes` | `ARCHIVE_MAX_TOTAL_BYTES` |
| One EPUB file | `100 MB` | `internal/epub` `maxEntryBytes` | `EPUB_MAX_ENTRY_BYTES` |
| One comic page | `200 MB` | `internal/comic` `maxPageBytes` | `COMIC_MAX_PAGE_BYTES` |
| Full image decode (desktop only) | `100` megapixels and `32768` px per side | `limits.MaxImagePixels` / `MaxImageSide` | - (the browser decodes images itself) |

The rules that go with the numbers:

- **Listing first.** Entry count and unpacked total are checked from the ZIP central directory, the TAR
  headers or `7z l -slt` before any entry is unpacked; over either, the whole archive is refused with a
  localized message naming the limit. The total counts only the entries that will be unpacked (an entry
  skipped for its own size does not count).
- **Per-entry caps skip by name.** An entry whose listed size is over its cap is skipped with a warning
  that names it (Go: console/run log; JS: `console.warn`), and the rest of the book converts. An entry that
  holds more bytes than its listing states is an error for that entry, never a silently shortened file
  (Go: `limits.CopyCapped`, and the zip reader's own size check; JS: `inflateRawCapped` capped at the
  listed size).
- **Inflation counts bytes.** Neither edition inflates an entry whole and measures afterwards.
- **The extension's page raster is capped** (2026-09-25, extension-only): a scanned PDF page rasterized
  for OCR is drawn at scale 2 unless that canvas would pass `RASTER_MAX_PIXELS` (16 MP) or
  `RASTER_MAX_SIDE` (8192 px), in which case the scale shrinks to fit ([`pdf-images.js`](../extension/src/pdf-images.js)
  `rasterScale`). No shared constant: the desktop app renders pages through its own tools and has the
  decode budget above instead.
- **Symlinks are never followed.** Symlink entries are skipped from the listing; on the desktop, files
  7-Zip unpacked are `Lstat`-checked.
- **Container by signature.** `PK\x03\x04` ZIP, `Rar!\x1a\x07` RAR, `7z\xBC\xAF\x27\x1C` 7z, `ustar` at
  offset 257 TAR; the extension is only a fallback. Go: `internal/comic` `sniffContainer`; JS: `comic.js`
  `detectContainer`. A RAR saved as `.cbz` converts on the desktop through 7-Zip and is declined in the
  extension with the "use the desktop app" notice.

Desktop-only, by capability: a TIFF frame or a PDF TIFF whose header is over the pixel budget is refused
(converting it needs the full decode). An image on a page over the budget is shown untouched and OCR
degrades: Tesseract still reads it in its own process, while the in-process passes (staging, grey ladder,
screen pass, plate colours) are skipped with a warning, because shrinking it would itself need the full
decode. The OCR worker pool is `min(CPU count, memory count)`, the memory count assuming four 4-byte
copies of the largest image against 1 GB (32-bit) or 4 GB (64-bit).

### OCR

**Guard:** Guarded by the `TestParityOCR*` family, `TestPlateRulesHaveOneImplementation`
([`tests/parity_test.go`](../tests/parity_test.go)) and, for the plate's CSS, the [shared
appearance](#shared-appearance-ocr-overlay-unit-reader-theme-palette) gate. Individual rows below name
their own test where one exists.

| Contract | Value / rule | Go | JS |
|---|---|---|---|
| Bundled language | `eng` only, provisioned at build time (not committed) | `scripts/build.ps1` -> `<exe>/tessdata/eng.traineddata` | `npm run vendor` -> `vendor/tesseract/lang/` |
| traineddata filename | `<code>.traineddata`, `code` = Tesseract name | [`tessdata.go`](../internal/ocr/tessdata.go) | [`ocr-lang.js`](../extension/src/ocr-lang.js) |
| Plate granularity | one plate per **proximity cluster of confident text lines** (not per paragraph - the engine folds imagery into text paragraphs and splits uniform prose arbitrarily). Flatten the recognition to lines, drop noise (below), then grow a plate while the next line keeps the **line pitch** - top of one line to top of the next - within `OCR_CLUSTER_PITCH_FACTOR (1.2) x` the page's reference pitch and the lines share an x-extent; a bigger step - a figure, a section break, a new column - starts a new plate. The reference is the **median pitch over the image**, taken over successive kept lines that share a column and sit no further apart than `OCR_MAX_LEADING_RATIO (3) x` the median ink height (beyond that it is a section break, not leading); a page that yields no pitch at all falls back to the ink-box gap. The factor multiplies the pitch and **never the height of the recognized ink box** - all-caps lettering boxes far shorter than its own line, and measuring against the ink split one balloon into three plates. Proximity is not the whole test: a line also has to be the **same type size** as the cluster it would join - its ink height within `OCR_TYPE_SIZE_RATIO (1.6)` of the cluster's own median, either way round - because a page with separated regions gives the page-wide pitch estimate steps that belong to no single text, and a headline can then sit closer to the body than the body's own missing lines do. A fourth rule then looks at the page instead of at the neighbours: a finished cluster that covers more than `OCR_MAX_PLATE_COVERAGE (0.52)` of the image **and** whose own line boxes fill less than `OCR_MIN_PLATE_LINE_FILL (0.72)` of its height is **released into one plate per line** - a form, a list or an application window carries one type at one pitch, so nothing in its typography separates its regions, and the whole page arrives as one plate. Released, not refused: every recognized word still reaches a plate | [`tesseract.go`](../internal/ocr/tesseract.go) `clusterLines` / `medianLinePitch` / `sameTypeSize` | [`ocr-cluster.js`](../extension/src/ocr-cluster.js) `clusterLines` / `medianLinePitch` / `sameTypeSize` |
| Line integrity (before clustering) | A recognizer "line" is not always one line: PSM 3's layout analysis can walk across a picture and return a phrase from the left of the page and a phrase from the right as **one line box**. Every grouping rule below then reads them as one text and none can recover - the stitched box genuinely spans both columns, so the column test sees a real overlap, and the coverage release does not fire either because the plate is wide but short. So before anything else, a line is **cut between two consecutive words whose boxes stand more than `OCR_MAX_WORD_GAP_RATIO (3.5) x` the line's median word height apart**, each run boxed to its own words and carrying its own mean confidence. The gap is measured **between the boxes**, not left-to-right, so a right-to-left line is judged the same way round. Cutting alone is not enough: the runs then interleave left, right, left, right down the page, and the clustering closes a plate on the first line that does not belong to it - so the runs of a **page** that was cut are **regrouped into columns** (x-overlap, the clustering's own test) and handed over column by column, top to bottom. The scope is the page and not the paragraph, because the clustering deliberately merges across the paragraph boundaries the engine invents and the engine invents them mid-column. A page nothing was cut on keeps the engine's order untouched | [`tesseract.go`](../internal/ocr/tesseract.go) `(*ocrLine).splitWideGaps` / `lineFromWords` / `orderColumns` | [`ocr-cluster.js`](../extension/src/ocr-cluster.js) `splitWideGaps` / `orderColumns`, [`ocr-overlay.js`](../extension/src/ocr-overlay.js) `collectLines` |
| Plate geometry | percent of natural image size; plate bbox = **union of the cluster's line boxes**; font-size in `cqw` from the cluster's median line height x `0.92` fit factor (the starting size); block-level container `display:block; width:100%; aspect-ratio:W/H; container-type:inline-size; line-height:1.1` with the image at `width:100%; margin:0; max-height:none` on the image role itself, on both editions (a page-level `img` reset must not offset or shrink the overlay image, or the percent-positioned plates drift vertically - up above the image centre, down below it); the container and image CSS come from [`internal/appearance`](../internal/appearance/appearance.json); plates **centre their text** (`align-items:center`) inside their source region (`min-height`) with `overflow:hidden` | [`overlay.go`](../internal/ocr/overlay.go), [`tesseract.go`](../internal/ocr/tesseract.go) | [`ocr-plates.js`](../extension/src/ocr-plates.js) `plateSpecs` / `buildOverlay` |
| Plate runtime re-fit | The compile-time font size is computed from the **source** geometry and cannot know the reflowed - or later translator-swapped - text length, so a fixed size clips a third of plates. After layout each plate's font is shrunk (down to `0.5 x` the starting `cqw`) until the text fits its source-region box; if it still overflows at that floor the box is allowed to grow (`height:auto`) so **nothing is ever clipped**. Re-runs on window resize and whenever a `MutationObserver` sees the page translator swap a plate's text. Degrades safely (CSS `overflow:hidden`) if the script does not run | [`overlay.go`](../internal/ocr/overlay.go) `ocrScript` / `ensureScript` | [`ocr-plates.js`](../extension/src/ocr-plates.js) `fitPlate` / `scheduleFit` |
| Plate colours | adaptive, sampled from the source image (best-effort; falls back to white `#fff` / dark `#111`): background = median colour over the whole block ("paper"); text = **median** of pixels standing out from bg (L1 dist > `90`) within the first line (`1.3 x` line height), else near-black/near-white; contrast floor `55` luma; `0.015`/`6`-px min-ink threshold. Median and not mean on both counts: a glyph's edge is a ramp of antialiased pixels running from the ink to the paper and the deviation test admits most of that ramp, so averaging lands between the two by construction - measured on a caption of rgb(17,17,17) on rgb(253,253,253), mean rgb(61,61,61) against median rgb(7,7,7). **Which of the two is the paper is then decided by the band just outside the block** (`1/3` of a **line height** - not of the 1.3-line ink strip - on each side, floor 2 px, deciding only on `>= RING_MIN_SAMPLES (40)` sampled pixels), and the pair is swapped when that band sits nearer the ink: the median assumes the text is the minority of its own box, which holds for body text in a balloon and fails for heavy display capitals, whose strokes fill more of a tight box than the paper between them - measured, a poster's word came out as cream lettering on a near-black ground, the exact inverse of the poster | [`overlay.go`](../internal/ocr/overlay.go) `blockColors` | [`ocr-overlay.js`](../extension/src/ocr-overlay.js) `blockColors` |
| Noise filter | two gates. **Line confidence:** before clustering, drop a recognized line whose mean word confidence is `< OCR_MIN_LINE_CONF (50)` - real text scores ~80-97, "text" hallucinated from a drawing scores ~0-50, so this keeps plates off imagery and keeps oversized noise boxes from inflating the font. **Text (`isTranslatable`)** on the assembled plate text: drop when `< 5` letters (also kills numbers/symbols); letters but no vowels; the whole text is an address (URL/email/domain/path); or "mishmash" - among letter-bearing tokens, `< 0.5` are word-like (`>= 2` letters + a vowel), needs `>= 3` such tokens. Short CJK (`>= 2` ideographs) is kept | [`tesseract.go`](../internal/ocr/tesseract.go), [`text.go`](../internal/ocr/text.go) `isTranslatable` | [`ocr-cluster.js`](../extension/src/ocr-cluster.js), [`ocr-text.js`](../extension/src/ocr-text.js) `isTranslatable` |
| Pre-OCR resolution handling | Gate on **estimated DPI**, not raw pixel count (a page scan clears 1000 px even at ~100 DPI, so a pixel gate upscales clean renders for nothing or misses the scans that need it). Estimate DPI from the long side over an assumed `OCR_ASSUMED_PAGE_INCHES (11)`-tall page; below `OCR_UPSCALE_DPI_FLOOR (120)` enlarge `OCR_UPSCALE_FACTOR (2 x)` (high-quality) before recognition and divide recognized coordinates back; **always declare the resolution** to Tesseract (`user_defined_dpi`, clamped `>= OCR_MIN_DECLARED_DPI (70)`, doubled when upscaled) so layout analysis separates regions - adjacent balloons - it otherwise merges. Measured: a ~90-DPI newsprint scan gains hugely from the upscale, a ~150-DPI scan only needs the DPI declared (upscaling over-segments it) | [`tesseract.go`](../internal/ocr/tesseract.go) `prepareForOCR` / `estimateDPI` / `scaleDown` | [`ocr-overlay.js`](../extension/src/ocr-overlay.js) `upscaleForOcr` / `estimateDpi` |
| Page-segmentation mode | Tesseract runs in **PSM 3 (AUTO)** on both editions so layout analysis isolates real text regions on an illustrated/scanned page (a speech bubble, a caption) instead of reading the whole frame as one block. The desktop CLI's default is already PSM 3 (made explicit via `--psm`); the extension must set it because tesseract.js defaults to PSM 6 (SINGLE_BLOCK), which folds scene edges into the recognized text (stray punctuation, digits) and mis-merges separate regions into one plate | [`tesseract.go`](../internal/ocr/tesseract.go) `ocrPageSegMode` | [`ocr-overlay.js`](../extension/src/ocr-overlay.js) `OCR_PSM` |
| Grey rescue ladder | An image whose ordinary colour pass returns **no plates at all** is retried on a greyscale copy, first with the engine's own thresholder (`thresholding_method 0`), then with Leptonica's tiled one (`1`), then with the engine's own again but asking for **sparse text (PSM 11)** instead of a page; **every rung runs and the strongest result wins** - more confident words, with a tie keeping the earlier rung - and an image that reads normally never enters the ladder. Tesseract's default thresholder runs Otsu **per RGB channel** and takes ink only where every channel agrees - harmless on flat paper, but on saturated artwork (a brick-red comic panel behind a white balloon, a coloured poster) the channels disagree over the lettering and the mask that reaches recognition holds no text. The second rung then changes *who* thresholds: one global cut-off cannot survive a background that varies across the image (on a sky gradient Otsu splits the gradient itself and a white caption comes out the same value as its ground), while a tiled thresholder decides locally. The third rung changes neither the pixels nor the thresholder but what Tesseract is told to find: a poster is a few large words placed for effect, and the layout analysis PSM 3 runs finds no page in it and drops them. It is a **ladder, not a replacement** - the colour pass wins where lettering is separated by hue rather than brightness, so retrying only after an empty result leaves every image that works today unchanged | [`tesseract.go`](../internal/ocr/tesseract.go) `greyRescuePasses` / `greyRescue` / `greyRendition` / `strictlyBetter` | [`ocr-overlay.js`](../extension/src/ocr-overlay.js) `GREY_RESCUE_PASSES` / `greyRescue` / `greyRendition` / `strictlyBetter` |

**OCR download version and catalog are aligned** (2026-07-01 parity pass) and guarded by
`tests/parity_test.go`:

- **tessdata version = 4.0.0** on both sides. Go pins GitHub `tessdata_fast/raw/4.0.0` (plain,
  [`tessdata.go`](../internal/ocr/tessdata.go)); the extension loads
  `tessdata.projectnaptha.com/4.0.0_fast` (gzip, [`ocr-lang.js`](../extension/src/ocr-lang.js)) and
  bundles eng from the same 4.0.0 ([`build.mjs`](../extension/build.mjs)). Different host/format,
  identical upstream bytes -> matching recognition.
- **Download integrity is desktop-only (intentional).** The desktop app accepts only catalogue codes
  (`ocr.CheckLang`, called by `-ocr-download`, the GUI's `/api/ocr-download` and `ocr.Download`
  itself) and installs a pack only when its size and SHA-256 match the table pinned in
  [`download.go`](../internal/ocr/download.go) `packDigests` (plain 4.0.0 files), through a unique
  temp file and a rename, into the per-user folder (`os.UserCacheDir()/doc-html-translate/tessdata`,
  looked up before `<exe>/tessdata`). The extension cannot share that table: tesseract.js fetches the
  gzipped build from projectnaptha itself and caches it in IndexedDB, so the bytes it receives are
  neither the plain files nor seen by extension code. Its catalogue gate is the `LANGS` list the
  picker is built from.
- **Language catalog = 13** on both sides (`eng rus ukr jpn jpn_vert deu fra spa ita por pol chi_sim
  kor`): `tessdata.go` `Available` == `ocr-lang.js` `LANGS`.
- **Overlay grouping constants** identical: `OCR_MIN_LINE_CONF = 50`, `OCR_CLUSTER_PITCH_FACTOR = 1.2`,
  `OCR_MAX_LEADING_RATIO = 3`, `OCR_TYPE_SIZE_RATIO = 1.6`, `OCR_MAX_PLATE_COVERAGE = 0.52` and
  `OCR_MIN_PLATE_LINE_FILL = 0.72` (`tesseract.go` `ocrMinLineConf` /
  `ocrClusterPitchFactor` / `ocrMaxLeadingRatio` / `ocrTypeSizeRatio` / `ocrMaxPlateCoverage` /
  `ocrMinPlateLineFill` ==
  [`ocr-cluster.js`](../extension/src/ocr-cluster.js)). The **quantity** each factor multiplies is part
  of the invariant, not only its value: equal numbers over different quantities still group all-caps
  lettering differently, so `TestParityOCRClustering` also pins the expression that computes the bound
  (`pitchMax` from the reference pitch), the one that measures a pitch (`y0` to `y0`), that the size
  break is weighed against the **cluster's** median height rather than the page's, and that it is
  symmetric (the ratio multiplies the smaller of the two heights) - on both sides. **What the ratio
  measures is the median of a line's own word heights, not its line box** (`tesseract.go`
  `(*ocrLine).inkHeight` == `ocr-cluster.js` `lineInkHeight`, pinned by the same test). The box is the
  union of the words, so one tall artefact sets it for the whole line: measured in the extension
  edition on `synth-adjacent-balloons` (2026-08-15), the balloon's left outline is recognized as `|`
  and the line `| NOT EVEN` boxes 37 px beside 13 px for `SLIGHTLY.`, a 2.85x step the ratio reads as
  two type sizes - so one balloon became two plates and the taller one reached onto the protected
  outline. A line whose engine reports no word boxes falls back to the line box on both sides, so the
  rule never becomes looser through a quantity it does not have. **The same artefact is also kept out
  of the line's own box** (`trimOutlierWords`, both sides, pinned by the same test): reading the type
  size correctly regroups the balloon but the box is still the union of its words, so the plate drawn
  from it kept reaching onto the protected outline - measured at 148 px of damage before the type-size
  fix and 160 px after it, because the plate then spanned both lines. A word is dropped from the box
  only when it carries no letter or digit **and** is taller than `OCR_TYPE_SIZE_RATIO` times the
  line's median word height; either condition alone would delete real words or real punctuation, and a
  line made of nothing else keeps its box. The type-size ratio
  is bracketed by two measurements on the corpus's hand-drawn line boxes: the widest spread a single
  text shows on its own is 1.42x (`samson-and-delilah-03-scroll`, 19 lines of one caption) and the
  narrowest step between two texts a reader separates is 1.86x (`poster-display-type-on-flat-colour`,
  headline over body). Moving it needs a new measurement of both, not a scene that would like it moved.
- **The plate-coverage release** is the fourth grouping rule and the only one that looks at the page
  rather than at a line's neighbours. Pitch and type size cannot separate a form, a list or an
  application window - one type, one column, an even pitch - so the whole page arrives as one plate.
  A finished cluster is **released into one plate per line** when it is *both* too big and too loose:
  its box covers more than `OCR_MAX_PLATE_COVERAGE (0.52)` of the image **and** its own line boxes
  account for less than `OCR_MIN_PLATE_LINE_FILL (0.72)` of the box's height. `tesseract.go`
  `releaseOversized` == `ocr-cluster.js` `releaseOversized`, and `TestParityOCRClustering` pins both
  conditions on both sides, because either alone releases a scene the corpus says is one plate.
  Releasing and not refusing is part of the invariant: every recognized word still reaches a plate,
  and the released lines skip `isTranslatable` (the assembled text already passed it). Both bounds are
  bracketed from opposite directions in
  [`DEV/research/ocr_plate_coverage_2026-08-13.md`](../DEV/research/ocr_plate_coverage_2026-08-13.md);
  the *area* version of the fill was measured on the same run and separates nothing, which is why the
  rule is stated on the vertical axis.
- **The word-gap split** is the only rule that runs *before* the clustering, because it repairs the
  clustering's input rather than its output. `OCR_MAX_WORD_GAP_RATIO = 3.5` on both sides
  (`tesseract.go` `ocrMaxWordGapRatio` == [`ocr-cluster.js`](../extension/src/ocr-cluster.js)), and
  `TestParityOCRClustering` pins three expressions besides the number, because equal constants over
  different quantities would again say nothing: the gap is measured **between the two boxes**
  (`max(w.x0-prev.x1, prev.x0-w.x1)`) so a right-to-left line is not silently exempt; it is weighed
  against the median of the line's own **word** heights, not the line box, for the same reason
  `inkHeight` is; and each run is boxed to **its own words** (`lineFromWords` / `unionOf`), because
  handing both halves the stitch's box back leaves the clustering exactly where it started. The
  companion rule `orderColumns` is pinned on both sides too - without it the split trades one
  oversized plate for three fragments. The ratio is bracketed over the 46 lab scenes plus
  `test_doc/1.png`, on the 199 multi-word lines that clear the confidence floor
  ([`DEV/research/ocr_word_gap_2026-09-12.md`](../DEV/research/ocr_word_gap_2026-09-12.md)): the
  widest gap inside a line that really is one line is 2.57x, the narrowest cross-region stitch above
  it is 4.80x, and 3.5 is the geometric middle. The band 1.87-2.57x **overlaps** - comic balloons drawn
  side by side stitch there while real lines reach into it - and is deliberately left alone, because
  no ratio separates it; that case needs evidence from the pixels between the two words.
- **Overlay resolution constants** identical: `OCR_UPSCALE_DPI_FLOOR = 120`, `OCR_ASSUMED_PAGE_INCHES = 11`,
  `OCR_MIN_DECLARED_DPI = 70` and `OCR_UPSCALE_FACTOR = 2` (`tesseract.go` `ocrUpscaleDPIFloor` /
  `ocrAssumedPageInches` / `ocrMinDeclaredDPI` / `ocrUpscaleFactor` == `ocr-overlay.js`). Guarded by
  `TestParityOCRClustering`.
- **Page-segmentation mode** identical: PSM `3` (AUTO). `tesseract.go` `ocrPageSegMode` (passed to the
  CLI as `--psm 3`) == `ocr-overlay.js` `OCR_PSM` (applied via `setParameters({tessedit_pageseg_mode})`).
  The extension must set it explicitly because tesseract.js defaults to PSM 6 (SINGLE_BLOCK); the
  desktop CLI's own default is already 3.
- **Grey rescue ladder** identical: the retry order for an image the colour pass could not read is
  `[engine-default thresholder (0) at PSM 3, Leptonica tiled Otsu (1) at PSM 3, engine-default at
  PSM 11]` over a greyscale copy - `tesseract.go` `greyRescuePasses` == `ocr-overlay.js`
  `GREY_RESCUE_PASSES`, and `thresholdEngineDefault` / `thresholdLeptonicaOtsu` /
  `ocrSparsePageSegMode` == `THRESHOLD_ENGINE_DEFAULT` / `THRESHOLD_LEPTONICA_OTSU` /
  `OCR_SPARSE_PSM` (guarded by `TestParityOCRGreyRescue`).
- **Which rung wins** identical: every rung runs and the strongest result is kept, where strength is
  the number of words the rung placed and a tie keeps the earlier rung - `tesseract.go`
  `resultStrength` / `strictlyBetter` == `ocr-cluster.js` `resultStrength` / `strictlyBetter`
  (guarded by `TestParityOCRRungComparator`). It replaces a first-non-empty-wins rule that let a rung
  recovering one word end the search before a later rung could recover six.
- **The rescue floor was re-measured and did not move** - `ocrRescueLineConf` **80** ==
  `OCR_RESCUE_LINE_CONF`, guarded by `TestParityOCRGreyRescue`. It is recorded here because the
  re-measurement is the reason the number is now trustworthy rather than inherited, and because it
  says what the next attempt must not repeat. Over the 46 lab scenes and the 13 annotated ones
  (2026-08-15,
  [`DEV/research/ocr_rescue_floor_2026-08-15.md`](../DEV/research/ocr_rescue_floor_2026-08-15.md)),
  genuine rescued lettering runs 32.8-69.2 and invented lettering 8.4-73.9 - they **overlap**, so no
  single floor separates them and moving the number only trades one scene's loss for another's. The
  rule the distribution did support - a lower gate for a line carrying a run of four letters, at the
  middle of the empty band those lines bracket (36.1 `allie` / 58.3 `KPECTbAHHH!`) - was
  implemented in both editions, run over the corpus and **rejected by the corpus**: under the app's
  default `eng` a Cyrillic poster then gets a 782x310 px plate of transliterated debris
  (`TPAXATBCR: 4 y`) over its own lettering where it previously got none, which is the regression
  the floor exists to prevent. Both editions are back at 80 and the gap is left open.
- **The confidence floor keeps a record of what it rejected** - `tesseract.go` `keepLine` +
  `Result.Dropped` == `ocr-cluster.js` `keepLine` + `droppedLines`, guarded by
  `TestParityOCRDroppedLines`. The floor is the one place the overlay decides against words the
  recognizer *did* read, and until 2026-08-15 it was silent: a scene where the poster's first word
  came back correctly at 69.2 and was thrown away at a floor of 80 looked, in every output either
  edition produced, exactly like a scene where nothing was recognized. Three things are part of the
  invariant, not decoration:
  - **one predicate.** Both editions ask `keepLine` from `clusterLines` *and* from the record, so
    the record cannot describe a decision that is no longer taken. Two copies of `conf >= floor`
    would pass every constant check and drift the first time either moved.
  - **diagnostics only.** Nothing renders it and `resultStrength` does not weigh it - a rung's
    strength is still the words it *placed* - so the record cannot change which rung a reader sees.
  - **it travels with the rung that won**, not merged across rungs, because a merged set describes
    no single decision.

  Both editions write it as one JSON line per recognized image into an `ocr-diag.jsonl` in the same
  shape - `file`, `width`, `height`, `blocks`, `dropped` - and **also for an image that produced no
  plates at all**, the case OCR-OVERLAY rule 12 says the record exists for. `blocks` and `dropped`
  are always arrays, never omitted: both empty is "read fine, found no text", no blocks with a
  non-empty `dropped` is "everything was thrown away". The desktop writes it from `applyOverlays`
  into the `DOCHT_OCR_DIAG` sidecar (`diag.go`, guarded by `TestDiagnosticsRecordDiscardsForNoPlateImage`);
  until 2026-09-25 the no-plate arm kept the record as far as `applyOverlays` and then wrote nothing.
  The extension has no environment variable to hang a sidecar on, so `overlayImage` leaves the
  recognizer's record on the overlay container as a JS property (`ocrRecord`, never an attribute -
  it cannot reach the rendered DOM) and the lab harness `extension/scripts/ocrlab.mjs` writes it
  through `makeDiagRecord`, whose output is pinned byte-for-byte to the desktop line by
  `test/ocrlab-evidence.test.mjs`. **Intentional difference:** the desktop line also carries each
  block's rendered `style` and sampled colours; the browser resolves those at layout time, and the
  evidence plates already record them as laid out.
- **Plate colour orientation** identical: the band just outside a block decides which sampled colour
  is the paper, over `RING_MIN_SAMPLES (40)` pixels - `overlay.go` `ringNearerInk` / `ringMinSamples`
  == `ocr-overlay.js` `ringNearerInk` / `RING_MIN_SAMPLES` (guarded by
  `TestParityOCRPlateColourOrientation`). **Intentional implementation difference:** the desktop app writes an
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
- **The opaque paper is on the plate box** - identical on both sides, and this is a decision taken
  against the corpus rather than a default. The box carries the sampled paper (the plate role's
  `background:#fff` in `internal/appearance/appearance.json`, overridden per plate by the sampled
  colour in the inline style) and the text sits directly in it; there is no inner span, and
  `TestParityOCRFontFit` fails if the string `ocr-ink` reappears in either edition. The box is what
  covers the source region, so the box is what has to be opaque.
  **Both carriers were shipped and both were measured.** Carrying the paper on an inline span around
  the string (`box-decoration-break:clone`, painted once per *rendered* line) gives it the shape of
  the words, which is why it was tried: a block box is wider than centred copy on its last line, and
  the rectangle put 91 px of paper over the photograph on either side of a 984 px caption's 759 px
  last line. But over the 46 lab scenes the string carrier left a mean **93%** of the source
  lettering still showing under a plate against **17%** for the box - a plate that conceals almost
  nothing it covers, and a regression against the lab's recorded 0.28 bound - so it lasted one day
  (2026-08-13) and the box carrier is back. Sizing paper from the **source** line boxes remains
  rejected for its own measured reason: the rendered string wraps where the source did not, and the
  page translator changes its length again, so paper cut to the source lines comes apart from the
  words on it. The box's over-cover is bounded from the other side now, by the plate-coverage rule
  and the type-size rule in the grouping row above. Evidence:
  [`DEV/research/ocr_plate_coverage_2026-08-13.md`](../DEV/research/ocr_plate_coverage_2026-08-13.md).
- **The plate prints its own colours** - `print-color-adjust:exact` (plus the `-webkit-` prefix
  Chromium still reads) on the plate role in `internal/appearance/appearance.json`, which both
  editions derive from; `TestParityOCRPrintPlate` pins that the source keeps it. Scoped
  to the plate, so the rest of the document keeps the browser's print economy. **What it recovers
  was measured rather than assumed, and the measurement corrected the expectation.** Printing an
  overlaid page with "Background graphics" unchecked - the default - does not leave the plate
  transparent over legible source lettering, which is what
  [`DEV/plan/2026-08-12_ocr-exchange-followups.md`](../DEV/plan/done/2026-08-12_ocr-exchange-followups.md)
  item 2 predicted: Chromium repaints the plate **white** and darkens its text to keep contrast
  against it. So the printed page stays readable and stops matching the artwork - every sampled
  balloon, panel and paper tone becomes a stark white patch. Measured 2026-08-15 on
  `img-png_Nyoka-comic-page` through `Page.printToPDF(printBackground:false)`, the same code path as
  the print dialog, by diffing the PDF's colour operators: **20 of 20 plate papers forced to
  `1 1 1`** and 14 ink colours darkened without the declaration, **0 forced** with it, out of 59
  colour operators on the page. The extension edition, measured with the same instrument on a
  harness carrying the shipped `ocr-overlay.css`: 3 of 3 papers forced, 0 with it. Chrome's
  `--print-to-pdf` command line cannot see this difference at all - it never prints backgrounds and
  ignores the opt-in - so a check for this has to go through CDP.
- **Plate ink is a median**, not a mean, of the pixels that stand out from the sampled paper - the
  deviation test admits a glyph's antialiased edge, and averaging that ramp lands between the ink
  and the paper (measured rgb(61,61,61) for source lettering of rgb(17,17,17)). Same guard.
- **Runtime fit runs both ways**: the cqw font shrinks to a floor of 50% and grows to a ceiling of
  `1.15x` the compile-time size (`overlay.go` `ocrScript` `cap=base*1.15` == `ocr-plates.js`
  `FONT_GROW_CAP`), stopping one step before the content overflows. The ceiling is the guard against
  "fill the box": a block box includes the leading between its lines, so filling it would print the
  translation larger than the words it covers.
- **Plate font-fit factor** identical: `0.92` - `overlay.go` `fontFitFactor` == `ocr-plates.js`
  `FONT_FIT` (guarded by `TestParityOCRFontFit`). Plate font-size = median line height x this factor;
  below `1.0` so translated text (often longer) has room before it overflows the block.

- **Display space is the only coordinate space** - the shared invariant behind every percentage
  above. Boxes are in the pixels of the picture *as a reader sees it*, which is the file's stored
  pixels turned by its EXIF orientation, and the image must fill its overlay container exactly, or
  percentages of the container stop being percentages of the image. Each edition reaches it by its
  own route and neither has a constant to share:
  - the desktop app applies the orientation itself, to the copy tesseract reads
    ([`internal/ocr/exif.go`](../internal/ocr/exif.go), used by `stageForOCR`), and turns the
    decoded image it samples plate colours from with it. EXIF parsing in Go is JPEG-only (PNG `eXIf`
    and WebP can carry orientation tags, but a rotated camera photo is a JPEG in practice, and an
    untested parser is worse than a documented gap); the extension relies on `createImageBitmap`
    which handles all browser-supported image formats with EXIF. Without this, a portrait phone shot is
    recognized on its side - no OSD runs in either PSM this app uses - and whatever does read lands
    in a space the plates are not in;
  - the extension reaches it through the decoder: every `createImageBitmap` in `ocr-overlay.js` is
    passed `BITMAP_OPTS = { imageOrientation: "from-image" }`, and the plates go over the same
    `<img>`, so both sides of the comparison are in display space. The option is **named rather than
    inherited**: `createImageBitmap`'s own default moved from `"none"` to `"from-image"` while the
    spec settled, so an unnamed call makes the agreement hold for as long as the browser default
    does and no longer. `TestParityOCRExifOrientation` pins the constant *and* that no bare
    `createImageBitmap` call is left - the recognizer's bitmap, the colour sample and the grey rungs
    must all be in the same space, and one bare call is enough to break that.

  The container half of the invariant is a desktop-only hazard, because only the desktop page ships
  other scripts: the navbar's image-aspect guard used to write an inline `width` on every image,
  which beats `.ocr-fig>img{width:100%}` and left the picture at its natural width inside a
  column-width container. Measured on a 640 px scene in a 1216 px column, every plate rendered at
  1.9x its size and off its text - while drifting 0 px between viewports, because it was equally
  wrong at all of them. The guard now skips images inside `.ocr-fig`
  (`internal/htmlgen/navbar.go`, guarded by `TestImageAspectGuardSkipsOCROverlay`).

- **Vertical lettering is catalogued, not supported by design** - `jpn_vert` is in the language
  catalog on both sides (`tessdata.go` == `ocr-lang.js`), so a user can choose the vertical-text model
  and recognition works. The plate itself is a horizontal flex container and clustering assumes vertical
  pitch between horizontally overlapping lines (whereas vertical writing places lines side by side).
  Vertical layout is unsupported in both editions.

- **RTL and CJK handling** - short CJK ideographs bypass the alphabetic minimum-length and vowel rules
  in `isTranslatable` (`text.go` == `ocr-text.js`). Plates inherit the document's DOM `dir` without
  edition-specific RTL word re-sorting. Geometry robustness under RTL replacement is verified by the
  `rtl-arabic` translation-stress case (1.8x length in `tools/ocrlab/runner/stress.go` == `STRESS_CASES`
  in `ocrlab.mjs`), ensuring no clipping or drift occurs.

- **Positioning acceptance gates absolute IoU floor, not drift alone** - `DEV/ocrlab/thresholds.json`
  and `tools/ocrlab/report/gate.go` gate the `position` dimension on mean IoU against ground truth
  (overall floor `0.77`, category floors for comic `0.75` and texture `0.74`) rather than on drift alone.
  A systematic offset (such as a mis-sized container) drifts 0 px across viewports because it is equally
  wrong at all of them; the IoU floor catches it.

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

- **OCR layer toggle** present in both, with the same three rules: the control is **revealed only on a
  document that carries plates** (a text book must not show a control that would do nothing); the
  choice is **global and persisted**, like the theme, because a reader who wants to see the artwork
  wants it for the whole book; and hiding is **`display:none`, never `visibility`/`opacity`** - a
  hidden plate must not keep taking pointer events, must not be read out by a screen reader, and must
  not offer the page translator text to swap inside a layer the reader turned off. The plates cover
  the source lettering, which is the point when translating and in the way when reading the art.
  Per-edition wiring (the class names differ, see below, and so does the storage):
  `internal/htmlgen/navbar.go` `#dht-ocr-toggle` -> `html.dht-ocr-off` + localStorage `dht_ocr`;
  `extension/src/viewer.js` `#btn-ocr` -> `html.ocr-layer-off` + `viewerPrefs.ocrLayer`.

**Still divergent (tracked in the parity ticket):**

- **CSS class names** differ: Go `.ocr-fig` / `.ocr-box`; JS `.ocr-overlay` / `.ocr-plate` /
  `.ocr-overlay-img` / `.ocr-badge`. Cosmetic; deferred. The toggle's off-state class is part of this
  split (`dht-ocr-off` / `ocr-layer-off`) and moves with it if the names are ever unified. Named in
  the `divergences` list of `internal/appearance/appearance.json`; the appearance gate is blind to
  names and compares only the declarations.
- **Default OCR language rule**: Go derives from `-src` (`TessLang`, else `eng`); the extension uses a
  fixed persisted `eng` (it has no translation source language). Intentional for now.
- **The script check that corrects an unchosen language is desktop-only.** When `-ocr-lang` is empty
  the Go app puts the book's first image through Tesseract's `--psm 0` script pass and lets the answer
  correct the default - adding a language for the detected script where its data is installed
  (`rus+eng`), and otherwise producing **no plates** plus a line naming the script and the download
  ([`internal/ocr/script.go`](../internal/ocr/script.go)). The extension has no port, for two reasons
  that are about the edition rather than about the rule: its OCR language is an explicit, persisted
  choice in the popup rather than a value inferred from a translation flag, and the pass needs
  `osd.traineddata`, ~10 MB the extension neither vendors nor downloads today. The consequence is real
  and is not pretended away - a reader who never opens the popup gets the same transliterated debris on
  a Cyrillic page that the desktop no longer produces. Tracked in the parity ticket.

### Settings defaults

**Guard:** Prose only for the default values and the extension side. `TestParityGUIExposesEveryCLIFlag`
([`tests/ui_cli_parity_test.go`](../tests/ui_cli_parity_test.go)) guards only that the GUI exposes every
CLI flag; a future ticket would pin the shared defaults (`-src`/`-dst`, OCR language, `enabledByDefault`)
across `flags.go`, `ui.html` and `defaults.js`.

Canonical defaults (from [`flags.go`](../internal/config/flags.go)): `-split 5000`, `-toc-depth 0`
(unlimited), `-src en`, `-dst ru`, `-max-cost 0` (no limit), `-ocr false`, `-ocr-lang ""` (falls back
to `-src`, else `eng`), `-ollama-model gemma3:12b`, `-ollama-parallel 1`, `-ollama-ctx 8192`.

Invariant: **the GUI must expose every CLI flag** (see [`ui-cli-parity`](../CLAUDE.md) memory). Known
default mismatches (GUI split=0, extension source-lang=auto, extension OCR-lang fixed `eng`) are tracked
in the parity ticket.

The GUI always forwards `-split` (its default 0 differs from the CLI's 5000, so leaving it out would
turn splitting on), forwards the Ollama fields only with `-ollama` and `-max-cost` only with `-google`,
and treats an empty or malformed box as its own default. `TestAssembledArgsSurviveGarbageFields`
([`hardening_test.go`](../cmd/doc-html-ui/hardening_test.go)) feeds such values through the real CLI parser.

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

**Guard:** Prose only. Spot-checked consistent on 2026-08-15 across the 13 splash files, the GUI and the
three extension pages; a future ticket would pin both strings in every file the table lists.

Every edition surfaces the same product page and the same feedback address. Both are duplicated string
constants - changing either is a cross-edition change; update all rows together.

| Element | Value | Sources |
|---|---|---|
| Product site | `https://serzhyale.github.io/doc-html-translate/` | CLI splash [`app.go`](../internal/app/app.go) (`internal/app/splash/*.txt`), navbar [`projectURL`](../internal/htmlgen/navbar.go#L17), GUI [`ui.html`](../cmd/doc-html-ui/ui.html) byline, extension [`popup.html`](../extension/src/popup.html) / [`viewer.html`](../extension/src/viewer.html) / [`options.html`](../extension/src/options.html) |
| Feedback | `mailto:sza@ukr.net` | CLI splash [`app.go`](../internal/app/app.go), GUI [`ui.html`](../cmd/doc-html-ui/ui.html) byline, extension [`popup.html`](../extension/src/popup.html) / [`viewer.html`](../extension/src/viewer.html) / [`options.html`](../extension/src/options.html) |

### Report field labels

**Guard:** Guarded by `TestParityReportFields`.

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

**Guard:** Guarded by `TestParityOCRLabEvidenceSchema` and `test/ocrlab-evidence.test.mjs`.

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

**Guard:** Prose only for the code list, its order and the RTL/font sets - nothing compares `i18n.Codes`
with the extension's `LOCALES` and `_locales/`; a future ticket would. The chrome-only rule is guarded by
`TestConvertedChromeLanguage` (desktop side).

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
  in-memory DOM** and therefore sanitizes (drops `script/style/inline styles/on*` and `name`, keeps only
  `http`/`https`/`mailto`/`tel`/relative/fragment links), rewrites `<img>` to `blob:` URLs, and
  namespaces ids/anchors. So sanitize, image-blobbing and anchor remapping exist **only in the
  extension** by design. The shared URL rules live in [`url-policy.js`](../extension/src/url-policy.js).
- **Remote content is opt-in in the extension only** (2026-09-25). The viewer parks every remote
  `src`/`srcset`/`poster`/`background`/SVG `href` a document carries until the reader allows it - per
  document from a notice, or always (`allowRemoteContent`, default off, in [`defaults.js`](../extension/src/defaults.js)).
  Opening a document must not tell its author, or an embedded tracker, that it was opened, and the
  viewer's fetches carry the extension's host access. The desktop app writes a local file the reader's
  own browser opens under its own rules and fetches nothing itself, so it has no such setting. The
  extension's "save as HTML" file also carries its own script-free content policy
  ([`export-html.js`](../extension/src/export-html.js)), because the file leaves the extension's policy
  behind; the desktop output's own `javascript:` question is ticket `bugfix-reader-layer-and-single-page`.
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
- **Whole-page OCR on a live web page is extension-only, and there must not be a desktop
  counterpart.** The reader right-clicks an ordinary page - a webcomic, a scanned archive, a gallery -
  and every picture on it is recognized in place, the plates drawn over the art in the page the reader
  is already on, so the words can be selected, found and translated by the browser along with the rest
  of the page ([`page-ocr.js`](../extension/src/page-ocr.js) broker, [`page-agent.js`](../extension/src/page-agent.js)
  in the page, [`ocr-host.js`](../extension/src/ocr-host.js) engine host). The desktop app converts
  documents it is handed; it has no live page to sit inside, so this is not a gap to close but the
  mirror of the Go-only divergences above. What the two editions **do** share is the plate unit:
  recognition, clustering and plate geometry are the same code the viewer uses
  ([`ocr-plates.js`](../extension/src/ocr-plates.js)), pinned by `TestPlateRulesHaveOneImplementation`,
  so the page agent can never grow a second set of plate rules. Decided in
  [`DEV/plan/done/2026-09-19_page-ocr-overlay.md`](../DEV/plan/done/2026-09-19_page-ocr-overlay.md) (ADR-3).
- **Where the recognition engine runs, in the extension, is decided at runtime.** The engine never
  runs inside the reader's document - a third-party page's content security policy governs what
  compiles there, and a great many sites would refuse the WebAssembly module silently. It runs in an
  extension-owned document instead: the browser's offscreen document where that exists, and an
  extension-origin frame parked inside the page where it does not. The fallback exists because the
  offscreen API is newer than the extension's declared `minimum_chrome_version`, and raising that
  minimum would shrink reach - which a release here never does. Guarded by
  `TestPageOcrKeepsTheDeclaredMinimumBrowser`. The Go app has no counterpart: it runs `tesseract`
  as a process.
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
- **Reading a local file is a permission in the browser and nothing at all on the desktop.** The desktop
  editions open a path the user handed them; the extension may not read a `file:` URL until the user turns
  on "Allow access to file URLs", which is off on a store install. So the extension carries two guards the
  Go side has no counterpart for and must not grow one: the viewer's `vLoadFailFile` notice for documents,
  and the image-OCR page's pre-check
  ([`ocr.js`](../extension/src/ocr.js), `fileAccessAllowed` -> `ocrFileBlocked` plus a button onto
  `chrome://extensions/?id=<id>`). A UNC `file://server/share` image is refused outright on both of those
  paths - file-scheme match patterns only cover empty-host URLs, so no toggle can ever grant it. The
  browser context menu is likewise extension-only, including the rule that a title's `&` is a keyboard
  mnemonic and has to be doubled to print ([`background.js`](../extension/src/background.js), `menuTitle`).

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
4. **Prefer a single source of truth in code** when practical over manual copy + a "values match"
   comment. The shared appearance is the worked case: `internal/appearance/appearance.json`, a builder
   per edition, and a gate that compares every declaration rather than a chosen few (see
   [Shared appearance](#shared-appearance-ocr-overlay-unit-reader-theme-palette)).
5. **Guard tests** parse both codebases and fail on drift: `tests/appearance_parity_test.go` (the OCR
   overlay CSS and the theme palette, against their single source), `tests/parity_test.go` (OCR
   tessdata version, OCR language catalog, PDF reflow constants) and `tests/ui_cli_parity_test.go` (the
   GUI exposes every CLI flag). They run in the normal `scripts/test.ps1` gate - extend them whenever you
   pin a new invariant here. These guard the **value** invariants. The extension's own DOM path (chapter
   sanitize / image / link / TOC) is covered by `extension/test/epub-dom.test.mjs` +
   `extension/test/sanitize.test.mjs` under `npm test` (a dev-only `linkedom` DOM, never bundled) - these
   assert the JS-side behaviour that mirrors `internal/epub`, complementing the value guards above.
6. **Structural drift-check (advisory):** `scripts/parity-check.ps1` (alias `a pc`, and run in the
   `scripts/check.ps1` gate) reads the pairs from `configs/parity-map.json` and *warns* when a Go extractor
   changed without its paired JS module, or vice versa. It never blocks - drift is an advisory (exit 3, named
   on the gate's verdict line), unless `-Strict` makes it a failure. Touching this file (docs/PARITY.md) in
   the same change set silences it, so the escape hatch for an intentional one-sided change is to record it
   here under [Intentional divergences](#intentional-divergences-do-not-fix).
   `tests/parity_map_test.go` fails when `configs/parity-map.json` and the port map table at the top of this
   file disagree, so a row added here must be watched there (or excused under `notWatched` with a reason).
