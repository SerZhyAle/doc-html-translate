<#
.SYNOPSIS
  Build the browser extension: stamp a date-time version, run unit tests, ensure the
  vendored deps exist, and (optionally) produce the store zip. Local and free - nothing is
  published. The extension analogue of scripts/build.ps1 (which builds the desktop CLI).

.DESCRIPTION
  Steps:
    1. test    - node --test on the pure modules (skip with -NoTest)
    2. stamp   - node build.mjs stamp -> manifest.json + package.json version = yy.MMdd.HHmm
                 (same date-time scheme as scripts/build.ps1; build.mjs is the single source
                 of truth). The version shows in the popup, the options footer and
                 chrome://extensions, so you can tell which build you loaded.
    3. vendor  - node build.mjs vendor when extension/vendor is missing (force with -Vendor)
    4. zip     - node build.mjs zip -> extension/dist/*.zip (only with -Zip)

  For local testing you load the extension unpacked from extension/ - stamping is all you
  need, then Reload in chrome://extensions. -Zip is for the store package.

.PARAMETER Zip
  Also build the store-ready zip (extension/dist/doc-html-translate-extension.zip).

.PARAMETER Vendor
  Force re-vendoring even if extension/vendor already exists.

.PARAMETER NoTest
  Skip the node unit tests.

.EXAMPLE
  ./scripts/build-extension.ps1            # stamp + test + ensure vendored (then load unpacked)

.EXAMPLE
  ./scripts/build-extension.ps1 -Zip       # also produce the store zip
#>
param(
    [switch]$Zip,
    [switch]$Vendor,
    [switch]$NoTest
)

$ErrorActionPreference = "Stop"
$repo = Split-Path -Parent $PSScriptRoot
$ext = Join-Path $repo "extension"

function Invoke-Native {
    param([string]$What, [scriptblock]$Cmd)
    & $Cmd
    if ($LASTEXITCODE -ne 0) { throw "$What failed (exit $LASTEXITCODE)" }
}

Push-Location $ext
try {
    if (-not $NoTest) {
        Write-Host "== [1/4] test ==" -ForegroundColor Cyan
        Invoke-Native "extension tests" { node --test "test/*.test.mjs" }
    }

    Write-Host "== [2/4] stamp version ==" -ForegroundColor Cyan
    Invoke-Native "stamp" { node build.mjs stamp }

    Write-Host "== [3/4] vendor ==" -ForegroundColor Cyan
    if ($Vendor -or -not (Test-Path (Join-Path $ext "vendor/pdf.mjs"))) {
        Invoke-Native "vendor" { node build.mjs vendor }
    }
    else {
        Write-Host "vendor/ present - skipping (use -Vendor to force)" -ForegroundColor DarkGray
    }

    if ($Zip) {
        Write-Host "== [4/4] zip ==" -ForegroundColor Cyan
        Invoke-Native "zip" { node build.mjs zip }
    }
}
finally {
    Pop-Location
}

$version = (Get-Content (Join-Path $ext "manifest.json") -Raw | ConvertFrom-Json).version
Write-Host ""
Write-Host "Extension version stamped: $version" -ForegroundColor Green
if ($Zip) {
    Write-Host "Store zip: extension/dist/doc-html-translate-extension.zip" -ForegroundColor Green
}
Write-Host "Load unpacked from 'extension/' (chrome://extensions -> Reload) - version shows in the popup." -ForegroundColor DarkGray
Write-Host "Local build - nothing published. To publish, run the extension release flow." -ForegroundColor DarkGray
