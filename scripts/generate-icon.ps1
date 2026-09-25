param(
    # Also copy the program icon here (the installer stages it next to its payload).
    [string]$Output = "assets/doc-html-translate.ico"
)

$ErrorActionPreference = "Stop"

# Draw every committed system-surface icon from the product mark and the vendored glyphs
# (internal/iconart, ICON-RENDER rule 9): the program ICO and its two favicon copies, the
# Explorer verb and document-type ICOs embedded beside it, and the extension's action icons.
# They are render targets - tests/icons_test.go fails when one drifts from this output.

$repoRoot = Split-Path -Parent $PSScriptRoot
Push-Location $repoRoot
try {
    # A caller may have left GOOS/GOARCH set for a cross build; the generator runs here.
    $savedArch = $env:GOARCH; $savedOS = $env:GOOS
    Remove-Item Env:GOARCH, Env:GOOS -ErrorAction SilentlyContinue
    try {
        go run ./tools/icongen | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "icongen failed" }
    } finally {
        if ($savedArch) { $env:GOARCH = $savedArch }
        if ($savedOS) { $env:GOOS = $savedOS }
    }
} finally {
    Pop-Location
}

$program = Join-Path $repoRoot "assets/doc-html-translate.ico"
if ($Output) {
    $dest = if ([IO.Path]::IsPathRooted($Output)) { $Output } else { Join-Path $repoRoot $Output }
    if ([IO.Path]::GetFullPath($dest) -ne [IO.Path]::GetFullPath($program)) {
        New-Item -ItemType Directory -Force -Path (Split-Path -Parent $dest) | Out-Null
        Copy-Item $program $dest -Force
    }
}

Write-Host "Icons generated: assets/doc-html-translate.ico, convert-verb.ico, document-type.ico, extension/icons"
