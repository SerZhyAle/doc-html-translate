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
./scripts/build-local.ps1 -Message "fix: ..."   # gate + build CLI + build UI + commit
./scripts/build-local.ps1 -NoCommit             # build-only smoke test, no commit
```

Steps it runs:

1. `scripts/check.ps1` - `go test` + `golangci-lint` + `typos` (the quality gate)
2. `scripts/build.ps1` - `doc-html-translate.exe` (CLI)
3. `scripts/build-ui.ps1` - `doc-html-ui.exe` (GUI)
4. commit on the **current** branch + append to `DEV/COMMIT_LOG.md`

No tags, no push, no CI. `scripts/commit_after_build.ps1` is a deprecated shim that delegates here.

Optional real-install test of the Store artifact (still local, still free):

```powershell
./msix/build-msix.ps1 -SelfSign
```

---

## Release ("релиз") - published, paid

`scripts/release.ps1` **prints this checklist and runs nothing** - it only reads git/file state
to fill in the current version and commands. Run it to get the exact commands, then execute each
step by hand. `[PAID]` = uses paid GitHub Actions minutes; `[PUBLIC]` = publishes to a store/index.

0. **Preflight** (free) - clean tree, green gate: `./scripts/build-local.ps1 -Message "..."`.
1. **Docs & site** (free) - update README.md, `docs.html` / `docs.ru.html` / `docs.uk.html`,
   `index.html`, `extension.html`, `extension/store/LISTING.md`, `extension/README.md`,
   `DEV/CHANGELOG.md`; commit via `build-local.ps1`.
2. **GitHub Release - app** `[PAID]` - push a `v*` tag → `.github/workflows/release.yml` builds
   the exes and creates the GitHub Release:
   `git tag -a v<ver> -m "Release v<ver>"; git push origin v<ver>`.
3. **winget** `[PUBLIC]` - after the release exists. **Always local-install-test the manifest
   first** - `winget install --manifest winget` (one-time: `winget settings --enable
   LocalManifestFiles`) - it downloads the release zip and verifies the SHA256 end-to-end, the
   single best gate (`winget validate` only checks schema, not the hash/URL). Then submit:
   version-only bump = `wingetcreate update SerZhyAle.DocHtmlTranslate --version <ver> --urls
   <zip-url> --submit`; **to also change the description/tags, edit `winget/` and `wingetcreate
   submit winget`** (`update` copies the old metadata forward). Sign the CLA on the PR if
   prompted. See [docs/how-i-posted-this-project-to-winget.md](../docs/how-i-posted-this-project-to-winget.md).
4. **Windows Store (MSIX)** `[PUBLIC]` - build unsigned and upload by hand in Partner Center:
   `./msix/build-msix.ps1 -IdentityName "<name>"`. See [../msix/README.md](../msix/README.md).
5. **Chrome / Edge extension** `[PAID]` `[PUBLIC]` - Chrome and Edge publish **independently**
   (separate tags, separate build-time versions; each CI run does its own `npm run build`). Push
   `ext-cws-v*` → Chrome (`.github/workflows/publish-cws.yml`) or `ext-edge-v*` → Edge
   (`publish-edge.yml`), e.g. `git tag ext-cws-v<label>; git push origin ext-cws-v<label>`.
   See [../extension/PUBLISHING.md](../extension/PUBLISHING.md).
6. **Verify** - `gh release view v<ver>`, `winget search SerZhyAle.DocHtmlTranslate` (≈30-60 min
   after the winget PR merges), confirm the Store and extension dashboards show the new version.

Not every release needs every target: a code-only release may skip the extension; an extension-only
release uses only steps 0, 1 and 5. The version stamp format is `yy.MMdd.HHmm`; app tags are
`v<stamp>`, extension tags are `ext-cws-v<label>` (Chrome) / `ext-edge-v<label>` (Edge), published independently.
