param(
    [string]$Output = "build/doc-html-translate.exe"
)

$ErrorActionPreference = "Stop"

function New-VersionResourceFile {
    param(
        [string]$TemplatePath,
        [string]$OutputPath,
        [string]$VersionString,
        [int]$Major,
        [int]$Minor,
        [int]$Patch,
        [int]$Build
    )

    $json = Get-Content -LiteralPath $TemplatePath -Raw | ConvertFrom-Json
    $json.FixedFileInfo.FileVersion.Major = $Major
    $json.FixedFileInfo.FileVersion.Minor = $Minor
    $json.FixedFileInfo.FileVersion.Patch = $Patch
    $json.FixedFileInfo.FileVersion.Build = $Build
    $json.FixedFileInfo.ProductVersion.Major = $Major
    $json.FixedFileInfo.ProductVersion.Minor = $Minor
    $json.FixedFileInfo.ProductVersion.Patch = $Patch
    $json.FixedFileInfo.ProductVersion.Build = $Build
    $json.StringFileInfo.FileVersion = $VersionString
    $json.StringFileInfo.ProductVersion = $VersionString
    $json | ConvertTo-Json -Depth 10 | Set-Content -LiteralPath $OutputPath -Encoding utf8
}

New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Output) | Out-Null
New-Item -ItemType Directory -Force -Path "temp/logs" | Out-Null

# Step 1: Generate icon source to assets/
$iconSource = "assets/doc-html-translate.ico"
New-Item -ItemType Directory -Force -Path "assets" | Out-Null
./scripts/generate-icon.ps1 -Output $iconSource

# Keep the favicon embedded in the converter (written next to every generated index.html,
# so a converted book carries our icon in the browser tab) in sync with the app icon -
# same arrangement as cmd/doc-html-ui/favicon.ico in build-ui.ps1.
Copy-Item $iconSource "internal/htmlgen/favicon.ico" -Force

# Step 2: Embed icon into exe via goversioninfo (generates resource.syso)
$cmdDir = "cmd/doc-html-translate"
$versionDate = Get-Date
$versionMajor = [int]$versionDate.ToString('yy')
$versionMinor = [int]$versionDate.ToString('MM')
$versionPatch = [int]$versionDate.ToString('dd')
$versionBuild = [int]$versionDate.ToString('HHmm')
$version = "{0}.{1:D2}{2:D2}.{3:D4}" -f $versionMajor, $versionMinor, $versionPatch, $versionBuild
$versionInfoPath = Join-Path $cmdDir "versioninfo.generated.json"
New-VersionResourceFile -TemplatePath (Join-Path $cmdDir "versioninfo.json") -OutputPath $versionInfoPath -VersionString $version -Major $versionMajor -Minor $versionMinor -Patch $versionPatch -Build $versionBuild

Push-Location $cmdDir
try {
    & goversioninfo -64 -o resource.syso versioninfo.generated.json
    if ($LASTEXITCODE -ne 0) { throw "goversioninfo failed" }
} finally {
    Pop-Location
}

# Step 3: Build (auto-links resource.syso → icon embedded in exe)
# GOARCH=amd64 ensures a 64-bit binary matching the -64 goversioninfo resource.
# This also avoids the ~2 GB address-space limit of 32-bit builds.
$absOutput = [System.IO.Path]::GetFullPath($Output)
$ldflags = "-s -w -X main.Version=$version"

$env:GOARCH = "amd64"
$env:GOOS   = "windows"
go build -trimpath -ldflags "$ldflags" -o $absOutput ./cmd/doc-html-translate *>&1 | Tee-Object -FilePath "temp/logs/build.log"
if ($LASTEXITCODE -ne 0) {
    throw "build failed. See temp/logs/build.log"
}

# Step 4: Clean up temporary syso
Remove-Item "$cmdDir/resource.syso" -ErrorAction SilentlyContinue
Remove-Item $versionInfoPath -ErrorAction SilentlyContinue

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

# Step 6: Ship the bundled OCR language data (English) into <dir>/tessdata next to the
# exe, so -ocr works offline out of the box. Prefer the copy the extension already
# vendored; otherwise fetch it. Other languages are downloaded on demand by the app.
function Ensure-EngTessdata {
    param([string]$destDir)
    $dest = Join-Path $destDir "tessdata"
    New-Item -ItemType Directory -Force -Path $dest | Out-Null
    $engDest = Join-Path $dest "eng.traineddata"
    if (Test-Path $engDest) { return }
    $vendored = "extension/vendor/tesseract/lang/eng.traineddata"
    if (Test-Path $vendored) {
        Copy-Item -Path $vendored -Destination $engDest -Force
        Write-Host "Bundled eng.traineddata -> $dest"
        return
    }
    try {
        Invoke-WebRequest -Uri "https://github.com/tesseract-ocr/tessdata_fast/raw/main/eng.traineddata" -OutFile $engDest -UseBasicParsing
        Write-Host "Downloaded eng.traineddata -> $dest"
    } catch {
        Write-Host "NOTE: could not provision eng.traineddata ($_); -ocr will need a system tesseract or a manual download."
    }
}
Ensure-EngTessdata -destDir (Split-Path -Parent $absOutput)
Ensure-EngTessdata -destDir $deployDir
