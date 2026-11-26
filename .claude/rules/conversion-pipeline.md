---
paths:
  - "internal/**"
  - "cmd/doc-html-translate/**"
  - "docs/cli-commands.json"
  - "tools/clidoc/**"
---

# Conversion pipeline and its behaviour

Moved verbatim from `AGENTS.md` / `CLAUDE.md` (2026-11-25); the short core there still binds. A section
name quoted below is a section of `AGENTS.md` unless it says otherwise.

## Where to start, run order, command front (from CLAUDE.md)

Trace from [internal/pipeline/pipeline.go](../../internal/pipeline/pipeline.go) into the format package
(`internal/epub`, `internal/pdf`, ..). Entry points: [cmd/doc-html-translate/main.go](../../cmd/doc-html-translate/main.go),
app wiring in [internal/app/app.go](../../internal/app/app.go) and [internal/config/flags.go](../../internal/config/flags.go).

One run, in order (`Runner.run` / `Runner.build` in the pipeline file): reuse check against the completion
record -> `.fd-sec` unwrap -> format extractor -> JPEG XL normalize -> optional page split -> navigation and
reader injection (`internal/htmlgen`) -> OCR overlay -> translation -> TOC, index and search index ->
completion record -> open the browser. Every extractor, whatever the format, returns the same `*epub.Book`
(spine + TOC), and every later stage works on that one model. The order carries meaning: navigation and the
OCR overlay go in before translation so that their text is translated too, the TOC is built after it so that
its labels are translated, and the completion record is the last write - an interrupted run leaves none and
is rebuilt.

The verb-style command line (`convert`, `doctor`, `input check`, `output status`, ..) is `internal/commands`,
entered through `commands.Front` by both binaries before the classic flags; its manifest is
`docs/cli-commands.json` and [docs/CLI.md](../../docs/CLI.md) is the render, never edited by hand.

## Image, comic and HTML packages

  - internal/img: standalone image input (PNG/JPG/WebP/..) - wraps the picture in a one-page HTML doc; the pipeline forces the OCR overlay so the image gets translatable text plates (see internal/ocr)
  - internal/comic: comic archives (CBZ/CBR/CB7/CBT) - one page image per spine entry in natural filename order, forced OCR (same rationale as internal/img). CBZ=zip, CBT=tar (stdlib); CBR/CB7 shell out to 7-Zip (LookPath + probe paths, the MOBI/Calibre precedent)
- HTML processing/generation:
  - internal/htmlproc, internal/htmlsplit, internal/htmlgen
  - internal/assets: local asset copying shared by internal/htmlconv and internal/md (img src/srcset, picture sources, CSS url() and local stylesheets); names unique case-insensitively, generated names reserved, nothing resolved outside the source folder is copied
  - EPUB chapters are normalized on the parsed tree (internal/epub normalize.go, links.go `rewriteLinks`), never by text replacement; the splitter retargets TOC entries and links through an id-to-part map

## External helpers

- External helpers (pdftotext, Tesseract, Calibre, 7-Zip, ffmpeg/ImageMagick, FileDO):
  - internal/procrun: the one way to run a helper - per-tool size-scaled deadline (`DOCHT_TOOL_TIMEOUT_SCALE` multiplies it), process-tree kill (job object on Windows, process group elsewhere), capped output, one error type naming the tool. Every call takes the run's context, so Ctrl+C stops a helper at once and the run ends as interrupted. Do not call exec.Command for a helper directly.
  - internal/readiness: what this machine can do, as data (ticket E0146) - `Check` is the window's pre-flight for one document (the GUI `/api/readiness` handler and the command `input check` both call it, so they cannot disagree) and `Diagnose` is `doctor` (helper tools found with version numbers, input formats and the tool each needs, OCR languages, whether a Google key is stored; no path, no key, no service asked unless `--ollama`). `Probes` are the same discovery functions a conversion uses, swapped by tests.
  - internal/bundledtools: the Windows-only bundled pdftotext, unpacked into a content-hash-named cache folder. Nothing is ever installed on the user's machine (tests/no_auto_install_test.go).
  - internal/fdsec: FileDO secret files (.fd-sec) - screened by length, decrypted by the installed FileDO (`DOCHT_FILEDO` -> PATH -> setup folder -> winget Links -> Store alias) into one private copy under the user's temp folder, password only in the child's environment, FileDO's output never forwarded (read: the banner build and, beside exit 5, the exact stderr line `reason=destination-name`, which gets its own message); the container stays the run's identity and the plain copy is gone as soon as the extractor has read it (pointer [docs/contracts/FDSEC.md](../../docs/contracts/FDSEC.md); `internal/pipeline/container.go` is the pre-stage, `cmd/doc-html-ui/fdsecanswer.go` the window's masked-password answer). A new GUI file named `*secret*` is gitignored - name it otherwise.

## Output reuse, cost guard, reader layer

- Idempotent output reuse: an output is reopened only when its completion record (internal/outputpath completion.go, written as the run's last step) matches the source identity and the result-affecting options; anything else is rebuilt with a logged reason, and -force always rebuilds. Pipeline page writes go through internal/fsutil (temp file + rename), never an in-place truncating write.
- Paid engines respect -max-cost: the estimate (characters, not bytes, over pages + title + TOC labels, /1e6*$20) is enforced as a pre-flight guard in internal/pipeline/cost.go before any request is sent. A set limit pre-approves spending up to it (no dialog); without one the dialog asks above 1000 characters, defaulting to "do not spend" (OK/Cancel, Cancel default; under the GUI the question travels over the child's stdio - internal/dialog HostStdio - and is asked in the GUI's window). The Google key travels in the X-Goog-Api-Key header, never the URL.
- The output HTML carries a client-side reader layer injected by internal/htmlgen (navbar.go readerScript/readerCSS on chapter pages, plus a matching toolbar/script on index.html). Keep storage scopes distinct: fit-width uses sessionStorage; reading themes and reading position use localStorage (must survive sessions). Reading position is namespaced by `epub.Book.ReaderKey` (htmlgen.ReaderKey: source file name + size, original title, page count), set once in the pipeline before translation and read by every page and index.html - never recompute it from the (translated) title. A saved position is restored only when the URL has no fragment. Values enter generated scripts only through `jsString`, paths enter hrefs only through `epub.URLPath`.
