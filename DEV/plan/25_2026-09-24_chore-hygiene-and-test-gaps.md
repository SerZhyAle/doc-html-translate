# Strategic spec: 25_2026-09-24_chore-hygiene-and-test-gaps - Small correctness fixes and missing tests

**Ticket:** 25_2026-09-24_chore-hygiene-and-test-gaps
**Status:** Draft
**Priority:** 40
**Date:** 2026-09-24
**Tier:** Quick Win
**Tactical plan:** `DEV/plan/25_2026-09-24_chore-hygiene-and-test-gaps/` (created by /spec-tech)
**Findings:** P18 P19 P21 P22 Q1-Q5, plus the test gaps listed in §1 (see the [findings register](../research/audit_2026-09-24/README.md))

> **Scope:** STRATEGIC.

---

## 1. Problem
A set of small defects and hygiene issues.
- **Error handling:** the version request is recognised by comparing error text.
- **Flag validation:** numeric flags accept negative values, and some are clamped without telling the user.
- **Console output:** it is written outside the lock that the code comment claims covers it, so parallel progress lines interleave.
- **Log names:** run logs started in the same second share one file, a single run's log has no size cap, and report files started in the same minute overwrite each other.
- **Static analysis:** five findings (an unused function on one platform, error-string style, a raw bidi control character in a literal).
- **Test coverage:** the areas where most of this audit's serious findings live have no tests at all:
  - the pipeline's run, reuse and cleanup logic;
  - document splitting;
  - Markdown input;
  - Windows registration;
  - browser opening;
  - the dialog;
  - helper extraction;
  - the extension's viewer, background and page-OCR modules.

## 2. Goals
1. Special control-flow results are matched by identity, not by text.
2. Every numeric flag is validated at parse time, and any adjustment is reported to the user.
3. Console and log output lines never interleave.
4. Concurrent runs get distinct log and report files, and each run's log is size-capped.
5. The static analyser reports nothing on both platforms.
6. Each untested area listed above has at least a behaviour test for its main path and its error path.

**Non-goals:**
- Tests for findings owned by other tickets; those tickets add their own.

## 3. Wishes and constraints
### 3.2 Hard constraints
- **Platform / versions:** the tests must run on Windows, and on Linux where the package is not Windows-only.
- **Localization:** new validation messages in 13 languages.

### 3.3 Owner inputs (Approval gate)
- **Related tickets:** all tickets in this audit; this one should land early, because the new pipeline tests make the others safer to change.
- **Validation level:** `scripts/check.ps1` green; staticcheck clean for `GOOS=windows` and `GOOS=linux`.

## 4. Current architecture context
Flag parsing uses the standard library with post-parse checks for a subset of flags. Logging
writes to the console first and to the run log under a lock. Log and report names are timestamps.
The pipeline's run method is large and has been exercised only through smoke tests.

## 5. Proposed approach
### 5.1 Pillars / modules
- **Sentinel errors:** for version and help.
- **Central flag validation:** with user-visible adjustment notices.
- **Single locked write path:** for console and log.
- **Collision-free names:** a timestamp plus a process-unique suffix; a per-run log cap.
- **Static analysis to zero:** the unused function moves behind the platform split, and error-string and escape fixes.
- **Test scaffolding:** a filesystem sandbox harness for pipeline runs (success, failure cleanup, reuse, force) that later tickets extend.

## 6. Open questions / research items
No open questions.

## 7. Risks
- **Stricter flag validation rejects values existing scripts pass.** Likelihood: low. Impact: the scripts fail. Mitigation: reject only values that were already meaningless (negative sizes).

## 8. User impact (docs)
No changes to user docs.

## 9. Architecture decisions (ADR)
No ADRs. The decision follows established project patterns.

## 10. Links to other specs
All tickets under this audit.

## 11. Done criteria (strategic)
1. `staticcheck ./...` is clean for Windows and Linux.
2. `-split -5` is rejected with a message.
3. Two runs started in the same second write two separate logs.
4. The pipeline test harness covers success, failure cleanup, reuse and `-force`.

## 12. Next step
`/spec-tech 25_2026-09-24_chore-hygiene-and-test-gaps`
