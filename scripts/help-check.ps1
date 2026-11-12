<#
.SYNOPSIS
  The in-app help check (CONTEXT-HELP rule 19, ticket 118): the GUI launcher's surfaces against the help map, the
  pages against the locale set, the links, rule 12 (a page makes no request), the house text style, the two copies
  of the help set and the set against the inputs it was generated from. Ends in one CHECK-VERDICT line.

.DESCRIPTION
  The judging is internal/helpcheck, run through tools/helpcheck so the gate and `go test ./tests/` hold the same rules.
  What it holds, in the contract's words:
    - every surface of the window (the data-help regions and the openDialog calls of cmd/doc-html-ui/ui.html, the
      routes of its server) has a row in cmd/doc-html-ui/help/help-map.json, and every row has a surface;
    - a `none` or `inherit` row says why;
    - every page row exists in every shipped locale, with the parts of rule 8 in order, an alt text, lang and dir, no
      request to another origin, no inline script, and every link resolved;
    - the locales the GUI ships and the help does not yet have are the set baseline configs/help-baseline.txt, which
      may fall and never rise (CHECK-BASELINE); it is compared with `git show HEAD:configs/help-baseline.txt`, a line
      HEAD did not hold fails unless the update path recorded a reason for it
      (`go run ./tools/helpcheck -accept-baseline "<reason>"`);
    - help/ at the repository root is the same bytes as the embedded set;
    - freshness: the set records the digests of its inputs, and an input that changed after the last generation is
      named. Between releases that is an advisory (exit 3) - the screenshot half of the fix builds the GUI, which the
      5 s line does not allow, so CONTEXT-HELP rule 18 carries a dated exception for it (ticket E0125); the wording
      half is `node tools/helpgen/repage.mjs`. With -Release it is a failure (exit 1).

  Exit codes follow CHECK-VERDICT (scripts/lib/verdict.ps1): 0 PASS, 1 FAIL, 2 COULD NOT VERIFY (no Go toolchain, no
  git checkout, no help set), 3 PASS WITH ADVISORIES.

.PARAMETER Release
  The full gate and the release pre-flight: a help set older than its inputs fails instead of advising. scripts/check.ps1
  passes it to this child when it runs without -Fast; the fast tier keeps the advisory.
#>
param(
    [switch]$Release
)

$ErrorActionPreference = "Stop"
. "$PSScriptRoot/lib/verdict.ps1"

$root = (& git rev-parse --show-toplevel 2>$null)
if ($LASTEXITCODE -ne 0 -or -not $root) {
    Write-Host "help-check: not inside a git checkout - the repository root cannot be found."
    Exit-Verdict 'help-check' 2 'no git checkout'
}
Set-Location $root.Trim()
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "help-check: go not found on PATH"
    Exit-Verdict 'help-check' 2 'go toolchain absent'
}

# The tool prints its own banner and findings and ends in its verdict line; its exit code is the verdict's.
$verdictLine = '^help-check: (PASS|FAIL|COULD NOT VERIFY)'
$toolArgs = @()
if ($Release) { $toolArgs += '-strict-freshness' }
$lines = @(& go run ./tools/helpcheck @toolArgs *>&1 | ForEach-Object { if ("$_" -notmatch $verdictLine) { Write-Host $_ }; "$_" })
$code = $LASTEXITCODE
if ($code -notin 0, 1, 2, 3) {
    Write-Host "help-check: the check ended with exit $code and no verdict"
    Exit-Verdict 'help-check' 2 "the check did not run (exit $code)"
}
$last = @($lines | Where-Object { $_ -match $verdictLine }) | Select-Object -Last 1
# go run exits 1 for any non-zero exit of the program it ran (it prints "exit status 3"), so an advisory
# or a COULD NOT VERIFY would arrive as FAIL: the tool's own verdict word carries the code (CHECK-VERDICT rule 2.5).
$byWord = @{ 'PASS WITH ADVISORIES' = 3; 'COULD NOT VERIFY' = 2; 'FAIL' = 1; 'PASS' = 0 }
if ($last -and $last -match '^help-check: (PASS WITH ADVISORIES|COULD NOT VERIFY|FAIL|PASS)\b') { $code = $byWord[$Matches[1]] }
$detail = if ($last -and $last -match '\((.*)\)\s*$') { $Matches[1] } else { '' }
Exit-Verdict 'help-check' $code $detail
