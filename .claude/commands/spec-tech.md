# Tactical Specification Writer

Break an approved strategic spec into **sequenced, verifiable phases**. Creates
`DEV/plan/YYYY-MM-DD_<slug>_<slug>/INDEX.md` + phase files. Language: English, imperative, no rationale
prose (the *why* lives in the strategic spec).

Requires the strategic spec at `Status: Approved` or later (gate table: `docs/SPEC_LIFECYCLE.md`).
If `Draft`, auto-promote to `Approved` first and note it in chat. A `Block*` status is a hard
abort — resolve first.

## Usage

```text
/spec-tech <ID-or-slug>
/spec-tech <ID-or-slug> --phase <NN>
/spec-tech <ID-or-slug> --dry-run
```

## Directory layout

```text
DEV/plan/YYYY-MM-DD_<slug>_<slug>.md            # strategic — owned by /spec
DEV/plan/YYYY-MM-DD_<slug>_<slug>/
  INDEX.md
  research/<NN>__<topic>.md          # research artifacts (first-class planning input)
  PHASE_01__<slug>.md
  ..
  PHASE_NN__docs-cleanup.md          # always the final phase
```

Phase slug: kebab-case, ≤ 4 words (`foundations`, `input-dispatch`, `db-migration`).

## Process

**1 — Validate the strategic spec.** Read it. Extract: feature name, tier, priority, goals
(§2), constraints (§3.2), pillars (§5.1), open research items (§6) + their artifact links,
ADRs (§9), criteria (§11). Auto-promote from Draft if needed.

**2 — Read project context.** Read **every** file in `research/` in full first — a resolved
finding that contradicts the intended approach is a planning input, not a footnote. Then the
repo map, architecture doc, and the actual source for the affected area. Every file path you
reference in a step must exist or be explicitly marked **New**.

**2.5 — Complexity gate (PRIMITIVE check).** Same checklist as `/spec` step 2.5. If ALL
pass: implement directly, no phase files, set status per acceptance (Implemented or
BlockNeedUserTest), report, stop. No `/spec-dev` chain. If ANY fails: continue.

**3 — Design the phase graph.** This is the highest-risk output of the skill: a wrong order
or a missed strategic requirement costs a full `/spec-dev` cycle. Do not write any file
until 3.1–3.4 pass.

**3.1 — Coverage inventory.** Re-read the strategic spec end-to-end plus every research
file. Build a scratch inventory (chat-side, never a file): one line per §2 goal, §5.1
pillar, §3.2 constraint with implementation impact, resolved §6 finding, §9 ADR, §11
criterion. Map every line to ≥ 1 planned phase, or mark it `out-of-scope: <reason>`. An
unmapped line means the phase set is incomplete — fix it before proceeding.

**3.2 — Produces/Consumes topology.** For each candidate phase list two sets: `Produces`
(new or changed artifacts: types, functions, schema, DI bindings, resources, config fields)
and `Consumes` (artifacts it needs — either pre-existing in code, verified in step 2, or
produced by a strictly earlier phase). Validate the topological order: no phase consumes
something a later phase produces. A forward reference means the order is wrong — reorder now.

**3.3 — Ordering heuristics** (refine the 3.2 topology, never override it):
1. Foundations first: data types, interfaces, DI, schema + migration, config/feature flags.
2. Producer before consumer for every new symbol; migration before code reading new fields;
   resources/strings before or with the UI referencing them.
3. User-visible changes last within their area.
4. The final phase is always `PHASE_NN__docs-cleanup.md`: changelog, index regen, and
   user-facing docs only if strategic §8 mandates an update.
5. At least one phase per strategic pillar (§5.1); small pillars may fuse.

**3.4 — Real-work filter (anti-bureaucracy).** Every step's primary action must change
source, resources, config, or scripts. Forbidden as steps:
- Edits to `DEV/plan/**` text — status flips, counters, retitling. Progress tracking is
  `/spec-dev` bookkeeping; plan authoring is this skill's own output — neither is plan
  *content*.
- "Review / sync / align docs" without a concrete file delta outside `DEV/plan/`.
- Restating or re-verifying a previous step's outcome as a separate step.

Sole exception: the final docs-cleanup phase. A phase where most steps fail this filter is
not a phase — merge the survivors into a real one.

Phase shape: each phase mergeable as a coherent unit; one build-time invariant proving
completion; no half-broken state between steps. Target 3–8 phases. > 10 → split the feature
into multiple specs.

**4 — Write `INDEX.md`** (template below).

**5 — Write each `PHASE_NN__<slug>.md`** (template below). Steps numbered `NN.M`.

**5.5 — Plan self-review (mandatory).** After all phase files exist and before any status
flip, re-read INDEX + every phase against the 3.1 inventory and 3.2 topology:
- Every inventory line maps to a *written* step (not an intended one), or carries its
  `out-of-scope` reason.
- Every symbol a step consumes either greps in the current codebase or is created by an
  earlier step — check the actual `Files Touched` + prompts, not intent.
- Every `Depends on` matches the topology; no step references a later phase's artifact.
- No step violates the 3.4 real-work filter.
- Research findings are reflected. A step contradicting a resolved §6 artifact is a planning
  bug to fix *here*.

Fix findings directly (reorder, rewrite, renumber), re-run the failed check once. Report:
`Plan self-check: PASS — N inventory items mapped, M reorders applied.` Never skip this —
phase-order bugs are the dominant tactical-plan defect.

**6 — Update the strategic spec.** Flip `Status:` to `Tactical`; add the tactical-plan link.

**7 — Auto-chain to `/spec-dev`.** If no unchecked Pre-Implementation Blocker in INDEX,
invoke `/spec-dev YYYY-MM-DD_<slug>`. Otherwise list the blockers and stop.

**Chat output:** `YYYY-MM-DD_<slug>: N phases. Blockers: [list or none]. -> Running /spec-dev…`

## `INDEX.md` template

```markdown
# Tactical plan: YYYY-MM-DD_<slug> — <slug>

**Strategic spec:** [`../YYYY-MM-DD_<slug>_<slug>.md`](../YYYY-MM-DD_<slug>_<slug>.md)
**Research inputs:** [`research/<NN>__<topic>.md`](research/<NN>__<topic>.md) <or "none">
**Tier:** <label> · **Priority:** <0..100>
**Status:** Not started
**Phases:** 0 / N done
**Last updated:** <YYYY-MM-DD>

> **Scope:** tactical, English, developer handoff. Every step has a verification predicate.
> Rationale lives in the strategic spec.

## Phase overview
| # | Phase | Depends on | Status | Steps | File |
|---|-------|-----------|--------|------:|------|
| 01 | <slug> | - | ⬜ Not started | 0/N | [PHASE_01__<slug>.md](PHASE_01__<slug>.md) |
| NN | docs-cleanup | all | ⬜ Not started | 0/N | [PHASE_NN__docs-cleanup.md](PHASE_NN__docs-cleanup.md) |

Legend: ⬜ Not started · 🚧 In Progress · ✅ Done · ⛔ Blocked · ⏭️ Skipped

## Pre-implementation blockers
<Every §6 item with Status: Open becomes a checkbox. Phase 01 must not start while any is
unchecked.>
- [ ] **Research:** <title> — required before Phase <NN>.

## Completion gate
- [ ] All phases ✅ Done.
- [ ] User-facing docs updated only if strategic §8 mandates it.
- [ ] Changelog has an entry per modified file.
- [ ] `/spec-check YYYY-MM-DD_<slug>` returns Verified.

## How to track progress
1. Before a phase: flip its row to 🚧, update `Phases: X/N`.
2. During: flip a step to `[~]` when started, `[x]` when its Verification passes — never on
   intent.
3. On phase done: confirm every step `[x]`, confirm Done Criteria, flip row to ✅, bump.
4. If blocked: flip to ⛔, log it; if the whole spec is blocked, set a `Block*` status.

## Change log
- <YYYY-MM-DD> — initial tactical plan authored by /spec-tech.
```

## Phase file template

```markdown
# Phase NN — <Title>

**Strategic spec:** [`../YYYY-MM-DD_<slug>_<slug>.md`](../YYYY-MM-DD_<slug>_<slug>.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ⬜ Not started
**Depends on:** Phase NN-M (or "none — foundation phase")
**Steps done:** 0 / N

## Objective
<One sentence: what this phase produces.>

## Prerequisites
- [ ] All "Depends on" phases are ✅ Done.
- [ ] Strategic §6 items blocking this phase are Resolved.
- [ ] Working tree clean or on a feature branch.

## Files touched
| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `internal/ and cmd//<path>/<File>` | New | ≤ 250 |
| `internal/ and cmd//<path>/<Existing>` | Modified | ≤ 500 |

> Backup any file over ~500 lines before editing; split anything heading past your file-size
> budget via a helper/extraction step.

## Steps

### Step NN.1 — <imperative title>
**Files:** `path/to/File`
**Depends on:** - start of phase

**Prompt for developer:**
> <Self-contained imperative, 1-4 sentences. The reader must not need the strategic spec.>

**Verification:**
- File `path/to/File` exists.
- `<symbol declaration>` matches exactly once (declaration, not comment).
- `<expected signature / value>` present.

**Status:** `[ ]` not done

---

### Step NN.2 — <imperative title>
**Files:** ..
**Depends on:** Step NN.1
**Prompt for developer:** > ..
**Verification:** - ..
**Status:** `[ ]` not done

## Phase done criteria
- [ ] Every `Step NN.*` is `[x] done`.
- [ ] Narrowest meaningful check for this phase passes — pick the lowest sufficient rung from
      `docs/VALIDATION.md` (a compile / type-check, a targeted test, or `./scripts/build.ps1 (UI: ./scripts/build-ui.ps1)` only when
      this phase touches packaging, resources, or wiring). Name the actual command here.
- [ ] Grep for `TODO(phase-<NN>)` returns zero hits.
- [ ] Changelog entry added for every file in "Files touched".

## Handoff notes
<Invariants this phase established. Final phase → "See INDEX.md Completion gate.">

## Rollback plan
<Which commits to revert / config to restore. Low-risk → "Revert phase commit(s).">
```

## Constraints

- One step = one atomic, independently committable unit that does not break the build.
- Every Verification must be static (file exists / symbol present / value equality) — never
  "works correctly".
- File > ~500 lines after edit → backup step required. Heading past the size budget →
  refuse; split first.
- The final phase is always the docs-cleanup phase.
- Do not duplicate strategic content — tactical says *what*, not *why*.
- No step's primary action edits `DEV/plan/**` text (the real-work filter binds every
  step, not just the planning pass), except the final cleanup phase.
- Research artifacts under `research/` are mandatory planning input — read all before step 3
  and list them in INDEX `Research inputs:`.
- If your project has multiple form factors (e.g. portrait/landscape, light/dark, locale
  variants), a step editing one variant must cover its counterpart or note its absence.
