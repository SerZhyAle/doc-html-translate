# The release gate can be turned green without the full gate having passed

**Status:** In Progress - all six goals implemented and pinned; acceptance 3 (a green gate) waits on ticket 43 (X25)
**Priority:** 90
**Date:** 2026-09-26

> Filed by the pre-release audit, [ticket 34](done/34_2026-09-25_full-code-audit-pre-release.md). Finding ids
> refer to its register, [`FINDINGS.md`](done/34_2026-09-25_full-code-audit-pre-release/FINDINGS.md).

## 1. Problem

`scripts/release.ps1` blocks the tag step unless `temp/logs/gate-evidence.json` says the gate passed on
HEAD's tree. That evidence is the only link between the tested tree and the shipped binaries, and it is
weaker than it looks:

- **R1 (high)** - `check.ps1 -Plan <one child>` writes release-grade evidence: code 0 and the real tree
  hash, but not which checks ran. `release.ps1` reads only the code and the tree, so a lint-only run
  unlocks the tag line.
- **R3** - the tree is hashed after the children finish, so an edit made while the gate runs is recorded
  as tested.
- **R2** - the documented `build-local.ps1` flow can never produce matching evidence: it rewrites the
  tracked `build/doc-html-translate.exe` and amends the commit with `DEV/COMMIT_LOG.md` after the check,
  so HEAD's tree never equals the checked one and the BLOCKED hint loops, which invites a bypass.
- **R5** - the typo gate FAILs at HEAD (non-English i18n, RTF and translator files are not excluded in
  `configs/.typos.toml`), so `check.ps1` cannot pass at all right now.
- **R12** - `contract-gate.ps1` reports PASS after checking zero contracts.
- **R4** - `commit-push.ps1` says it stays local and free, but pushes `main` with no gate, which
  publishes the Pages site.

## 2. Goals

1. Evidence records the plan and every child's verdict; `release.ps1` accepts only the full default
   plan, every child PASS or PASS WITH ADVISORIES.
2. The tree is hashed before and after the run; a mismatch voids the evidence.
3. The documented local flow ends with evidence that matches HEAD, or the docs name the one command
   that produces it.
4. The gate is green on a clean HEAD - the typos config fixed, not the check weakened.
5. A contract gate that checked nothing is COULD NOT VERIFY, not PASS.
6. `commit-push.ps1` either says that a push to `main` publishes the site and asks, or refuses it.

## 3. Constraints

- CHECK-VERDICT stays the contract: exit codes 0/1/2/3 and the verdict as the last line.
- No release, tag or push is run to prove any of this; `tests/verdict_scripts_test.go` drives the
  scripts over scratch trees, as it already does.

## 4. Acceptance

- A `-Plan` subset run leaves evidence that `release.ps1` rejects, pinned by a scratch-tree test.
- An edit during the gate run voids the evidence, pinned by a test.
- `./scripts/check.ps1` ends in PASS or PASS WITH ADVISORIES on a clean HEAD, output cited.

## 5. Implementation (2026-09-26)

- **R1** - `check.ps1` records `plan`, `fullPlan` and every child's verdict. `release.ps1` derives the full
  plan from `configs/check-placement.jsonl` (class `gate`, runner `scripts/check.ps1`), not from the
  evidence's own claim, and blocks a partial plan or any child outside 0/3.
- **R3** - the tree is hashed before the first child and after the last; on a mismatch `tree` is empty
  and the verdict is COULD NOT VERIFY (a FAIL stays a FAIL).
- **R2** - `build-local.ps1` now builds both exes first, then runs the gate, then commits, so the gate
  reads the commit's tree. `DEV/COMMIT_LOG.md`, appended after the commit, is the one path
  `release.ps1` lets differ between the gated tree and HEAD's.
- **R5** - `configs/.typos.toml`: the remaining i18n catalogs and the legacy-encoding fixtures excluded
  by name, RTF control words and `flate` / `unparseable` accepted, the escaped-`café` fixtures and two
  verbatim upstream strings ignored by exact pattern; two two-letter Go identifiers renamed (`urlErr`, `styleNames`).
- **R12** - `contract-gate.ps1` ends in COULD NOT VERIFY (exit 2) when it checked zero contracts.
- **R4** - `commit-push.ps1` refuses to push `main` without `-PublishSite`, before anything is staged.

Evidence:

- `go test ./tests/ -run 'TestGateEvidenceBindsTheRelease|TestCheckAggregatorOutcomes|TestContractGateOutcomes|TestPlacement' -count=1`
  -> `ok doc-html-translate/tests 23.245s`. `TestGateEvidenceBindsTheRelease` pins acceptance 1 and 2
  (subset rejected, edit during the run voids, COMMIT_LOG-only difference accepted, any other rejected).
- `./scripts/typo.ps1` -> `typo: PASS` (exit 0).
- `./scripts/check.ps1` on the working tree -> `check: FAIL (1: test)`, every other child PASS. The one
  failure is `TestExtract_Volume3ImagesOnTheirPages` ("spine has 4 pages, want 10"), red on a clean HEAD
  worktree too: finding X25, owned by [ticket 43](43_2026-09-26_bugfix-pdf-text-and-image-fidelity.md).
  Acceptance 3 is re-run once 43 lands.
