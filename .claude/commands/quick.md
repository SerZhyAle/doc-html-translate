# Quick Fix

> **GLOBAL DIRECTIVES (anti-bureaucracy):**
> 1. Strictly technical language — dry prose, no filler.
> 2. Autonomy over bureaucracy — no confirmation on trivial changes. Execute, log, report.
> 3. Terse reporting — one dry sentence stating what changed.

Fast path for **very minor** edits: a typo, one constant, one string, a color/spacing
tweak, renaming a single local. No spec is created; docs and build gates are skipped.

## Usage

```text
/quick <short description of the edit>
```

- `/quick fix typo in the Save button label`
- `/quick bump default timeout constant from 10s to 30s`
- `/quick change accent color token to the new brand value`

## When NOT to use

`/quick` is forbidden for:
- Any change to business logic, control flow, or data flow. Narrow behaviour fix → `/fix`;
  wider → `/spec`.
- New classes / functions / modules / public API.
- Schema/migration, dependency, or build-config changes.
- Multi-file refactors (> 3 files or > ~50 lines total).
- Any change to user-facing behaviour, visibility, or states. Clear bug → `/fix`; needs
  UX decisions → `/ui-clarify` + `/spec`.
- "Add a feature" / "make it possible to .." framings.

On any trigger: **refuse and route** to `/fix` (narrow bug) or `/spec` (wider). Do not run.

## Process

**Step 1 — Sanity gate.** Read `$ARGUMENTS`. If it exceeds "very minor", return one line
and stop:
`/quick does not fit — this is <reason>. Use /fix for a narrow bug, else /spec.`

**Step 2 — Locate the file.** Use your code index / grep for symbols; direct path or glob
for resource files. No deep audits.

**Step 3 — Read the target file, make the edit.**
- Match the file's existing style and conventions exactly.
- User-facing text → keep tone consistent with the rest of the product (see `/ui-clarify`
  if the project has a copy/tone policy).
- Anti-slop still applies (`docs/CODE_QUALITY.md`): no trivial comments, no hardcoded
  values where a token exists, no broad swallowing catches — even in a one-liner.

**Step 4 — Minimal validation.** Run the narrowest meaningful check for the file type
(compile for code, lint/format for config, none for a pure-text typo). Record
`expected | actual`.

**Step 5 — Do NOT run** the heavy machinery: no spec, no user-doc updates, no full build,
no `/ui-clarify` gate. If any of those is needed, the task was not "very minor" — go back to
Step 1 and refuse.

**Step 6 — Report.** One sentence: what changed, in which file. No summaries, no plan.
Example: `Save button label typo fixed in profile_screen — "Setttings" -> "Settings".`
