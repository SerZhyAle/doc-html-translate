$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Force -Path "temp/logs" | Out-Null

if (-not (Get-Command typos -ErrorAction SilentlyContinue)) {
    throw "typos not found. Run ./scripts/bootstrap-tools.ps1"
}

typos --config configs/.typos.toml . *>&1 | Tee-Object -FilePath "temp/logs/typo.log"
if ($LASTEXITCODE -ne 0) {
    throw "typo check failed. See temp/logs/typo.log"
}

Write-Host "Typo check passed"
