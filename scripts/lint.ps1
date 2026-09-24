# golangci-lint gate. Exit codes and the last line follow CHECK-VERDICT (scripts/lib/verdict.ps1):
# a missing linter is "could not verify" (2), never "failed" - nothing was inspected.
$ErrorActionPreference = "Stop"
. "$PSScriptRoot/lib/verdict.ps1"

New-Item -ItemType Directory -Force -Path "temp/logs" | Out-Null

if (-not (Get-Command golangci-lint -ErrorAction SilentlyContinue)) {
    Write-Host "golangci-lint not found on PATH. Run ./scripts/bootstrap-tools.ps1"
    Exit-Verdict 'lint' 2 'golangci-lint absent'
}
Write-Subject 'lint' "Go source, configs/.golangci.yml, $(Get-TreeLabel)"

golangci-lint run --config configs/.golangci.yml ./... *>&1 | Tee-Object -FilePath "temp/logs/lint.log"
if ($LASTEXITCODE -ne 0) {
    Write-Host "See temp/logs/lint.log"
    Exit-Verdict 'lint' 1
}

Exit-Verdict 'lint' 0
