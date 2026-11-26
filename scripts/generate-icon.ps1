param(
    # Also copy the program icon here (the installer stages it next to its payload).
    [string]$Output = "assets/doc-html-translate.ico"
)

$ErrorActionPreference = "Stop"

# The one generator of PRODUCT-IDENTITY 0.1 (docs/contracts/PRODUCT-IDENTITY.md): every picture of the
# product is drawn from the one mark (internal/iconart/mark.go) and the vendored glyphs (ICON-RENDER rule 9).
#   1. go run ./tools/icongen - the master and its reduced forms as SVG (assets/identity/master), the help-page
#      head, the program ICO and its favicon copies, the Explorer verb and document-type ICOs, the extension's
#      action icons, the Microsoft Store display images (tools/store/logos) and the social card's mark;
#   2. tools/store/make-social-card.ps1 - the Open Graph card and the GitHub social preview, composed from that mark;
#   3. tools/store/make-chrome-promo.ps1 - the Chrome tile from the mark and extension screenshot;
#   4. go run ./tools/icongen -manifest - assets/identity/identity-manifest.json, every picture with its SHA-256.
# All are render targets - tests/icons_test.go fails when an icon drifts from this output, and the catalog's
# reference checker (P:/Contracts/product-identity/reference/Test-ProductIdentity.ps1) fails when a file drifts from
# the manifest. Rerun this after a store screenshot or the demo is regenerated: the manifest records their hashes too.
# The MSIX logos are drawn into the package by msix/build-msix.ps1 (icongen -msix). The help pages inline the head
# picture: run node tools/helpgen/repage.mjs after a change of the mark.

$repoRoot = Split-Path -Parent $PSScriptRoot
Push-Location $repoRoot
try {
    # A caller may have left GOOS/GOARCH set for a cross build; the generator runs here.
    $savedArch = $env:GOARCH; $savedOS = $env:GOOS
    Remove-Item Env:GOARCH, Env:GOOS -ErrorAction SilentlyContinue
    # amd64 draws, as the test suite and CI do: the 386 toolchain rounds some large renditions differently.
    $env:GOARCH = 'amd64'
    try {
        go run ./tools/icongen | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "icongen failed" }
        & (Join-Path $repoRoot "tools/store/make-social-card.ps1") | Out-Null
        & (Join-Path $repoRoot "tools/store/make-chrome-promo.ps1") | Out-Null
        go run ./tools/icongen -manifest | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "icongen -manifest failed" }
    } finally {
        if ($savedArch) { $env:GOARCH = $savedArch } else { Remove-Item Env:GOARCH -ErrorAction SilentlyContinue }
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

Write-Host "Identity generated: assets/identity (master, manifest), program and favicon ICOs, verb and document-type ICOs, extension icons, store logos, social cards"
