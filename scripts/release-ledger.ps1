<#
.SYNOPSIS
  Forwarder to the sza plugin's release-ledger tool (DEV/releases/).

.DESCRIPTION
  Runs tools/harness/releases/release-ledger.ps1 against this repository. With no
  arguments, defaults to `show`. Exposed via `./a.ps1 rs`.
#>
$ErrorActionPreference = 'Stop'
[string[]]$pass = @($args)
if ($pass.Count -eq 0) { [string[]]$pass = @('show') }

. (Join-Path $PSScriptRoot 'lib/release-ledger.ps1')

$scriptPath = Find-ReleaseLedgerScript
if (-not $scriptPath) {
    Write-Host 'release-ledger: sza plugin tools/harness/releases/release-ledger.ps1 not found (install sza plugin or set CLAUDE_PLUGIN_ROOT).' -ForegroundColor Red
    exit 1
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$pwsh = (Get-Process -Id $PID).Path
if (-not $pwsh -or -not (Test-Path -LiteralPath $pwsh)) { $pwsh = 'pwsh' }

Push-Location $repoRoot
try {
    & $pwsh -NoProfile -File $scriptPath @pass
    exit $LASTEXITCODE
} finally {
    Pop-Location
}
