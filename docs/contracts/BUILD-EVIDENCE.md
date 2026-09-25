# Pointer: BUILD-EVIDENCE

- **Id:** `BUILD-EVIDENCE`
- **Version:** 0.9 (draft)
- **Home:** the shared contracts catalog, `automated-checks/README.md` section 5 (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer
- **Owner:** FastMediaSorter Android

When a check result is evidence about the thing actually shipped: the check names its subject, the
artifact carries its own version, no test is retried to green, and a hand-kept twin ships with the check
that proves it agrees.

**Why it matters here.** The release binaries are rebuilt in CI from the tag, and the tag workflow runs no
test. The local gate runs on a local build. What binds the two is this repo's own mechanism below.

**What this repo owes it**

- **Subject banner (rule 1).** Every check prints `<check>: subject = ..` before anything else - for the
  test runs, the edition (Go or extension), `GOOS/GOARCH`, the Go or Node version and the tree.
- **Artifact version (rule 2).** [`../../scripts/verify-exe-version.ps1`](../../scripts/verify-exe-version.ps1)
  reads the stamp out of the built exe (the version resource, the linked `-X main.Version` bytes, and the
  CLI's own `-version`) and fails on the `dev` default or a mismatch. It runs inside `build.ps1` and
  `build-ui.ps1`, and [`../../scripts/release.ps1`](../../scripts/release.ps1) prints it as a free step
  against the downloaded release assets.
- **Tested tree = tagged tree.** [`../../scripts/check.ps1`](../../scripts/check.ps1) writes
  `temp/logs/gate-evidence.json` with the verdict, the plan, every child's verdict and the git tree hash
  of what it read, hashed before and after the run (a mismatch voids it); `release.ps1` shows the tag step
  as BLOCKED unless it is the full default plan with every child passing, the tree is HEAD's (bar
  `DEV/COMMIT_LOG.md`, appended after the build commit), and the working tree is clean. Pinned by
  `tests/verdict_scripts_test.go` (`TestGateEvidenceBindsTheRelease`).
- **No retry (rule 3).** No test here is re-run to green; the Go toolchain's arch is pinned instead.
- **Twin and check ship together (rule 7).** `configs/parity-map.json` (read by `parity-check.ps1`) is
  compared with the port map in [`../PARITY.md`](../PARITY.md) by
  [`../../tests/parity_map_test.go`](../../tests/parity_map_test.go).

**Conformance.** No vectors in the catalog. The tracked `build/*.exe` are version-checked but not proven to
be built from the committed source; that gap is a dated exception in the catalog's registry.
