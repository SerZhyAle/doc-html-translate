# Phase 02 - Ground truth

**Strategic spec:** [`../07_2026-08-11_ocr-visual-fidelity-lab.md`](../07_2026-08-11_ocr-visual-fidelity-lab.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 01
**Steps done:** 6 / 6

## Objective

An engine-independent annotation record per scene - transcript, reading groups, protected areas, review
state - plus a generator of deterministic synthetic scenes whose truth is exact by construction, so
Phase 03 can be tested with no downloaded media.

## Prerequisites

- [x] Phase 01 is ✅ Done.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `tools/ocrlab/truth/truth.go` | New | ≤ 300 |
| `tools/ocrlab/truth/validate.go` | New | ≤ 220 |
| `tools/ocrlab/truth/truth_test.go` | New | ≤ 260 |
| `tools/ocrlab/synth/synth.go` | New | ≤ 400 |
| `tools/ocrlab/synth/synth_test.go` | New | ≤ 200 |
| `tools/ocrlab/main.go` | Modified | ≤ 320 |

## Steps

### Step 02.1 - Define the annotation record

**Files:** `tools/ocrlab/truth/truth.go`
**Depends on:** - start of phase

**Prompt for developer:**
> Create package `truth` with `Annotation` keyed by the Phase 01 `Scene.ID`: `SchemaVersion`, `SceneID`,
> `ImageWidth`, `ImageHeight`, `Groups []Group`, `Protected []Region`, `Ambiguity` (`clear`,
> `partly-illegible`, `illegible`), `StressCases []string`, `ReviewerNote`, and a `Review` sub-record
> (`AnnotatedBy`, `AnnotatedOn`, `CheckedBy`, `CheckedOn`, `Disagreements []string`). A `Group` has
> `ID`, `Type` (`document-line`, `paragraph`, `balloon`, `caption`, `label`, `title`, `incidental`),
> `Transcript`, `Language`, `Direction` (`ltr`/`rtl`), `ReadingOrder int`, `Lines []Region`, `Bounds`
> (the region the group occupies) and `ReplaceArea` (the region a plate is *permitted* to cover, which
> may be tighter than `Bounds`). A `Region` is a `Kind` (`box` or `polygon`) plus `Points [][2]int` in
> natural image pixels - `box` carries two points, `polygon` three or more. Coordinates are always
> natural image pixels; nothing here is expressed in rendered or percent units.

**Verification:**
- `type Annotation struct` and `type Group struct` each match exactly once.
- `ReplaceArea` and `Protected` both appear in the struct declarations.
- `type Region struct` declares `Kind` and `Points [][2]int`.

**Status:** `[x]` done

---

### Step 02.2 - Add the truth loader and its review gate

**Files:** `tools/ocrlab/truth/truth.go`, `tools/ocrlab/truth/validate.go`
**Depends on:** Step 02.1

**Prompt for developer:**
> Add `LoadDir(dir string) (map[string]*Annotation, error)` reading `DEV/ocrlab/annotations/<id>.json`,
> and `Validate(a *Annotation, s *corpus.Scene) []Problem` checking: the scene exists; image dimensions
> match the manifest; every region lies inside the image; `ReplaceArea` is contained in `Bounds`; no
> `ReplaceArea` intersects a `Protected` region; reading orders are a permutation of `1..N`; a group
> with a non-empty transcript has at least one line; and - the gate that matters - a scene in the
> `holdout` split whose `Review.CheckedBy` is empty is a problem, so an unreviewed annotation can
> never silently become a gate. Add `func (a *Annotation) IsTruth() bool` returning false when the
> record is an OCR-seeded draft (see 02.3) or unreviewed.

**Verification:**
- `func Validate(a *Annotation, s *corpus.Scene) []Problem` matches exactly once.
- `func (a *Annotation) IsTruth() bool` matches exactly once.
- A test asserts `Validate` reports a problem for a holdout scene with an empty `CheckedBy`.

**Status:** `[x]` done

---

### Step 02.3 - Add the OCR-seeded draft, marked as not-truth

**Files:** `tools/ocrlab/truth/truth.go`, `tools/ocrlab/main.go`
**Depends on:** Step 02.2

**Prompt for developer:**
> Add an `Origin` field to `Annotation` with values `human` and `ocr-seed`, defaulting to `human` when
> absent is not acceptable - make the JSON tag non-omitempty so an unset origin is visible. Add the
> `ocrlab seed <scene-id>` subcommand: run `ocr.Recognize` over the scene's media, convert each
> recognized block into a draft `Group` (type `incidental`, transcript = recognized text), write
> `DEV/ocrlab/annotations/<id>.draft.json` with `Origin: ocr-seed` and an empty `Review`, and refuse to
> overwrite a non-draft file. `IsTruth()` returns false for `ocr-seed`, so the scorer can never grade
> the engine against itself.

**Verification:**
- The `seed` subcommand is registered in `tools/ocrlab/main.go` exactly once.
- `seed` writes to a `*.draft.json` path and the write path refuses an existing `<id>.json`.
- A test asserts `(&Annotation{Origin: "ocr-seed"}).IsTruth() == false`.

**Status:** `[x]` done

---

### Step 02.4 - Generate deterministic synthetic scenes

**Files:** `tools/ocrlab/synth/synth.go`
**Depends on:** Step 02.2

**Prompt for developer:**
> Create package `synth` producing scenes whose text and geometry are known exactly because the lab
> drew them, using `golang.org/x/image/font` with a bundled basic face - no new dependency, no network,
> no licence question. Emit at least these eight, each as a PNG plus its `Annotation`: uniform paper
> with three document lines; two-column document; a white speech balloon with a border and a tail over
> a coloured panel (the border is a `Protected` region); two adjacent balloons close enough to tempt a
> merge; a caption box over a photographic gradient; text over a halftone texture; large display
> lettering on a coloured fill; and a right-to-left line. Every generator is a pure function of a
> fixed seed, so re-running produces byte-identical PNGs.

**Verification:**
- `func Generate(root string) ([]corpus.Scene, []*truth.Annotation, error)` matches exactly once.
- At least eight distinct scene builders are declared.
- A test generates twice into different directories and asserts the SHA-256 of each PNG is equal.

**Status:** `[x]` done

---

### Step 02.5 - Wire `ocrlab synth`

**Files:** `tools/ocrlab/main.go`
**Depends on:** Step 02.4

**Prompt for developer:**
> Add the `synth` subcommand: regenerate the synthetic scenes under `<root>/synthetic/`, write their
> annotations to `DEV/ocrlab/annotations/`, and merge or update their entries in the manifest with
> `Licence: SYNTHETIC`, `LicenceVerifiedBy: "generated"`, the measured SHA-256, and
> `Split: dev`. Re-running must be a no-op in the manifest diff.

**Verification:**
- `go run ./tools/ocrlab synth` exits 0 and creates at least eight PNGs under `test_doc/ocrlab/synthetic/`.
- Running it twice leaves `DEV/ocrlab/corpus.json` byte-identical the second time.
- Every generated scene's entry carries `"licence": "SYNTHETIC"`.

**Status:** `[x]` done

---

### Step 02.6 - Fold truth validation into `ocrlab verify`

**Files:** `tools/ocrlab/main.go`, `tools/ocrlab/truth/validate.go`
**Depends on:** Step 02.5

**Prompt for developer:**
> Extend `verify` to load the annotation directory and report `truth.Validate` problems alongside the
> corpus problems, plus one more class: a scored scene (any scene in the manifest that is not marked
> `Note: "unscored"`) with no annotation at all. Keep the exit-code contract: any problem exits 1.

**Verification:**
- `go run ./tools/ocrlab verify` output contains both a corpus section and an annotation section.
- A scene present in the manifest with no annotation file is reported by ID.

**Status:** `[x]` done

## Phase done criteria

- [x] Every `Step 02.*` is `[x] done`.
- [x] `go test ./tools/ocrlab/...` passes.
- [x] Grep for `TODO(phase-02)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Deviations from the written plan

Two, both found by running the code and both recorded rather than quietly absorbed:

1. **Step 02.2 said `ReplaceArea` must be contained in `Bounds`. That rule is wrong** and the
   validator rejected three correct scenes. Inside a speech balloon the permitted paint area is the
   balloon's *interior*, which is legitimately larger than the text box - that surplus is what gives a
   longer translation room to reflow. The rule implemented instead is the one that carries the
   meaning: the replacement area must **cover the group's own text** (2% slack for descender
   overhang), and must not intersect a protected region. `RuleReplaceEscapes` became
   `RuleReplaceMissesText`.
2. **Geometry lives in `truth/region.go`, not `metrics/geometry.go`.** `truth.Validate` needs exact
   polygon overlap - a replacement area clipping a balloon outline has to be caught when the
   annotation is written, not only when a plate lands on it - and `metrics` already imports `truth`,
   so putting the rasterizer in `metrics` would be an import cycle. Phase 03 step 03.2 consumes
   `truth.Region.Rasterize` / `truth.Mask` instead of declaring its own; its verification predicate
   was updated to match.

A third finding was a defect in the fixture rather than the plan: the adjacent-balloon scene was
first drawn with 65 px between its two text blocks, while the clusterer only merges within
`1.2 x` the median line height (~26 px at that type size). The scene looked like a merge trap and
was not one. The balloons were moved to a 20 px text gap and the test now derives the bound from the
drawn line height instead of a hand-picked number.

## Handoff notes

The synthetic scenes are the only corpus content that exists without human licence work, so every
Phase 03 metric test binds to them. `IsTruth()` is the single predicate the scorer must consult before
grading a scene - nothing else decides whether an annotation counts.

Current honest state of `ocrlab verify`: 10 scenes, 8 gradable (the 8 generated), 0 holdout. The two
Commons images wait on the human licence gate and on annotation; every §4.1 category minimum is unmet
and the command says so and exits 1.

## Rollback plan

Revert the phase commit(s) and delete `test_doc/ocrlab/synthetic/` (regenerable).
