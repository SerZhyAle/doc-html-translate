# Pointer: OCR-OVERLAY

- **Id:** `OCR-OVERLAY`
- **Version:** 1.4
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
  excuse their absence. Both rule 10 scopes run until 2027-03-31 and carry ticket E0129 as their
  remediation: the desktop one is renewed record-only (owner decision 2026-10-30), the extension one is
  renewed on 2026-11-08 with the reason a measurement of its own engine gave (3 of 5 Cyrillic images get
  debris plates under the default `eng`; the discard record gives no threshold). See [`../PARITY.md`](../PARITY.md).
- Rule 13 holds: every constant of the section 5 status table carries its status marker in both editions
  (`TestOCRConstantsCarryTheirRule13Status`, run 2026-10-20).
- Evidence owed to the records of other products (`OCR-ACCURACY`, `OCR-EXCHANGE`) was carried by ticket E0121 and is
  delivered. The word-gap ratio at PSM 11 was accepted as record on 2026-10-29; the type-size ratio on screenshots and
  the bracket behind the 0.72 line fill were measured on 2026-10-30 on the lab's own-UI screenshot scenes (no change:
  nothing on the window falls inside the 1.42-1.86 bracket, and no cluster reaches the 0.52 coverage that the line fill
  is judged under). Their truth is the window's rendered DOM, unsigned by a human; the owner accepted it as the
  evidence of record on 2026-10-30, so those figures stay exploratory. A record binds nobody, so this was evidence
  owed, not an exception.

**Conformance.** The catalog has no shared vector set for this contract yet; section 6 of its README names
the ladder. This product holds the first three rungs - the discard record, the per-image diagnostic line
(`DOCHT_OCR_DIAG`, [`../../internal/ocr/diag.go`](../../internal/ocr/diag.go)) and a scene corpus with
`bounds` separated from `replaceArea` ([`../../tools/ocrlab/`](../../tools/ocrlab/)) - and since 2026-09-26
the fourth: `go run ./tools/ocrlab exchange <run-dir>` writes the section 7 comparison record per scene from
either edition's run, `translation` absent.

**Re-read 2026-10-29 (ticket E0126) at 1.4.** The 2026-10-29 amendment: a silent or under-floor script verdict is not evidence either way (A), a product that runs a script check records the language source and the outcome in its rule 12 record (B - the desktop edition does, `internal/ocr/diag.go`), and proceeding on a recorded silent verdict is not adopted (C). The desktop rule 10 exception therefore closes only by refusing or by a derived threshold (ticket E0121 then, ticket E0129 since 2026-11-08).
