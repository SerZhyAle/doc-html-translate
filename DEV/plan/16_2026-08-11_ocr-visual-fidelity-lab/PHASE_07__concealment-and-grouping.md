# Phase 07 - Concealment and grouping

**Strategic spec:** [`../16_2026-08-11_ocr-visual-fidelity-lab.md`](../16_2026-08-11_ocr-visual-fidelity-lab.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** 🚧 In Progress - 07.3 done 2026-09-25 with the grouping halves of 07.4-07.7 (see "Step 07.3
landed" below); 07.1 / 07.2 and the concealment halves of 07.4-07.7 stay ⛔ Blocked on strategic §9.1 / §9.2
**Depends on:** Phase 06
**Steps done:** 1 / 7 (07.3), four more in part

## Objective

The first user-visible change: a concealment-mode decision that stops covering artwork, grouping that
respects balloon boundaries, and both editions moving together with a guarded shared contract.

## Prerequisites

- [ ] Phase 06 is ✅ Done and `thresholds.json` exists.
- [ ] Every step below cites the baseline table that motivates it. A step with no measured cause does
      not run - strategic §5.4 forbids tuning several things in one unmeasured move.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `internal/ocr/conceal.go` | New | ≤ 320 |
| `internal/ocr/conceal_test.go` | New | ≤ 300 |
| `internal/ocr/overlay.go` | Modified | ≤ 780 |
| `internal/ocr/tesseract.go` | Modified | ≤ 560 |
| `extension/src/ocr-overlay.js` | Modified | - |
| `extension/src/ocr-overlay.css` | Modified | - |
| `docs/PARITY.md` | Modified | - |
| `tests/parity_test.go` | Modified | - |
| `internal/ocr/boundary.go` | New (07.3) | - |
| `internal/ocr/boundary_test.go` | New (07.3, 07.7) | - |
| `internal/ocr/cluster_test.go`, `internal/ocr/tsv_test.go` | Modified (07.3, signatures) | - |
| `extension/src/ocr-cluster.js` | Modified (07.4) | - |
| `extension/test/ocr-cluster.test.mjs` | Modified (07.4, 07.7) | - |
| `tools/ocrlab/synth/synth.go`, `DEV/ocrlab/corpus.json`, `DEV/ocrlab/annotations/synth-side-by-side-balloons.json` | Modified / New (07.7) | - |
| `docs/contracts/OCR-PIPELINE.md` | Modified (catalog 1.1) | - |

## Steps

### Step 07.1 - Add the concealment-mode decision (Go)

**Files:** `internal/ocr/conceal.go`
**Depends on:** - start of phase

**Prompt for developer:**
> Implement `func chooseMode(img image.Image, b Block) (Mode, float64)` returning one of `ModeFill`,
> `ModeReconstruct`, `ModeMask` and a confidence in `[0,1]`, from local evidence only: sample the
> block's border ring, measure background variance and gradient, and detect a strong edge crossing the
> block (a bubble outline or panel rule). Uniform ring and low variance gives `ModeFill`; a smooth
> gradient gives `ModeReconstruct`; high variance, a crossing edge or low confidence gives `ModeMask`.
> The estimator must look at a local ring, never at the median of the whole paragraph - strategic §7.

**Verification:**
- `func chooseMode(img image.Image, b Block) (Mode, float64)` matches exactly once.
- `ModeFill`, `ModeReconstruct` and `ModeMask` are declared exactly once each.
- Unit tests on the Phase 02 synthetic scenes assert uniform paper gives `ModeFill` and the
  bordered-balloon-over-panel scene gives `ModeMask`.

**Status:** `[ ]` not done

---

### Step 07.2 - Emit the mode in the overlay (Go)

**Files:** `internal/ocr/overlay.go`, `internal/ocr/conceal.go`
**Depends on:** Step 07.1

**Prompt for developer:**
> Have `wrapImage` call `chooseMode` and write the outcome onto the plate as `data-ocr-mode` and
> `data-ocr-mode-conf`, and render accordingly: `fill` keeps today's opaque block; `reconstruct` paints
> the sampled local background with a soft edge; `mask` restricts the painted area to the union of the
> cluster's *line* boxes inset to the ink extent rather than the block rectangle. The base image is
> never modified - every mode is CSS over the untouched `<img>`.

**Verification:**
- `data-ocr-mode` appears in the rendered plate attributes.
- `internal/ocr/overlay_test.go` asserts the source image bytes are unchanged after an overlay.
- A test asserts a `mask`-mode plate's painted area is strictly smaller than its block rectangle.

**Status:** `[ ]` not done

---

### Step 07.3 - Stop plates crossing reading groups (Go)

**Files:** `internal/ocr/tesseract.go`
**Depends on:** Step 07.2

**Prompt for developer:**
> Address the merge count the baseline reports for adjacent balloons: extend `clusterLines` so a
> candidate line joins the open cluster only when it also passes a boundary test - no strong horizontal
> edge between the two lines' bands and a background-similarity check on the strip between them. Keep
> `ocrMinLineConf` and `ocrClusterPitchFactor` (renamed from `ocrClusterGapFactor` on 2026-08-11)
> unchanged; this is an added condition, not a retuned one, and it must be a named constant.

**Verification:**
- The new constant is declared once with a comment naming the baseline table it came from.
- `internal/ocr/tsv_test.go` gains a case where two vertically adjacent lines separated by a strong
  edge produce two blocks instead of one.
- `go run ./tools/ocrlab gate <new run>` shows the merge count improved and no other dimension regressed.

**Status:** `[x]` done 2026-09-25, on the horizontal axis - see "Step 07.3 landed" for the deviation and
the evidence against each verification item.

---

### Step 07.4 - Port the mode decision to the extension

**Files:** `extension/src/ocr-overlay.js`, `extension/src/ocr-overlay.css`
**Depends on:** Step 07.3

**Prompt for developer:**
> Hand-port `chooseMode`, the mode rendering and the clustering boundary test into `ocr-overlay.js`
> with the same constant names and values, and add the corresponding classes to `ocr-overlay.css`. The
> extension must emit the same `data-ocr-mode` attribute so Phase 05's runner records it without a
> special case.

**Verification:**
- `npm test` green in `extension/`.
- The extension's emitted plates carry `data-ocr-mode`.
- The ported constants match the Go values character for character.

**Status:** `[~]` the clustering boundary test is ported (`ocr-cluster.js` `strokeBetween`, `paperLuma`,
`OCR_BOUNDARY_REACH`, orphans; `ocr-overlay.js` `strokePlane`), `npm test` 271/271; the mode decision waits
on 07.1.

---

### Step 07.5 - Record the new shared contract

**Files:** `docs/PARITY.md`
**Depends on:** Step 07.4

**Prompt for developer:**
> Add the concealment modes, their decision inputs, the mode confidence and the new clustering
> boundary constant to the OCR invariant table, and state which behaviour is a hard visual gate. Note
> the `DOCHT_OCR_DIAG` sidecar as desktop-only and intentional.

**Verification:**
- `docs/PARITY.md` names all three modes and the new constant.
- The OCR table row for plate granularity mentions the boundary test.

**Status:** `[~]` the boundary test is in `docs/PARITY.md` (the "Line integrity" row and the word-gap note:
`OCR_BOUNDARY_REACH`, the reused `PLATE_MIN_CONTRAST`, the paper ring, orphans, the one uncovered case);
the three modes wait on 07.1. The `DOCHT_OCR_DIAG` note is moot - since ticket 15 the extension writes
the same `ocr-diag.jsonl`.

---

### Step 07.6 - Guard the contract mechanically

**Files:** `tests/parity_test.go`
**Depends on:** Step 07.5

**Prompt for developer:**
> Extend the parity suite with `TestParityOCRConcealment`: parse both codebases, assert the three mode
> names, the new boundary constant and the mode-confidence floor are identical. Do not copy the numbers
> into the test - read them from both sources and compare, so the test cannot pass on a stale literal.

**Verification:**
- `TestParityOCRConcealment` matches exactly once.
- Changing the constant in only one edition makes `go test ./tests/...` fail.

**Status:** `[~]` the boundary constant is guarded in the existing `TestParityOCRClustering` rather than a
new test (it is a clustering constant, and that test already reads both sources): the value pair plus
nine expression pins. Proven to fail on a one-sided change (`OCR_BOUNDARY_REACH = 0.15` in the extension
only: `boundary reach drift: tesseract.go=0.14 ocr-cluster.js=0.15`). `TestParityOCRConcealment` waits on
the modes of 07.1.

---

### Step 07.7 - Turn every confirmed failure into a fixture

**Files:** `internal/ocr/conceal_test.go`, `DEV/ocrlab/corpus.json`, `DEV/ocrlab/annotations/`
**Depends on:** Step 07.6

**Prompt for developer:**
> For each defect class this phase fixed, add a minimized regression scene: prefer a synthetic scene
> when the defect can be reproduced by construction; otherwise a minimized crop of a licensed corpus
> scene, recorded as a derivative with `DerivedFrom` set. Add the deterministic unit test that would
> have caught it. Strategic §2.8 requires this before the fix is accepted.

**Verification:**
- Each fixed class has one new test and one new scene entry.
- `go run ./tools/ocrlab verify` still exits 0 on the manifest (or reports only the pre-existing
  coverage gaps).

**Status:** `[~]` for the grouping class: scene `synth-side-by-side-balloons` (exact by construction, in
`corpus.json` and `DEV/ocrlab/annotations/`) and the deterministic tests `internal/ocr/boundary_test.go`
(8) and their mirrors in `extension/test/ocr-cluster.test.mjs` (8). `ocrlab verify` reports only the
pre-existing gaps. The concealment classes wait on 07.1 / 07.2.

## Phase done criteria

- [ ] Every `Step 07.*` is `[x] done`.
- [ ] `./scripts/check.ps1` green (test + lint + typos).
- [ ] `npm test` green in `extension/`.
- [ ] `ocrlab gate` PASSes against the Phase 06 summary for both editions.
- [ ] Grep for `TODO(phase-07)` returns zero hits.
- [ ] Changelog entry added for every file in "Files touched".

## Out-of-order work already landed (2026-08-11)

This is "the first phase allowed to change user-visible rendering", and one such change was already
made on the owner's instruction: the grey rescue ladder
([`../../research/ocr_grey_rescue_2026-08-11.md`](../../research/ocr_grey_rescue_2026-08-11.md))
gives an overlay to three diagnostic scenes that previously got none. It changes **recognition**, not
concealment or grouping, so no step below is done and the phase stays ⬜ Not started.

Two of this phase's own targets are now measured and still open:

- ~~`synth-balloon-on-panel` is recognized perfectly and grouped wrongly - three plates instead of one
  balloon.~~ **Closed 2026-08-11** as its own ticket rather than as a second half of Step 07.3, which
  is the decision this note asked for: 07.3 answers over-merging with an *added* condition, this was
  over-*splitting* and the honest fix changed what the factor is measured against, so bundling them
  would have moved one gate twice in one step. See
  [`ocr-grouping-splits-one-balloon`](../done/2026-08-11_ocr-grouping-splits-one-balloon.md). The
  gate now compares line pitch against the image's median pitch (`ocrClusterPitchFactor`,
  `ocrMaxLeadingRatio`); the balloon is one plate, `synth-adjacent-balloons` is still two, and the
  dev split's `merges`, `splits` and `crossGroup` are unchanged. **Step 07.3 is not affected in its
  substance** - its instruction was to keep the *value* of the grouping constants and add a boundary
  condition, and no boundary condition was added here - but its prompt now names a constant that has
  been renamed, so read `ocrClusterGapFactor` there as `ocrClusterPitchFactor`.
- `synth-two-columns` still merges its two columns into one plate that crosses the other column at
  every stress case.

## Reconciliation (2026-09-25)

The queue asked for this phase to be reconciled with what landed out of band before it is planned.
Done against a fresh measurement rather than the 2026-08-11 tables: `ocrlab run` over the eight
synthetic scenes, desktop edition, in a cloud session (tesseract 5.3.4 with the distribution's `eng`
data, Chromium 141, the corpus media absent - `test_doc/ocrlab/` is gitignored, so only the scenes
`ocrlab synth` redraws are measurable there). The run folder is temp and not committed; the numbers:

| Scene | IoU | Residual | Cut-glyph ink | Damage px | Merges | Cross-group |
|---|---:|---:|---:|---:|---:|---:|
| `synth-uniform-paper` | 0.96 | 0.14 | 0.00 | 0 | 0 | 0 |
| `synth-two-columns` | 0.94 | 0.10 | 0.00 | 0 | 0 | 0 |
| `synth-balloon-on-panel` | 0.91 | 0.12 | 0.00 | 0 | 0 | 0 |
| `synth-adjacent-balloons` | 0.84 | 0.19 | 0.00 | 0 | 0 | 0 |
| `synth-caption-on-gradient` | 0.93 | 0.08 | 0.00 | 0 | 0 | 0 |
| `synth-text-on-halftone` | 0.76 | 0.18 | 0.21 | 0 | 0 | 0 |
| `synth-display-lettering` | 0.83 | 0.13 | 0.00 | 0 | 0 | 0 |
| `synth-rtl-layout` | 0.92 | 0.05 | 0.00 | 0 | 0 | 0 |

Residual and cut-glyph ink are the self-diagnosis at the desktop viewport; the scored report's
"worst residual" column (0.38 on `synth-display-lettering`) is the annotated-group measure. Every hard
gate is at zero and recall is 1.00 on all eight.

What that says about each step, under this phase's own prerequisite ("a step with no measured cause
does not run"):

- **07.3 has no measured cause left in the repo.** Its target, `synth-two-columns` merging its two
  columns, was closed on 2026-09-12 by a different mechanism - the word-gap split and column
  regrouping of [`done/2026-09-12_ocr-line-stitched-across-the-picture`](../done/2026-09-12_ocr-line-stitched-across-the-picture.md) -
  and the run above confirms it: 0 merges and 0 cross-group overlaps across all eight scenes. The
  remaining candidate, the comic-balloon band at 1.87-2.57x line pitch (release-1 worklog §1.7), was
  measured on the gitignored corpus only. Re-measure it there before 07.3 is written; if it holds, the
  step's substance (an *added*, named boundary condition) still stands.
- **07.1 / 07.2 have a cause but not a method.** Residual ink on non-paper backgrounds (baseline
  §8.4) is real, but the step would pick its mask and reconstruction rules by intuition: strategic
  §9.1 (mask fidelity) and §9.2 (background reconstruction) are still Open in baseline §10, blocked on
  annotated `texture` scenes and on protected polygons, both human-owned. The synthetic run adds two
  facts against guessing: residual is 0.14 on flat paper, where the decision would pick today's fill
  anyway, so the mode choice is not what drives it; and the one scene with a finding
  (`synth-text-on-halftone`, 21% cut-glyph ink) is a plate that stops short of its glyphs - a mask
  restricted to the line boxes paints less, not more. With damage at zero everywhere there is also
  nothing the mask mode could be shown to protect.
- **07.4 - 07.7** follow 07.1 - 07.3 and inherit their block.

**Unblocks when:** the §4.3 annotation gate delivers annotated `texture` scenes with protected
polygons (resolving §9.1 / §9.2 for 07.1 - 07.2), and an owner-machine corpus run re-measures the
balloon merge band (for 07.3). No step below is marked done by this reconciliation, and no rendering
decision moved.

## Step 07.3 landed (2026-09-25)

Run on the owner's machine, where the corpus is: the half of the unblock condition above that needed no
annotation. Research: [`DEV/research/ocr_balloon_boundary_2026-09-25.md`](../../research/ocr_balloon_boundary_2026-09-25.md).

**The re-measure.** The balloon band holds and is wider than recorded: over all 1043 word gaps of the
corpus (desktop engine, tesseract 5.4.0, the app's staging), stitches between two balloons sit at
1.00-3.46x and real lines reach 3.07x. So the step's substance stood - an *added*, named boundary
condition from the pixels.

**Deviation from the prompt, and why.** The prompt put the condition in `clusterLines`, between two
*vertically* adjacent lines. The measured cause is horizontal: facing balloons returned as one recognizer
line, a merge `clusterLines` never sees as two lines. So the condition sits in the line split, before
the clustering (`splitWideGaps`): a gap is also cut when a stroke crosses it and runs on past the line on
both sides (`strokeBetween`, `internal/ocr/boundary.go`). `ocrMinLineConf` and `ocrClusterPitchFactor`
are untouched. It also runs ahead of 07.1 / 07.2, which it does not depend on in substance - the
reconciliation gave the two halves separate unblock conditions.

**Against the verification items:**

- *The new constant is declared once with a comment naming the table it came from:* `ocrBoundaryReach =
  0.14` in `tesseract.go`'s shared block (and `OCR_BOUNDARY_REACH` in `ocr-cluster.js`), bracketed
  between the two measured failures - a real line cut at 0.07, a stitch lost at 0.30. The ink threshold
  reuses `plateMinContrast` (every value from 20 to 128 separates the labelled gaps identically).
- *A test where two lines separated by a strong edge produce two blocks instead of one:*
  `TestParseTSVSplitsBalloonsStitchedAtANarrowGap` (horizontal - one block without pixels, one per balloon
  with them) with 7 more in `internal/ocr/boundary_test.go`, including the regression below.
- *`ocrlab gate` shows the merge count improved and no other dimension regressed:* on the new scene
  `synth-side-by-side-balloons`, stroke test off -> on, both editions: merges 1 -> 0, cross-group 6 -> 0,
  protected damage 448 -> 0 px. Over the 14 annotated scenes (`temp/ocrlab/p16-final` against
  `p16-base`) the gate fails the same 6 stale-threshold checks both times and is equal or better on
  every line; every hard gate is 0 in both.

**What the first corpus run caught.** On `atomicwar0401` the test correctly cut a speck read as `A` off
a balloon's line, and the speck then split the balloon into two plates: the page-wide headline makes
one column, and the speck sat in it at the height of the line it was cut from. Every untranslatable run
a stroke cuts off on the corpus is artwork, so such a run is now an *orphan* and `orderColumns` parks it;
`TestParseTSVParksWhatAStrokeCutOff` reproduces the split and fails without the parking.

**Left open, measured:** when both outlines of a balloon pair are read as one token (`ff`, `fj`, `|`),
no gap holds a stroke - two plates on `samson-and-delilah-15` still cross balloons this way. A test
through the token also fires on a real word the recognizer glued an outline to (`TO}`), so it is a
second design, not a follow-on threshold.

**Catalog first:** `OCR-PIPELINE` 1.1 - a dated amendment in the catalog's `ocr-overlay/ocr-pipeline.md`
covering the whole line split (the 2026-09-12 word-gap stage was never written there), both registry rows
and `docs/contracts/OCR-PIPELINE.md` bumped.

## Handoff notes

This phase is the template for every later iteration: one measured cause, one scoped change, both
editions, a parity guard, a regression fixture, and a gate run against the last accepted summary.

## Rollback plan

Revert the phase commit(s) in reverse step order. The mode decision is additive - reverting 07.2
restores today's single opaque-fill behaviour without touching recognition.
