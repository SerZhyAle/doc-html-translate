<#
.SYNOPSIS
  "Релиз" (release) - prints the complete, ordered release checklist. RUNS NOTHING.

.DESCRIPTION
  This is intentionally a checklist-only helper (see DEV/RELEASE.md for the
  canonical definition). It performs ONLY read-only git/file inspection to fill
  in the current state, then prints every release step with the exact command to
  run by hand. It NEVER pushes a tag, submits to winget, uploads to the Store, or
  publishes the extension - so running it can never trigger a paid GitHub CI run
  or an external publish. You execute each step yourself.

  Steps that cost money / are public are marked [PAID] / [PUBLIC].

.PARAMETER Version
  Optional version stamp (yy.MMdd.HHmm) to inline into the printed commands.
  Default: the suggested stamp computed from the current time.

.EXAMPLE
  ./scripts/release.ps1
#>
param(
    [string]$Version
)

$ErrorActionPreference = "Stop"

function Get-GitValue([string[]]$GitArgs) {
    try {
        $out = & git @GitArgs 2>$null
        if ($LASTEXITCODE -ne 0) { return "" }
        return ($out | Out-String).Trim()
    } catch { return "" }
}

# ── read-only state ──────────────────────────────────────────
$branch   = Get-GitValue @("branch", "--show-current")
$dirty    = Get-GitValue @("status", "--porcelain")
$lastVer  = Get-GitValue @("describe", "--tags", "--abbrev=0", "--match", "v*")
$lastExt  = Get-GitValue @("describe", "--tags", "--abbrev=0", "--match", "ext-*-v*")

if (-not $Version) {
    $now = Get-Date
    $Version = "{0}.{1:D2}{2:D2}.{3:D4}" -f [int]$now.ToString('yy'), [int]$now.ToString('MM'), [int]$now.ToString('dd'), [int]$now.ToString('HHmm')
}
$tag = "v$Version"

$extVer = ""
$extPkg = "extension/package.json"
if (Test-Path $extPkg) {
    try { $extVer = (Get-Content -LiteralPath $extPkg -Raw | ConvertFrom-Json).version } catch { $extVer = "" }
}

# ── report ───────────────────────────────────────────────────
Write-Host ""
Write-Host "================ RELEASE CHECKLIST (nothing is executed) ================" -ForegroundColor Cyan
Write-Host ""
Write-Host "  branch          : $branch"
Write-Host ("  working tree    : {0}" -f $(if ($dirty) { "DIRTY - commit/clean first (run scripts/build-local.ps1)" } else { "clean" }))
Write-Host "  last app tag    : $(if ($lastVer) { $lastVer } else { '(none)' })"
Write-Host "  last ext tag    : $(if ($lastExt) { $lastExt } else { '(none)' })"
Write-Host "  app version now : $Version   ->  tag $tag"
Write-Host "  ext version now : $(if ($extVer) { $extVer } else { '(unknown)' })   (bump via: cd extension; npm run version:bump)"
Write-Host ""
Write-Host "  Legend: [PAID] uses paid GitHub Actions minutes   [PUBLIC] publishes to a store/index" -ForegroundColor DarkGray
Write-Host ""

function Step([string]$n, [string]$title) { Write-Host "[$n] $title" -ForegroundColor Yellow }
function Cmd([string]$c)                  { Write-Host "      $c" -ForegroundColor Gray }
function Note([string]$c)                 { Write-Host "      - $c" -ForegroundColor DarkGray }

Step "0" "Preflight (local, free)"
Note "Working tree must be clean and all checks green."
Cmd  "./scripts/build-local.ps1 -Message ""..."""
Note "Optional real-install smoke test of the Store package:"
Cmd  "./msix/build-msix.ps1 -SelfSign"
Write-Host ""

Step "1" "Docs & site (local commit, free)"
Note "Update versioned/dated content, then commit:"
Note "README.md, docs.html / docs.ru.html / docs.uk.html, index.html, extension.html,"
Note "extension/store/LISTING.md, extension/README.md, DEV/CHANGELOG.md"
Cmd  "./scripts/build-local.ps1 -Message ""docs: release $Version"""
Write-Host ""

Step "2" "GitHub Release - app binaries  [PAID]"
Note "Pushing a v* tag triggers .github/workflows/release.yml (builds exes + GitHub Release)."
Cmd  "git tag -a $tag -m ""Release $tag"""
Cmd  "git push origin $tag"
Cmd  "gh run watch   # or: gh release view $tag --json assets"
Note "Attach the universal installer (CI does not build it): build locally, then upload to the release:"
Cmd  "./scripts/build-installer.ps1"
Cmd  "gh release upload $tag dist/doc-html-translate-setup-$Version.exe"
Write-Host ""

Step "3" "winget (Microsoft community index)  [PUBLIC]"
Note "Needs the GitHub Release from step 2 to exist (stable URL + SHA256)."
Note "ALWAYS local-install-test the manifest first (downloads the zip, verifies SHA256 - the best gate):"
Cmd  "winget install --manifest winget   # one-time: winget settings --enable LocalManifestFiles"
Cmd  "wingetcreate update SerZhyAle.DocHtmlTranslate --version $Version ``"
Cmd  "  --urls https://github.com/SerZhyAle/doc-html-translate/releases/download/$tag/doc-html-translate-$Version-windows-x64.zip ``"
Cmd  "  --submit"
Note "To ALSO change description/tags: edit winget/ then 'wingetcreate submit winget' (update copies old metadata forward)."
Note "Then sign the CLA on the PR if prompted: gh pr comment <PR#> --repo microsoft/winget-pkgs --body ""@microsoft-github-policy-service agree"""
Note "Replace the empty auto-generated PR body (wingetcreate leaves an unchecked template):"
Cmd  "gh pr edit <PR#> --repo microsoft/winget-pkgs --body-file <notes>   # version, release URL, 'SHA256 verified via winget install', tick the checklist"
Note "Details: docs/how-i-posted-this-project-to-winget.md"
Write-Host ""

Step "4" "Windows Store - MSIX  [PUBLIC]"
Note "Build the UNSIGNED package (Microsoft re-signs at certification), then upload by hand."
Cmd  "./msix/build-msix.ps1 -IdentityName ""<Package/Identity/Name from Partner Center>"""
Note "Upload msix/out/*.msix in Partner Center (no API for create/listing). Details: msix/README.md"
Write-Host ""

Step "5" "Chrome / Edge extension  [PAID] [PUBLIC]"
Note "Chrome and Edge publish INDEPENDENTLY: separate tags, separate build-time versions."
Note "Chrome -> ext-cws-v* (publish-cws.yml); Edge -> ext-edge-v* (publish-edge.yml). Each CI run does its own npm run build (fresh stamp)."
Cmd  "git tag ext-cws-v<label>; git push origin ext-cws-v<label>    # Chrome only"
Cmd  "git tag ext-edge-v<label>; git push origin ext-edge-v<label>  # Edge only"
Note "Details: extension/PUBLISHING.md"
Write-Host ""

Step "6" "Verify"
Cmd  "gh release view $tag --json assets"
Cmd  "winget search SerZhyAle.DocHtmlTranslate   # appears ~30-60 min after winget PR merge"
Note "Confirm Store listing updated and extension live in the Chrome/Edge dashboards."
Write-Host ""
Write-Host "========================================================================" -ForegroundColor Cyan
Write-Host "Nothing above was executed. Copy a command to run it. Full doc: DEV/RELEASE.md" -ForegroundColor Green

# Read-only git probes above (e.g. `git describe` with no matching tag) leave a
# non-zero $LASTEXITCODE; this helper only printed a checklist, so exit clean.
exit 0
