# Pointer: RULE-DELIVERY

- **Id:** `RULE-DELIVERY`
- **Version:** 0.9 draft
- **Home:** the shared contracts catalog, `rule-adoption/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - receives the canon rule set through the `sza` Claude Code plugin
- **Wire carrier:** plugin version, derived from the canon's `CANON_VERSION`

**What this repo owes it**

- Staleness is judged from the digest in [`.sza-canon.json`](../../.sza-canon.json) against the live rule
  documents, not from a version count (rule 5). A differing digest is a warning; past 180 days from
  `canon.adoptedOn` it is an error and a full re-adoption is owed.
- A warning is cleared by reconciling the changed rule documents through the adopt-canon flow, which
  rewrites the stamp (rule 7 and `REPO-STAMP` rule 8).

**Known state.** Stale - the stamp is at canon `2026.09.06.1`; ticket
`DEV/plan/22_2026-09-23_contract-rule-adoption-sync.md` owns the reconciliation.

**Conformance.** The canon plugin's compliance gate, finding `SZA-CANON03`.
