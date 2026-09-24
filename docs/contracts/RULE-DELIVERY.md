# Pointer: RULE-DELIVERY

- **Id:** `RULE-DELIVERY`
- **Version:** 0.9 draft
- **Home:** the shared contracts catalog, `rule-adoption/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - canon rule distribution and verification
- **Wire carrier:** plugin version (semver), derived from `CANON_VERSION`

Canon delivery and staleness rules:
- Rule delivery managed via `sza` Claude Code plugin.
- Canon version and digest tracked in `.sza-canon.json`.
- Validated via `check-compliance.ps1`.

**Conformance.** Canon compliance checks.
