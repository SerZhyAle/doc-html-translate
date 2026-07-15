<#
.SYNOPSIS
  Quiet local reinstall loop: rebuild -> kill -> uninstall -> install -> launch.
  Local and free: no gate, no commit, no push, no Store upload.

.DESCRIPTION
  The fast "test the real MSIX package" loop. From the repo root it:
    1. Kills any running app processes (doc-html-ui / doc-html-translate).
    2. Rebuilds the self-signed MSIX via msix/build-msix.ps1 -SelfSign
       (rebuilds both binaries, packs, signs, exports the test cert).
    3. Trusts the test cert in LocalMachine\Root if not already trusted
       (self-elevates just for that one step; needed only the first time -
       the cert thumbprint is stable across builds).
    4. Uninstalls the currently installed package (same Identity Name).
    5. Installs the freshly built .msix.
    6. Launches the GUI app.

  This is NOT a release. It only touches THIS machine. For the published
  path use scripts/release.ps1; for the plain compile+deploy use build.ps1.

.PARAMETER SkipBuild
  Skip the rebuild and reuse the newest .msix already in msix\out.

.PARAMETER NoLaunch
  Reinstall but do not launch the app afterwards.

.PARAMETER IdentityName
  Package/Identity/Name. Must match the -SelfSign default in build-msix.ps1.

.EXAMPLE
  ./reinstall.ps1

.EXAMPLE
  ./reinstall.ps1 -SkipBuild        # reinstall the last package without rebuilding
#>
param(
    [switch]$SkipBuild,
    [switch]$NoLaunch,
    [string]$IdentityName = "SerZhyAle.DocHtmlTranslate"
)

$ErrorActionPreference = "Stop"
Set-Location $PSScriptRoot

$AppId  = "DocHtmlUi"                       # <Application Id> in msix/AppxManifest.xml
$OutDir = Join-Path $PSScriptRoot "msix\out"
$CerPath = Join-Path $OutDir "doc-html-translate-test.cer"

# ── Step 1: kill running instances (quiet) ───────────────────
Write-Host "== [1/6] stop running app ==" -ForegroundColor Cyan
foreach ($p in "doc-html-ui", "doc-html-translate") {
    Get-Process -Name $p -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue
}

# ── Step 2: rebuild the signed MSIX ──────────────────────────
if ($SkipBuild) {
    Write-Host "== [2/6] build (skipped) ==" -ForegroundColor DarkGray
} else {
    Write-Host "== [2/6] rebuild signed MSIX (build-msix.ps1 -SelfSign) ==" -ForegroundColor Cyan
    ./msix/build-msix.ps1 -SelfSign -IdentityName $IdentityName
}

$msix = Get-ChildItem $OutDir -Filter "*.msix" -ErrorAction SilentlyContinue |
        Sort-Object LastWriteTime | Select-Object -Last 1
if (-not $msix) { throw "No .msix found in $OutDir. Run without -SkipBuild first." }
Write-Host "Package: $($msix.Name)" -ForegroundColor DarkGray

# ── Step 3: trust the test cert (self-elevate only if needed) ─
Write-Host "== [3/6] trust test cert ==" -ForegroundColor Cyan
if (-not (Test-Path $CerPath)) { throw "Test cert not found: $CerPath (run without -SkipBuild to regenerate)." }
$cert  = New-Object System.Security.Cryptography.X509Certificates.X509Certificate2 $CerPath
$thumb = $cert.Thumbprint
if (Test-Path "Cert:\LocalMachine\Root\$thumb") {
    Write-Host "Already trusted." -ForegroundColor DarkGray
} else {
    $isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole(
        [Security.Principal.WindowsBuiltInRole]::Administrator)
    if ($isAdmin) {
        Import-Certificate -FilePath $CerPath -CertStoreLocation Cert:\LocalMachine\Root | Out-Null
    } else {
        Write-Host ""
        Write-Host "  Admin rights are needed for ONE step: trusting the local test certificate." -ForegroundColor Yellow
        Write-Host "  Why:  Windows only installs a signed MSIX whose signer it trusts. This package" -ForegroundColor Yellow
        Write-Host "        is self-signed for local testing, so its test cert must be added to the" -ForegroundColor Yellow
        Write-Host "        machine's Trusted Root store - a machine-wide change that requires admin." -ForegroundColor Yellow
        Write-Host "  What you get:  Add-AppxPackage will accept this and future rebuilds without prompting" -ForegroundColor Yellow
        Write-Host "        again (the cert thumbprint is stable). This is a local dev cert only - it does" -ForegroundColor Yellow
        Write-Host "        NOT affect the Store build, which Microsoft re-signs at certification." -ForegroundColor Yellow
        Write-Host "  A UAC prompt will appear now (first time only). Decline it to abort the install." -ForegroundColor Yellow
        Write-Host ""
        Start-Process -Verb RunAs -Wait -FilePath "powershell" -ArgumentList @(
            "-NoProfile", "-Command",
            "Import-Certificate -FilePath '$CerPath' -CertStoreLocation Cert:\LocalMachine\Root")
        if (-not (Test-Path "Cert:\LocalMachine\Root\$thumb")) {
            throw "Cert was not trusted (UAC declined?). Aborting - Add-AppxPackage would fail without it."
        }
    }
    Write-Host "Trusted." -ForegroundColor DarkGray
}

# ── Step 4: uninstall existing package (quiet) ───────────────
Write-Host "== [4/6] uninstall existing ==" -ForegroundColor Cyan
Get-AppxPackage -Name $IdentityName -ErrorAction SilentlyContinue | ForEach-Object {
    Remove-AppxPackage -Package $_.PackageFullName -ErrorAction SilentlyContinue
}

# ── Step 5: install the fresh package ────────────────────────
Write-Host "== [5/6] install ==" -ForegroundColor Cyan
Add-AppxPackage -Path $msix.FullName -ForceUpdateFromAnyVersion -ForceApplicationShutdown

$pkg = Get-AppxPackage -Name $IdentityName
if (-not $pkg) { throw "Install reported success but the package is not registered." }
Write-Host "Installed $($pkg.PackageFullName)" -ForegroundColor Green

# ── Step 6: launch ───────────────────────────────────────────
if ($NoLaunch) {
    Write-Host "== [6/6] launch (skipped) ==" -ForegroundColor DarkGray
} else {
    Write-Host "== [6/6] launch ==" -ForegroundColor Cyan
    $aumid = "$($pkg.PackageFamilyName)!$AppId"
    Start-Process "shell:AppsFolder\$aumid"
    Write-Host "Launched $aumid" -ForegroundColor Green
}

Write-Host ""
Write-Host "Reinstalled locally from the self-signed MSIX - nothing was pushed or published." -ForegroundColor DarkGray
