$ErrorActionPreference = "Stop"

New-Item -ItemType Directory -Force -Path "temp/logs" | Out-Null

# Test the arch we actually ship: amd64, pure Go (matches build.ps1, build-ui.ps1
# and release.yml). A dev machine's Go may default to 386, whose ~2 GB address
# space cumulatively OOMs the full-corpus integration run (TestConvertTestDoc)
# even though each real single-file conversion is fine. amd64 sidesteps that and
# keeps the gate representative. CGO stays off (the app shells out to tesseract).
$env:GOOS = "windows"; $env:GOARCH = "amd64"; $env:CGO_ENABLED = "0"

go test ./... -count=1 *>&1 | Tee-Object -FilePath "temp/logs/test.log"
if ($LASTEXITCODE -ne 0) {
    throw "tests failed. See temp/logs/test.log"
}

Write-Host "Tests passed"
