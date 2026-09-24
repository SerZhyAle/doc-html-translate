# Pointer: OCR-OVERLAY

- **Id:** `OCR-OVERLAY`
- **Version:** 1.0
- **Home:** the shared contracts catalog, `ocr-overlay/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** reference implementation - this product both produces the overlay and consumes its own output
- **Other products bound:** FastMediaSorter Android, FastMediaSorter_Lite

The function contract for reading text off an image and drawing text back over it: display-space
recognition, line-level boxes, plate geometry and sampled colour, what is never clipped, what is never
trusted, and how a constant earns its value. Seventeen numbered rules; this repo implements all of them in
two editions (Go and the browser extension) held to one constant table by the guard tests in
[`../../tests/parity_test.go`](../../tests/parity_test.go).

**What this repo owes it**

- Change the contract before the code whenever behaviour at the overlay boundary moves: the catalog commit
  comes first, then both editions.
- Keep every shared constant in step across the two editions, and keep its derivation dated - `OCR-OVERLAY
  rule 13`. The numbers themselves are this implementation's own; they are not facts another product may
  copy.
- Cite rules in source as `OCR-OVERLAY rule N`, never as a path.
- Two deviations are recorded as dated exceptions in the catalog registry, not fixed here in silence: the
  discard record of rule 12 is not written for an image that produced no plates, and the extension edition
  recognizes with an assumed `eng` because it has no script-detection pass (rule 10's failure mode).
  Tickets: [`../../DEV/plan/2026-09-22_ocr-discard-record-missing-for-blank-images.md`](../../DEV/plan/2026-09-22_ocr-discard-record-missing-for-blank-images.md)
  and [`../PARITY.md`](../PARITY.md) ("Intentional divergences").

**Conformance.** The catalog has no shared vector set for this contract yet; section 6 of its README names
the ladder. This product holds the first three rungs - the discard record, the per-image diagnostic line
(`DOCHT_OCR_DIAG`, [`../../internal/ocr/diag.go`](../../internal/ocr/diag.go)) and a scene corpus with
`bounds` separated from `replaceArea` ([`../../tools/ocrlab/`](../../tools/ocrlab/)) - and does not emit the
comparison JSON of section 7.
