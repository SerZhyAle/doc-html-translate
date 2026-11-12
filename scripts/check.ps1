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

  A child that exits 4 ("not looked at yet": a shared lock is held) is answered as a 2. A child
  that ends in NOT APPLICABLE (exit 0) is counted with the skipped: an otherwise clean run says
  "check: PASS (n skipped: <child>, ..)", and a run in which every child answered so says
  "check: NOT APPLICABLE (n skipped: <child>, ..)" with exit 0.

  The full gate runs scripts/help-check.ps1 with -Release, so an in-app help set older than its inputs fails; the fast
  tier runs it bare and only advises (CONTEXT-HELP rule 18 exception, ticket E0125).

  Every run writes temp/logs/gate-evidence.json: the verdict, HEAD, the plan and every child's
  verdict, and the git tree hash of the working tree the gate read (untracked files included,
  exactly what `git add -A` would commit). The tree is hashed before the children run and again
  after; when the two differ, an edit landed while the gate ran, so the evidence carries no tree
  and the verdict is COULD NOT VERIFY (a FAIL stays a FAIL).
  scripts/release.ps1 compares that tree with the one it is about to tag - BUILD-EVIDENCE: the
  release binaries are rebuilt in CI from the tag, so the only thing binding them to a tested
  state is that the tag's tree is the tree the gate passed on.

.PARAMETER Plan
  Test hook: the child scripts to run instead of the default set. The default set is the one
  configs/check-placement.jsonl declares for runner scripts/check.ps1. A subset run still writes
  evidence, with its plan in it, and scripts/release.ps1 refuses anything short of the full set.
  With -Fast it narrows the fast tier to the children of the touched area.

.PARAMETER Fast
  The between-release tier (canon TESTING_AND_QA section 2: between releases nothing above 5 s runs).
  Runs only the gate children whose configs/check-placement.jsonl record says "tier": "fast" (each
  measured at 5 s or less), all at once, and prints each child's output in plan order. A child of the
  release tier named in -Plan is refused (exit 2), never run. A fast run hashes no tree and writes no
  gate evidence: it can never unlock the release, and it never overwrites the evidence of the last
  full gate. The default (no switch) is the full gate - the release tier included - and it is what
  scripts/build-local.ps1 -Release and the release pre-flight run.
#>
param(
    [string[]]$Plan,
    [switch]$Fast
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
    'scripts/doc-registry.ps1'
    'scripts/site-addresses.ps1'
    'scripts/release-notes.ps1'
    'scripts/capability-matrix.ps1'
    'scripts/security-posture.ps1'
    'scripts/help-check.ps1'
    'scripts/settings-inventory.ps1'
)
# `pwsh -File check.ps1 -Plan a,b` binds "a,b" as one string; split it the way a call from inside
# PowerShell would have.
$Plan = @($Plan | ForEach-Object { $_ -split ',' } | Where-Object { $_ })

if ($Fast) {
    # The tier is read from the placement records, so the fast set has one home and the placement
    # test holds every fast record to its measured seconds.
    $tiers = @{}
    if (Test-Path -LiteralPath 'configs/check-placement.jsonl') {
        foreach ($l in Get-Content -LiteralPath 'configs/check-placement.jsonl' -Encoding utf8) {
            if (-not $l.Trim()) { continue }
            $r = $l | ConvertFrom-Json
            if ($r.class -eq 'gate' -and $r.runner -eq 'scripts/check.ps1') { $tiers[$r.check] = $r.tier }
        }
    }
    $fastPlan = @($defaultPlan | Where-Object { $tiers[$_] -eq 'fast' })
    if (-not $fastPlan) { Exit-Verdict 'check' 2 'configs/check-placement.jsonl records no fast-tier gate child' }
    if ($Plan) {
        $Plan = @($Plan | ForEach-Object { ($_ -replace '\\', '/') -replace '^\./', '' })
        $slow = @($Plan | Where-Object { $_ -notin $fastPlan })
        if ($slow) {
            Write-Host "not in the fast tier: $($slow -join ', ') - it runs in the full gate (scripts/check.ps1 with no -Fast), at the release"
            Exit-Verdict 'check' 2 "not in the fast tier: $($slow -join ', ')"
        }
    } else {
        $Plan = $fastPlan
    }
}
if (-not $Plan) { $Plan = $defaultPlan }

# ── gate evidence: the tree ──────────────────────────────────
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

# Hashed before any child runs: the tree the children are about to read. A fast run writes no
# evidence, so it hashes nothing.
$treeBefore = if ($Fast) { $null } else { Get-WorkingTreeHash }

$pwsh = (Get-Process -Id $PID).Path
$tierLabel = if ($Fast) { 'fast tier, in parallel' } else { 'full gate' }
Write-Subject 'check' "$($Plan.Count) child check(s), $tierLabel, $(Get-TreeLabel); per-child subjects below"

# The fast tier runs its children at once; each one's output is collected and printed below in plan
# order, so the log reads the same as a sequential run.
$runs = $null
if ($Fast) {
    $here = (Get-Location).Path
    $runs = @(0..($Plan.Count - 1) | ForEach-Object -ThrottleLimit ([Environment]::ProcessorCount) -Parallel {
            Set-Location -LiteralPath $using:here
            $child = ($using:Plan)[$_]
            $sw = [System.Diagnostics.Stopwatch]::StartNew()
            $out = @(& $using:pwsh -NoProfile -File $child *>&1 | ForEach-Object { "$_" })
            [pscustomobject]@{ Index = $_; Lines = $out; Code = $LASTEXITCODE; Seconds = $sw.Elapsed.TotalSeconds }
        } | Sort-Object Index)
}

$results = @()
for ($i = 0; $i -lt $Plan.Count; $i++) {
    $child = $Plan[$i]
    $name = [System.IO.Path]::GetFileNameWithoutExtension($child)
    Write-Host ""
    Write-Host "== $name ==" -ForegroundColor Cyan
    $logFile = "temp/logs/check-$name.log"
    if ($runs) {
        $lines = @($runs[$i].Lines)
        Set-Content -LiteralPath $logFile -Value $lines -Encoding utf8
        foreach ($line in $lines) { Write-Host $line }
        $code = $runs[$i].Code
        $seconds = $runs[$i].Seconds
    } else {
        $sw = [System.Diagnostics.Stopwatch]::StartNew()
        # The full gate holds the in-app help set to its inputs: a stale screenshot fails here, where the fast tier
        # only advises (CONTEXT-HELP rule 18 exception, ticket E0125).
        $childArgs = @()
        if ((($child -replace '\\', '/') -replace '^\./', '') -eq 'scripts/help-check.ps1') { $childArgs += '-Release' }
        $lines = @(& $pwsh -NoProfile -File $child @childArgs *>&1 | Tee-Object -FilePath $logFile | ForEach-Object { Write-Host $_; "$_" })
        $code = $LASTEXITCODE
        $seconds = $sw.Elapsed.TotalSeconds
    }
    # The wall time is the record the fast / release tier is judged from (configs/check-placement.jsonl).
    Write-Host "$name took $([math]::Round($seconds, 1).ToString([cultureinfo]::InvariantCulture)) s" -ForegroundColor DarkGray
    # The child's own verdict line, when it printed one; a child that died without one is quoted
    # by its exit code, and an unknown code is read as a failure (CHECK-VERDICT section 6).
    $verdict = @($lines | Where-Object { $_ -match "^${name}: (PASS|FAIL|COULD NOT VERIFY|NOT APPLICABLE)" }) | Select-Object -Last 1
    if (-not $verdict) { $verdict = "${name}: (no verdict line; exit $code)" }
    # NOT APPLICABLE travels with exit 0 (CHECK-VERDICT 10A): the caller switched the work off, so it
    # is counted as skipped, never as passed. The exit code wins over the word.
    $notApplicable = ($code -eq 0) -and ($verdict -match "^${name}: NOT APPLICABLE")
    # Exit 4 is "not looked at yet" (a lock held by another run), not a verdict: an aggregator that
    # does not queue answers it as could-not-verify (rule 2.1); any other unknown code is a failure.
    if ($code -eq 4) { $code = 2 } elseif ($code -notin 0, 1, 2, 3) { $code = 1 }
    $results += [pscustomobject]@{ Name = $name; Code = $code; Verdict = $verdict; NotApplicable = $notApplicable }
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

# Hashed again after the last child: a different tree means an edit landed while the gate ran, and
# the children judged a mix of the two. That is not evidence about either, so it is not a pass.
$treeAfter = if ($Fast) { $null } else { Get-WorkingTreeHash }
$treeMoved = $treeBefore -ne $treeAfter
if ($treeMoved) {
    Write-Host "  the working tree changed while the gate ran (tree $treeBefore before, $treeAfter after)" -ForegroundColor Magenta
    if ($code -ne 1) { $code = 2; $detail = 'the working tree changed during the run' }
}

# Not-applicable children are counted with the skipped, in the parenthesis of an otherwise clean
# verdict. Only when every quoted child answered so does the aggregator answer NOT APPLICABLE
# (exit 0, still not a pass); any other mix follows the precedence above.
$skipped = @($results | Where-Object NotApplicable)
$word = Get-VerdictWord $code
if ($code -eq 0 -and $skipped) {
    $detail = "$($skipped.Count) skipped: " + (($skipped | ForEach-Object Name) -join ', ')
    if ($skipped.Count -eq $results.Count) { $word = 'NOT APPLICABLE' }
}

if (-not $Fast) {
    $prevEap = $ErrorActionPreference
    $ErrorActionPreference = 'SilentlyContinue'
    $head = (& git rev-parse HEAD 2>$null)
    $dirty = [bool](& git status --porcelain 2>$null)
    $ErrorActionPreference = $prevEap

    $evidence = [ordered]@{
        verdict    = "check: $word" + $(if ($detail) { " ($detail)" } else { '' })
        code       = $code
        head       = if ($head) { $head.Trim() } else { $null }
        # The tree scripts/release.ps1 matches against HEAD: empty when the tree moved under the gate.
        tree       = if ($treeMoved) { $null } else { $treeAfter }
        treeBefore = $treeBefore
        treeAfter  = $treeAfter
        dirty      = $dirty
        time       = (Get-Date).ToString('o')
        subject    = 'Go edition (windows/amd64) + extension edition (node); see each child banner'
        # What ran, as given. release.ps1 compares it with configs/check-placement.jsonl itself rather
        # than trusting fullPlan, which is here for a reader.
        plan       = @($Plan)
        fullPlan   = ($Plan.Count -eq $defaultPlan.Count) -and -not @($defaultPlan | Where-Object { $_ -notin $Plan })
        children   = @($results | ForEach-Object { [ordered]@{ name = $_.Name; code = $_.Code; verdict = $_.Verdict; notApplicable = $_.NotApplicable } })
    }
    $evidence | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath "temp/logs/gate-evidence.json" -Encoding utf8
}

Write-Host ""
if ($word -eq 'NOT APPLICABLE') {
    # verdict.ps1 maps a code to a word, and this word shares exit 0 with PASS.
    Write-Host "check: NOT APPLICABLE ($detail)" -ForegroundColor Yellow
    exit 0
}
Exit-Verdict 'check' $code $detail
