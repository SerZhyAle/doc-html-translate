$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Force -Path "temp/logs" | Out-Null

./scripts/test.ps1 *>&1 | Tee-Object -FilePath "temp/logs/check-test.log"
./scripts/lint.ps1 *>&1 | Tee-Object -FilePath "temp/logs/check-lint.log"
./scripts/typo.ps1 *>&1 | Tee-Object -FilePath "temp/logs/check-typo.log"

# Advisory: warn on cross-edition drift (Go extractor changed without its paired JS,
# or vice versa). Never blocks the gate - it only prints. tests/parity_test.go (run above)
# still hard-fails on VALUE drift; this catches the STRUCTURAL "one side moved" signal.
./scripts/parity-check.ps1 *>&1 | Tee-Object -FilePath "temp/logs/check-parity.log"

Write-Host "All checks passed"
