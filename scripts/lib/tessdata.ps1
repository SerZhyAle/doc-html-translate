# Dot-sourced by every desktop packaging path (build.ps1, build-ui.ps1, build-installer.ps1,
# msix/build-msix.ps1, the release workflow's zip). It puts the bundled English OCR data,
# <dir>/tessdata/eng.traineddata, next to the exe - where internal/ocr/tessdata.go bundledDataDir
# looks for it - and refuses to ship anything that is not the pinned build.
#
# The pin is tessdata_fast 4.0.0 eng, the same bytes as internal/ocr/download.go packDigests["eng"]
# and extension/build.mjs ENG_TRAINEDDATA_SHA256 (decompressed). tests/tessdata_pin_test.go holds the
# three together; the version is a both-editions parity invariant (docs/PARITY.md "OCR").

$script:EngTessdataUrl    = "https://github.com/tesseract-ocr/tessdata_fast/raw/4.0.0/eng.traineddata"
$script:EngTessdataSize   = 4113088
$script:EngTessdataSha256 = "7d4322bd2a7749724879683fc3912cb542f19906c83bcc1a52132556427170b2"

# Test-EngTessdata reports whether a file is exactly the pinned build: size first (cheap), then SHA-256.
function Test-EngTessdata([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) { return $false }
    if ((Get-Item -LiteralPath $Path).Length -ne $script:EngTessdataSize) { return $false }
    return (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant() -eq $script:EngTessdataSha256
}

# Install-EngTessdata provisions <DestDir>/tessdata/eng.traineddata or throws - a package without
# the English data is not built. Sources, each verified before it is used:
#   1. the extension's vendored copy (npm run vendor, itself digest-checked by build.mjs);
#      a copy that fails the check is an error, not something to route around;
#   2. a download cache under the repo's temp/ (gitignored); a cache that fails the check is
#      discarded and fetched again;
#   3. the pinned 4.0.0 URL, with a timeout, into the cache.
# A destination file that already matches is left alone; one that does not is replaced.
function Install-EngTessdata {
    param(
        [Parameter(Mandatory)][string]$DestDir,
        [string]$Vendored = (Join-Path (Split-Path -Parent (Split-Path -Parent $PSScriptRoot)) "extension/vendor/tesseract/lang/eng.traineddata"),
        [string]$CacheDir = (Join-Path (Split-Path -Parent (Split-Path -Parent $PSScriptRoot)) "temp/tessdata-cache"),
        [string]$Url = $script:EngTessdataUrl,
        [int]$TimeoutSec = 300
    )
    $tessDir = Join-Path $DestDir "tessdata"
    New-Item -ItemType Directory -Force -Path $tessDir | Out-Null
    $dest = Join-Path $tessDir "eng.traineddata"
    if (Test-Path -LiteralPath $dest) {
        if (Test-EngTessdata $dest) { return }
        Write-Host "eng.traineddata in $tessDir does not match the pinned digest; replacing it" -ForegroundColor Yellow
        Remove-Item -LiteralPath $dest -Force
    }

    if (Test-Path -LiteralPath $Vendored) {
        if (-not (Test-EngTessdata $Vendored)) {
            throw "vendored $Vendored does not match the pinned tessdata_fast 4.0.0 digest ($script:EngTessdataSha256); re-run 'npm run vendor' in extension/ or delete the file"
        }
        Copy-Item -LiteralPath $Vendored -Destination $dest -Force
        Write-Host "Bundled eng.traineddata (vendored, digest verified) -> $tessDir"
        return
    }

    New-Item -ItemType Directory -Force -Path $CacheDir | Out-Null
    $cached = Join-Path $CacheDir "eng.traineddata"
    if ((Test-Path -LiteralPath $cached) -and -not (Test-EngTessdata $cached)) {
        Write-Host "cached $cached does not match the pinned digest; fetching it again" -ForegroundColor Yellow
        Remove-Item -LiteralPath $cached -Force
    }
    if (-not (Test-Path -LiteralPath $cached)) {
        $tmp = "$cached.download.tmp"
        Remove-Item -LiteralPath $tmp -Force -ErrorAction SilentlyContinue
        try {
            Invoke-WebRequest -Uri $Url -OutFile $tmp -UseBasicParsing -TimeoutSec $TimeoutSec
        } catch {
            Remove-Item -LiteralPath $tmp -Force -ErrorAction SilentlyContinue
            throw "could not download eng.traineddata from ${Url}: $_"
        }
        if (-not (Test-EngTessdata $tmp)) {
            $got = if (Test-Path -LiteralPath $tmp) { (Get-FileHash -LiteralPath $tmp -Algorithm SHA256).Hash.ToLowerInvariant() } else { "nothing" }
            Remove-Item -LiteralPath $tmp -Force -ErrorAction SilentlyContinue
            throw "eng.traineddata from $Url does not match the pinned digest: got $got, pinned $script:EngTessdataSha256"
        }
        Move-Item -LiteralPath $tmp -Destination $cached -Force
        Write-Host "Downloaded eng.traineddata (digest verified) -> $CacheDir"
    }
    Copy-Item -LiteralPath $cached -Destination $dest -Force
    Write-Host "Bundled eng.traineddata (cached, digest verified) -> $tessDir"
}
