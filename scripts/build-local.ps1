<#
.SYNOPSIS
  "Сборка" (local build) - the free, local-only flow. NOTHING here touches GitHub.

.DESCRIPTION
  One entry point for the build concept defined in DEV/RELEASE.md:
    1. check.ps1   - go test + golangci-lint + typos (the quality gate)
    2. build.ps1   - doc-html-translate.exe (CLI, icon + version embedded)
    3. build-ui.ps1 - doc-html-ui.exe (GUI launcher)
    4. commit      - if the gate passed and there are changes, commit on the
                     current branch and append to DEV/COMMIT_LOG.md

  No tags, no push, no CI, no store publish - so a "test build" never costs a
  paid GitHub Actions run. The paid/published path is "релиз": scripts/release.ps1.

.PARAMETER Message
  Commit message. Required unless -NoCommit is passed.

.PARAMETER NoCommit
  Run the gate + both builds but stop before committing (build-only smoke test).

.EXAMPLE
  ./scripts/build-local.ps1 -Message "fix: navbar position on index"

.EXAMPLE
  ./scripts/build-local.ps1 -NoCommit
#>
param(
    [string]$Message,
    [switch]$NoCommit
)

$ErrorActionPreference = "Stop"

if (-not $NoCommit -and [string]::IsNullOrWhiteSpace($Message)) {
    throw "Provide -Message ""...""  (or pass -NoCommit for a build-only smoke test)."
}

# ── Step 1: quality gate (test + lint + typo) ────────────────
Write-Host "== [1/4] check (test + lint + typo) ==" -ForegroundColor Cyan
./scripts/check.ps1

# ── Step 2-3: build both binaries ────────────────────────────
Write-Host "== [2/4] build CLI (doc-html-translate.exe) ==" -ForegroundColor Cyan
./scripts/build.ps1

Write-Host "== [3/4] build UI (doc-html-ui.exe) ==" -ForegroundColor Cyan
./scripts/build-ui.ps1

if ($NoCommit) {
    Write-Host "== done (NoCommit): gate passed, both binaries built, nothing committed ==" -ForegroundColor Green
    exit 0
}

# ── Step 4: commit on the current branch ─────────────────────
Write-Host "== [4/4] commit ==" -ForegroundColor Cyan

$branch = (git branch --show-current).Trim()
if (-not $branch) {
    throw "Detached HEAD (no current branch). Checkout a branch before committing."
}

& git add -A

$staged = git diff --cached --name-only
if (-not $staged) {
    Write-Host "Gate passed and both binaries built, but no changes to commit." -ForegroundColor Yellow
    exit 0
}

& git commit -m $Message
if ($LASTEXITCODE -ne 0) { throw "git commit failed" }

$hash = (git rev-parse --short HEAD).Trim()
$timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
$files = git diff-tree --no-commit-id --name-only -r HEAD

$logFile = "DEV/COMMIT_LOG.md"
if (-not (Test-Path $logFile)) {
    "# COMMIT LOG`n`n| Timestamp | Branch | Commit | Message |`n|---|---|---|---|" | Out-File -FilePath $logFile -Encoding utf8
}

"| $timestamp | $branch | $hash | $Message |" | Add-Content -Path $logFile -Encoding utf8
"" | Add-Content -Path $logFile -Encoding utf8
"Changed files:" | Add-Content -Path $logFile -Encoding utf8
foreach ($file in $files) {
    "- $file" | Add-Content -Path $logFile -Encoding utf8
}
"" | Add-Content -Path $logFile -Encoding utf8

& git add $logFile
& git commit --amend --no-edit
if ($LASTEXITCODE -ne 0) { throw "git commit amend failed" }

Write-Host "Committed $hash on '$branch' and updated DEV/COMMIT_LOG.md" -ForegroundColor Green
Write-Host "This was a local build - nothing was pushed. For a published release run scripts/release.ps1." -ForegroundColor DarkGray
