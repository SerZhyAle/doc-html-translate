# Git Guide

> **GLOBAL DIRECTIVES (anti-bureaucracy):**
> 1. Dry technical prose, no filler.
> 2. Autonomy: stage/group/diff freely. Block only for irreversible or shared-history ops.
> 3. Terse report: one statement of what was done and why.
> 4. **Never** `git commit`, `git push`, or history rewrites unless the user asked. On a
>    protected/default branch, branch first.

Git workflow: branching, staging, grouping changes into clean commits, inspecting history.

## Usage

```text
/git [optional question or action]
```

- `/git` — full reference
- `/git what should I commit now?`
- `/git analyze current changes and suggest commit groups`
- `/git show me the old version of <file>`

## Process

On `$ARGUMENTS`:
- **Empty** → output the reference below.
- **"Analyze changes"** →
  1. `git status` — all modified + untracked.
  2. `git branch --show-current` — confirm the active branch.
  3. Group files by feature/concern.
  4. Identify files that should NOT be committed (see exclusion list).
  5. Suggest 2–4 logical commit groups + proposed messages.
  6. Show the exact `git add` commands per group.
- **Specific file/topic** → focus there, use exact commands, never guess.

## Branching Model (adapt to your team)

A simple, robust default — replace with your team's flow if you have one:

- **`main`** — release-stable. No direct development. Receives merges from work branches
  and release-only fixes.
- **work branches** (`feat/<x>`, `fix/<x>`, or sequential `dev-vNNN`) — all development.
- Before any task: `git branch --show-current` — confirm you are where you expect.
- Keep the number of live work branches small.

> Many teams use trunk-based development or GitFlow instead. The kit does not impose one —
> it only insists you know which branch you are on before you commit.

## Daily flow

```bash
git branch --show-current
git status
git diff --stat

git add path/to/file
git commit -m "feat: description"
git push origin <branch>
```

## Inspect changes

```bash
git status                       # overview
git diff                         # all unstaged changes
git diff -- path/to/file         # one file
git diff --stat                  # changed file names only
git diff --cached                # what is staged
git log --oneline -10            # recent commits
```

## Stage

```bash
git add path/to/file
git add path/to/dir/             # a whole new directory
git add -p path/to/file          # choose hunks within a file
git diff --cached                # verify before commit
```

## Commit

```bash
git commit -m "feat: concise imperative summary"

# multi-line (heredoc), body only when the why is non-obvious
git commit -m "$(cat <<'EOF'
feat: short summary

- bullet on what and why
- bullet on a non-obvious consequence
EOF
)"
```

Conventional prefixes: `feat` · `fix` · `refactor` · `perf` · `docs` · `chore` · `test`
· `build` · `ci` · `style` · `revert`.

## Research old versions

```bash
git show <hash>:path/to/file     # file as it was at a commit
git log --oneline -- path/to/file
git blame path/to/file
git diff <hash1>..<hash2> -- path/to/file
git checkout <hash> -- path/to/file   # restore one file (overwrites working copy — careful)
```

## What NOT to commit

Never stage: build output, caches, local env/secrets, machine-specific config,
generated/auto-indexed files, large binaries, anything under your scratch dir. When in
doubt, check `.gitignore` and ask before adding a binary or a config file.

## Typical commit grouping

Split unrelated work into logical commits:

- **Feature** — the source change + its spec/plan file.
- **Docs & changelog** — user docs, dev log, changelog.
- **Infra** — build config, dependencies, settings.

One concern per commit; one commit message that explains the *why* when it is not obvious.
