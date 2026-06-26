# Specification Implementation Audit

Audit a spec against the **actual repository state**. Status comes from what the code proves,
not from a filename or a hope. Auto-detects strategic vs tactical scope.

> No separate audit file is written. Findings go in a compact `## Last Audit` block at the
> bottom of the strategic spec, overwritten each run. Old audit history is intentionally
> discarded.

## Usage

```text
/spec-check <ID-or-slug>
/spec-check <ID-or-slug> --strategic
/spec-check <ID-or-slug> --tactical
/spec-check <ID-or-slug> --phase <NN>
/spec-check <ID-or-slug> --strict      # treat WARN as FAIL
/spec-check <ID-or-slug> --quick       # skip grep-heavy invariants
```

## Auto-detection

| Strategic file | Tactical folder | Flag | Mode |
| :---: | :---: | --- | --- |
| exists | exists | none | full — strategic + every phase |
| exists | missing | none | strategic only |
| exists | exists | `--strategic` | strategic only |
| exists | exists | `--tactical` | every phase |
| exists | exists | `--phase NN` | single phase |
| missing | any | any | abort |

## Process

**1 — Locate the spec.** Read `DEV/plan/YYYY-MM-DD_<slug>_<slug>.md` (abort if missing). Record whether
`INDEX.md` exists. Apply the auto-detection table.

**2 — Extract the verification contract.**
- Strategic: §2 Goals, §3.2 Constraints, §6 Research items, §8 user-doc text, §11 Criteria.
- Tactical: INDEX Phase Overview / Blockers / Completion Gate; each phase's Files Touched,
  Steps Verification, Done Criteria.

**3 — Run checks.** Record per check: `PASS` / `WARN` / `FAIL` / `MANUAL` / `UNCHECKABLE` /
`EXEMPT`. Mechanics:

| Check | How |
| --- | --- |
| File exists | glob the exact path |
| Symbol declared | grep for the declaration — verify the hit is a declaration, not a comment/string |
| No forbidden call | grep the pattern; PASS iff zero hits |
| Schema version | read the schema source, match the expected version |
| Changelog entry | grep the file path in the changelog |
| Index up to date | grep the symbol in your code index, if any |
| Dead weight introduced | grep for remnants the change should have removed (orphaned symbols, keep-rules for deleted code, unreferenced resources); WARN per remnant. Cross-check `DEV/plan/` before calling a zero-ref artifact dead — it may be active-ticket scaffolding |
| User docs | read strategic §8 first. "No changes" → EXEMPT. Else grep the keyword in the user docs — PASS only if present |
| File size vs budget | read the file, count lines, compare to the step budget |
| Step status consistency | parse `[x] done`; cross-check against the Verification predicates |
| Phase status consistency | INDEX row status == phase header status |
| Verification-tag invariant | if status is `BlockNeedUserTest`: grep for `YYYY-MM-DD_<slug>:` tags — PASS iff ≥ 1 hit. For any other status: PASS iff zero hits — surviving tags are stale (WARN; step 6 deletes them) |

**4 — Score.** (Gates are defined once in `docs/SPEC_LIFECYCLE.md`; this is that rule applied.)
- `Verified` — every check is PASS or EXEMPT. Zero FAIL, zero WARN, and **no open MANUAL item**.
- `BlockNeedUserTest` — zero FAIL and zero WARN, but ≥ 1 open MANUAL / on-target item. The build
  is sound; a human signal is still outstanding. Do **not** call this Verified and do **not**
  remove verification tags — re-run `/spec-check` once the human closes the item.
- `Partial` — zero FAIL, ≥ 1 WARN. Collapses to `Broken` under `--strict`.
- `Broken` — ≥ 1 FAIL.

A MANUAL check can never sit beneath a `Verified`: status comes from reality, and an unclosed
manual signal is reality saying "not proven yet".

**5 — Write the `## Last Audit` block** at the bottom of the strategic spec (overwrite).
Keep it under ~40 lines: PASS counts + FAIL/WARN action items only. The action items are
exactly what `/spec-fix` consumes.

**6 — Update `Status:`.** Full/strategic mode → flip the strategic `Status:` to the score.
Whenever the verdict flips the status **out of** `BlockNeedUserTest`, enforce the tag
invariant: grep all source for `YYYY-MM-DD_<slug>:` verification tags and delete every matching line
(idempotent if none). This is the only source mutation this skill performs. Tactical-only
mode → update INDEX + audited phase rows only; do not touch the strategic status or tags.

**7 — Finalize.** Changelog line per modified file (the spec, plus any source file that lost
a tag). Update the user-facing feature docs only on a `Verified` flip for a genuinely
user-visible change.

**8 — Auto-chain to `/spec-fix`** if the verdict is `Partial` or `Broken`. On `Verified`, no
further action. On `BlockNeedUserTest` (sound build, open manual item), stop and await the human
signal — do not chain, do not remove tags.

**Chat output:** `YYYY-MM-DD_<slug>: <score>. PASS/WARN/FAIL: N/N/N. Tags removed: N. Top issues: [list].`

## `## Last Audit` — compact template

```markdown
## Last Audit

**Date:** <YYYY-MM-DD>
**Mode:** full | strategic | tactical | phase-<NN>
**Outcome:** Verified | BlockNeedUserTest | Partial | Broken
**Counts:** PASS N · WARN N · FAIL N · MANUAL N · EXEMPT N

### Action items
1. **[FAIL §3.2.2 — Step 02.3]** <one line> — <concrete fix>.
2. **[WARN §2.3]** <one line> — <concrete fix>.

### Manual / on-target
- [ ] <signal from §11 that needs a human or a real run>

<Verified requires every Manual item checked [x] or absent. An open [ ] item forbids Verified —
the outcome is BlockNeedUserTest until a human closes it. If Verified: drop "Action items".>
```

## Constraints

- Never mutate spec content beyond `Status:`, a recomputed `Priority`, and the `## Last
  Audit` block. The one source mutation allowed is deleting `YYYY-MM-DD_<slug>:` verification tags on a
  status flip out of `BlockNeedUserTest`.
- Strategic audit is qualitative (keyword overlap for goal coverage); tactical audit is
  strict (static predicates only).
- A grep miss is FAIL. A hit-count mismatch (expected 1, found 3) is WARN with all hits
  listed.
- Never run a build — static analysis only.
- Read-only zones ignored.
- `--quick` skips grep-heavy invariants and annotates the block; it still emits the status
  transition and tag removal.
- Never approve `Verified` while any tactical phase is Broken. Grep hits count only on
  declaration lines, not comments or string literals.
