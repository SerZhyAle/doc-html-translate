# Pointer: RELEASE-LEDGER

- **Id:** `RELEASE-LEDGER`
- **Version:** 0.2 draft
- **Home:** the shared contracts catalog, `release-ledger/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** producer - maintains this repository's release ledger under `DEV/releases/` (`paths.releasesDir` in `.sza-profile.json`) via the canon's `release-ledger.ps1` writer (`scripts/release-ledger.ps1`, `a rs`)
- **Wire carrier:** `schema` (`"1"`) in `DEV/releases/channels.json` and `DEV/releases/builds.json`; `DEV/releases/history.jsonl` rows carry none (append-only)

**What this repo owes it**

- `DEV/releases/channels.json` declares both independent packages (`app`, `extension`) and all six channels listed in `.sza-canon.json` (`github`, `winget`, `msstore`, `installer`, `chrome`, `edge`) with their public identities (rules 1-3).
- Every local build script (`scripts/build.ps1`, `scripts/build-ui.ps1`, `scripts/build-installer.ps1`, `scripts/build-extension.ps1`, `msix/build-msix.ps1`) calls `release-ledger build` via `scripts/lib/release-ledger.ps1` as its last step and reports without failing the build on a ledger error (rule 4).
- `DEV/releases/history.jsonl` is append-only (written by `release-ledger` `import`, `probe`, `publish`, `confirm`), and `DEV/releases/RELEASES.md` is a render target never edited by hand (rules 5, 10).
- No secret, credential, or local machine path enters any ledger file (rule 9).

**Conformance.** Backfilled with `import`, `confirm` (`installer` and migrated in-flight `edge` state), and read-only `probe` on 2026-11-18 (`E0157`); `tools/store/test-store-submission.ps1` (`cases=14 failed=0`) exercises `msstore` `publish` calls against stubs; the build-script hooks are proved at the next real build.

Release re-verification: 2026-11-27: 0.2 read - no Play channel, so rule 12 skips nothing; writer is the canon plugin 2026.1126.9. channels.json covers the 6 stamp channels, 44 events parse, RELEASES.md equals a fresh render. Commit DEV/releases with the release (rule 1).
