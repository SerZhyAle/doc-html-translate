<#
.SYNOPSIS
  Build a single universal Windows setup.exe (Inno Setup) that installs on both x86 and
  x64 Windows. Local and free: no gate, no commit, no push, no Store upload.

.DESCRIPTION
  The "copy setup.exe to any Windows PC" artifact. From the repo root it:
    1. Computes the version stamp (YY.MMDD.HHmm), same scheme as the other build scripts.
    2. Builds CLI + GUI for amd64 AND 386 (pure Go, CGO off, icon + version embedded).
    3. Stages the payload under temp\installer-staging (x64\, x86\, tessdata\, icon, docs).
    4. Compiles installer\doc-html-translate.iss with ISCC -> dist\doc-html-translate-setup-<ver>.exe.

  The setup is per-user (installs to %LocalAppData%\Programs, no admin) and, on 64-bit
  Windows, deploys the x64 binaries; on 32-bit Windows, the x86 binaries. There is no
  trouble with x86 - the app is pure Go with CGO_ENABLED=0 and calls tesseract.exe as an
  external process, so GOARCH=386 cross-compiles with no changes.

  For the plain compile+deploy path use build.ps1; for the Store package use
  msix\build-msix.ps1; to publish use scripts\release.ps1.

.PARAMETER Stamp
  Override the version stamp (YY.MMDD.HHmm). Default: now.

.PARAMETER KeepStaging
  Leave temp\installer-staging in place after the build (for inspection).

.EXAMPLE
  ./scripts/build-installer.ps1
#>
param(
    [string]$Stamp,
    [switch]$KeepStaging
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $RepoRoot

# ── ISCC (Inno Setup compiler) discovery ─────────────────────
function Get-Iscc {
    if ($env:DOCHT_ISCC -and (Test-Path $env:DOCHT_ISCC)) { return $env:DOCHT_ISCC }
    $onPath = Get-Command ISCC.exe -ErrorAction SilentlyContinue
    if ($onPath) { return $onPath.Source }
    $candidates = @(
        "$env:LocalAppData\Programs\Inno Setup 6\ISCC.exe",
        "C:\Program Files (x86)\Inno Setup 6\ISCC.exe",
        "C:\Program Files\Inno Setup 6\ISCC.exe"
    )
    foreach ($c in $candidates) { if (Test-Path $c) { return $c } }
    throw "ISCC.exe (Inno Setup) not found. Install it: winget install JRSoftware.InnoSetup  (or set DOCHT_ISCC)."
}
$Iscc = Get-Iscc

if (-not (Get-Command go -ErrorAction SilentlyContinue))            { throw "go not on PATH." }
if (-not (Get-Command goversioninfo -ErrorAction SilentlyContinue)) { throw "goversioninfo not on PATH. Run: go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest" }

# ── version stamp ────────────────────────────────────────────
if ($Stamp) {
    if ($Stamp -notmatch '^\d{2}\.\d{4}\.\d{4}$') { throw "-Stamp must look like YY.MMDD.HHmm (e.g. 26.0715.2003)" }
    $p = $Stamp.Split('.')
    $vMajor = [int]$p[0]; $vMinor = [int]$p[1].Substring(0,2); $vPatch = [int]$p[1].Substring(2,2); $vBuild = [int]$p[2]
} else {
    $now = Get-Date
    $vMajor = [int]$now.ToString('yy'); $vMinor = [int]$now.ToString('MM'); $vPatch = [int]$now.ToString('dd'); $vBuild = [int]$now.ToString('HHmm')
    $Stamp  = "{0}.{1:D2}{2:D2}.{3:D4}" -f $vMajor, $vMinor, $vPatch, $vBuild
}
Write-Host "Version: $Stamp" -ForegroundColor Cyan

# ── staging layout ───────────────────────────────────────────
$Staging = Join-Path $RepoRoot "temp\installer-staging"
$OutDir  = Join-Path $RepoRoot "dist"
Remove-Item $Staging -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $Staging, (Join-Path $Staging "x64"), (Join-Path $Staging "x86"), $OutDir | Out-Null

# ── icon (shared) ────────────────────────────────────────────
$icon = Join-Path $Staging "doc-html-translate.ico"
./scripts/generate-icon.ps1 -Output $icon
Copy-Item $icon "cmd/doc-html-ui/favicon.ico" -Force  # keep GUI window icon in sync (see build-ui.ps1)

# ── build one exe (icon + version embedded) for a given arch ─
function New-VersionResourceFile([string]$TemplatePath, [string]$OutputPath, [string]$VersionString) {
    $json = Get-Content -LiteralPath $TemplatePath -Raw | ConvertFrom-Json
    foreach ($fi in @($json.FixedFileInfo.FileVersion, $json.FixedFileInfo.ProductVersion)) {
        $fi.Major = $vMajor; $fi.Minor = $vMinor; $fi.Patch = $vPatch; $fi.Build = $vBuild
    }
    $json.StringFileInfo.FileVersion = $VersionString
    $json.StringFileInfo.ProductVersion = $VersionString
    $json | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $OutputPath -Encoding utf8
}

function Build-Exe([string]$CmdDir, [string]$OutExe, [string]$Arch, [string]$ExtraLdflags) {
    $vinfo = Join-Path (Join-Path $RepoRoot $CmdDir) "versioninfo.generated.json"
    New-VersionResourceFile (Join-Path (Join-Path $RepoRoot $CmdDir) "versioninfo.json") $vinfo $Stamp
    Push-Location (Join-Path $RepoRoot $CmdDir)
    try {
        # -64 emits an amd64 COFF resource; omit it for the 386 (32-bit) resource so the
        # linker does not choke on a mismatched relocation.
        $goverArgs = @()
        if ($Arch -eq "amd64") { $goverArgs += "-64" }
        & goversioninfo @goverArgs -o resource.syso versioninfo.generated.json
        if ($LASTEXITCODE -ne 0) { throw "goversioninfo failed for $CmdDir ($Arch)" }
        $env:GOOS = "windows"; $env:GOARCH = $Arch; $env:CGO_ENABLED = "0"
        go build -trimpath -ldflags "-s -w -X main.Version=$Stamp $ExtraLdflags" -o $OutExe .
        if ($LASTEXITCODE -ne 0) { throw "go build failed for $CmdDir ($Arch)" }
    } finally {
        Remove-Item "resource.syso" -ErrorAction SilentlyContinue
        Remove-Item $vinfo -ErrorAction SilentlyContinue
        Pop-Location
    }
    Write-Host "  built $Arch/$(Split-Path $OutExe -Leaf)" -ForegroundColor DarkGray
}

foreach ($arch in "amd64", "386") {
    $dest = if ($arch -eq "amd64") { Join-Path $Staging "x64" } else { Join-Path $Staging "x86" }
    Write-Host "== build $arch ==" -ForegroundColor Cyan
    Build-Exe "cmd/doc-html-translate" (Join-Path $dest "doc-html-translate.exe") $arch ""
    Build-Exe "cmd/doc-html-ui"        (Join-Path $dest "doc-html-ui.exe")        $arch "-H windowsgui"
}

# ── shared payload: tessdata (English), LICENSE, README ──────
$tessDest = Join-Path $Staging "tessdata"
New-Item -ItemType Directory -Force -Path $tessDest | Out-Null
$engDest = Join-Path $tessDest "eng.traineddata"
$vendored = "extension/vendor/tesseract/lang/eng.traineddata"
if (Test-Path $vendored) {
    Copy-Item $vendored $engDest -Force
    Write-Host "Bundled eng.traineddata (from vendored copy)" -ForegroundColor DarkGray
} else {
    try {
        Invoke-WebRequest -Uri "https://github.com/tesseract-ocr/tessdata_fast/raw/main/eng.traineddata" -OutFile $engDest -UseBasicParsing
        Write-Host "Downloaded eng.traineddata" -ForegroundColor DarkGray
    } catch {
        Write-Host "NOTE: could not provision eng.traineddata ($_); -ocr will need a system tesseract or a manual download." -ForegroundColor Yellow
    }
}
Copy-Item (Join-Path $RepoRoot "LICENSE")   $Staging -Force
Copy-Item (Join-Path $RepoRoot "README.md") $Staging -Force

# ── compile the installer ────────────────────────────────────
Write-Host "== compile installer (ISCC) ==" -ForegroundColor Cyan
& $Iscc `
    "/DAppVersion=$Stamp" `
    "/DStaging=$Staging" `
    "/DOutput=$OutDir" `
    (Join-Path $RepoRoot "installer\doc-html-translate.iss")
if ($LASTEXITCODE -ne 0) { throw "ISCC failed" }

if (-not $KeepStaging) { Remove-Item $Staging -Recurse -Force -ErrorAction SilentlyContinue }

$setup = Join-Path $OutDir "doc-html-translate-setup-$Stamp.exe"
Write-Host ""
Write-Host "Installer built: $setup" -ForegroundColor Green
Write-Host "Universal x86/x64, per-user (no admin). Copy it to any Windows PC and run." -ForegroundColor DarkGray
Write-Host "Local build - nothing committed, pushed, or published." -ForegroundColor DarkGray
