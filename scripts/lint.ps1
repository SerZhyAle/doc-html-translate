$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Force -Path "temp/logs" | Out-Null

if (-not (Get-Command golangci-lint -ErrorAction SilentlyContinue)) {
    throw "golangci-lint not found. Run ./scripts/bootstrap-tools.ps1"
}

golangci-lint run --config configs/.golangci.yml ./... *>&1 | Tee-Object -FilePath "temp/logs/lint.log"
if ($LASTEXITCODE -ne 0) {
    throw "lint failed. See temp/logs/lint.log"
}

Write-Host "Lint passed"
