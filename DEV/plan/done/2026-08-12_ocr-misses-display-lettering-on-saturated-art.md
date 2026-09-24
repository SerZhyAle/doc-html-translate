# A poster reads as one dead word, and that one word suppresses the cure

**Status:** Implemented (2026-08-13)
**Priority:** 46
**Date:** 2026-08-12
**Tactical plan:** [`2026-08-12_ocr-misses-display-lettering-on-saturated-art/INDEX.md`](2026-08-12_ocr-misses-display-lettering-on-saturated-art/INDEX.md)

> Cross-edition feature ticket. One feature = one ticket covering every edition.
> Read [`docs/PARITY.md`](../../../docs/PARITY.md) before starting; update it when a shared invariant moves.

## What / why

A standalone image of a poster - display lettering printed over saturated flat colour, the single
most common thing a user drops on this app that is not a book page - produces **one** plate holding
one word. The reader sees a picture with a single stray box on it and no way to tell whether the app
failed or the picture has no text.

Reported from use, on a 960x1280 Soviet-style poster carrying eleven words of Cyrillic display type
in red and black on cream. Reproduction sweep and full transcripts:
[`DEV/research/ocr_display_lettering_2026-08-12.md`](../../research/ocr_display_lettering_2026-08-12.md).
What the shipped path produces, and what the same image gives under
configurations the code already contains, measured with `rus` data, the app's own staging (2x
upscale, `user_defined_dpi=232`) and the app's own two gates - `ocrMinLineConf` = 50 for an ordinary
pass, `ocrRescueLineConf` = 80 for a rescue:

| Configuration | words at 50 | words at 80 | transcript |
|---|---:|---:|---|
| colour + PSM 3 - **what ships today** | 0 | 0 | one line, `\| МОЖЕМ` |
| grey + PSM 3 - **the ladder's first rung, already written** | 10 | 5 | `ЗАЧЕМ ВЗРОСЛЫЕ ‹ ЛЮДИ, \| МОЖЕМ ПРОСТО ОБ ЭТОМ` |
| grey + PSM 11 (sparse text) | 10 | 6 | `ЗАЧЕМ ТРАХАТЬСЯ: МЫ ЖЕ ВЗРОСЛЫЕ ЛЮДИ, МОЖЕМ ПРОСТО ОБ ПОГОВОРИТЬ` - 10 of 11 words, no debris |
| colour + PSM 11 | 16 | 6 | debris interleaved with three real words |

> **Partial, 2026-08-12 - what shipped and what did not.** The ladder now runs every rung and keeps
> the strongest, and a sparse-text rung was added; the class is in the corpus with hand-measured
> annotations; both editions carry it and a parity test pins it; nothing in the corpus regressed. The
> sibling scene went from no plates at all to recall 1.00 with an exact transcript. **The reported
> scene still fails its own done-criterion**: the reader gets seven words instead of two, but the six
> recovered lines land in one plate that matches neither annotated group at the IoU floor, so the lab
> scores the scene as a miss. What is left is grouping, not recognition - with three of nine lines
> still missing, the page's median line pitch is wide enough to join the headline to the body. That
> needs its own measurement and its own ticket rather than a blind adjustment here.

> **Implemented, 2026-08-13 - the grouping half was finished rather than deferred.** The scene was
> re-measured line by line instead of being handed to P47, and the measurement changed the answer:
> **no pitch bound separates this poster in the right place.** The step a reader would cut at, the
> headline to the body, is 335 px; a step *inside* the body, over a line the confidence floor
> dropped, is 381 px. Any pitch threshold that separates the headline tears the body into three. What
> separates them is what a reader uses - **type size**: 281 px of display ink over a 155 px body
> median. So a second condition joins pitch and shared column (`sameTypeSize` / `ocrTypeSizeRatio`
> 1.6), bracketed by two corpus measurements on hand-drawn line boxes - the widest spread inside one
> text is 1.42x (`samson-and-delilah-03-scroll`) and the narrowest step between two texts is 1.86x
> (this poster). `ocrClusterPitchFactor` was **not** moved.
>
> Measured: the reported scene goes from recall 0.00 to **0.50** with its plate crossing an unrelated
> reading group in every stress case to **none**, and twelve of the thirteen annotated scenes are
> identical to the digit. What keeps it at 0.50 is not grouping - the headline's first word scores
> 69.2 against a rescue floor of 80, so one of its two lines is never recognized and its plate cannot
> reach the lab's 0.5 IoU floor. That lever is **P49**, opened here because the tactical plan
> pre-authorised exactly this follow-up. The rule separates sizes, not regions: `accounts.jpg` goes
> from one plate over 80.6% of the image to 68.3%, and same-size separated regions remain **P47**
> ([`2026-08-13_ocr-sweep-plate-composition`](../2026-08-13_ocr-sweep-plate-composition.md)).
>
> **Found while measuring, and not this ticket's:** concealment collapsed corpus-wide between the two
> runs - residual ink mean 17% -> 93% over 46 scenes - and it is the 2026-08-13 plate-shape change,
> proven by a scene whose plate is byte-identical across the two runs while its residual moves from
> 0.085 to 0.998. It is P47's second defect, now measured against the gate. Details:
> [`DEV/research/ocr_display_lettering_2026-08-12.md`](../../research/ocr_display_lettering_2026-08-12.md).

> **Corrected during implementation, 2026-08-12.** Cause 1 below is wrong as written, and the
> correction is the useful part of this ticket. Measured per *line* rather than per word - which is
> what the app gates on - the ordinary pass returns **no plates at all** on this poster in both
> language configurations, so the outer trigger fires and the whole ladder runs; the eng conversion
> spends eleven seconds climbing it and still reports "no text found". What suppressed the cure was
> inside the ladder: `greyRescue` returned the **first rung that produced any plate**, and rung 1
> produces exactly one word. The fix that shipped is a comparator - every rung runs, the strongest
> wins - and no weakness floor was added, because no scene in the corpus asked for one. Cause 2
> stands and is what recovers the text. Full table:
> [`DEV/research/ocr_display_lettering_2026-08-12.md`](../../research/ocr_display_lettering_2026-08-12.md).

Two independent causes, and the first is the one that stings:

1. **The cure is written and unreachable.** `greyRescue` exists precisely for saturated artwork -
   per-channel Otsu splits red-on-cream in three different places and the mask that reaches
   recognition holds no letters. It fires on `len(res.Blocks) == 0`. This image produced one weak
   plate, so it is formally "read", so the whole ladder - grey, its thresholding variants, and the
   halftone rung behind them - is skipped. **A single junk plate is enough to suppress every rescue
   the app has.** That is the same structural fault
   [`2026-08-12_ocr-screen-pass-for-pages-that-already-read`](2026-08-12_ocr-screen-pass-for-pages-that-already-read.md)
   found on the screen axis; this ticket is the grey axis of it, and the two should share a shape.
2. **Page segmentation assumes a page.** `ocrPageSegMode` is fixed at 3, which analyses a document
   layout. A poster has no columns, no line flow and no body text - it has a few large words placed
   for effect. PSM 11 (sparse text) is what recovers `ТРАХАТЬСЯ`, `МЫ ЖЕ` and `ПОГОВОРИТЬ`, the three
   lines PSM 3 loses even on the grey rendition.

**A claim in the roadmap has to be handled honestly here, not quietly reversed.** Wave 4 records
"PSM 3 is **not** the lever", measured on comic balloons and clean renders. That measurement stands
and this one does not contradict it: it was about pages, this is about a single image that is not a
page. The tactical work has to keep both true - any PSM change must be scoped to input that is not a
document page, and proven on the corpus, not on this poster.

## The shape the fix has to have

- **Reachability is a measured threshold, not a rewrite of the trigger.** "Almost nothing was found"
  needs a definition that a corpus can defend - confident words, or plated area against image area,
  or lines against the image's own line estimate. Whatever it is, the ladder must remain unable to
  damage an image that genuinely read: a rescue result may replace a weak one only when it is
  strictly better by a stated comparator, and the comparator has to be named before it is coded.
- **The rescue floor is inherited, and that is now a question.** 80 was derived for "second guess
  after finding nothing". Replacing a weak-but-real result is a different prior.
- **PSM belongs to the staging decision, not to a constant.** The app already knows whether the input
  is one image the user opened or a page extracted from a book. That distinction is the honest place
  for a segmentation choice.
- **The corpus is missing this class.** It holds posters (`join-the-ranks-of-the-red-army`,
  `polish-soviet-propaganda-poster-18y`) but nothing annotated for *display lettering on saturated
  flat colour as a standalone input*. The reported poster should enter the corpus with reviewed
  annotations before any threshold is chosen, exactly as the halftone cycle harvested screened scenes
  before tuning - that ordering changed the answer twice there and is the reason it is repeated here.
- **The measured risk is loss, not gain.** The same sweeps that justify a transform record it
  destroying results elsewhere (16 confident words to 0 on one cover, 71 to 49 on a poster). Any
  change here is scored over the whole corpus with the gate, or it is not landed.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` | `greyRescue` runs every rung and keeps the strongest (`strictlyBetter`); the sparse rung is `greyRescuePasses[2]`; `clusterLines` also breaks on type size (`sameTypeSize`). No flag, no CLI surface change |
| GUI (`doc-html-ui`) | `[x]` | inherits the pipeline output; no flag appeared, so nothing to expose |
| MSIX Store app | `[x]` | inherits the GUI; no packaging impact, confirmed - nothing outside `internal/ocr` moved |
| Browser extension | `[x]` | hand-ported: `GREY_RESCUE_PASSES` / `strictlyBetter` / `OCR_SPARSE_PSM` in `ocr-overlay.js`, `sameTypeSize` / `OCR_TYPE_SIZE_RATIO` in `ocr-cluster.js`, pinned by `TestParityOCRRungComparator` and `TestParityOCRClustering` |
| Website / docs | `[x]` | decided: no change. Recognition quality is not claimed publicly, and the "text in pictures" wording describes what the feature is, not how well it reads |

## Shared invariants touched

- The rescue ladder's trigger condition and rung order - recorded in `docs/PARITY.md` and guarded by
  `TestParityOCRScreenRung`. A weakness threshold and its comparator become shared invariants of the
  same kind.
- `ocrPageSegMode` - a single shared constant today. If it becomes a decision it must be recorded as
  the decision, not as a second constant, so the two editions cannot drift on *when* each mode runs.
- `ocrRescueLineConf` if the floor is re-derived.
- `ocrTypeSizeRatio` - added 2026-08-13, a shared invariant of the same kind as the pitch factor: it
  decides whether two lines are one text. Recorded in `docs/PARITY.md` with both measurements that
  bracket it, and pinned by `TestParityOCRClustering` in value **and** in meaning (which median it is
  weighed against, and that the test is symmetric).

## Done criteria

- [x] The reported poster is in the corpus with reviewed annotations, tied to the class it represents
      rather than to itself. - `poster-display-type-on-flat-colour`, joined by the sibling
      `a-propaganda-poster-from-the-soviet-union-in-the-1920s` so the class is not one image.
- [x] That scene produces plates covering its display lettering, and its transcript is scored against
      the annotation - not against the recognizer's own output. - two plates: `ТРАХАТЬСЯ:` over the
      headline it recognized, and `МЫ ЖЕ ЛЮДИ, МОЖЕМ ПРОСТО ПОГОВОРИТЬ` matching the annotated `body`
      group at **IoU 0.93**, its transcript compared against the annotation. Recall 0.00 -> **0.50**,
      cross-group overlaps 6 -> 0. The remaining false negative is the headline group, whose first
      word (`ЗАЧЕМ`, 69.2) falls under the rescue floor of 80 - a plate over one of its two lines
      reaches IoU 0.455 against the group's bounds. That is the confidence floor, not grouping, and it
      is P49.
- [x] No scene that reads today loses a plate: recall, IoU, concealment, damage, merges/splits and
      cross-group overlap do not regress on the dev split, and the gate is run against a named
      baseline run rather than an impression. - `temp/ocrlab/p46` against `temp/ocrlab/base06-desktop`:
      all eleven scenes the baseline scored keep their recall to the digit, both hard rules identical.
- [x] The weakness threshold, its comparator and any change to the rescue floor are each stated with
      the measurement that chose them, in a research note under `DEV/research/`. - no threshold and no
      floor change was made, and *that* is the recorded answer: nothing in the corpus asked for one.
      The comparator (`resultStrength` in words, ties to the incumbent) is derived in the note.
- [x] The added cost is stated: what an image that reads normally now pays, and what an image that
      triggers the extra pass pays. - 0 and +2.3 s (3.8 s -> 6.1 s), serial end-to-end with `rus` data.
- [x] Both editions, with the shared invariants in `docs/PARITY.md` and a parity test pinning them. -
      `TestParityOCRRungComparator`, plus `TestParityOCRPlateColourOrientation` for the colour defect
      found from the same scene.
- [x] `./scripts/test.ps1` green, `./scripts/lint.ps1` green, `npm test` green; changelog entry in
      `DEV/CHANGELOG.md`. - re-run 2026-08-13, all green, `npm test` 131/131.

## Open questions

Answers recorded 2026-08-13, from the cycle's own measurements. An unanswered one is marked as such
rather than closed with a plausible sentence.

- **What makes a result "weak" enough to retry?** **Still open, deliberately.** No corpus scene asked
  for a weakness floor: measured per *line*, which is what the app gates on, the reported poster's
  ordinary pass returns no plates at all, so the existing emptiness trigger already fires and the
  ladder already runs. A threshold was therefore not invented to answer a question the evidence did
  not pose. The question returns the day a scene reads weakly *and* the ladder would have helped it.
- **Replace or merge?** **Replacement, and it is now the shipped rule.** The rescue re-reads the same
  region rather than finding a new one, so merging would stack two plates over one word.
  `strictlyBetter` is what makes replacement safe: the candidate must be ahead on words placed, and a
  tie keeps the incumbent, because two looks at the same pixels agreeing is one piece of evidence.
- **Is the segmentation choice really "standalone image vs book page", or is it a property of the
  picture?** **Neither, as it turned out - it is a property of the *attempt*.** Sparse text became a
  rescue rung rather than a staging decision, so it needs no plumbing through the pipeline and is
  unreachable for any image that already reads. That sidesteps the splash-page case the question was
  worried about: a splash that reads as a page keeps PSM 3 and never sees the rung.
- **Does this class need its own language question?** **Still open, and the lab is what blocks it.**
  The runner recognizes the whole corpus with one language, so the two Cyrillic scenes can only score
  zero under the default `eng` - they had to be re-run by hand with `-lang rus` to be measured at all.
  A per-scene language in the corpus is the prerequisite; script detection remains where P42 left it.
