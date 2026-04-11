$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Force -Path "temp/logs" | Out-Null

./scripts/test.ps1 *>&1 | Tee-Object -FilePath "temp/logs/check-test.log"
./scripts/lint.ps1 *>&1 | Tee-Object -FilePath "temp/logs/check-lint.log"
./scripts/typo.ps1 *>&1 | Tee-Object -FilePath "temp/logs/check-typo.log"

Write-Host "All checks passed"
