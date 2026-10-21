# Pointer: OCR-OVERLAY

- **Id:** `OCR-OVERLAY`
- **Version:** 1.3
- **Home:** the shared contracts catalog, `ocr-overlay/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** reference implementation - this product both produces the overlay and consumes its own output
- **Other products bound:** FastMediaSorter Android, FastMediaSorter_Lite

The function contract for reading text off an image and drawing text back over it: display-space
recognition, line-level boxes, plate geometry and sampled colour, what is never clipped, what is never
trusted, and how a constant earns its value. Seventeen numbered rules; this repo implements them in
two editions (Go and the browser extension) held to one constant table by the guard tests in
[`../../tests/parity_test.go`](../../tests/parity_test.go), except for the dated exceptions below (rules 1 and 10).

**What this repo owes it**

- Change the contract before the code whenever behaviour at the overlay boundary moves: the catalog commit
  comes first, then both editions.
- Keep every shared constant in step across the two editions, and keep its derivation dated - `OCR-OVERLAY
  rule 13`. The numbers themselves are this implementation's own; they are not facts another product may
  copy.
- Cite rules in source as `OCR-OVERLAY rule N`, never as a path.
- Open dated exceptions: JPEG-only EXIF (rule 1), and the assumed language without qualifying script evidence
  (rule 10) in two scopes - the extension, which recognizes with a persisted assumed `eng` and has no script
  pass, and the desktop edition, where a silent or under-floor script verdict leaves the language assumed
  (since 2026-10-21 the run log and the `DOCHT_OCR_DIAG` line record the language source and the verdict,
  silent or not; the record changes no decision). Amendment 1.3 permits recorded evidence; it does not
  excuse their absence. Both rule 10 scopes are re-verified under ticket 121 (until 2026-12-31). See
  [`../PARITY.md`](../PARITY.md).
- Rule 13 holds: every constant of the section 5 status table carries its status marker in both editions
  (`TestOCRConstantsCarryTheirRule13Status`, run 2026-10-20).
- Evidence owed to the records of other products (`OCR-ACCURACY`, `OCR-EXCHANGE`) is carried by ticket 121: the
  type-size ratio on screenshots, the word-gap ratio at PSM 11 and the bracket behind the 0.72 line fill. A record
  binds nobody, so it is evidence owed, not an exception.

**Conformance.** The catalog has no shared vector set for this contract yet; section 6 of its README names
the ladder. This product holds the first three rungs - the discard record, the per-image diagnostic line
(`DOCHT_OCR_DIAG`, [`../../internal/ocr/diag.go`](../../internal/ocr/diag.go)) and a scene corpus with
`bounds` separated from `replaceArea` ([`../../tools/ocrlab/`](../../tools/ocrlab/)) - and since 2026-09-26
the fourth: `go run ./tools/ocrlab exchange <run-dir>` writes the section 7 comparison record per scene from
either edition's run, `translation` absent.
