# Pointer: CHECK-PLACEMENT

- **Id:** `CHECK-PLACEMENT`
- **Version:** 0.10 (draft)
- **Home:** the shared contracts catalog, `automated-checks/README.md` section 4 (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - this repo keeps its own placement registry in the contract's shape
- **Owner:** FastMediaSorter Android

Every check declares which runner owns it, one JSON Lines record per check, and the declaration is
compared with the wiring in both directions.

**This repo's registry:** [`../../configs/check-placement.jsonl`](../../configs/check-placement.jsonl).
Classes used here: `gate` (run by [`scripts/check.ps1`](../../scripts/check.ps1)), `build` (run by the build
scripts on the artifact they just produced), `release` (run by [`scripts/release.ps1`](../../scripts/release.ps1)
before the tag step, because its input - the contracts catalog - exists only where a release is cut),
`hand-run` (a named human or agent step), `none`.

**Declared: there is no per-change runner.** Nothing runs on every change - no pre-commit hook, no CI test
job. The `gate` class is an operator-typed batch: `scripts/check.ps1`, reached through
`scripts/build-local.ps1` and the `/build` agent command. The reasons are a solo developer and paid CI;
the contract's rule 3 says such a batch does not satisfy a per-change class, and that gap is a dated
exception in the catalog's registry rather than a silent one.

**What this repo owes it**

- A new check gets a record in the same change that adds it, with a `reason`; a check that moves between
  classes gets its record edited with a new `decided` date and a reason (rule 5).
- [`../../tests/placement_test.go`](../../tests/placement_test.go) fails when a `gate` record is not in
  `check.ps1`'s plan, when the plan runs an unrecorded check, when a `build` record is not called by the
  build scripts it names, when a `release` record is not called by `scripts/release.ps1`, or when a
  script that reports a `CHECK-VERDICT` verdict has no record.
- A `seeded` record names the open ticket that owns judging it (rule 7). Every record here is `judged`.

**Conformance.** No vectors in the catalog. The registry is read by this repo's own test only; no second
program reads it yet.
