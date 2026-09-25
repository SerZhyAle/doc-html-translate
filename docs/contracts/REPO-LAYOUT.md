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

**Not held: rule 3, a dated exception in the catalog registry (until 2026-12-31).** Documents outside
`docs/contracts/` carry no type-first prefix:

- Tickets are `DEV/plan/NN_<YYYY-MM-DD>_<slug>.md`, a scheme declared in `CLAUDE.md` ("Spec / plan
  tickets"). The catalog is asked to let a declared scheme stand wherever it lives, not only under `PLAN/`
  (`PROPOSAL-2026-09-23-own-spec-scheme.md`, seconded by
  `PROPOSAL-2026-09-25-doc-html-translate-rule-adoption.md`). Tickets keep their names until it answers.
- Notes under `DEV/research/` follow no declared scheme. That is an ordinary gap, not an adaptation, and
  ticket `DEV/plan/31_2026-09-25_canon-resync-new-duties.md` closes it.

**Conformance.** The canon plugin's compliance gate.
