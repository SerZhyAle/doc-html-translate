# Release - "Релиз" (published, paid)

> **GLOBAL DIRECTIVES:**
> 1. Dry technical prose, no filler.
> 2. **Nothing paid or public happens without explicit per-step confirmation.** Pushing a
>    `v*` / `ext-v*` tag costs paid CI; winget / Store / extension are public and partly
>    irreversible. Confirm before EACH `[PAID]` / `[PUBLIC]` step. Never batch them silently.
> 3. The deliverable that must not be lost: an updated **"What's new in version XXX"**.
> 4. Typography in all content/docs/commits: short hyphens, Russian **ё**, `..` not `...` (CLAUDE.md).
> 5. Canonical checklist: [DEV/RELEASE.md](../../DEV/RELEASE.md). This skill drives it; do not duplicate it.

Prepare and execute a full **release** ("релиз") without losing a step: curate the version's
"What's new", update docs and site, then publish to GitHub, winget, the Microsoft Store and the
Chrome/Edge extension - each step gated by confirmation.

## Usage

```text
/release [version stamp, or which targets to ship]
```

- `/release` - full release of the app (all targets)
- `/release 26.0627.1604` - pin the version stamp
- `/release extension only` - ship only the Chrome/Edge extension (steps 0, 1, 5)
- `/release no store` - skip the Microsoft Store step

## When NOT to use

- You just want to compile/test/commit locally → `/build` (free, nothing published).
- The working tree is dirty or the gate is red → finish with `/build` first.

## Process

**Step A - Print the live checklist.** Run the read-only helper to capture current state (branch,
dirty tree, last `v*`/`ext-v*` tags, current versions) and the exact commands:

```powershell
./scripts/release.ps1                # prints only, executes nothing
./scripts/release-ledger.ps1 show    # print the release ledger (a rs)
```

Track publish-state in the release ledger ([DEV/releases/RELEASES.md](../../DEV/releases/RELEASES.md)), not in prose - after each
`[PUBLIC]` step below, record it (`a rs publish -Key <channel> -Version <ver> -State <pending|submitted|live|blocked>`), so a
later session reads the ledger instead of guessing.

**Step B - Decide scope & version.** Confirm with the user: version stamp (`yy.MMdd.HHmm`, default
= now), and which targets ship (GitHub / winget / Store / extension). Not every release needs all.

**Step C - Curate "What's new in version XXX".** This is the core deliverable:
1. Collect changes since the last tag: `git log <lastTag>..HEAD --pretty="- %s"`.
2. Reconcile with [DEV/CHANGELOG.md](../../DEV/CHANGELOG.md) - every shipped change has an entry; add
   any missing rows in project typography.
3. The GitHub Release body is auto-generated from commit **subjects** between tags
   (`.github/workflows/release.yml`). Verify those subjects read as user-facing "What's new"; if any
   are unclear, note it - the cleanup belongs in `/build` (commit subjects), not here.
4. Draft the human "What's new in <version>" summary for the user to approve before any tag push.
5. Once approved, add it as the newest record of [docs/release-notes.json](../../docs/release-notes.json) (en, ru and
   uk, the same bullets, plain text, house typography) and run `./scripts/release-notes.ps1 -Render`: the site's
   release-notes page is assembled from that file at the release, never edited per ticket, and the check fails on
   an app tag with no record.

**Step D - Preflight (free).** This is where the full build and every test tier run - between releases only
the fast tier does (AGENTS.md "Build, Test, Lint"). Confirm clean tree + green full gate, and that the **gate
evidence** line in the Step A output is green (the last full `check.ps1` passed on HEAD's exact tree; a `-Fast`
run writes no evidence). Quote the **contract gate** line
of the same output - PASS, WARN, FAIL or UNVERIFIED (`scripts/contract-gate.ps1` against the catalog's
registry) - in the report to the user, with every WARN it names - and the **store preflight** line: PASS, MANUAL
(the Store channel is unavailable; the Store step is manual and the release goes on) or BLOCKED (a pending
submission, a version not above the live one, a price that is not Free). `release.ps1` exits 1 while any of the
three lines blocks the tag. Run the `RELEASE` items the tickets since the last tag recorded (for example
`node tools/uicheck/run.mjs` after a `ui.html` change, and `node tools/helpgen/generate.mjs --check` for the in-app help set - it builds the GUI and runs about two minutes; the screenshots wait for the release (CONTEXT-HELP rule 18 exception, ticket E0125) and the full gate's `help-check -Release` fails a stale set, so run `node tools/helpgen/generate.mjs` first when a window input moved and commit the set). Commit anything outstanding through the full gate:

```powershell
./scripts/build-local.ps1 -Release -Message "..."
```

**Step E - Docs & site (free, local commit).** Update every surface via `/docs-sync` (manifest
[DEV/DOCS_SURFACES.md](../../DEV/DOCS_SURFACES.md) - README.md, the `docs.*` en/ru/uk trio, `index.html`,
`extension.html`, `extension/store/LISTING.md`, `extension/README.md`, `_locales/*/messages.json`, and the
`release-notes.html` render from Step C) plus the
`DEV/CHANGELOG.md` "What's new" entries from Step C, then commit via `build-local.ps1 -Release`.

**Step F - GitHub Release `[PAID]`.** Re-run `./scripts/release.ps1` and **refuse to push the tag** unless it
exits 0: the gate-evidence line green, the contract gate at PASS or WARN and the store preflight not BLOCKED
(an extension-only release pushes no `v*` tag and is not held by that line). A contract-gate FAIL is fixed in
the catalog first (the contract change comes before the code, never after); UNVERIFIED means this machine
cannot see the catalog, and the release is cut where it can. Then confirm, and push the tag (triggers
`release.yml`):

```powershell
git tag -a v<ver> -m "Release v<ver>"; git push origin v<ver>
gh run watch
```

Verify the auto-generated release notes match the approved "What's new"; edit the GitHub Release body
if they drifted.

**Step G - winget `[PUBLIC]`.** Needs the release from Step F. Confirm, then:

```powershell
wingetcreate update SerZhyAle.DocHtmlTranslate --version <ver> --urls <zip-url> --submit
```

Sign the CLA on the PR if prompted. Details: [docs/how-i-posted-this-project-to-winget.md](../../docs/how-i-posted-this-project-to-winget.md).

**Step H - Microsoft Store / MSIX `[PUBLIC]`.** Run the agent-run submission channel (ticket E0156,
WINDOWS-STORE section D, rules 22 to 26). The commit into certification behind `-Commit` is the owner's
standing pre-authorization of 2026-11-17 - the one outward act exempt from global directive 2's
per-step confirmation; the tag, winget, Chrome and Edge steps keep their own confirmations:

```powershell
./tools/store/submit-submission.ps1 -Tag v<ver> -Commit
```

The step builds the unsigned package from the tag itself (`-Tag` refuses unless HEAD is the tag's
commit and the tree is clean), re-runs the read-only store preflight in the same run (the Step A
`store preflight` line stays the check before the tag: BLOCKED stops, MANUAL means the channel is
unavailable and this step stays manual), refreshes the 13 ReleaseNotes from the decks updated in
Step E (`tools/store/listing/*.txt`), proves the draft read-back, commits, and records the evidence
itself: the verdicts file `DEV/release-verdicts/<msixVersion>.store-submission.jsonl` (rule 26) and
one publish event to the release ledger (`DEV/releases/`) per run - no account identifiers or secrets. Without `-Commit` the same
command only prepares: it leaves the draft in the Store and prints its identity and the console
address. Give `-CredentialExpiry yyyy-MM-dd` when the owner's expiry list has one. Exit 2 is the
manual path, not a failure: the package and the listing import folder are built for a hand upload in
Partner Center and the release goes on. `msstore` is never run by hand in this flow; this channel is
the one pre-authorized machine write. Details: [msix/README.md](../../msix/README.md).

**Step I - Chrome / Edge extension `[PAID]` `[PUBLIC]`.** Chrome and Edge publish **independently** -
separate tags, separate runs. Record the version first, so the site shows it and the tags name it; the build
stamps the version its tag names and stops on any other, so no manual bump is needed:

```powershell
node extension/build.mjs record-version; ./scripts/release-notes.ps1 -Render   # then commit
git tag ext-cws-v<version>; git push origin ext-cws-v<version>    # Chrome only (publish-cws.yml)
git tag ext-edge-v<version>; git push origin ext-edge-v<version>  # Edge only (publish-edge.yml)
./scripts/release-ledger.ps1 publish -Key chrome -Version <version> -State submitted -Ref ext-cws-v<version>
./scripts/release-ledger.ps1 publish -Key edge -Version <version> -State submitted -Ref ext-edge-v<version>
```

Details: [extension/PUBLISHING.md](../../extension/PUBLISHING.md).

**Step J - Verify.** `gh release view v<ver> --json assets`; `winget search SerZhyAle.DocHtmlTranslate`
(≈30-60 min after the winget PR merges); confirm Store and Chrome/Edge dashboards show the new version,
and run `./scripts/release-ledger.ps1 probe` (`a rs probe`) to refresh the ledger.
Once the Pages build of the release commit is live, ask the user to resubmit `sitemap.xml` in Google Search
Console and Bing Webmaster Tools - once per release, never per edit (DEV/RELEASE.md step 6).

**Step K - Report & record state.** Ensure the release ledger ([DEV/releases/RELEASES.md](../../DEV/releases/RELEASES.md)) records every
channel (`a rs publish -Key <channel> -Version <ver> -State <live|submitted|blocked> -Ref <PR#/tag/id> -Note "..."`, or `confirm` for `installer`) so the
publish-state lives in the ledger, not in memory. Then report: what shipped to which targets, the version, the
published "What's new", and any target still pending review (winget PR, Store certification, Edge cert).
