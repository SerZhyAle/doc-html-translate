# Pointer: REPO-LAYOUT

- **Id:** `REPO-LAYOUT`
- **Version:** 0.11 draft
- **Home:** the shared contracts catalog, `rule-adoption/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - repository structure and named entry points
- **Wire carrier:** none - file and directory names

**What this repo owes it**

- `CLAUDE.md`, `AGENTS.md`, `README.md`, `LICENSE` at the root.
- Contract pointers under `docs/contracts/<ID>.md`, indexed by `docs/contracts/README.md`; pointers, never
  copies (rule 2).
- Engineering ledger at `DEV/CHANGELOG.md` (ledger shape 2, as the stamp declares).
- `DEV/plan/done/` is an archive and is left as it is (rule 6).

**0.11 review (2026-10-07, ticket 108).** Rule 3 admits the declared ticket scheme in
`CLAUDE.md` and the release queue files named there. New research notes use `RESEARCH_`;
the pre-existing archive names remain frozen (rule 6). No renaming is owed by this round.

**Conformance.** No catalog vectors; the document registry and research naming gates check
the repository's declared shape. The canon plugin owns the general compliance reader.
