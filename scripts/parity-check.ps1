<#
.SYNOPSIS
  Flag cross-edition parity DRIFT: a Go extractor changed without its paired JS module
  (or docs/PARITY.md), or vice versa. Advisory by default - it warns, it does not block.

.DESCRIPTION
  The app ships two independent codebases with no shared code (Go app + JS extension);
  logic is hand-ported Go <-> JS and drifts silently. tests/parity_test.go already guards
  the VALUE invariants (theme palette, OCR/reflow constants). This guards the STRUCTURAL
  signal those tests can't see: one side of a ported capability moved and the other did not.

  It reads the change set (staged by default) and, for each paired capability in the port
  map below (mirrored from docs/PARITY.md "The port map"), warns when exactly one side is
  touched. Touching docs/PARITY.md or tests/parity_test.go in the same change set is treated
  as "parity was considered" and silences the warnings - so the escape hatch is to update the
  map/tests, exactly what an intentional divergence needs anyway.

  It never inspects intentionally one-sided code (sanitize.js, lang.js, Go-only image copy,
  ..) - those are not in the map, so a one-sided change there is silent by design.

.PARAMETER Range
  A git range to inspect instead of the staged diff, e.g. "origin/main...HEAD" (CI use) or
  "HEAD~1..HEAD". Overrides the default staged/working-tree detection.

.PARAMETER Strict
  Exit non-zero when drift is found (for a CI gate). Default: advisory, always exit 0.

.EXAMPLE
  ./scripts/parity-check.ps1                       # check staged changes, warn only
.EXAMPLE
  ./scripts/parity-check.ps1 -Range origin/main...HEAD -Strict   # CI gate
#>
param(
    [string]$Range,
    [switch]$Strict
)

$ErrorActionPreference = "Stop"

$repoRoot = (& git rev-parse --show-toplevel 2>$null)
if ($LASTEXITCODE -eq 0 -and $repoRoot) { Set-Location ($repoRoot.Trim()) }

# ── the port map (mirror of docs/PARITY.md "The port map (Go <-> JS)") ────────
# Each entry is a PAIRED capability: both sides are real hand-ports that should move
# together. Go/Js are path prefixes (a trailing '/' means "anything under this dir").
# Intentionally one-sided code (sanitize.js, lang.js, Go-only HTML image copy, ..) is
# deliberately absent - a one-sided change there is not drift.
$map = @(
    @{ Name = 'PDF (reflow / TOC / page-images)'; Go = @('internal/pdf/'); Js = @('extension/src/reflow.js', 'extension/src/toc.js', 'extension/src/pdf-images.js') }
    @{ Name = 'EPUB (unzip / spine / TOC)';       Go = @('internal/epub/'); Js = @('extension/src/epub.js') }
    @{ Name = 'Plain text (paragraphs / decode)'; Go = @('internal/txt/'); Js = @('extension/src/txt.js') }
    @{ Name = 'RTF';                              Go = @('internal/rtf/'); Js = @('extension/src/rtf.js') }
    @{ Name = 'Markdown';                         Go = @('internal/md/'); Js = @('extension/src/md.js') }
    @{ Name = 'FB2';                              Go = @('internal/fb2/'); Js = @('extension/src/fb2.js') }
    @{ Name = 'HTML body extract';                Go = @('internal/htmlconv/'); Js = @('extension/src/html.js') }
    @{ Name = 'MOBI / AZW3';                      Go = @('internal/mobi/'); Js = @('extension/src/ebook.js') }
    @{ Name = 'Comic archives (CBZ/CBR/CB7/CBT)'; Go = @('internal/comic/'); Js = @('extension/src/comic.js') }
    @{ Name = 'OCR overlay / language';           Go = @('internal/ocr/'); Js = @('extension/src/ocr-overlay.js', 'extension/src/ocr-text.js', 'extension/src/ocr-lang.js') }
    @{ Name = 'Reader chrome (themes/fonts/UI)';  Go = @('internal/htmlgen/navbar.go'); Js = @('extension/src/viewer.css', 'extension/src/viewer.js', 'extension/src/viewer.html') }
    @{ Name = 'Settings / options surface';       Go = @('internal/config/flags.go'); Js = @('extension/src/popup.js', 'extension/src/options.js', 'extension/src/background.js') }
)

# Touching either of these = parity was acknowledged -> suppress warnings.
$ackFiles = @('docs/PARITY.md', 'tests/parity_test.go')

# ── the change set ───────────────────────────────────────────
if ($Range) {
    $changed = git diff --name-only $Range
} else {
    $changed = git diff --cached --name-only            # staged (the commit-time view)
    if (-not $changed) { $changed = git diff --name-only }   # fall back to the working tree
}
$changed = @($changed | Where-Object { $_ } | ForEach-Object { $_.Replace('\', '/') })

if (-not $changed) {
    Write-Host "parity-check: no changes to inspect." -ForegroundColor DarkGray
    exit 0
}

function Test-Touched([string[]]$patterns) {
    foreach ($p in $patterns) {
        foreach ($f in $changed) {
            if ($p.EndsWith('/')) { if ($f -like "$p*") { return $true } }
            elseif ($f -eq $p) { return $true }
        }
    }
    return $false
}

function Get-Touched([string[]]$patterns) {
    $hits = @()
    foreach ($p in $patterns) {
        foreach ($f in $changed) {
            if ($p.EndsWith('/')) { if ($f -like "$p*") { $hits += $f } }
            elseif ($f -eq $p) { $hits += $f }
        }
    }
    return ($hits | Select-Object -Unique)
}

$acked = Test-Touched $ackFiles

# ── evaluate each paired capability ──────────────────────────
$warnings = @()
foreach ($cap in $map) {
    $goHit = Get-Touched $cap.Go
    $jsHit = Get-Touched $cap.Js
    if (($goHit -and -not $jsHit) -or ($jsHit -and -not $goHit)) {
        $warnings += [pscustomobject]@{
            Name    = $cap.Name
            Side    = if ($goHit) { 'Go' } else { 'JS' }
            Touched = if ($goHit) { $goHit } else { $jsHit }
            Missing = if ($goHit) { $cap.Js } else { $cap.Go }
        }
    }
}

# ── report ───────────────────────────────────────────────────
if (-not $warnings) {
    Write-Host "parity-check: no cross-edition drift in the change set." -ForegroundColor Green
    exit 0
}

if ($acked) {
    Write-Host "parity-check: $($warnings.Count) one-sided capability change(s), but docs/PARITY.md (or tests/parity_test.go) was updated - treated as acknowledged." -ForegroundColor DarkGray
    foreach ($w in $warnings) { Write-Host "  - $($w.Name): $($w.Side) side only" -ForegroundColor DarkGray }
    exit 0
}

Write-Host ""
Write-Host "parity-check: possible cross-edition DRIFT (one side changed, the other did not)" -ForegroundColor Yellow
foreach ($w in $warnings) {
    Write-Host ""
    Write-Host "  [$($w.Name)] - $($w.Side) side changed, counterpart untouched" -ForegroundColor Yellow
    foreach ($t in $w.Touched) { Write-Host "      changed : $t" -ForegroundColor Gray }
    Write-Host  ("      expected: " + ($w.Missing -join ', ')) -ForegroundColor DarkGray
}
Write-Host ""
Write-Host "  -> Port the change to the other edition, OR record it in docs/PARITY.md" -ForegroundColor DarkGray
Write-Host "     (an intentional divergence goes under 'Intentional divergences'; touching PARITY.md silences this)." -ForegroundColor DarkGray
Write-Host ""

if ($Strict) { exit 1 }
exit 0
