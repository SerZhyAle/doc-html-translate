# The discard record is not written for the image it exists for

**Status:** Implemented (2026-09-25; repo criteria and the catalog step both done)
**Priority:** 50
**Date:** 2026-09-22

> Cross-edition ticket. One feature = one ticket covering every edition.
> Read [`docs/PARITY.md`](../../../docs/PARITY.md) before starting; update it when a shared invariant moves.
> Contract: `OCR-OVERLAY rule 12`, pointer [`docs/contracts/OCR-OVERLAY.md`](../../../docs/contracts/OCR-OVERLAY.md).

## What / why

`OCR-OVERLAY rule 12` asks that every gate which drops a line record the text, the confidence, the box and
**which** threshold it failed, through the same predicate the pipeline applies - and that it record it
**also for an image that produced nothing at all**, "which is the case it exists for". Without that record,
"the engine found nothing" and "we threw four lines away" are the same line in a bug report.

Both editions compute the discard set correctly and through the shared predicate. Neither writes it for a
blank image.

**Go.** The plumbing goes all the way and stops one step short. `recognizePaths` deliberately keeps the
dropped lines for the no-plate case - [`internal/ocr/overlay.go:384-390`](../../../internal/ocr/overlay.go#L384-L390),
whose own comment says it exists "so the diagnostics can say *why*" - but `applyOverlays` writes
diagnostics only in the arm that drew plates
([`internal/ocr/overlay.go:204-226`](../../../internal/ocr/overlay.go#L204-L226)): the `!r.ok` arm (line 213)
counts `NoText` and returns. Line numbers re-verified 2026-09-25 - the file was reshaped by `6b952c2`
since the 2026-09-22 reading (it was lines 419 and 242-251 then, which the registry row below still
quotes); the gap itself is unchanged: `recordDiagnostics` ([`internal/ocr/diag.go:73`](../../../internal/ocr/diag.go#L73))
is still called only from the `default` arm. Measured with a throwaway probe in package `ocr` on 2026-09-22 (a fixture image, one
dropped line at confidence 61.5 against a floor of 80, `DOCHT_OCR_DIAG` set):

```
applyOverlays: changed=false NoText=1 Overlaid=0 Failed=0
PROBE RESULT: no diagnostics file at all for a no-plate image
```

**Extension.** `droppedLines` is computed and travels with the result through `ocr-overlay.js`, and nothing
consumes it: no module outside `ocr-cluster.js` / `ocr-overlay.js` reads `dropped`, and
`extension/scripts/ocrlab.mjs` does not either. The edition has no discard surface at all. Re-verified
2026-09-25 by grep over `extension/src` and `extension/scripts`: `droppedLines` / `.dropped` still occur
only in `ocr-cluster.js` and `ocr-overlay.js`; `extension/src/diagnostics.js` does not read them.

This also corrects a claim in this repo's own ledger: `DEV/plan/done/2026-08-15_release-1-worklog.md` §1.3 states that the
record "is written **even for a page that produced no plates**, the case that used to write nothing". What
landed was the preservation of the record up to `applyOverlays`, not its writing.

## Done criteria

- [x] With `DOCHT_OCR_DIAG` set, an image that produced no plates writes one diagnostics line carrying its
      size and its `dropped` array (empty when the engine genuinely read nothing - "read fine, found no
      text" and "everything was thrown away" must be distinguishable in the file).
- [x] A test in `internal/ocr/` fails if that line stops being written - the probe above, kept.
- [x] The extension writes the same record from its lab harness, or `docs/PARITY.md` records a named
      divergence with the reason the browser edition cannot.
- [x] `DEV/plan/done/2026-08-15_release-1-worklog.md` §1.3 is corrected in the entry that lands this, rather than rewritten in
      place.
- [x] **⛔ Local only - changes the contract catalog.** The exception row for `OCR-OVERLAY` rule 12 in the
      catalog registry is closed, not re-dated - after the three repo criteria above are met.
      The same change corrects the two catalog places that state the same deviation, or they contradict the
      closed row: this product's `OCR-OVERLAY` adoption row in the registry ("Two deviations ..") and the
      doc-html-translate row of section 8 in `ocr-overlay/README.md` (with a document-log row).

## Delivered (2026-09-25)

- **Go.** `applyOverlays` writes the record in the `!r.ok` arm too (`recordDiagnostics(job.file, r.res, nil)`);
  `diagImage.Blocks` / `Dropped` always serialize as arrays, so "read nothing" (`[]`/`[]`) and "threw
  everything away" (`[]`/non-empty) differ in the file. `TestDiagnosticsRecordDiscardsForNoPlateImage` and
  `TestDiagnosticsRecordEmptyDiscardsForBlankImage` keep the probe; both fail with the new call removed.
- **Extension.** `overlayImage` leaves `{ width, height, blocks, dropped }` on the container as the
  `ocrRecord` property (not an attribute, so the rendered DOM is untouched); `scripts/ocrlab.mjs` writes it
  to the run's `ocr-diag.jsonl` through `makeDiagRecord` for every scene, plated or not.
- **Parity.** One literal line pins both editions' output; `TestParityOCRDiscardRecord` holds the two
  literals equal and checks both write sites. `docs/PARITY.md` "OCR" rewritten, the "named gap" removed.
- **Ledger.** The release-1 worklog gains a dated correction section; §1.3 itself is left as written.
- **Catalog (owner machine).** In `_meta/REGISTRY.md` the rule 12 exception row is struck through and marked
  `CLOSED 2026-09-25` (until `closed`), and the adoption row's "Two deviations" now says this one is closed
  and the `eng` one stays open. In `ocr-overlay/README.md` section 8 the doc-html-translate row moves the
  blank-image case from "Known deviation" to "Holds", with a 2026-09-25 correction row in the document log.
  `docs/contracts/OCR-OVERLAY.md` now names one open deviation. Evidence re-run before the edit:
  `go test ./internal/ocr/ -run TestDiagnosticsRecord -v` and `go test ./tests/ -run TestParityOCRDiscardRecord -v`,
  both exit 0.

## Notes

Behaviour at a contract boundary does not change here: the overlay a reader sees is identical either way,
and the diagnostics file is off unless the environment variable is set.
