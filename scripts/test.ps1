$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Force -Path "temp/logs" | Out-Null

go test ./... -count=1 *>&1 | Tee-Object -FilePath "temp/logs/test.log"
if ($LASTEXITCODE -ne 0) {
    throw "tests failed. See temp/logs/test.log"
}

Write-Host "Tests passed"
