# Pointer: CHECK-VERDICT

- **Id:** `CHECK-VERDICT`
- **Version:** 0.14 draft
- **Home:** the shared contracts catalog, `automated-checks/README.md` section 2 (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, `partial` - this repo's checks produce verdicts in the contract's vocabulary, and its own
  aggregator and release checklist read them; read at 0.13 on 2026-10-20 (ticket 120)
- **Owner:** FastMediaSorter Android

What a check hands to whoever reads its result: four exit codes (`0` pass, `1` fail, `2` could not
verify, `3` advisory), one machine-readable verdict line at the end of every run, advisories named rather
than counted, and no stopping at the first failure. `4` is reserved as a non-verdict (not looked at yet - a
shared lock held by another run); an aggregator maps a child's `4` to `2`, and a child that reports
`NOT APPLICABLE` counts as skipped (0.12, 0.13). Only `PASS` not followed by `WITH ADVISORIES` is clean.

**Scope here.** The gate and check scripts under [`../../scripts/`](../../scripts/) and every tool that prints
a verdict line: the judging subcommands of `tools/ocrlab` (`verify`, `gate`), `tools/auditstand`,
`tools/doccorpus`, `tools/helpcheck` (through `scripts/help-check.ps1`) and `tools/uicheck/run.mjs`. **Not** the
app's own CLI: `doc-html-translate.exe` exits `0`-`4` under [`OCR-INVOCATION`](OCR-INVOCATION.md), a different
boundary with its own published codes.

**What this repo owes it**

- Every check script dot-sources [`../../scripts/lib/verdict.ps1`](../../scripts/lib/verdict.ps1) and
  ends in `<check>: PASS | PASS WITH ADVISORIES (..) | FAIL (..) | COULD NOT VERIFY (..)`, on the failure
  path too. A missing tool, input or git result is `2`, never `1` and never `0`.
- A run in which no test ran is not a pass: [`../../scripts/test.ps1`](../../scripts/test.ps1) ends
  `COULD NOT VERIFY (no test ran)`, and `scripts/test-extension.ps1` counts executed tests (`run=` excludes the
  skipped ones, which it names above the line) (0.13, 2026-10-20).
- Never `Write-Error` before an `exit N` under `$ErrorActionPreference = 'Stop'` (rule 3).
- [`../../scripts/check.ps1`](../../scripts/check.ps1) runs every child, quotes each child's own verdict
  line, and ends in one line of its own: any `FAIL` wins, then any `COULD NOT VERIFY`, then advisories.
- A Go test whose skip means a declared input of the suite is absent starts its skip message with
  `input absent:`; `scripts/test.ps1` turns that into `2`. Any other skip is counted as `skipped=n` on the
  verdict line and named above it.
- Adding a fifth exit code is a breaking change of the contract, not a local decision.

**Open deviations, in the catalog registry (until 2026-12-31):** rule 8 - `scripts/parity-check.ps1` derives its
change set from git; rule 2 - the skip counting described above (the audit finds it now matches the text of
2.13, and closing the row is left to the owner). Read, not driven: the thrown-exception path (rule 11), a child
exit `4`, a `NOT APPLICABLE` child.

**Conformance.** No vectors in the catalog yet. This repo drives its own scripts to every documented outcome:
[`../../tests/verdict_scripts_test.go`](../../tests/verdict_scripts_test.go) (`check.ps1`, `parity-check.ps1`,
`test.ps1`) and [`../../tools/ocrlab/report/gate_test.go`](../../tools/ocrlab/report/gate_test.go) (`ocrlab gate`).

**Re-read 2026-10-29 (ticket E0126) at 0.14.** Item A: `tools/auditstand -SelfTest` now prints `run=N skipped=M` (A1, each of its three checks one test); the test runners already printed A2-A4's forms; `scripts/test.ps1`'s input-absent line keeps the bare count A4 allows (no group is left unrun). Rules 8 and 2 stay dated exceptions.
