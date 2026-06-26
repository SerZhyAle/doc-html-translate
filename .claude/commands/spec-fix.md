# Specification Audit Fix-up

Apply the **mechanical** action items from the latest `/spec-check` audit, then re-audit. This
is the repair half of the audit loop: `/spec-check` writes a `## Last Audit` block with concrete
FAIL/WARN action items, and this skill closes the ones that need no human decision.

## Usage

```text
/spec-fix <ID-or-slug>
/spec-fix <ID-or-slug> --dry-run     # list what would change, write nothing
```

## Status gate

Follows the canonical gate table in `docs/SPEC_LIFECYCLE.md`.

- `Partial` / `Broken` → proceed.
- `Verified` → nothing to fix; report and stop.
- Anything else → abort: run `/spec-check` first to produce action items.

## Process

**1 — Read the audit.** Open `DEV/plan/YYYY-MM-DD_<slug>_<slug>.md`, read the `## Last Audit` block. If it
is missing or stale (no prior `/spec-check` this cycle), abort and ask to run `/spec-check`.

**2 — Classify each action item.**
- **Mechanical** — deterministic, fully specified by the item: a missing changelog entry, a
  stale `YYYY-MM-DD_<slug>:` verification tag to delete, a dead-weight remnant the change should have removed,
  a user-doc keyword the spec mandates but is absent, a renamed/missing symbol the item names
  exactly, a keep-rule for a deleted symbol.
- **Needs a decision** — anything requiring a design choice, a new name, a schema/DI shape, or
  user input. These are **not** fixed here.

**3 — Apply mechanical fixes**, one per action item, with `/spec-dev`'s edit discipline: scope
strictly to the item, no surrounding refactor, no invented names, anti-slop applies
(`docs/CODE_QUALITY.md`). Run each item's check (file exists / symbol present / keyword greps /
zero forbidden hits) before considering it closed.

**4 — Leave the rest.** List every "needs a decision" item verbatim and stop short of guessing.
If an item blocks all progress, set `BlockQuestions` with a one-line note.

**5 — Re-audit.** Auto-chain to `/spec-check YYYY-MM-DD_<slug>` so the status is recomputed from reality —
never set `Verified` from this skill. The re-audit decides the new status and removes any
verification tags on a flip out of `BlockNeedUserTest`.

**Chat output:** `YYYY-MM-DD_<slug>: fixed N/M action items. Left for you: [decision items]. -> Running /spec-check…`

## Hard stops

- An action item is ambiguous or under-specified → leave it, do not guess.
- A "fix" would require a design decision, a new public name, or user input → leave it, set
  `BlockQuestions` if it blocks the rest.
- A read-only zone or external-system touch → stop and require explicit permission.

## Constraints

- Never invent a fix the audit did not name; never set `Verified` (that is `/spec-check`'s job).
- Idempotent: a second run with the audit already satisfied is a no-op.
- One changelog line per modified file; the re-audit handles tag removal and the status flip.
