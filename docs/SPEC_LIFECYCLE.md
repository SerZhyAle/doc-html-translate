# The Spec Lifecycle - tooling-agnostic

This is the methodology behind the `/spec*` skills. It needs no special tooling: a ticket is
one Markdown file, and a status is a line in that file you keep honest. If you later want a
JSONL index, a CLI, or a database behind it, fine - the method does not depend on it.

## Why split strategy from tactics

A spec answers two different questions that get muddled when written together:

- **Strategic (what / why):** the problem, the goals, the constraints, the open questions.
  No class names, no file paths, no signatures. If a reviewer disagrees here, you saved
  yourself an implementation.
- **Tactical (how):** the exact files, symbols, and the *order* to build them, each step
  ending in a check anyone can run. If a step is ambiguous here, the executor stops instead
  of guessing.

Keeping them in separate documents is the single highest-leverage habit in this kit. The
strategic doc is the contract; the tactical doc is the build instructions; the audit checks
the building against the contract.

## The status flow

```
Draft ──► Approved ──► Tactical ──► In Progress ──► Implemented ──► Verified
   │                                                      │
   └─ rough is fine here                                  └─ audit may instead yield:
                                                             Partial  (warnings, no failures)
                                                             Broken   (at least one failure)
```

Plus explicit block states, each entered with a one-line note:

- `BlockNeedUserTest` - built, awaiting a hands-on / manual check.
- `BlockByOtherTask` - depends on another ticket.
- `BlockQuestions` - awaiting a decision from the owner.
- `BlockExternal` - waiting on a library release, hardware, or a third party.

The status is the **first** `**Status:**` line in the file. It is true because the code makes
it true, never because the filename or a wish says so.

## The two complexity paths

Not every change deserves a phased plan. Before writing phases, score the task:

- ≤ 3 existing files change, no new files
- no new public types
- no schema/migration change
- no new dependency-injection wiring
- no new screen/route/destination
- mechanically deterministic - no deferred decisions
- under ~100 lines of delta

**All true → PRIMITIVE:** write a three-section spec (Problem / Approach / Done criteria) and
implement directly. **Any false → COMPLEX:** run the full pipeline. This gate is what keeps
the ceremony proportional to the work. (Even cheaper: a typo is `/quick`; a known bug is
`/fix`; neither needs a spec at all.)

## Designing the phase graph (the part that goes wrong)

Phase ordering is where tactical plans fail. Three passes prevent it:

1. **Coverage inventory.** List every goal, pillar, constraint-with-impact, resolved research
   finding, ADR, and criterion. Map each to a phase, or mark it `out-of-scope: <reason>`. An
   unmapped line means the plan is incomplete.
2. **Produces / Consumes topology.** For each phase, list what it produces (new/changed
   artifacts) and what it consumes. No phase may consume what a *later* phase produces. A
   forward reference means the order is wrong - reorder before writing anything.
3. **Real-work filter.** Every step's primary action changes source/resources/config. Steps
   that only edit plan text, "review docs", or restate a previous step are not steps. The one
   exception is the final docs-cleanup phase.

Then a mandatory **self-review**: re-read the written plan against the inventory and the
topology. Every inventory line maps to a *written* step; every consumed symbol greps in the
code or is produced earlier; every dependency matches the topology. Fix in place. This pass
catches the phase-order bug that would otherwise cost a full execution cycle.

## Verifiable steps

Every step ends in a **static** check - file exists, symbol declared, value equals, command
exits 0 - never "works correctly". A step is done only when its check passes *in this run*,
not when you intended it. The executor runs one step, runs its check, and hard-stops on the
first failure or ambiguity rather than guessing forward.

## Verification tags (optional but neat)

When a ticket needs manual testing before it can be called done, insert a temporary log line
at each changed-flow entry point:

```
internal/logging (logging.Printf/Println/Errorf)("YYYY-MM-DD_<slug>: <what this code path proves>")
```

The invariant: such a tag exists **iff** the ticket is currently `BlockNeedUserTest`. You
insert them as the last code edits before the final build (so one build validates code +
tags), and the audit deletes them when the ticket leaves that status. Permanent logs never
embed a ticket id. The payoff: during a manual run you grep the log for `YYYY-MM-DD_<slug>:` and *see*
which paths actually executed. It complements automated tests, it does not replace them - reach
for it only where a test cannot observe which path a manual or on-device run took. Keep each tag a
single greppable line so removal stays mechanical and safe.

## The skills, end to end

```
/research      gather evidence, persist findings           (before anything non-trivial)
/spec          write the strategic what/why                Draft -> Approved
/spec-tech     turn it into a phased, verifiable plan       Approved -> Tactical
/spec-dev      execute one step at a time, check each       Tactical -> Implemented / BlockNeedUserTest
/spec-check    audit the build against the spec             -> Verified / Partial / Broken
/spec-fix      apply the audit's action items, re-audit     Partial / Broken -> (re-check)
/verify        run it and observe, when behaviour matters   (read-only on status)
```

`/quick` and `/fix` sit *below* this pipeline for changes too small to deserve it. `/ui-clarify`
sits *before* it whenever a user-facing decision is unresolved.

## Status gates (the one rule)

Every skill reads and writes the same ticket status, so the gates live here once. Each skill
**requires** an entry status, **produces** an exit status, and **auto-chains** to the next skill
unless a stop condition fires. This table is the single source of truth - a skill restates only
its own row, and `VALIDATION.md` and the skill files defer to it rather than redefining the flow.

| Skill | Requires | Produces | Auto-chains to | Stops instead when |
| --- | --- | --- | --- | --- |
| `/research` | any | no status change | the caller | - |
| `/spec` | none (allocates an id) | `Approved`, or `In Progress` for a primitive | `/spec-tech` (primitive: implement directly, then stop) | a required research item is Open |
| `/spec-tech` | `Approved` or later | `Tactical` | `/spec-dev` | an unchecked pre-implementation blocker remains |
| `/spec-dev` | `Tactical` / `In Progress` | `Implemented` or `BlockNeedUserTest` | `/spec-check` | ambiguity, a failed check, or a `Block*` condition |
| `/spec-check` | `Implemented` or later | `Verified` / `Partial` / `Broken` / `BlockNeedUserTest` | `/spec-fix` (only if `Partial`/`Broken`) | an open manual check holds at `BlockNeedUserTest` |
| `/spec-fix` | `Partial` / `Broken` | re-runs `/spec-check` | `/spec-check` | a fix needs a decision the audit cannot supply |
| `/verify` | any | no status change | - | - |

Three rules bind the whole table:

- **Auto-chain by default; stop only at a real decision.** The pipeline flows on its own and
  pauses only at the encoded conditions above - an open required question, an unchecked blocker,
  an ambiguity. It does **not** wait for a sign-off at every stage; that would be bureaucracy, not
  safety. `/spec` records its own `Approved` flip and continues. To review between stages, run the
  skills individually or pass `--dry-run`.
- **`Verified` means nothing is left open.** A ticket reaches `Verified` only when every check is
  PASS or EXEMPT - zero FAIL, zero WARN, and **no open MANUAL item**. An unresolved manual /
  on-target signal keeps the ticket at `BlockNeedUserTest` (or `Partial`) until a human closes it
  and a re-audit turns it green. Status comes from reality, so an open manual check can never sit
  beneath a `Verified`.
- **Two branches leave the straight line, and the rows above encode them.** A *primitive* `/spec`
  (the complexity gate) implements without a tactical plan and stops at `Implemented` or
  `BlockNeedUserTest` - it never enters `/spec-tech`. A sound build with an unresolved manual check
  produces `BlockNeedUserTest`, not `Verified`. Everything else follows the linear flow.

## Adapting it

- Replace `YYYY-MM-DD_<slug>` with whatever id scheme you like (`T0042`, `JIRA-123`, a date-slug).
- Replace `DEV/plan` with your docs location.
- If you do not want block states, drop them - the linear flow still works.
- If your team already has tickets in an external tracker, keep the *file* as the spec and
  link the tracker id; the discipline is in the document, not the tool.
