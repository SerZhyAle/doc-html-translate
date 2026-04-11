param(
    [string]$Output = "build/doc-html-translate.exe"
)

$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Output) | Out-Null
New-Item -ItemType Directory -Force -Path "temp/logs" | Out-Null

# Step 1: Generate icon source to assets/
$iconSource = "assets/doc-html-translate.ico"
New-Item -ItemType Directory -Force -Path "assets" | Out-Null
./scripts/generate-icon.ps1 -Output $iconSource

# Step 2: Embed icon into exe via goversioninfo (generates resource.syso)
$cmdDir = "cmd/doc-html-translate"
Push-Location $cmdDir
try {
    & goversioninfo -64 -o resource.syso versioninfo.json
    if ($LASTEXITCODE -ne 0) { throw "goversioninfo failed" }
} finally {
    Pop-Location
}

# Step 3: Build (auto-links resource.syso → icon embedded in exe)
# GOARCH=amd64 ensures a 64-bit binary matching the -64 goversioninfo resource.
# This also avoids the ~2 GB address-space limit of 32-bit builds.
$absOutput = [System.IO.Path]::GetFullPath($Output)
$version = "$(Get-Date -Format 'yy.MMdd.HHmm')"
$ldflags = "-s -w -X main.Version=$version"

$env:GOARCH = "amd64"
$env:GOOS   = "windows"
go build -trimpath -ldflags "$ldflags" -o $absOutput ./cmd/doc-html-translate *>&1 | Tee-Object -FilePath "temp/logs/build.log"
if ($LASTEXITCODE -ne 0) {
    throw "build failed. See temp/logs/build.log"
}

# Step 4: Clean up temporary syso
Remove-Item "$cmdDir/resource.syso" -ErrorAction SilentlyContinue

Write-Host "Build completed: $Output (icon embedded)"

# Step 5: Copy exe + key file to deploy folder
$deployDir = "C:\GD\tc\SZA\_APP"
New-Item -ItemType Directory -Force -Path $deployDir | Out-Null
Copy-Item -Path $absOutput -Destination $deployDir -Force

$keyFile = "DEV/private/google_api.key"
if (Test-Path $keyFile) {
    Copy-Item -Path $keyFile -Destination $deployDir -Force
    Write-Host "Copied google_api.key to $deployDir"
} else {
    Write-Host "NOTE: DEV/private/google_api.key not found - Google Translate will be disabled in the deployed build."
}

Write-Host "Copied to $deployDir"
