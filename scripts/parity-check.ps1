<#
.SYNOPSIS
  Flag cross-edition parity DRIFT: a Go extractor changed without its paired JS module
  (or docs/PARITY.md), or vice versa. Advisory by default - it warns, it does not block.

.DESCRIPTION
  The app ships two independent codebases with no shared code (Go app + JS extension);
  logic is hand-ported Go <-> JS and drifts silently. tests/parity_test.go already guards
  the VALUE invariants (theme palette, OCR/reflow constants). This guards the STRUCTURAL
  signal those tests can't see: one side of a ported capability moved and the other did not.

  It reads the change set (staged by default) and, for each paired capability in
  configs/parity-map.json (the machine-readable twin of docs/PARITY.md "The port map";
  tests/parity_map_test.go fails when the two disagree), warns when exactly one side is
  touched. Touching docs/PARITY.md, tests/parity_test.go or the map in the same change set is
  treated as "parity was considered" and silences the warnings - so the escape hatch is to
  update the map/tests, exactly what an intentional divergence needs anyway.

  Exit codes follow CHECK-VERDICT (scripts/lib/verdict.ps1):
    0  no drift in the change set - including an empty change set, reported as "0 file(s)
       inspected" (a vacuous pass; see the contract exception on rule 2.1 in the registry)
    3  drift found - an advisory: the finding is named, the caller decides
    1  drift found and -Strict was passed
    2  the change set could not be determined (not a git checkout, a bad -Range, a git failure)
       or the map could not be read

.PARAMETER Range
  A git range to inspect instead of the staged diff, e.g. "origin/main...HEAD" (CI use) or
  "HEAD~1..HEAD". Overrides the default staged/working-tree detection.

.PARAMETER Strict
  Exit 1 when drift is found (for a CI gate). Default: advisory, exit 3.

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
. "$PSScriptRoot/lib/verdict.ps1"

$repoRoot = (& git rev-parse --show-toplevel 2>$null)
if ($LASTEXITCODE -ne 0 -or -not $repoRoot) {
    Write-Host "parity-check: not inside a git checkout - the change set cannot be determined."
    Exit-Verdict 'parity-check' 2 'no git checkout'
}
Set-Location ($repoRoot.Trim())

# ── the port map ─────────────────────────────────────────────
$mapPath = 'configs/parity-map.json'
try {
    $mapDoc = Get-Content -LiteralPath $mapPath -Raw | ConvertFrom-Json
} catch {
    Write-Host "parity-check: cannot read ${mapPath}: $($_.Exception.Message)"
    Exit-Verdict 'parity-check' 2 'port map unreadable'
}
$map = @($mapDoc.pairs)
$ackFiles = @($mapDoc.acknowledge)

# ── the change set ───────────────────────────────────────────
function Get-GitNames([string[]]$gitArgs) {
    $out = & git @gitArgs 2>$null
    if ($LASTEXITCODE -ne 0) {
        Write-Host "parity-check: 'git $($gitArgs -join ' ')' failed (exit $LASTEXITCODE) - the change set cannot be determined."
        Exit-Verdict 'parity-check' 2 'git diff failed'
    }
    return @($out | Where-Object { $_ } | ForEach-Object { $_.Replace('\', '/') })
}

if ($Range) {
    $source = "range $Range"
    $changed = Get-GitNames @('diff', '--name-only', $Range)
} else {
    $source = 'staged changes'
    $changed = Get-GitNames @('diff', '--cached', '--name-only')          # the commit-time view
    if (-not $changed) {
        $source = 'working-tree changes'
        $changed = Get-GitNames @('diff', '--name-only')                  # fall back to the working tree
    }
}
Write-Subject 'parity-check' "$source, $($changed.Count) file(s), $($map.Count) paired capabilities from $mapPath"

if (-not $changed) {
    Exit-Verdict 'parity-check' 0 '0 file(s) inspected - empty change set'
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

$acked = [bool](Get-Touched $ackFiles)

# ── evaluate each paired capability ──────────────────────────
$warnings = @()
foreach ($cap in $map) {
    $goHit = Get-Touched @($cap.go)
    $jsHit = Get-Touched @($cap.js)
    if (($goHit -and -not $jsHit) -or ($jsHit -and -not $goHit)) {
        $warnings += [pscustomobject]@{
            Name    = $cap.name
            Side    = if ($goHit) { 'Go' } else { 'JS' }
            Touched = if ($goHit) { $goHit } else { $jsHit }
            Missing = if ($goHit) { @($cap.js) } else { @($cap.go) }
        }
    }
}

# ── report ───────────────────────────────────────────────────
if (-not $warnings) {
    Exit-Verdict 'parity-check' 0 "$($changed.Count) file(s) inspected, no one-sided change"
}

if ($acked) {
    Write-Host "parity-check: $($warnings.Count) one-sided capability change(s), but docs/PARITY.md, tests/parity_test.go or the map was updated - treated as acknowledged." -ForegroundColor DarkGray
    foreach ($w in $warnings) { Write-Host "  - $($w.Name): $($w.Side) side only" -ForegroundColor DarkGray }
    Exit-Verdict 'parity-check' 0 "$($changed.Count) file(s) inspected, $($warnings.Count) acknowledged"
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

$names = ($warnings | ForEach-Object { $_.Name }) -join '; '
if ($Strict) { Exit-Verdict 'parity-check' 1 "$($warnings.Count): $names" }
Exit-Verdict 'parity-check' 3 "$($warnings.Count): $names"
