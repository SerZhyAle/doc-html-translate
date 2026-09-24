# Pointer: REPO-LAYOUT

- **Id:** `REPO-LAYOUT`
- **Version:** 0.9 draft
- **Home:** the shared contracts catalog, `rule-adoption/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - repository structure and named entry points
- **Wire carrier:** none - file and directory names

Standard repository layout rules:
- Top-level `AGENTS.md` and `README.md`.
- Contract pointers under `docs/contracts/<ID>.md` indexed by `docs/contracts/README.md`.
- Engineering ledger at `DEV/CHANGELOG.md` (ledgerShape 2).
- Source tree structured under `cmd/` (CLI and GUI entry points) and `internal/` (Go packages), with independent extension under `extension/`.

**Conformance.** Canon compliance checks.
