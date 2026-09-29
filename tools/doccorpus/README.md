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

`run.mjs` flags: `--case <id>` / `--class <class>` (repeatable), `--split dev|holdout|all`,
`--ocr default|on`, `--cli <exe>` (default `temp/doccorpus/bin/doc-html-translate.exe`, build it with
`go build -o temp/doccorpus/bin/doc-html-translate.exe ./cmd/doc-html-translate`), `--run <dir>`,
`--budget <seconds>` per case. Each pass lands in `<run>/<edition>-<ocr mode>/<case>/`.

A full run takes about an hour on the owner's machine; start it detached from anything with a
short timeout.

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
