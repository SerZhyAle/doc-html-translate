# Tactical plan: 2026-07-28_thirteen-ui-languages - thirteen-ui-languages

**Strategic spec:** [`../2026-07-28_thirteen-ui-languages.md`](../2026-07-28_thirteen-ui-languages.md)
**Research inputs:** none
**Tier:** cross-edition feature · **Priority:** 30
**Status:** Partial (audited 2026-07-28)
**Phases:** 7 / 9 done (07 and 08 each carry one open step)
**Last updated:** 2026-07-28

> **Scope:** tactical, English, developer handoff. Every step has a verification predicate.
> Rationale lives in the strategic spec. Decisions D1-D6 there are settled - do not reopen them in a step.

## Phase overview

| # | Phase | Depends on | Status | Steps | File |
|---|-------|-----------|--------|------:|------|
| 01 | typography-guard-scope | - | ✅ Done | 4/4 | [PHASE_01__typography-guard-scope.md](PHASE_01__typography-guard-scope.md) |
| 02 | go-i18n-layer | 01 | ✅ Done | 5/5 | [PHASE_02__go-i18n-layer.md](PHASE_02__go-i18n-layer.md) |
| 03 | cli-page-and-flag | 02 | ✅ Done | 9/9 | [PHASE_03__cli-page-and-flag.md](PHASE_03__cli-page-and-flag.md) |
| 04 | gui-thirteen-languages | 03 | ✅ Done | 7/7 | [PHASE_04__gui-thirteen-languages.md](PHASE_04__gui-thirteen-languages.md) |
| 05 | extension-thirteen-languages | 01 | ✅ Done | 7/7 | [PHASE_05__extension-thirteen-languages.md](PHASE_05__extension-thirteen-languages.md) |
| 06 | site-and-readme | 03 | ✅ Done | 6/6 | [PHASE_06__site-and-readme.md](PHASE_06__site-and-readme.md) |
| 07 | screenshots | 03, 04, 05 | 🚧 In Progress | 4/5 | [PHASE_07__screenshots.md](PHASE_07__screenshots.md) |
| 08 | packaging-and-listings | 07 | 🚧 In Progress | 6/7 | [PHASE_08__packaging-and-listings.md](PHASE_08__packaging-and-listings.md) |
| 09 | docs-cleanup | all | ✅ Done | 5/5 | [PHASE_09__docs-cleanup.md](PHASE_09__docs-cleanup.md) |

Legend: ⬜ Not started · 🚧 In Progress · ✅ Done · ⛔ Blocked · ⏭️ Skipped

**Why this order.** Phase 01 first because the typography guard fails on correct Chinese, German and
Arabic punctuation - every later commit would be red. Phase 02 produces the layer every Go consumer needs.
Phase 03 lands the flag the screenshot tooling drives and the parity test demands. Phase 05 depends only
on 01 (separate codebase, no Go artifacts) and can run in parallel with 03/04. Phase 07 consumes the
finished UIs; phase 08 consumes phase 07's images.

## Language set (all phases)

`en ru uk de it es fr pt ar hi bn ur zh` - 13, in this order, `en` is the key language.
RTL: `ar`, `ur`. Font stack: `Nirmala UI` for `hi`/`bn`, `Microsoft YaHei UI` for `zh`.
Extension `_locales` dirs use `pt` and `zh_CN`; Store locales use `pt-br` and `zh-hans`.

## Pre-implementation blockers

None. Strategic decisions D1-D6 are settled (2026-07-28); no open research items.

## Completion gate

- [ ] All phases ✅ Done.
- [x] `docs/PARITY.md` carries the UI-language-set invariant and the `<html lang>` rule.
- [x] `DEV/DOCS_SURFACES.md` states which surfaces are 13-language and which stay 3-language.
- [x] Changelog has an entry per modified file.
- [x] `./scripts/test.ps1` green, `./scripts/lint.ps1` green, `npm test` 88/88 green in `extension/` (2026-07-28).
- [ ] `/spec-check 2026-07-28_thirteen-ui-languages` returns Verified.

## How to track progress

1. Before a phase: flip its row to 🚧, update `Phases: X/9`.
2. During: flip a step to `[~]` when started, `[x]` when its Verification passes - never on intent.
3. On phase done: confirm every step `[x]`, confirm Done Criteria, flip row to ✅, bump the counter.
4. If blocked: flip to ⛔, log it; if the whole spec is blocked, set a `Block*` status on the strategic spec.

## Translation-content note

Steps that say "add translations for <languages>" mean 13 columns of real copy, not placeholders. An empty
string is a silent English fallback at runtime and passes every key-set test - so the per-phase tests check
for **non-empty** values. en/ru/uk are author-proofread; the other ten are machine-produced and disclosed as
such (strategic D5).

## Change log

- 2026-07-28 - initial tactical plan authored by /spec-tech.
