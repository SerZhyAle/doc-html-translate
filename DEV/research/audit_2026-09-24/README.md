# Code audit 2026-09-24 - findings register and spec set

**Status:** Research complete; the 17 tickets under `specs/` are `Draft`.
**Scope:** the whole repo - Go CLI/GUI (`cmd/`, `internal/`) and the JS extension (`extension/src`).
**Method:** a full read of every non-test source file, split into six areas. The findings were
cross-checked against the live tree, and `go vet` (linux + `GOOS=windows`), `go test -race ./internal/... ./cmd/...`
and `staticcheck` were run. Line numbers refer to commit `41fbc1b`.

## Why the tickets live here and not in `DEV/plan/`

`DEV/plan/` is gitignored ("tickets stay local"), and this audit ran in an ephemeral cloud container.
A ticket written only there would have been lost. The specs are committed here in the ticket shape
(`YYYY-MM-DD_<slug>.md`, first `**Status:**` line) so they can be moved into `DEV/plan/` locally as-is,
with a line added to `DEV/plan/RELEASE_QUEUE.md` for each. Each ticket cites finding ids from this
register, not file paths, so the specs stay strategic and the evidence has one home.

## Reproduced by hand

- **P1** is confirmed by execution. `doc-html-translate -noopen <dir>` with `<dir>/a.jpg` inside printed
  `extract as txt: no text content found` and **deleted `<dir>` recursively**.
- **Q6**: `go test` on linux fails `TestPdfTitle`, because the test hard-codes a Windows path.
- **Q7**: on non-Windows the PDF path tries to exec `pdftotext.exe` from the cache, then suggests `winget`.
- **X1, X4, X5** were executed by the auditing agent: RTF `\uN` becomes NUL, windows-1251 FB2 fails to
  parse, and truncated UTF-8 TXT turns into mojibake.

## Suggested execution order (release packages)

- **rel 1 - data safety:** `hotfix-output-dir-ownership`, `hotfix-epub-href-containment`,
  `bugfix-shell-open-injection`, `bugfix-gui-local-api-hardening`.
- **rel 2 - output truth:** `bugfix-output-completeness`, `bugfix-translation-engine-correctness`,
  `bugfix-reader-layer-and-single-page`, `bugfix-legacy-text-decoding`.
- **rel 3 - robustness:** `bugfix-external-process-bounds`, `bugfix-resource-budgets`,
  `bugfix-ocr-language-data-and-detection`, `bugfix-pdf-extraction-accuracy`.
- **rel 4 - fidelity and extension:** `bugfix-epub-html-content-fidelity`, `bugfix-extension-lifecycle-leaks`,
  `bugfix-extension-content-security`, `bugfix-windows-registration-honesty`, `chore-hygiene-and-test-gaps`.

## Ticket index

- [01 hotfix-output-dir-ownership](specs/2026-09-24_hotfix-output-dir-ownership.md) - P95 - **BlockNeedUserTest** (implemented 2026-09-25) - P1 P2 P6 P7 P16 P17 G9 G11
- [02 hotfix-epub-href-containment](specs/2026-09-24_hotfix-epub-href-containment.md) - P95 - E1 E5 E10 E20 B19 B20
- [03 bugfix-shell-open-injection](specs/2026-09-24_bugfix-shell-open-injection.md) - P90 - P3 P24 G4 G13
- [04 bugfix-gui-local-api-hardening](specs/2026-09-24_bugfix-gui-local-api-hardening.md) - P90 - G1 G3 G5-G8 G10 G12 G14-G19
- [05 bugfix-output-completeness](specs/2026-09-24_bugfix-output-completeness.md) - P90 - P4 P5 P8 P9 T5 O10
- [06 bugfix-translation-engine-correctness](specs/2026-09-24_bugfix-translation-engine-correctness.md) - P85 - T1-T4 T6-T9 P10 P11 P20
- [07 bugfix-reader-layer-and-single-page](specs/2026-09-24_bugfix-reader-layer-and-single-page.md) - P80 - E2 E3 E4 E13 E17 E18 E19 E23 E24 X23
- [08 bugfix-epub-html-content-fidelity](specs/2026-09-24_bugfix-epub-html-content-fidelity.md) - P65 - E6-E9 E11 E12 E14 E15 E21 E22
- [09 bugfix-legacy-text-decoding](specs/2026-09-24_bugfix-legacy-text-decoding.md) - P80 - X1-X5 X18 X19 X21 X22 P23 B21
- [10 bugfix-external-process-bounds](specs/2026-09-24_bugfix-external-process-bounds.md) - P75 - X7 X9 X10 O5 O7 O9 P15
- [11 bugfix-resource-budgets](specs/2026-09-24_bugfix-resource-budgets.md) - P75 - X11-X17 X20 X24 O6 E16 B23
- [12 bugfix-pdf-extraction-accuracy](specs/2026-09-24_bugfix-pdf-extraction-accuracy.md) - P65 - X6 X8 Q6 Q7
- [13 bugfix-ocr-language-data-and-detection](specs/2026-09-24_bugfix-ocr-language-data-and-detection.md) - P70 - O1-O4 O8 O11 O12
- [14 bugfix-windows-registration-honesty](specs/2026-09-24_bugfix-windows-registration-honesty.md) - P50 - P12 P13 P14
- [15 bugfix-extension-lifecycle-leaks](specs/2026-09-24_bugfix-extension-lifecycle-leaks.md) - P60 - B1-B13 B25 B27 B28
- [16 bugfix-extension-content-security](specs/2026-09-24_bugfix-extension-content-security.md) - P60 - B14-B18 B22 B24 B26 B29
- [17 chore-hygiene-and-test-gaps](specs/2026-09-24_chore-hygiene-and-test-gaps.md) - P40 - P18 P19 P21 P22 Q1-Q5 + test gaps

Every finding below maps to exactly one ticket.

## Findings register

Format: `id - severity - confidence - finding - evidence - ticket`.
Severity is crit / high / med / low. Confidence: `conf` means confirmed by reading or running the code;
`plaus` means plausible, because it depends on OS or third-party behaviour.

### P - CLI, pipeline, app wiring, support packages

- P1 - crit - conf (reproduced) - a directory given as input maps its output dir onto itself, and the failed "txt" extraction then `RemoveAll`s it - `outputpath/outputpath.go:19-27`, `pipeline/pipeline.go:71,200-210` - 01
- P2 - crit - conf - every extraction failure and `-force` `RemoveAll` a pre-existing directory that this run did not create. With `-folder`, the output dir can even contain the source file - `pipeline.go:85,105,118..222` - 01
- P3 - high - plaus - `cmd /c start "" <path>`: `&`, `^`, `%VAR%` in a book filename are interpreted by cmd.exe (command injection, broken open) - `browser/browser_windows.go:12` - 03
- P4 - high - conf - `index.html` is written before OCR and translation. An interrupted or failed run leaves an output that the next run reuses as complete - `pipeline.go:237-251,316-321,345` - 05
- P5 - med - conf - the reuse check ignores the options (engine, `-dst`, `-ocr`, `-multipage`) and the source's mtime/size - `pipeline.go:82` - 05
- P6 - high - conf - `book.epub`, `book.pdf` and `book.txt` in one folder share one output dir. One silently opens the other, or `-force` destroys it - `outputpath.go:20-22` - 01
- P7 - med - conf - no lock: two concurrent runs on one output dir, and one failure deletes the other's work - `pipeline.go:82-105` - 01
- P8 - med - plaus - the Ctrl+C handler calls `os.Exit(130)` during a truncating `os.WriteFile`. Defers are skipped, and the goroutine stays parked after `signal.Stop` - `pipeline.go:316-323`, `htmlproc/htmlproc.go:101` - 05
- P9 - med - conf - a failed translation returns exit 0 and says "opened WITHOUT translation" while pages 0..i-1 are already translated. `ExitAPI` is never used - `pipeline.go:423-432` - 05
- P10 - med - conf - the Google cost confirmation is always a modal MessageBox (stdin EOF on non-Windows), even with `-noopen` or within `-max-cost`. It blocks scripted runs - `pipeline.go:288-301`, `dialog/dialog_windows.go:24-33` - 06
- P11 - med - conf - the `-max-cost` guard is skipped at 1000 chars or fewer, ignores the title and TOC labels, and counts bytes rather than characters - `pipeline.go:288,452-462,644` - 06
- P12 - med - plaus - default-handler registration writes only `HKCU\..\Classes\.ext`: no `SHChangeNotify`, and `UserChoice` wins on Win8+, while the message says "DONE" - `windowsreg/register_windows.go:277-285`, `app/app.go:147-154` - 14
- P13 - med - conf - Unregister does not restore the previous handler, ignores `DeleteValue` errors and always returns nil - `register_windows.go:237-256` - 14
- P14 - low - conf - registration errors are discarded, and a total failure on first run prints nothing - `app.go:112,123-124,146,231-233` - 14
- P15 - med - conf - the cached `pdftotext.exe` is reused with no version or hash check. The write is not atomic and races across instances - `bundledtools/pdftotext.go:32-64` - 10
- P16 - low - conf - the reserved-name check misses `CON.x`, `CONIN$`, `COM¹`, and invalid characters. Dotfiles collapse to one name - `outputpath.go:43-58` - 01
- P17 - low - conf - stat errors other than NotExist fall through, `MkdirAll` runs before readability is proven, and a 0-byte file gets a vague parse error - `pipeline.go:71` - 01
- P18 - low - conf - the version sentinel is matched by string (`err.Error() == "version"`) - `config/flags.go:104`, `cmd/doc-html-translate/main.go:26` - 17
- P19 - low - conf - negative `-split`, `-toc-depth` and `-ollama-ctx` are accepted, and the Ollama values are clamped silently - `flags.go:62-64`, `translator/ollama.go:91-105` - 17
- P20 - low - conf - a NaN, negative or Inf `-max-cost` silently means "no limit" - `flags.go:71` - 06
- P21 - low - conf - the console write happens outside the lock, although the comment says "under the lock" - `logging/log.go:43-50` - 17
- P22 - low - conf - run-log names have one-second resolution with `O_APPEND` (batch runs interleave), a single run has no size cap, and report names have minute resolution - `report/store.go:44-46`, `report/archive.go:40` - 17
- P23 - low - plaus - invalid UTF-8 from pdftotext is dropped silently (`ToValidUTF8(.., "")`) - `textutil/lines.go:22-24`, `pdf/extract.go:198` - 09
- P24 - low - conf - `Browser.Open` never calls `Wait` or `Release`, so process handles leak in a long-lived GUI - `browser_windows.go:14` - 03

### G - GUI launcher (`cmd/doc-html-ui`)

- G1 - high - conf - the stdout and stderr goroutines write the same `http.ResponseWriter` concurrently - `main.go:642-652` - 04
- G2 - duplicate of P3, reached from the GUI - `main.go:514,550`, `report.go:121` - 03
- G3 - high - conf - no token and no Origin/Host check. GET-triggerable `/api/register`, and a `text/plain` JSON CSRF can reach `/api/run`, `/api/delete-output`, `/api/drop`, `/api/google-key` and `/api/settings`. DNS rebinding can read settings - `main.go:66-94,583,601` - 04
- G4 - med - conf - the input is appended without `--`, so an input starting with `-` is parsed as a CLI flag - `main.go:861-863` - 03
- G5 - med - conf - empty or non-integer Ollama fields are forwarded even when Ollama is off, and the CLI parse error breaks every run - `main.go:816-824`, `ui.html:373-374,813-814` - 04
- G6 - med - conf - the child ignores the request context, there is no cancel and no per-output guard, and killing the GUI leaves the child running - `main.go:621,1073` - 04
- G7 - med - plaus - a `bufio.Scanner` line over 64 KiB stops draining, the child blocks on the pipe, and the request and `activeRuns` hang - `main.go:644-648` - 04
- G8 - med - plaus - the CLI child is spawned without `hideWindow`, so a console pops up and closing it aborts the run - `main.go:621-635` - 04
- G9 - med - conf - dropped files with the same name overwrite each other and reuse stale output. A partial upload clobbers the good copy, and the drop folder is never cleaned - `main.go:202-211` - 01
- G10 - med - plaus - the heartbeat grace is 15 s while Chrome throttles a hidden page's timers to 1/min, so a minimized GUI server exits - `ui.html:554`, `main.go:1077-1079` - 04
- G11 - med - conf - `delete-output` `RemoveAll`s any `<folder>/<name>` holding an `index.html` - `main.go:579-583` - 01
- G12 - low - conf - settings and history writes are not atomic, unlocked read-modify-write, and corruption silently resets everything - `main.go:243-263,471,587-590,660-662` - 04
- G13 - low - conf - `quoteArg` breaks on a trailing backslash and on cmd metacharacters, and the PowerShell form needs `& ` - `main.go:880-887` - 03
- G14 - low - plaus - `-register` and the PowerShell dialogs have no timeout and are not counted in `activeRuns` - `main.go:681,986` - 04
- G15 - low - conf - a multi-file drop silently uses only the first file - `ui.html:585,615` - 04
- G16 - low - conf - an empty Split field silently becomes the CLI default 5000, while the GUI default is 0 - `main.go:825-827` - 04
- G17 - low - conf - the saved `engine` string is spliced into a CSS selector, and a bad value aborts `applySettings` halfway - `ui.html:828` - 04
- G18 - low - conf - `runConvert` ignores `resp.ok` and force-scrolls the log on every chunk - `ui.html:1204-1212` - 04
- G19 - low - conf - the UI/CLI parity test only checks that the flag string appears, not its value types - `tests/ui_cli_parity_test.go` - 04

### E - EPUB, HTML processing, generation, split, htmlconv, md

- E1 - crit - conf - OPF manifest, spine and `container.xml` hrefs are joined with no containment check. Single-page mode reads a file outside the book and then **deletes** it; multipage rewrites outside files - `epub/epub.go:382-387,450`, `htmlgen/singlepage.go:147-152`, `htmlsplit/split.go:245-250`, `navbar.go:705` - 02
- E2 - high - conf - the reading-position key is hashed from the title before translation on chapter pages and after translation on index, so "Continue reading" never matches - `navbar.go:692`, `pipeline.go:454`, `htmlgen.go:109` - 07
- E3 - high - conf - the single-page merge copies bodies verbatim into `BasePath/index.html`, which breaks relative images and links for chapters in subfolders (Sigil layout) - `singlepage.go:64,133` - 07
- E4 - high - conf - the single-page merge deletes the chapter files that footnote and cross-ref links point to, and merged ids can collide - `singlepage.go:147-152` - 07
- E5 - high - conf - percent-encoded manifest hrefs are never decoded, and one missing file aborts the XHTML normalization of the whole book - `epub.go:266-269,382-387` - 02
- E6 - high - conf - the cover-SVG regex can span two `<svg>` blocks and replace the text between them with one `<img>` - `epub.go:319-336` - 08
- E7 - med - conf - splitting never updates TOC hrefs or cross-file fragments - `htmlsplit/split.go:76-98` - 08
- E8 - med - conf - split pages drop the `<html lang/dir>` and `<body>` attributes (RTL, and Chrome's translate offer) - `split.go:163` - 08
- E9 - med - conf - split is a no-op for a single wrapper `<div>` and counts bytes, not characters - `split.go:131-141,199` - 08
- E10 - med - conf - the reserved `index.html` check is case-sensitive, so `Index.xhtml` is overwritten on NTFS - `epub.go:201-202` - 02
- E11 - med - conf - global substring replacement of `index.html` and `.xhtml` rewrites unrelated URLs and prose, in random map order - `epub.go:232-235,273-274,304-305` - 08
- E12 - med - plaus - XHTML is renamed to `.html` without re-serializing, so `<script/>`, `<title/>` and `<a id=../>` are misparsed as HTML - `epub.go:271-281` - 08
- E13 - med - conf - the reading-position restore ignores `location.hash` and overrides TOC and fragment navigation - `navbar.go:547-553` - 07
- E14 - med - conf - HTML input and EPUB XHTML are parsed as UTF-8 with no charset detection - `htmlconv/extract.go:109,289`, `singlepage.go:55` - 08
- E15 - med - conf - `srcset` and `<picture><source>` are not rewritten, so images still break - `htmlconv/extract.go:148-153` - 08
- E16 - med - conf - EPUB extraction has no total-size or entry-count cap, and the per-file 100 MB cap truncates silently - `epub.go:421-423` - 11
- E17 - med - conf - `lang` is emitted with Go `%q` inside an HTML attribute (attribute injection) - `singlepage.go:102` - 07
- E18 - med - conf - external TOC hrefs get a `BasePath/` prefix, and `javascript://` counts as "external" - `epub/toc.go:298-331`, `htmlgen.go:189,206-211` - 07
- E19 - low - conf - filenames and heading ids go into hrefs and JS without percent-encoding (`#`, `%`, `?`) - `navbar.go:595,600`, `htmlgen.go:144,365`, `toc_scan.go:139` - 07
- E20 - low - plaus - zip entries that differ only by case overwrite each other on Windows - `epub.go:390-424` - 02
- E21 - low - conf - htmlconv image names: a case-sensitive collision check, a clash with `favicon.ico`, and symlinks followed out of the source tree - `htmlconv/extract.go:184-225`, `htmlgen/favicon.go:32` - 08
- E22 - low - conf - MD and HTML output hardcode `lang="en"`, htmlconv drops the head CSS, and MD copies no local images - `md/extract.go:113`, `htmlconv/extract.go:113-124,288`, `htmlgen.go:71` - 08
- E23 - low - conf - navbar injection is not idempotent (a duplicate spine href gets two navbars and zoom applied twice), and `xml:lang` is ignored - `navbar.go:741-763`, `singlepage.go:235-249` - 07
- E24 - low - conf - `bookStorageKey = fnv32(title|count)`: books with the same title and page count (or no title) share a reading position - `navbar.go:583-587` - 07

### X - format extractors (pdf, mobi, fb2, rtf, txt, img, comic)

- X1 - high - conf (executed) - RTF `\uN`: the parameter is consumed twice, so NUL is written. Negative values are dropped and `\ucN` is ignored - `rtf/extract.go:131-148,170` - 09
- X2 - high - conf - RTF destinations (`fonttbl`, `colortbl`, `info`, `pict`, `\*`) and `\binN` land in the body text, and `\~` and `\_` print literally - `rtf/extract.go:90-98,151-153` - 09
- X3 - high - conf - RTF `\'XX` is always decoded as cp1251, `\ansicpg` is ignored, and a decoder is created per byte - `rtf/extract.go:120,225-234` - 09
- X4 - high - conf (executed) - FB2 declared `windows-1251` fails outright because there is no `CharsetReader` - `fb2/extract.go:76,179` - 09
- X5 - high - conf (executed) - one invalid byte in a UTF-8 TXT sends the whole file to the legacy detector, which yields mojibake - `txt/extract.go:66-69`, `txt/legacy.go:68,89` - 09
- X6 - high - conf - `parseImagePageNum` takes the first `_N_` segment, so a PDF named `Volume_3.pdf` puts every image on page 3 - `pdf/extract.go:1312-1324` - 12
- X7 - high - plaus - pdfcpu calls on the main pdftotext path and in repair have no `recover`, and the pipeline has none either - `pdf/extract.go:673,912-940` - 10
- X8 - med - conf - trailing image-only pages are lost because the page count comes from pdftotext's trimmed output - `pdf/extract.go:202-213,929` - 12
- X9 - med - conf - no timeout or context on pdftotext, winget, ffmpeg/magick, Calibre or 7-Zip - `pdf/extract.go:101-105,192,1188`, `mobi/extract.go:61`, `comic/readers_sevenzip.go:69` - 10
- X10 - med - conf - an unrequested system-wide `winget install ossia.poppler` runs mid-conversion with auto-accepted agreements - `pdf/extract.go:95-130` - 10
- X11 - med - plaus - no pixel budget before `tiff.Decode`, and the PDF flip doubles peak memory - `img/extract.go:198`, `pdf/extract.go:1261-1271` - 11
- X12 - med - conf - the TIFF IFD walk casts `uint32` to `int`, which goes negative on the 386 build and panics - `img/extract.go:168-174` - 11
- X13 - med - conf - the whole TIFF file is copied once per frame (up to 4096 frames) - `img/extract.go:179,195-196` - 11
- X14 - med - conf - the PDF TIFF flip runs per pixel through `At`/`Set` interface calls - `pdf/extract.go:1272-1276` - 11
- X15 - med - conf - CBR/CB7: 7-Zip extracts everything to `%TEMP%` before any cap is checked (disk bomb) - `readers_sevenzip.go:69-108` - 11
- X16 - low - plaus - symlinks extracted by 7-Zip are followed by `ReadFile` - `readers_sevenzip.go:83-98` - 11
- X17 - med - conf - every comic page is held in RAM, up to a 4 GB cap, which exceeds the 386 address space - `comic/extract.go:58-62,80-119` - 11
- X18 - med - conf - FB2 collects only `<p>`, so `<poem>/<v>`, `<subtitle>`, `<text-author>` and table cells are dropped - `fb2/extract.go:213-225` - 09
- X19 - low - conf - FB2 image names: Cyrillic ids of equal length collide, and an id of `..` fails the conversion - `fb2/extract.go:350,375-389` - 09
- X20 - low - conf - FB2 peak memory is several times the file size (the base64 is copied three times) - `fb2/extract.go:70-76,249-264` - 11
- X21 - low - conf - BOM-less UTF-16 TXT is valid UTF-8 and passes through with NULs - `txt/extract.go:66` - 09
- X22 - low - conf - non-Cyrillic legacy TXT and RTF high bytes are written raw into `charset=UTF-8` HTML - `txt/legacy.go:89-90`, `rtf/extract.go:151-153` - 09
- X23 - low - conf - the image `src` is only HTML-escaped, not URL-escaped (`scan#1.png`, `50%.png`) - `img/extract.go:66,257` - 07
- X24 - low - plaus - comic routing is by extension only, so a `.cbz` that is really RAR fails - `comic/extract.go:82-89` - 11

### T - translation engines

- T1 - high - conf - the Google API key is in the URL query and leaks through `*url.Error` into the console and the run log (only reports are redacted) - `translator/translator.go:164,180-183`, `pipeline.go:425` - 06
- T2 - high - conf - plain DOM text is sent with `format: "html"` and the entity-escaped reply is stored as raw text, so `&#39;` shows literally and `a<b` is mangled - `translator.go:152-157`, `htmlproc.go:53-58,91` - 06
- T3 - high - conf - batches are capped by characters only, while Google v2 allows 128 `q` per request. A 400 is not retried, which stops the book - `translator.go:223-257` - 06
- T4 - med - conf - the translation count is not checked against the input count. The cache panics, or results shift onto the wrong nodes - `translator.go:212-216,144`, `translator/cache.go:51-55` - 06
- T5 - med - conf - partial translation persists: Google stops at the first page error, and Ollama discards a whole page when one batch fails - `pipeline.go:423-439`, `ollama.go:150-194` - 05
- T6 - med - conf - retry is 3x at 1/2/4 s with no jitter, `Retry-After` is ignored, and a rate-limit 403 is not retried - `translator.go:167-201` - 06
- T7 - med - conf - Ollama: a multi-line segment breaks the numbered prompt and parser, so text after the first line is lost and a `1984.` line overwrites a slot - `ollama.go:288-290,349-359` - 06
- T8 - duplicate of P11 (cost guard) - 06
- T9 - low - plaus - Ollama has one 300 s client timeout that also covers cold model load, and no context - `ollama.go:23,83-85` - 06

### O - OCR

- O1 - med - conf - the OCR language code is used unvalidated in a path and URL (`../../x`), reachable from the GUI API and `-ocr-download` - `ocr/tessdata.go:153-186` - 13
- O2 - med - plaus - concurrent downloads of one language share a fixed `.tmp` name, and a corrupted file is renamed into place - `tessdata.go:172-173` - 13
- O3 - low - conf - no checksum and no size bound on traineddata downloads, and a `.tmp` is left behind after a crash - `tessdata.go:163-177` - 13
- O4 - med - plaus - tessdata sits next to the exe, which is read-only under MSIX, so no language download can succeed there - `tessdata.go:51-57` - 13
- O5 - med - conf - Tesseract runs with no timeout, so one hung image freezes `wg.Wait` for the whole conversion - `ocr/tesseract.go:121,246`, `ocr/script.go:93`, `overlay.go:435` - 10
- O6 - high - plaus - images are decoded with no pixel budget, several full-frame copies exist at once, 16 workers run, and the build is 386 - `overlay.go:552-563`, `tesseract.go:501-512`, `exif.go:161-163`, `screen.go:229-242` - 11
- O7 - med - plaus - no `recover` in the OCR worker pool, so a decoder panic kills the "best-effort" OCR run - `overlay.go:401-429` - 10
- O8 - med - conf - script detection skips ASCII-path staging and silently fails on Cyrillic book folders - `script.go:88-93`, `overlay.go:162` - 13
- O9 - low - plaus - no `OMP_THREAD_LIMIT=1`, so NumCPU-2 processes times OpenMP threads oversubscribe the CPU - `overlay.go:344-360` - 10
- O10 - low - conf - the overlaid page is truncated before re-render, so a failure loses the original page - `overlay.go:218-228` - 05
- O11 - low - plaus - the image `src` is not URL-decoded and `?`/`#` are not stripped, so such images are skipped silently - `overlay.go:335` - 13
- O12 - low - plaus - `ensureStyle`/`ensureScript` and figure wrapping are not idempotent - `overlay.go:757-826` - 13

### B - browser extension (`extension/`)

- B1 - med - conf - pdf.js documents and loading tasks are never destroyed - `src/viewer.js:142,1083` - 15
- B2 - med - conf - the `scheduleFit` stop function is discarded, so window listeners and observers accumulate for every OCR'd image - `src/ocr-plates.js:96,155-178` - 15
- B3 - med - conf - the offscreen OCR host never closes, because finished runs stay in `runs` with `hostKind = offscreen` - `src/page-ocr.js:111,196-198` - 15
- B4 - med - conf - a rejected Tesseract worker promise is cached forever - `src/ocr-overlay.js:44-64` - 15
- B5 - med - conf - the page agent's rAF loop never stops while anchors exist - `src/page-agent.js:155-167` - 15
- B6 - low - conf - the page agent's `anchors` Map pins detached images - `src/page-agent.js:48,96,344` - 15
- B7 - low - conf - the ImageBitmap decoded to read an image's size is never closed - `src/ocr-overlay.js:108-109` - 15
- B8 - low - conf - big files are held twice, blob URLs live until teardown, and the export runs `toDataURL` synchronously for every image - `src/viewer.js:414,581-599,823-825,1051` - 15
- B9 - low - conf - `rasterizePage` has no canvas size cap - `src/pdf-images.js:202-209` - 15
- B10 - med - conf - a stale URL or EPUB load can overwrite a newer document and leak its `revoke` - `src/viewer.js:821-837,1194-1217,1385-1398` - 15
- B11 - low - conf - old-document image extraction and OCR corrupt the new document's OCR counters - `src/viewer.js:275-296,386-428` - 15
- B12 - med - conf - Stop in one tab kills another tab's job on the shared offscreen host (a module-level `stopped` flag) - `src/ocr-host.js:30,41,65`, `src/page-ocr.js:239` - 15
- B13 - med - plaus - `forgetTab` never settles the pending job (a 150 s hang), SPA navigation orphans the agent, and 20 s pings keep the SW alive - `src/page-ocr.js:293-309`, `src/page-agent.js:274-277` - 15
- B14 - med - conf - the sanitizer keeps `javascript:` hrefs. The CSP protects the viewer, but the HTML export drops it - `src/sanitize.js:17-35`, `src/viewer.js:527-551,647-664` - 16
- B15 - low - conf - `ocr.html`, `ocr-host.html`, all modules and `vendor/*` are web-accessible to `<all_urls>`, and host messages are accepted from any sender - `manifest.json:46-82`, `src/page-ocr.js:258-268` - 16
- B16 - low - conf - remote `src` in documents is kept and re-fetched by OCR with host permissions (beacons, intranet GETs) - `src/epub.js:339,357`, `src/ocr-overlay.js:85-89` - 16
- B17 - low - plaus - DOM clobbering via `name` attributes (`<img name=getElementById>`) - `src/sanitize.js:26-31`, `src/epub.js:417-423` - 16
- B18 - med - conf - ids are prefixed but `href="#.."` is not, so in-page links break for HTML, MD and MOBI - `src/sanitize.js:30` - 16
- B19 - med - conf - EPUB spine and manifest lookups do not percent-decode hrefs, so chapters are dropped - `src/epub.js:456,461` - 02
- B20 - low - conf - `resolvePath` mishandles a root-relative `/x` and a decoded `%23` - `src/epub.js:96-105` - 02
- B21 - med - conf - RTF `\uN` with a `\'XX` fallback duplicates the character, and negative values are dropped (Go has a different bug in the same spot, X1) - `src/rtf.js:278-283` - 09
- B22 - low - conf - the TAR reader ignores GNU `L` long names and PAX `path=` (drift from Go `archive/tar`) - `src/comic.js:294-318` - 16
- B23 - med - conf - no zip-bomb or size caps on EPUB/CBZ (undocumented drift from the Go caps) - `src/epub.js:46-89`, `src/comic.js:279-283` - 11
- B24 - low - conf - the DNR regex also matches `.pdf` inside a query string - `src/background.js:131,146` - 16
- B25 - low - plaus - `syncRules` has no catch, concurrent calls apply stale options, and an IPv6 host fails the atomic update - `src/background.js:263-269,319` - 15
- B26 - low - plaus - the viewer's `fetch` sends no credentials, so a login-protected PDF gets 401 or HTML - `src/viewer.js:821`, `src/ocr-overlay.js:87` - 16
- B27 - low - conf - diagnostics keep a stale error, and unawaited read-modify-writes clobber each other - `src/diagnostics.js:436`, `src/viewer.js:848,855` - 15
- B28 - low - conf - "Original PDF" opens the page-load URL after another file is picked - `src/viewer.js:463-472,1065` - 15
- B29 - low - conf - the build ships `eng.traineddata` with no SHA-256 pin and no timeout - `extension/build.mjs:116-124` - 16

### Q - static checks

- Q1 - low - `browser.normalizeTarget` is reported unused on non-Windows (U1000); it is used only on the Windows side - `internal/browser/browser.go:12` - 17
- Q2 - low - an error string ends with punctuation (ST1005) - `internal/comic/readers_sevenzip.go:54` - 17
- Q3 - low - a raw U+200F in a string literal (ST1018) - `internal/i18n/i18n_cli.go:158` - 17
- Q4 - low - a capitalized error string (ST1005) - `internal/mobi/extract.go:44` - 17
- Q5 - low - an error string ends with punctuation (ST1005) - `tools/ocrlab/cmd_add.go:51` - 17
- Q6 - low - `TestPdfTitle` fails on non-Windows because a Windows path is hard-coded - `internal/pdf/extract_test.go:175` - 12
- Q7 - low - on non-Windows, PDF tries `pdftotext.exe` from the cache and suggests `winget` - observed in the test log - 12

Clean: `go vet` passes on linux and `GOOS=windows`, and `go test -race` passes apart from Q6.

## Checked and found sound

These were checked and need no ticket:

- The zip-slip guard on entry names.
- Tar symlink skip.
- Per-entry zip/tar caps.
- Natural sort.
- EXIF bounds checks.
- OCR plate escaping and coordinate guards.
- Worker-pool channel shutdown.
- Response bodies closed.
- The GUI binds `127.0.0.1`.
- The GUI drop name uses `filepath.Base`.
- The report open/reveal confinement.
- The i18n key coverage.
- Extension listeners registered at top level.
- `return true` on async `sendResponse`.
- TOC and titles rendered via `textContent`.
- The viewer frame guard.
