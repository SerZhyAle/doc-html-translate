# Release Queue

What is still LEFT TO DO before the next release, in execution order. Everything that has not reached
`Implemented` lives here, including work blocked on another ticket, on an owner answer, or on a human
step - those need sorting too. Finished work is not here: the moment a ticket reaches `Implemented` or
`Verified` its file moves to [`done/`](done/) and its line leaves this table.

Rebuilt 2026-08-15 against the ticket files themselves. The previous edition of this file was written
on 2026-08-11 and had gone stale: four of the six tickets it listed are now in `done/`, and none of the
five OCR tickets opened between 08-12 and 08-15 appeared in it at all.

- `rel` - the release package this ticket ships in. It is an **ordinal**, not a version: this product's
  version is derived mechanically from the build date (`26.MMDD.HHmm`) and is never hand-picked.
  `--` = not scheduled, no code work left here.
- `ticket` - spec file name in `DEV/plan/` without the extension, or `(no ticket)` for work that is
  real, evidenced and unfiled - see "Unfiled work" below.
- `changed` - the date the STATUS last moved (from the ticket's own status line or its tactical
  `INDEX.md` "Last updated"), not the date the prose was last edited.
- `status` - copied from the ticket file itself. There is no catalog to mirror in this repo, so this
  column is maintained by hand and the **ticket wins** whenever the two disagree.

**This file decides what gets worked on next.** When it and a ticket disagree about order, this file
wins; when they disagree about a status, the ticket file wins. Deviating from the order below is
allowed with a stated reason, never silently.

The line order inside a package is execution order, built by these rules, top to bottom:

1. Work already in flight comes first - it is half-paid for, and here it is literally uncommitted.
2. Then whatever the rest of the package is validated against: the measuring instrument before the
   thing it measures.
3. Then ready-to-build work, closest to done first - Tactical before Approved before Draft.
4. A blocker always sits above the ticket that waits for it, so the chain reads downwards.
5. Work that only adds strings or docs goes last in its package, after every change that will add keys.
6. Anything waiting on an owner answer sinks to the bottom of the package.
7. Anything waiting on a human pass or a third party sinks below that. It is not schedulable, so it
   must not sit where it looks like the next task.

Packages, so a new ticket lands in the right one:

- `1` - **must ship before the next release.** A user-visible defect in shipped code, or the
  instrument that certifies the release is honest. Nothing enters this package for tidiness.
- `2` - real, evidenced, and survivable for one more release. Hygiene, gates, and quality work whose
  absence costs no reader anything today.
- `--` - living backlogs, human-gated corpus work, and items with no code left, so no package.

current-next-release: 1 (worked 2026-08-15; see the notes under each line)

## release 1 - must ship

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

[`2026-08-13_ocr-rescue-floor-drops-genuine-lettering`](2026-08-13_ocr-rescue-floor-drops-genuine-lettering.md)
- now **Partial**. Evidence:
[`DEV/research/ocr_rescue_floor_2026-08-15.md`](../research/ocr_rescue_floor_2026-08-15.md).

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
[`done/2026-08-15_pdf-mrc-foreground-layer-extracted-as-page.md`](done/2026-08-15_pdf-mrc-foreground-layer-extracted-as-page.md).
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
[`done/2026-09-12_ocr-line-stitched-across-the-picture.md`](done/2026-09-12_ocr-line-stitched-across-the-picture.md),
measurement in [`../research/ocr_word_gap_2026-09-12.md`](../research/ocr_word_gap_2026-09-12.md).

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

## release 2 - can slip one release, with the reason stated

```
rel  ticket                                          changed     status
2    2026-09-22_ocr-discard-record-missing-for-      2026-09-22  Draft
     blank-images
2    2026-08-15_plate-styling-single-source          2026-09-25  Partial (4 manual, owner machine)
2    (no ticket) plate box rides over the logo       2026-08-13  Evidenced, unfiled
2    (no ticket) tesseract.js misses a caption on    2026-08-15  Evidenced, unfiled
     a gradient
2    2026-08-11_ocr-visual-fidelity-lab              2026-08-15  In Progress (6/8 phases)
2    2026-08-13_ocr-sweep-plate-composition          2026-08-13  Partial (7/8 criteria)
2    2026-09-22_tsv-columns-read-by-position         2026-09-22  Draft
2    2026-09-23_contract-ocr-pipeline-sync           2026-09-23  Draft - catalog amendment first, then code
2    2026-09-23_contract-rule-adoption-sync          2026-09-23  Draft - no product code
2    2026-09-23_contract-desktop-app-ux-sync         2026-09-23  Draft
2    2026-09-23_contract-iconography-sync            2026-09-23  Draft - proposals before code
2    2026-09-23_contract-product-web-pages-sync      2026-09-23  Draft - site, every authored locale
2    2026-09-22_install-trust-page                   2026-09-22  Draft - docs only, every authored locale
2    2026-09-19_page-ocr-overlay                     2026-09-19  BlockNeedUserTest - code done, hands-on pass + store permission text left
```

**The six `2026-09-23_contract-*` tickets** (five left - `automated-checks` reached Implemented on 2026-09-24 and moved to `done/`) come out of one contract-sync pass over every catalog domain
that touches this product. Each has two halves: what the repo changes to conform, and what the catalog
lacks - written as a dated amendment where this product owns the contract (`OCR-PIPELINE`,
`OCR-INVOCATION`) and as a proposal beside the contract everywhere else. `OCR` leads because this product
owns the document another product ports from, and it was registered stale. `rule-adoption` is next because
its first item - committing `docs/contracts/` - is what every other pointer depends on. The web-pages
ticket carries one user-visible bug that should not wait for the rest of it: the shared `sza-lang` value is
`ua` on the landing page and `uk` on the extension page, so a language chosen on one shows all three on the
other - a `/fix` candidate on its own. `WAVE-PARTICLES` was read and does not apply (no canvas backdrop).

[`2026-09-22_ocr-discard-record-missing-for-blank-images`](2026-09-22_ocr-discard-record-missing-for-blank-images.md)
is first in the package by rule 2: it is the instrument the rest is measured with. Opened by the contract
alignment run of 2026-09-22 against `OCR-OVERLAY rule 12`, and it corrects §1.3 of this file - the record
was preserved as far as `applyOverlays` and is then not written, so an image that produced **no** plates
still leaves nothing behind, which is the one case the record exists for. Proven with a throwaway probe in
package `ocr` (`applyOverlays: changed=false NoText=1`, no diagnostics file). Nothing a reader sees changes
when it lands; what changes is that a blank scene can be told from a discarded one.

`plate-styling-single-source` is built (2026-09-24): `internal/appearance/appearance.json` is the one
description of the OCR overlay and the reader palette, both editions derive from it, and
`tests/appearance_parity_test.go` fails on any declaration one side has and the other lacks. What is left
needs the owner's machine: the catalog's `ocr-pipeline.md` repoint (Step 05.3 - the catalog was not
reachable), the four PowerShell gates, and the corpus comic render (Step 06.2); a Chromium render of old
against new CSS was pixel-identical meanwhile. Then `/spec-check`.

**The `First-Earthman` cover plate rides over the `PLANET COMICS` logo.** The plate's text is the
cover's own top banner line and it sits on that banner, but its box is taller than its line, so it
reaches up over the logo. Coverage 0.2774, inside the bound - a plate-box height question, not a merge.
Needs its own ticket. Package 2 because comics are the core case and this is visible, but it damages
one logo rather than the reading.

**`synth-caption-on-gradient` is read by native tesseract and missed by tesseract.js.** An engine
difference, which the lab spec anticipates in §6, but it is still a scene the reader loses in one
edition and keeps in the other. The cheap resolution is to record it as a divergence in
[`docs/PARITY.md`](../../docs/PARITY.md) with the consequence stated; the expensive one is a rescue
rung. Decide, do not leave it unwritten.

`ocr-visual-fidelity-lab` is 6 of 8 phases. Phase 07 (concealment and grouping) reads "Not started"
while most of its subject matter was in fact delivered out of band by the P46 and P47 tickets - that
mismatch needs reconciling before the phase is planned, not after. Phase 08 is 5 of 6 with its last step
waiting on 07. Its two genuinely open pieces are human-owned and sit in package `--`.

`ocr-sweep-plate-composition` is `Partial` with 7 of 8 done-criteria met and measured. **No code work is
left in it.** The open criterion is corpus entry for `Soldiers-Creed` and the `First-Earthman` cover,
which the lab's own contributor rules put outside what an agent may do: an annotation is truth only when
a person has corrected it, and `licenceVerifiedBy` is filled by hand after opening the asset's licence
page. It stays out of `done/` because `Partial` is not `Implemented`, and it stays out of package 1
because nothing in it ships. Moving it is the owner's call.

[`2026-09-22_tsv-columns-read-by-position`](2026-09-22_tsv-columns-read-by-position.md) is the other
finding of the alignment run: `parseTSV` skips the header row that names the columns and then reads fixed
indices, so a Tesseract build that inserts a column would not fail - it would put plates in the wrong place
with the wrong confidences. No build in use today does, which is why it is package 2 and not 1.

[`2026-09-22_install-trust-page`](2026-09-22_install-trust-page.md) is docs-only work, so rule 5 puts it
last among the schedulable lines. Three of the four download channels are unsigned - the setup exe and both
portable exes - and nothing we ship says the word SmartScreen, so a user who meets "Windows protected your
PC" reads nothing from us. It is package 2 rather than 1 because no shipped code is wrong; it is not `--`
because every unanswered warning is a user who does not come back. The contract it closes is
`INSTALL-TRUST` 1.0, and until it lands the gap is a dated exception in the shared registry.

[`2026-09-19_page-ocr-overlay`](2026-09-19_page-ocr-overlay.md) is the first **new user-facing feature**
in this queue rather than a defect or an instrument: recognize every picture on an ordinary live web
page and lay the plates over them in place, so the words can be copied and the browser's own page
translation reaches them. **The code is written and the gates are green**; what gates it now is a human
pass, not a decision. The reach question that blocked it - the obvious host for the recognition engine
is newer than the extension's declared minimum Chrome version, and raising that minimum is forbidden -
was answered in code: the newer host is used where it exists and an extension-origin frame in the page
is used where it does not, and `minimum_chrome_version` stays at 105, pinned by a test. What is left is
a hands-on pass on real sites (which is also how research items 2, 3 and 4 get their measurements) and
the store-listing permission text, which needs the owner's sign-off first. It stays last in package 2
for the reason rule 7 gives: it is waiting on a human pass, so it must not sit where it looks like the
next task.

## not scheduled - no package

```
--   lab corpus growth + holdout annotation          2026-08-11  Human-owned
--   thresholds.json is stale against its own        2026-08-15  Unblocked by 1.2, next baseline
     corpus
--   the halo metric penalises a correct plate on    2026-08-13  Evidenced, no gate
     a screened ground
```

The lab's remaining human phases - acquiring 200+ licence-verified scenes, each licence page read by a
person, and the two-reviewer holdout annotation - are the real gate on the whole visual-fidelity
package and cannot be scheduled against a release date.

`thresholds.json` is now the **first thing after the release**: 1.2 landed, so the blocker named below is gone and the concealment bound is knowingly derived against a blind scorer. It was derived from 11 annotated scenes; there are 13, and the two added since are the
hardest in the corpus, so `recall` and `review` fail on arithmetic rather than on a regression, and
`cost` is stale because the rescue ladder now runs every rung. Re-deriving belongs to a dated baseline
run; the baseline it should be derived against is `temp/ocrlab/20260815-190756`, the first run whose
concealment numbers cover every scene.

## Bookkeeping found while rebuilding this file

Recorded rather than quietly fixed, because a status that two files state differently is a status
nobody can trust.

1. **Four referenced plan files do not exist on this machine.** `DEV/plan/ROADMAP.md` (referenced by
   this file and by `CLAUDE.md` as "the queue"), `DEV/plan/2026-07-01_cross-edition-parity.md`
   (referenced by `CLAUDE.md` as the standing parity backlog),
   `DEV/plan/_TEMPLATE_cross-edition.md` (the cross-edition ticket template `CLAUDE.md` tells every new
   ticket to use) and `DEV/plan/2026-07-28_thirteen-ui-languages.md`. `DEV/plan/` is in `.gitignore`, so
   none of them can be recovered from history. Either they were deleted or they never existed on this
   clone; both `CLAUDE.md` and three files in `done/` still link to them.
2. **`2026-08-11_ocr-visual-fidelity-lab.md` said `Tactical` while its `INDEX.md` said `In Progress`.**
   Corrected 2026-08-15 in favour of the INDEX, which is the authority on phase state. The previous
   edition of this file recorded the same disagreement and left it standing, and it had also gone stale
   in the other direction - it reported 4 of 8 phases where the INDEX says 6.
3. **Two tickets moved to `done/` on 2026-08-15** with their tactical folders and their relative links
   repaired: `2026-08-12_extension-crashes-the-tab-on-a-detailed-scan` (Implemented 2026-08-12) and
   `2026-08-12_ocr-misses-display-lettering-on-saturated-art` (Implemented 2026-08-13, 7/7 phases).
   Earlier moves had not repaired their links; the seven that could be resolved were fixed at the same
   time, and the nine that point at the files from item 1 were left visible.
