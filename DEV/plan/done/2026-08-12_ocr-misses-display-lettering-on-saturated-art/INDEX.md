# Tactical plan: 2026-08-12_ocr-misses-display-lettering-on-saturated-art

**Strategic spec:** [`../2026-08-12_ocr-misses-display-lettering-on-saturated-art.md`](../2026-08-12_ocr-misses-display-lettering-on-saturated-art.md)
**Research inputs:** [`../../research/ocr_display_lettering_2026-08-12.md`](../../../research/ocr_display_lettering_2026-08-12.md), [`../../research/ocr_halftone_2026-08-12.md`](../../../research/ocr_halftone_2026-08-12.md), [`../../research/ocr_grey_rescue_2026-08-11.md`](../../../research/ocr_grey_rescue_2026-08-11.md)
**Tier:** Moderate · **Priority:** 46
**Status:** Implemented - every phase done, every strategic done-criterion met (see below)
**Phases:** 7 / 7 done
**Last updated:** 2026-08-13

> **Scope:** tactical, English, developer handoff. Every step has a verification predicate.
> Rationale lives in the strategic spec.

## Phase overview
| # | Phase | Depends on | Status | Steps | File |
|---|-------|-----------|--------|------:|------|
| 01 | corpus-scenes | - | ✅ Done | 3/3 | [PHASE_01__corpus-scenes.md](PHASE_01__corpus-scenes.md) |
| 02 | strength-measure | 01 | ✅ Done | 4/4 | [PHASE_02__strength-measure.md](PHASE_02__strength-measure.md) |
| 03 | sparse-rung | 02 | ✅ Done | 3/3 | [PHASE_03__sparse-rung.md](PHASE_03__sparse-rung.md) |
| 04 | ladder-reachability | 03 | ✅ Done | 4/4 | [PHASE_04__ladder-reachability.md](PHASE_04__ladder-reachability.md) |
| 05 | extension-port | 04 | ✅ Done | 4/4 | [PHASE_05__extension-port.md](PHASE_05__extension-port.md) |
| 06 | docs-cleanup | all | ✅ Done | 3/3 | [PHASE_06__docs-cleanup.md](PHASE_06__docs-cleanup.md) |
| 07 | type-size-break | 04, 05, 06 | ✅ Done | 5/5 | [PHASE_07__type-size-break.md](PHASE_07__type-size-break.md) |

Legend: ⬜ Not started · 🚧 In Progress · ✅ Done · ⛔ Blocked · ⏭️ Skipped

## Pre-implementation blockers

None blocking Phase 01. The strategic spec's open questions are answered *by* this plan rather than
before it:

- **What counts as weak** - chosen in Phase 02 from the baseline run, not from the reported scene.
- **Replace or merge** - replacement, settled in Phase 04: the rescue re-reads the same region, so a
  merge would stack two plates over one word. The comparator is what makes replacement safe.
- **Where the segmentation choice lives** - the strategic spec proposed keying it on input origin
  (standalone image versus book page). This plan does not: sparse segmentation becomes a **rescue
  rung** instead, which needs no new plumbing through the pipeline and is unreachable for any image
  that already reads. If the corpus run in Phase 04 shows the rung firing too late to help real
  posters, input origin returns as a follow-up ticket rather than as a late change here.

**Deliberately out of scope:** re-deriving `ocrRescueLineConf`. The strategic spec raises it, and it
is a real question - 80 was chosen for "second guess after finding nothing", and replacing a weak
result is a different prior. Nothing in Phase 04 depends on the answer, and moving a shared
confidence floor is a change of its own size with its own corpus evidence. If the Phase 04 run shows
the floor rejecting genuine rescued lettering, it becomes a follow-up ticket.

## Baseline

`temp/ocrlab/base06-desktop` - the most recent **desktop** run on this machine, made before any
change in this ticket. Every "did not regress" claim in Phases 04 and 05 is against that directory by
name.

**Correction, made during execution.** This plan first named `temp/ocrlab/fix02` as the baseline.
That run's `summary.json` records `"edition": "extension"`, so it cannot serve as the desktop
edition's baseline; `base06-desktop` (11 scenes, mean recall 0.73) is the desktop run and `fix02` is
the extension one. Both are named where they belong rather than one standing in for the other.

## What the measurement changed in this plan

Recorded here because the plan's own reasoning moved, and a reader comparing plan to diff would
otherwise think the diff drifted:

- **The strategic spec's first cause is not what happens.** It said one junk plate leaves the rescue
  ladder unreachable. Measured on the reported scene in both language configurations, the ordinary
  pass returns *no* plates, the outer trigger fires, and the whole ladder runs and still returns
  almost nothing. No weakness floor was added: nothing in the corpus asked for one, and adding an
  unmeasured threshold is what rule 2 of the lab exists to prevent. The question is recorded as open
  rather than answered.
- **What actually suppressed the cure is inside the ladder:** `greyRescue` returned the first rung
  that produced any plate at all, and rung 1 produces exactly one word. Phase 04 therefore replaces
  *first-non-empty-wins* with *strongest-wins* under the Phase 02 comparator, which needs no
  threshold at all.
- **`resultStrength` is a word count, not a struct of three quantities.** Area and line count were
  planned as candidate signals for a floor; with no floor to choose, keeping them would be dead code.
- **One extra defect was fixed on the way, reported by the user from this scene:** plate colours came
  back inverted on display capitals (`blockColors` took the median over the block as the paper). It
  is in the same files and the same parity contract, so it is carried here rather than in a ticket of
  its own, and it has its own test and its own parity pin.

## Completion gate
- [x] All phases ✅ Done.
- [x] User-facing docs untouched - strategic §8 mandates no change (recognition quality is not
      claimed publicly).
- [x] Changelog has an entry per modified file.
- [x] The strategic spec's second done-criterion is met: the reported scene is scored against its
      annotation and its body group matches at IoU 0.93 (recall 0.00 -> 0.50, cross-group overlaps
      6 -> 0). Phase 07. The earlier reading - that closing it would mean nudging
      `ocrClusterPitchFactor` - was wrong, and measuring the scene is what corrected it: no pitch
      bound cuts in the right place here, because the step a reader would cut at is smaller than a
      step inside the body. `ocrClusterPitchFactor` was not touched.
- [ ] `/spec-check 2026-08-12_ocr-misses-display-lettering-on-saturated-art` returns Verified.
      Not run yet - the audit is the next step, not a claim this plan may make about itself.

## How to track progress
1. Before a phase: flip its row to 🚧, update `Phases: X/N`.
2. During: flip a step to `[~]` when started, `[x]` when its Verification passes - never on intent.
3. On phase done: confirm every step `[x]`, confirm Done Criteria, flip row to ✅, bump.
4. If blocked: flip to ⛔, log it; if the whole spec is blocked, set a `Block*` status.

## Change log
- 2026-08-12 - initial tactical plan authored by /spec-tech.
- 2026-08-13 - Phase 06 closed. Its three steps had been executed the day before without their
  checkboxes being flipped, so the plan read 5/6 while the work was on disk; each predicate was
  re-run against the tree, `add_to_dev_log.ps1` was fixed (it appended to the oldest end of a
  newest-first table) and the two rows it misfiled were moved. Gate re-run fresh and green:
  `test.ps1`, `lint.ps1`, `typo.ps1`, `check.ps1`, `npm test` 131/131. The completion gate's last
  box stays open on purpose - see the note under it.
- 2026-08-13 - **Phase 07 added and done.** The plan closed at `Partial` believing the reported
  scene's merge was a pitch question and therefore P47's. Measuring the scene's own line boxes showed
  otherwise - the cut a reader wants is *smaller* than a step inside the body, so no pitch bound
  reaches it - and the separating quantity is type size. `ocrTypeSizeRatio` (1.6) is bracketed by two
  corpus measurements and joins the shared invariants; the ticket is now `Implemented`.
