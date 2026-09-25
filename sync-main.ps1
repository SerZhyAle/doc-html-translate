#!/usr/bin/env pwsh
<#
.SYNOPSIS
  Pull everything, merge every branch that is ahead of main back into main, push main.

.DESCRIPTION
  From the repo root it:
    1. Fetches all remotes (with prune).
    2. Stashes uncommitted work (tracked + untracked) if the tree is dirty.
    3. Checks out main and fast-forwards it from origin/main.
    4. Merges every local branch and every origin/* branch that has commits main
       does not have. A clean fast-forward stays a fast-forward; otherwise a merge
       commit is made. A conflict confined to the append-only ledgers in -UnionFiles
       (default DEV/CHANGELOG.md, where every branch adds rows on top) is resolved by
       keeping both sides. Any other conflict is aborted and reported, never
       half-merged - the run continues with the next branch.
    5. Runs `go build ./...` as a cheap gate (skip with -SkipBuild). A red build stops
       the run before the push; the merges stay local for inspection.
    6. Pushes main to origin (skip with -NoPush).
    7. Returns to the branch you started on and restores the stash.

  Branches matching -Exclude (default: backup/*) are never merged.
  This pushes a BRANCH only - no tag, no release; every CI workflow fires on tags.

.PARAMETER DryRun
  Fetch and list what would be merged; change nothing.

.PARAMETER NoPush
  Merge locally, do not push.

.PARAMETER SkipBuild
  Do not run the `go build ./...` gate before the push.

.PARAMETER Exclude
  Branch-name wildcards (without the origin/ prefix) never to merge. Default: backup/*.

.PARAMETER UnionFiles
  Repo-relative append-only files whose conflicts are resolved by keeping both sides.

.EXAMPLE
  .\sync-main.ps1 -DryRun
.EXAMPLE
  .\sync-main.ps1 -Exclude 'backup/*','wip/*'

.NOTES
  Exit codes: 0 = done; 1 = error (nothing pushed); 2 = pushed, but some branches
  conflicted and were skipped.
#>
[CmdletBinding()]
param(
    [switch]$DryRun,
    [switch]$NoPush,
    [switch]$SkipBuild,
    [string[]]$Exclude = @('backup/*'),
    [string[]]$UnionFiles = @('DEV/CHANGELOG.md'),
    [string]$Remote = 'origin',
    [string]$Main = 'main'
)

$ErrorActionPreference = 'Stop'
Set-Location -LiteralPath $PSScriptRoot

function Invoke-Git {
    param([Parameter(ValueFromRemainingArguments)][string[]]$GitArgs)
    $out = & git @GitArgs 2>&1
    if ($LASTEXITCODE -ne 0) {
        throw "git $($GitArgs -join ' ') failed (exit $LASTEXITCODE):`n$($out -join "`n")"
    }
    $out
}

# After a failed merge: when every conflicted file is an append-only ledger in
# $UnionFiles, keep both sides of each hunk (the branch's lines first - ledgers are
# newest-on-top) and commit the merge. Any other conflicted file -> $false.
function Resolve-UnionConflicts {
    $unmerged = @(& git diff --name-only --diff-filter=U)
    if ($unmerged.Count -eq 0) { return $false }
    foreach ($f in $unmerged) {
        if ($UnionFiles -notcontains $f) { return $false }
    }
    $hunk = [regex]::new('<<<<<<< [^\n]*\n(.*?)=======\r?\n(.*?)>>>>>>> [^\n]*\n', 'Singleline')
    $utf8 = [System.Text.UTF8Encoding]::new($false)
    foreach ($f in $unmerged) {
        $path = Join-Path $PSScriptRoot $f
        $text = $hunk.Replace([IO.File]::ReadAllText($path), { param($m) $m.Groups[2].Value + $m.Groups[1].Value })
        if ($text -match '(?m)^(<<<<<<<|>>>>>>>) ') { return $false }
        [IO.File]::WriteAllText($path, $text, $utf8)
        Invoke-Git add -- $f | Out-Null
    }
    Invoke-Git commit --no-edit | Out-Null
    $true
}

function Test-Excluded([string]$Name) {
    foreach ($pattern in $Exclude) {
        if ($Name -like $pattern) { return $true }
    }
    $false
}

Write-Host "== fetch --all --prune" -ForegroundColor Cyan
Invoke-Git fetch --all --prune | Out-Host

# Candidates: local branches and origin/* branches, minus main, HEAD and exclusions.
$candidates = [System.Collections.Generic.List[string]]::new()
$refs = Invoke-Git for-each-ref --format='%(refname)' "refs/heads" "refs/remotes/$Remote"
foreach ($ref in $refs) {
    if ($ref -match "^refs/heads/(.+)$") {
        $name = $Matches[1]; $short = $name
    } elseif ($ref -match "^refs/remotes/$Remote/(.+)$") {
        $name = $Matches[1]; $short = "$Remote/$name"
    } else { continue }
    if ($name -eq $Main -or $name -eq 'HEAD' -or (Test-Excluded $name)) { continue }
    $candidates.Add($short)
}

$mainRef = "$Remote/$Main"
$pending = @($candidates | Where-Object { [int](Invoke-Git rev-list --count "$mainRef..$_") -gt 0 })

if ($pending.Count -eq 0) {
    Write-Host "No branch is ahead of $mainRef." -ForegroundColor Green
} else {
    Write-Host "Branches ahead of ${mainRef}:" -ForegroundColor Cyan
    foreach ($b in $pending) {
        $n = Invoke-Git rev-list --count "$mainRef..$b"
        Write-Host "  $b (+$n)"
    }
}

if ($DryRun) {
    Write-Host "DryRun: nothing changed." -ForegroundColor Yellow
    exit 0
}

$startBranch = (Invoke-Git rev-parse --abbrev-ref HEAD) | Select-Object -First 1
$stashed = $false
$conflicts = @()
$merged = @()
$exitCode = 0

try {
    if (Invoke-Git status --porcelain) {
        $msg = "sync-main autostash $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')"
        Write-Host "== stash uncommitted work: $msg" -ForegroundColor Cyan
        Invoke-Git stash push -u -m $msg | Out-Host
        $stashed = $true
    }

    Write-Host "== checkout $Main + fast-forward from $mainRef" -ForegroundColor Cyan
    Invoke-Git checkout $Main | Out-Host
    Invoke-Git merge --ff-only $mainRef | Out-Host

    foreach ($b in $pending) {
        # An earlier merge may already have brought this branch in.
        if ([int](Invoke-Git rev-list --count "$Main..$b") -eq 0) {
            Write-Host "  $b - already in $Main" -ForegroundColor DarkGray
            continue
        }
        Write-Host "== merge $b" -ForegroundColor Cyan
        $out = & git merge --no-edit $b 2>&1
        if ($LASTEXITCODE -eq 0) {
            $out | Select-Object -Last 2 | Out-Host
            $merged += $b
        } elseif (Resolve-UnionConflicts) {
            Write-Host "  resolved $($UnionFiles -join ', ') by keeping both sides' rows" -ForegroundColor Yellow
            $merged += $b
        } else {
            $out | Out-Host
            & git merge --abort 2>&1 | Out-Null
            Write-Host "  CONFLICT in $b - merge aborted, branch skipped" -ForegroundColor Red
            $conflicts += $b
        }
    }

    $ahead = [int](Invoke-Git rev-list --count "$mainRef..$Main")
    if ($ahead -eq 0) {
        Write-Host "$Main is already at $mainRef; nothing to push." -ForegroundColor Green
    } else {
        if (-not $SkipBuild) {
            Write-Host "== gate: go build ./..." -ForegroundColor Cyan
            & go build ./...
            if ($LASTEXITCODE -ne 0) {
                throw "go build failed after the merges; nothing pushed. Inspect $Main (ahead of $mainRef by $ahead)."
            }
        }
        if ($NoPush) {
            Write-Host "NoPush: $Main is ahead of $mainRef by $ahead, not pushed." -ForegroundColor Yellow
        } else {
            Write-Host "== push $Remote $Main" -ForegroundColor Cyan
            Invoke-Git push $Remote $Main | Out-Host
        }
    }
} catch {
    Write-Host $_ -ForegroundColor Red
    $exitCode = 1
} finally {
    if ($startBranch -and $startBranch -ne $Main -and $startBranch -ne 'HEAD' -and $exitCode -eq 0) {
        & git checkout $startBranch 2>&1 | Out-Host
    }
    if ($stashed) {
        Write-Host "== restore stash" -ForegroundColor Cyan
        & git stash pop 2>&1 | Out-Host
        if ($LASTEXITCODE -ne 0) {
            Write-Host "stash pop did not apply cleanly - your work is still in 'git stash list'." -ForegroundColor Red
        }
    }
}

Write-Host ""
Write-Host "Merged:  $(if ($merged) { $merged -join ', ' } else { '(none)' })"
if ($conflicts) {
    Write-Host "Skipped (conflict, merge by hand): $($conflicts -join ', ')" -ForegroundColor Red
    if ($exitCode -eq 0) { $exitCode = 2 }
}
exit $exitCode
