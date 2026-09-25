# A release can ship binaries that are not the tagged tree

**Status:** Implemented (2026-09-26; checked by reading, actionlint and a dry run - the first real tag run proves the workflow half)
**Priority:** 90
**Date:** 2026-09-26

> Filed by the pre-release audit, [ticket 34](34_2026-09-25_full-code-audit-pre-release.md). Finding ids
> refer to its register, [`FINDINGS.md`](34_2026-09-25_full-code-audit-pre-release/FINDINGS.md).

## 1. Problem

- **R18 (high, plaus)** - `.github/workflows/release.yml` accepts `workflow_dispatch` with a free-text
  tag. The checkout has no `ref:`, so a dispatch builds the branch it was started from and publishes it
  under that tag (the release action creates the tag if it is missing). Nothing compares HEAD with the
  tag, and `inputs.tag` is pasted raw into the script. A "re-run for vX" after `main` moved ships `main`
  as vX, and winget then pins that zip's hash. Plaus only because the checkout default and the release
  action's tag creation are third-party behaviour.
- **R19** - the installer and MSIX builds stamp the current time and build the working tree, with no
  clean-tree or HEAD-equals-tag check; `release.ps1` runs the installer build without `-Stamp` and then
  uploads `setup-<tag version>.exe`, a file that does not exist.
- **R22** - `build-msix.ps1 -IdentityName` defaults to the winget id, not the frozen MSIX identity
  `SZA.Doc-HTML-Translate`, and the checklist passes a placeholder.
- **R23** - release notes are ranged with `git describe --tags`, which also matches the extension's
  `ext-cws-` / `ext-edge-` tags, so the notes leave out commits.
- **R14** - `release.ps1 -Version` is not validated against `YY.MMDD.HHmm`.
- **R24, R25** - the release action is pinned by a movable tag with `contents: write`; the extension
  publish workflows can be dispatched from any branch and have no `permissions:` block.
- **R36** - the tracked `winget/manifests/` archive inside `winget/` breaks the documented
  `winget install --manifest winget` gate.

## 2. Goals

1. A GitHub Release is built only from the tag it names: dispatch removed, or checked out at the tag
   with a HEAD-equals-tag check; the input validated and passed through the environment.
2. Local installer and MSIX builds take the version from the tag and refuse a dirty or mismatched tree.
3. The MSIX identity defaults to the frozen anchor; the checklist passes no placeholder.
4. The release-notes range matches `v*` tags only.
5. Third-party actions are pinned by commit; each workflow declares least permissions.
6. The winget gate works on the folder the docs name.

## 3. Constraints

- Frozen anchors do not move (winget id, MSIX identity, Inno AppId, extension ids).
- Nothing is proven by publishing: workflow changes are checked by reading and a linter, scripts by
  scratch-tree tests and `-WhatIf`.

## 4. Acceptance

- A dispatch of the release workflow cannot publish a tree other than the tag's (shown by the workflow
  text and a reviewed dry run).
- `build-msix.ps1` with no `-IdentityName` produces `SZA.Doc-HTML-Translate`.

## 5. What was done

| Finding | Change |
| --- | --- |
| R18 | `release.yml`: the tag is resolved and validated (`^vYY.MMDD.HHmm$` plus month/day/hour/minute ranges) **before** the checkout; the dispatch input reaches the script only as `$env:INPUT_TAG`. The checkout takes `ref: refs/tags/<tag>`, so a missing tag fails there and the release step never creates one; a new step stops unless `HEAD` is the tag's commit and the checkout is clean. A dispatch runs only from `main` (the job's `if:`), and the concurrency group is the tag, not the branch. |
| R19 | `build-installer.ps1 -Tag` and `build-msix.ps1 -Tag`: version from the tag, refusal unless `HEAD` is the tag's commit and `git status --porcelain` is empty, and a second check after the build (the icon step rewrites tracked files). `-Tag` and `-Stamp` are exclusive. `release.ps1` now prints `build-installer.ps1 -Tag <tag>`, so the uploaded `setup-<ver>.exe` is the file the build writes. |
| R22 | `build-msix.ps1 -IdentityName` defaults to `SZA.Doc-HTML-Translate`; an unsigned build without `-Tag` is refused; the checklist, `DEV/RELEASE.md`, `/release` and `msix/README.md` pass no identity. `reinstall.ps1` keeps its own test name on purpose, so a self-signed install never collides with the Store copy. |
| R23 | Release notes range: `git describe --tags --abbrev=0 --match 'v*'`. |
| R14 | `release.ps1 -Version` must be a real `YY.MMDD.HHmm` stamp, else exit 2 before the checklist. |
| R24, R25 | Every third-party action pinned by commit (checkout v4.4.0, setup-go v5.6.0, setup-node v4.4.0, action-gh-release v2.6.2 - each the commit its moving major pointed at on 2026-09-26, so behaviour is unchanged). `release.yml`: `contents: read` at the top, `contents: write` only on the job. The extension workflows: `contents: read`, and the job runs only on an `ext-cws-v*` / `ext-edge-v*` ref, so a dispatch from a branch is skipped. |
| R36 | The per-version archive moved `winget/manifests/` -> `DEV/winget-history/manifests/`; `winget/` is flat again and `DEV/RELEASE.md` says why it must stay so. |

Evidence (2026-09-26):

- `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.7` on the three workflows: exit 0, no output.
- Dry run (`temp/t37/dryrun.ps1`, gitignored): each `run: |` block of `release.yml` extracted verbatim and run
  with the runner's environment. Resolve step: a push of `v26.0912.2026` and a dispatch of `' v26.0912.2026 '`
  pass with the old outputs (`major=26, minor=9, patch=12, build=2026`); a dispatch of `v1'; Write-Host PWNED; '`,
  of `ext-cws-v1.2`, a push of `v26.1399.2500` and a push from `refs/heads/main` all fail. Verify step: in this
  clone (HEAD past the tag) exit 1; in a worktree at `v26.0912.2026` exit 0. Notes range for `v26.0912.2026`:
  the old `describe` stopped at `ext-cws-v26.0815` (8 commits), `--match 'v*'` at `v26.0815.2020` (9) - R23 was
  losing a commit. Verdict `dry run: PASS`, exit 0.
- Refusals in this clone (HEAD past the tag, dirty tree), each exit 1 with its message: `build-msix.ps1` with no
  `-Tag` and no `-SelfSign`, a malformed tag, `-Tag` with `-Stamp`, HEAD not the tag, a missing tag; the same
  for `build-installer.ps1`. `release.ps1 -Version 26.1340.2500` exits 2.
- Happy path in a scratch clone of the working tree (local tag `v26.0926.1300`, origin removed, never pushed):
  `build-msix.ps1 -Tag v26.0926.1300` exit 0 ->
  `SZA.Doc-HTML-Translate_26.926.1300.0_x64.msix`, manifest `Name="SZA.Doc-HTML-Translate"`,
  `Version="26.926.1300.0"`, clone still clean. `build-installer.ps1 -Tag v26.0926.1300` passed its tree checks
  and stamped `26.0926.1300`, then failed in the 386 `go build` - see below; its post-build check did not run.

## 6. Found on the way

- **The installer's 386 build does not link on this machine.** With `goversioninfo` v1.7.0 the x86 resource
  fails: `resource.syso: unknown relocation type 3` (amd64 links). The `goversioninfo` on `PATH` is v1.4.1,
  which cannot read the three-ICO `IconPath` at all (`open ../../assets/doc-html-translate.ico,..: The system
  cannot find the path specified`), so `build-msix.ps1` and `build-installer.ps1` fail with it before any of
  this ticket's code matters; `msix/README.md` already asks for v1.5.0+. Independent of this ticket (the build
  code is unchanged), unfiled - it blocks the next release's setup.exe.
