# Format verification sweep - protocol and findings

**Instrument, not a ticket.** This file plans and records a verification pass over the whole test
corpus, for both editions. It fixes nothing. Every confirmed problem leaves here as either a spec
ticket under [`DEV/plan/`](../plan/) or an automated-test scenario (see [Outputs](#outputs)).

Corpus and its measured numbers: [`test_doc/CORPUS.md`](../../test_doc/CORPUS.md) (gitignored;
62 files, 807 MB). Re-measure with `test_doc/_fetch/verify-corpus.ps1`.

## Owner decisions (2026-07-17)

1. **Order is by real-world popularity**, not by how the code is organised - PDF first, roadmap
   formats last.
2. **Both editions.** Desktop EXE (CLI + GUI) and the browser extension. **Chrome only** - Edge,
   Firefox and Safari are out of scope for this pass.
3. **Translation:** the signature free flow (convert -> Chrome's built-in Translate page) is checked
   everywhere; `-ollama` is checked on a model that is actually installed. **Google API is out of
   scope** (paid).
4. **No stress tier.** `Aphrodite's Mirror (1).pdf` (388 MB, 2304 pages, no text layer - a serial
   OCR run of roughly half an hour) is **excluded**. Accepted consequence: the OCR-pool defect
   ([`2026-07-17_ocr-pool-per-book`](../plan/2026-07-17_ocr-pool-per-book.md)) leaves this sweep
   without a headline number. Everything else in the corpus is in scope.
5. **Extension is driven by hand** this pass, with screenshots against a checklist. A Playwright
   harness is a *recommendation to spec*, not work to do mid-audit - the audit must not mutate the
   repo's dependencies.
6. **Deliverables split:** protocol + raw findings here; each confirmed defect becomes its own
   ticket in `DEV/plan/`.

## The five dimensions

Every wave answers the same five questions. They are ordered by how expensive the answer is to get
wrong - a false claim (D1) outlives any single bug.

| | Dimension | Question | Verdict is |
|---|---|---|---|
| **D1** | Declared vs actual | Does the product do what every published surface says it does - and does it admit what it cannot do? | Per claim: `true` / `false` / `unstated` |
| **D2** | It works | Does the fixture convert and open? | Per fixture: `pass` / `fail` / `silently-wrong` |
| **D3** | Speed | How long, and is that defensible for the input? | Wall-clock, s/page, MB/s |
| **D4** | Quality | Is the result *right* - readable, complete, navigable, nothing broken? | Screenshot + checklist |
| **D5** | Documentation | Is every surface complete, current and consistent across en/ru/uk? | Matrix cell: `ok` / `stale` / `missing` |

`silently-wrong` in D2 is a first-class verdict, not a footnote. The pre-flight already found the
shape: a `.cbz` converts with **exit code 0**, in 62 s, into 1169 HTML files totalling 24 MB of
binary-read-as-text - under a tidy "Page 1" heading. Exit code alone would have called that a pass.
Any check that only reads exit codes is not measuring this product.

### D1 - the claim sources

D1 is only as good as its list of surfaces. These are the places the product makes a promise. A
claim found in any of them is in scope; **verified paths, not assumed ones**:

| Surface | File(s) |
|---|---|
| Repo readme | [`README.md`](../../README.md) |
| Site - hero | [`index.html`](../../index.html) |
| Site - docs, 3 languages | [`docs.html`](../../docs.html), [`docs.ru.html`](../../docs.ru.html), [`docs.uk.html`](../../docs.uk.html) |
| Site - extension page | [`extension.html`](../../extension.html) |
| Site - privacy | [`privacy.html`](../../privacy.html), [`extension-privacy.html`](../../extension-privacy.html) |
| winget listing | [`winget/SerZhyAle.DocHtmlTranslate.locale.en-US.yaml`](../../winget/SerZhyAle.DocHtmlTranslate.locale.en-US.yaml) |
| MSIX / Store | [`msix/AppxManifest.xml`](../../msix/AppxManifest.xml) |
| Chrome / Edge listing | [`extension/store/LISTING.md`](../../extension/store/LISTING.md), [`extension/store/PRIVACY.md`](../../extension/store/PRIVACY.md) |
| Extension manifest | [`extension/manifest.json`](../../extension/manifest.json) (`description`) |
| Extension file picker | [`extension/src/viewer.html`](../../extension/src/viewer.html) (the `accept` list *is* a claim) |
| GUI | [`cmd/doc-html-ui/ui.html`](../../cmd/doc-html-ui/ui.html) |
| CLI | `doc-html-translate.exe -h` |

D1 runs **in both directions**, and the second is the one that gets skipped:

- **Claimed -> real?** Every format/feature named on a surface must actually work.
- **Real -> claimed?** Every format the code accepts must be named. An accepted-but-unlisted format
  is a discoverability bug, and the pre-flight already found one: `internal/img` accepts `.tif` and
  `.tiff`, the extension's picker accepts neither.
- **Cannot -> admitted?** Runtime dependencies and refusals must be stated where a *buyer* reads,
  not only in the repo. MOBI/AZW3 needs Calibre; CBR/CB7 will need 7-Zip.

## Existing test surface

**Read this before proposing any test.** The repo is not a blank slate, and the sweep's job is to
find what these already-existing checks let through - not to re-invent them.

| Where | What it covers |
|---|---|
| [`tests/testdoc_test.go`](../../tests/testdoc_test.go) | **Corpus-driven.** Runs every supported file in `test_doc/` through the real pipeline (`NoTranslate`, `NoOpen`), one subtest per file, into a temp dir. Skips cleanly when the folder is absent (CI's normal state). Override the location with `DOC_HTML_TEST_DOC` |
| [`tests/parity_test.go`](../../tests/parity_test.go) | Go<->JS invariants: theme palette, OCR version / catalog / clustering / font-fit, reflow constants |
| [`tests/ui_cli_parity_test.go`](../../tests/ui_cli_parity_test.go) | GUI exposes every CLI flag |
| [`tests/smoke_test.go`](../../tests/smoke_test.go) | `TestSmoke` |
| `internal/*` unit tests | `epub` (+`toc`), `fb2`, `md`, `mobi`, `rtf`, `txt`, `htmlconv`, `img`, `ocr` (`tsv`, `upscale`), `pdf` (`extract_test.go` 17 tests + `toc_test.go` 3), `config`, `htmlgen`, `outputpath`, `textutil`, `translator` |
| **No test at all** | `internal/pipeline` - the format switch itself |
| `cmd/doc-html-ui/main_test.go` | Arg assembly, settings round-trip, drop handling, Cyrillic dialog paths |
| `extension/test/*.test.mjs` | 7 `node --test` units: `ebook`, `epub`, `ocr-text`, `pdf-images`, `reflow`, `rtf`, `txt` |

Four properties of that surface shape this sweep, and each is a finding in its own right:

- **`TestConvertTestDoc` asserts only `exit == ExitOK` + `index.html` exists and is non-empty.**
  That is precisely the assertion the CBZ case defeats: 24 MB of binary-as-text satisfies all three.
  It is a *plumbing* test, and it is being read as a *correctness* test.
- **Its `supportedExts` map omits every image extension** (`.png`, `.jpg`, ..) - the comment says so
  outright. So the entire `internal/img` + forced-OCR path, a shipped feature, is untouched by the
  corpus test. That is a **third** unsynchronised format list, next to the six JS ones and the Go
  registration points.
- **`internal/pipeline` has no test at all** - the format switch, including the unknown-extension
  fallback that turns a `.cbz` into 24 MB of garbage, is neither pinned nor forbidden. This is the
  gap that matters. (An earlier draft of this file also claimed `internal/pdf` had "only
  `toc_test.go`". That was false and is retracted - see [Corrections](#corrections).)
- **No CI runs the tests.** `.github/workflows/` is `publish-cws`, `publish-edge`, `release` - all
  publishing. `scripts/test.ps1` is a local gate only.

Also note `TestConvertTestDoc` guards MOBI/AZW3 with `exec.LookPath("ebook-convert")`, while
`internal/mobi` resolves Calibre with `LookPath` **plus** a probe list of install paths. The test's
detection is narrower than the code it tests, so on a machine where Calibre is installed off `Path`
the test skips a path that actually works. On this machine Calibre *is* on the machine `Path`, so
the subtests run - but the mismatch stands.

## Rules of engagement

- **Do not fix.** Not even one-liners. A fix mid-audit invalidates every later measurement.
- **Never convert next to the input.** Always pass `-folder <scratch>`. Existing output is reused
  when `index.html` exists unless `-force`, and `test_doc/boarding-pass (5)/` is exactly that trap
  already sitting in the corpus: a leftover output dir beside its own input. A harness that writes
  in place will **silently skip conversion and report success**.
- **`-folder X` does not mean "write into X".** It nests `X/<input-basename>/`. Measured, not
  assumed - a pre-flight harness counted zero pages because of it.
- **Single-page is the default**; `-multipage` is what produces `page_*.html` plus the TOC. Both
  need covering: they are different products to the reader.
- **Record what a user would see**, not what the log says. Screenshot + console, not just stdout.

## Wave order - most popular first

| Wave | Formats | Why here | EXE | Ext |
|---|---|---|---|---|
| **1** | **PDF** | The dominant input, and the extension's whole pitch | yes | yes |
| **2** | **EPUB** | The other half of the core | yes | yes |
| **3** | **DOCX**, **TXT** | DOCX is among the most common documents on earth and is **not supported** - which makes it a D1 question, not a D2 one | yes | txt only |
| **4** | **FB2**, **MOBI/AZW3** | FB2 is big in the Russian-speaking market this app targets; MOBI/AZW3 is Kindle's | yes | yes |
| **5** | **HTML**, **RTF**, **MD** | Long tail of text inputs | yes | yes |
| **6** | **Images** | Standalone image -> forced OCR overlay | yes | partial |
| **7** | **CBZ/CBR/CB7/CBT**, **DjVu**, **XLSX/PPTX/ODT** | Roadmap. Nothing claims them - so the only question is *how they refuse* | yes | yes |

Wave 7 is not padding. Nothing declares these formats, so D2 cannot fail - but the desktop app
answers a `.cbz` with 24 MB of garbage and exit 0, and the extension routes any ZIP to the EPUB
reader (`detectFormat` returns `"epub"` for any `PK\x03\x04` before the filename is consulted) and
fails with a misleading "not an EPUB". Both are D1/D4 findings about *refusal quality*, and both are
already predicted by [`2026-07-17_comic-archives`](../plan/2026-07-17_comic-archives.md).

### Fixture assignment

Named fixtures per wave, so the sweep is reproducible rather than "some PDFs". Sizes and page counts
are the measured ones from `CORPUS.md`.

**Wave 1 - PDF**

| Fixture | Pages | Angle |
|---|---|---|
| `LV nabyvatele 14.09.2023.pdf` | 2 | Smallest real text-layer doc |
| `Кэрролл Л. - Алиса .. 2018.pdf` | 320 | Big text layer (183 834 ch), Cyrillic + parens in the name |
| `Pelevin_V. .. .a4.pdf` | 609 | Most pages with a text layer |
| `pdf-1page-tiny_NASA-Quaoar-sky-chart.pdf` | 1 | 50 KB, labels only - the degenerate input |
| `pdf-1page-blackletter_Plague-Proclamation-1625.pdf` | 1 | A text layer that is *present and terrible* - does the app trust it? |
| `pdf-1page-scan_Lincoln-Proclamation-broadside.pdf` | 1 | Scan -> OCR, dense letterpress |
| `pdf-1page-poster_Soldiers-Creed.pdf` | 1 | Scan -> OCR, very large lettering (plate geometry) |
| `comic-scan-tiny_First-Earthman-on-Mars-1944.pdf` | 6 | Scan -> OCR vs its own text-layer twin |
| `comic-textlayer-tiny_First-Earthman-on-Mars-1944.pdf` | 6 | The twin. Same pages, text layer - **compare the two outputs directly** |
| `Kupní smlouva .. .pdf` | - | **Known bug.** Non-ANSI path; `pdftotext` fails on it every run |
| `комикс-скан_První-pozemšťan-na-Marsu.pdf` | 6 | **Known bug**, fast repro |

**Wave 2 - EPUB**

`epub-tiny_Poe-The-Raven` (3 docs), `epub-textonly-deep-toc_Grimms-Fairy-Tales` (64 docs - TOC
depth), `epub-illustrated-mid_Alice-in-Wonderland` (30/55), `epub-illustrated-deep-toc_Three-Men-in-a-Boat`
(76/131 - depth *and* images), `epub3-standardebooks_Alice` (`toc.ncx`+`toc.xhtml`),
`epub3-advanced_Alice` (**nav-only, no NCX**), `epub-large_The-Jungle-Book` (10.9 MB),
`epub-huge_Treasure-Island` (48 MB - the EPUB perf number; no OCR, so it is not a stress case).

**Wave 3 - DOCX, TXT.** `DRAFT of tax opinion Dronto Investments LTD.docx` (880 KB, real-world);
`txt-big_Pride-and-Prejudice.txt` (754 KB, 11 890 lines), `Лицензионное Соглашение.txt` (4 KB, Russian).

**Wave 4 - FB2, MOBI/AZW3.** `fb2-illustrated_Alice-in-Wonderland` (29 sections, 55 `<binary>`),
`fb2-russian_Moskoviya` (Cyrillic); `mobi_Alice-in-Wonderland`, `azw3_Alice-in-Wonderland`.

**Wave 5 - HTML, RTF, MD.** `html-textonly-big_Grimms-Fairy-Tales` (0 images),
`html-danglingimages_Pride-and-Prejudice` (**164 refs that resolve to nothing**),
`html-images_Alice-in-Wonderland/pg19033-images.html` (55 refs that do resolve);
`Рабичев .. royallib.ru.rtf`; `README.md`, `CORPUS.md`.

**Wave 6 - Images.** The one-page/seven-encoding set (`img-*_Nyoka-comic-page.*`) isolates the
container as the only variable; plus `accounts.jpg`, `Свид_О_Браке_.. .jpg` (Cyrillic name, real
scan), `image.png`.

**Wave 7 - Roadmap.** `cbz-tiny_Nyoka` (+ `cbt`/`cb7` twins - same pages, different container),
`cbr-mid_Plastic-Man-v1-017` (needs 7-Zip), `djvu-tiny_First-Earthman`,
`office-xlsx_Financial-Sample`, `office-pptx_sample-presentation`, `office-odt_sample-document`.
`cbz-big_Planet-Comics-011` (71 MB) is **time-boxed**: the tiny CBZ already took 62 s to produce
garbage, so the big one costs minutes for the same finding. Run it once, for the number, or skip.

## How each dimension is measured

### D2 + D3 - EXE

One scripted pass per wave. Per fixture, in both `-multipage` and default single-page, always with
`-notranslate` (translation is checked separately) and its own `-folder`:

- exit code
- wall-clock -> derived **s/page** and **MB/s**
- artifacts: `index.html` present, `page_*.html` count, total HTML bytes, image count
- **output sanity**: does page 1 contain readable text, or bytes? (the `silently-wrong` detector -
  a share-of-printable-characters ratio separates a real extraction from a binary read as text)
- stderr lines matching `error|failed|cannot`

### D2 + D3 - extension

By hand in Chrome, same fixture list, per the checklist below. Record wall-clock with the browser's
own timing, and note that extension OCR is **lazy on scroll** - so its "time to first page" and the
EXE's "time to whole book" are not the same measurement and must not be tabled as if they were.

### D4 - quality checklist

Screenshot `index.html`, one content page, and the TOC (headless Chrome:
`chrome.exe --headless=new --screenshot=<png> --window-size=1280,2000 "file:///<path>"`). Then:

- [ ] Text is readable at default zoom; no horizontal scroll on the body
- [ ] TOC depth and entry count match the source (EPUB: its NCX/nav; PDF: its outline)
- [ ] Every TOC link resolves - no dead anchors
- [ ] Images present and not broken; nothing 404s in the console
- [ ] OCR plates sit **over** their text, at the right size, and are selectable
- [ ] Chrome's "Translate page" offers itself and translates the body (the signature flow)
- [ ] Console: zero errors
- [ ] Theme (light/dark) renders both ways
- [ ] Same fixture, EXE vs extension: do they agree on page count and TOC?

The last one is where parity findings come from, and `docs/PARITY.md` is the reference for what is
*supposed* to match.

### D1 - claims audit

Per surface in the list above, extract every claim about formats, features and dependencies into a
table: `claim | surface | language | verdict | evidence`. Two known starting points, both from the
pre-flight and both **already confirmed**:

- `internal/img` accepts `.tif`/`.tiff`; the extension's picker `accept` list has neither. It is
  also missing `.mobi`/`.azw3` while `ebook.js` reads them - the format lists have drifted from the
  code, as [`2026-07-17_comic-archives`](../plan/2026-07-17_comic-archives.md) predicted ("six
  unsynchronised format lists").
- The CLI's Ollama default is `gemma3:12b`. On this machine Ollama is installed with
  `qwen2.5:7b`, `gemma2`, `qwen2.5:3b`, `aya` - and **not** `gemma3:12b`. So `-ollama` with default
  flags fails on a box that has a working Ollama. **What the user sees in that moment is a D1/D4
  finding**: an actionable "pull this model" or an opaque error? Test it deliberately.

Also test the **dependency-absent paths**, which are invisible once the tool is installed: rename
Calibre's directory and confirm `.mobi` fails with the documented, actionable notice rather than a
crash. Same for 7-Zip once CBR/CB7 land. These paths are what a fresh user hits first and what the
store listing must have warned them about.

### D5 - documentation matrix

One row per surface, one column per question. `ok` / `stale` / `missing`:

| Question |
|---|
| Format list matches what the code actually accepts (both directions) |
| Runtime dependencies stated where a buyer reads them (Calibre for MOBI/AZW3; 7-Zip when CBR/CB7 land) |
| Feature list current - no shipped feature missing, no unshipped feature promised |
| Screenshots current (`tools/store/`, `extension/store/screenshots/`) |
| en / ru / uk all present and saying the same thing |
| Typography honours the house rules (short hyphens, `ё`, `..` not `...`) |
| Store-listing copy is publishable as-is: no dev-speak, no dead links, no unstated limits |

The 3-language rule is not cosmetic: a new user-facing feature must lead the hero and be mirrored in
en/ru/uk across every surface. Any surface where only English moved is a finding.

## Outputs

Nothing is fixed here. Each confirmed problem leaves as exactly one of:

1. **A spec ticket** - `DEV/plan/<YYYY-MM-DD>_<slug>.md`, cross-edition template
   ([`_TEMPLATE_cross-edition.md`](../plan/_TEMPLATE_cross-edition.md)) when it touches both
   editions. Status starts at `Draft`.
2. **An automated-test scenario** - a spec for a check that would have caught it. These are
   **extensions to the harness that already exists**, not new frameworks; see
   [Existing test surface](#existing-test-surface) before specifying anything:
   - **`tests/testdoc_test.go`** already drives the whole corpus through the real pipeline, one
     subtest per file. Two specs come off it: widen `supportedExts` to the image extensions it
     currently drops, and strengthen the assertion beyond "`index.html` is non-empty".
   - **`internal/pipeline`**: a format-switch test, including the unknown-extension fallback. The
     package has no test at all today, which is why the CBZ behaviour is neither pinned nor
     forbidden. This is the single biggest test gap found.
   - **`internal/pdf`**: already well covered (`extract_test.go`: `TestRowsToText`,
     `TestClassifyBlock`, `TestParsePDFLayoutPage_*`, `TestBuildPageHTML*`, `TestExtract_*`). The
     only reflow helper with no test named for it is `isLigaturesArtifact`
     ([`extract.go:428`](../../internal/pdf/extract.go#L428)), and it may be covered indirectly via
     `TestRowsToText`. Low value - do not spec a reflow harness here; spec the `(with text: N)`
     counter fix instead (W1-04).
   - **Extension**: a Playwright harness loading the unpacked MV3 extension in Chrome, driving the
     picker, asserting page count and console-clean. Today the extension has 7 `node --test` unit
     files (`ebook`, `epub`, `ocr-text`, `pdf-images`, `reflow`, `rtf`, `txt`) and **nothing that
     opens a browser** - so every end-to-end claim about it is hand-verified.
   - **Parity**: extend [`tests/parity_test.go`](../../tests/parity_test.go) - which already pins
     the theme palette, OCR version/catalog/clustering/font-fit and the reflow constants - to also
     assert the *format lists* agree across the JS surfaces and the Go registration points. That is
     exactly the class of drift that produced `.mobi` missing from the picker's `accept`.
   - **CI**: the repo's workflows are `publish-cws.yml`, `publish-edge.yml`, `release.yml` - **none
     of them runs `go test` or `npm test`.** Tests are local-only via `scripts/test.ps1`. Any
     recommendation of the form "a test would have caught it" is currently false unless someone runs
     the suite by hand, so a test-on-push workflow is itself a finding to spec.
   - **Docs**: a link-and-claim linter over the site + listings (dead links, missing locale,
     format-list drift vs the code).
3. **A CORPUS.md gap entry** - when the sweep cannot answer a question because no fixture exists.
   Known already: `.tiff`, multi-page TIFF, real-world FB2, **and no malformed/truncated/DRM input
   of any kind** - so no error path is exercised by the corpus at all.

## Findings

> Filled in as waves run. One row per finding; `#` is the ticket once it is cut.

| # | Wave | Dim | Edition | Finding | Evidence | Verdict |
|---|---|---|---|---|---|---|
| - | pre-flight | D2 | EXE | `.cbz` converts with exit 0 into 1169 files / 24 MB of binary-as-text in 62 s. The unknown-ext fallback turns "cannot read this" into "read it wrong, confidently" | smoke run 2026-07-17 | confirmed |
| - | pre-flight | D1 | Ext | Picker `accept` omits `.mobi`/`.azw3` (which `ebook.js` reads) and `.tif`/`.tiff` (which `internal/img` accepts) | `extension/src/viewer.html:47` | confirmed |
| - | pre-flight | D1 | EXE | Ollama default model `gemma3:12b` is absent on a machine with a working Ollama (`qwen2.5:7b`, `gemma2`, `qwen2.5:3b`, `aya`). What the user is told at that moment is untested | `-h` vs `/api/tags` | confirmed |
| - | pre-flight | D1 | plan | Comic-archives plan says "detect 7-Zip on PATH"; 7-Zip is in neither the machine nor the user `Path` | registry `Path` read; recorded in the ticket | confirmed |
| - | pre-flight | test | EXE | `TestConvertTestDoc` asserts only exit-OK + non-empty `index.html` - the CBZ garbage would satisfy it | `tests/testdoc_test.go:110-130` | confirmed |
| - | pre-flight | test | EXE | `TestConvertTestDoc`'s `supportedExts` omits all image extensions, so the shipped `internal/img` + forced-OCR path has no corpus coverage | `tests/testdoc_test.go:27-39` | confirmed |
| - | pre-flight | test | both | No CI workflow runs `go test` or `npm test`; the only workflows publish | `.github/workflows/` re-verified 2026-07-17: `publish-cws`, `publish-edge`, `release`; zero matches for `go test`/`npm test` | confirmed |
| - | pre-flight | test | EXE | `internal/pipeline` has no test file at all - the format switch and its unknown-ext fallback are unpinned | `Get-ChildItem internal/pipeline -Filter *_test.go` -> empty | confirmed |
| ~~-~~ | ~~pre-flight~~ | ~~test~~ | ~~EXE~~ | ~~`internal/pdf` has only `toc_test.go`~~ | **RETRACTED - false.** See [Corrections](#corrections) | **withdrawn** |

### Wave 1 - PDF (2026-07-17)

Binary: `go build` at `ba754cf`, run from a scratch dir; 11 fixtures x {single, multipage},
`-notranslate -noopen`, each with its own `-folder`. Full data: `_results.csv` in the run dir.

**D2/D3 headline: PDF is healthy.** 22/22 runs exit 0, all produce `index.html`, and the
printable-character ratio is **1.0 on every fixture** - no `silently-wrong` anywhere in PDF,
including the blackletter and Cyrillic cases. Speed is defensible: 319-page Кэрролл 8.0 s
single / 20.6 s multi; 609-page Pelevin 6.7 s / 2.3 s; every 1-6 page fixture under 1.8 s. No
fixture came close to the 600 s timeout. The two "known bug" fixtures (`Kupní smlouva`,
`комикс-скан_První-pozemšťan`) both converted **successfully** - see W1-05.

| # | Wave | Dim | Edition | Finding | Evidence | Verdict |
|---|---|---|---|---|---|---|
| W1-01 | 1 | D1/D2 | CLI | **Any flag placed after the input path is silently ignored.** `flag.Parse` stops at the first non-flag arg ([`flags.go:65`](../../internal/config/flags.go#L65)); `rest[0]` becomes the input and **`rest[1:]` is discarded with no check** ([`flags.go:109-113`](../../internal/config/flags.go#L109-L113)). So `doc-html-translate book.pdf -noopen -folder D:\out` writes next to the input and opens a browser | Measured both orders on the same fixture. Flags-after: `-folder` dir got 0 files, 5 landed next to the input, log said `Opening in browser...`. Flags-before: 4 files in `-folder`, log said `Browser open skipped (-noopen)` | confirmed |
| W1-02 | 1 | D2 | CLI | **`waitOnError` blocks forever when stdin is an open idle handle.** `fmt.Scanln()` is unconditional on every error exit ([`main.go:41-44`](../../cmd/doc-html-translate/main.go#L41-L44)). With stdin redirected it returns instantly (measured: 0.02-0.03 s); with an inherited console it never returns. The GUI already works around this by feeding a newline ([`main.go:658-660`](../../cmd/doc-html-ui/main.go#L658-L660)) - the workaround exists, the root cause does not | `-h` hung >120 s with inherited stdin; same call with a redirected empty file exited in 0.03 s | confirmed |
| W1-03 | 1 | D1/D4 | CLI | **`-h` is treated as an error**: help text goes to **stderr**, is prefixed `Error: flag: help requested`, printed in red, and exits **1**. `-version` correctly exits 0 | `-h` -> exit=1; `-version` -> exit=0 | confirmed |
| W1-04 | 1 | D4 | CLI | **The `(with text: N)` counter is mislabelled.** `generated` counts pages emitted with text **or images** ([`extract.go:230`](../../internal/pdf/extract.go#L230)) but prints as `with text` ([`extract.go:252`](../../internal/pdf/extract.go#L252) and [`:621`](../../internal/pdf/extract.go#L621), both paths). A scan with **zero** text layer reports `Pages: 6 (with text: 6)` - the one diagnostic a user would check to see whether a text layer was found actively lies | `comic-scan-tiny` (pure scan, `pdftotext extracted no content`) logged `Pages: 6 (with text: 6)`; extracted body text was 52 chars | confirmed |
| W1-05 | 1 | D2 | EXE | **The two "known bug" PDF fixtures no longer reproduce.** Both converted exit 0 with correct page counts (`Kupní smlouva` 9pp, `комикс-скан` 6pp). `pdftotext` does fail on them, but it fails *because they are scans with no text layer* ("extracted no content"), and the pdflib fallback handles them. The non-ANSI-path theory recorded in CORPUS.md is unsupported: `pdf-1page-tiny_NASA-Quaoar` has a pure-ASCII name and takes the same fallback | Run logs; note `needsPDFToTextPathStaging` + `TestStagePDFForPDFToText` exist, i.e. the path issue was already fixed | confirmed |
| W1-06 | 1 | D4 | EXE | **Single-page mode still emits `page_*.html`** as boilerplate-free fragments beside the merged `index.html` (LV nabyvatele: `index.html` 19 981 B + `page_001` 3 309 B + `page_002` 2 151 B). Whether these are reachable, intentional (`-split` for the GT extension), or dead weight is unresolved - they are not linked from the single-page `index.html` | Output listing, single vs multi | needs-triage |
| W1-07 | 1 | D4 | EXE | **A per-image OCR failure is invisible.** [`pipeline.go:507-511`](../../internal/pipeline/pipeline.go#L507-L511) logs per-*file* errors, but a failure on an individual image inside `OverlayFile` is not surfaced - only the aggregate `N image(s) overlaid` is printed, and the exit code stays 0. `ocr.Recognize` *does* return tesseract's stderr ([`tesseract.go:112-113`](../../internal/ocr/tesseract.go#L112-L113)), so the detail exists and is discarded upstream. Result: "OCR found no text here" and "OCR could not read this file at all" are indistinguishable to the user | 4 of 6 comic pages failed with tesseract's `cannot read input file`; the log said only `2 image(s) overlaid`, no error line, exit 0 | confirmed |
| - | 1 | corpus | - | `test_doc` holds **61** top-level files / 807.4 MB; CORPUS.md says 62. Nothing was lost (all deletions verified as 13:11-created output), so the doc's count is suspect | `Get-ChildItem -File` count vs CORPUS.md | open |

**OCR (D2/D4) - the engine is healthy; my first reading of it was not.** With `eng` installed
(`tessdata` is `<exe dir>/tessdata`, so a binary built to a bare directory reports every language as
"available - download"), forced OCR works: Soldiers-Creed poster 40 -> 703 chars, blackletter-1625
correctly unchanged at 2574 (it already has a text layer, so there is nothing to OCR).

The **scan-vs-text-layer twin** is the sharpest instrument in the corpus and it is worth keeping:
same 6 comic pages, one scanned, one with a real text layer. Text layer yields ~3200 chars/page;
the scan yields boilerplate-only on the pages OCR could not read. Once the path problem below is
removed, this pair measures OCR quality directly against ground truth - the only fixture in the
corpus that can.

An apparent "OCR silently produces nothing on dense scans" finding was **investigated and withdrawn**
- it was an artefact of the harness's own deep scratch path. The mechanism is worth recording because
it is a real (if narrow) product edge and it explains an otherwise baffling result:

- Tesseract is a Win32 binary with no long-path manifest, and `LongPathsEnabled = 0` on this machine.
  An image whose path exceeds `MAX_PATH` (260) fails with `cannot read input file`.
- `ocrUpscaleBelow = 1000` ([`tesseract.go:131`](../../internal/ocr/tesseract.go#L131)): images with
  a long side **below 1000 px** are upscaled into `%TEMP%` - a *short* path - and therefore succeed,
  while larger images are recognized **in place** at the long path and fail. This exactly predicted
  the observed 2-of-6: comic pages 2 (971 px) and 6 (911 px) overlaid; pages 1/3/4/5 (1001 px) did
  not. Nothing about the images' legibility differed.
- Proof: the Lincoln scan (1847x2949) at a 288-char path OCR'd **0** chars in 0.2 s; the identical
  file copied to `C:\t\lincoln.png` OCR'd **3182** chars in 1.57 s.
- **Reachability, measured rather than asserted:** the app embeds the book title *twice* in the image
  path (`<out>/<title>/pdf_images/<title>_<page>_<img>.jpg`), so headroom is
  `260 - (2 x title) - 23`. A realistic re-test - a 76-char Russian title under
  `C:\t\OneDrive\Документы\Книги` - produced a 203-char path and OCR'd fine. It takes a genuinely
  long title (the corpus has one at 97 chars) plus a deep folder to cross 260. **Narrow, not
  common** - which is why the finding filed above is W1-07 (the silence), not the path limit itself.

**Not findings** (checked, behaving correctly - recorded so the next pass does not re-litigate):

- **Blank-page skipping is deliberate and safe.** A page is dropped only when it has *neither* text
  *nor* images ([`extract.go:226`](../../internal/pdf/extract.go#L226)), so illustrations and scans
  are never lost. Output pages are renumbered contiguously (319 -> 317 for Кэрролл, 609 -> 607 for
  Pelevin), and `pdfPageToHref` exists specifically so PDF bookmarks still resolve across the
  renumbering. An earlier read of this as "silent data loss" was wrong.
- **Documentation teaches the correct flag order.** Every CLI example on every surface puts flags
  before the path (README L87-L102, docs.html L245-L248, all three locales), the GUI appends
  `req.Input` last ([`main.go:802-804`](../../cmd/doc-html-ui/main.go#L802-L804)), and the registry
  verb is `"<exe>" "%1"` with no flags. W1-01 therefore bites only hand-typed and scripted use - it
  is a real trap, not a shipped-surface outage.

### Corrections

Kept visible on purpose - a wrong "measured" fact is worse than an open question. Both entries here
are the same mistake: a claim written as *measured* that was never measured.

- **"Calibre is not on PATH" was wrong.** The winget install *does* register
  `C:\Program Files\Calibre2\` on the machine `Path`; the shell that reported otherwise had started
  before the install and held a stale environment. Read the persisted `Path` from the registry, not
  from a long-lived process. The 7-Zip finding survived that same re-check and is real.
- **"`internal/pdf` has only `toc_test.go`" was wrong.** It also has
  [`extract_test.go`](../../internal/pdf/extract_test.go) - **17 tests**, including `TestRowsToText`
  and `TestClassifyBlock`, i.e. exactly the reflow behaviour this file claimed was "unpinned". The
  claim came from a paginated `Test\w+` grep whose second page was never read, and it had been
  promoted to `confirmed` in the findings table. The `internal/pipeline` half of that row is true
  and survives on its own. Re-verified at the same time and **holding**: the weak
  `TestConvertTestDoc` assertions (`exit == ExitOK`, index exists, `Size() != 0`), `supportedExts`
  omitting every image extension, and no CI workflow running tests.
- **The first Wave 1 run wrote 11 output dirs into `test_doc/`** - the exact "never convert next to
  the input" trap this file warns about, caused by W1-01 silently voiding `-folder`. All 11 were
  verified as containing only run-timestamped output and removed; the corpus is intact (807.4 MB,
  `boarding-pass (5)` untouched at its original 07/01 date). The harness now passes flags before the
  path. Worth noting the trap fired *through* a product bug, not through carelessness alone - which
  is why W1-01 is filed as D1/D2 rather than a docs nit.

## Run 2 - 2026-07-18 post-fix re-test

Second pass, after the fourteen `2026-07-17_*` tickets were implemented and moved to
`DEV/plan/done/`. Binary: fresh `build/doc-html-translate.exe` `26.0718.0052` (gate green - full
`go test` incl. the corpus-driven `tests/testdoc_test.go`, lint, typos). This run does what the
first pass never reached: it carries D2/D3 across **all seven waves**, not just PDF. Same rules of
engagement (own `-folder`, flags before the path per W1-01, never next to the input). Stress file
`Aphrodite's Mirror (1).pdf` excluded as before. Harness + raw data: `temp/sweep/` (gitignored) -
`results.csv`, `shots/*.png`.

**Headline: the re-test is clean. Zero `silently-wrong` anywhere.** Every converted fixture has a
control-char ratio of 0 % on its content page; every unsupported binary now exits 3 with an
actionable refusal instead of emitting garbage. The old pre-flight verdict ("a `.cbz` is 24 MB of
binary-as-text at exit 0") is gone.

### D2/D3 - all waves (both single-page and -multipage)

| Wave | Fixtures | Result |
|---|---|---|
| W1 PDF | 11 | all exit 0, ctrl 0 %. Text layers clean (Кэрролл 317 pp 5 s, Pelevin 607 pp 1.7 s). Both non-ANSI-path "known bugs" convert (Kupní 9 pp, První 6 pp). Scan counter now honest: `Pages: 6 (with text: 0, image-only: 6)` (W1-04 fixed) |
| W2 EPUB | 8 | all exit 0, ctrl 0 %. Single-page `index.html` is a 142 B redirect to `OEBPS/index.html` (by design, not empty output) |
| W3 DOCX/TXT | 3 | docx **exit 3** - `looks like a ZIP archive .. refusing to convert it into garbage`; txt clean (86/196 pp) |
| W4 FB2/MOBI/AZW3 | 4 | all exit 0, ctrl 0 %. FB2 15/12 pp; MOBI/AZW3 via Calibre |
| W5 HTML/RTF/MD | 5 (+html-images) | all exit 0, ctrl 0 %. html-images (missed in W5 list, run separately) resolves 28/28 |
| W6 images | 9 | all exit 0, ctrl 0 %. TIFF transcodes to PNG and renders. Forced OCR overlay on every one |
| W7 roadmap | 8 | **comics now convert**: cbz/cbt/cb7 37 pp, cbr 36 pp (7-Zip), ctrl 0 %. djvu + xlsx/pptx/odt **exit 3** with a named refusal each |

### D4 - visual (headless Edge, full-page PNG, eyeballed)

| Case | Verdict |
|---|---|
| fb2-illustrated | 55 imgs, 0 broken - cover + text render (fixes `fb2-drops-every-image`) |
| html-images resolvable | 28 imgs, 0 broken - CORPUS said "all 28 come out broken"; now none (fixes `html-input-drops-local-images`) |
| epub-illustrated | 55 imgs, 0 broken |
| cbz-tiny comic | 37 pages render; OCR plate over the masthead sits tight and readable |
| img-tif (TIFF->PNG) | renders; **OCR overlay plates on this 800 px low-DPI indicia page are loose - oversized and overlapping**. Documented resolution-limited case, not a regression (archive/scan path looks clean) |
| html-danglingimages | 164 refs resolve to nothing (by design); page degrades gracefully - text + TOC intact, only broken-image glyphs |
| Кэрролл multi | TOC tree "Chapters: 317", nested expanders, Cyrillic clean, links live |

### D1 / D5 - claims and docs

Both dimensions run read-only over every published surface. **Regression is clean**: every
prior-broken claim now matches the code - extension picker accepts `.mobi/.azw3` again; comics are
claimed on README/site/docs/PARITY and registered in the MSIX FTA; `.tif` absence from the extension
picker is correct (no browser TIFF decoder), so that pre-flight item is withdrawn as a non-bug. All
hand-authored site/README copy is mirrored across en/ru/uk - no "English-only moved" break.

The only gap is a small cluster where the 07-17 work did not reach the **metadata** surfaces:

- **winget en-US locale** (`winget/SerZhyAle.DocHtmlTranslate.locale.en-US.yaml`) omits comics +
  standalone images + the 7-Zip dependency, and its version is stale (`26.0702.1530`). wingetcreate
  carries locale text forward, so this ships unless edited at the next submission.
- **MSIX VisualElements Description** (`msix/AppxManifest.xml:65`) lags its own file associations -
  the FTA registers `.cbz/.cbr/.cb7/.cbt`, the description still lists only the nine doc formats.
- **Store-copy typography**: em dash in `extension/store/LISTING.md:84` and
  `extension/store/PRIVACY.md:5`, `...` in `extension.html` UI-label quotes. House rule is short
  hyphens / `..`; these slipped the typos gate (it does not scan `.md`/`.html` for em dashes).
- **Extension screenshots** incomplete: only `shot1/2-en,-ru`; no `shot3-*`, no `-uk`, none depict
  the new comic/image features.
- **(minor)** standalone image input under-surfaced on the site meta descriptions.
- **Ollama prerequisite**: default `-free`/`-ollama` silently needs `ollama pull gemma3:12b`; no
  buyer surface states it, only a raw runtime error does.

### Still-open pre-existing (not regressions, carried from Run 1)

- **W1-01** flags after the input path are silently ignored (`flags.go:109-113`) - reconfirmed this
  run (it is what voided `-folder` on the first harness attempt). Bites hand-typed / scripted use
  only; every shipped surface orders flags first.
- **W1-07** a per-image OCR failure is not surfaced (aggregate count only, exit 0).

### Verdict

All fourteen `2026-07-17_*` fixes hold under a full-corpus re-run. No functional regression, no
`silently-wrong`, refusals are clean and actionable. Remaining work is documentation/metadata polish
(the winget/MSIX/store-copy cluster) plus two pre-existing low-severity CLI-diagnostics items, and
one visual judgement call on OCR plate density for low-DPI standalone images. The extension edition
(D2/D4 by hand in Chrome) is **not covered by this pass** - it needs a human driver; checklist in
the run notes.
