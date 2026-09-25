# Pointer: REPO-LAYOUT

- **Id:** `REPO-LAYOUT`
- **Version:** 0.9 draft
- **Home:** the shared contracts catalog, `rule-adoption/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - repository structure and named entry points
- **Wire carrier:** none - file and directory names

**What this repo owes it**

- `CLAUDE.md`, `AGENTS.md`, `README.md`, `LICENSE` at the root.
- Contract pointers under `docs/contracts/<ID>.md`, indexed by `docs/contracts/README.md`; pointers, never
  copies (rule 2).
- Engineering ledger at `DEV/CHANGELOG.md` (ledger shape 2, as the stamp declares).
- `DEV/plan/done/` is an archive and is left as it is (rule 6).

**Open adaptation (rule 3).** Tickets are `DEV/plan/NN_<date>_<slug>.md` and research is
`DEV/research/<topic>_<date>.md`, with no type prefix. The scheme is declared in `AGENTS.md` and
`CLAUDE.md`; whether a declared scheme outside `PLAN/` is an allowed adaptation is an open question to the
catalog, owned by ticket `DEV/plan/22_2026-09-23_contract-rule-adoption-sync.md`. New documents keep the
current scheme until it is answered.

**Conformance.** The canon plugin's compliance gate.
