# Desktop OCR: TSV output independent of the tessdata configs dir

**Status:** Implemented
**Priority:** 1
**Date:** 2026-07-02

## What / why

The desktop app's OCR silently produced no overlay whenever it recognized with its **bundled** tessdata
directory. `Recognize` asked tesseract for TSV by passing the `tsv` **config file**, which lives in
`<tessdata>/configs/tsv`. But the app ships only `eng.traineddata` in `<exe>/tessdata` (no `configs/`),
and passing `--tessdata-dir <exe>/tessdata` redirects tesseract's config lookup there - so `tsv` is not
found (`read_params_file: Can't open tsv`) and tesseract falls back to **plain-text** output. `parseTSV`
then finds no tab-delimited rows and returns zero blocks, so no text is overlaid. The fix requests the
TSV renderer with `-c tessedit_create_tsv=1` instead, which is independent of any `configs/` directory
and works whether or not `--tessdata-dir` is passed. Found while fixing the OCR PSM parity bug
([2026-07-02_ocr-psm-parity](2026-07-02_ocr-psm-parity.md)); this is a separate defect.

Repro: `C:\TEMP\Becoming A Milf Chapter 1.pdf` p.1 via the Tesseract 5.4.0 CLI against the app's bundled
tessdata dir - `... --tessdata-dir build\tessdata -l eng tsv` emits plain text + "Can't open tsv";
`... -c tessedit_create_tsv=1` emits proper TSV (header + word rows) with no error.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` | `internal/ocr/tesseract.go` `Recognize`: `tsv` config file -> `-c tessedit_create_tsv=1` |
| GUI (`doc-html-ui`) | `[x]` | inherits the pipeline; no flag change |
| MSIX Store app | `[x]` | inherits the GUI - this is the edition most affected (always uses bundled tessdata) |
| Browser extension | `[ ]` | Not applicable: tesseract.js returns structured blocks/lines/words via its JS `recognize()` API, not via a CLI config file, so it never had this bug |
| Website / docs | `[ ]` | Declined: internal correctness fix, no user-facing copy |

## Shared invariants touched

None. No constant/default/palette/host change; no new shared value. Behaviour on the system-tesseract
path (no `--tessdata-dir`) is unchanged - both the old config file and the new `-c` flag produce the
same TSV there.

## Cross-references

- Go: [`internal/ocr/tesseract.go`](../../internal/ocr/tesseract.go) `Recognize` (args), `parseTSV`.
- Related: [`internal/ocr/tessdata.go`](../../internal/ocr/tessdata.go) `DataDir` (the bundled dir with
  no `configs/`), [`internal/pipeline/pipeline.go`](../../internal/pipeline/pipeline.go) (passes
  `ocr.DataDir()` into `OverlayFile` -> `Recognize`).

## Done criteria

- [x] `internal/ocr/tesseract.go` requests TSV via `-c tessedit_create_tsv=1`.
- [x] Go green (`go build ./...`, `go vet`, `go test ./internal/ocr ./tests`); `gofmt` clean.
- [x] End-to-end CLI check: TSV produced both with `--tessdata-dir <bundled>` and without (system data).
- [x] Changelog entry in `DEV/CHANGELOG.md`.
- [ ] Owner to confirm on the real EXE (open an image-only PDF with OCR on; overlay plates should appear).
