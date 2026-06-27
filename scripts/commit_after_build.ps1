<#
.SYNOPSIS
  Deprecated shim - use scripts/build-local.ps1 (the "сборка" flow).

.DESCRIPTION
  Kept for backward compatibility. The canonical local build+commit flow now
  lives in scripts/build-local.ps1, which also runs the test/lint/typo gate and
  builds the UI binary, and commits on the CURRENT branch (the old version here
  hard-required 'master' and only built the CLI). See DEV/RELEASE.md.
#>
param(
    [Parameter(Mandatory = $true)][string]$Message
)

$ErrorActionPreference = "Stop"

Write-Host "scripts/commit_after_build.ps1 is deprecated - delegating to scripts/build-local.ps1" -ForegroundColor Yellow
./scripts/build-local.ps1 -Message $Message
