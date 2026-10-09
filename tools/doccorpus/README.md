# doccorpus - the cross-edition document corpus

The instrument of [ticket 68](../../DEV/plan/68_2026-09-29_multilingual_comic_book_verification.md):
six languages (en, fr, ru, ja, zh, ko) times six input classes (comic image, EPUB, text PDF, scanned
PDF, illustrated PDF, comic archive), each document run through **both** editions from the same
hashed bytes and graded against an expectation that neither edition wrote.

**Not shipped.** No build script compiles it and it is in no package.

## Commands

Run from the repository root, in PowerShell (Node and Go are on PATH there).

| Command | What it does |
|---|---|
| `go run ./tools/doccorpus verify` | Validate `DEV/doccorpus/cases.json`, hash every source, print the coverage matrix and the human-review counts. Exit 1 on a manifest problem or changed bytes, 2 (COULD NOT VERIFY) when media is missing. |
| `go run ./tools/doccorpus draft [-force] [id..]` | Write draft expectations into `DEV/doccorpus/expect/` from the sources' own bytes. Never overwrites a reviewed one. |
| `node tools/doccorpus/run.mjs --edition windows` | Convert each case with the desktop CLI, open the generated page in headless Chrome, record the probe, text, DOM, log and screenshots. |
| `node tools/doccorpus/run.mjs --edition extension` | Load the unpacked extension and open the same bytes in its viewer; same probe, same records. |
| `go run ./tools/doccorpus report <run-dir>` | Grade every result and write `report.md` + `report.json` into the run. |
| `go run ./tools/doccorpus verdict <case-folder>` | Grade one case folder with today's judge and print the verdict as one JSON line. `run.mjs` asks it before reusing a case. |

`run.mjs` flags: `--case <id>` / `--class <class>` (repeatable), `--split dev|holdout|all`,
`--ocr default|on`, `--cli <exe>` (default `temp/doccorpus/bin/doc-html-translate.exe`, build it with
`go build -o temp/doccorpus/bin/doc-html-translate.exe ./cmd/doc-html-translate`), `--run <dir>`,
`--budget <seconds>` per case, `--fresh`, `--reuse-from <dir>`. Each pass lands in
`<run>/<edition>-<ocr mode>/<case>/`.

A full run takes about an hour on the owner's machine; start it detached from anything with a
short timeout.

## Reusing an unchanged case

A long case costs up to half an hour, so a case is not collected again when its inputs are identical
to an earlier run that settled and passed. `run.mjs` computes a **reuse key** per case and records it
(with the sizes of the case folder's files) in `result.json` as `reuse`; a later run copies the
newest earlier case folder with the same key instead of rendering it again.

| Key component | |
|---|---|
| source | the case's file hash, plus its file, class, language and effective OCR language |
| edition and OCR mode | `windows` / `extension`, `default` / `on` - the two editions' keys cannot collide |
| producer digest | every non-test byte of `internal`, `cmd/doc-html-translate`, `extension/src` and `tools/doccorpus` (test files and `testdata` excluded) |
| browser and tools | the browser build; for `windows` the CLI's version and **exe bytes**, Tesseract, the installed pack's hash and the bundled helper; for `extension` its version and tesseract.js |
| expectation | the hash of the case's `DEV/doccorpus/expect/<id>.json` |
| platform | `process.platform/arch` |

Reuse also needs `result.collection.outcome === "settled"`, no error, the case folder still holding
exactly the files and sizes recorded when it finished, and **today's judge** (`doccorpus verdict`)
answering `PASS` or `PASS WITH ADVISORIES`. An unsettled, timed-out, errored, failing or
could-not-verify case, a folder that was edited, and a result from before this feature are collected
again. Only the collection is reused: the report still grades every result.

The copied `result.json` carries `reusedFrom` (`caseDir`, `keyDigest`, `producer`, `collectedAt`,
`reusedAt`, `verdict`); `run.mjs` logs `reused <case> from <folder>` and ends with `collected N,
reused M`, and `report.md` marks reused cells with `*` and lists them. `--fresh` collects every case
(its results are still recorded under their key). `--reuse-from` takes one run directory or a folder
of runs; the default is `temp/doccorpus`.

## Layout

```text
DEV/doccorpus/cases.json        versioned case records - metadata, rights, hashes, split
DEV/doccorpus/expect/<id>.json  versioned expectations (drafts until a person reviews them)
test_doc/                       the media - gitignored; test_doc/INVENTORY.json holds the measured facts
temp/doccorpus/<run>/           result.json, convert.log, text.txt, plates.txt, dom.html, *.png, report.md
```

The case record is a layer beside the OCR lab's scene manifest (`DEV/ocrlab/corpus.json`), not a
second account of the same asset: a case that is also an OCR scene names it in `ocrScene`, and the
lab keeps its own annotations and scoring.

The image lab's [declared evidence and review workflow](../ocrlab/README.md) keeps machine provenance,
annotation drafts and human approvals distinct. Legacy `commons-api` licence stamps cannot supply
a human rights decision. An image lab `selected-dev` verdict does not substitute for this campaign's
document-level expectations, lazy OCR, navigation checks or per-edition campaign verdict. Existing
shared CDP support remains in place; ticket 109 does not remove a document runner without equivalent
evidence.

## Verdicts

Each case gets one verdict per edition and mode:

- `auto` judges only what the bytes and the rendered page show - conversion failure, a notice
  instead of the document, broken images, dangling in-page links, lost pages, gross text loss
  against the source's text layer, a wrong document language, no plate on a page the author
  lettered.
- `campaign` also requires both human gates - a rights review and a reviewed expectation - and
  stays `COULD NOT VERIFY` until both are closed. A `FAIL` stands either way.

Missing or changed source bytes, an OCR language that is not offered or not installed, a probe
that ran out of budget: `COULD NOT VERIFY`, never a pass.

The two text-survival bounds in `judge.go` (under 50 % of the source text layer is a failure,
under 85 % an advisory) are gross-damage detectors set on 2026-09-29 for run 1, not quality
thresholds. OCR text quality is reported as a measurement (`letteringRecall`) with **no** bound:
a bound needs a dated baseline and a measured reason, which the campaign has not produced yet.

## When the collector calls a page finished

A page is not finished because the DOM stopped changing: the extension renders a PDF in chunks of
100 pages built off-DOM, so the tree is frozen at a chunk boundary while the status bar still reads
`Rendering page 13 / 184`. `collect.mjs` (pure, tested without a browser in
`extension/test/doccorpus-collect.test.mjs`) classifies every probe into one stage, first match wins:

| Stage | Meaning |
|---|---|
| `starting` | Reader not ready, or a loader status (`Downloading..`, `Reading EPUB..`) with nothing on the page yet. |
| `rendering-active` | Status `Rendering page n / N`, or fewer page sections than the page total, or pictures still decoding. |
| `awaiting-scroll` | Status `Pages 1-K of N - keep scrolling`: the next chunk builds when the page is scrolled. |
| `extracting` | Page-image extraction still pending (`data-pdf-page` / `.pdf-page-pending`). |
| `ocr-active` | `OCR: done/total` below the total, a `.ocr-pending` wrapper, or `Recognizing text..`. |
| `settled` | None of the above. |
| `producer-error` | The viewer replaced the document with a notice heading (the "Little or no text" banner is a result, not an error). |

`collect` reports `settled` only when the stage is `settled`, the scroll is at the bottom and the
page signature held for three polls. Otherwise the run ends with `outcome` `timeout` (budget spent)
or `producer-error`, written to `result.json` as `collection`:
`{outcome, stage, rendered, total, ocrDone, ocrTotal, extractionPending, polls}`. `truncated` stays
and equals `outcome !== "settled"`.

**An unsettled run is incomplete evidence about the collector, never a product failure.** The judge
reports `incomplete evidence: stage S, a/b` (rendered/total pages, OCR done/total, or extracted/rendered
pages, by stage) as `COULD NOT VERIFY` and does not grade page loss, the OCR-plate check or the
partial-text counts on it. A settled run with missing pages still fails. Re-run the case with a larger
`--budget` to get a verdict.

The status literals the classifier matches are listed in `VIEWER_STATUS`; a test asserts each one still
exists in `extension/src/viewer.js` and the English catalog, so a reworded status fails a test instead of
quietly bringing back the early "settled".

## The rules a contributor must not break

1. **No edition grades itself.** A draft expectation comes from the source's own bytes - a PDF's
   text layer, an EPUB's XHTML, the author's lettering files - never from either edition's output
   and never from a weak text layer (someone else's OCR). OCR output never becomes truth.
2. **A person signs the gates.** `rights.reviewedBy` / `reviewedOn` and an expectation's
   `origin: "human"` + `reviewedBy` / `reviewedOn` are filled by hand after opening the original.
   No code path writes them.
3. **One file fills one cell.** The manifest check refuses the same file under two ids.
4. **A missing input is not a smaller test.** Missing media, a missing language pack or a
   truncated probe is recorded and reported, never skipped.

## Traps, already handled

- A whole-document reply over the DevTools socket (a CJK book's DOM, every character escaped)
  drops the connection and looks like a browser crash; long strings are read in slices.
- The viewer's OCR queue is fed by an `IntersectionObserver`, so the extension pass scrolls in
  viewport steps; the desktop page only lazy-loads and is walked in larger steps.
- The desktop's index for an EPUB is a `location.replace` stub; the probe waits for the real page.
