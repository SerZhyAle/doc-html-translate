# OCR grey rescue - measure-fix cycle, 2026-08-11

**Ticket:** [`2026-08-11_ocr-visual-fidelity-lab`](../plan/2026-08-11_ocr-visual-fidelity-lab.md)
**Question:** the loop1 report showed 14 of 40 scenes producing no plates at all. Why, and what is
the smallest correction that recovers them without regressing anything?
**Runs:** `temp/ocrlab/loop1` (baseline) -> `loop2` (grey rescue ladder) -> `loop3` (+ rescue
confidence floor). Desktop edition, tesseract 5.4.0.20240606, lang `eng`, edge 151.0.4129.72,
`-split dev`.

> **Phase note.** The plan puts threshold-setting in Phase 06 and rendering changes in Phase 07,
> both `Not started`. This cycle was run against the loop1 baseline on the owner's instruction and
> lands in that territory ahead of the phase order. The number it sets (`ocrRescueLineConf`) is
> derived from measurements recorded below, not chosen, but Phase 06 should re-derive it when it
> runs over the full corpus rather than inherit it.

---

## 1. What was actually wrong

Not `prepareForOCR`'s upscale gate, which was the starting suspicion. The scenes fail at a stage
before any of the app's own logic: **Tesseract's default thresholder cannot see the lettering.**

`thresholding_method=0` - the engine default - runs Otsu on each RGB channel separately and marks a
pixel as ink only where every channel agrees. On flat paper the three channels agree and the rule is
harmless. On saturated artwork each channel splits the picture somewhere else, the channels disagree
over the lettering, and the mask that reaches recognition contains no text.

Measured directly on the diagnostic scenes, no app code involved:

| Scene | as-is (RGB) | 8-bit grey | grey + tiled Otsu (`method=1`) |
|---|---|---|---|
| `synth-adjacent-balloons` (black on white balloons, brick panel) | *nothing* | full transcript | full transcript |
| `synth-balloon-on-panel` (same) | *nothing* | full transcript | full transcript |
| `synth-caption-on-gradient` (white caption on a sky ramp) | *nothing* | *nothing* | full transcript |
| `synth-text-on-halftone` (black type on a dot screen) | *nothing* | *nothing* | *nothing* |

Two separate mechanisms, not one:

1. **Channel disagreement** - fixed by handing over luminance instead of RGB.
2. **A global cut-off cannot survive a varying background** - on the gradient, Otsu splits the
   gradient itself, and the white caption ends up the same value as the ground it sits on. Fixed by
   a thresholder that decides locally (Leptonica's tiled Otsu).

Scale was ruled out: the failure reproduces at 1x, 2x, 3x and 4x, and greyscale fixes it at every
one of them. The upscale gate was innocent.

The harness behind that table is scratch tooling under `temp/ocrdiag/` - `prep/` renders the
transform variants and drives tesseract over each, `rgbgrey/` is the equal-channel proof. It is
gitignored on purpose: it reads the lab's own scenes and answers one question, so it is cheaper to
re-derive from this table than to maintain. Note that it compiles as part of the module, so
`go test ./...` will list `temp/ocrdiag/*` as packages with no test files while it exists.

## 2. Why a ladder and not a replacement

Greyscale is **not** globally better. Scored by confident-word count (words clearing the app's
confidence gate) over all 40 scenes, neither variant dominates - e.g. `commons-right-to-read-poster-1970`
goes 38 -> 2 confident words when converted to grey, because its lettering is separated by hue
rather than brightness and luminance throws that separation away.

So the change is a **rescue ladder**: the ordinary colour pass runs exactly as before, and only an
image that produced **no plates at all** is retried - grey with the engine thresholder, then grey
with the tiled one. Every scene that works today is untouched by construction, and the extra pass is
spent only where the first one produced nothing to lose.

Verified along the way: an RGB image whose three channels hold equal luminance recognizes
identically to an 8-bit grey PNG (per-channel Otsu over three equal channels is one decision). That
is what lets the extension reach the same place through a canvas `grayscale(1)` filter while the
desktop app writes 8-bit grey.

## 3. The defect loop2 introduced, and the floor that closes it

loop2 gained plates on 5 scenes and lost none. Three were the diagnostic scenes above. **The other
two were garbage**: `polish-soviet-propaganda-poster-18y` and `-19y` are Cyrillic posters read with
English data, where the correct answer is "no text", and the rescue turned that into an opaque plate
of invented words over the artwork. A plate of nonsense covering a drawing is worse for a reader
than no overlay, so this counts as a regression on those two images even though no scored metric
moved.

The two bands separate cleanly by line confidence:

| | mean line confidence |
|---|---|
| genuine rescued lettering (7 lines across 3 scenes) | 93.1 - 97.0 |
| hallucinated rescued lettering (Cyrillic posters) | 50.8 |

`ocrRescueLineConf = 80` sits between them with 13 points of margin below the worst genuine line and
29 above the best hallucination. It is anchored on the scale the file already documents:
`ocrMinLineConf = 50` is the top of the band Tesseract hallucinates in, and 80 is the bottom of the
band real text occupies (~80-97). The rationale is that a rescue is a **second guess** - the
ordinary pass already looked and found nothing, so the prior that there is text here at all is
weaker and the bar should be higher.

## 4. Results

| Metric (dev split, 8 scored scenes) | loop1 | loop2 | loop3 |
|---|---:|---:|---:|
| Failing scenes | 5 | 2 | **2** |
| Mean recall | 0.375 | 0.625 | **0.625** |
| Mean precision | 0.375 | 0.625 | **0.625** |
| Mean covered (concealment) | 0.433 | 0.736 | **0.736** |
| Mean IoU | 0.729 | 0.695 | 0.695 |
| Mean CER | 0.129 | 0.163 | 0.163 |
| Merges / splits / clipped / protected-px | 1 / 0 / 0 / 0 | 1 / 0 / 0 / 0 | 1 / 0 / 0 / 0 |
| Cross-group / worst drift | 6 / 0 | 6 / 0 | 6 / 0 |
| Total OCR ms | 5806 | 6158 | **5028** |
| Scenes with plates, of 40 | 26 | 31 | **29** |

Read honestly:

- **Recall, precision and concealment rose by two thirds**; failing scenes halved; nothing regressed
  on any safety dimension (merges, splits, clipping, protected-area damage, cross-group overlap and
  drift are all unchanged).
- **Mean IoU and CER got slightly worse, and that is not a regression.** Both averages are taken over
  scenes that produced plates. `synth-balloon-on-panel` joined that set - it went from *nothing
  recognized* to correct text - but its plates score IoU 0.18 / CER 0.62, which drags the mean down.
  Nothing that was measured before got worse; a previously unmeasurable scene entered the average
  with a poor score.
- **Time went down, not up**, because the confidence floor lets a hopeless rescue give up on the
  first rung instead of running the second.

## 5. What is still failing, and the next smallest work item

1. **`synth-balloon-on-panel` is recognized but mis-grouped.** The text is perfect - `WELL, THAT IS`
   / `ONE WAY TO` / `SOLVE IT!` - but it becomes three plates instead of one balloon. Cause is
   arithmetic, not recognition: the recognized line boxes are 15-17 px tall (all-caps bold has no
   descenders, so the ink box is much shorter than the drawn 36 px leading), the gaps between them
   are 19-21 px, and `ocrClusterGapFactor` 1.2 x median height 15 = 18 px, just under the gap. The
   clustering gate measures the gap against the *ink* height rather than the line pitch. **This is
   the next hypothesis** and it belongs to grouping, not recognition - deliberately not changed in
   the same step.
2. **`synth-text-on-halftone` still reads nothing.** No thresholder tried recovers it; only a blur
   that removes the dot screen does, and that blur costs accuracy elsewhere (it merged `THAT IS`
   into `THATIS`). Screen removal is its own hypothesis and needs real halftone scenes from the
   corpus before it is worth tuning against one synthetic pitch-6 screen.
3. **`synth-two-columns` still merges the two columns** into one plate and crosses reading groups on
   every stress case. Untouched by this cycle; also grouping.
4. **32 of 40 scenes are still unscored** ("no annotation"). Every claim about the real corpus above
   is a plate/word count, not a quality measurement, and is labelled as such. The corpus needs
   annotations before a real-scene regression can be detected at all.

## 6. Code

| Change | Where |
|---|---|
| Rescue ladder, thresholder constants, grey rendition | [`internal/ocr/tesseract.go`](../../internal/ocr/tesseract.go) `greyRescuePasses` / `greyRescue` / `greyRendition` / `recognizePass` |
| Rescue confidence floor | same file, `ocrRescueLineConf` |
| Extension port | [`extension/src/ocr-overlay.js`](../../extension/src/ocr-overlay.js) `GREY_RESCUE_PASSES` / `greyRescue` / `greyRendition` / `OCR_RESCUE_LINE_CONF` |
| Shared-invariant record | [`docs/PARITY.md`](../../docs/PARITY.md) "Grey rescue ladder" |
| Drift guard | [`tests/parity_test.go`](../../tests/parity_test.go) `TestParityOCRGreyRescue` |
| Unit tests | [`internal/ocr/rescue_test.go`](../../internal/ocr/rescue_test.go) |

The extension port is code-complete and guarded, but **not measured** - the extension runner is
Phase 05 and has not been built, so no browser-side evidence exists for it yet.
