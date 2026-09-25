<#
.SYNOPSIS
  Build a Microsoft Store (MSIX) package for doc-html-translate.

.DESCRIPTION
  Self-contained packaging pipeline (no dependency on scripts/build*.ps1, which deploy
  to an environment-specific folder). Steps:
    1. Compute version  -> stamp YY.MMDD.HHmm  +  4-part MSIX identity version YY.MMDD.HHmm.0
    2. Build CLI + GUI   (go build, amd64, icon + version embedded via goversioninfo)
    3. Visual assets     (go run ./tools/icongen -msix: the mark and the document-type glyph
                          in the targetsize-* / altform-* / scale-* set)
    4. Fill manifest     (placeholders -> staging\AppxManifest.xml)
    5. makepri new       -> staging\resources.pri, which resolves the qualified asset names
    6. makeappx pack     -> out\<name>_<version>_x64.msix
    7. -SelfSign (opt)   sign for LOCAL testing and print the install commands

  All three identity values default to the reserved Partner Center identity of this product,
  so a STORE upload passes none of them. It passes -Tag instead and DOES NOT use -SelfSign -
  upload the unsigned .msix; Microsoft re-signs it during certification. An unsigned build
  without -Tag is refused: the Store package must hold exactly the tagged tree.

.EXAMPLE
  # Local smoke test (self-signed, installable on this machine, stamped now):
  .\msix\build-msix.ps1 -SelfSign

.EXAMPLE
  # Store package (unsigned, ready to upload) of the checked-out release tag:
  .\msix\build-msix.ps1 -Tag v26.0612.0124
#>
param(
    # Package/Identity/Name - the frozen anchor reserved in Partner Center (Product > Product
    # identity). Never change it for a Store build; a self-signed test that must not collide with
    # an installed Store copy passes its own name (reinstall.ps1 does).
    [string]$IdentityName = "SZA.Doc-HTML-Translate",
    # Package/Identity/Publisher. Account-wide for the SZA publisher and identical across all
    # SZA products, so it is the default. For -SelfSign this MUST equal the cert subject (it does).
    [string]$Publisher = "CN=F98ACEDB-1E22-4C39-AF63-F9FCFE807DCD",
    # Package/Properties/PublisherDisplayName (shown on the Store listing). Account-wide for SZA.
    [string]$PublisherDisplayName = "SZA",
    # Release build: the app tag (vYY.MMDD.HHmm). The version comes from the tag, and the build
    # refuses a HEAD that is not the tag's commit or a working tree with changes. Required for an
    # unsigned (Store) package; exclusive with -Stamp.
    [string]$Tag,
    # Development build: override the version stamp (YY.MMDD.HHmm). Default: now. No tree check.
    [string]$Stamp,
    # Sign the package with a self-signed cert and print local-install commands.
    [switch]$SelfSign
)

$ErrorActionPreference = "Stop"
$script:RepoRoot = Split-Path -Parent $PSScriptRoot
$Staging = Join-Path $PSScriptRoot "staging"
$OutDir  = Join-Path $PSScriptRoot "out"

# ── tool discovery ───────────────────────────────────────────
function Get-SdkTool([string]$Name) {
    $hit = Get-ChildItem "C:\Program Files (x86)\Windows Kits\10\bin\*\x64\$Name" -ErrorAction SilentlyContinue |
           Sort-Object FullName | Select-Object -Last 1
    if (-not $hit) { throw "$Name not found. Install the Windows SDK: winget install Microsoft.WindowsSDK.10.0.26100" }
    return $hit.FullName
}
if (-not (Get-Command go -ErrorAction SilentlyContinue))            { throw "go not on PATH." }
if (-not (Get-Command goversioninfo -ErrorAction SilentlyContinue)) { throw "goversioninfo not on PATH. Run: go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest" }
$MakeAppx = Get-SdkTool "makeappx.exe"
$MakePri  = Get-SdkTool "makepri.exe"

# ── release tree (ticket 37) ─────────────────────────────────
# -Tag: HEAD must be the tag's commit and nothing may be uncommitted, so the working tree the build
# reads is the tag's tree. Returns the version the tag names.
function Assert-ReleaseTree([string]$ReleaseTag) {
    if ($ReleaseTag -notmatch '^v\d{2}\.\d{4}\.\d{4}$') { throw "-Tag must look like vYY.MMDD.HHmm (e.g. v26.0612.0124)" }
    Push-Location $RepoRoot
    try {
        $want = & git rev-parse --verify --quiet "refs/tags/$ReleaseTag^{commit}"
        if ($LASTEXITCODE -ne 0 -or -not $want) { throw "Tag $ReleaseTag does not exist in this clone (git fetch --tags)" }
        $head = & git rev-parse HEAD
        if ($head -ne $want) { throw "HEAD $head is not the commit of $ReleaseTag ($want); check the tag out first: git switch --detach $ReleaseTag" }
        $dirty = @(& git status --porcelain)
        if ($dirty) { throw "The working tree has changes $ReleaseTag does not hold:`n$($dirty -join "`n")" }
    } finally { Pop-Location }
    return $ReleaseTag.Substring(1)
}

if ($Tag) {
    if ($Stamp) { throw "-Tag and -Stamp are exclusive: a release build takes its version from the tag" }
    $Stamp = Assert-ReleaseTree $Tag
} elseif (-not $SelfSign) {
    throw "An unsigned package is the Store upload and must be built from its release tag: pass -Tag vYY.MMDD.HHmm (or -SelfSign for a local test)"
}

# ── version ──────────────────────────────────────────────────
if ($Stamp) {
    if ($Stamp -notmatch '^\d{2}\.\d{4}\.\d{4}$') { throw "-Stamp must look like YY.MMDD.HHmm (e.g. 26.0612.0124)" }
    $p = $Stamp.Split('.')
    $vMajor = [int]$p[0]; $mmdd = $p[1]; $vMinor = [int]$mmdd.Substring(0,2); $vPatch = [int]$mmdd.Substring(2,2); $vBuild = [int]$p[2]
} else {
    $now = Get-Date
    $vMajor = [int]$now.ToString('yy'); $vMinor = [int]$now.ToString('MM'); $vPatch = [int]$now.ToString('dd'); $vBuild = [int]$now.ToString('HHmm')
    $Stamp  = "{0}.{1:D2}{2:D2}.{3:D4}" -f $vMajor, $vMinor, $vPatch, $vBuild
}
# MSIX identity needs a 4-part version with revision 0 and each part <= 65535.
# YY . MMDD . HHmm . 0   (monotonic over time, unique per minute, all parts in range).
$MsixVersion = "{0}.{1}.{2}.0" -f $vMajor, ([int]($Stamp.Split('.')[1])), $vBuild
Write-Host "Version: stamp=$Stamp  msix=$MsixVersion" -ForegroundColor Cyan

# ── clean staging ────────────────────────────────────────────
Remove-Item $Staging -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $Staging, (Join-Path $Staging "Assets"), $OutDir | Out-Null

# ── build one exe with embedded icon + version resource ──────
function New-VersionResourceFile([string]$TemplatePath, [string]$OutputPath, [string]$VersionString) {
    $json = Get-Content -LiteralPath $TemplatePath -Raw | ConvertFrom-Json
    foreach ($fi in @($json.FixedFileInfo.FileVersion, $json.FixedFileInfo.ProductVersion)) {
        $fi.Major = $vMajor; $fi.Minor = $vMinor; $fi.Patch = $vPatch; $fi.Build = $vBuild
    }
    $json.StringFileInfo.FileVersion = $VersionString
    $json.StringFileInfo.ProductVersion = $VersionString
    $json | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $OutputPath -Encoding utf8
}

function Build-Exe([string]$CmdDir, [string]$OutExe, [string]$ExtraLdflags) {
    $abs   = Join-Path $Staging $OutExe
    $vinfo = Join-Path (Join-Path $RepoRoot $CmdDir) "versioninfo.generated.json"
    New-VersionResourceFile (Join-Path (Join-Path $RepoRoot $CmdDir) "versioninfo.json") $vinfo $Stamp
    Push-Location (Join-Path $RepoRoot $CmdDir)
    try {
        & goversioninfo -64 -o resource.syso versioninfo.generated.json
        if ($LASTEXITCODE -ne 0) { throw "goversioninfo failed for $CmdDir" }
        $env:GOARCH = "amd64"; $env:GOOS = "windows"
        # CWD is the command's package dir, so build "." — this also links resource.syso.
        go build -trimpath -ldflags "-s -w -X main.Version=$Stamp $ExtraLdflags" -o $abs .
        if ($LASTEXITCODE -ne 0) { throw "go build failed for $CmdDir" }
    } finally {
        Remove-Item "resource.syso" -ErrorAction SilentlyContinue
        Remove-Item $vinfo -ErrorAction SilentlyContinue
        Pop-Location
    }
    Write-Host "  built $OutExe" -ForegroundColor DarkGray
}

# go build paths are relative to repo root; CmdDir is e.g. cmd/doc-html-ui
Push-Location $RepoRoot
try {
    Write-Host "Building binaries..." -ForegroundColor Cyan
    Build-Exe "cmd/doc-html-translate" "doc-html-translate.exe" ""
    Build-Exe "cmd/doc-html-ui"        "doc-html-ui.exe"        "-H windowsgui"
} finally { Pop-Location }

# third-party material inside the binaries (the Material Icons glyphs of the reader chrome and GUI)
Copy-Item (Join-Path $RepoRoot "THIRD-PARTY-NOTICES.txt") $Staging -Force

# ── generate the visual assets (ICON-RENDER rule 9, ticket 32) ──
# internal/iconart draws the product mark and the document-type glyph into the MRT-qualified
# set: Square44x44Logo in targetsize-16/24/32/48/256 with its altform-unplated and
# altform-lightunplated forms, the scale-100/200 tiles and StoreLogo, and DocumentType for the
# file type association. Windows picks among them only through resources.pri (below).
Write-Host "Generating visual assets..." -ForegroundColor Cyan
$A = Join-Path $Staging "Assets"
Push-Location $RepoRoot
try {
    # Build-Exe left GOOS/GOARCH set for the package; the generator runs on this machine.
    Remove-Item Env:GOARCH, Env:GOOS -ErrorAction SilentlyContinue
    go run ./tools/icongen -msix $A | Out-Null
    if ($LASTEXITCODE -ne 0) { throw "icongen failed" }
} finally { Pop-Location }

# ── fill manifest ────────────────────────────────────────────
(Get-Content (Join-Path $PSScriptRoot "AppxManifest.xml") -Raw) `
    -replace '\{\{IDENTITY_NAME\}\}',          $IdentityName `
    -replace '\{\{PUBLISHER\}\}',              $Publisher `
    -replace '\{\{PUBLISHER_DISPLAY_NAME\}\}', $PublisherDisplayName `
    -replace '\{\{VERSION\}\}',                $MsixVersion |
    Set-Content (Join-Path $Staging "AppxManifest.xml") -Encoding utf8

# ── resources.pri ────────────────────────────────────────────
# The index Windows resolves the qualified asset names through (scale-*, targetsize-*,
# altform-*). Without it the taskbar and Start fall back to the plated tile scaled down.
# The config lives outside the staging folder so it is not packed.
Write-Host "Indexing resources (makepri)..." -ForegroundColor Cyan
$PriConfig = Join-Path $OutDir "priconfig.xml"
& $MakePri createconfig /cf $PriConfig /dq en-US /pv 10.0.0 /o | Out-Null
if ($LASTEXITCODE -ne 0) { throw "makepri createconfig failed" }
# The default config splits scale and language candidates into resources.<qualifier>.pri files
# meant for resource packs of a bundle; this is a single package, so everything stays in one.
[xml]$pri = Get-Content -LiteralPath $PriConfig -Raw
$pack = $pri.SelectSingleNode("/resources/packaging")
if ($pack) { [void]$pri.resources.RemoveChild($pack) }
$pri.Save($PriConfig)
& $MakePri new /pr $Staging /cf $PriConfig /mn (Join-Path $Staging "AppxManifest.xml") /of (Join-Path $Staging "resources.pri") /o | Out-Null
if ($LASTEXITCODE -ne 0) { throw "makepri new failed" }

# ── pack ─────────────────────────────────────────────────────
$safeName = ($IdentityName -replace '[^A-Za-z0-9._-]', '_')
$MsixOut = Join-Path $OutDir ("{0}_{1}_x64.msix" -f $safeName, $MsixVersion)
Write-Host "Packing $MsixOut ..." -ForegroundColor Cyan
& $MakeAppx pack /d $Staging /p $MsixOut /o
if ($LASTEXITCODE -ne 0) { throw "makeappx pack failed" }
Write-Host "Package created: $MsixOut" -ForegroundColor Green

# A release build whose tree moved while it ran did not build the tag.
if ($Tag) {
    $after = @(& git -C $RepoRoot status --porcelain)
    if ($after) { throw "The build left the tree dirty, so $MsixOut is not $Tag's tree:`n$($after -join "`n")" }
}

# ── optional self-sign for local testing ────────────────────
if ($SelfSign) {
    $SignTool = Get-SdkTool "signtool.exe"
    Write-Host "Self-signing (subject must equal Publisher='$Publisher')..." -ForegroundColor Cyan
    # Reuse a prior test cert with the same subject so repeated builds don't pile up certs.
    $cert = Get-ChildItem "Cert:\CurrentUser\My" |
        Where-Object { $_.Subject -eq $Publisher -and $_.FriendlyName -eq "doc-html-translate MSIX test" } |
        Sort-Object NotAfter | Select-Object -Last 1
    if (-not $cert) {
        $cert = New-SelfSignedCertificate -Type Custom -Subject $Publisher `
            -KeyUsage DigitalSignature -FriendlyName "doc-html-translate MSIX test" `
            -CertStoreLocation "Cert:\CurrentUser\My" `
            -TextExtension @("2.5.29.37={text}1.3.6.1.5.5.7.3.3", "2.5.29.19={text}")
    }
    & $SignTool sign /fd SHA256 /sha1 $cert.Thumbprint $MsixOut
    if ($LASTEXITCODE -ne 0) { throw "signtool failed" }
    $cerPath = Join-Path $OutDir "doc-html-translate-test.cer"
    Export-Certificate -Cert $cert -FilePath $cerPath | Out-Null
    Write-Host ""
    Write-Host "Signed. To install on THIS machine:" -ForegroundColor Green
    Write-Host "  1) Trust the test cert (run once, as Administrator):" -ForegroundColor Yellow
    Write-Host "     Import-Certificate -FilePath '$cerPath' -CertStoreLocation Cert:\LocalMachine\Root"
    Write-Host "  2) Install the package:" -ForegroundColor Yellow
    Write-Host "     Add-AppxPackage '$MsixOut'"
    Write-Host "  3) Launch it from the Start menu (Add-AppxPackage installs but does not launch)." -ForegroundColor Yellow
} else {
    Write-Host ""
    Write-Host "Unsigned package ready to UPLOAD to Partner Center (Microsoft re-signs at certification)." -ForegroundColor Green
    Write-Host "For a local smoke test instead, re-run with -SelfSign." -ForegroundColor DarkGray
}
