# Phase 03 - Metrics

**Strategic spec:** [`../07_2026-08-11_ocr-visual-fidelity-lab.md`](../07_2026-08-11_ocr-visual-fidelity-lab.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 02
**Steps done:** 9 / 9

## Objective

The evidence contract, then every dimension of the strategic §3.2 table as a pure, unit-tested Go
function over Phase 02 truth and that evidence - measurement only, no thresholds and no verdicts.

## Prerequisites

- [x] Phase 02 is ✅ Done.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `tools/ocrlab/evidence/evidence.go` | New | ≤ 260 |
| `tools/ocrlab/metrics/geometry.go` | New | ≤ 240 |
| `tools/ocrlab/metrics/recognition.go` | New | ≤ 200 |
| `tools/ocrlab/metrics/grouping.go` | New | ≤ 240 |
| `tools/ocrlab/metrics/placement.go` | New | ≤ 200 |
| `tools/ocrlab/metrics/concealment.go` | New | ≤ 300 |
| `tools/ocrlab/metrics/damage.go` | New | ≤ 200 |
| `tools/ocrlab/metrics/replacement.go` | New | ≤ 200 |
| `tools/ocrlab/metrics/score.go` | New | ≤ 260 |
| `tools/ocrlab/metrics/*_test.go` | New | ≤ 800 total |

## Steps

### Step 03.1 - Define the cross-edition evidence schema

**Files:** `tools/ocrlab/evidence/evidence.go`
**Depends on:** - start of phase

**Prompt for developer:**
> Create package `evidence` with `Run` (`SchemaVersion`, `RunID`, `StartedAt`, `Edition`
> (`desktop`/`extension`), `Engine` {`Tesseract`, `TessdataVersion`, `Lang`}, `Browser` {`Name`,
> `Version`}, `Viewports []Viewport`, `Scenes []Scene`) and `Scene` (`SceneID`, `ImageWidth`,
> `ImageHeight`, `Plates []Plate`, `Screenshots` {`Source`, `Rendered`, `Stress map[string]string`},
> `OcrMs`, `RenderMs`, `PeakRSSBytes`, `Error`). A `Plate` carries `Text`, `Rect` (natural image px),
> `Viewport`, `StressCase`, `FontPx`, `Background`, `Ink`, `Mode` (`fill`/`reconstruct`/`mask`),
> `ModeConfidence`, `ScrollHeight`, `ClientHeight`. Add `Save`/`LoadRun`. This is the contract both
> runners emit and every metric consumes; `SchemaVersion` starts at 1 and is the only way to change it.

**Verification:**
- `type Run struct` and `type Plate struct` each match exactly once.
- `Mode` and `ModeConfidence` are present on `Plate` (Phase 07 fills them; Phase 04 writes `fill`).
- `func (r *Run) Save(path string) error` and `func LoadRun(path string) (*Run, error)` exist.

**Status:** `[x]` done

---

### Step 03.2 - Geometry primitives

**Files:** `tools/ocrlab/metrics/geometry.go`
**Depends on:** Step 03.1

**Prompt for developer:**
> Rasterization already exists as `truth.Region.Rasterize` / `truth.Mask` (Phase 02 needed exact
> polygon overlap in its own validator, and `metrics` imports `truth`, so a second rasterizer here
> would be an import cycle). Build on it: IoU, containment fraction (`|A ∩ B| / |A|`), and normalized
> edge error (`dx0,dy0,dx1,dy1` divided by the reference box's width or height). No floating-point
> comparison in the API - return `float64` and let the caller compare.

**Verification:**
- `func IoU(a, b truth.Region, w, h int) float64` matches exactly once.
- `metrics` declares no rasterizer of its own; it calls `truth.Region.Rasterize`.
- A test asserts IoU of two identical boxes is 1 and of two disjoint boxes is 0.

**Status:** `[x]` done

---

### Step 03.3 - Recognition: CER, WER, detection precision/recall

**Files:** `tools/ocrlab/metrics/recognition.go`
**Depends on:** Step 03.2

**Prompt for developer:**
> Implement `CER`/`WER` as Levenshtein distance over a normalized string (collapse whitespace, NFC,
> case-fold, strip the punctuation Tesseract routinely invents at glyph edges), and
> `Detection(plates []evidence.Plate, groups []truth.Group, iouMin float64) DetectionScore` returning
> precision, recall, F1 and the raw TP/FP/FN counts, where a plate matches a group at `IoU >= iouMin`.
> Score recognition only for groups whose annotation `Ambiguity` is `clear`; count the rest separately
> as `Skipped` so uncertain historical lettering cannot masquerade as a regression.

**Verification:**
- `func CER(got, want string) float64` and `func WER(got, want string) float64` each match exactly once.
- `DetectionScore` declares `Precision`, `Recall`, `F1`, `TP`, `FP`, `FN`, `Skipped`.
- A test asserts `CER("abc","abc") == 0` and that an `illegible` group is counted in `Skipped`, not `FN`.

**Status:** `[x]` done

---

### Step 03.4 - Reading groups: merges, splits, order

**Files:** `tools/ocrlab/metrics/grouping.go`
**Depends on:** Step 03.3

**Prompt for developer:**
> Implement `Grouping(plates, groups) GroupingScore`: build the plate-to-group overlap matrix, take the
> maximum-weight one-to-one matching (greedy by descending overlap is sufficient and deterministic -
> document that choice in a comment), then count `Merges` (one plate covering ≥ 2 groups above a
> coverage fraction), `Splits` (one group covered by ≥ 2 plates), `Unmatched` on both sides, and
> `OrderInversions` (pairs whose matched plates disagree with the annotated `ReadingOrder`).

**Verification:**
- `func Grouping(plates []evidence.Plate, groups []truth.Group, w, h int) GroupingScore` matches exactly once.
- `GroupingScore` declares `Merges`, `Splits`, `OrderInversions`.
- A test on the two-adjacent-balloons synthetic scene asserts one plate spanning both counts as one merge.

**Status:** `[x]` done

---

### Step 03.5 - Placement: IoU and edge error at every viewport

**Files:** `tools/ocrlab/metrics/placement.go`
**Depends on:** Step 03.4

**Prompt for developer:**
> Implement `Placement(matched []Match, w, h int) PlacementScore` returning mean/median/worst IoU, mean
> and worst normalized edge error, and `Drift` - the maximum change in a plate's normalized position
> across the viewports present in the evidence. Because plate geometry is recorded in natural image
> coordinates per viewport, drift is measured, not inferred.

**Verification:**
- `func Placement(matched []Match, w, h int) PlacementScore` matches exactly once.
- `PlacementScore` declares `MeanIoU`, `WorstIoU`, `WorstEdgeError`, `Drift`.
- A test asserts identical geometry across two viewports gives `Drift == 0`.

**Status:** `[x]` done

---

### Step 03.6 - Concealment: covered glyph mask, residual ink, rendered contrast

**Files:** `tools/ocrlab/metrics/concealment.go`
**Depends on:** Step 03.5

**Prompt for developer:**
> Implement three measures over the rendered screenshot mapped back to natural image coordinates.
> `Covered(plates, groups) float64` is the fraction of each group's annotated text area hidden by an
> opaque plate. `ResidualInk(source, rendered image.Image, g truth.Group) ResidualScore` estimates the
> source lettering still visible: build a local ink mask inside the group's `Bounds` from the source
> image (pixels whose luma differs from the region's median by more than a named constant), then
> measure what fraction of that mask still shows ink-like pixels in the rendered image; also return
> `Halo` - the residual measured in a narrow band just inside the plate edge, where a block fill
> typically leaves the tops and tails of the original glyphs. `RenderedContrast(rendered image.Image,
> p evidence.Plate) float64` verifies the strategic §7 rule that colour adaptation is not permission to
> draw unreadable text: measure the actual luma separation between the plate's drawn text pixels and
> its drawn background *in the rendered image*, not from the sampled source values.

**Verification:**
- `func ResidualInk(source, rendered image.Image, g truth.Group, w, h int) ResidualScore` matches exactly once.
- `func RenderedContrast(rendered image.Image, p evidence.Plate) float64` matches exactly once.
- `ResidualScore` declares `Residual` and `Halo`.
- A test on the uniform-paper synthetic scene, comparing the source against a fully-painted copy,
  asserts `Residual == 0`; comparing against the untouched source asserts `Residual > 0.5`.

**Status:** `[x]` done

---

### Step 03.7 - Damage: overlay pixels where they are forbidden

**Files:** `tools/ocrlab/metrics/damage.go`
**Depends on:** Step 03.6

**Prompt for developer:**
> Implement `Damage(plates []evidence.Plate, a *truth.Annotation, w, h int) DamageScore` returning
> `OutsideReplaceArea` (overlay pixels not inside any group's `ReplaceArea`), `ProtectedHit` (overlay
> pixels inside a `Protected` region) and `WorstProtectedRegion` (the region ID with the largest hit).
> Protected damage is reported in absolute pixels and as a fraction of the protected region, because
> "0.4% of the panel" and "the whole balloon outline" need to be distinguishable.

**Verification:**
- `func Damage(plates []evidence.Plate, a *truth.Annotation, w, h int) DamageScore` matches exactly once.
- `DamageScore` declares `OutsideReplaceArea`, `ProtectedHit`, `WorstProtectedRegion`.
- A test on the bordered-balloon synthetic scene asserts a plate covering the border reports a non-zero
  `ProtectedHit` and names the border region.

**Status:** `[x]` done

---

### Step 03.8 - Replacement: clipping, collision, overflow

**Files:** `tools/ocrlab/metrics/replacement.go`
**Depends on:** Step 03.7

**Prompt for developer:**
> Implement `Replacement(plates []evidence.Plate, groups []truth.Group, w, h int) ReplacementScore`
> counting `Clipped` (plates the browser reported as overflowing after the re-fit ran - the evidence
> carries `scrollHeight`/`clientHeight` per plate, so this is read, not guessed), `CrossGroupOverlap`
> (a plate overlapping another *annotated* group's bounds above a coverage fraction), `OutOfBounds`
> (plate pixels outside the image) and `PlateOverlap` (plate-on-plate). Score it once per stress case
> present in the evidence and return the per-case breakdown.

**Verification:**
- `func Replacement(...) ReplacementScore` matches exactly once.
- `ReplacementScore` declares `Clipped`, `CrossGroupOverlap`, `OutOfBounds` and a per-stress-case map.
- A test asserts a plate with `scrollHeight > clientHeight` counts as clipped.

**Status:** `[x]` done

---

### Step 03.9 - Compose the per-scene score record

**Files:** `tools/ocrlab/metrics/score.go`
**Depends on:** Step 03.8

**Prompt for developer:**
> Implement `Score(ev *evidence.Scene, a *truth.Annotation, s *corpus.Scene, src, rendered image.Image)
> (*SceneScore, error)` folding all seven measures plus `Cost` (the evidence's `OcrMs`, `RenderMs`,
> `PeakRSSBytes`) into one record, and `Aggregate(scores []*SceneScore) Summary` grouping by category
> and by split. `Score` returns an error - never a zero score - when the annotation is not
> `IsTruth()`, so an unreviewed or OCR-seeded scene cannot enter an aggregate.

**Verification:**
- `func Score(...) (*SceneScore, error)` matches exactly once and returns an error for a non-truth annotation.
- `Aggregate` returns per-category and per-split breakdowns.
- No file in `tools/ocrlab/metrics/` contains the word `threshold` or compares a measure to a bound.

**Status:** `[x]` done

## Phase done criteria

- [x] Every `Step 03.*` is `[x] done`.
- [x] `go test ./tools/ocrlab/...` passes, with at least one test per dimension bound to a
      synthetic scene from Phase 02.
- [x] Grep for `TODO(phase-03)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Handoff notes

`evidence.Run` is the contract between the two editions - Phase 04 and Phase 05 both emit it, Phase 05
is guarded against drifting from it. The metrics package measures and never judges: no thresholds live
here, so Phase 06 can set bounds without touching measurement code, and a later threshold change can
never silently alter a metric. Every function takes truth and evidence, never an OCR engine - scoring
runs offline from a saved run.

## Rollback plan

Revert the phase commit(s). Nothing outside `tools/ocrlab/metrics/` and `tools/ocrlab/evidence/` is touched.
