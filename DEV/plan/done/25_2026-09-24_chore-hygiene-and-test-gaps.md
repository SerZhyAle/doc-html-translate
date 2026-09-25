# Strategic spec: 25_2026-09-24_chore-hygiene-and-test-gaps - Small correctness fixes and missing tests

**Ticket:** 25_2026-09-24_chore-hygiene-and-test-gaps
**Status:** BlockNeedUserTest - implemented and covered by tests; the Windows-only tests (registration, dialog, browser open, bundled pdftotext) compile on Linux and run only on Windows, and `scripts/check.ps1` needs a Windows pass.
**Priority:** 40
**Date:** 2026-09-24
**Tier:** Quick Win
**Tactical plan:** none - implemented directly from this spec (Quick Win tier).
**Findings:** P18 P19 P21 P22 Q1-Q5, plus the test gaps listed in §1 (see the [findings register](../../research/audit_2026-09-24/README.md))

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

## Resolution

Implemented 2026-09-25 without a tactical plan. Findings, then tests.

- **P18:** `config.ErrVersion` is a sentinel; `main` matches it with `errors.Is`.
- **P19:** `ParseArgs` refuses a negative `-split`, `-toc-depth`, `-ollama-ctx` or `-ollama-parallel`.
  A value below the floor (`-ollama-parallel 0`, `-ollama-ctx` under 512) is raised and reported:
  `Config.Notices`, printed by `app.Run` before the pipeline starts. Both messages follow `-ui-lang`,
  13 languages in `i18n_cli.go`. The translator keeps its own clamps as a second line.
- **P21:** `logging.emit` takes the lock before the console write, not after it.
- **P22:**
  - Run logs are `run-<yyyymmdd-hhmmss>-<pid>.log`, so same-second runs get separate files.
  - `report.CapRunLog` stops a run's log at `MaxRunLogBytes` (5 MiB) with one marker line.
  - Report archives carry seconds, and a taken name gets `-2`, `-3`.. instead of being overwritten.
- **Q1:** `normalizeTarget` moved to `browser_windows.go`, its only caller's side.
- **Q2:** already clean when this ran.
- **Q3:** the raw U+200F in `i18n_cli.go` is now the `\u200f` escape.
- **Q4/Q5:** error-string style in `mobi/extract.go` and `tools/ocrlab/cmd_add.go`.
- **Test seams** (behaviour unchanged): the registry calls in `windowsreg`, `MessageBoxW` in `dialog` and
  the embedded pdftotext set in `bundledtools` are indirected, like `shellExecute` in `browser`.
- **Tests:**
  - pipeline sandbox harness (`internal/pipeline/harness_test.go`) and runs over it covering success,
    failure cleanup (first run and rebuild), reuse vs rebuild triggers, and `-force`;
  - document splitting (`htmlsplit/split_flat_test.go`), Markdown input (`md/sections_test.go`);
  - registration, dialog, browser open, helper extraction - Windows sides with fakes, plus non-Windows
    sides;
  - flag validation, log serialization, run-log names and cap, report names;
  - extension viewer, background and page-OCR (`extension/test/{viewer,background,page-ocr}.test.mjs`,
    17 tests, no `src/` change; `vendor/` imports are stubbed so the tests do not need a vendored build).
- **Found, not fixed here:** a refused page-OCR host frame leaves its wait and timer pending - added to
  ticket 18 (`lifecycle-leaks`), which owns that code.
- **Extension:** `npm test` 215/216; the one failure is `ebook.test.mjs`, because `vendor/foliate` is
  absent here (the vendor step stops on a tessdata download refused by the network), not this change.
- **Checks here (Linux):** `go test ./...`, `go vet` and `staticcheck ./...` for both `GOOS=linux` and
  `GOOS=windows` are clean; every Windows-only test package builds with `GOOS=windows go test -c`.
  `golangci-lint` was not run: the container has v2 and `configs/.golangci.yml` is v1.
- **Left for Windows:** run `scripts/check.ps1`; it executes the Windows-only tests above.
