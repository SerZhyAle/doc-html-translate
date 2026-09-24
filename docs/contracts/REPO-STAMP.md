# Pointer: REPO-STAMP

- **Id:** `REPO-STAMP`
- **Version:** 0.9 draft
- **Home:** the shared contracts catalog, `rule-adoption/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** producer/consumer - `.sza-canon.json` declaration file at repository root
- **Wire carrier:** none - `.sza-canon.json` JSON fields

Repository metadata declaration:
- Located at `.sza-canon.json` at the root of the repository.
- Declares canon version, core digest, adoption date, model (`reference`), overlay (`A`), editions (`browser-extension`), channels, and exemptions.
- Validated by canon compliance checks.

**Conformance.** `tools/check-compliance.ps1` and `.sza-canon.json` structure.
