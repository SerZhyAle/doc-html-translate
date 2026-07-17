# Changelog — record "What's new"

> **GLOBAL DIRECTIVES:**
> 1. Dry, factual, evidence-rich prose - no filler, no marketing.
> 2. Typography: short hyphens (no long dashes), Russian **ё** where applicable, ".." not "..." (CLAUDE.md).
> 3. You author **only the Description**. Timestamp, Path and (optionally) Target are filled by the script.
> 4. Local + free. This never pushes or publishes.

Append one row to [DEV/CHANGELOG.md](../../DEV/CHANGELOG.md) for the change that is done (or staged).
The changelog is the build-granularity record; at release time it feeds the version's "What's new".

## Usage

```text
/changelog [short hint about the change, optional]
```

Run it after the code/content edit is complete - ideally with the change **staged** (`git add -A`),
because the script reads the changed-file list from git.

## The format (do not retype it by hand)

The file is a fixed 4-column Markdown table: `| Timestamp | Path | Target | Description |`.
Three columns are mechanical and the script fills them:

- **Timestamp** - now.
- **Path** - the changed files, taken from `git diff --cached --name-only` (staged), else the unstaged diff.
- **Target** - the commit subject you pass as `-Target`.

You write only **Description**.

## Process

**Step 1 — Make sure the change is staged.** `git add -A` (or stage the relevant files) so the Path column
is accurate. If you deliberately want a different Path, pass `-Path "a.go, b.js"`.

**Step 2 — Pick the Target (commit subject).** A conventional, user-readable subject - the same one the
build/commit will use: `feat: ...` · `fix: ...` · `perf: ...` · `docs: ...` · `refactor: ...` · `chore: ...`.
GitHub Release notes are generated from commit subjects, so this line becomes part of the next version's
"What's new".

**Step 3 — Write the Description.** Match the house style of the existing entries:
- State what was wrong and what changed, with **measured** evidence, not adjectives
  (e.g. "a 128x104 postage stamp", "Images: 2 -> 1", "3x scale duplicate collapsed").
- When behaviour was checked in a browser, say so concretely: `Chrome-checked: total=1 render=1 broken=0`,
  `<embed application/pdf> present`. (`/verify-view` produces exactly these lines - paste them in.)
- Cross-edition changes: note both the Go and the JS side, and whether docs/PARITY.md moved.
- One physical line is fine - inline **bold**/`code` is fine; do **not** use a raw `|` (the script escapes it,
  but avoid it). Keep project typography.

**Step 4 — Append the row:**

```powershell
./scripts/add_to_dev_log.ps1 -Target "<commit subject>" -Description @'
<your Description prose>
'@
```

(`a log -Target "..." -Description "..."` is the short alias.)

**Step 5 — Report** the one line appended. Do not commit here unless asked - `/build` commits, and it calls
this same script at its Step 3.

## When NOT to use

- You are running `/build` - it already records the changelog at Step 3 (this skill is what it uses).
- Nothing changed yet - make the change first.
