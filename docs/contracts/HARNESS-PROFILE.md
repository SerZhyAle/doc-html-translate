# Pointer: HARNESS-PROFILE

- **Id:** `HARNESS-PROFILE`
- **Version:** 0.14 draft
- **Home:** the shared contracts catalog, `rule-adoption/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - the canon's shipped harness (ticket ledger, leases, portfolio runner, probe gates) runs against this repository
- **Wire carrier:** `.sza-profile.json` at the repository root, `"version": "1"` (wire 1.0)

**What the profile maps.** Only what differs from the shipped defaults (rule 4: every key it sets is a key of
the defaults): `specsDir` and `specArchiveDir` (`DEV/plan`, `DEV/plan/done`), the ledger, archive and burned-id
journals under `DEV/plan/`, the queue files, `paths.probeBaseline`, the ticket grammar (`^E\d{4}$`, files
`DEV/plan/E<nnnn>_<slug>.md`) and the probe roots (`cmd`, `internal`, `extension/src`). The gate is still this
repository's own [`scripts/check.ps1`](../../scripts/check.ps1); the harness drives the ticket flow, not the build.

**What this repo owes it.** A profile that the shipped reader accepts at its version, with keys the defaults
declare. The file is not yet in git, so a clone has no profile until it is committed.

**Conformance.** No catalog vectors are consumed; the shipped `_profile.ps1` validates the `version` carrier and
the keys on every harness call.

**History.** Read as not applicable at 0.10 and 0.11 (2026-10-07, ticket 108) and at 0.12 (2026-10-20, ticket 120):
no profile existed and nothing here invoked the harness.

**Re-read 2026-10-29 (ticket E0126) at 0.14.** 0.13 adds the optional `version` carrier (absent or `"1"` = wire
1.0, which this profile writes); 0.14 adds rules 8-10 (runner instance fields, per-instance prompt and model
maps, an optional `portfolio` object), all optional and not used here. The not-applicable reading is withdrawn:
since 2026-10-29 this repository runs the shipped harness through `.sza-profile.json`.
