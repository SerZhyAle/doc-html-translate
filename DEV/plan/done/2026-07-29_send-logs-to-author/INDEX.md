# Tactical plan: 2026-07-29_send-logs-to-author - send-logs-to-author

**Strategic spec:** [`../2026-07-29_send-logs-to-author.md`](../2026-07-29_send-logs-to-author.md)
**Research inputs:** none
**Tier:** Moderate · **Priority:** 50
**Status:** BlockNeedUserTest
**Phases:** 7 / 7 done
**Last updated:** 2026-08-11

> **Scope:** tactical, English, developer handoff. Every step has a verification predicate.
> Rationale lives in the strategic spec.

## Phase overview

| # | Phase | Depends on | Status | Steps | File |
|---|-------|-----------|--------|------:|------|
| 01 | run-log-store | - | ✅ Done | 5/5 | [PHASE_01__run-log-store.md](PHASE_01__run-log-store.md) |
| 02 | report-archive | 01 | ✅ Done | 5/5 | [PHASE_02__report-archive.md](PHASE_02__report-archive.md) |
| 03 | gui-endpoints | 02 | ✅ Done | 4/4 | [PHASE_03__gui-endpoints.md](PHASE_03__gui-endpoints.md) |
| 04 | gui-about-section | 03 | ✅ Done | 5/5 | [PHASE_04__gui-about-section.md](PHASE_04__gui-about-section.md) |
| 05 | cli-report-action | 02, 03 | ✅ Done | 4/4 | [PHASE_05__cli-report-action.md](PHASE_05__cli-report-action.md) |
| 06 | extension-diagnostics | - | ✅ Done | 5/5 | [PHASE_06__extension-diagnostics.md](PHASE_06__extension-diagnostics.md) |
| 07 | docs-cleanup | all | ✅ Done | 8/8 | [PHASE_07__docs-cleanup.md](PHASE_07__docs-cleanup.md) |

Legend: ⬜ Not started · 🚧 In Progress · ✅ Done · ⛔ Blocked · ⏭️ Skipped

Phase 06 has no code dependency on 01-05 (separate codebase, clipboard text only) and may run
in parallel; it must land before phase 07, which documents both shapes.

## Pre-implementation blockers

None. The three decisions that gated this ticket were taken by the owner on 2026-07-29 and are
recorded in strategic §3.3 and §6 (items 1-3): no programmatic attachment, standard archive
contents, extension in scope as clipboard text.

Strategic §6 items 4 and 5 are non-blocking defaults chosen here:

- **Log-store bound** (§6.4): 20 runs or 20 MB, whichever comes first, trimmed oldest-first.
- **Archive cap** (§6.4): 15 MB, oldest logs dropped first, drop count reported.
- **Subject shape** (§6.5): `doc-html-translate <version> - log report`.

## Completion gate

- [x] All phases ✅ Done.
- [x] `./scripts/check.ps1` green (lint + tests + typo gate). 2026-08-11, exit 0.
- [x] `npm test` green in `extension/`. 2026-08-11, 93 pass / 0 fail.
- [x] User-facing docs updated per strategic §8 (phase 07 covers the surface list in
      [`DEV/DOCS_SURFACES.md`](../../../DOCS_SURFACES.md)).
- [x] Changelog has an entry per modified file.
- [ ] **Hands-on checks (strategic §11 criteria 7-10) - these cannot be asserted statically:**
  - [ ] Portable build: button opens the default mail program with the author's address, the
        subject and the body template; the archive's folder is open with the file selected and
        the path pastes from the clipboard.
  - [ ] Packaged MSIX build: the same, from the read-only install directory.
  - [ ] No network connection is opened by the app during a report (observed, not inferred).
  - [ ] Extension: the copy-diagnostics button puts the report text on the clipboard in Chrome
        and in Edge.
- [ ] `/spec-check 2026-07-29_send-logs-to-author` returns Verified.

Set the strategic status to `BlockNeedUserTest` when phases 01-07 are ✅ and only the hands-on
list above is open.

## How to track progress

1. Before a phase: flip its row to 🚧, update `Phases: X/N`.
2. During: flip a step to `[~]` when started, `[x]` when its Verification passes - never on
   intent.
3. On phase done: confirm every step `[x]`, confirm Done Criteria, flip row to ✅, bump.
4. If blocked: flip to ⛔, log it; if the whole spec is blocked, set a `Block*` status.

## Change log

- 2026-07-29 - initial tactical plan authored by /spec-tech.
- 2026-08-11 - phase 07 executed by /spec-dev; status -> BlockNeedUserTest. Steps 07.1-07.2 were found
  already satisfied by uncommitted work from an earlier session and were verified rather than
  rewritten. The owner confirmed the support-block (not hero) placement for step 07.3. Only the four
  hands-on checks below are open.
