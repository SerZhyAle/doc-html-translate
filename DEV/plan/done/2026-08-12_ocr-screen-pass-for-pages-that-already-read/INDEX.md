# Tactical plan: 2026-08-12_ocr-screen-pass-for-pages-that-already-read

**Strategic spec:** [`../2026-08-12_ocr-screen-pass-for-pages-that-already-read.md`](../2026-08-12_ocr-screen-pass-for-pages-that-already-read.md)
**Research inputs:** [`../../research/ocr_halftone_2026-08-12.md`](../../../research/ocr_halftone_2026-08-12.md) (section 5 - the measured gain)
**Tier:** Cross-edition feature · **Priority:** 45
**Status:** BlockNeedUserTest
**Phases:** 4 / 4 done
**Last updated:** 2026-08-12

> Scope: tactical, English, developer handoff. Every step has a verification predicate.
> Rationale lives in the strategic spec.

## Phase overview

| # | Phase | Depends on | Status | Steps | File |
|---|-------|-----------|--------|------:|------|
| 01 | uncovered-screen-trigger | - | ✅ Done | 3/3 | [PHASE_01__uncovered-screen-trigger.md](PHASE_01__uncovered-screen-trigger.md) |
| 02 | merge-rule | 01 | ✅ Done | 3/3 | [PHASE_02__merge-rule.md](PHASE_02__merge-rule.md) |
| 03 | wire-the-sweep | 01, 02 | ✅ Done | 2/2 | [PHASE_03__wire-the-sweep.md](PHASE_03__wire-the-sweep.md) |
| 04 | parity-and-docs | 03 | ✅ Done | 3/3 | [PHASE_04__parity-and-docs.md](PHASE_04__parity-and-docs.md) |

Legend: ⬜ Not started · 🚧 In Progress · ✅ Done · ⛔ Blocked · ⏭️ Skipped

**Why this order.** The two pieces the strategic spec calls "a real design, not a repositioning" are
the trigger and the merge rule, and both are pure functions over pixels and rectangles. They are
built and unit-tested first, in phases that change no user-visible behaviour at all, so that phase 03
- the one that can regress a page which reads fine today - is a wiring change over parts already
proven. Parity and docs come last because the shared constants they record are only final once the
wiring names them.

## The design decisions this plan makes

The strategic spec lists four questions. Three are answered here; the fourth is not answerable
without material nobody has yet, and is recorded as such rather than guessed.

- **Trigger** - the second recognition runs only when a halftone screen is measured **in the parts of
  the picture no plate covers**. The detector already votes tile by tile, so "screened area with no
  plate over it" costs one extra sweep over at most 96 tiles rather than a second pass over the page.
  A tile more than half covered by an existing plate is dropped from the vote: it is area the reader
  is already served on, and counting it would let a screened balloon trigger a sweep that can only
  find what is already there.
- **Merge** - every plate the ordinary pass produced survives untouched; a screen-pass plate joins it
  only when the ordinary plates cover at most `ocrScreenMergeMaxOverlap` of its area. The measure is
  the **union** of the overlaps, not their sum, which is what answers the spec's "what happens to a
  new plate that overlaps two": two existing plates each covering a third of a candidate leave it two
  thirds covered, and it is dropped.
- **Confidence** - the sweep keeps `ocrRescueLineConf` (80) and the plan states plainly that this is
  a floor argued for, not a floor re-derived. The local prior is the same one the rescue floor was
  derived against - *nothing was found in this region* - while the cost of a wrong plate is strictly
  higher here, because it lands on a page the reader is otherwise happy with. That argues 80 is a
  lower bound. Turning it into a measured number needs the annotated whole pages below.
- **Measurement** - not closed by this plan. See the human-owned gate.

## Human-owned gate (blocks the spec, not these phases)

- [ ] **Three real screened whole pages with reviewed annotations.** The corpus holds three screened
      scenes and all three are caption crops, so they cannot score plate composition - concealment,
      damage, merges/splits and cross-group overlap are exactly the dimensions an additive pass moves.
      Until those pages exist, the strategic spec's first three Done criteria cannot be ticked and the
      confidence floor cannot be re-derived. `ocrlab` grades them the day they land.

## Non-negotiables carried into every phase

- **Every plate the ordinary pass produced is still there, unchanged.** The sweep is additive; it
  never re-reads, re-boxes or drops an existing plate.
- **A picture with no screen outside its plates costs nothing but the detector.** No second
  recognition, no blurred copy written.
- Both editions move together, or the difference is recorded in `docs/PARITY.md` with its reason.
- Every new shared constant gets a drift guard in `tests/`.

## Change log

- 2026-08-12 - tactical plan authored, strategic spec Draft -> Tactical -> In Progress.
- 2026-08-12 - all four phases done. The feature is in both editions, the two new shared constants
  and three structural facts are guarded, `./scripts/test.ps1`, `./scripts/lint.ps1` and `npm test`
  (117/117) are green. Strategic status `BlockNeedUserTest`, **not** `Implemented`: three of the
  ticket's six Done criteria are about what the pass does to plate composition on real screened
  pages, and the corpus has none - its three screened scenes are caption crops. Writing the code
  did not create the material to judge it on, and calling it done would be claiming otherwise.
- 2026-08-12 - deviations, both recorded in their phase files rather than absorbed: the line budgets
  for `screen.go` and `ocr-screen.js` were raised (they were guessed before the union-overlap helper
  existed), and phase 02's straddling test case was rebuilt after its first draft asked for
  something arithmetically impossible - full coverage by two plates that each stay under the bound.
