# Tactical plan: 2026-07-01_ocr-image-overlay — ocr-image-overlay

**Strategic spec:** [`../2026-07-01_ocr-image-overlay.md`](../2026-07-01_ocr-image-overlay.md)
**Research inputs:** [`research/01__ocr-stack-decisions.md`](research/01__ocr-stack-decisions.md)
**Tier:** Strategic · **Priority:** 50
**Status:** Verified - closed 2026-07-17; production use accepted as the manual test (see the strategic spec)
**Phases:** 8 / 8 done
**Last updated:** 2026-07-01

> **Scope:** tactical, English, developer handoff. Every step has a verification predicate.
> Rationale lives in the strategic spec.

## Phase overview
| # | Phase | Depends on | Status | Steps | File |
|---|-------|-----------|--------|------:|------|
| 01 | ocr-engine-vendor | - | ✅ Done | 4/4 | [PHASE_01__ocr-engine-vendor.md](PHASE_01__ocr-engine-vendor.md) |
| 02 | language-manager | 01 | ✅ Done | 3/3 | [PHASE_02__language-manager.md](PHASE_02__language-manager.md) |
| 03 | overlay-core | 02 | ✅ Done | 4/4 | [PHASE_03__overlay-core.md](PHASE_03__overlay-core.md) |
| 04 | context-menu-page | 03 | ✅ Done | 4/4 | [PHASE_04__context-menu-page.md](PHASE_04__context-menu-page.md) |
| 05 | controls-ui | 02 | ✅ Done | 4/4 | [PHASE_05__controls-ui.md](PHASE_05__controls-ui.md) |
| 06 | epub-integration | 03, 05 | ✅ Done | 3/3 | [PHASE_06__epub-integration.md](PHASE_06__epub-integration.md) |
| 07 | pdf-integration | 06 | ✅ Done | 3/3 | [PHASE_07__pdf-integration.md](PHASE_07__pdf-integration.md) |
| 08 | docs-cleanup | all | ✅ Done | 4/4 | [PHASE_08__docs-cleanup.md](PHASE_08__docs-cleanup.md) |

Legend: ⬜ Not started · 🚧 In Progress · ✅ Done · ⛔ Blocked · ⏭️ Skipped

## Pre-implementation blockers
- All strategic §6 research items are Resolved (see `research/01__ocr-stack-decisions.md`). None
  block Phase 01.

## Completion gate
- [ ] All phases ✅ Done.
- [ ] User-facing docs updated (strategic §8 mandates a feature-docs + privacy-policy update).
- [ ] Changelog has an entry per modified file.
- [ ] `/spec-check 2026-07-01_ocr-image-overlay` returns Verified.

## How to track progress
1. Before a phase: flip its row to 🚧, update `Phases: X/N`.
2. During: flip a step to `[~]` when started, `[x]` when its Verification passes — never on intent.
3. On phase done: confirm every step `[x]`, confirm Done Criteria, flip row to ✅, bump.
4. If blocked: flip to ⛔, log it; if the whole spec is blocked, set a `Block*` status.

## Change log
- 2026-07-01 — initial tactical plan authored by /spec-tech.
