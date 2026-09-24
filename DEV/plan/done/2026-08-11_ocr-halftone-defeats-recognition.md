# Lettering printed over a halftone screen recognizes as nothing

**Status:** Implemented
**Priority:** 44
**Date:** 2026-08-11

> Cross-edition feature ticket. One feature = one ticket covering every edition.
> Read [`docs/PARITY.md`](../../../docs/PARITY.md) before starting; update it when a shared invariant moves.

## What / why

`synth-text-on-halftone` - bold black lettering over a dot screen - produces zero plates. Measured
again 2026-08-11 with the fresh build, after the grey rescue ladder landed: `0 image(s) overlaid, 1
with no text found`. It is the one diagnostic scene the ladder does not recover; both of its rungs -
the engine's own thresholder and Leptonica's tiled one, each on a greyscale copy - find nothing.

Why this matters beyond one synthetic scene: a halftone screen is not an edge case in this app's core
material. It is how every printed comic, newspaper page and mid-century poster puts down a tone, the
lab corpus is full of them, and comics and scans are the app's headline use. A page the recognizer
cannot see is a page the reader gets untranslated, with no explanation.

What is already known, from the cycle that produced the rescue ladder
([`DEV/research/ocr_grey_rescue_2026-08-11.md`](../../research/ocr_grey_rescue_2026-08-11.md) section 5):
the only transform that recovered this scene at all was a blur wide enough to dissolve the screen,
and that same blur cost accuracy elsewhere - it merged two words into one on a clean balloon. So
"always blur" is not the answer. Something screen-aware is, and it belongs in the same position the
rescue ladder occupies: **only for an image the ordinary pass could not read at all**, where there is
nothing to lose and therefore nothing to regress.

**Do not tune this against one synthetic screen.** The scene has a single pitch (6 px) at a single
tone and angle; real screens vary in all three, and a fix tuned to pitch 6 is a fix for pitch 6. The
corpus already holds real screened material - annotate a few of those first, then measure, then
choose. That ordering is the ticket's main constraint, not a nicety.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x]` | `internal/ocr/screen.go` plus the `screenRescue` rung at the end of `greyRescue` |
| GUI (`doc-html-ui`) | `[x]` | inherits the pipeline output |
| MSIX Store app | `[x]` | inherits the GUI; the pass writes only the temp PNG the ladder already wrote, so no new writable-directory requirement |
| Browser extension | `[x]` | **no decline needed.** The measurement is the detector, not a library: `ocr-screen.js` is arithmetic over a typed array, and the kernel is CSS `blur(<sigma>px)` applied in the same canvas draw as `grayscale(1)` - one extra `getImageData` and one extra `recognize`, and only on an image that produced nothing |
| Website / docs | Declined (expected) | recognition quality on screened art is not claimed publicly |

## Shared invariants touched

- If a pre-pass lands, its **trigger** and its **parameters** become shared invariants. The grey
  rescue ladder is the precedent: rung order plus a confidence floor, both recorded in
  `docs/PARITY.md` and both guarded in `tests/parity_test.go`.
- The rescue ladder's own position is touched by construction: a screen pass is either a new rung or
  a step before the ladder, and the order of the rungs is itself the guarded invariant.

## Cross-references

- Go: [`internal/ocr/tesseract.go`](../../../internal/ocr/tesseract.go) - `prepareForOCR` and
  `greyRescue`, the position a screen pass would occupy.
- JS: `extension/src/ocr-overlay.js` - `greyRendition` / `greyRescue`, the twin.
- The scene: `tools/ocrlab/synth/synth.go` - `sceneTextOnHalftone`, pitch 6.
- Real material to annotate first: `test_doc/ocrlab/commons/` (mid-century comic covers and
  political cartoons, several visibly screened).

## Done criteria

- [x] At least three **real** screened scenes carry reviewed annotations in the corpus before any
      parameter is chosen. A synthetic-only fix does not close this ticket.
      → `samson-and-delilah-03-scroll`, `samson-and-delilah-15-court-caption`,
      `le-petit-journal-1908-map-labels`, annotated before the divisor was chosen. Finding them
      required extending the corpus: see "What landed" below.
- [x] `synth-text-on-halftone` produces its line, ~~and the real screened scenes improve~~.
      → The line comes back exact through the shipped path. **The second half was measured to be
      unreachable from this position** and is struck rather than quietly dropped - see "What landed".
- [x] No scene that produced plates before produces fewer or worse ones - measured on the dev split,
      not judged by eye.
- [x] The added cost is stated: how much slower a page gets, and on which images the pass runs.
- [x] Both editions, or a decline recorded in `docs/PARITY.md` with its reason. → both.
- [x] `./scripts/test.ps1` green, `npm test` green; changelog entry in `DEV/CHANGELOG.md`.

## Open questions

- ~~Is screen removal detectable cheaply enough to be a *conditional* pass rather than a blind one? A
  regular dot lattice has an obvious frequency-domain signature, but a frequency transform on every
  unreadable image may cost more than it saves.~~ **Answered: yes, and no transform is needed.** A
  lattice repeats, so tile-wise autocorrelation of a 3x3 high-pass finds its period at O(n) with the
  tile count capped at 96 whatever the image size. Over 59 images the diagnostic scene scores 1.00 of
  textured tiles agreeing at peak 0.78 against 0.44 for the strongest ordinary artwork.
- ~~Would a resolution change do it instead? Downsampling averages a screen away and the app already
  has an upscale stage - the interaction between the two is unexplored and cheap to measure.~~
  **Answered: no.** Measured at 2x, 3x and 4x: still nothing. Block-averaging only removes a screen
  once the block reaches the screen's own period, and by then the lettering has gone too. The
  interaction with the upscale stage did matter, though, in the other direction: the scene is
  upscaled 2x before recognition, so the screen the rung meets has pitch 12, not the file's 6.
- ~~Does the extension have any realistic route at all, or is the honest answer a recorded
  divergence? Decide it from a measurement of canvas cost on a real page, not from a guess.~~
  **Answered: it has a route, and a cheap one.** No divergence recorded.
- **Still open, and now the point:** the position. A page that mixes clean balloons with screened
  captions never reaches the rescue, so the caption stays unread. Taken up by
  [`2026-08-12_ocr-screen-pass-for-pages-that-already-read`](2026-08-12_ocr-screen-pass-for-pages-that-already-read.md).

## What landed

**The pass, where the ticket said to put it.** A third rung on the rescue ladder, after both grey
rungs: measure the screen's period, and if there is one, recognize a copy of the greyscale image
low-passed with a Gaussian of `sigma = pitch / 4`. `synth-text-on-halftone` goes from nothing to its
exact transcript, `MEANWHILE, ELSEWHERE`, in a box within 5 px of the annotation on every edge.
Nothing that produced a plate can be touched, because the rung is unreachable for such an image.

**The corpus did not hold what the ticket assumed, and that was measured rather than argued.** The
premise was that the lab corpus is full of screened material with text on it. Scoring all 30 commons
scenes for a lattice and then looking at the top of that list at 1:1 zoom showed three different
reasons why almost none qualifies: the screen is in the artwork while the lettering sits on clean
balloons; or the scan is too coarse to resolve a screen at all; or the texture is wood engraving and
crosshatching rather than halftone. So the corpus was **extended** - 19 candidates harvested and
scored, and the three that qualify are now annotated, with their parent pages added and tied to them
through `derivedFrom` (`ocrlab add` grew a `-derived-from` flag rather than the manifest being
hand-edited).

**The finding that matters more than the fix.** With real screened scenes in hand, a corpus-wide
sweep ran every image twice - ordinary grey against the pitch-driven low-pass. **Of 59 images,
exactly one goes from nothing to something: the synthetic scene.** On real screened material the
ordinary passes already read the text (46 confident words on the scroll crop, 76-96 on a whole
screened comic page), so the rescue position is never reached. The one real scene that does reach it
reads nothing either way, because its obstacle is outline letterforms, not the screen.

Where screen removal genuinely pays on real material, it pays on images that **already produce
plates**: `le-petit-journal` 55 -> 81 confident words, `samson-and-delilah-03` 76 -> 90, and the
caption band finishes its sentence. Collecting that needs an additive second pass with a merge rule,
a re-derived confidence floor and whole-page annotations to score plate composition - a different
change with a different regression surface, so it is its own ticket rather than a quiet widening of
this one.

**What this ticket therefore is:** the safe half, shipped and guarded, plus the measurement that says
where the rest of the value is. Full record:
[`DEV/research/ocr_halftone_2026-08-12.md`](../../research/ocr_halftone_2026-08-12.md).

**Cost**, measured as an A/B with the rung off and on: **nothing** on an image that reads (605 ->
610 ms on a screened comic page - the rung is unreachable), 2-27 ms on an unreadable image with no
screen, and one extra recognition where a screen is found (+141 ms on the diagnostic scene, +443 ms
on the 1750x1350 map crop). A 12-megapixel page pays the same measurement as a panel, because the
tile count is capped at 96.

**Dev split** (`temp/ocrlab/pitch1` -> `loop4`, the correct baseline being the grouping ticket's run
and not `loop3`): 5 scenes gained plates, **0 lost**, 40 unchanged - and four of the five gains are
the new corpus scenes, so the only existing scene that moved is `synth-text-on-halftone`, 0 -> 1, at
recall 1.00 and IoU 0.76. Merges, splits, protected-area damage, cross-group overlap and drift are
all unchanged. The aggregate mean recall dips 0.750 -> 0.727 and mean IoU 0.820 -> 0.776 purely
because three new scenes joined the average, one of which reads nothing; `worstIou` is unchanged and
not one of the eight previously scored scenes moved on any dimension.
