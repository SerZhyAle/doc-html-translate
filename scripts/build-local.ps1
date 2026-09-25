<#
.SYNOPSIS
  "Сборка" (local build) - the free, local-only flow. NOTHING here touches GitHub.

.DESCRIPTION
  One entry point for the build concept defined in DEV/RELEASE.md:
    1. build.ps1   - doc-html-translate.exe (CLI, icon + version embedded)
    2. build-ui.ps1 - doc-html-ui.exe (GUI launcher)
    3. check.ps1   - the full quality gate (tests, lint, typos, ..)
    4. commit      - if the gate passed and there are changes, commit on the
                     current branch and append to DEV/COMMIT_LOG.md

  The builds run before the gate on purpose: they rewrite tracked files (the two build/*.exe and
  the generated icons), so building after the gate would leave HEAD on a tree the gate never read
  and scripts/release.ps1 could never go green. The gate's tree is then the commit's tree; the one
  path that changes after it is DEV/COMMIT_LOG.md, which needs the commit's hash and which
  release.ps1 allows by name.

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

# ── Step 1-2: build both binaries ────────────────────────────
# The build scripts run in this process and set GOOS/GOARCH for their cross-build; put them back
# before the gate, whose children inherit this environment.
$goEnv = @{ GOOS = $env:GOOS; GOARCH = $env:GOARCH }
try {
    Write-Host "== [1/4] build CLI (doc-html-translate.exe) ==" -ForegroundColor Cyan
    ./scripts/build.ps1

    Write-Host "== [2/4] build UI (doc-html-ui.exe) ==" -ForegroundColor Cyan
    ./scripts/build-ui.ps1
} finally {
    foreach ($k in $goEnv.Keys) {
        if ($goEnv[$k]) { Set-Item -Path "Env:$k" -Value $goEnv[$k] } else { Remove-Item "Env:$k" -ErrorAction SilentlyContinue }
    }
}

# ── Step 3: quality gate, over the tree the commit will hold ─
Write-Host "== [3/4] check (the full gate) ==" -ForegroundColor Cyan
./scripts/check.ps1
# check.ps1 speaks CHECK-VERDICT: 0 pass, 3 pass with advisories (parity drift, named above),
# 1 fail, 2 could not verify. Only the first two may be committed on: "could not verify" is not a
# pass, however green the rest of the output looks.
$gate = $LASTEXITCODE
if ($gate -notin 0, 3) {
    Write-Host "build-local: stopped - the gate did not pass (exit $gate). Both binaries were rebuilt; nothing was committed." -ForegroundColor Red
    exit $gate
}

if ($NoCommit) {
    Write-Host "== done (NoCommit): both binaries built, gate passed, nothing committed ==" -ForegroundColor Green
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
