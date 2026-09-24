# Extension-edition unit tests (node --test over extension/test/*.test.mjs). Exit codes and the last
# line follow CHECK-VERDICT (scripts/lib/verdict.ps1): no Node or no installed dev dependencies is
# "could not verify" (2) - the JS edition was not inspected, which is not the same as it passing.
$ErrorActionPreference = "Stop"
. "$PSScriptRoot/lib/verdict.ps1"

New-Item -ItemType Directory -Force -Path "temp/logs" | Out-Null
$log = Join-Path (Get-Location) "temp/logs/test-extension.log"

if (-not (Get-Command node -ErrorAction SilentlyContinue)) {
    Write-Host "node not found on PATH - the extension edition cannot be tested here."
    Exit-Verdict 'test-extension' 2 'node absent'
}
if (-not (Test-Path -LiteralPath "extension/node_modules")) {
    Write-Host "extension/node_modules is missing (the DOM tests need linkedom). Run: cd extension; npm ci"
    Exit-Verdict 'test-extension' 2 'dev dependencies not installed'
}
$nodeVersion = (& node --version).Trim()
Write-Subject 'test-extension' "extension edition (JS), node $nodeVersion, extension/test/*.test.mjs, $(Get-TreeLabel)"

Push-Location extension
try {
    # The spec reporter is forced: piped output would otherwise switch node to TAP, and the summary
    # lines read below are the spec reporter's.
    $lines = @(& node --test --test-reporter=spec "test/*.test.mjs" *>&1 | Tee-Object -FilePath $log | ForEach-Object { "$_" })
    $code = $LASTEXITCODE
} finally {
    Pop-Location
}

# Print the failures and the summary, not 150 green ticks.
$lines | Where-Object { $_ -notmatch '^\s*[\u2714]' -and ($_ -match '^\s*[\u2716\u2139]' -or $_ -match 'Error' -or $_ -match 'tests\s+\d+' -or $_ -match 'pass\s+\d+') } | ForEach-Object { Write-Host $_ }

function Get-Count([string]$label) {
    $m = $lines | Select-String -Pattern "(?:^|\s)$label\s+(\d+)" | Select-Object -Last 1
    if ($m) { return [int]$m.Matches[0].Groups[1].Value }
    return 0
}
$tests = Get-Count 'tests'
$fail = Get-Count 'fail'
$skipped = Get-Count 'skipped'

if ($code -ne 0 -or $fail -gt 0) {
    Write-Host "See temp/logs/test-extension.log"
    Exit-Verdict 'test-extension' 1 "$(if ($fail -gt 0) { $fail } else { 1 })"
}
if ($tests -eq 0) {
    Write-Host "node --test ran no test (no summary line with a count) - see temp/logs/test-extension.log"
    Exit-Verdict 'test-extension' 2 'no test ran'
}
Exit-Verdict 'test-extension' 0 "run=$tests skipped=$skipped"
