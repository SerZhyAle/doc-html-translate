# Release — "Релиз" (published, paid)

> **GLOBAL DIRECTIVES:**
> 1. Dry technical prose, no filler.
> 2. **Nothing paid or public happens without explicit per-step confirmation.** Pushing a
>    `v*` / `ext-v*` tag costs paid CI; winget / Store / extension are public and partly
>    irreversible. Confirm before EACH `[PAID]` / `[PUBLIC]` step. Never batch them silently.
> 3. The deliverable that must not be lost: an updated **"What's new in version XXX"**.
> 4. Typography in all content/docs/commits: short hyphens, Russian **ё**, ".." not "..." (CLAUDE.md).
> 5. Canonical checklist: [DEV/RELEASE.md](../../DEV/RELEASE.md). This skill drives it; do not duplicate it.

Prepare and execute a full **release** ("релиз") without losing a step: curate the version's
"What's new", update docs and site, then publish to GitHub, winget, the Microsoft Store and the
Chrome/Edge extension — each step gated by confirmation.

## Usage

```text
/release [version stamp, or which targets to ship]
```

- `/release` — full release of the app (all targets)
- `/release 26.0627.1604` — pin the version stamp
- `/release extension only` — ship only the Chrome/Edge extension (steps 0, 1, 5)
- `/release no store` — skip the Microsoft Store step

## When NOT to use

- You just want to compile/test/commit locally → `/build` (free, nothing published).
- The working tree is dirty or the gate is red → finish with `/build` first.

## Process

**Step A — Print the live checklist.** Run the read-only helper to capture current state (branch,
dirty tree, last `v*`/`ext-v*` tags, current versions) and the exact commands:

```powershell
./scripts/release.ps1            # prints only, executes nothing
./scripts/release-state.ps1 -Version <ver>   # seed/print the 5-channel publish-state (a rs)
```

Track publish-state in [DEV/RELEASE_STATE.md](../../DEV/RELEASE_STATE.md), not in prose - after each
`[PUBLIC]` step below, record it (`a rs -Channel <name> -Status <pending|submitted|live|blocked>`), so a
later session reads the file instead of guessing.

**Step B — Decide scope & version.** Confirm with the user: version stamp (`yy.MMdd.HHmm`, default
= now), and which targets ship (GitHub / winget / Store / extension). Not every release needs all.

**Step C — Curate "What's new in version XXX".** This is the core deliverable:
1. Collect changes since the last tag: `git log <lastTag>..HEAD --pretty="- %s"`.
2. Reconcile with [DEV/CHANGELOG.md](../../DEV/CHANGELOG.md) — every shipped change has an entry; add
   any missing rows in project typography.
3. The GitHub Release body is auto-generated from commit **subjects** between tags
   (`.github/workflows/release.yml`). Verify those subjects read as user-facing "What's new"; if any
   are unclear, note it — the cleanup belongs in `/build` (commit subjects), not here.
4. Draft the human "What's new in <version>" summary for the user to approve before any tag push.

**Step D — Preflight (free).** Confirm clean tree + green gate; commit anything outstanding:

```powershell
./scripts/build-local.ps1 -Message "..."
```

**Step E — Docs & site (free, local commit).** Update every surface via `/docs-sync` (manifest
[DEV/DOCS_SURFACES.md](../../DEV/DOCS_SURFACES.md) - README.md, the `docs.*` en/ru/uk trio, `index.html`,
`extension.html`, `extension/store/LISTING.md`, `extension/README.md`, `_locales/*/messages.json`) plus the
`DEV/CHANGELOG.md` "What's new" entries from Step C, then commit via `build-local.ps1`.

**Step F — GitHub Release `[PAID]`.** Confirm, then push the tag (triggers `release.yml`):

```powershell
git tag -a v<ver> -m "Release v<ver>"; git push origin v<ver>
gh run watch
```

Verify the auto-generated release notes match the approved "What's new"; edit the GitHub Release body
if they drifted.

**Step G — winget `[PUBLIC]`.** Needs the release from Step F. Confirm, then:

```powershell
wingetcreate update SerZhyAle.DocHtmlTranslate --version <ver> --urls <zip-url> --submit
```

Sign the CLA on the PR if prompted. Details: [docs/how-i-posted-this-project-to-winget.md](../../docs/how-i-posted-this-project-to-winget.md).

**Step H — Microsoft Store / MSIX `[PUBLIC]`.** Confirm, build the **unsigned** package, then upload
by hand in Partner Center (no API for create/listing):

```powershell
./msix/build-msix.ps1 -IdentityName "<Package/Identity/Name from Partner Center>"
```

Details: [msix/README.md](../../msix/README.md).

**Step I — Chrome / Edge extension `[PAID]` `[PUBLIC]`.** Chrome and Edge publish **independently** -
separate tags, separate build-time versions. Push `ext-cws-v*` for Chrome (triggers `publish-cws.yml`)
or `ext-edge-v*` for Edge (`publish-edge.yml`); each CI run does its own `npm run build`, which stamps
a fresh version, so no manual bump is needed:

```powershell
git tag ext-cws-v<label>; git push origin ext-cws-v<label>    # Chrome only
git tag ext-edge-v<label>; git push origin ext-edge-v<label>  # Edge only
```

Details: [extension/PUBLISHING.md](../../extension/PUBLISHING.md).

**Step J — Verify.** `gh release view v<ver> --json assets`; `winget search SerZhyAle.DocHtmlTranslate`
(≈30-60 min after the winget PR merges); confirm Store and Chrome/Edge dashboards show the new version.

**Step K — Report & record state.** Update [DEV/RELEASE_STATE.md](../../DEV/RELEASE_STATE.md) for every
channel (`a rs -Channel <name> -Status <live|submitted|blocked> -Ref <PR#/tag/id> -Note "..."`) so the
publish-state lives in a file, not in memory. Then report: what shipped to which targets, the version, the
published "What's new", and any target still pending review (winget PR, Store certification, Edge cert).
