---
paths:
  - ".claude/commands/**"
  - ".claude/agents/**"
---

# Slash commands and subagents

Moved verbatim from `AGENTS.md` / `CLAUDE.md` (2026-11-25); the short core there still binds. A section
name quoted below is a section of `AGENTS.md` unless it says otherwise.

## Skill routing (from CLAUDE.md)

Imported from the Universal Agent Kit, plus the `sza` plugin's ticket-queue forwards; the live set is
whatever `.claude/commands/` holds. Pick the cheapest path that fits:

- `/quick` - trivial edit (typo, one constant, one string). No spec, no build gate.
- `/fix` - narrow bug/behaviour fix. Local validation only (a targeted `go test` of the touched package, 5 s or less; `scripts/test.ps1` is the release tier).
- `/research` - investigate before any non-trivial change; persist findings under `DEV/research/`.
- `/spec` - write the strategic *what/why* for a feature (no file paths, no symbols).
- `/spec-tech` - break an approved spec into a phased, verifiable plan.
- `/spec-dev` - execute a tactical plan one step at a time, checking each.
- `/spec-check` - audit implementation against the spec; set status from reality.
- `/spec-fix` - apply the audit's mechanical action items, then re-audit.
- `/git` - branch model, staging, commit grouping.
- `/changelog` - record the "What's new" entry in the engineering ledger.
- `/docs-sync` - propagate a finished change across every documentation surface at once.
- `/verify-view` - headless-check converted HTML output.
- `/build` - "сборка": local+free flow (gate, build CLI+UI, changelog entry, commit). Never publishes.
- `/release` - "релиз": published+paid flow. Curates "What's new in version XXX" and drives the full
  checklist (GitHub release, winget, Store, extension) with per-step confirmation. See [DEV/RELEASE.md](../../DEV/RELEASE.md).
- `/spec-draft` (`/sd`), `/spec-quiz` (`/sq`), `/spec-next` (`/sn`), `/spec-all` (`/sa`) - the ticket-queue
  commands: capture a Draft, clear a `BlockQuestions` ticket, work the next ticket of the queue, take one
  ticket to its audit. Each file is a thin forward to the `sza` plugin skill of the same name and carries no
  behaviour of its own.

Subagents: `solution-researcher` (read-only investigator), `implementer` (focused code writer),
`rd-lead-kit` (senior orchestrator - opt-in, not the default agent). Methodology references:
[docs/SPEC_LIFECYCLE.md](../../docs/SPEC_LIFECYCLE.md), [DEV/research/CODE_QUALITY.md](../../DEV/research/CODE_QUALITY.md),
[DEV/research/VALIDATION.md](../../DEV/research/VALIDATION.md), [DEV/research/RESEARCH_INDEX.md](../../DEV/research/RESEARCH_INDEX.md).
Where these conflict with [AGENTS.md](../../AGENTS.md), **AGENTS.md wins**.
