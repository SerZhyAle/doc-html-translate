# Pointer: CHECK-VERDICT

- **Id:** `CHECK-VERDICT`
- **Version:** 0.9 (draft)
- **Home:** the shared contracts catalog, `automated-checks/README.md` section 2 (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - this repo's checks produce verdicts in the contract's vocabulary, and its own
  aggregator and release checklist read them
- **Owner:** FastMediaSorter Android

What a check hands to whoever reads its result: four exit codes (`0` pass, `1` fail, `2` could not
verify, `3` advisory), one machine-readable verdict line at the end of every run, advisories named rather
than counted, and no stopping at the first failure.

**Scope here.** The gate and check scripts under [`../../scripts/`](../../scripts/) and the judging
subcommands of `tools/ocrlab` (`verify`, `gate`). **Not** the app's own CLI: `doc-html-translate.exe`
exits `0`-`4` under [`OCR-INVOCATION`](OCR-INVOCATION.md), which is a different boundary with its own
published codes.

**What this repo owes it**

- Every check script dot-sources [`../../scripts/lib/verdict.ps1`](../../scripts/lib/verdict.ps1) and
  ends in `<check>: PASS | PASS WITH ADVISORIES (..) | FAIL (..) | COULD NOT VERIFY (..)`, on the failure
  path too. A missing tool, input or git result is `2`, never `1` and never `0`.
- Never `Write-Error` before an `exit N` under `$ErrorActionPreference = 'Stop'` (rule 3).
- [`../../scripts/check.ps1`](../../scripts/check.ps1) runs every child, quotes each child's own verdict
  line, and ends in one line of its own: any `FAIL` wins, then any `COULD NOT VERIFY`, then advisories.
- A Go test whose skip means a declared input of the suite is absent starts its skip message with
  `input absent:`; [`../../scripts/test.ps1`](../../scripts/test.ps1) turns that into `2`. Any other skip is
  counted as `skipped=n` on the verdict line and named above it.
- Adding a fifth exit code is a breaking change of the contract, not a local decision.

**Conformance.** No vectors in the catalog yet. This repo drives its own scripts to every documented
outcome: [`../../tests/verdict_scripts_test.go`](../../tests/verdict_scripts_test.go) (`check.ps1`,
`parity-check.ps1`) and [`../../tools/ocrlab/report/gate_test.go`](../../tools/ocrlab/report/gate_test.go)
(`ocrlab gate`). Open deviations are dated exceptions in the catalog's registry.
