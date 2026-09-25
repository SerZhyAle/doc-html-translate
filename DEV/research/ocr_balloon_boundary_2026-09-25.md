# Balloons drawn side by side are stitched into one line at a gap no ratio can tell from a real line

2026-09-25. Feeds Phase 07 Step 07.3 of
[`DEV/plan/16_2026-08-11_ocr-visual-fidelity-lab.md`](../plan/16_2026-08-11_ocr-visual-fidelity-lab.md)
([`PHASE_07__concealment-and-grouping.md`](../plan/16_2026-08-11_ocr-visual-fidelity-lab/PHASE_07__concealment-and-grouping.md)).
Follows [`ocr_word_gap_2026-09-12.md`](ocr_word_gap_2026-09-12.md), which left this band open on purpose.

## The question

The word-gap rule of 2026-09-12 cuts a recognizer line wherever two consecutive words stand more than
`ocrMaxWordGapRatio` (3.5) median word heights apart. Below that it cannot decide: on the same corpus
comic balloons drawn side by side are stitched at 1.87-3.46x while real lines reach 2.57x. The phase
reconciliation asked for this band to be re-measured on the corpus before Step 07.3 was written, and
the step's substance - an *added*, named boundary condition from the pixels, not a retuned ratio - to be
kept if the band held. It holds, and it is wider than the 2026-09-12 note said.

## Method

Desktop engine (tesseract 5.4.0 on this machine, `eng`, PSM 3, the app's own staging - `prepareForOCR`,
so upscale and declared DPI are the shipped ones), over all 46 scenes of `DEV/ocrlab/corpus.json` plus
`test_doc/1.png`. For every line that clears `ocrMinLineConf` (50) and every pair of consecutive words:
the gap over the line's median word height, and pixel features of the strip between the two boxes on the
frame's own luminance. **1043 gaps.** Every gap the verdict depends on was then labelled by eye from a
crop of the scene: `S` - a stitch between two regions (two balloons, two notices), `R` - one real line,
`A` - an outline or scrap of artwork the recognizer read as a token of the line, `X` - separate items on
paper, `N` - noise. The harness was a throwaway test file in `internal/ocr` and is not committed; the
last section says how to rebuild it.

## The band is wider than recorded, and it reaches down to one word height

The 2026-09-12 note put the lowest stitch at 1.87x. The desktop engine also stitches at **1.00x and
1.16x**: the recognizer reads one of the two outlines as a bracket and folds it into a word
(`HILLS/)` then `MUIRK!`, `IS)` then `COMPANION`), so what is left between the words is one outline and a
narrow strip. So the overlap is 1.00-3.46x on the stitch side against real lines up to 3.07x (the
letter-spaced `Petit Journal` masthead) - no ratio separates anything in it.

## Ink in the gap is not enough

The first features asked whether the strip between the words holds ink at all (share of rows with an
off-paper pixel, the tallest single column). They separate most stitches, and they also fire on real
lines, every time for the same reason: the recognizer left a letter out of its word box, so the letter
sits in the "gap" - the T of `DON'T`, the V of `IV. VINTER-`, the `n` of `in God;`, the colon after
`ministration`. Measured on the labelled band: a row-coverage feature cut 7 of the 26 real lines.

Two more lessons from the first pass, both cheap to repeat by accident:

- **Paper is not the median of the word box.** Bold comic lettering fills about half of a tight word
  box; on `samson-and-delilah-15` the median inside the boxes read 111 on a balloon whose paper is 228.
  Two rows above and two below each word, skipping the row that touches the box, read 220-231 there.
- A letter's residue and a balloon outline look alike *inside* the line's band. They differ outside it.

## The rule: a stroke that runs on past the line

A balloon outline is drawn clean through the line and continues above and below it; a letter stays
inside the band of the words around it. So the test is a path of ink, 8-connected, through the strip
strictly between the two word boxes, from `reach` pixels above the two words' band to `reach` pixels
below it, where ink is a pixel at least `plateMinContrast` (55) from the paper in luma. That contrast is
the overlay's own minimum between a plate's paper and its ink; on the labelled band every threshold from
20 to 128 separates `R` from `S` identically, so no new number was introduced for it.

### `ocrBoundaryReach` - the one new constant

`reach` is `ocrBoundaryReach` times the line's median word height. Swept on the labelled gaps and on all
1043, with the shipped function:

| reach | real lines cut (of 26 labelled) | stitches caught (of 9 labelled, band 1.2-4.0x) | what changes |
|---:|---:|---:|---|
| 0.05 - 0.07 | **1** | 9 | the J of `Petit Journal`, left out of its box, descends that far |
| 0.08 - 0.25 | 0 | 9 | - |
| 0.30 | 0 | **8** | the stitch between two paper notices on a photographed wall is lost |
| 0.50 | 0 | 8 | a balloon stitch (`HILLS/)` - `MUIRK!`, 1.00x) is lost |
| 0.75 | 0 | **6** | two more: an outline beside a balloon's first or last line turns away before it reaches that far |

**0.14 is the geometric middle of the two measured failures, 0.07 and 0.30** - about 2x of margin each
way. The one-sided variant (a stroke past the band on either side, not both) was measured too and cut
the `Petit Journal` masthead at every reach: a descender runs below the line and nowhere above it.

### What the rule cuts on the whole corpus

Under `ocrMaxWordGapRatio`, across all 1043 gaps, the test fires **21 times and never on a real line**:

| class | count | examples |
|---|---:|---|
| stitch between two regions | 11 | 10 balloon pairs on `samson-and-delilah-15` / `-03` (`MUST] - HIS` 1.87x, `TO} - THOUSAND` 2.12x, `BE - WHO` 3.04x, `THE - YOU,` 3.46x, `HILLS/) - MUIRK!` 1.00x, `IS - COMPANION` 1.16x), two notices on a wall |
| outline or artwork read as a token | 8 | `ARCHIE - |`, `WILL - |`, `| - ISAMSON,`, `ME. - }`, `€ - Owings`, `tii - PLACE!`, `A - RUSSKIES`, `gs - "STRENGTH` |
| noise line | 2 | a binarized scan, a photographed wall |

Of the 878 gaps under 1.00x - ordinary word spacing - it fires twice, both in front of an outline read
as `|`. No ratio floor is needed to keep it off ordinary spacing, so none was added.

## The fragment a stroke cuts off: the regression the first run found

The first full-corpus run with the rule found one scene that got worse. On `atomicwar0401` a speck of
artwork beside a balloon is read as the word `A` at the head of the balloon's last line; the outline
between them cuts it off, correctly. But the page's headline spans every column, so the whole page is
one column, and the speck lands in it at exactly the height of the line it was cut from - between two
lines of one balloon. Its type size is a quarter of theirs, so the clustering closed the plate there:
one balloon, two plates.

Every run a stroke cuts off that cannot be a plate on its own (`isTranslatable`) is artwork on this
corpus - all 8 of the `A` row above - and every real fragment passes that test (`TAKE THE`,
`COMPANION`, `WHO STOLE`, `1 SUDDENLY FEEL`). So such a run is marked an **orphan** and `orderColumns`
parks it at the end with the lines the confidence floor will drop; the translatability gate records it
there (OCR-OVERLAY rule 12). A fragment cut by the ratio rule is not touched - that path was measured on
2026-09-12 and nothing here says it needs to change.

## Results

### Regression scene - `synth-side-by-side-balloons`

Added to `tools/ocrlab/synth` (Step 07.7). Two balloons side by side, lettering facing each other across
both outlines, recognized as two stitched lines at **2.9x** with the app's staging. Both editions, the
stroke test switched off in the source for the "off" run and restored byte for byte:

| edition | stroke test | plates | merges | cross-group (6 stress cases) | protected damage |
|---|---|---:|---:|---:|---:|
| desktop | off | 1 | 1 | 6 | 448 px |
| desktop | on | **2** | **0** | **0** | **0** |
| extension (tesseract.js 7.0.0, Chrome 151) | off | 1 | 1 | 6 | 448 px |
| extension | on | **2** | **0** | **0** | **0** |

The scene's construction matters and is written into its comment: with the lettering 6-10 px from the
outlines the recognizer reads both outlines as a word of their own (`ff`, `fj`, `})j`), both inside one
box, and no gap holds a stroke - see "What is left".

### The corpus, desktop edition

`temp/ocrlab/p16-base` (before, 46 scenes) against `temp/ocrlab/p16-final` (after, the same 46 plus the
new scene), tesseract 5.4.0, Chrome 151. 7 of the 46 scenes change plates:

- `samson-and-delilah-15`: 35 -> 45 plates. Cut apart: `WOMAN SCORNED 157 | COMPANION`,
  `OH KING? | TAKE THE`, `...MEN TO} | THOUSAND MEN,`, `...HILLS/) | MUIRK! AND`, `KILL YOU! | OF YOUR
  HOME!`, and most of a plate that spanned five balloons (`NOT FIND HIM... FEARED! I MUST] HIS
  BRIDE!...`, 1531x153 px). Two plates still cross balloons - see "What is left".
- `samson-and-delilah-03`: 9 -> 8. `RUN, SAMSON! THE BULLY? | 1 SUDDENLY FEEL` cut apart, and the right
  balloon, torn into three plates by the token `gs` before, is one plate.
- `atomicwar0401`: one plate for the balloon, as before, its box now starting at x=652 where the
  lettering starts instead of x=618 over the outline and the speck. (Before the orphan rule: two plates.)
- `cover-of-archie-and-me-no-1`: `ARCHIE |` becomes `ARCHIE`, the box no longer over the outline.
- `handwritten-notices-...` 21 -> 17 (lists of stops on one notice now one plate each),
  `diaz-political-cartoon` and `cover-of-archie-s-pals-n-gals-no-25` (one new plate each: a misread state
  name, and the cover's own logo lettering `PALS'N'GALS`).

`ocrlab gate` over the 14 annotated scenes fails on the same 6 checks for both runs - the stale
`thresholds.json` the queue records (recall, texture IoU, concealment, review count, cost) - and is
equal or better on every line: recall 0.6923 -> 0.7143, comic recall 0.75 -> 0.80, comic IoU 0.7532 ->
0.7672, OCR time 33.9 s -> 33.5 s; every hard gate (damage, merges, clipping, cross-group) at 0 in both.
The overall mean IoU moves 0.8344 -> 0.8334 only by averaging in the new scene.

## What is left, measured and not fixed here

- **An outline taken for letters.** When both outlines of a balloon pair are read as one token
  (`ff`, `fj`, `J`, `})j`, `4`), the gaps on either side are clean balloon interior and no stroke
  crosses them. Reproduced on the synthetic scene at tight padding. On `samson-and-delilah-15` two plates
  still span balloons this way (`HIS BRIDE! KEEP THIS IN MY | IF HE SEES...`, `US WHERE HE IS AND /
  WE WILL...`), with the outline read as a separate `|` or `/`; the first one now also takes the line
  `ISAMSON, FOR THE L055 | LAST`, which the orphan rule lets rejoin the column it stands in. A test through the token itself - a
  stroke running past the token's own box - would see them, but it also fires on a real word the
  recognizer glued an outline to (`TO}`, `MUST]`, `HILLS/)`), and deciding which side of such a word to
  cut is a second design, not a threshold. Left for its own measured step.
- **Protected-area damage from an absorbed outline** at the end of a run (`GO? fj`) remains when it
  happens: the plate's box includes the token.

## Reproducing

Throwaway, under `internal/ocr` as a `_test.go` file so it can call the unexported staging: for each
`file` in `DEV/ocrlab/corpus.json`, `prepareForOCR`, `runTesseract` with `tesseractArgs(..., ocrPageSegMode)`,
parse the level-4/5 TSV rows into lines of words, keep lines whose mean word confidence clears 50, and
for every consecutive pair record the gap over the median word height and `strokeBetween(frame.grey(),
a, b, int(med * reach))` for each reach in the sweep. Crops for labelling: the two words' boxes padded by
half a word height, stacked into contact sheets with the gap marked. The extension edition is
`npm run ocrlab -- --scene <id>` from `extension/`, scored with `go run ./tools/ocrlab score <dir>`.
