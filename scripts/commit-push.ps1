<#
.SYNOPSIS
  Quick commit on the current branch, then push (unless -NoPush). Auto-generates a
  timestamp message when none is given. Port of the Android project's commit-push.ps1.

.DESCRIPTION
  The fast "just save my work" flow, driven by the a.ps1 aliases:
    a c   -> commit-push.ps1           (commit + push)
    a cc  -> commit-push.ps1 -NoPush   (commit only)

  It is a PURE git commit - it does NOT run the test/lint/typo gate or build anything.
  For the gated "сборка" (build both exes + gate + commit + COMMIT_LOG entry) use
  scripts/build-local.ps1 (alias: a bl). No workflow fires on a branch push - every one
  fires on v* / ext-cws-v* / ext-edge-v* TAGS only - so a feature-branch push stays free.

  A push of main is NOT local: GitHub Pages serves the product site from main's root, so the
  push publishes every site page in the commit, ungated. The script refuses it unless
  -PublishSite says that is the intent; commit with -NoPush (a cc) otherwise.

.PARAMETER Message
  Commit message. Also binds positionally, so `a cc "fix typo"` works. When omitted,
  a timestamp message ("wip: yyMMdd-HHmm") is used.

.PARAMETER NoPush
  Commit but do NOT push (the `cc` flow).

.PARAMETER PublishSite
  Allow the push when the current branch is main, knowing it publishes the Pages site.

.EXAMPLE
  ./scripts/commit-push.ps1                                  # auto-message, commit + push (not on main)

.EXAMPLE
  ./scripts/commit-push.ps1 -Message "fix: navbar" -NoPush   # commit only

.EXAMPLE
  ./scripts/commit-push.ps1 -Message "site: fix typo" -PublishSite   # push main, publishing the site
#>
param(
    [string]$Message,
    [switch]$NoPush,
    [switch]$PublishSite
)

$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)

# The branch GitHub Pages builds the site from (a static site from main's root).
$siteBranch = 'main'

# Auto-generate a commit message when none is provided (like the Android original).
if ([string]::IsNullOrWhiteSpace($Message)) {
    $Message = "wip: " + (Get-Date -Format "yyMMdd-HHmm")
}

$branch = (git branch --show-current).Trim()
if (-not $branch) { throw "Detached HEAD - checkout a branch before committing." }

# Decided before anything is staged, so a refusal leaves the working tree as it was.
if (-not $NoPush -and $branch -eq $siteBranch -and -not $PublishSite) {
    Write-Host "Refusing to push '$branch': GitHub Pages serves the site from it, so the push publishes the site, with no gate." -ForegroundColor Red
    Write-Host "Commit only with -NoPush (a cc), or pass -PublishSite to push and publish." -ForegroundColor Red
    exit 1
}

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

if ($branch -eq $siteBranch) {
    Write-Host "Committed and pushed '$branch' - the Pages site rebuilds from it (no tag, no CI)." -ForegroundColor Yellow
} else {
    Write-Host "Committed and pushed '$branch' (branch push, no tag - no CI runs)." -ForegroundColor Green
}
