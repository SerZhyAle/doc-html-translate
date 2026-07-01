<#
.SYNOPSIS
  Build both binaries and copy them to the deploy folder C:\GD\tc\SZA\_APP.
  Local and free: no quality gate (unless -Gate), no commit, no push.

.DESCRIPTION
  Root-level convenience wrapper. From the repo root it runs:
    1. scripts/build.ps1     -> doc-html-translate.exe (CLI)
    2. scripts/build-ui.ps1  -> doc-html-ui.exe (UI, -H windowsgui)
  Each embeds the icon + a date-stamped version and copies its exe to
  C:\GD\tc\SZA\_APP (build.ps1 also copies DEV/private/google_api.key when present).

  This is the quick "compile and deploy" path. For the gated + committed flow
  ("сборка" / the /build skill) use scripts/build-local.ps1 instead; to publish
  ("релиз") use scripts/release.ps1.

.PARAMETER Gate
  Run the quality gate (test + lint + typo) before building.

.EXAMPLE
  ./build.ps1

.EXAMPLE
  ./build.ps1 -Gate
#>
param(
    [switch]$Gate
)

$ErrorActionPreference = "Stop"

# Sub-scripts use paths relative to the repo root; build.ps1 lives there.
Set-Location $PSScriptRoot

if ($Gate) {
    Write-Host "== gate (test + lint + typo) ==" -ForegroundColor Cyan
    ./scripts/check.ps1
}

Write-Host "== build CLI (doc-html-translate.exe) ==" -ForegroundColor Cyan
./scripts/build.ps1

Write-Host "== build UI (doc-html-ui.exe) ==" -ForegroundColor Cyan
./scripts/build-ui.ps1

Write-Host ""
Write-Host "Done: both binaries built and copied to C:\GD\tc\SZA\_APP." -ForegroundColor Green
Write-Host "Local build - nothing committed or pushed. Gated+committed flow: scripts/build-local.ps1" -ForegroundColor DarkGray
