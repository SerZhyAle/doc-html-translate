param(
    [Parameter(Mandatory = $true)][string]$Path,
    [Parameter(Mandatory = $true)][string]$Target,
    [Parameter(Mandatory = $true)][string]$Description
)

$ErrorActionPreference = "Stop"

$logFile = "dev/CHANGELOG.md"
if (-not (Test-Path $logFile)) {
    "# DEV CHANGELOG`n`n| Timestamp | Path | Target | Description |`n|---|---|---|---|" | Out-File -FilePath $logFile -Encoding utf8
}

$ts = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
"| $ts | $Path | $Target | $Description |" | Add-Content -Path $logFile -Encoding utf8
