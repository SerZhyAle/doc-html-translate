# Fix — Narrow Bug / Behaviour Fix

> **GLOBAL DIRECTIVES (anti-bureaucracy):**
> 1. Dry technical prose, no filler.
> 2. Fix the bug, not the neighbourhood. No opportunistic refactors.
> 3. Terse report: root cause in one line, fix in one line, validation result.

A fast, low-ceremony path for a **well-understood bug or behaviour fix** that is bigger
than `/quick` but does not need a spec: no new design decisions, bounded blast radius,
local validation is enough.

## Usage

```text
/fix <description of the broken behaviour>
```

- `/fix list scroll position jumps to top after returning from detail`
- `/fix retry loop never backs off, hammers the server on 503`

## When NOT to use

- The fix needs design/UX decisions → `/ui-clarify` then `/spec`.
- The fix introduces new abstractions, schema/migration, or public API → `/spec`.
- You do not yet understand the root cause → `/research` first.
- It is a one-token edit → `/quick`.

## Process

**Step 1 — Reproduce / locate.** Confirm the failing path. Use `/research` mentally or for
real if the cause is not obvious. Name the root cause before touching code — if you cannot,
stop and research.

**Step 2 — Scope the change.** List the files you expect to touch and why. If the list
grows past a handful of files or starts needing new types, stop and escalate to `/spec`.

**Step 3 — Apply the fix.**
- Smallest change that fixes the root cause, not the symptom.
- No surrounding refactor, no unrelated import cleanup, no comment spray.
- Honour existing comments/invariants in the area.
- Anti-slop applies (`docs/CODE_QUALITY.md`).
- A handled edge case or non-obvious cause earns one English *why* comment — nothing more.

**Step 4 — Validate locally.** Run the narrowest meaningful check that proves the fix:
a targeted test if one exists (add/adjust one if cheap and the project tests), else compile
+ a manual run of the path. Record `expected | actual`.

**Step 5 — Report.** Root cause (1 line), fix (1 line), validation (1 line). If you found
adjacent bugs, list them as follow-ups — do **not** fix them in this pass.
