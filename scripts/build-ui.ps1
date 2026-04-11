param(
    [string]$Output = "build/doc-html-ui.exe"
)

$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Output) | Out-Null
New-Item -ItemType Directory -Force -Path "temp/logs" | Out-Null

# ── Step 1: Icon ─────────────────────────────────────────
$iconSource = "assets/doc-html-translate.ico"
New-Item -ItemType Directory -Force -Path "assets" | Out-Null
if (-not (Test-Path $iconSource)) {
    ./scripts/generate-icon.ps1 -Output $iconSource
}

# ── Step 2: goversioninfo ────────────────────────────────
$cmdDir = "cmd/doc-html-ui"
Push-Location $cmdDir
try {
    & goversioninfo -o resource.syso versioninfo.json
    if ($LASTEXITCODE -ne 0) { throw "goversioninfo failed" }
} finally {
    Pop-Location
}

# ── Step 3: Build (pure Go, no CGO needed; -H windowsgui hides console) ──
$absOutput = [System.IO.Path]::GetFullPath($Output)
$version   = "1.0.0-$(Get-Date -Format 'yyyyMMdd')"
$ldflags   = "-s -w -H windowsgui -X main.Version=$version"

go build -trimpath -ldflags "$ldflags" -o $absOutput ./cmd/doc-html-ui *>&1 | Tee-Object -FilePath "temp/logs/build-ui.log"
if ($LASTEXITCODE -ne 0) {
    throw "build failed. See temp/logs/build-ui.log"
}

# ── Step 4: Cleanup ──────────────────────────────────────
Remove-Item "$cmdDir/resource.syso" -ErrorAction SilentlyContinue

Write-Host "UI build completed: $Output (embedded web UI, icon embedded)"

# ── Step 5: Copy exe to deploy folder ────────────────────
$deployDir = "C:\GD\tc\SZA\_APP"
New-Item -ItemType Directory -Force -Path $deployDir | Out-Null
Copy-Item -Path $absOutput -Destination $deployDir -Force
Write-Host "Copied to $deployDir"
