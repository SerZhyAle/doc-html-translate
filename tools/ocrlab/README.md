# ocrlab - the OCR visual-fidelity lab

The instrument required by [`DEV/plan/2026-08-11_ocr-visual-fidelity-lab.md`](../../DEV/plan/2026-08-11_ocr-visual-fidelity-lab.md)
before any OCR or redraw change is accepted. It measures the shipped program: plate geometry comes
from the DOM the app produced and from the app's own diagnostics sidecar, never from a
reimplementation.

**Not shipped.** No build script compiles it, it is in no package, and neither `doc-html-translate`
nor `doc-html-ui` links it.

## Commands

Run from the repository root.

| Command | What it does |
|---|---|
| `go run ./tools/ocrlab verify` | Validate the manifest, media hashes, annotations and the coverage table. Exits 1 on any problem. |
| `go run ./tools/ocrlab fetch [id..]` | Idempotently download licence-verified media. Refuses anything a human has not verified. |
| `go run ./tools/ocrlab synth` | Redraw the deterministic diagnostic scenes and their exact annotations. |
| `go run ./tools/ocrlab seed <id..>` | Write an OCR-seeded annotation draft for a human to correct. Never counts as truth. |
| `go run ./tools/ocrlab run [-split dev\|holdout\|all] [-scene id]` | Convert, render at three viewports, apply the stress cases, record evidence, then score and report. |
| `go run ./tools/ocrlab score <run-dir>` | Grade a saved run offline - no browser, no recognizer. |
| `go run ./tools/ocrlab report <run-dir>` | Render `report.md` and a self-contained `report.html`. |
| `go run ./tools/ocrlab gate [-against <run-dir>] <run-dir>` | Judge a scored run against `DEV/ocrlab/thresholds.json`. Exits 1 on FAIL. |

Flags shared by most commands: `-manifest` (default `DEV/ocrlab/corpus.json`), `-root` (default
`test_doc/ocrlab`), `-annotations` (default `DEV/ocrlab/annotations`).

## The other edition

The browser extension is a separate codebase with its own OCR engine, so it produces its own
evidence and is graded by the same `score` / `report` / `gate` against the same annotations:

```text
cd extension && npm run ocrlab -- --help
```

It writes the identical record (`extension/scripts/_ocrlab-evidence.mjs` mirrors the Go `evidence`
package field for field, and `TestParityOCRLabEvidenceSchema` fails when one side moves alone).
Recognized *text* will differ between the Tesseract CLI and tesseract.js and is meant to - each
edition is compared with the annotations, never with the other's output.

## What it needs

- `tesseract` on `PATH`, next to the app, or at `DOCHT_TESSERACT` - the same resolution the app uses.
- Chrome or Edge. Override with `DOCHT_BROWSER=<path to the executable>`.

A missing dependency is a hard error, never a quietly smaller run.

## Layout

```text
DEV/ocrlab/corpus.json          versioned manifest - metadata only, no media
DEV/ocrlab/annotations/*.json   versioned ground truth
DEV/ocrlab/thresholds.json      acceptance bounds (written by the baseline phase only)
test_doc/ocrlab/                the media - gitignored, ships in nothing
temp/ocrlab/<run-id>/           evidence.json, scores.json, summary.json, shots/, report.html
```

## The three rules a contributor must not break

1. **OCR output never becomes truth.** A seeded draft carries `origin: ocr-seed` and `IsTruth()`
   is false for it, so the engine can never be graded against its own output. Correct a draft by
   hand, set `origin: human`, fill `review.annotatedBy`, and drop the `.draft` from the filename.
2. **No threshold outside a dated report.** The metrics package contains no bound at all, and a test
   parses it to keep that true. Acceptance numbers come from a recorded baseline and cite the value
   they came from, so a retrofit is visible in the diff.
3. **A licence is verified by a person, not by tooling.** `licenceVerifiedBy` is filled by hand after
   opening the asset's own licence page. A search-result snippet or a site's category label is not
   proof, and no code path may write that field.

## Windows traps, already handled

Recorded because each one presents as "the runner hangs" and costs an afternoon to find again:

- `msedge.exe --version` does not print a version and exit - it launches a browser and stays. The
  version is read from the installer's sibling version directory instead.
- `cmd.Stdout` as an `io.Writer` makes `cmd.Wait` block forever: browser helper processes inherit
  the pipe. Output goes to real files.
- A relative `--user-data-dir` hangs headless Edge rather than failing. Absolute paths only, and a
  test guards it.
- Headless helpers outlive the launch, so `Browser.Close` kills exactly the processes whose command
  line names this run's profile directory - never by image name, which would take the user's own
  browser with it.
- **`--dump-dom` and `--screenshot` stopped working** (measured 2026-08-12, Chrome 151 and Edge, in
  every headless mode): zero bytes, no file, no error. The runner speaks the DevTools protocol
  instead. Two traps of that transport: Chrome refuses a debugger WebSocket carrying an `Origin`
  header unless it is allow-listed, and a fragment-only navigation does not reload - so every page
  state is opened via `about:blank`, or the probe reports the previous state's numbers.
