<#
.SYNOPSIS
  Build a Microsoft Store (MSIX) package for doc-html-translate.

.DESCRIPTION
  Self-contained packaging pipeline (no dependency on scripts/build*.ps1, which deploy
  to an environment-specific folder). Steps:
    1. Compute version  -> stamp YY.MMDD.HHmm  +  4-part MSIX identity version YY.MMDD.HHmm.0
    2. Build CLI + GUI   (go build, amd64, icon + version embedded via goversioninfo)
    3. Generate logos    (Store/44/71/150/wide PNGs rendered from the brand color)
    4. Fill manifest     (placeholders -> staging\AppxManifest.xml)
    5. makeappx pack     -> out\<name>_<version>_x64.msix
    6. -SelfSign (opt)   sign for LOCAL testing and print the install commands

  For a STORE upload, pass the three identity values from Partner Center
  (Product > Product identity) and DO NOT use -SelfSign — upload the unsigned .msix;
  Microsoft re-signs it during certification.

.EXAMPLE
  # Local smoke test (self-signed, installable on this machine):
  .\msix\build-msix.ps1 -SelfSign

.EXAMPLE
  # Store package (unsigned, ready to upload). -Publisher / -PublisherDisplayName already
  # default to the SZA account values, so only the reserved per-product Name is required:
  .\msix\build-msix.ps1 -IdentityName "<Package/Identity/Name from Partner Center>"
#>
param(
    # Package/Identity/Name from Partner Center. Default suits self-signed local tests;
    # for a Store upload, pass the reserved per-product Name (Partner Center > Product identity).
    [string]$IdentityName = "SerZhyAle.DocHtmlTranslate",
    # Package/Identity/Publisher. Account-wide for the SZA publisher and identical across all
    # SZA products, so it is the default. For -SelfSign this MUST equal the cert subject (it does).
    [string]$Publisher = "CN=F98ACEDB-1E22-4C39-AF63-F9FCFE807DCD",
    # Package/Properties/PublisherDisplayName (shown on the Store listing). Account-wide for SZA.
    [string]$PublisherDisplayName = "SZA",
    # Override the version stamp (YY.MMDD.HHmm). Default: now.
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

# ── generate logo PNGs (brand color #1E3A8A, white DOC/HTML) ──
Add-Type -AssemblyName PresentationCore, PresentationFramework, WindowsBase
function New-LogoPng([string]$Path, [int]$W, [int]$H, [string[]]$Lines, [double]$FontFrac) {
    $bg = New-Object Windows.Media.SolidColorBrush ([Windows.Media.Color]::FromRgb(30,58,138))
    $fg = New-Object Windows.Media.SolidColorBrush ([Windows.Media.Color]::FromRgb(255,255,255))
    $tf = New-Object Windows.Media.Typeface(
        (New-Object Windows.Media.FontFamily("Segoe UI")),
        [Windows.FontStyles]::Normal, [Windows.FontWeights]::Bold, [Windows.FontStretches]::Normal)
    $size = [Math]::Max(8, [int]($H * $FontFrac))
    $visual = New-Object Windows.Media.DrawingVisual
    $ctx = $visual.RenderOpen()
    $ctx.DrawRectangle($bg, $null, (New-Object Windows.Rect(0,0,$W,$H)))
    $texts = foreach ($l in $Lines) {
        New-Object Windows.Media.FormattedText($l,
            [Globalization.CultureInfo]::InvariantCulture, [Windows.FlowDirection]::LeftToRight,
            $tf, $size, $fg, 1.0)
    }
    $gap = $size * 0.05
    $totalH = ($texts | Measure-Object Height -Sum).Sum + ($gap * ($texts.Count - 1))
    $y = ($H - $totalH) / 2
    foreach ($t in $texts) {
        $x = ($W - $t.Width) / 2
        $ctx.DrawText($t, (New-Object Windows.Point($x, $y)))
        $y += $t.Height + $gap
    }
    $ctx.Close()
    $rtb = New-Object Windows.Media.Imaging.RenderTargetBitmap($W, $H, 96, 96, [Windows.Media.PixelFormats]::Pbgra32)
    $rtb.Render($visual)
    $enc = New-Object Windows.Media.Imaging.PngBitmapEncoder
    $enc.Frames.Add([Windows.Media.Imaging.BitmapFrame]::Create($rtb))
    $fs = [IO.File]::Open($Path, [IO.FileMode]::Create, [IO.FileAccess]::Write)
    try { $enc.Save($fs) } finally { $fs.Dispose() }
}
Write-Host "Generating logos..." -ForegroundColor Cyan
$A = Join-Path $Staging "Assets"
New-LogoPng (Join-Path $A "StoreLogo.png")        50  50  @("DOC","HTML")     0.30
New-LogoPng (Join-Path $A "Square44x44Logo.png")  44  44  @("DH")             0.55
New-LogoPng (Join-Path $A "Square71x71Logo.png")  71  71  @("DOC","HTML")     0.30
New-LogoPng (Join-Path $A "Square150x150Logo.png") 150 150 @("DOC","HTML")    0.26
New-LogoPng (Join-Path $A "Wide310x150Logo.png")  310 150 @("DOC-HTML")       0.30

# ── fill manifest ────────────────────────────────────────────
(Get-Content (Join-Path $PSScriptRoot "AppxManifest.xml") -Raw) `
    -replace '\{\{IDENTITY_NAME\}\}',          $IdentityName `
    -replace '\{\{PUBLISHER\}\}',              $Publisher `
    -replace '\{\{PUBLISHER_DISPLAY_NAME\}\}', $PublisherDisplayName `
    -replace '\{\{VERSION\}\}',                $MsixVersion |
    Set-Content (Join-Path $Staging "AppxManifest.xml") -Encoding utf8

# ── pack ─────────────────────────────────────────────────────
$safeName = ($IdentityName -replace '[^A-Za-z0-9._-]', '_')
$MsixOut = Join-Path $OutDir ("{0}_{1}_x64.msix" -f $safeName, $MsixVersion)
Write-Host "Packing $MsixOut ..." -ForegroundColor Cyan
& $MakeAppx pack /d $Staging /p $MsixOut /o
if ($LASTEXITCODE -ne 0) { throw "makeappx pack failed" }
Write-Host "Package created: $MsixOut" -ForegroundColor Green

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
