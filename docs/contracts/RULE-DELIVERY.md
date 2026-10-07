# Pointer: RULE-DELIVERY

- **Id:** `RULE-DELIVERY`
- **Version:** 0.12 draft
- **Home:** the shared contracts catalog, `rule-adoption/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - receives the canon rule set through the `sza` Claude Code plugin
- **Wire carrier:** plugin version, derived from the canon's `CANON_VERSION`

**What this repo owes it**

- Staleness is judged from the digest in [`.sza-canon.json`](../../.sza-canon.json) against the live rule
  documents, not from a version count (rule 5). A differing digest is a warning; past 180 days from
  `canon.adoptedOn` it is an error and a full re-adoption is owed.
- A warning is cleared by reconciling the changed rule documents through the adopt-canon flow, which
  rewrites the stamp (rule 7 and `REPO-STAMP` rule 8).

**Current reading (2026-10-07, ticket 108).** Rule 5 reads a parseable `canon.reconciledOn`,
then `canon.adoptedOn`; with neither parseable the age is unknown, treated as past 180 days.
An equal digest owes no finding; a differing digest with unknown age is an error. This product
has no stamp-age reader: it receives the plugin's checker rather than copying it. The stamp
still carries its actual adoption values; no synchronization claim is written by hand.

**Conformance.** No catalog vectors. The canon plugin owns the compliance reader and its tests;
its missing-date case must be verified there, not implemented in this repository.
