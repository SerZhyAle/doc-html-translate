---
paths:
  - "scripts/**"
  - "configs/**"
  - "tests/**"
  - "a.ps1"
  - "build.ps1"
  - ".github/workflows/**"
---

# Checks and gates

Moved verbatim from `AGENTS.md` / `CLAUDE.md` (2026-11-25); the short core there still binds. A section
name quoted below is a section of `AGENTS.md` unless it says otherwise.

## Commands (from CLAUDE.md)

From the repository root, in PowerShell. Between releases only what takes 5 s or less is run; the rule and the
measured list are in AGENTS.md ("Build, Test, Lint").

- Fast gate: `./scripts/check.ps1 -Fast`; for the touched area only, `-Fast -Plan scripts/<child>.ps1,..`.
- One Go test: `go test -tags nodynamic ./internal/<pkg>/ -run <TestName>`. The cross-package suite is the
  `tests/` package: `go test -tags nodynamic ./tests/ -run <TestName>`.
- One extension test: `node --test extension/test/<file>.test.mjs`.
- Vet, lint, spelling: `go vet -tags nodynamic ./internal/<pkg>/`, `./scripts/lint.ps1`, `./scripts/typo.ps1`.
- Release tier, not run between releases: `./scripts/test.ps1` (the whole Go suite), `./scripts/test-extension.ps1`,
  `./scripts/check.ps1` with no switch, `./scripts/build-local.ps1 -Release`.
- `./a.ps1 <alias>` is the launcher over the scripts (`fc` fast gate, `l` lint, `ty` spelling, `cli` / `ui` one
  build each); the header of [a.ps1](../../a.ps1) lists every alias.

## Fast and release commands, measured list, script list

- **Fast command (between releases):** `./scripts/check.ps1 -Fast` - the gate children recorded
  `"tier": "fast"` in `configs/check-placement.jsonl`, all at once (about 5 s warm; `a fc`). For the touched
  area name its children: `./scripts/check.ps1 -Fast -Plan scripts/help-check.ps1,scripts/settings-inventory.ps1`
  (a release-tier child is refused, exit 2). Go code: `go test -tags nodynamic ./internal/<pkg>/ -run <Test>`
  of the touched package, or `go vet -tags nodynamic ./internal/<pkg>/`; extension code:
  `node --test extension/test/<file>.test.mjs` - each a direct call whose run takes 5 s or less.
- **Release command (the release pre-flight only):** `./scripts/build-local.ps1 -Release -Message ".."` - both
  builds, then the full gate (`./scripts/check.ps1` with no switch), whose evidence `release.ps1` requires.
- **Measured list:** each gate record's `tier`, `seconds` and `timed` (date and how it was measured) in
  `configs/check-placement.jsonl`, held by `tests/placement_test.go` (a fast record carries at most 5 s).
  Release tier on 2026-10-29: `scripts/test.ps1` (212 s) and `scripts/test-extension.ps1` (20 s); every other
  gate child is fast (0.6 - 4.9 s). `check.ps1` prints `<child> took N s` for every child, so each run is a new record.
- Local build + commit (the "сборка" flow): ./scripts/build-local.ps1 -Message "<commit message>" (fast tier; `-Release` for the full gate)
- Build CLI only: ./scripts/build.ps1
- Build UI only: ./scripts/build-ui.ps1
- Build universal installer (setup.exe, x86+x64, per-user, local/free): ./scripts/build-installer.ps1 (needs Inno Setup / ISCC)
- Test (release tier): ./scripts/test.ps1 (Go edition) and ./scripts/test-extension.ps1 (extension edition, node --test)
- Lint: ./scripts/lint.ps1
- Full local checks (release tier): ./scripts/check.ps1

## Verdicts

- Every check speaks `CHECK-VERDICT` (pointer: docs/contracts/CHECK-VERDICT.md): exit `0` PASS, `1` FAIL,
  `2` COULD NOT VERIFY (a missing tool, input or git result - **never a pass**), `3` PASS WITH ADVISORIES,
  and the **last line** is the verdict (`check: PASS`, `test: COULD NOT VERIFY (1)`, ..). Quote that line
  as evidence, not the scrollback above it. A fresh clone without `test_doc/` ends in COULD NOT VERIFY
  by design. Which runner owns each check: `configs/check-placement.jsonl` (a new check gets a record).

## Gate evidence

- `check.ps1` writes `temp/logs/gate-evidence.json` (verdict, plan, every child's verdict, and the tree
  hash taken before and after the run - an edit during the run voids it); `release.ps1` blocks the tag
  step unless it is the full default plan, every child passing, on HEAD's tree (only `DEV/COMMIT_LOG.md`,
  appended by build-local after its commit, may differ). A `-Plan` subset never unlocks it, and a `-Fast`
  run writes no evidence at all.

## Cross-edition parity checks (from CLAUDE.md)

Two independent codebases that share no code: the Go app (CLI/GUI/MSIX) and the JS browser extension
(`extension/src`). Logic is hand-ported Go -> JS, so it drifts. The port map, the invariants that must
match and the intentional differences live in [docs/PARITY.md](../../docs/PARITY.md), with the process rules in
[AGENTS.md](../../AGENTS.md) ("Cross-Edition Parity") - read both before adding a user-facing feature.
`scripts/parity-check.ps1` is the advisory structural gate; `tests/parity_test.go` hard-fails on value drift.

## Sliced code audit

- Sliced code audit: ./scripts/audit-slices.ps1 cuts the audit scope (shipped code + release path) into
  slices a session can read line by line (`-Write` writes the manifest into the audit ticket's folder,
  `-Tail` slices files added since, `-Summary` measures the campaign and exits 0 only when it is closed).
  Hand-run; ticket 34 is its first campaign.

## Tool bootstrap

Tool bootstrap (when missing):

- ./scripts/bootstrap-tools.ps1

Notes:

- scripts/lint.ps1 expects golangci-lint.
- scripts/typo.ps1 expects typos (typos-cli via cargo).
