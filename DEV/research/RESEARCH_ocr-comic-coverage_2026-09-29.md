# The Safety-First page: where recognition hides, and the two changes it bought - 2026-09-29

**Ticket:** [`67_2026-09-29_ocr-comic-coverage-safety-first`](../plan/67_2026-09-29_ocr-comic-coverage-safety-first.md)
**Question:** ticket 66's R1 found four FAIL-level behaviours on one public-service Superman page
(`test_doc/New_test/027f032012d82d5e01cc1cc083914fd9.jpg`, 505x720): two side-by-side balloons
read as one interleaved plate, the display-lettering title banner bare, the "WHEW!" balloon bare,
and a caption/banner losing its head, tail and most of its white-on-black lines. What in the
mechanism hides each one, and what does the corpus let us change?
**Answer:** two independent causes, two changes, both landed as `OCR-PIPELINE` amendment 1.7 in
both editions. The merged balloons are the engine's own line assembly - it stitches the rows across
the gutter and reads the touching outlines as tokens of those lines, so the stroke test of
amendment 1.1 finds nothing between the words: the boundary runs *inside* the boundary token's box.
The bare banner, balloon and caption lines are a coverage gap: the colour thresholder kills their
lettering outright while the rest of the page reads fine, so the rescue ladder (fires only when
*nothing* read) and the screen sweep (fires only where a screen is measured) both stand down - but
Tesseract's own layout analysis *marks* the dead regions, isolating text regions it returns no
words for. Both fixes are pinned to that evidence.

## What each pass actually reads on the page

Measured with the app's own staging (2x upscale, DPI 130 declared) and the raw TSV:

- **The ordinary pass** plates 9 of the page's lettering areas. The middle panel's balloons come
  back as stitched rows - row 1 is `GEE, TH-THANKS, “J OON'T DEPEND`, the `“J` box 35 px tall
  against the line's 15 px median, the touching outlines read as one token; the same a row lower
  (`(T'S A`, the `A` 32 px). Only the third row can be cut - its gap holds a real stroke - which is
  why the defect reads as one garbled plate plus a right-balloon fragment.
- **The screen sweep never runs:** `screenPitchOutside` returns 0 - this reprint carries no
  measurable dot lattice. Its trigger is dead on exactly the material the ticket is about.
- **The grey rung (PSM 3, floor 80)** reads the WHEW balloon (`WHEW! THAT WAS A CLOSE` at 95.7)
  and the caption's last line (`CROSS AGAINST A RED LIGHT!` at 93.8) - and nothing on the banner.
- **The sparse rung (PSM 11, floor 80)** reads nearly everything, including the banner - as
  `om FIRST` (94.5) - and the caption's second sentence line, and `WERE AROUND:`, but also `00
  MAGATA` (54.8) and, under the anchored admission, `j. BUT MANY OTHER ACCIDENTS ARE \` (68.3).
  The banner's honest reading is garbage: display lettering red-on-yellow defeats the engine
  whatever the rendition, and the rescue-floor philosophy (a wrong plate costs more than a missing
  one) says the banner stays bare.

## Change 1: the split cuts where the outline is the token (amendment 1.7 A)

A word box taller than the type-size ratio times the line's median word height reaches into a
neighbouring row - no lettering of one line does - so `splitWideGaps` cuts before it and the token
leads the next region's run, where the orphan rule answers for it. No new constant: it is the
outlier trim's own comparison (1.2 item K) applied at the cut stage. On the page, the two stitched
rows are cut at `“J` and `A`; the balloons come out as plates each carrying one balloon's words,
in order. What the cut cannot see is a *fused* token - the engine sometimes splices the two letters
that face the gutter plus the outlines between them into one normal-height word (`ATI`, measured on
a drawn fixture); no between-words rule can cut that, and none was added.

Pinned by the shared split fixture `tests/testdata/ocr_balloon_stitch_cases.json` (the real word
boxes, both editions) and by replaying the engine's own captured page TSV through `parseTSV`
(`internal/ocr/testdata/superman_stitch.tsv`): no plate carries both balloons' words.

## Change 2: the grey sweep for a page that did read (amendment 1.7 B, new 2.10)

The ordinary pass's wordless line rows are the evidence; the sweep fires only when a marked region
stands where the accepted plates leave at most the merge bound (0.2, union measure) of it covered,
runs the grey rendition at the rescue floor with no admission, and merges exactly like the screen
sweep - refusals recorded under the new gate `grey-merge`. On the page it adds exactly the two
missing plates - `WHEW! THAT WAS A CLOSE` and `CROSS AGAINST A RED LIGHT!` - and refuses twelve
candidates, every one a region an existing plate already serves. It costs one extra shell-out only
where wordless regions exist outside the plates; on this page that is a 1.0 s -> 1.4 s run.

The sweep deliberately does **not** include the sparse rung: measured on this page it would plate
`om FIRST` over the banner and `00 MAGATA` under the admission - the transliterated-debris pattern
the rescue floor exists to prevent, at a confidence that clears 80.

## The corpus bound

Same-day runs over the dev split, `eng`, desktop runner; baseline on clean `HEAD` (worktree at
`c32e341`), the with-fix run on this tree. Per-scene diff of the `ocr-diag.jsonl` records plus the
summary's hard gates (the gate's six standing threshold failures on a clean baseline are the
pre-existing drift of 2026-08-12 thresholds, not evidence about this change):

- **Hard gates held at zero in both runs:** merges, splits, protectedHitPx, clipped, crossGroup.
  The failing-scene set is identical (the three poster scenes that read nothing under `eng`).
- **13 of the 14 scored scenes are byte-identical** in plates - including all four annotated comic
  scenes and the scroll caption's neighbours. The annotated corpus is where the thresholds live,
  and nothing it scores moved but one scene.
- **The one scored scene that moved is `samson-and-delilah-03-scroll`, already failing** (recall 0
  before and after): the scroll's curved binding edge puts tall artefact tokens mid-line, so the
  cut fragments its caption further - CER 0.41 -> 0.61, IoU 0.45 -> 0.33 - and the two-letter word
  `BY`, isolated between two artefacts into a run of its own, leaves the plates into the discard
  record (the orphan rule parks a run `isTranslatable` refuses, and no two-letter word clears the
  five-letter floor). Recorded as this change's known cost; no gate moves; a refinement that
  distinguishes an artefact run from a real short word is future work, not this ticket's.
- **The unscored comic corpus shows the intended wins:** the 1920 Red Army poster's Polish body
  goes from three cross-column garbled plates to fifteen clean line plates; samson-15's
  cross-balloon stitches (`THE WRATH OFA \ YOU ARE THE`) separate; the Comics Authority marks on
  two Archie covers plate for the first time; the Superman-page class of merge no longer occurs.
- **Cost:** totalOcrMs 28.5 s -> 29.7 s over the 14 scored scenes (+4%); the sweep spends its pass
  only where wordless regions stand outside the plates.

## What was tried and rejected

- **A lab scene for the fused-token stitch.** Two drawn fixtures (rectangle outlines, then
  ellipse rings, tuned gutters and clearances) failed to reproduce the Superman shape on both
  rows: the engine either fused the facing letters plus both outlines into one normal-height token
  (uncuttable by any between-words rule) or stopped reading one balloon outright. The defect is
  engine behaviour on specific art, not geometry a synthetic scene can hold; the harness fixtures
  above carry the regression instead, and the corpus row for the attempt was removed.
- **The sparse rung in the sweep** - rejected above on the page's own evidence.
- **Replacing the banner's two garbled fragments** (`SORE IN`, `OPERATION WITH THE AD'`, both
  admitted by the ordinary floor at 56-60 before this change existed) - the sweep is additive by
  contract; a replace policy has never been measured and stays out of scope. The fragments stand,
  recorded in the ticket.

## Runs

| run | tree | language | change |
|---|---|---|---|
| `temp/ocrlab/t67-baseline` | `HEAD` (`c32e341`, worktree) | `eng` | none (baseline) |
| `temp/ocrlab/t67-fix` | this change | `eng` | amendment 1.7 |

The page's own verification: `temp/verify67/` - `ocr-diag.jsonl` (before), `ocr-diag-fixed.jsonl`
(after), the converted output under `jpg-default`/`fixed`, and the render at
`temp/verify67/render/page-01.png`.
