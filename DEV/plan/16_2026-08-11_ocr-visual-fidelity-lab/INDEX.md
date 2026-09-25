# Tactical plan: 16_2026-08-11_ocr-visual-fidelity-lab - ocr-visual-fidelity-lab

**Strategic spec:** [`../16_2026-08-11_ocr-visual-fidelity-lab.md`](../16_2026-08-11_ocr-visual-fidelity-lab.md)
**Research inputs:** none (strategic §9 items are resolved *by* Phase 06, not before it)
**Tier:** Complex, cross-edition · **Priority:** 40
**Status:** In Progress
**Phases:** 6 / 8 done
**Last updated:** 2026-08-11

> **Scope:** tactical, English, developer handoff. Every step has a verification predicate.
> Rationale lives in the strategic spec.

## Phase overview

| # | Phase | Depends on | Status | Steps | File |
|---|-------|-----------|--------|------:|------|
| 01 | corpus-manifest | - | ✅ Done | 6/6 | [PHASE_01__corpus-manifest.md](PHASE_01__corpus-manifest.md) |
| 02 | ground-truth | 01 | ✅ Done | 6/6 | [PHASE_02__ground-truth.md](PHASE_02__ground-truth.md) |
| 03 | metrics | 02 | ✅ Done | 9/9 | [PHASE_03__metrics.md](PHASE_03__metrics.md) |
| 04 | desktop-runner | 03 | ✅ Done | 6/6 | [PHASE_04__desktop-runner.md](PHASE_04__desktop-runner.md) |
| 05 | extension-runner | 04 | ✅ Done | 5/5 | [PHASE_05__extension-runner.md](PHASE_05__extension-runner.md) |
| 06 | baseline-and-thresholds | 04, 05 | ✅ Done | 5/5 | [PHASE_06__baseline-and-thresholds.md](PHASE_06__baseline-and-thresholds.md) |
| 07 | concealment-and-grouping | 06 | ⬜ Not started | 0/7 | [PHASE_07__concealment-and-grouping.md](PHASE_07__concealment-and-grouping.md) |
| 08 | docs-cleanup | all | 🚧 In Progress | 5/6 | [PHASE_08__docs-cleanup.md](PHASE_08__docs-cleanup.md) |

Legend: ⬜ Not started · 🚧 In Progress · ✅ Done · ⛔ Blocked · ⏭️ Skipped

**Why this order.** The strategic spec forbids tuning before measuring, so every code-changing phase
sits behind an instrument. 01 defines what a scene *is* before anything can hold one. 02 defines truth
independent of the engine, and ships the synthetic scenes that let 03 be unit-tested without any
downloaded media. 03 is pure measurement over 01+02 types, so it can be proven correct before it judges
anything. 04 and 05 are the two editions' evidence producers, both emitting the schema 03 scores; 04
first because it defines the shared evidence schema. 06 is the only phase permitted to set a number.
07 is the first phase allowed to change user-visible rendering, and only against 06's report.

## Layout produced by this plan

```text
tools/ocrlab/                     the lab (Go, never linked into a shipped binary)
  main.go                         ocrlab verify | fetch | synth | seed | run | score | report
  corpus/                         manifest schema, licence gate, split validator
  truth/                          annotation schema, review state
  synth/                          deterministic diagnostic scenes (exact known text + geometry)
  metrics/                        the eight §3.2 dimensions
  evidence/                       the cross-edition run-evidence schema
  report/                         JSON + Markdown + side-by-side HTML
DEV/ocrlab/corpus.json            versioned manifest (metadata only - no media)
DEV/ocrlab/annotations/*.json     versioned ground truth
DEV/ocrlab/thresholds.json        acceptance bounds, written by Phase 06 only
test_doc/ocrlab/                  the media itself - gitignored, never ships
temp/ocrlab/<run-id>/             run artifacts (screenshots, evidence, report)
extension/scripts/ocrlab.mjs      the extension edition's evidence producer
```

## Pre-implementation blockers

None for Phase 01-05. Strategic §9 items 1-5 are open **by design** - each says "decide from the
measurement, not by intuition" - so they gate Phase 07, not Phase 01, and Phase 06 is what resolves
them.

Two gates are **human-owned** and cannot be closed by an agent. They block the *completion* of the
spec, not the phases that build the instrument:

- [ ] **Corpus acquisition (§4.1):** 200+ scenes at the category minimums. Every entry needs a human to
      open the asset's own licence page (§4.2 - "search-result snippets and a site's category label are
      not licence proof"). Phase 01 delivers the manifest, the licence gate and the idempotent fetcher;
      it cannot deliver the human verification.
- [ ] **Holdout annotation review (§4.3):** two reviewers, or one plus a later independent check, per
      holdout scene. Phase 02 delivers the schema, the OCR-seeded draft and the review-state field; a
      draft never counts as truth.

## Deliberately out of scope

Two rows of strategic §6 produce no step, and that is a decision rather than an omission:

- **GUI failure path.** §6 says the GUI "can reveal an honest visual-OCR failure/report path *if the
  tactical design introduces one*". Phase 07 introduces no user-visible failure surface - a scene that
  cannot be concealed is marked a benchmark failure, and the reader-facing fallback stays today's
  behaviour. Revisit only if a later iteration makes the failure user-visible.
- **MSIX.** The lab is a developer tool and is in no package. The only shipped-code change (Phase 04's
  `DOCHT_OCR_DIAG` sidecar) takes its path from the caller and adds no writable-install-directory
  requirement, so the §6 MSIX row is satisfied by construction rather than by a step.

## Completion gate

- [ ] All phases ✅ Done.
- [ ] `DEV/ocrlab/corpus.json` passes `ocrlab verify` with every §4.1 category minimum met and a
      stratified holdout ≥ 30%.
- [ ] `ocrlab run` produces both editions' evidence and one report at a pinned viewport; a missing
      dependency or asset fails the run explicitly.
- [ ] Holdout report shows zero clipping after the stress cases, zero cross-group plate overlap, zero
      unresolved protected-area damage.
- [ ] `docs/PARITY.md` carries every new shared constant / mode, each with a drift guard in `tests/`.
- [ ] `./scripts/test.ps1` green, `./scripts/lint.ps1` green, `npm test` green in `extension/`.
- [ ] Changelog has an entry per modified file.
- [ ] `/spec-check 16_2026-08-11_ocr-visual-fidelity-lab` returns Verified.

## How to track progress

1. Before a phase: flip its row to 🚧, update `Phases: X/8`.
2. During: flip a step to `[~]` when started, `[x]` when its Verification passes - never on intent.
3. On phase done: confirm every step `[x]`, confirm Done Criteria, flip row to ✅, bump the counter.
4. If blocked: flip to ⛔, log it; if the whole spec is blocked, set a `Block*` status on the strategic spec.

## Non-negotiables carried into every phase

- The lab never modifies a source image. Evidence and reconstruction are overlays or copies.
- The lab never calls a paid translation service and never uploads a user image. Stress strings are
  deterministic constants.
- A missing corpus asset, a missing browser or a missing tesseract is an explicit failure, never a
  silently reduced scene count.
- OCR output may seed an annotation; it is never scored as truth.
- No phase before 06 changes a threshold, a constant or a rendering decision.

## Change log

- 2026-08-11 - initial tactical plan authored by /spec-tech.
- 2026-08-11 - phases 01-04 implemented. The lab runs end to end and produced its first baseline:
  8 of 10 scenes scored, 5 carrying a named hard failure. Phases 05-08 not started.
- 2026-08-11 - plan corrections found by running the code are recorded in each phase file's
  Deviations section rather than silently absorbed.
- 2026-08-12 - phase 05 done: all five steps verified, `npm test` 115/115, the new cross-edition
  drift guard proven to fail on a one-sided change, and the extension's evidence grades through
  `ocrlab score` with the same defect the desktop edition reports.
- 2026-08-12 - phase 06 started and stopped at its instrument: `ocrlab gate` and the thresholds
  contract landed (step 06.4, ahead of 06.1-06.3), and a desktop run over all 45 scenes is in
  `temp/ocrlab/base-desktop/` as raw material. **No threshold was written**, which is the honest
  outcome: `ocrlab verify` reports 11 gradable annotations and zero holdout scenes, and a bound
  derived from that would be a number with nothing behind it. Annotating and splitting the corpus
  is the next real work, not more code.
- 2026-08-12 - **two of phase 03's measures were wrong and were corrected before anything was
  tuned against them.** Starting phase 07 against the baseline's top two defect classes found that
  neither was real: the leftover-glyph measure asked whether ink outside a plate survived the
  render, which it always does (an overlay never touches those pixels), and the clip test read
  integer layout rounding as hidden text on 17 scenes - a hard gate that would have failed on
  rounding forever. Both are replaced with measures that discriminate, each with a deterministic
  test that fails on the old behaviour. **No rendering decision was changed**, so the "no phase
  before 06" rule stands and phase 07 is still ⬜ Not started. The corrected picture is in
  [PHASE_06__baseline-and-thresholds.md](PHASE_06__baseline-and-thresholds.md).
- 2026-08-12 - **phase 04's browser transport was rebuilt, out of phase order, on the owner's
  instruction.** Closing phase 05 turned up a red check it did not own: `--dump-dom` and
  `--screenshot` had both silently stopped working in this machine's browsers, so the desktop
  runner produced nothing and phase 06 could not have had a baseline. `tools/ocrlab/runner/cdp.go`
  now drives the browser over the DevTools protocol, with no new module. This changes no threshold,
  no constant and no rendering decision, so the "no phase before 06" rule stands; phase 04 keeps
  its ✅ and carries the write-up.
- 2026-08-12 - **phase 06 done: both editions measured, and the bounds that measurement can carry
  are written.** [`DEV/research/ocrlab/2026-08-11__baseline.md`](../../research/ocrlab/2026-08-11__baseline.md)
  is the reference every later change is compared against. Two results dominate it. The extension
  edition **crashes the browser tab on 17 of 45 scenes**, with a sharp boundary at image size -
  15 of 15 above 2.8 Mpx died, 28 of 30 below survived - while the desktop app completed all 45
  including a 17 Mpx newspaper page; that is a user-visible failure, not a lab artifact, and it is
  the extension's first Phase 07 item. The desktop app's own worst result is **11 scenes with no
  overlay at all**. `thresholds.json` is deliberately small and says why in its own `caveat`: no
  `regressionTolerance` anywhere (nothing to regress against without a holdout), per-category
  bounds only where a category has more than two scored scenes, and overall bounds set *at* the
  measurement rather than above it. Getting there needed a repair first: **the extension's DevTools
  transport had no per-call timeout**, so a wedged renderer hung a whole run for ever with no
  evidence and no failed scene - a one-sided gap against `runner/cdp.go`, which has always bounded
  its calls at 90 s. No shipped code changed and no rendering decision moved.
- 2026-08-11 - a recognition fix landed **out of phase order**, on the owner's instruction to act on
  the loop1 report's 14 unreadable scenes: the grey rescue ladder, written up in
  [`../../research/ocr_grey_rescue_2026-08-11.md`](../../research/ocr_grey_rescue_2026-08-11.md). It
  sets one number (`ocrRescueLineConf`), which is Phase 06's exclusive right, and it changes what a
  reader sees on scenes that previously got no overlay at all, which is Phase 07's. Neither phase
  advances: no step of either is done, both stay ⬜ Not started, and each now carries a note saying
  what to re-derive rather than inherit. The non-negotiable "no phase before 06 changes a threshold,
  a constant or a rendering decision" is recorded as broken here rather than quietly rewritten.
