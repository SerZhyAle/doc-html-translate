<#
.SYNOPSIS
  Quick commit on the current branch, then push (unless -NoPush). Auto-generates a
  timestamp message when none is given. Port of the Android project's commit-push.ps1.

.DESCRIPTION
  The fast "just save my work" flow, driven by the a.ps1 aliases:
    a c   -> commit-push.ps1           (commit + push)
    a cc  -> commit-push.ps1 -NoPush   (commit only)

  It is a PURE git commit - it does NOT run the test/lint/typo gate or build anything.
  For the gated "сборка" (gate + build both exes + commit + COMMIT_LOG entry) use
  scripts/build-local.ps1 (alias: a bl). Pushing a BRANCH never triggers CI - every
  workflow in this repo fires on v* / ext-cws-v* / ext-edge-v* TAGS only - so this
  stays local + free.

.PARAMETER Message
  Commit message. Also binds positionally, so `a cc "fix typo"` works. When omitted,
  a timestamp message ("wip: yyMMdd-HHmm") is used.

.PARAMETER NoPush
  Commit but do NOT push (the `cc` flow).

.EXAMPLE
  ./scripts/commit-push.ps1                                  # auto-message, commit + push

.EXAMPLE
  ./scripts/commit-push.ps1 -Message "fix: navbar" -NoPush   # commit only
#>
param(
    [string]$Message,
    [switch]$NoPush
)

$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)

# Auto-generate a commit message when none is provided (like the Android original).
if ([string]::IsNullOrWhiteSpace($Message)) {
    $Message = "wip: " + (Get-Date -Format "yyMMdd-HHmm")
}

$branch = (git branch --show-current).Trim()
if (-not $branch) { throw "Detached HEAD - checkout a branch before committing." }

Write-Host "Adding all changes.." -ForegroundColor Cyan
& git add -A

# Nothing to commit -> stop cleanly (matches the reference).
if (-not (git status --porcelain)) {
    Write-Host "No changes to commit." -ForegroundColor DarkGray
    exit 0
}

Write-Host "Committing on '$branch': $Message" -ForegroundColor Cyan
& git commit -m $Message
if ($LASTEXITCODE -ne 0) { throw "git commit failed" }

if ($NoPush) {
    Write-Host "Committed locally on '$branch' - not pushed (-NoPush)." -ForegroundColor Green
    exit 0
}

Write-Host "== push $branch -> origin ==" -ForegroundColor Cyan
& git push --set-upstream origin $branch
if ($LASTEXITCODE -ne 0) { throw "git push failed" }

Write-Host "Committed and pushed '$branch' (branch push, no tag - no CI runs)." -ForegroundColor Green
