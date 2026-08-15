# The extension edition, measured against the desktop run it was ported from

**Date:** 2026-08-15
**Runs:** `temp/ocrlab/ext-0815` (extension, this run) against `temp/ocrlab/p47b` (desktop, 2026-08-13)
**Question:** the plate-composition cycle
([`2026-08-13_ocr-sweep-plate-composition`](../../plan/2026-08-13_ocr-sweep-plate-composition.md))
landed in both editions but was measured only on the desktop one, which that note recorded as an
open item: *"The extension edition was not run (`npm run ocrlab`). .. its own evidence is a separate
run."* This is that run.

## Method

`npm run ocrlab -- --split all` from `extension/`, tesseract.js 7.0.0, `eng`, the three pinned
viewports, the six stress cases. The corpus is 46 scenes and every one of them is `dev` - the
holdout is still empty - so `--split all` is the same scene set the desktop run scored with
`-split dev`. Both runs were then scored by the same Go metrics package
(`go run ./tools/ocrlab score <run>`), so nothing in the comparison depends on two implementations
of a measure.

46 of 46 scenes completed, none errored, no browser relaunch.

## The eight dimensions, 13 annotated scenes

| Measure | Desktop `p47b` | Extension `ext-0815` |
|---|---:|---:|
| mean recall | 0.6154 | 0.6154 |
| mean CER | 0.1459 | 0.1620 |
| mean IoU / worst IoU | 0.7756 / 0.3489 | 0.7780 / 0.3489 |
| mean covered | 0.6765 | 0.6154 |
| worst residual ink | 0.2705 | 0.9992 |
| worst halo | 0.6532 | 0.9996 |
| min contrast | 65 | 60 |
| merges / splits | 1 / 0 | 1 / 0 |
| protected-area damage (px) | **0** | **148** |
| clipped / cross-group | 0 / 6 | 0 / **13** |
| worst drift | 0 | 0 |
| OCR wall-clock | 34 503 ms | 18 192 ms |
| scenes with a hard failure | 4 of 13 | 7 of 13 |

Read the residual and halo rows only after the next section. They do not mean what they look like.

## 1. The residual gap is an artefact of the two runners, not a difference in concealment

Per scene, on every scene where both editions produced plates, the extension conceals **as well or
better** than the desktop app:

| Scene | Desktop | Extension |
|---|---:|---:|
| `samson-and-delilah-03-scroll` | 0.2705 | **0.2042** |
| `synth-adjacent-balloons` | 0.1781 | **0.1353** |
| `synth-balloon-on-panel` | 0.0873 | **0.0709** |
| `synth-rtl-layout` | 0.1024 | **0.0695** |
| `synth-two-columns` | 0.0939 | **0.0867** |
| `synth-uniform-paper` | 0.0797 | **0.0784** |
| `synth-display-lettering` | 0.2162 | 0.2141 |
| `synth-text-on-halftone` | 0.2375 | 0.2330 |
| `samson-and-delilah-15-court-caption` | 0.1144 | 0.1079 |

The extension's 0.9992 comes from scenes where it produced **no plates at all** - and on three of
those four, the desktop app produced none either, yet scored `0`:

| Scene | Desktop detection tp/fp/fn | Extension | Desktop residual | Extension residual |
|---|---|---|---:|---:|
| `poster-display-type-on-flat-colour` | 0/0/2 | 0/0/2 | 0 | 0.9992 |
| `le-petit-journal-1908-map-labels` | 0/0/4 | 0/0/4 | 0 | 0.7575 |
| `a-propaganda-poster-..-1920s` | 0/0/1 | 0/0/1 | 0 | 0.7397 |

Same behaviour, opposite scores. The cause is in the runners' output, not in the product: on a scene
that yields no plates the desktop runner writes its stress renders as `raw-<case>.png`, while the
extension runner writes `<case>.png` as always. The scorer finds a render for the extension and none
for the desktop, so it measures the extension honestly - no plates means the original lettering is
100% visible - and defaults the desktop side to `0`, which reads as perfect concealment.

**Consequence for a gate that has already been used.** The desktop `worstResidual` of 0.2705 that
closed the plate-composition ticket against the recorded 0.28 bound is a real number on a real scene
(`samson-and-delilah-03-scroll`), so that result stands. But the aggregate it sits in excludes every
scene where recognition found nothing, because those scenes silently score `0` instead of ~1.0. The
concealment gate is therefore blind to the failure mode "recognized nothing, left the page as-is" -
which is exactly what three of the four desktop failures are. This is a defect in the lab, and it
flatters the desktop edition. It needs its own ticket.

## 2. Three real differences, all on the extension's side of the ledger

Desktop fails 4 of 13, the extension 7. The four desktop failures are a subset of the extension's
seven, so the difference is exactly three scenes:

- **`synth-adjacent-balloons` - protected-area damage, 148 px, worst in `right-outline`.** The
  extension produces one plate the desktop does not (detection `2/1/0` against `2/0/0`), and that
  extra plate lands on a balloon outline the annotation marks as protected. By strategic §3.2 of the
  lab spec, visible border damage is a **blocker**, not a score. It also accounts for 6 of the
  extension's 7 extra cross-group counts - one per stress case.
- **`synth-caption-on-gradient` - not recognized at all** (`0/0/1` against the desktop's `1/0/0`).
  A caption over a gradient is read by native tesseract and missed by tesseract.js. This is an engine
  difference, which the lab spec anticipates in §6, but it is still a scene the reader loses in one
  edition and keeps in the other.
- **`samson-and-delilah-03-scroll` - one plate crosses a reading group** under the `long-cyrillic`
  stress case. Worth noting that on this same scene the extension **recognizes better**: `1/1/0`
  against the desktop's `0/4/1`.

## 3. What is genuinely equal

Recall (0.6154 both), worst IoU (0.3489 both), merges and splits (1/0 both), clipping (0 both) and
drift (0 both, across all three viewports). Mean IoU is a hair higher on the extension. The extension
is roughly twice as fast on OCR wall-clock over the same corpus.

## Answer to the question that prompted this

The two editions are **not** identical, and the parity tests were never evidence that they are: they
pin constants and structural facts against the Go source, which is a guarantee about rules, not about
outcomes. Measured against one corpus and one metrics package, the ported rules behave equivalently
on grouping, placement, drift and clipping; concealment is equal or slightly better in the extension;
and the extension carries two defects the desktop does not - a protected-area blocker on
`synth-adjacent-balloons` and a missed caption on a gradient.

## Follow-ups this run creates

1. **Lab defect (highest value):** the scorer defaults concealment to `0` when a run wrote no
   post-overlay render, so "recognized nothing" scores as "concealed perfectly" on the desktop side.
   Fix the runner naming or make a missing render an explicit non-score, and re-derive nothing until
   it is fixed.
2. **Extension defect:** the extra plate on `synth-adjacent-balloons` damages a protected outline -
   a blocker by the lab's own contract.
3. **Engine gap:** `synth-caption-on-gradient` is read by one engine and not the other; decide
   whether it becomes a recorded divergence or a rescue-ladder case.
