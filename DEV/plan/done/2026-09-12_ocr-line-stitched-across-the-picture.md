# A "line" the recognizer stitched across the picture becomes a bar across the artwork

**Status:** Implemented
**Priority:** 46
**Date:** 2026-09-12

> **Implemented 2026-09-12** in both editions, with the measurement behind the constant in
> [`DEV/research/ocr_word_gap_2026-09-12.md`](../../research/ocr_word_gap_2026-09-12.md). What landed:
>
> - **A line is cut before the clustering ever sees it**, between two consecutive words whose boxes
>   stand more than `OCR_MAX_WORD_GAP_RATIO (3.5)` times the line's median word height apart. Each
>   run is boxed to its own words and carries its own mean confidence. Bracketed over the 46 lab
>   scenes plus `test_doc/1.png`: the widest gap inside a real line is 2.57x, the narrowest
>   cross-region stitch above it is 4.80x, 3.5 is the geometric middle.
> - **The cut runs are put back in reading order** (`orderColumns`), because cutting alone trades one
>   oversized plate for three fragments - measured, the left balloon on `test_doc/1.png` went
>   1 bar -> 3 fragments -> 1 plate. A page nothing was cut on keeps the engine's order.
> - **Both editions land on the same ten plates** over the reported image's nine balloons, in reading
>   order, none of them touching the artwork. `synth-two-columns` - the corpus's one merge, named in
>   `thresholds.json` - goes **1 plate crossing the gutter -> 2 plates, one per column**, matching
>   both of its hand-drawn transcripts verbatim.
> - **The band 1.87-2.57x was measured, found to overlap, and deliberately left alone.** Comic
>   balloons drawn side by side stitch there while real lines reach into it, so no ratio separates
>   them; the consequence (some adjacent-balloon stitches stay merged) is recorded rather than hidden.
> - Both editions, one shared invariant, pinned by `tests/parity_test.go` on the constant **and** on
>   the three expressions that carry its meaning.

## The report

`test_doc/1.png` - a photograph with ten comic speech balloons over it, five on each side of the
figure - converted through the browser extension. The overlay produced **three grey bars across the
full width of the picture**, each covering the man's face or torso and carrying one speaker's words
run into the other's:

> "It's about your facial hair. I love everything about you, but that patchy mustache and beard.. it's
> just not working for me. **Oh, come on, Em. It's**"

9 plates over 10 balloons, three of them wrong in both geometry and content.

## Why

Not our clustering. **The recognizer's own line assembly** stitches the two columns: on this image
PSM 3's layout analysis walks across the figure and returns five line boxes 987-1727 px wide, each
holding text from both sides. Both engines do it - the desktop `tesseract` CLI and tesseract.js
return the same boxes.

Nothing downstream can recover, and each rule that looks like it should fails differently:

- the clustering's column test sees a **real** x-overlap, because the stitched box honestly spans
  both columns;
- `OCR_MAX_PLATE_COVERAGE (0.52)` does not fire - the bar is wide but short, 1593x105 px on a
  2048x2048 image is **0.04** of it;
- pitch and type size compare a line with its neighbours, and by then the two texts *are* one line.

The full evidence, the corpus distribution and the bracketing are in
[`DEV/research/ocr_word_gap_2026-09-12.md`](../../research/ocr_word_gap_2026-09-12.md).

## What it costs a reader

The worst outcome the OCR overlay has: not text that is missing, but **artwork covered by a plate
that should never have existed**, carrying a sentence neither speaker said. A translator then
translates the run-on as one sentence, so the damage survives into every language. This is the
`merges` hard gate in `DEV/ocrlab/thresholds.json`, fixed at 0 by the strategic spec, and
`synth-two-columns` - the corpus's one known merge - is the same defect at 12-18x.

## Done criteria

- [x] A line the engine stitched across separated regions is cut into those regions, on a threshold
      bracketed from the corpus rather than chosen.
- [x] No line a reader would call one line is cut: verified over all 199 corpus lines that clear the
      confidence floor. 36 lines are cut, on 9 scenes, every one a genuine stitch, checked by eye.
- [x] The fix does not trade the merge for splits: `orderColumns` measured on the reported image and
      on `synth-two-columns`, which goes to exactly its two annotated reading groups.
- [x] Both editions move together, with the invariant and the expressions that carry it pinned in
      `tests/parity_test.go` and recorded in `docs/PARITY.md`.
- [x] Unit tests on both sides, on fixtures taken from the real recognizer output rather than
      invented.
- [ ] **Not done: a full lab re-baseline.** `npm run ocrlab` was run on `synth-two-columns` alone,
      which is the scene the corpus names for this defect. A whole-corpus run would also have to
      re-derive `thresholds.json`, and that is blocked on the concealment gate already being red for
      an unrelated reason (see `2026-08-15_release-1-worklog.md` §1.2) - it belongs to Phase 06's re-derivation, not
      here.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` | `splitWideGaps` / `lineFromWords` / `orderColumns` in `internal/ocr/tesseract.go`, applied in `parseTSV` per recognizer paragraph |
| GUI (`doc-html-ui`) | `[x]` | inherits the pipeline; no new flag, nothing to expose |
| MSIX Store app | `[x]` | inherits the GUI; no packaging impact |
| Browser extension | `[x]` | `splitWideGaps` / `orderColumns` in `extension/src/ocr-cluster.js`, applied in `collectLines` (`ocr-overlay.js`); tests mirror the Go ones case for case |
| Website / docs | `[x]` | `docs/PARITY.md` gains a "Line integrity" row and the invariant note; the research file carries the measurement |

## Shared invariants touched

- **New:** `ocrMaxWordGapRatio` / `OCR_MAX_WORD_GAP_RATIO = 3.5`, and the three expressions that give
  it meaning - the gap is measured between the boxes (not left-to-right, so a right-to-left line is
  not exempt), it is weighed against the median of the line's own word heights (not the line box),
  and each run is boxed to its own words.
- **New:** `orderColumns` on both sides, and two rules about it that cost a cycle each to find: its
  scope is the **page**, not the recognizer paragraph (the engine invents paragraph boundaries
  mid-column - `synth-two-columns` puts the third row of both columns in a second one), and its
  columns are formed only from the lines that clear the confidence floor (a full-page empty "line"
  the engine returns on `test_doc/1.png` otherwise chains both columns into one, and the desktop
  edition came out at 19 plates). It runs only on a page something was cut on.
- `ocrWord` gains `conf` / `hasConf` (Go), so a cut run's mean confidence is its own rather than the
  stitch's.

## What is deliberately left open

- **Adjacent balloons that stitch below 3.5x stay merged** (`samson-and-delilah-03/15`, nearest
  declined case 3.46x). Geometry cannot separate them - the measured bands overlap - and the evidence
  that can is the pixels between the two words. That is Phase 07 Step 07.3 of
  [`16_2026-08-11_ocr-visual-fidelity-lab`](../16_2026-08-11_ocr-visual-fidelity-lab/PHASE_07__concealment-and-grouping.md),
  whose objective this ticket does not replace: 07.3 adds a boundary test to the *clustering*, this
  repairs the clustering's *input*. Both are needed.
- **One tail fragment on the reported image** ("better.") stays its own plate, in both editions. It
  is a line-pitch question on a balloon whose last two line boxes the engine overlaps, not a stitch.
- `I` read as `|` throughout `test_doc/1.png` is an engine/typeface question and is not this ticket.
