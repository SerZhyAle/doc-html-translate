<#
.SYNOPSIS
  Capture the doc-html-ui window, one PNG per interface language, for the store listing.

.DESCRIPTION
  doc-html-ui is a local HTTP app: it listens on an ephemeral 127.0.0.1 port and opens a browser
  window against it. That is what makes a deterministic capture possible - the script starts the
  app, finds the port it is listening on, and screenshots `http://127.0.0.1:<port>/?lang=<code>`
  in headless Edge at 1366x768. The `?lang=` override is read-only: it never touches the saved
  language, so a 13-language run leaves the developer's own GUI exactly as it was.

  Output: tools/store/gui-<store-locale>.png (Partner Center locale codes).

.EXAMPLE
  .\tools\store\make-gui-screenshot.ps1 -Language de
#>
param(
    # Path to doc-html-ui.exe. Default: msix/staging then build/.
    [string]$Ui,
    # Interface languages to capture. Default: all 13 the app ships.
    [string[]]$Language = @("en", "ru", "uk", "de", "it", "es", "fr", "pt", "ar", "hi", "bn", "ur", "zh")
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$OutDir   = $PSScriptRoot
$WorkDir  = Join-Path $RepoRoot "temp\gui-screenshot"

$StoreLocale = @{
    en = "en-us"; ru = "ru"; uk = "uk"; de = "de"; it = "it"; es = "es"; fr = "fr"
    pt = "pt-br"; ar = "ar"; hi = "hi"; bn = "bn"; ur = "ur"; zh = "zh-hans"
}
$RtlLanguages = @("ar", "ur")

if (-not $Ui) {
    foreach ($c in @(
        (Join-Path $RepoRoot "msix\staging\doc-html-ui.exe"),
        (Join-Path $RepoRoot "build\doc-html-ui.exe"))) {
        if (Test-Path $c) { $Ui = $c; break }
    }
}
if (-not $Ui -or -not (Test-Path $Ui)) {
    throw "doc-html-ui.exe not found. Build it first (.\scripts\build-ui.ps1) or pass -Ui."
}

$Edge = @(
    "$env:ProgramFiles\Microsoft\Edge\Application\msedge.exe",
    "${env:ProgramFiles(x86)}\Microsoft\Edge\Application\msedge.exe"
) | Where-Object { Test-Path $_ } | Select-Object -First 1
if (-not $Edge) { throw "msedge.exe not found (needed for headless capture)." }

Remove-Item $WorkDir -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $WorkDir | Out-Null
Add-Type -AssemblyName System.Drawing

# The app persists its settings per user; back them up so a crash mid-run cannot leave the
# developer's GUI in a language they did not choose.
$SettingsPath = Join-Path $env:LOCALAPPDATA "doc-html-translate\ui-settings.json"
$SettingsBackup = $null
if (Test-Path $SettingsPath) {
    $SettingsBackup = Join-Path $WorkDir "ui-settings.backup.json"
    Copy-Item $SettingsPath $SettingsBackup -Force
}

$app = $null
try {
    $app = Start-Process -FilePath $Ui -PassThru
    # Discover the ephemeral port the app is listening on. It never prints it - the window is
    # opened from inside the process - so the OS connection table is the only honest source.
    $port = $null
    for ($i = 0; $i -lt 100 -and -not $port; $i++) {
        Start-Sleep -Milliseconds 200
        $conn = Get-NetTCPConnection -State Listen -OwningProcess $app.Id -ErrorAction SilentlyContinue |
            Where-Object { $_.LocalAddress -eq "127.0.0.1" } | Select-Object -First 1
        if ($conn) { $port = $conn.LocalPort }
    }
    if (-not $port) { throw "doc-html-ui did not start listening on 127.0.0.1" }
    Write-Host "doc-html-ui listening on 127.0.0.1:$port" -ForegroundColor Cyan

    $EdgeProfile = Join-Path $WorkDir "edge-profile"
    foreach ($lang in $Language) {
        $locale = $StoreLocale[$lang]
        if (-not $locale) { throw "unknown language '$lang' - expected one of $($StoreLocale.Keys -join ' ')" }
        $outPath = Join-Path $OutDir "gui-$locale.png"
        $dumpPath = Join-Path $WorkDir "gui-$lang.html"
        $url = "http://127.0.0.1:$port/?lang=$lang"

        # Same rule as the reading view: the app chrome mirrors for RTL, and nothing else does.
        # Dump the rendered DOM first so a wrong direction fails the run instead of shipping.
        & $Edge --headless=new --disable-gpu --no-first-run --user-data-dir="$EdgeProfile" `
            --virtual-time-budget=4000 --dump-dom $url 2>$null | Set-Content -LiteralPath $dumpPath -Encoding utf8
        $dom = Get-Content -LiteralPath $dumpPath -Raw
        $expectRtl = $RtlLanguages -contains $lang
        $root = [regex]::Match($dom, '<html[^>]*>')
        if (-not $root.Success) { throw "SCREENSHOT-RTL: no <html> in the dumped DOM for $lang" }
        if ($root.Value -notmatch ('lang="' + [regex]::Escape($lang) + '"')) {
            throw "SCREENSHOT-RTL: the GUI did not switch to $lang ($($root.Value))"
        }
        $isRtl = $root.Value -match 'dir="rtl"'
        if ($isRtl -ne $expectRtl) {
            throw "SCREENSHOT-RTL: GUI dir=rtl is $isRtl but $lang expects $expectRtl"
        }

        Remove-Item $outPath -ErrorAction SilentlyContinue
        & $Edge --headless=new --disable-gpu --no-first-run --hide-scrollbars `
            --user-data-dir="$EdgeProfile" --force-device-scale-factor=1 `
            --virtual-time-budget=4000 --window-size=1366,768 "--screenshot=$outPath" $url 2>$null | Out-Null
        for ($i = 0; $i -lt 50 -and -not (Test-Path $outPath); $i++) { Start-Sleep -Milliseconds 100 }
        if (-not (Test-Path $outPath)) { throw "Edge did not produce gui-$locale.png" }
        $img = [System.Drawing.Image]::FromFile($outPath)
        Write-Host ("  {0,-22} {1}x{2}" -f "gui-$locale.png", $img.Width, $img.Height) -ForegroundColor Green
        $img.Dispose()
    }
}
finally {
    if ($app -and -not $app.HasExited) { Stop-Process -Id $app.Id -Force -ErrorAction SilentlyContinue }
    if ($SettingsBackup -and (Test-Path $SettingsBackup)) {
        Copy-Item $SettingsBackup $SettingsPath -Force -ErrorAction SilentlyContinue
    }
}
Write-Host "GUI screenshots written to $OutDir" -ForegroundColor Cyan
