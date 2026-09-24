# A check says what it could not verify, and the shipped build carries its own evidence

**Status:** Implemented (2026-09-24; not committed yet - see "Resolution")
**Priority:** 55
**Date:** 2026-09-23

> Contract sync ticket, both directions.
> Contracts: `CHECK-VERDICT`, `CHECK-BASELINE`, `CHECK-PLACEMENT`, `BUILD-EVIDENCE`, all 0.9 (domain
> `automated-checks/`, owner FastMediaSorter Android, drafts). Pointers now in
> [`docs/contracts/`](../../../docs/contracts/).

## What / why

The domain says what an automated check reports to whoever reads it: an exit-code vocabulary that
includes "could not verify", one verdict line, a baseline file for accepted debt, a registry saying which
runner owns each check, and when a build or test result is evidence about the artifact actually shipped.

The contract names this product as a **candidate** consumer: it runs its own checks, and a candidate
becomes bound only by writing its own adoption row. This ticket decides to adopt and closes the gaps.

The gaps matter here for a concrete reason: a fresh clone runs the gate, the full-corpus integration test
skips because `test_doc/` is absent, `test.ps1` prints "Tests passed", `check.ps1` prints "All checks
passed" - and the release is then built in CI from the tag with **no test step at all**. Every word of
that green is true and none of it is evidence about the binary users download.

Found by reading the scripts and workflows on 2026-09-23; no script was driven to each outcome.

## Findings that drive the work

**`CHECK-VERDICT`:**
- only exit codes 0 and 1 exist; a missing tool (`golangci-lint`, `typos`, headless Chrome for
  `verify-html.ps1`) reports "failed" instead of "could not verify" (rules 1, 2);
- "could not verify" becomes a pass in three places: Go `t.Skip` of the corpus test, `parity-check.ps1`
  on an empty or undeterminable change set (`git diff` result unchecked), and `ocrlab gate` on an absent
  category or holdout (`tools/ocrlab/report/gate.go` sets `Pass: true` while its own text says "not a
  pass, an absence");
- no machine-readable verdict line; the failure path ends in an exception, not a line (rule 5);
- `check.ps1` stops at the first failing child and then prints "All checks passed" over parity
  advisories (rules 6, 9);
- held: every documented code is reachable; every non-zero carries a printed reason (rules 3, 4).
- The closest to conformance already: `ocrlab gate` (exit 2 on usage, verdict kept as data, subject
  banner) and `verify-html.ps1`'s closing summary line.

**`CHECK-BASELINE`:** mostly not applicable - no accepted-findings file exists. `DEV/ocrlab/thresholds.json`
is a third shape (metric bounds with provenance) the contract does not name (B5).

**`CHECK-PLACEMENT`:** no placement registry; every check lives in an operator-typed batch; no pre-commit
hook, no CI test step; `verify-html`, `ocrlab` and the extension's `node --test` are outside even the
batch; the extension tests cannot run on the dev machine (no Node).

**`BUILD-EVIDENCE`:**
- the local gate runs on a `Get-Date`-stamped local build; the release binaries are rebuilt in CI from
  the tag without tests; the extension's CI build excludes its tests (rule 2);
- nothing asserts that a built exe's embedded version is not the `"dev"` default and equals the tag;
  `build/*.exe` is tracked and hand-restamped;
- the parity port map in `parity-check.ps1` is a hand mirror of [`docs/PARITY.md`](../../../docs/PARITY.md)
  with no compare check (rule 7);
- no subject banner says which edition and architecture a run judged (rule 1);
- held: no retry of flaky tests - the architecture is pinned instead (rule 3).

## Direction A - the repo conforms

1. **Exit codes** - missing tool, missing input or git failure exits 2 in `lint.ps1`, `typo.ps1`,
   `verify-html.ps1`, `parity-check.ps1` and `ocrlab`; parity drift is an advisory (3) rather than a
   silent 0.
2. **`test.ps1`** surfaces Go skips (`go test -json`, count SKIP); a skip that covers a declared input is
   not a clean PASS.
3. **`check.ps1`** runs every child, collects codes, names advisories, and ends with one line:
   `check: PASS | PASS WITH ADVISORIES (n) | FAIL (n) | COULD NOT VERIFY (n)` - on the failure path too.
4. **`ocrlab gate`** - absence is not a pass; its last line names the gate.
5. **Placement** - one JSON Lines record per check (go test, lint, typos, parity-check, verify-html,
   ocrlab verify/gate, extension tests, canon compliance) with its runner class and reason; either a
   declared per-change runner or a declared "none" with the reason (solo developer, paid CI).
6. **Build evidence** - the release is tied to tested code: a test step in the tag workflow, or a recorded
   gate commit equal to the tag commit (owner's choice, open question 2); a post-build assertion that the
   embedded version equals the stamp and is never `dev`; a decision on tracked `build/*.exe`.
7. **Subject banner** - test and check print edition and GOARCH ("Go edition, amd64; extension not
   tested").
8. A compare check between the parity port map and `docs/PARITY.md`.
9. Tests that drive `check.ps1` and `parity-check.ps1` to each outcome (the contract's rung 1).

## Direction B - what the contracts need from this product

Filed as `PROPOSAL-2026-09-23-<topic>.md` in the catalog's `automated-checks/` folder.

- **B1** `CHECK-VERDICT` [NEW]: a wrapper over a runner that knows only pass/fail (Go `t.Skip` prints `ok`)
  must surface skips; suggest a `skipped=n` field on the verdict line.
- **B2** [clarify]: for diff-based checks, "the change set is empty" (vacuous pass, `0 inspected`) differs
  from "the change set could not be determined" (2). Read literally, rule 2.1 makes every clean-tree run
  exit 2.
- **B3** [clarify]: the verdict line is owed on the failure path too; state how an aggregator's line relates
  to its children's.
- **B4** `CHECK-PLACEMENT` [NEW]: "no per-change runner" as an explicit declared state with a reason; an
  "agent closure" runner class (`/build`, `/fix`).
- **B5** `CHECK-BASELINE`: metric acceptance-bound files are a third shape, or out of scope - say which.
- **B6** `BUILD-EVIDENCE` [NEW]: when CI rebuilds the shipped binary from a tag and tests ran on a local
  build, state what binds them (same commit + reproducible build, or tests in the publishing job).
- **B7** name "edition" (native app vs browser extension) as an axis-of-confusion example.
- **B8** say that the telemetry rules apply where telemetry exists, and what a product without it owes
  for a prose runtime figure.

## Done criteria

- [x] Pointer files for the four ids, listed in the pointer README - `docs/contracts/CHECK-VERDICT.md`,
      `CHECK-BASELINE.md`, `CHECK-PLACEMENT.md`, `BUILD-EVIDENCE.md`.
- [x] Registry: this product's consumer row; every deviation still open is a dated exception - four
      adoption rows and five exceptions (until 2026-12-31), written 2026-09-24.
- [x] Removing `typos` from PATH makes `typo.ps1` and `check.ps1` report "could not verify", exit 2 -
      `typo: COULD NOT VERIFY (typos absent)` exit 2; `check: COULD NOT VERIFY (1: typo)` exit 2.
- [x] A fresh clone without `test_doc/` does not produce a clean PASS line - with `DOC_HTML_TEST_DOC`
      pointed at a missing folder: `test: COULD NOT VERIFY (1)`, exit 2.
- [x] Every gate script ends with one verdict line, success and failure alike - via
      `scripts/lib/verdict.ps1`; driven to every outcome by `tests/verdict_scripts_test.go` and
      `tools/ocrlab/report/gate_test.go`.
- [x] The placement registry exists and every check in `scripts/` and `tools/ocrlab` has a record -
      `configs/check-placement.jsonl`, held by `tests/placement_test.go`.
- [x] The release artifact's version is asserted after build, and the evidence for a release names the
      commit it was taken on - `scripts/verify-exe-version.ps1` in both build scripts and as a release
      step; `temp/logs/gate-evidence.json` names HEAD and the tree hash, `release.ps1` blocks on a mismatch.
      The binding is the tree hash rather than the commit, because `build-local.ps1` commits after the gate.
- [x] B1-B8 filed or withdrawn in writing here - all eight filed 2026-09-24 in the catalog's
      `automated-checks/` folder as seven `PROPOSAL-2026-09-24-*.md` files (B7 and B8 share one).

## Open questions - answered by the owner 2026-09-24

1. Adopt now - done.
2. Build evidence: the free local binding (gate tree hash = tagged tree), not a test step in the paid tag
   workflow. `.github/workflows/` is untouched.
3. Extension tests: the premise was wrong - Node v24.16.0 is installed and the suite passes locally
   (149/149). Owner chose to put them in the gate: `scripts/test-extension.ps1` in `check.ps1`.
4. `build/*.exe` stay tracked; they get the version check, and the missing source-freshness proof is a
   dated exception.

## Resolution

- `scripts/lib/verdict.ps1` - the exit-code vocabulary, the verdict line, the subject banner.
- `test.ps1` (`go test -json`, skips counted, `input absent:` -> 2), `test-extension.ps1` (new), `lint.ps1`,
  `typo.ps1`, `verify-html.ps1`, `parity-check.ps1` (map moved to `configs/parity-map.json`; drift = 3,
  git failure = 2) and `check.ps1` (runs every child, one line, gate evidence) all speak it.
  `build-local.ps1` stops on 1 and 2.
- `ocrlab`: `gate` treats an unjudged bound as absent, not a pass; input and usage errors exit 2;
  `verify` ends in a verdict line and says COULD NOT VERIFY when only the media root is missing.
- `verify-exe-version.ps1` (new) in `build.ps1` / `build-ui.ps1`; `release.ps1` prints gate evidence and
  the post-release asset check.
- `tests/parity_map_test.go` found three watched files `docs/PARITY.md` did not pair; two port map rows
  were added to the doc, and the map now watches `ocr-plates.js`, `ocr-cluster.js` and `ui.html` too.
- Full gate on 2026-09-24: `check: PASS`, exit 0 - `test: PASS (run=516 skipped=4)`,
  `test-extension: PASS (run=149 skipped=0)`, `lint: PASS`, `typo: PASS`, `parity-check: PASS`.
- Not committed: the owner asks for a commit (or runs `/build`).
