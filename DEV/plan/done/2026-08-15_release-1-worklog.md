# Release 1 worklog, 2026-08-15 .. 2026-09-12

**Status:** Implemented - history, not a ticket. Moved out of `RELEASE_QUEUE.md` on 2026-09-25 so the
queue holds only what is left to do. Section numbers are the ones the queue and other files cite
("queue §1.3").

## release 1 - what was worked

Worked 2026-08-15. Every line below reached a stated outcome; two of them are outcomes the work did
not expect, and those are the entries worth reading.

```
rel  ticket                                          changed     status
--   (no ticket) uncommitted OCR + lab work          2026-08-15  Gates green, unproven -> proven
--   (no ticket) lab scores "found nothing" as       2026-08-15  Fixed and measured
     "concealed perfectly"
1    2026-08-13_ocr-rescue-floor-drops-genuine-      2026-08-15  Partial - rule measured and refused
     lettering
--   2026-08-12_ocr-exchange-followups (items 2, 6)  2026-08-15  Done, both measured
--   (no ticket) verify-html.ps1 false-FAILs EPUB    2026-08-15  Fixed and measured
--   (no ticket) blackletter PDF extracts a          2026-08-15  Fixed; ticket in done/
     corrupt raster
```

### 1.1 The tree is committed-ready, and its two shipped-code fixes are proven

The gates the previous edition of this file said had never been run against this tree have been:
`./scripts/test.ps1` exit 0 (`tests` 137 s, no FAIL), `./scripts/lint.ps1` and
`./scripts/typo.ps1` pass, `npm test` 140/140. `docs/PARITY.md` carries `inkHeight` and the
changelog carries its row - both were already in the tree when the gates were run, so what was
missing was the proof, and the proof exists now.

Two things were found while proving it and fixed here: the former coordinate field
names that `scripts/typo.ps1` read as misspellings (renamed to `inkX0`/`inkY0`), and
`extension/eng.traineddata` - 4 MB that `npm run ocrlab` drops beside the extension and that
nothing ignored, so it would have gone into the release commit. Now in `.gitignore`.

### 1.2 The concealment gate can see the failure it exists to catch - and it is now red

Fixed and **verified by a run**: `temp/ocrlab/20260815-190756`, dev split, 13 annotated scenes.
`unmeasuredConcealment` is **0** - every scene was measured - and `worstResidual` goes
**0.2705 -> 0.9992**, exactly the number the extension run predicted for the same scenes. Everything
else is identical to the reference run to the digit: recall 0.6154, mean IoU 0.7756, worst IoU
0.3489, merges 1, splits 0, cross-group 6, clipped 0, drift 0, protected damage 0.

**The consequence is that `ocrlab gate` now fails on concealment (0.9992 against a 0.28 bound), and
that is the fix working.** The bound was derived while the scorer was blind to every scene where
recognition found nothing. The `comic` category still passes at 0.2705, which is the real number
from the plate-composition ticket. Re-deriving `thresholds.json` was already listed as blocked on
this item; it is now unblocked and is the next dated baseline run, not a release blocker.

### 1.3 The floor could not be re-derived, and the rule that followed was refused by the corpus

[`04_2026-08-13_ocr-rescue-floor-drops-genuine-lettering`](../04_2026-08-13_ocr-rescue-floor-drops-genuine-lettering.md)
- now **Partial**. Evidence:
[`DEV/research/ocr_rescue_floor_2026-08-15.md`](../../research/ocr_rescue_floor_2026-08-15.md).

The ticket asked for the band behind `ocrRescueLineConf` to be re-measured. It was, and **the band
does not exist**: genuine rescued lettering runs 32.8-69.2 and invented lettering 8.4-73.9, with the
highest invention above the highest genuine line, so no single floor admits `ЗАЧЕМ` (69.2) while
rejecting `ОБ ЗЛОМ` (73.9). The ticket's own third bullet asked for exactly this to be said rather
than for the number to be nudged.

The axis that does separate them is length - of 175 rejected lines the eight highest-scoring are
debris of one to six characters, and a four-letter run leaves nine that bracket an empty band
(36.1 / 58.3). **That rule was implemented in both editions, run over the dev split, and the corpus
refused it:** the whole delta is one scene, `poster-display-type-on-flat-colour`, which under the
default `eng` goes from no plates to one 782x310 px plate of transliterated debris across its own
lettering. Reverted; the floor stays at 80.

**What ships is the instrument.** Both editions now record the lines the floor rejected - text,
confidence, box and the floor failed - through one `keepLine` predicate that `clusterLines` also
asks, and the record is written **even for a page that produced no plates**, the case that used to
write nothing. That is the ticket's fourth done-criterion, and it is what makes the next attempt a
measurement instead of a guess. The next attempt needs a third axis; the most promising is not
running the rescue ladder at all when the script check says the language is wrong.

### 1.4 Both cheap items out of the positioning-exchange list are done and measured

[`2026-08-12_ocr-exchange-followups`](2026-08-12_ocr-exchange-followups.md) items 2 and 6.

**Item 2 - print. The ticket's premise was wrong and the fix is still right.** Measured through
`Page.printToPDF(printBackground:false)` - the print dialog's own path - on
`img-png_Nyoka-comic-page`: Chromium does not leave the plate transparent over legible source
lettering. It repaints it **white** and darkens its text, so the sheet stays readable and stops
matching the artwork. **20 of 20 sampled plate papers forced to `1 1 1`** and 14 ink colours
darkened without `print-color-adjust:exact`, **0** with it, out of 59 colour operators; the
extension, same instrument on the shipped stylesheet, 3 of 3 forced against 0. Chrome's
`--print-to-pdf` switch cannot see the difference at all, which is recorded because the first
measurement attempt looked like the fix not working.

**Item 6 - EXIF.** Every `createImageBitmap` in `ocr-overlay.js` now names
`imageOrientation: "from-image"`; `TestParityOCRExifOrientation` fails on any bare call.

Item 1 (grade absolute position) stays package 2. Items 3, 4, 5, 7 stay package `--`.

### 1.5 The pre-flight sweep can gate an EPUB conversion

`scripts/verify-html.ps1` now resolves a redirecting entry page before anything is checked - the
JS `location.replace` stub and `<meta refresh>`, up to four hops, size-guarded so it never runs on
a real chapter - and for a folder it enumerates the directory the stub points into, so a multi-page
EPUB's `page_*.html` is checked too. Measured on `temp/ocrsweep/19_epub-illustrated`:
**`broken=55` -> `total=55 render=55 broken=0`**; a PDF output in the same sweep is unchanged at
`total=1 render=1 broken=0` on both its pages.

### 1.6 The PDF smear was a layer, not a decoder

Fixed, with its own ticket in
[`done/2026-08-15_pdf-mrc-foreground-layer-extracted-as-page.md`](2026-08-15_pdf-mrc-foreground-layer-extracted-as-page.md).
It is a mixed-raster-content scan: a 1455x2065 background layer plus a 4363x6193 **foreground** layer
painted through a stencil `/Mask`, undefined wherever the mask does not select it, and 91 KB for 27
megapixels. `selectPageImages` kept the larger of the same-shape pair, so it kept the layer that is
not a picture. Inside a duplicate group a masked raster now loses to an unmasked one however big it
is; `/SMask` deliberately does not demote and a lone masked illustration is still kept.

**Both plausible answers were wrong and are recorded as such:** ffmpeg's native JPEG 2000 decoder
(the only JPX converter on this machine) decodes the background layer correctly and reports
`0 decode errors` on the foreground one, and the duplicate-collapse rule is right for what it was
written for. Class width measured over `test_doc/`: **1 file of 21, 1 image XObject of 2 560** - the
triage the queue asked for, and narrow, but fixed as a rule because the rule is one comparison and
the input class is one this product is aimed at.

### 1.7 A "line" the recognizer stitched across a picture became a bar across the artwork

Worked 2026-09-12, out of order and for a stated reason: it arrived as a user report on
`test_doc/1.png` and is the worst class of overlay defect there is - not text that is missing, but
**artwork covered by a plate that should not exist**, carrying a sentence neither speaker said. Own
ticket in
[`done/2026-09-12_ocr-line-stitched-across-the-picture.md`](2026-09-12_ocr-line-stitched-across-the-picture.md),
measurement in [`../research/ocr_word_gap_2026-09-12.md`](../../research/ocr_word_gap_2026-09-12.md).

The defect is in the **engine's own line assembly**, identically in both editions: PSM 3's layout
analysis walks across the photographed figure and returns line boxes 987-1727 px wide holding text
from both columns. Nothing downstream could recover - the clustering's column test sees a genuine
overlap, and `ocrMaxPlateCoverage` never fires because the bar is wide but short (0.04 of the image).
So the repair runs before the clustering: cut a line at a word gap wider than
`ocrMaxWordGapRatio (3.5) x` its median word height, then regroup the page's runs into columns.

**This closes the one merge `DEV/ocrlab/thresholds.json` names in its grouping baseline.**
`synth-two-columns` goes 1 plate crossing the gutter -> 2 plates matching both hand-drawn
transcripts verbatim, with no split traded for it; the reported image goes 9 plates with 3 bars ->
10 plates, one per balloon, in both editions. Two things were got wrong on the way and are recorded
in the research note, because both were invisible until the corpus was run: the reordering's scope is
the page and not the recognizer paragraph, and a line the confidence floor will drop must not be
allowed to form a column.

It does **not** close Phase 07 Step 07.3 of the lab ticket, and the two are not alternatives: 07.3
adds a boundary test to the clustering, this repairs the clustering's input. The band where comic
balloons and real lines overlap (1.87-2.57x) is measured, stated, and left to 07.3.


## Bookkeeping found while rebuilding the queue (2026-08-15)

Recorded rather than quietly fixed, because a status that two files state differently is a status
nobody can trust.

1. **Four referenced plan files do not exist on this machine.** `DEV/plan/ROADMAP.md` (referenced by
   this file and by `CLAUDE.md` as "the queue"), `DEV/plan/2026-07-01_cross-edition-parity.md`
   (referenced by `CLAUDE.md` as the standing parity backlog),
   `DEV/plan/_TEMPLATE_cross-edition.md` (the cross-edition ticket template `CLAUDE.md` tells every new
   ticket to use) and `DEV/plan/2026-07-28_thirteen-ui-languages.md`. `DEV/plan/` was in `.gitignore` then (it is
   tracked since), so none of them could be recovered from history. Either they were deleted or they never existed on this
   clone; both `CLAUDE.md` and three files in `done/` still link to them.
2. **`16_2026-08-11_ocr-visual-fidelity-lab.md` said `Tactical` while its `INDEX.md` said `In Progress`.**
   Corrected 2026-08-15 in favour of the INDEX, which is the authority on phase state. The previous
   edition of this file recorded the same disagreement and left it standing, and it had also gone stale
   in the other direction - it reported 4 of 8 phases where the INDEX says 6.
3. **Two tickets moved to `done/` on 2026-08-15** with their tactical folders and their relative links
   repaired: `2026-08-12_extension-crashes-the-tab-on-a-detailed-scan` (Implemented 2026-08-12) and
   `2026-08-12_ocr-misses-display-lettering-on-saturated-art` (Implemented 2026-08-13, 7/7 phases).
   Earlier moves had not repaired their links; the seven that could be resolved were fixed at the same
   time, and the nine that point at the files from item 1 were left visible.
