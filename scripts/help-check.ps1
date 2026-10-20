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
      may fall and never rise (CHECK-BASELINE);
    - help/ at the repository root is the same bytes as the embedded set;
    - freshness (an advisory, exit 3): the set records the digests of its inputs, and an input that changed after the
      last `node tools/helpgen/generate.mjs` is named.

  Exit codes follow CHECK-VERDICT (scripts/lib/verdict.ps1): 0 PASS, 1 FAIL, 2 COULD NOT VERIFY (no Go toolchain, no
  git checkout, no help set), 3 PASS WITH ADVISORIES.
#>
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
$lines = @(& go run ./tools/helpcheck *>&1 | ForEach-Object { if ("$_" -notmatch $verdictLine) { Write-Host $_ }; "$_" })
$code = $LASTEXITCODE
if ($code -notin 0, 1, 2, 3) {
    Write-Host "help-check: the check ended with exit $code and no verdict"
    Exit-Verdict 'help-check' 2 "the check did not run (exit $code)"
}
$last = @($lines | Where-Object { $_ -match $verdictLine }) | Select-Object -Last 1
$detail = if ($last -and $last -match '\((.*)\)\s*$') { $Matches[1] } else { '' }
Exit-Verdict 'help-check' $code $detail
