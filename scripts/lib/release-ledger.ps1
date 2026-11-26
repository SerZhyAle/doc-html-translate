# Shared helper for calling the sza plugin's release-ledger tool (RELEASE-LEDGER 0.1).
# Dot-source from build scripts or scripts/release-ledger.ps1:
#   . (Join-Path $PSScriptRoot "lib/release-ledger.ps1")
#
# Locates tools/harness/releases/release-ledger.ps1 from $env:CLAUDE_PLUGIN_ROOT or
# ~/.claude/plugins/installed_plugins.json (sza@sza-unified-rules). Never fails a build
# when the plugin is absent or the ledger write returns non-zero: reports a warning and continues.

function Find-ReleaseLedgerScript {
    $roots = @()
    if ($env:CLAUDE_PLUGIN_ROOT) { $roots += $env:CLAUDE_PLUGIN_ROOT }
    $record = Join-Path $HOME '.claude/plugins/installed_plugins.json'
    if (Test-Path -LiteralPath $record -PathType Leaf) {
        try {
            $plugins = (Get-Content -LiteralPath $record -Raw -Encoding utf8 | ConvertFrom-Json).plugins.'sza@sza-unified-rules'
            foreach ($p in @($plugins | Sort-Object lastUpdated -Descending)) {
                if ($p.installPath) { $roots += [string]$p.installPath }
            }
        } catch {
            Write-Verbose "installed_plugins.json unreadable: $($_.Exception.Message)"
        }
    }
    foreach ($root in $roots) {
        if (-not $root) { continue }
        $candidate = Join-Path $root 'tools/harness/releases/release-ledger.ps1'
        if (Test-Path -LiteralPath $candidate -PathType Leaf) { return $candidate }
    }
    return $null
}

function Invoke-ReleaseLedgerBuild {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory)][string]$Package,
        [Parameter(Mandatory)][ValidateSet('release', 'test', 'debug')][string]$Kind,
        [Parameter(Mandatory)][string]$Version,
        [string]$Flavor,
        [string]$Artifact,
        [string]$Note,
        [string]$RepoRoot
    )
    $scriptPath = Find-ReleaseLedgerScript
    if (-not $scriptPath) {
        Write-Warning 'release ledger not updated: sza plugin release-ledger.ps1 not found'
        return
    }
    if (-not $RepoRoot) {
        $RepoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
    }
    $pwsh = (Get-Process -Id $PID).Path
    if (-not $pwsh -or -not (Test-Path -LiteralPath $pwsh)) { $pwsh = 'pwsh' }
    $argv = @('-NoProfile', '-File', $scriptPath, 'build', '-RepoRoot', $RepoRoot, '-Package', $Package, '-Kind', $Kind, '-Version', $Version)
    if ($Flavor)   { $argv += @('-Flavor', $Flavor) }
    if ($Artifact) { $argv += @('-Artifact', $Artifact) }
    if ($Note)     { $argv += @('-Note', $Note) }
    try {
        & $pwsh @argv
        if ($LASTEXITCODE -ne 0) {
            Write-Warning "release ledger not updated (exit $LASTEXITCODE)"
            $global:LASTEXITCODE = 0
        }
    } catch {
        Write-Warning "release ledger not updated: $($_.Exception.Message)"
        $global:LASTEXITCODE = 0
    }
}
