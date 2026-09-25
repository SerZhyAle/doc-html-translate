# The discard record is not written for the image it exists for

**Status:** Draft
**Priority:** 50
**Date:** 2026-09-22

> Cross-edition ticket. One feature = one ticket covering every edition.
> Read [`docs/PARITY.md`](../../docs/PARITY.md) before starting; update it when a shared invariant moves.
> Contract: `OCR-OVERLAY rule 12`, pointer [`docs/contracts/OCR-OVERLAY.md`](../../docs/contracts/OCR-OVERLAY.md).

> **Remote execution (2026-09-25):** the contract text this ticket needs is quoted in "Contract snapshot" below, so every step not marked ⛔ runs in a cloud session from this repository alone. Steps marked **⛔ Local only** edit the shared contracts catalog (or another repository) and can run only on the owner's machine, where the catalog is mounted.

## What / why

`OCR-OVERLAY rule 12` asks that every gate which drops a line record the text, the confidence, the box and
**which** threshold it failed, through the same predicate the pipeline applies - and that it record it
**also for an image that produced nothing at all**, "which is the case it exists for". Without that record,
"the engine found nothing" and "we threw four lines away" are the same line in a bug report.

Both editions compute the discard set correctly and through the shared predicate. Neither writes it for a
blank image.

**Go.** The plumbing goes all the way and stops one step short. `recognizePaths` deliberately keeps the
dropped lines for the no-plate case - [`internal/ocr/overlay.go:384-390`](../../internal/ocr/overlay.go#L384-L390),
whose own comment says it exists "so the diagnostics can say *why*" - but `applyOverlays` writes
diagnostics only in the arm that drew plates
([`internal/ocr/overlay.go:204-226`](../../internal/ocr/overlay.go#L204-L226)): the `!r.ok` arm (line 213)
counts `NoText` and returns. Line numbers re-verified 2026-09-25 - the file was reshaped by `6b952c2`
since the 2026-09-22 reading (it was lines 419 and 242-251 then, which the registry row below still
quotes); the gap itself is unchanged: `recordDiagnostics` ([`internal/ocr/diag.go:73`](../../internal/ocr/diag.go#L73))
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

- [ ] With `DOCHT_OCR_DIAG` set, an image that produced no plates writes one diagnostics line carrying its
      size and its `dropped` array (empty when the engine genuinely read nothing - "read fine, found no
      text" and "everything was thrown away" must be distinguishable in the file).
- [ ] A test in `internal/ocr/` fails if that line stops being written - the probe above, kept.
- [ ] The extension writes the same record from its lab harness, or `docs/PARITY.md` records a named
      divergence with the reason the browser edition cannot.
- [ ] `DEV/plan/done/2026-08-15_release-1-worklog.md` §1.3 is corrected in the entry that lands this, rather than rewritten in
      place.
- [ ] **⛔ Local only - changes the contract catalog.** The exception row for `OCR-OVERLAY` rule 12 in the
      catalog registry (quoted below) is closed, not re-dated - after the three repo criteria above are met.
      The same change corrects the two catalog places that state the same deviation, or they contradict the
      closed row: this product's `OCR-OVERLAY` adoption row in the registry ("Two deviations ..") and the
      doc-html-translate row of section 8 in `ocr-overlay/README.md` (with a document-log row).

## Contract snapshot (2026-09-25)

Quoted from the catalog on 2026-09-25 - OCR-OVERLAY 1.0. A working copy for executing this ticket without the catalog, not a second source: the catalog stays authoritative, and this section is deleted when the ticket moves to done/.

From the shared contracts catalog, `ocr-overlay/README.md`, section 4 "The rules that keep it honest":

> 12. **A silent decision leaves a record.** Every gate that drops a line records the text, the confidence,
>     the box and **which** threshold it failed - through the *same predicate* the pipeline applies, not a
>     second copy of the condition - and it records it **also for an image that produced nothing at all**,
>     which is the case it exists for. Until that record exists, "the engine found nothing" and "we threw
>     four lines away" are indistinguishable in a bug report. (`ocr-overlay-accuracy.md` §8)

> 16. **Failure is best-effort, never fatal.** A missing engine, an unreadable image or a failed page never
>     aborts the surrounding run. "Read fine, found no text" is tracked apart from "recognition failed", in
>     every product. (`ocr-pipeline.md` §4)
>
> 17. **A developer can see the source box and the final plate without changing the data.** Diagnostics are
>     recomputed with the same pure functions the renderer used, so turning them on cannot change what is
>     rendered. (exchange §5.6, `ocr-pipeline.md` §7)

From the same file, section 6 "Conformance" (the first two rungs - the ones this ticket touches):

> 1. **The discard record of rule 12** - the instrument. Everything else is measured with it.
> 2. **A per-image diagnostic line**: image size and, per block, the text, the box, the line height, the
>    resolved style and the sampled colours. One JSON line per overlaid image, behind an environment
>    variable or a developer setting.

The exception this ticket closes, from the shared contracts catalog, `_meta/REGISTRY.md`, section 3
"Exceptions" (columns: Contract | Product | Deviation | Reason | Until):

> | `OCR-OVERLAY` | doc-html-translate | rule 12: the discard record is not written for an image that produced **no** plates - the case the rule exists for. The Go edition keeps the dropped lines as far as `applyOverlays` (`internal/ocr/overlay.go:419`) and then writes diagnostics only in the arm that drew plates (`overlay.go:242-251`); the extension computes `droppedLines` and no module consumes it | found by reading the code against the rule on 2026-09-22 and proven with a throwaway probe in package `ocr` (`applyOverlays: changed=false NoText=1`, no diagnostics file written). The repo's own ledger claimed this case was covered, so the gap was invisible from the inside; what landed earlier was the plumbing, not the last write. No user-visible behaviour is involved - the file is off unless `DOCHT_OCR_DIAG` is set - so it is scheduled rather than hot-fixed. Ticket `2026-09-22_ocr-discard-record-missing-for-blank-images` | 2026-12-31 |

Re-verified 2026-09-25: the row is still open (until 2026-12-31), and the section 8 row of
`ocr-overlay/README.md` still names the deviation ("rule 12's discard record is written only for an image
that produced plates, not for one that produced none").

## Notes

Behaviour at a contract boundary does not change here: the overlay a reader sees is identical either way,
and the diagnostics file is off unless the environment variable is set.
