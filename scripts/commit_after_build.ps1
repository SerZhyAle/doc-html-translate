param(
    [Parameter(Mandatory = $true)][string]$Message
)

$ErrorActionPreference = "Stop"

$branch = (git branch --show-current).Trim()
if ($branch -ne "master") {
    throw "Current branch is '$branch'. Required branch: 'master'."
}

./scripts/build.ps1

# Stage all changes
& git add -A

# Nothing to commit check
$staged = git diff --cached --name-only
if (-not $staged) {
    Write-Host "No staged changes after successful build. Nothing to commit."
    exit 0
}

& git commit -m $Message
if ($LASTEXITCODE -ne 0) {
    throw "git commit failed"
}

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
if ($LASTEXITCODE -ne 0) {
    throw "git commit amend failed"
}

Write-Host "Committed $hash on $branch and updated DEV/COMMIT_LOG.md"
