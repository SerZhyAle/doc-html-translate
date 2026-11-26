---
paths:
  - "DEV/research/**"
  - "tests/research_naming_test.go"
---

# Research notes

Moved verbatim from `AGENTS.md` / `CLAUDE.md` (2026-11-25); the short core there still binds. A section
name quoted below is a section of `AGENTS.md` unless it says otherwise.

## Naming and registration (from CLAUDE.md)

- A standalone research note is `DEV/research/RESEARCH_<topic-slug>_<YYYY-MM-DD>.md`, or a folder of that
  name holding a `README.md` when it carries fixtures or test pages. The `RESEARCH_` prefix is the interface
  (`REPO-LAYOUT` rule 3): a glob or an index finds the notes by it, whatever folder they end up in.
  Findings that belong to one ticket go into that ticket's own folder, `DEV/plan/<ticket>/research/`.
- The notes named before 2026-09-25 (lowercase, no prefix) keep their names: tickets and the frozen
  `DEV/plan/done/` archive link to them. `tests/research_naming_test.go` holds that list, which only
  shrinks, and fails on any new name without the prefix.
- `CODE_QUALITY.md`, `VALIDATION.md` and `RESEARCH_INDEX.md` beside them are method references, not notes.
- A new note is covered by the `research-notes` record of `docs/DOCUMENT_REGISTRY.jsonl` by its name alone;
  `scripts/doc-registry.ps1` fails on a file there that no record claims.
