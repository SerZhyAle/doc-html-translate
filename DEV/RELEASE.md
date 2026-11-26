# Build vs Release ("сборка" vs "релиз")

Two distinct flows. The whole point of separating them: a **build** is local and free, so
asking for a "test build" never costs a paid GitHub Actions run; a **release** is the
published path, with a single checklist so no step is forgotten.

| | Build ("сборка") | Release ("релиз") |
|---|---|---|
| Goal | compile exes, test locally, commit | update docs/site, build on GitHub, publish everywhere |
| Cost | free, local only | paid CI minutes + public store/index submissions |
| Touches GitHub/CI? | **no** | yes (tags trigger paid workflows) |
| Entry point | `scripts/build-local.ps1` | `scripts/release.ps1` (checklist only) |
| Reversible? | yes (local commit) | partly - published artifacts are public |

> **Rule for agents and humans:** "сборка"/"build" means the local flow only. Never push a
> tag, submit to winget, upload to the Store, or publish the extension unless the request is
> explicitly a "релиз"/"release".

---

## Build ("сборка") - local, free

```powershell
./scripts/build-local.ps1 -Message "fix: ..."            # build CLI + build UI + fast gate tier + commit
./scripts/build-local.ps1 -NoCommit                      # build-only smoke test, no commit
./scripts/build-local.ps1 -Release -Message "docs: .."   # the same with the full gate - the release pre-flight
```

Steps it runs:

1. `scripts/build.ps1` - `doc-html-translate.exe` (CLI); asserts the exe carries the stamp it was built with
2. `scripts/build-ui.ps1` - `doc-html-ui.exe` (GUI); same assertion
3. The gate. By default `scripts/check.ps1 -Fast` - the fast tier, the gate children recorded at 5 s or less
   in `configs/check-placement.jsonl` (`golangci-lint` + `typos` + parity drift + documentation registry +
   site addresses + release notes + security posture + help + settings inventory), run at once; between
   releases nothing slower runs (canon `TESTING_AND_QA` section 2), and a fast run writes no gate evidence.
   With `-Release` it is the full gate, `scripts/check.ps1` - the fast tier plus the release tier (`go test`,
   about 3.5 minutes, and the extension's `node --test`). Either ends in one verdict line; the commit
   goes on only on `check: PASS` (exit 0) or `check: PASS WITH ADVISORIES` (exit 3) - `COULD NOT VERIFY`
   (exit 2) stops it like a failure. The full gate also records `temp/logs/gate-evidence.json`: the plan, every
   child's verdict, and the tree hash, taken before and after the run (a mismatch voids it). The full gate runs
   `help-check` with `-Release`: an in-app help set older than its inputs **fails** there (the fast tier only
   advises). CONTEXT-HELP rule 18 lets the screenshots wait for the release (dated exception, ticket E0125), so the
   pre-flight regenerates them before the gate: `node tools/helpgen/generate.mjs` (builds the GUI, about two
   minutes; it rewrites tracked files, so it runs before the commit), then `node tools/helpgen/generate.mjs --check`.
4. commit on the **current** branch + append to `DEV/COMMIT_LOG.md`

The builds run first because they rewrite tracked files (the two `build/*.exe`, the generated icons):
the gate then reads the tree the commit holds, so the evidence matches HEAD. `DEV/COMMIT_LOG.md` is
appended after the commit (it needs the hash) and is the one path `release.ps1` lets differ.

No tags, no push, no CI. `scripts/commit_after_build.ps1` is a deprecated shim that delegates here.

Optional real-install test of the Store artifact (still local, still free):

```powershell
./msix/build-msix.ps1 -SelfSign
```

Optional universal installer - a single `setup.exe` for x86 + x64, per-user, no admin (local, free):

```powershell
./scripts/build-installer.ps1        # -> dist/doc-html-translate-setup-<ver>.exe (needs Inno Setup / ISCC)
```

---

## Release ("релиз") - published, paid

`scripts/release.ps1` **prints this checklist and runs nothing** - it only reads git/file state
to fill in the current version and commands. Run it to get the exact commands, then execute each
step by hand. `[PAID]` = uses paid GitHub Actions minutes; `[PUBLIC]` = publishes to a store/index.

0. **Preflight** (free) - clean tree, green full gate: `./scripts/build-local.ps1 -Release -Message "..."`. This
   is where the release tier (the Go suite and the extension suite) runs - between releases only the fast
   tier does. The last commit must go through `build-local.ps1 -Release` (it re-runs the full gate): `release.ps1` prints **gate evidence** in its
   header and shows it as BLOCKED unless the last `check.ps1` ran the full default plan (a `-Plan` subset
   never counts, a `-Fast` run writes no evidence), every child passed, and it read exactly HEAD's tree. That gate
   includes the documentation registry (`scripts/doc-registry.ps1`) and the security posture consistency
   check (`scripts/security-posture.ps1`), so a stale sitemap or a privacy text that disagrees with the
   manifests blocks here.
   The header also prints the **contract gate** (`scripts/contract-gate.ps1`, read-only): every pointer in
   `docs/contracts/` against the catalog's registry - a current row verified since the last release, the
   pointer's version equal to the catalog's or behind it with a reason. Each open exception and its
   deviation produces a WARN; exception deadlines are not required. PASS and WARN let the release go;
   FAIL blocks it, and so does UNVERIFIED,
   which is what a clone without the catalog gets - a release is cut on the machine that has it.
   The header also prints the **store preflight** (the sza plugin's shared `tools/Store-Preflight.ps1`, WINDOWS-STORE 0.4
   section D, ticket E0132 Part 2; read-only - the only Store call it can make is `msstore submission get`): PASS when
   the last submission is published with none pending, the price is Free and the package version about to be built is
   above every live one; BLOCKED on a pending submission, a version not above the live one, a price that is not Free or a
   state it cannot read; MANUAL - never a block - when the channel is unconfigured, the credential is missing or
   expired or the token is refused (the Store step is then manual and the release goes on). `-StoreCredentialExpiry
   yyyy-MM-dd` passes the expiry from the owner's expiry list. `release.ps1` exits 1 while any of the three lines blocks
   the tag.
1. **Docs & site** (free) - update README.md, `docs.html` / `docs.ru.html` / `docs.uk.html`,
   `index.html`, `extension.html`, `extension/store/LISTING.md`, `extension/README.md`,
   `DEV/CHANGELOG.md`, and the `What's new` (`@@ReleaseNotes`) of **all 13 Store copy decks**
   `tools/store/listing/*.txt` - the rule-16 accumulated bullets, en/ru/uk authored and the other ten
   machine-translated, folded to 1500 characters, all 13 in the same edit (step 4 sends them unchanged);
   add this release's record (en, ru, uk, the same bullets) to `docs/release-notes.json`
   and run `./scripts/release-notes.ps1 -Render` - the site's release-notes page is assembled from it at
   the release, never per ticket, and the check fails on a tag with no record; commit via `build-local.ps1 -Release`.
2. **GitHub Release - app** `[PAID]` - push a `v*` tag → `.github/workflows/release.yml` builds
   the exes and creates the GitHub Release:
   `git tag -a v<ver> -m "Release v<ver>"; git push origin v<ver>`. Push only while the gate-evidence line
   is green - the tag workflow runs no test, so that line is the only link between the tested tree and the
   shipped binaries - and the contract-gate line reads PASS or WARN. Afterwards (free): `gh release download v<ver> -p "*.exe" -D temp/release-<ver>` and
   `./scripts/verify-exe-version.ps1 -Path (Get-ChildItem temp/release-<ver>/*.exe).FullName -Expect <ver>`.
   **Then attach the universal installer** (CI does not build it): `./scripts/build-installer.ps1 -Tag v<ver>`,
   write its sidecar (`$h = (Get-FileHash -Algorithm SHA256 -LiteralPath "dist/doc-html-translate-setup-<ver>.exe").Hash; "$h  doc-html-translate-setup-<ver>.exe" | Set-Content -LiteralPath "dist/doc-html-translate-setup-<ver>.exe.sha256" -Encoding ascii`),
   then `gh release upload v<ver> dist/doc-html-translate-setup-<ver>.exe dist/doc-html-translate-setup-<ver>.exe.sha256`, and record both in the release ledger:
   `./scripts/release-ledger.ps1 publish -Key github -Version <ver> -State live -Ref v<ver> -Url https://github.com/SerZhyAle/doc-html-translate/releases/tag/v<ver>` and
   `./scripts/release-ledger.ps1 confirm -Key installer -Version <ver> -State live -Ref v<ver> -Url https://github.com/SerZhyAle/doc-html-translate/releases/tag/v<ver>`. `-Tag` takes the version from
   the tag and refuses to build unless HEAD is the tag's commit and the working tree is clean (and fails
   if the build leaves it dirty), so the installer holds the tree the GitHub Release was built from.

   The workflow builds only the tree of the tag it names. A `workflow_dispatch` re-run takes an
   **existing** tag, must be started from `main`, checks that tag out and stops unless HEAD is its commit -
   it never creates a tag. Third-party actions are pinned by commit; bump a pin by resolving the new
   release's commit (`git ls-remote --tags https://github.com/<owner>/<action>`), not by a moving `@vN`.
3. **winget** `[PUBLIC]` - after the release exists, download its zip and calculate its SHA256.
   Re-stamp all 15 flat `winget/*.yaml` files before validation: `PackageVersion` in each file,
   `InstallerUrl` and `InstallerSha256` in the installer file, and `ReleaseNotesUrl` in the default
   locale. Run `winget validate --manifest winget`, then **always local-install-test the new manifest**
   with `winget install --manifest winget` (one-time: `winget settings --enable
   LocalManifestFiles`) - it downloads the release zip and verifies the SHA256 end-to-end, the
   single best gate (`winget validate` only checks schema, not the hash/URL). `winget/` must stay flat -
   `--manifest` rejects a folder with subdirectories - so the per-version copies of past submissions
   live in `DEV/winget-history/manifests/`, not under `winget/`. `wingetcreate update --out <dir>` writes
   a `manifests/s/..` tree under `<dir>`, so give it `--out DEV/winget-history`, never `--out winget`. Then submit:
   version-only bump = `wingetcreate update SerZhyAle.DocHtmlTranslate --version <ver> --urls
   <zip-url> --submit`; **to also change the description/tags, edit `winget/` and `wingetcreate
   submit winget`** (`update` copies the old metadata forward). Sign the CLA on the PR if
   prompted. **Then replace the auto-generated PR body** (`wingetcreate` leaves the empty template
   with unchecked boxes): `gh pr edit <PR#> --repo microsoft/winget-pkgs --body-file <notes>` with a
   real description (version, upstream release URL, "SHA256 verified via local `winget install`")
   and the manifest checklist boxes ticked - moderators triage faster with a filled-in PR. Record in the
   release ledger: `./scripts/release-ledger.ps1 publish -Key winget -Version <ver> -State submitted -Ref PR#<PR#>`. See
   [docs/how-i-posted-this-project-to-winget.md](../docs/how-i-posted-this-project-to-winget.md).
4. **Windows Store (MSIX)** `[PUBLIC]` - the agent-run submission channel (ticket E0156, WINDOWS-STORE
   section D, rules 22 to 26): `./tools/store/submit-submission.ps1 -Tag v<ver> -Commit`. The step
   builds the unsigned package from the tag itself (it refuses unless HEAD is the tag with a clean
   tree, and it does not run while the installer build does - both rewrite `cmd/*/resource.syso`),
   re-runs the read-only **store preflight** in the same run (the header line of item 0 stays the
   check before the tag; nothing is written to the Store before the tag is pushed), refreshes the
   13 ReleaseNotes from the step-1 decks, proves the draft read-back, and commits it into
   certification behind `-Commit` - the one outward act under the owner's standing pre-authorization
   of 2026-11-17; the tag, winget, Chrome and Edge steps keep their own confirmations. Without
   `-Commit` the same command only prepares: it leaves the draft in the Store and prints its identity
   and the console address. Give `-CredentialExpiry yyyy-MM-dd` when the owner's expiry list has one.
   Channel unavailable (CLI missing or unconfigured, credential missing or expired, token refused)
   exits 2 into the manual path: the package and the listing import folder are built for a hand
   upload in Partner Center and the release goes on. Every run appends its record to
   `DEV/release-verdicts/<msixVersion>.store-submission.jsonl` (rule 26) and one publish event to the
   release ledger (`DEV/releases/history.jsonl` + `RELEASES.md`, via `scripts/release-ledger.ps1`) - no
   account identifiers or secrets - so the publish state is no longer typed by hand. Nobody runs `msstore`
   by hand in this flow; this channel is the one pre-authorized machine write. See [../msix/README.md](../msix/README.md).
5. **Chrome / Edge extension** `[PAID]` `[PUBLIC]` - Chrome and Edge publish **independently**
   (separate tags, separate runs). Before the tags: `node extension/build.mjs record-version`, then
   `./scripts/release-notes.ps1 -Render`, and commit; the site shows that version beside the store buttons.
   At the next authorized Chrome release, upload `extension/store/chrome-promo-small.png` as the small promo tile in the Chrome developer dashboard.
   Push `ext-cws-v<version>` → Chrome (`.github/workflows/publish-cws.yml`) or `ext-edge-v<version>` → Edge
   (`publish-edge.yml`), e.g. `git tag ext-cws-v<version>; git push origin ext-cws-v<version>`, and record each
   submission in the release ledger (`./scripts/release-ledger.ps1 publish -Key chrome|edge -Version <version> -State submitted -Ref <tag>`). The build
   stamps the version the tag names and stops on any other. A manual
   re-run from the Actions tab must pick that tag as its ref; started from a branch, the job is skipped.
   See [../extension/PUBLISHING.md](../extension/PUBLISHING.md).
6. **Verify** - `gh release view v<ver>`, `winget search SerZhyAle.DocHtmlTranslate` (≈30-60 min
   after the winget PR merges), confirm the Store and extension dashboards show the new version, and run
   `./scripts/release-ledger.ps1 probe` (`a rs probe`) to update [DEV/releases/RELEASES.md](releases/RELEASES.md).
   **Then, once per release and never per edit, tell search engines the site changed**: resubmit
   `https://serzhyale.github.io/doc-html-translate/sitemap.xml` in Google Search Console and Bing
   Webmaster Tools after the Pages build of the release commit is live.

Not every release needs every target: a code-only release may skip the extension; an extension-only
release uses only steps 0, 1 and 5. The version stamp format is `yy.MMdd.HHmm`; app tags are
`v<stamp>`, extension tags are `ext-cws-v<version>` (Chrome) / `ext-edge-v<version>` (Edge), published independently; the extension's version is the same stamp with the leading zeros removed (`yy.MDD.Hmm`).
