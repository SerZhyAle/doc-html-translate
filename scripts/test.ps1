<#
.SYNOPSIS
  Go test gate. Exit codes and the last line follow CHECK-VERDICT (see scripts/lib/verdict.ps1).

.DESCRIPTION
  Runs `go test -json` rather than plain `go test`, because plain `go test` prints `ok` for a
  package whose tests all skipped: a fresh clone without test_doc/ used to print "Tests passed"
  over a full-corpus integration run that never happened.

  Skips are read from the event stream and split in two:
    - a skip whose message starts with "input absent:" means a declared input of the suite is
      missing (the test_doc/ corpus) - the run could not verify what it exists for, exit 2;
    - any other skip (one sample over the size cap, Calibre or 7-Zip not installed, a fixture
      that is only present on some machines) is counted on the verdict line and named above it.

  A passing test may also log a line "advisory: <text>" (tests/site_l10n_test.go does, for a
  translation older than its English source): something found that the change under test cannot be
  charged with. Advisories are named above the verdict and make it PASS WITH ADVISORIES, exit 3.

  Last line: "test: PASS (run=N skipped=M)", "test: PASS WITH ADVISORIES (run=N skipped=M
  advisories=A)", "test: FAIL (n)" or "test: COULD NOT VERIFY (n)".
  Raw events go to temp/logs/test.json, the readable log to temp/logs/test.log.
#>
$ErrorActionPreference = "Stop"
. "$PSScriptRoot/lib/verdict.ps1"

New-Item -ItemType Directory -Force -Path "temp/logs" | Out-Null

# Test the arch we actually ship: amd64, pure Go (matches build.ps1, build-ui.ps1
# and release.yml). A dev machine's Go may default to 386, whose ~2 GB address
# space cumulatively OOMs the full-corpus integration run (TestConvertTestDoc)
# even though each real single-file conversion is fine. amd64 sidesteps that and
# keeps the gate representative. CGO stays off (the app shells out to tesseract).
$env:GOOS = "windows"; $env:GOARCH = "amd64"; $env:CGO_ENABLED = "0"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "go not found on PATH"
    Exit-Verdict 'test' 2 'go toolchain absent'
}
$goVersion = (& go env GOVERSION).Trim()
Write-Subject 'test' "Go edition, windows/amd64, CGO off, $goVersion, $(Get-TreeLabel); the extension edition is scripts/test-extension.ps1"

$logPath = Join-Path (Get-Location) "temp/logs/test.log"
$jsonPath = Join-Path (Get-Location) "temp/logs/test.json"
$log = [System.IO.StreamWriter]::new($logPath, $false)
$raw = [System.IO.StreamWriter]::new($jsonPath, $false)

$buffers = @{}        # "pkg test" -> the test's own output lines
$failedTests = [System.Collections.Generic.List[string]]::new()
$failedPkgs = [System.Collections.Generic.List[string]]::new()
$inputAbsent = [System.Collections.Generic.List[string]]::new()
$envSkips = [System.Collections.Generic.List[string]]::new()
$advisories = [System.Collections.Generic.List[string]]::new()
$passed = 0

function Out-Line([string]$text) {
    Write-Host $text
    $log.WriteLine($text)
}

# The reason a test gave for skipping: its own output minus the framework's RUN/SKIP lines.
function Get-SkipReason([string]$key) {
    $lines = @($buffers[$key] | ForEach-Object { $_.Trim() } |
            Where-Object { $_ -and $_ -notmatch '^(=== (RUN|PAUSE|CONT)|--- SKIP)' })
    if ($lines) { return ($lines -join ' ') -replace '^\S+\.go:\d+:\s*', '' }
    return '(no reason printed)'
}

try {
    & go test -json ./... -count=1 2>&1 | ForEach-Object {
        $line = "$_"
        $raw.WriteLine($line)
        if (-not $line.StartsWith('{')) { Out-Line $line; return }
        $ev = $line | ConvertFrom-Json
        $key = "$($ev.Package) $($ev.Test)"
        switch ($ev.Action) {
            'output' {
                if ($ev.Test) {
                    if (-not $buffers.ContainsKey($key)) { $buffers[$key] = [System.Collections.Generic.List[string]]::new() }
                    $buffers[$key].Add($ev.Output.TrimEnd())
                    if ($ev.Output -match '^\s*\S+\.go:\d+: advisory: (.*)$') { $advisories.Add("$($ev.Package) $($ev.Test): $($Matches[1].Trim())") }
                } else {
                    $text = $ev.Output.TrimEnd()
                    if ($text -and $text -ne 'PASS' -and $text -notmatch '^testing: warning: no tests to run') { Out-Line $text }
                }
            }
            'build-output' { Out-Line $ev.Output.TrimEnd() }
            'pass' { if ($ev.Test) { $passed++ } }
            'skip' {
                if ($ev.Test) {
                    $reason = Get-SkipReason $key
                    $entry = "$($ev.Package) $($ev.Test): $reason"
                    if ($reason -match '^input absent:') { $inputAbsent.Add($entry) } else { $envSkips.Add($entry) }
                }
            }
            'fail' {
                if ($ev.Test) {
                    $failedTests.Add("$($ev.Package) $($ev.Test)")
                    if ($buffers.ContainsKey($key)) { foreach ($l in $buffers[$key]) { Out-Line $l } }
                } else {
                    $failedPkgs.Add($ev.Package)
                }
            }
        }
    }
    $goExit = $LASTEXITCODE
} finally {
    $log.Dispose()
    $raw.Dispose()
}

Write-Host ""
if ($envSkips.Count -gt 0) {
    Write-Host "skipped ($($envSkips.Count)) - an environment the run did not have, not a pass on that item:" -ForegroundColor Yellow
    foreach ($s in $envSkips) { Write-Host "  - $s" -ForegroundColor Yellow }
}

if ($goExit -ne 0 -or $failedTests.Count -gt 0 -or $failedPkgs.Count -gt 0) {
    # A package that failed with no failing test failed to build or panicked outside a test.
    $pkgOnly = @($failedPkgs | Where-Object { $p = $_; -not ($failedTests | Where-Object { $_.StartsWith("$p ") }) })
    Write-Host "failed:" -ForegroundColor Red
    foreach ($f in $failedTests) { Write-Host "  - $f" -ForegroundColor Red }
    foreach ($p in $pkgOnly) { Write-Host "  - $p (package)" -ForegroundColor Red }
    Write-Host "See temp/logs/test.log"
    Exit-Verdict 'test' 1 "$([Math]::Max(1, $failedTests.Count + $pkgOnly.Count))"
}

# Named before the input-absent verdict too: exit 2 outranks 3, and a stale locale found on a
# machine without test_doc/ must not vanish just because the corpus run could not happen.
if ($advisories.Count -gt 0) {
    Write-Host "advisories ($($advisories.Count)) - found, not charged to this change:" -ForegroundColor Yellow
    foreach ($a in $advisories) { Write-Host "  - $a" -ForegroundColor Yellow }
}

if ($inputAbsent.Count -gt 0) {
    Write-Host "declared input absent - these tests inspected nothing:" -ForegroundColor Magenta
    foreach ($s in $inputAbsent) { Write-Host "  - $s" -ForegroundColor Magenta }
    Exit-Verdict 'test' 2 "$($inputAbsent.Count)"
}

if ($advisories.Count -gt 0) {
    Exit-Verdict 'test' 3 "run=$passed skipped=$($envSkips.Count) advisories=$($advisories.Count)"
}

Exit-Verdict 'test' 0 "run=$passed skipped=$($envSkips.Count)"
