<#
.SYNOPSIS
  The local quality gate: runs every child check, never stops at the first failure, and ends in
  one CHECK-VERDICT line.

.DESCRIPTION
  Children run as separate pwsh processes, so each one's exit code is its own and an environment
  change in one (test.ps1 sets GOARCH) cannot leak into the next. Every child runs whatever the one
  before it returned (CHECK-VERDICT rule 9), and its own verdict line is quoted in the summary.

  The last line is one of
      check: PASS
      check: PASS WITH ADVISORIES (n: <child>, ..)
      check: COULD NOT VERIFY (n: <child>, ..)
      check: FAIL (n: <child>, ..)
  with exit code 0, 3, 2 or 1. Precedence: any FAIL is a FAIL; otherwise any could-not-verify is
  a COULD NOT VERIFY, because a green line over an unrun check is the lie the contract forbids;
  advisories only colour an otherwise clean run. Only the bare PASS is a clean verdict.

  Every run writes temp/logs/gate-evidence.json: the verdict, HEAD, and the git tree hash of the
  working tree the gate read (untracked files included, exactly what `git add -A` would commit).
  scripts/release.ps1 compares that tree with the one it is about to tag - BUILD-EVIDENCE: the
  release binaries are rebuilt in CI from the tag, so the only thing binding them to a tested
  state is that the tag's tree is the tree the gate passed on.

.PARAMETER Plan
  Test hook: the child scripts to run instead of the default set. The default set is the one
  configs/check-placement.jsonl declares for runner scripts/check.ps1.
#>
param(
    [string[]]$Plan
)

$ErrorActionPreference = "Stop"
. "$PSScriptRoot/lib/verdict.ps1"

New-Item -ItemType Directory -Force -Path "temp/logs" | Out-Null

# The default children. tests/placement_test.go compares this list with configs/check-placement.jsonl
# in both directions, so a check added here without a record (or recorded without being run) fails.
$defaultPlan = @(
    'scripts/test.ps1'
    'scripts/test-extension.ps1'
    'scripts/lint.ps1'
    'scripts/typo.ps1'
    'scripts/parity-check.ps1'
)
if (-not $Plan) { $Plan = $defaultPlan }
# `pwsh -File check.ps1 -Plan a,b` binds "a,b" as one string; split it the way a call from inside
# PowerShell would have.
$Plan = @($Plan | ForEach-Object { $_ -split ',' } | Where-Object { $_ })

$pwsh = (Get-Process -Id $PID).Path
Write-Subject 'check' "$($Plan.Count) child check(s), $(Get-TreeLabel); per-child subjects below"

$results = @()
foreach ($child in $Plan) {
    $name = [System.IO.Path]::GetFileNameWithoutExtension($child)
    Write-Host ""
    Write-Host "== $name ==" -ForegroundColor Cyan
    $logFile = "temp/logs/check-$name.log"
    $lines = @(& $pwsh -NoProfile -File $child *>&1 | Tee-Object -FilePath $logFile | ForEach-Object { Write-Host $_; "$_" })
    $code = $LASTEXITCODE
    # The child's own verdict line, when it printed one; a child that died without one is quoted
    # by its exit code, and an unknown code is read as a failure (CHECK-VERDICT section 6).
    $verdict = @($lines | Where-Object { $_ -match "^${name}: (PASS|FAIL|COULD NOT VERIFY)" }) | Select-Object -Last 1
    if (-not $verdict) { $verdict = "${name}: (no verdict line; exit $code)" }
    if ($code -notin 0, 1, 2, 3) { $code = 1 }
    $results += [pscustomobject]@{ Name = $name; Code = $code; Verdict = $verdict }
}

Write-Host ""
Write-Host "summary:" -ForegroundColor Cyan
foreach ($r in $results) { Write-Host "  [$($r.Code)] $($r.Verdict)" }

$failed = @($results | Where-Object Code -EQ 1)
$unverified = @($results | Where-Object Code -EQ 2)
$advisory = @($results | Where-Object Code -EQ 3)
if ($failed) { $code = 1; $named = $failed }
elseif ($unverified) { $code = 2; $named = $unverified }
elseif ($advisory) { $code = 3; $named = $advisory }
else { $code = 0; $named = @() }
$detail = if ($named) { "$($named.Count): " + (($named | ForEach-Object Name) -join ', ') } else { '' }

# ── gate evidence ────────────────────────────────────────────
# The tree hash comes from a scratch copy of the index, so the real index (what the user staged)
# is never touched. Copying the index keeps git's stat cache: `git add -A` then re-hashes only
# what changed instead of every tracked file.
function Get-WorkingTreeHash {
    $prevEap = $ErrorActionPreference
    $ErrorActionPreference = 'SilentlyContinue'
    try {
        $indexPath = (& git rev-parse --git-path index 2>$null)
        if ($LASTEXITCODE -ne 0 -or -not $indexPath) { return $null }
        $scratch = Join-Path ([System.IO.Path]::GetTempPath()) ("check-index-" + [System.IO.Path]::GetRandomFileName())
        try {
            if (Test-Path -LiteralPath $indexPath.Trim()) { Copy-Item -LiteralPath $indexPath.Trim() -Destination $scratch }
            $env:GIT_INDEX_FILE = $scratch
            & git add -A 2>$null | Out-Null
            if ($LASTEXITCODE -ne 0) { return $null }
            $tree = (& git write-tree 2>$null)
            if ($LASTEXITCODE -ne 0) { return $null }
            return $tree.Trim()
        } finally {
            Remove-Item Env:GIT_INDEX_FILE -ErrorAction SilentlyContinue
            Remove-Item -LiteralPath $scratch -ErrorAction SilentlyContinue
        }
    } finally {
        $ErrorActionPreference = $prevEap
    }
}

$prevEap = $ErrorActionPreference
$ErrorActionPreference = 'SilentlyContinue'
$head = (& git rev-parse HEAD 2>$null)
$dirty = [bool](& git status --porcelain 2>$null)
$ErrorActionPreference = $prevEap

$evidence = [ordered]@{
    verdict  = "check: $(Get-VerdictWord $code)" + $(if ($detail) { " ($detail)" } else { '' })
    code     = $code
    head     = if ($head) { $head.Trim() } else { $null }
    tree     = Get-WorkingTreeHash
    dirty    = $dirty
    time     = (Get-Date).ToString('o')
    subject  = 'Go edition (windows/amd64) + extension edition (node); see each child banner'
    children = @($results | ForEach-Object { [ordered]@{ name = $_.Name; code = $_.Code; verdict = $_.Verdict } })
}
$evidence | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath "temp/logs/gate-evidence.json" -Encoding utf8

Write-Host ""
Exit-Verdict 'check' $code $detail
