# The discard record is not written for the image it exists for

**Status:** Draft
**Priority:** 50
**Date:** 2026-09-22

> Cross-edition ticket. One feature = one ticket covering every edition.
> Read [`docs/PARITY.md`](../../docs/PARITY.md) before starting; update it when a shared invariant moves.
> Contract: `OCR-OVERLAY rule 12`, pointer [`docs/contracts/OCR-OVERLAY.md`](../../docs/contracts/OCR-OVERLAY.md).

## What / why

`OCR-OVERLAY rule 12` asks that every gate which drops a line record the text, the confidence, the box and
**which** threshold it failed, through the same predicate the pipeline applies - and that it record it
**also for an image that produced nothing at all**, "which is the case it exists for". Without that record,
"the engine found nothing" and "we threw four lines away" are the same line in a bug report.

Both editions compute the discard set correctly and through the shared predicate. Neither writes it for a
blank image.

**Go.** The plumbing goes all the way and stops one step short. `recognizePaths` deliberately keeps the
dropped lines for the no-plate case - [`internal/ocr/overlay.go:419`](../../internal/ocr/overlay.go#L419),
whose own comment says it exists "so the diagnostics can say *why*" - but `applyOverlays` writes
diagnostics only in the arm that drew plates
([`internal/ocr/overlay.go:242-251`](../../internal/ocr/overlay.go#L242-L251)): the `!r.ok` arm counts
`NoText` and returns. Measured with a throwaway probe in package `ocr` on 2026-09-22 (a fixture image, one
dropped line at confidence 61.5 against a floor of 80, `DOCHT_OCR_DIAG` set):

```
applyOverlays: changed=false NoText=1 Overlaid=0 Failed=0
PROBE RESULT: no diagnostics file at all for a no-plate image
```

**Extension.** `droppedLines` is computed and travels with the result through `ocr-overlay.js`, and nothing
consumes it: no module outside `ocr-cluster.js` / `ocr-overlay.js` reads `dropped`, and
`extension/scripts/ocrlab.mjs` does not either. The edition has no discard surface at all.

This also corrects a claim in this repo's own ledger: `DEV/plan/done/2026-08-15_release-1-worklog.md` §1.3 states that the
record "is written **even for a page that produced no plates**, the case that used to write nothing". What
landed was the preservation of the record up to `applyOverlays`, not its writing.

## Done criteria

- [ ] With `DOCHT_OCR_DIAG` set, an image that produced no plates writes one diagnostics line carrying its
      size and its `dropped` array (empty when the engine genuinely read nothing - "read fine, found no
      text" and "everything was thrown away" must be distinguishable in the file).
- [ ] A test in `internal/ocr/` fails if that line stops being written - the probe above, kept.
- [ ] The extension writes the same record from its lab harness, or `docs/PARITY.md` records a named
      divergence with the reason the browser edition cannot.
- [ ] `DEV/plan/done/2026-08-15_release-1-worklog.md` §1.3 is corrected in the entry that lands this, rather than rewritten in
      place.
- [ ] The exception row for `OCR-OVERLAY` rule 12 in the catalog registry is closed, not re-dated.

## Notes

Behaviour at a contract boundary does not change here: the overlay a reader sees is identical either way,
and the diagnostics file is off unless the environment variable is set.
