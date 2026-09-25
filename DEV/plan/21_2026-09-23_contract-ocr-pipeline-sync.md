# The OCR contracts describe the mechanism that ships, and the mechanism holds them

**Status:** Draft
**Priority:** 49
**Date:** 2026-09-23

> Contract sync ticket, both directions.
> Contracts: `OCR-PIPELINE` 1.0 (owned by this product), `OCR-OVERLAY` 1.0 (shared owner, this product is
> the reference implementation), `OCR-INVOCATION` 1.0 (owned by this product). Domain `ocr-overlay/`.
> Pointers: [`OCR-PIPELINE`](../../docs/contracts/OCR-PIPELINE.md),
> [`OCR-OVERLAY`](../../docs/contracts/OCR-OVERLAY.md), [`OCR-INVOCATION`](../../docs/contracts/OCR-INVOCATION.md).
> Related open tickets, not duplicated here:
> [`15_2026-09-22_ocr-discard-record-missing-for-blank-images`](15_2026-09-22_ocr-discard-record-missing-for-blank-images.md),
> [`17_2026-09-22_tsv-columns-read-by-position`](17_2026-09-22_tsv-columns-read-by-position.md).

## What / why

The 2026-09-22 alignment run moved the OCR documents into the catalog and verified their *place*, half of
`OCR-OVERLAY` and all of `OCR-INVOCATION`. It left two things open on purpose: the `OCR-PIPELINE` registry
row is `pending` because nobody read its constant table against the code, and nine `OCR-OVERLAY` rules
(2, 3, 4, 7, 9, 11, 13, 14, 15) were not re-read.

That read was done on 2026-09-23. The constants hold - all twenty in `OCR-PIPELINE` §5 match in Go and JS.
The **document** does not: it was registered already stale against the word-gap stage of 2026-09-12 and
the page-OCR surface of 2026-09-19, so the "catalog first, then code" order this repo's own pointer states
was broken - and FastMediaSorter_Lite ports from that document. The read also found deviations the
previous run did not.

The working-tree edits in `internal/ocr/*.go` and `internal/pipeline/pipeline.go` are comment-only; no
pinned value, flag, exit code or lookup order moved since 2026-09-22.

## Findings that drive the work

**`OCR-PIPELINE` - missing from the document (code is right, document behind):**
- the word-gap split (`ocrMaxWordGapRatio` 3.5), the column reordering and the outlier-word trim of
  2026-09-12, and the fact that type size is now the median of word heights - all pinned by parity tests,
  none in the document;
- the fit ladder's grow branch (cap 1.15, step 4%, 20 iterations) - the document says shrink only;
- the colour fallback (luma > 140 -> 17, else 240), the min-6 floor on the ink share, the 1.3-line ink
  strip, the column-overlap test (0.1) and the negative-gap tolerance, plate padding and radius, the CJK
  and ratio details of the translatability test;
- the page-OCR surface (extension page agent, 96 px minimum picture) in §4 and §6;
- the script-confidence floor 6.4 is in prose only, not in the §5 table.

**`OCR-PIPELINE` - wrong in the document:**
- §2.7 points at `hasWordRun`, which exists in neither edition (a rolled-back rule);
- the ring band is "one line-height wide" in §3.3 - the code samples a third of one; a port built from the
  prose samples a band three times wider (open question 3);
- §3.2 "fully transparent pixels skipped" - the code skips alpha < 128;
- §2.5 "3 median ink heights" - the code multiplies the median line-box height;
- the module map names `ocr-overlay.js` for plate rendering (moved to `ocr-plates.js`) and a per-image
  diagnostics line in `diagnostics.js` that does not exist;
- §7 "every threshold above was measured" overclaims (see rule 13 below).

**`OCR-OVERLAY` - verdicts of the nine unread rules:**
- held: 3 (one explicit transform), 7 (opaque plate, padding part of it), 11 (OCR text never becomes
  truth), 14 (not measured is not nothing visible);
- **rule 2 partial** - Go `scaleDown` does not scale `Dropped`, so the discard record's boxes on an
  upscaled image stay in 2x space while width and height are halved;
- **rule 4 partial** - held for generated HTML and the viewer; the page-OCR surface reads the border-box
  rectangle and handles neither CSS rotation, `object-fit`, padding nor clear-on-rotation;
- **rule 9 partial** - the bounded ladder and release exist; the written UI rule for a released plate that
  overlaps the next one or runs past the image bottom does not;
- **rule 13 not held** - 0.52, 0.72, 1.6, 3.5, 6.4, 80 and the screen set are bracketed with dated
  reports; 50, 1.2, 3, 120/11/70, 0.92, 1.15, the colour numbers, the fit ladder and 40 are neither
  bracketed nor marked inherited;
- **rule 15 partial** - negative results (the word-gap overlap band 1.87-2.57, per-paragraph ordering
  rejected, trimmed box in decisions rejected) live in code and PARITY, not in the catalog.

**`OCR-OVERLAY` - findings outside the assigned rules:**
- **rule 5** - both editions size the plate font from the median of line-box heights; the rule says type
  size is the median of word heights, not the line box. The 2026-09-22 row recorded rule 5 as held, and
  FastMediaSorter_Lite reads the rule as binding the font;
- **rule 12** - silent gates beyond the blank-image ticket: whole clusters rejected by the translatability
  test, the screen-merge overlap rejection, and the sweep pass's own drops are recorded nowhere; Go appends
  primary + ladder drops while JS keeps only the ladder's, and `docs/PARITY.md` says "not merged";
- **rules 1 / 10** - the script-detection pass reads the raw first path, without the ASCII staging the
  document says recognition needs and without the EXIF turn, so on a Cyrillic file name it fails silently
  and no correction happens.

**`OCR-INVOCATION`:** unchanged in code. The document's "an omitted `-ocr-lang` derives from `-src`, else
`eng`" omits the script correction and the "no plates, with a note" stop that exist since 2026-08.

**§7 exchange JSON:** not emitted. Go blocks carry no confidence; `translation` does not exist at overlay
time (translation runs after); the lab's evidence schema is the nearest artefact.

## Direction B first - the catalog moves before the code

This product owns `OCR-PIPELINE` and `OCR-INVOCATION` and edits them directly (a dated amendment section,
never an in-place rewrite, plus a document-log row). `OCR-OVERLAY` has a shared owner, and the other
implementations are FastMediaSorter Android and FastMediaSorter_Lite, so a change to its rules is a
proposal beside the document.

1. **`OCR-PIPELINE` 1.0 corrections** (log rows, no version): the `hasWordRun` pointer, the module map,
   the alpha wording, the line-box wording in §2.5, the 1.3-line ink strip, the §7 overclaim.
2. **`OCR-PIPELINE` 1.1 (MINOR, additive):** the word-gap / column-order / trim stages and the word-median
   type size; the grow branch; the missing constants in §5 with a **status column** (derived with its
   report / inherited / policy); the page-OCR surface; the negative results of rule 15.
3. **Ring band width** - correction or MINOR with a notice to both consumers, per open question 3.
4. **`OCR-INVOCATION` 1.1 (MINOR, clarification):** script correction and the no-plates stop for an
   omitted `-ocr-lang`; flags and exit codes unchanged.
5. **`OCR-OVERLAY`:** this product's own adoption row rewritten with the per-rule result above (rule 5
   no longer "held"); new dated exceptions for rules 2, 4 (page surface), 5, 9, 12 (the extra gates), 13;
   a proposal if the owner group reads rule 5 as binding clustering only (open question 1).
6. **Registry:** the `OCR-PIPELINE` row moves from `pending` to verified with this read as its evidence.
7. Notice to FastMediaSorter Android and FastMediaSorter_Lite in their rows' notes: what 1.1 adds and
   which prose they may have ported wrong (ring band).

## Direction A - the code holds the contracts

After the catalog change it depends on:

1. `scaleDown` scales `Dropped` (rule 2); test asserts it.
2. The discard record covers every gate - translatability, screen merge, sweep - with the gate's name, in
   both editions; one merge semantic for primary + ladder drops in both editions, and PARITY says which.
3. Plate font from the word-height median in both editions (rule 5) - or a dated exception if the owner
   group decides otherwise.
4. Script detection stages the input to an ASCII path and applies the EXIF turn (rules 1, 10).
5. Rule 9's overflow rule written into the contract and implemented; the "JS disabled -> clipped" claim
   checked in a browser.
6. Page-OCR surface: clear the layer on rotation / `object-fit` / padding it cannot place (rule 4), or an
   exception.
7. Rule-13 markers on every unbracketed constant in both editions (derived with report / inherited /
   policy), matching the §5 status column.
8. Parity pins for the unpinned numbers: colour, fit ladder, column test, ring-pad rounding (Go floors, JS
   rounds - a small real drift no guard sees today).
9. JS unit tests for `ocr-plates.js` (`fitPlate`, `plateSpecs`).
10. Stale comment in `overlay.go` ("ink is the mean") corrected.
11. Optional, only if a consumer asks: the §7 exchange JSON, emitted beside the diagnostics line with the
    same pure-function principle.

## Done criteria

- [ ] `OCR-PIPELINE` in the catalog describes every stage and constant both editions run; every §5 row
      carries a status; its log has the correction rows and the 1.1 row.
- [ ] `OCR-INVOCATION` 1.1 in the catalog.
- [ ] Registry: `OCR-PIPELINE` verified with a date; `OCR-OVERLAY` row carries the per-rule result for all
      17 rules; each open deviation is a dated exception.
- [ ] Both consumer rows carry the notice.
- [ ] Code items 1-10 landed, or each carried by a dated exception; `scripts/test.ps1` and the extension
      tests green, output cited.
- [ ] Pointer files updated to 1.1 where the version moved.
- [ ] `docs/PARITY.md` and the FastMediaSorter Lite spec sync obligation (standing practice: OCR changes are
      mirrored into its overlay-accuracy spec) handled for every behaviour change.

## Open questions

1. Rule 5: does "type size is the median of word heights" bind the plate font, or only clustering? Change
   the code, or propose a wording to the `OCR-OVERLAY` owner group?
2. If the font basis changes, MINOR ("tightening what rule 5 already required") or MAJOR? Apply
   `VERSIONING.md` §3's test to FastMediaSorter Android and Lite.
3. Did either consumer port the "one line-height" ring band from prose? That decides correction vs MINOR
   with a notice.
4. Rule 12: every gate recorded with a gate-name field (FastMediaSorter_Lite does this)? Merge primary +
   ladder drops (Go today) or keep the winner's only (JS today, PARITY's wording)?
5. Rule 13 for policy constants (translatability, 0.92, padding): accept a third "policy" status, or force
   derived / inherited?
6. §7 exchange JSON: is `translation` optional; emit from the Go diagnostics, the lab runner, or both?

## Notes

`tesseract.go` (1407 lines), `overlay.go` (828) and `ocr-overlay.js` (611) are over the file budget and
most code items touch them. The 386 toolchain's memory ceiling can kill `go test ./tests/` on a first run;
a rerun that passes is not a regression.
