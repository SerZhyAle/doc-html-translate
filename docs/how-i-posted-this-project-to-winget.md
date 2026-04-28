# How I posted doc-html-translate to winget

A walkthrough of getting **doc-html-translate** into the official Windows Package Manager, with every command, every blocker, and every fix — written so you can repeat it for any future release in about 15 minutes.

The end result: anyone on Windows can now run

```powershell
winget install SerZhyAle.DocHtmlTranslate
# or via moniker:
winget install doc-html-translate
```

and get both `doc-html-translate` and `doc-html-ui` on their `PATH`, with no admin opt-in, no manual download, no PowerShell flags.

---

## 1. What winget actually is

The Windows Package Manager (`winget`) ships with Windows 10 1809+ and Windows 11. It pulls its package catalog from a single curated GitHub repository: **[microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs)**. To get a package listed you submit a pull request adding three small YAML files describing your release. Microsoft moderators review and merge. Once merged, every winget client on the planet sees your package on the next index rebuild (~30–60 minutes).

There is no developer account, no signing certificate requirement, no annual fee — just a GitHub PR.

---

## 2. The decision before any code

This project ships **two** independent executables:

- `doc-html-translate.exe` — the main CLI (console app)
- `doc-html-ui.exe` — a small GUI launcher (compiled with `-H windowsgui`, no console window)

That ruled out `InstallerType: portable` (single-exe only). Three options:

1. **Single portable EXE** — only ships one binary.
2. **Portable zip** (`InstallerType: zip` + `NestedInstallerType: portable`) — ships both binaries, each registered as a separate command alias.
3. **Real installer** (Inno Setup / WiX MSI) — heavyweight for a CLI tool.

I picked **option 2**. Each binary becomes a `PortableCommandAlias`, so users get both `doc-html-translate` and `doc-html-ui` from any shell.

One thing that makes this project simpler than a typical zip release: `pdftotext.exe` is **embedded inside the binary** via Go's `//go:embed` directive (`internal/bundledtools/pdftotext.go`). The release zip only needs to contain the two `.exe` files — no bundled tools folder, no extra DLLs.

Calibre is an optional runtime dependency for MOBI/AZW3 conversion. It's noted in the locale manifest description but the user installs it separately; it doesn't ship with this package.

---

## 3. The release pipeline (GitHub Actions)

winget needs a **stable HTTPS download URL with a known SHA256**. The standard place for that is a GitHub Release attached to a tag.

**File: [`.github/workflows/release.yml`](../.github/workflows/release.yml)**

The key shape:

```yaml
name: Release
on:
  push:
    tags: ['v*']
  workflow_dispatch:
    inputs:
      tag:
        description: 'Tag to release (e.g. v26.0427.1430)'
        required: true

permissions:
  contents: write

jobs:
  release:
    runs-on: windows-latest
    defaults:
      run:
        shell: pwsh
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      # ... resolve $version from tag
      # ... go install goversioninfo
      # ... generate icon via scripts/generate-icon.ps1
      # ... goversioninfo + go build each cmd with -trimpath -s -w
      # ... for doc-html-ui add -H windowsgui to ldflags
      # ... Compress-Archive both exes + LICENSE + README.md
      # ... Get-FileHash -Algorithm SHA256 → write to .sha256 file
      - uses: softprops/action-gh-release@v2
        with:
          files: |
            dist/${{ steps.pkg.outputs.name }}
            dist/${{ steps.pkg.outputs.name }}.sha256
```

### Why not call the existing `.ps1` build scripts?

`scripts/build.ps1` and `scripts/build-ui.ps1` both end with a step that copies the built binary to `C:\GD\tc\SZA\_APP`. That path doesn't exist on a CI runner and would hard-fail the workflow. Rather than patch the scripts, the workflow calls `go build` directly with the same flags (`-trimpath`, `-s -w`, `-X main.Version=...`, `-H windowsgui` for the UI). Icon embedding is still done via `goversioninfo`; the workflow installs it with `go install`.

### Version format

The build scripts produce versions in `yy.MMdd.HHmm` format — e.g. `26.0427.1713`. Git tags use a `v` prefix: `v26.0427.1713`. The workflow parses the tag to recover the four integer components (yy, MM, dd, HHmm) needed to populate the `FixedFileInfo` fields in `versioninfo.generated.json`.

### Cutting the first release

```bash
git add .github/workflows/release.yml winget/
git commit -m "Add winget submission pipeline"
git push origin main
git tag -a v26.0427.1713 -m "Release v26.0427.1713 — first winget-ready build"
git push origin v26.0427.1713
```

The workflow ran, built in **1m56s**, and published the release:

```
https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.0427.1713
```

The SHA256 was `C22B9E0A517361D6B237BC16C91783ECBD30B50E6108A06A87DB0DC72EC2DC4F` (also written to `doc-html-translate-26.0427.1713-windows-x64.zip.sha256` alongside the zip).

---

## 4. The three manifest files

A winget package is described by a **manifest set** of three YAML files in one folder:

```
winget/
├── SerZhyAle.DocHtmlTranslate.yaml               # version manifest
├── SerZhyAle.DocHtmlTranslate.installer.yaml     # installer manifest
└── SerZhyAle.DocHtmlTranslate.locale.en-US.yaml  # default-locale manifest
```

These are kept committed in the repo so future releases only need a version bump.

### Version manifest (`SerZhyAle.DocHtmlTranslate.yaml`)

```yaml
# yaml-language-server: $schema=https://aka.ms/winget-manifest.version.1.12.0.schema.json

PackageIdentifier: SerZhyAle.DocHtmlTranslate
PackageVersion: "26.0427.1713"
DefaultLocale: en-US
ManifestType: version
ManifestVersion: 1.12.0
```

### Installer manifest (`SerZhyAle.DocHtmlTranslate.installer.yaml`)

```yaml
# yaml-language-server: $schema=https://aka.ms/winget-manifest.installer.1.12.0.schema.json

PackageIdentifier: SerZhyAle.DocHtmlTranslate
PackageVersion: "26.0427.1713"
InstallerType: zip
NestedInstallerType: portable
NestedInstallerFiles:
- RelativeFilePath: doc-html-translate.exe
  PortableCommandAlias: doc-html-translate
- RelativeFilePath: doc-html-ui.exe
  PortableCommandAlias: doc-html-ui
Installers:
- Architecture: x64
  InstallerUrl: https://github.com/SerZhyAle/doc-html-translate/releases/download/v26.0427.1713/doc-html-translate-26.0427.1713-windows-x64.zip
  InstallerSha256: C22B9E0A517361D6B237BC16C91783ECBD30B50E6108A06A87DB0DC72EC2DC4F
ManifestType: installer
ManifestVersion: 1.12.0
ReleaseDate: 2026-04-27
```

`NestedInstallerFiles` with multiple portable entries requires schema **1.12.0** — earlier schema versions don't support it.

### Locale manifest (`SerZhyAle.DocHtmlTranslate.locale.en-US.yaml`)

```yaml
# yaml-language-server: $schema=https://aka.ms/winget-manifest.defaultLocale.1.12.0.schema.json

PackageIdentifier: SerZhyAle.DocHtmlTranslate
PackageVersion: "26.0427.1713"
PackageLocale: en-US
Publisher: SZA
PublisherUrl: https://github.com/SerZhyAle
PublisherSupportUrl: https://github.com/SerZhyAle/doc-html-translate/issues
Author: Serhii Zhyhunenko
PackageName: doc-html-translate
PackageUrl: https://github.com/SerZhyAle/doc-html-translate
License: MIT
LicenseUrl: https://github.com/SerZhyAle/doc-html-translate/blob/main/LICENSE
Copyright: Copyright (c) 2026 SerZhyAle
ShortDescription: Convert EPUB, PDF, TXT, MD, FB2, RTF, HTML, MOBI and AZW3 to local HTML, with optional Google or Ollama translation.
Moniker: doc-html-translate
Tags:
- azw3
- cli
- convert
- ebook
- epub
- ...
ReleaseNotesUrl: https://github.com/SerZhyAle/doc-html-translate/releases/tag/v26.0427.1713
ManifestType: defaultLocale
ManifestVersion: 1.12.0
```

`Moniker: doc-html-translate` is what makes `winget install doc-html-translate` work as a shorthand.

---

## 5. Local validation

```powershell
winget validate --manifest p:\WINDOWS\EPUB_2_HTML\winget
```

Output:
```
Manifest validation succeeded.
```

The initial run before adding schema header comments produced:

```
Manifest validation succeeded with warnings.
Manifest Warning: Schema header not found. File: ...
```

Fix: add `# yaml-language-server: $schema=https://aka.ms/winget-manifest.<type>.1.12.0.schema.json` as the first line of each file. Subsequent validation: clean.

---

## 6. Local install test

```powershell
# In an elevated PowerShell (one-time):
winget settings --enable LocalManifestFiles

# Normal shell:
winget install --manifest p:\WINDOWS\EPUB_2_HTML\winget
```

Output:
```
Found doc-html-translate [SerZhyAle.DocHtmlTranslate] Version 26.0427.1713
Downloading https://github.com/SerZhyAle/doc-html-translate/releases/download/...
Successfully verified installer hash
Extracting archive...
Successfully extracted archive
Starting package install...
Command line alias added: "doc-html-translate"
Command line alias added: "doc-html-ui"
Successfully installed
```

Both aliases resolved. This is the single best gate before submitting — if it passes locally it will pass CI too.

---

## 7. Submitting the PR with `wingetcreate`

```powershell
wingetcreate submit p:\WINDOWS\EPUB_2_HTML\winget
```

Output:
```
Manifest validation succeeded: True
Submitting pull request for manifest...
Pull request can be found here: https://github.com/microsoft/winget-pkgs/pull/365602
```

`wingetcreate` forks `microsoft/winget-pkgs`, creates a branch, copies the manifests into `manifests/s/SerZhyAle/DocHtmlTranslate/26.0427.1713/`, and opens the PR.

### Blocker — leaked PAT

I pasted the GitHub PAT into a chat window to use it as `--token`. The PAT was immediately visible in transcript history. The right response: **revoke the token immediately** at https://github.com/settings/tokens. With `public_repo` scope only, the blast radius is push-access to your own public repos. The cleaner alternative: run `wingetcreate` without `--token` and let it open a browser device-code flow — the token never touches your shell or chat.

---

## 8. Signing the Microsoft CLA

The PR template includes a CLA checkbox. Instead of clicking through the CLA portal, you can agree directly with a PR comment:

```powershell
gh pr comment 365602 --repo microsoft/winget-pkgs --body "@microsoft-github-policy-service agree"
```

The `microsoft-github-policy-service` bot picks it up within a minute and marks `license/cla` as green.

---

## 9. The PR's automated validation

Within ~10 minutes of opening the PR, Azure Pipelines ran:

- **ManifestValidation** — schema check (same as `winget validate` locally).
- **InstallerValidation** — downloads the URL, verifies SHA256, sandbox install.
- **URLValidation** — HTTP status checks on all URLs.
- **DefenderScan** — antivirus scan of the binary.

All green. Labels added: `Azure-Pipeline-Passed`, `Validation-Completed`, `New-Package`.

Check status without opening a browser:

```powershell
gh pr view 365602 --repo microsoft/winget-pkgs --json state,labels,mergedAt
```

---

## 10. After merge

A moderator approved it at **2026-04-28 00:06 UTC** — about 9 hours after the PR was opened.

After the community index rebuilt (~30–60 min after merge):

```powershell
winget search SerZhyAle.DocHtmlTranslate
winget search doc-html-translate
winget install SerZhyAle.DocHtmlTranslate
```

All work from any Windows machine with no flags or admin opt-in.

---

## 11. The follow-up release flow

```bash
git tag -a v<new-version> -m "Release v<new-version>"
git push origin v<new-version>
# wait ~2 minutes for CI to publish the release
```

Get the new SHA from the release (it's printed in the workflow log and in the `.sha256` file):

```powershell
gh release view v<new-version> --json assets | ConvertFrom-Json
```

Update the four values across the three manifest files (`PackageVersion` ×3, `InstallerUrl`, `InstallerSha256`, `ReleaseNotesUrl`), validate, then:

```powershell
wingetcreate update SerZhyAle.DocHtmlTranslate `
  --version <new-version> `
  --urls https://github.com/SerZhyAle/doc-html-translate/releases/download/v<new-version>/doc-html-translate-<new-version>-windows-x64.zip `
  --submit
```

`wingetcreate update` fetches the new zip, computes SHA256 itself, copies fields forward from the latest manifest, and opens the PR. ~30 seconds of effort per release.

---

## 12. Lessons worth remembering

1. **`go:embed` makes packaging cleaner.** `pdftotext.exe` is embedded inside the binary — no extra files to include in the zip, no extraction step the user sees. The zip is just the two `.exe` files + LICENSE + README.
2. **Don't call local-deploy scripts from CI.** The `.ps1` build scripts both have a `Copy-Item … "C:\GD\tc\SZA\_APP"` step that fails on a CI runner. Reproduce the build logic inline in the workflow instead.
3. **The `goversioninfo` tool must be installed in CI.** It's a dev dependency that `go build` itself doesn't pull. Add `go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest` before the build steps.
4. **Don't paste PATs into chat.** Use device-code auth (`wingetcreate submit` without `--token`) so the token never leaves the browser flow. If you do use `--token`, revoke it at https://github.com/settings/tokens the instant the PR is open.
5. **Sign the CLA via `gh pr comment`.** `@microsoft-github-policy-service agree` is faster than navigating the CLA portal.
6. **`gh pr view --json` beats the browser for status checks.** Labels like `Validation-Completed` and `Moderator-Approved` are visible instantly from the terminal.
7. **Source-of-truth your manifests in the repo.** The `winget/` directory mirrors what's in the live PR. Future releases just need a version bump — no archaeology required.

---

## 13. Files in this repo that are part of the winget pipeline

- [`.github/workflows/release.yml`](../.github/workflows/release.yml) — builds + publishes the release zip on tag push.
- [`winget/SerZhyAle.DocHtmlTranslate.yaml`](../winget/SerZhyAle.DocHtmlTranslate.yaml) — version manifest.
- [`winget/SerZhyAle.DocHtmlTranslate.installer.yaml`](../winget/SerZhyAle.DocHtmlTranslate.installer.yaml) — installer manifest.
- [`winget/SerZhyAle.DocHtmlTranslate.locale.en-US.yaml`](../winget/SerZhyAle.DocHtmlTranslate.locale.en-US.yaml) — locale manifest.

Total elapsed from starting to write the workflow to PR merged: **~9 hours** (mostly waiting overnight for moderator review). Active work time: under 30 minutes.
