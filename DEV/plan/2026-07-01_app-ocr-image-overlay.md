# App + GUI: OCR text overlay on document images

**Status:** Implemented - OCR engine path verified; full document/GUI flow pending manual test
**Priority:** 50
**Date:** 2026-07-01

> All 5 phases implemented and committed (build bc72494). Gate green (test + lint + typos); full
> `go test ./...` passes. Tesseract 5.4.0 installed locally (winget UB-Mannheim, added to user PATH);
> a throwaway smoke test drove ocr.Recognize on a generated image and correctly returned "Hello World"
> with sane bbox/lineH - so the tesseract shell-out, TSV parse, and block grouping are verified against
> the real binary. Still to verify by hand: the full pipeline on a real EPUB/PDF with image text
> (overlay placement + translation) and the GUI Image-OCR section / language download.

Bring the browser-extension's OCR-overlay feature to the main Go app (`doc-html-translate`) and its GUI
(`doc-html-ui`): an opt-in option that, while converting a document to HTML, recognizes text baked into
the document's images and overlays it as real, translatable HTML "frames" positioned over each image -
so the app's own translation (Google/Ollama) or the browser's "Translate page" translates the picture
text too, in place. Mirrors `DEV/plan/2026-07-01_ocr-image-overlay/` (the extension feature).

## Decisions (owner-confirmed)

- **OCR engine:** Tesseract CLI (free/local). Shell out to `tesseract`, parse its TSV output for
  per-word/line/block bounding boxes.
- **Language model (like the extension):** English (`eng.traineddata`) ships with the app; other
  languages are offered for on-demand download (CLI command + GUI buttons), cached in a local tessdata
  directory.
- **Overlay style:** opaque plates over the source text, positioned in percent of the image's natural
  size (same as the extension).
- **Scope:** formats whose images exist on disk at HTML stage - EPUB and PDF. Other formats are text-only
  (no-op).

## Constraints

- Tesseract binary is located at runtime: `DOCHT_TESSERACT` env -> app-dir `tesseract\tesseract.exe` ->
  PATH. If absent, a clear error with an install hint; conversion still produces HTML without overlays
  (OCR is best-effort, never fatal to the conversion).
- OCR runs BEFORE the translation step so injected overlay text is translated by the app's own engine
  too (and by Chrome in the free flow).
- Windows-first (matches the app); shell-out and paths use the existing platform split.
- Do not change existing CLI flag semantics; the feature is entirely behind a new opt-in flag.

## Phases

### Phase 1 - `internal/ocr` package (engine + tessdata) [DONE]
- `internal/ocr/tesseract.go`: `Locate()`, `Recognize(imgPath, lang, dataDir) (Result, error)` where
  `Result{ Width, Height int; Blocks []Block }`, `Block{ Text string; X0,Y0,X1,Y1 int }`. Parse TSV.
- `internal/ocr/tessdata.go`: `DataDir()`, `Installed()`, `Available` catalog, `Download(lang)` from
  tessdata_fast, `Bundled = ["eng"]`.
- `internal/ocr/tsv_test.go`: unit-test TSV parsing + block grouping (no tesseract needed).
- **Done when:** `go test ./internal/ocr/...` passes; `go build ./...` green.

### Phase 2 - Overlay HTML injection [DONE]
- `internal/ocr/overlay.go`: `OverlayFile(htmlPath, baseDir, lang, dataDir) (n int, err error)` -
  parse the page, find `<img>`, resolve each image file, OCR it, wrap the img in a positioned container
  with opaque text plates (percent positions), write back. Pure helpers (`percentBoxes`) unit-tested.
- `internal/htmlgen/navbar.go`: add overlay CSS (container + plate) to the injected stylesheet.
- **Done when:** `go test ./internal/ocr/...` passes; `go build ./...` green.

### Phase 3 - Pipeline + flags + language subcommands [DONE]
- `internal/config/flags.go`: add `OCR bool` (`-ocr`), `OCRLang string` (`-ocr-lang`), plus management
  flags `OCRList bool` (`-ocr-langs`) and `OCRDownload string` (`-ocr-download <lang>`).
- `internal/app/app.go`: handle `-ocr-langs` / `-ocr-download` as early management commands (like
  `-register`), then normal flow.
- `internal/pipeline/pipeline.go`: after content pages exist and before translation, when `cfg.OCR`,
  run `ocr.OverlayFile` over each content page (best-effort, logged).
- **Done when:** `go build ./...` green; `-ocr-langs` lists eng.

### Phase 4 - GUI [DONE]
- `cmd/doc-html-ui/ui.html`: an "OCR overlay on images" checkbox + a language row (installed languages +
  Download buttons), persisted with the other settings.
- `cmd/doc-html-ui/main.go`: `runRequest` gains `ocr` + `ocrLang`; `assembleArgs` passes `-ocr` /
  `-ocr-lang`; a small `/api/ocr-langs` + `/api/ocr-download` endpoint drives the language buttons.
- **Done when:** `go build ./...` green.

### Phase 5 - Build + docs [DONE]
- Build scripts ship `eng.traineddata` next to the exe (tessdata dir).
- README / AGENTS: document `-ocr`, `-ocr-lang`, language downloads; DEV/CHANGELOG entries.

## Done criteria (manual, needs tesseract installed)
1. `-ocr` on an EPUB/PDF with image text overlays translatable plates over the images; the app's
   translation (or Chrome) translates them.
2. English works with the bundled data; downloading another language makes it usable.
3. Without tesseract installed, conversion still completes (HTML without overlays) and prints a hint.
4. GUI exposes the toggle + language buttons and drives the CLI.
