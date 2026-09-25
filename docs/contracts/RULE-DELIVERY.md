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

**Known state (2026-09-25).** Current - the stamp declares canon `2026.09.24.1` and its digest, written by
the adopt-canon run that reconciled the 13 rule documents changed since `2026.09.06.1`. `canon.adoptedOn`
stays `2026-08-18`, as the skill directs, so from 2027-02-14 the first drift reads as an error however
recent the last reconciliation; which run should move that date is an open proposal in the catalog
(`PROPOSAL-2026-09-23-adoption-date.md`, seconded by this repo).

**Conformance.** The canon plugin's compliance gate, finding `SZA-CANON03`.
