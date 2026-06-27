# Build — "Сборка" (local, free)

> **GLOBAL DIRECTIVES:**
> 1. Dry technical prose, no filler.
> 2. This flow is **local and free**. It must NEVER push a tag, publish, or trigger CI.
>    If the user actually wants to publish → stop and switch to `/release`.
> 3. Typography in all generated content, docs and commit messages: short hyphens (no long
>    dashes), Russian **ё** where applicable, ".." not "..." (see CLAUDE.md).
> 4. Terse report: what built, commit hash, and an explicit "nothing pushed".

Prepare and execute a **build** ("сборка"): run the quality gate, build both binaries, record
"What's new" in the changelog, and commit — all locally. Canonical definition: [DEV/RELEASE.md](../../DEV/RELEASE.md).

## Usage

```text
/build [commit message or short description of the change]
```

- `/build fix navbar position on index`
- `/build` — when changes are already staged/described in the conversation

## When NOT to use

- You want to publish to GitHub / winget / Store / Chrome → `/release`.
- There is no code/content change yet → make the change first (`/fix`, `/quick`, `/spec-dev`).

## Process

**Step 0 — Confirm it's a build, not a release.** A build never pushes a tag or publishes.
If the user said "release"/"релиз", switch to `/release`. State this in one line and continue.

**Step 1 — Branch.** `git branch --show-current`. The build commits on the **current** branch.
If on a protected/default branch and the change is non-trivial, offer to branch first (see `/git`).

**Step 2 — Confirm the change is complete.** The actual code/content edit should already be done
(via `/fix`, `/quick`, `/spec-dev`, or this conversation). Do not start new feature work here.

**Step 3 — Record "What's new".** Append one row per meaningful change to [DEV/CHANGELOG.md](../../DEV/CHANGELOG.md)
(`| Timestamp | Path | Target | Description |`). Keep it factual and in project typography.
This is the build-granularity record; at release time it feeds the version's "What's new".

**Step 4 — Pick a clean commit subject.** The GitHub Release notes are auto-generated from commit
**subjects** between tags (`.github/workflows/release.yml`), so the subject you choose now becomes a
line in the next version's "What's new". Use a conventional, user-readable subject:
`feat: ...` · `fix: ...` · `perf: ...` · `docs: ...` · `refactor: ...` · `chore: ...`.

**Step 5 — Run the build.** This runs the gate (test + lint + typo), builds the CLI and UI
binaries, and commits (with `DEV/COMMIT_LOG.md` appended):

```powershell
./scripts/build-local.ps1 -Message "<conventional subject>"
```

- Build-only smoke test (no commit): `./scripts/build-local.ps1 -NoCommit`.
- Optional real-install test of the Store artifact (still local/free): `./msix/build-msix.ps1 -SelfSign`.

**Step 6 — On gate failure**, read `temp/logs/*.log`, fix the root cause, rerun. Do not bypass the gate.

**Step 7 — Report.** One line each: what was built, the short commit hash, and "nothing was pushed —
run `/release` for the published flow". List any follow-ups; do not act on them in this pass.
