# A poster the app reads as one word - reproduction sweep, 2026-08-12

**Ticket:** [`2026-08-12_ocr-misses-display-lettering-on-saturated-art`](../plan/done/2026-08-12_ocr-misses-display-lettering-on-saturated-art.md)
**Question:** a user-reported standalone poster produces a single plate holding one word. Which part
of the shipped path loses the other ten, and does anything the code already contains recover them?
**Scope:** one image. This is a **reproduction**, not a corpus measurement - nothing here chooses a
threshold, and no number below may be used to justify landing a change on its own. The tuning
evidence is the corpus run the ticket requires.

## Material

A 960x1280 JPEG of a Soviet-style poster: eleven words of Cyrillic display type, red and black over
flat cream, with a two-figure illustration occupying the right half. Estimated DPI 116 by
`estimateDPI`, which is under `ocrUpscaleDPIFloor` - so the app stages it at 2x and declares 232 DPI.
The sweep reproduces that staging rather than feeding the original.

Ground truth, read from the image: `ЗАЧЕМ ТРАХАТЬСЯ: МЫ ЖЕ ВЗРОСЛЫЕ ЛЮДИ, МОЖЕМ ПРОСТО ОБ ЭТОМ
ПОГОВОРИТЬ` - 11 words.

## What the app does today

```
OCR overlay: 1 image(s) overlaid
```

One plate, transcript `| МОЖЕМ`. The log is not wrong and that is the problem: the run reports a
success, and the reader gets a stray box.

## Sweep

`tesseract 5.4.0.20240606`, `-l rus` from the app's own tessdata, `tessedit_create_tsv=1`,
`user_defined_dpi=232`, the 2x staged image unless noted. Metric: words at the app's two line-gates,
`ocrMinLineConf` = 50 and `ocrRescueLineConf` = 80.

| Configuration | words at 50 | words at 80 |
|---|---:|---:|
| colour + PSM 3 - **ships today** | 0 | 0 |
| colour + PSM 4 | 6 | - |
| colour + PSM 6 | 1 | - |
| colour + PSM 11 (sparse) | 16 | 6 |
| colour + PSM 12 (sparse + OSD) | 15 | - |
| grey + PSM 3 - **rescue rung 1** | 10 | 5 |
| **grey + PSM 11** | **10** | 6 |
| colour + PSM 11, `thresholding_method=1` | 37 | - |
| colour + PSM 11, `thresholding_method=2` | 26 | - |
| colour + PSM 3, `thresholding_method=1` | 0 | - |
| colour + PSM 11, 1x | 25 | - |
| colour + PSM 11, 3x | 22 | - |

Transcripts matter more than counts here, because two configurations reach 10 words and only one of
them is usable:

- **grey + PSM 11:** `ЗАЧЕМ ТРАХАТЬСЯ: МЫ ЖЕ ВЗРОСЛЫЕ ЛЮДИ, МОЖЕМ ПРОСТО ОБ ПОГОВОРИТЬ` - 10 of the
  11 words, in reading order, no debris. Only `ЭТОМ` is missing.
- **grey + PSM 3:** ``ЗАЧЕМ ВЗРОСЛЫЕ ‹ ЛЮДИ, ` | МОЖЕМ ПРОСТО ОБ ЭТОМ`` - four real words fewer, and
  three pieces of punctuation debris (a guillemet, a backtick and a pipe) counted among the ten.
- **The high-count rows are noise, not recall.** `thresholding_method=1` scores 37 words because it
  counts `2%`, `и:`, `2$`, `+;`, `№` and `у` alongside the real ones, and it splits `ТРАХАТЬСЯ` into
  two. A stricter gate would be needed to use it at all.

Per-word confidence for the candidate, at the rescue floor: `МЫ`(96) `ЖЕ`(96) `ЛЮДИ,`(91)
`МОЖЕМ`(94) `ПРОСТО`(90) `ПОГОВОРИТЬ`(95). These sit in the band the halftone cycle recorded for
genuine lettering (93-97), not in the hallucination band.

## Findings

1. **The grey rendition is the lever, and it is already written.** `greyRescue` was built for exactly
   this material - per-channel Otsu disagreeing over saturated colour - and it takes the image from 0
   confident words to 10. It never ran, because the ladder fires on `len(res.Blocks) == 0` and the
   ordinary pass produced one junk plate. One weak plate suppresses every rescue the app has.
   **Corrected on the same day - the second half of this finding is wrong.** The ladder does run on
   this image, in both language configurations, and it still returns almost nothing. The word counts
   above are per word; the app gates on a line's *mean* word confidence, and that is where the
   transcript is lost. See "What the shipped path actually does, measured line by line" below.
2. **Segmentation is the second lever, worth about three lines of text.** On the grey rendition, PSM
   11 recovers `ТРАХАТЬСЯ`, `МЫ ЖЕ` and `ПОГОВОРИТЬ`, which PSM 3 loses. This does not contradict the
   wave-4 roadmap note that "PSM 3 is not the lever" - that was measured on comic balloons and clean
   page renders, where layout analysis has a layout to analyse. A poster has none.
3. **Resolution is not a lever here.** 1x, 2x and 3x all land in the same band; the staging is not
   what loses the text.
4. **Neither lever is corpus-proven.** Both sweeps that shipped before this one recorded a transform
   winning on its target scene and destroying results elsewhere. Nothing here has been run against
   anything but this poster, which is why the ticket's first done-criterion is annotating the scene
   into the corpus rather than changing code.

## What the shipped path actually does, measured line by line

The sweep above counts words. The app does not: `clusterLines` drops a whole recognized line whose
*mean* word confidence is under the floor, so a line holding one 97-confidence word and one
30-confidence fragment is discarded entire. Re-measured with the app's own staging and its own gates,
per line rather than per word:

| Rung | lines >= 50 | lines >= 80 | what survives the rescue floor |
|---|---:|---:|---|
| colour + PSM 3 (ordinary pass) | 1 | 0 | nothing - the pass returns no plates at all |
| grey + PSM 3 (rung 1) | 3 | 1 | `\| МОЖЕМ` (mean 85.5) |
| grey + PSM 3, Leptonica (rung 2) | 1 | 0 | nothing |
| **grey + PSM 11 (the new rung)** | 8 | **6** | `ТРАХАТЬСЯ:`(80.7) `МЫ ЖЕ`(96.1) `ЛЮДИ,`(92.6) `МОЖЕМ`(95.9) `ПРОСТО`(87.2) `ПОГОВОРИТЬ`(95.0) |
| grey + PSM 11, Leptonica | 28 | 6 | the same six, plus 200-odd debris lines under the floor |

Two corrections to the findings above follow from this table.

**The outer trigger is not what suppressed the cure.** The ordinary pass returns *no plates* on this
poster in both language configurations - the `| МОЖЕМ` line scores 51.8 as a colour pass and is
dropped by the ordinary floor of 50 only just, and with `rus` it is dropped outright. So
`len(res.Blocks) == 0` holds, the ladder runs, and the eng conversion spends eleven seconds climbing
every rung of it before reporting "no text found". A weakness floor on the outer trigger would have
changed nothing here, and none of the corpus's scenes demanded one, so none was added: the strategic
spec's first cause is recorded as unproven rather than quietly implemented.

**What suppressed the cure is inside the ladder.** `greyRescue` returned the first rung that produced
any plate at all. Rung 1 produces exactly one - `| МОЖЕМ` - and that ended the search before the rung
that recovers six lines could run. The fix is therefore a comparator, not a threshold: every rung
runs, and the strongest result wins, with a tie keeping the earlier rung (`resultStrength` /
`strictlyBetter`). Strength is counted in words rather than in plated area, because over the recorded
desktop baseline `temp/ocrlab/base06-desktop` the scenes that read carry 2 to 23 words, while their
plated area ranges from 3.9% to 20.9% of the image - and this poster's one recovered word covers 1.7%
of a much larger image, which is a difference in image size rather than in how much was found.

## End to end, after the change

Same command the reporter ran, with `rus` data:

| | before | after |
|---|---|---|
| plates | 1 | 1 |
| transcript | `\| МОЖЕМ` | `ТРАХАТЬСЯ: МЫ ЖЕ ЛЮДИ, МОЖЕМ ПРОСТО ПОГОВОРИТЬ` |
| words a reader can translate | 2 | 7 |
| plate colours | `background: rgb(47,42,32)`, `color: rgb(198,174,139)` - the negative of the poster | `background: rgb(226,185,143)`, `color: rgb(164,56,36)` |
| wall clock | 3.8 s | 6.1 s |

The extra 2.3 seconds is the ladder no longer stopping at its first non-empty rung: on an image the
ordinary pass could not read, it now pays up to two more recognitions. An image that reads normally
pays nothing - it never enters the ladder.

**The colour inversion was a separate defect, found from this scene and reported by the user.**
`blockColors` took the median colour over the block as the paper, which assumes the text is the
minority of its own box. That holds for body text inside a balloon and fails for heavy display
capitals, whose strokes fill more of a tight box than the paper between them - so the plate came back
as the photographic negative of the word it covered. The paper is now decided by the band just
outside the block, which is paper in both cases.

**Its port drifted, and the parity test did not see it (2026-08-13).** The ring is derived from the
line height; the ink is sampled in a strip `1.3 x` that height. The extension held both in one
variable and handed the strip to the ring, so it weighed a band `0.43 x` the line height where the
desktop app weighs `0.33 x` - the same block, two different votes, on the one decision that keeps a
plate from coming back inverted. `TestParityOCRPlateColourOrientation` pinned the vote floor (40) and
the shape of the call, and a drift in what the *argument means* passes both. It now also pins the
band divisor on each side, that the two heights are kept apart by name, and that the ink is a median.
The same read found `docs/PARITY.md` still describing the ink as a **mean**, which both editions
stopped computing on 2026-08-13 - the row was correct when written and was not re-read when the code
under it moved.

## Against the corpus

Dev split, desktop edition, `temp/ocrlab/p46` against the baseline `temp/ocrlab/base06-desktop`.

**Nothing regressed.** All eleven scenes the baseline scored keep their recall to the digit -
`samson-and-delilah-15-court-caption`, the five reading synth scenes and `synth-uniform-paper` stay
at 1.00, and the three that read nothing before still read nothing. The gate's two hard rules
(`reading groups merged` 1, `plates crossing another group` 6) come out identical to the baseline's,
which already fails both - a standing failure this ticket neither caused nor closed.

The gate still exits 1 on the new run, for two reasons that are not regressions and are named rather
than argued away:

- **Mean recall 0.727 -> 0.615 is arithmetic, not loss.** Two scenes joined the scored set and the
  lab recognizes with `eng` by default, so two Cyrillic posters can only score zero: `0.727 x 11 / 13
  = 0.615`. Per scene, nothing moved. That the runner has one language for the whole corpus is a gap
  in the instrument, and it is the reason the two scenes were re-run on their own below.
- **Total OCR time 18.5 s -> 36.4 s is not a usable measurement.** The full Go test suite was running
  on the same machine at the same time, and the cost rose on scenes that never enter the ladder
  (`synth-adjacent-balloons` 578 -> 978 ms, which nothing in this change touches). The controlled
  number is the serial end-to-end run above: 3.8 s -> 6.1 s on an image that does enter it.

Re-running the two Cyrillic scenes with their own language (`-lang rus`, `temp/ocrlab/p46-rus`):

| Scene | before | recall | CER | OCR ms |
|---|---|---:|---:|---:|
| `a-propaganda-poster-...-1920s` | no plates at all | **1.00** | **0.00** | 3257 |
| `poster-display-type-on-flat-colour` | one plate, `\| МОЖЕМ` | 0.00 (tp 0, fn 2) | - | 10671 |

**The sibling scene is the clean result:** an image that produced nothing now produces a plate over
its slogan with an exact transcript.

**The reported scene is not, and the ticket's own done-criterion is not met on it.** A reader gets
seven words instead of two, but the lab scores the scene as a miss: the six recovered lines land in
one plate, and that plate matches neither annotated group at the IoU floor - it spans both. Recall
stays 0 with two false negatives, and the plate is recorded as crossing a reading group at every
stress case. Grouping, not recognition, is what is left.

**One measured cost is not paid for: the six recovered lines land in a single plate.** `clusterLines`
groups by the page's median line pitch, and with three of the nine lines still missing, the surviving
steps (188, 190, 335, 363, 381 px in the staged space) put the median at 335, wide enough to join the
headline to the body. The union box spans 65% of the poster's width and 62% of its height, so it
covers part of the illustration. That is a grouping question rather than a recognition one, it needs
its own measurement, and it is recorded here rather than fixed blind.

## The grouping question, measured and answered (2026-08-13)

The paragraph above was written expecting the answer to be a pitch bound. It is not, and the reason
is worth stating before the fix: **no pitch bound separates this poster in the right place.** The
line boxes the sparse rung actually returns, in the 2x staged space, at the rescue floor:

| line | box | ink height | step from the line above |
|---|---|---:|---:|
| `ТРАХАТЬСЯ:` (80.7) | `[74,415,1328,696]` | 281 | - |
| `МЫ ЖЕ` (96.1) | `[79,750,521,905]` | 155 | 335 |
| `ЛЮДИ,` (92.6) | `[76,1131,456,1302]` | 171 | 381 |
| `МОЖЕМ` (95.9) | `[79,1319,538,1472]` | 153 | 188 |
| `ПРОСТО` (87.2) | `[80,1509,477,1658]` | 149 | 190 |
| `ПОГОВОРИТЬ` (95.0) | `[80,1872,644,2009]` | 137 | 363 |

The step a reader would cut at - headline to body - is **335 px**, and a step *inside* the body, over
the gap where `ВЗРОСЛЫЕ` was not recognized, is **381 px**. The cut a reader wants is the smaller of
the two. Any threshold on pitch that separates the headline also tears the body into three, and the
lab would score that as a split instead of a merge.

What does separate them is what a reader actually uses: **type size.** 281 px of display ink over a
155 px body median is 1.81x, and nothing inside the body comes near it (171/153 = 1.12 at worst). So
the rule added here is a second condition on joining a cluster, beside pitch and shared column: the
line's ink height must be within `ocrTypeSizeRatio` of the cluster's own median height, either way
round (`sameTypeSize`).

**Where 1.6 comes from, and why it is not a round number picked near one measurement.** The value has
to clear the spread a single text shows on its own and stay under the step between two texts. Both
bands are measured on the corpus's **hand-drawn line boxes** - human ground truth, never a
recognizer's output, which is rule 1 of the lab:

| | scene | measurement |
|---|---|---|
| widest spread inside one text | `samson-and-delilah-03-scroll` | 19 lines of one caption, ink 23-34 px; the worst line is **1.42x** its own group's median |
| next widest | `poster-display-type-on-flat-colour` body | 7 lines, 69-86 px, worst **1.12x** |
| narrowest step between two texts | `poster-display-type-on-flat-colour` | headline over body, **1.86x** |
| next narrowest | `synth-display-lettering` | headline over footnote, **2.57x** |

1.6 is the geometric middle of 1.42 and the recognized 1.81 - about 13% of margin on each side. A
value chosen next to either measurement would have had 2% of margin on that side.

### Against the corpus

Dev split, desktop edition, `temp/ocrlab/p46b` against `temp/ocrlab/p46`, same 13 annotated scenes:
**detection, grouping and placement are identical on twelve of them, to the digit** - same recall,
same tp/fp/fn, same merges and splits. The thirteenth is the reported poster, which reads nothing
under the lab's default `eng` and can only be measured with its own language.

Re-run with `-lang rus` (`temp/ocrlab/p46b-rus` against `temp/ocrlab/p46-rus`):

| Scene | recall | tp / fp / fn | plates crossing another group |
|---|---:|---|---:|
| `poster-display-type-on-flat-colour` before | 0.00 | 0 / 1 / 2 | 6 (one per stress case) |
| `poster-display-type-on-flat-colour` after | **0.50** | 1 / 1 / 1 | **0** |
| `a-propaganda-poster-...-1920s` before and after | 1.00 | 1 / 0 / 0 | 0 |

The two plates the app now writes, from its own diagnostics sidecar:

```text
ТРАХАТЬСЯ:                            [37,208,664,348]   lineH 141
МЫ ЖЕ ЛЮДИ, МОЖЕМ ПРОСТО ПОГОВОРИТЬ   [38,375,322,1005]  lineH 77
```

The body plate matches the annotated `body` group at **IoU 0.93** and its transcript is compared
against the annotation, which is what the ticket's second done-criterion asked for.

**The headline is still a false negative, and the reason is not grouping.** Its group is two lines
and only one of them was recognized: `ЗАЧЕМ` scores **69.2**, under `ocrRescueLineConf` (80). A plate
over one of two lines reaches IoU 0.455 against the group's bounds, just under the lab's 0.5
detection floor, so the scene tops out at recall 0.50 while that floor stands. The tactical plan put
re-deriving the rescue floor out of scope and said it becomes a follow-up ticket **if the run shows
the floor rejecting genuine lettering**. It does, twice on this one image (`ЗАЧЕМ` 69.2, `ОБ ЗЛОМ`
73.9), so the follow-up is opened rather than the floor nudged here.

**The size rule's own contribution, isolated.** Re-running the same scene with `ocrTypeSizeRatio`
neutralised and everything else identical (`temp/ocrlab/p46b-nosize`) gives recall 0.00, 6 cross-group
overlaps and residual ink 0.999; with the rule, 0.50, 0 and 0.915. The one cost the isolation shows
is a trade under stress: the headline plate is small enough that three of the six stress translations
clip in it, where the merged plate had room. Six cross-group overlaps in every case, including the
untranslated one, for three clips in the three longest cases.

**What the rule does not fix, stated so the boundary is not overclaimed.** It separates *sizes*, not
*regions*. Re-converting the sweep's worst case, `accounts.jpg`, after the change: the header splits
off (its ink is 22 px against the list's 51) and the picture goes from **one plate over 80.6% of the
image** to two, the larger covering **68.3%**. Better, and still a defect - the list's own rows share
one type size, so nothing here separates them. Same-size separated regions remain
[P47](../plan/2026-08-13_ocr-sweep-plate-composition.md).

### Found while measuring, and not caused by this ticket

**Concealment collapsed across the whole corpus between `p46` and `p46b`, and it is the 2026-08-13
plate-shape change rather than this one.** Residual ink - how much of the original lettering is still
visible under the plates - went from a mean of 17% over 46 scenes to a mean of **93%**, worst 100%,
and the gate's bound (0.28, set at the recorded baseline) now fails at 0.9996. The attribution is not
an inference: `synth-uniform-paper` draws three lines of one height, so `sameTypeSize` cannot fire on
it, and its diagnostics record a byte-identical plate - same box, same text, same `style` string -
across the two runs while its residual moved 0.085 -> 0.998. Same plate, different picture underneath
it: the change is in how a plate is drawn, not in how lines are grouped.

That is [P47](../plan/2026-08-13_ocr-sweep-plate-composition.md)'s second defect, which the sweep
found by eye. It also answers, with the corpus, the open question that ticket records as unasked -
whether the background belongs on the box or on the ink. The changelog row for that change says the
corpus re-measure was "running"; this is it, and it is a regression against a recorded bound rather
than a neutral trade.

**PSM 11 returns lines out of reading order.** The probe above shows `ОБ ЗЛОМ` (y0 1694) arriving
before `ПРОСТО` (y0 1509) - "sparse text, in no particular order" is meant literally, and
`clusterLines` walks its input assuming reading order. On this scene both lines fall under the rescue
floor so nothing depends on it, but a sparse-rung scene whose out-of-order lines survive the floor
would cluster by arrival rather than by position. Recorded here; it belongs with P47's composition
work rather than with this ticket.

## Reproducing

The sweep was a throwaway PowerShell harness (staging by bicubic scale, luminance by the standard
coefficients, TSV parsed for level-5 rows) run outside the repo. It is not kept: like the
`temp/ocrdiag` and `temp/ocrscreen` harnesses before it, re-deriving it from this table is cheaper
than maintaining it. What has to survive is the scene itself, in the corpus, with annotations.
