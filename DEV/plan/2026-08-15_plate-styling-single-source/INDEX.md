# Tactical plan: 2026-08-15_plate-styling-single-source

**Strategic spec:** [`../2026-08-15_plate-styling-single-source.md`](../2026-08-15_plate-styling-single-source.md)
**Research inputs:** none
**Tier:** Moderate · **Priority:** 44
**Status:** Not started
**Phases:** 0 / 6 done
**Last updated:** 2026-08-15

> **Scope:** tactical, English, developer handoff. Every step has a verification predicate.
> Rationale lives in the strategic spec.

## Phase overview

| # | Phase | Depends on | Status | Steps | File |
|---|-------|-----------|--------|------:|------|
| 01 | canonical-source | - | ⬜ Not started | 0/3 | [PHASE_01__canonical-source.md](PHASE_01__canonical-source.md) |
| 02 | desktop-derives | 01 | ⬜ Not started | 0/4 | [PHASE_02__desktop-derives.md](PHASE_02__desktop-derives.md) |
| 03 | extension-derives | 01 | ⬜ Not started | 0/5 | [PHASE_03__extension-derives.md](PHASE_03__extension-derives.md) |
| 04 | parity-gate | 02, 03 | ⬜ Not started | 0/4 | [PHASE_04__parity-gate.md](PHASE_04__parity-gate.md) |
| 05 | parity-record | 04 | ⬜ Not started | 0/3 | [PHASE_05__parity-record.md](PHASE_05__parity-record.md) |
| 06 | docs-cleanup | all | ⬜ Not started | 0/2 | [PHASE_06__docs-cleanup.md](PHASE_06__docs-cleanup.md) |

Legend: ⬜ Not started · 🚧 In Progress · ✅ Done · ⛔ Blocked · ⏭️ Skipped

Phases 02 and 03 both depend only on 01 and touch disjoint trees, so they may be done in either
order or in parallel; 04 needs both.

## Pre-implementation blockers

None. Strategic §6 item 1 is Resolved (scope = overlay unit + reader theme palette); items 2 and 3
are constraints carried into the phases below, not research gates:

- Item 2 (offline self-contained output) is honoured by Phase 02 deriving at run time and still
  emitting the styles inline.
- Item 3 (unknown equivalence cases surfacing on the first comparison) is Step 04.3, which triages
  what the gate finds into "unify" or "name it".

## Completion gate

- [ ] All phases ✅ Done.
- [ ] User-facing docs updated only if strategic §8 mandates it - it does not; the reader sees no
      change, so `README*` and the site stay untouched.
- [ ] Changelog has an entry per modified file.
- [ ] `/spec-check 2026-08-15_plate-styling-single-source` returns Verified.

## How to track progress

1. Before a phase: flip its row to 🚧, update `Phases: X/N`.
2. During: flip a step to `[~]` when started, `[x]` when its Verification passes - never on intent.
3. On phase done: confirm every step `[x]`, confirm Done Criteria, flip row to ✅, bump.
4. If blocked: flip to ⛔, log it; if the whole spec is blocked, set a `Block*` status.

## Change log

- 2026-08-15 - initial tactical plan authored by /spec-tech.
