<#
.SYNOPSIS
  The settings inventory check (APP-SETTINGS rule 11, SETTINGS-NAVIGATION section 6; ticket 122, item 4): what the GUI
  launcher persists, traced to its control, page, caption, default and help region, held equal to the committed render
  docs/settings-inventory.json and aligned with the manual. Ends in one CHECK-VERDICT line.

.DESCRIPTION
  The judging is internal/settingsinv, run through tools/settingsinv. What it holds, in the contract's words:
    - every key collectSettings() returns in cmd/doc-html-ui/ui.html is a setting of the window (a control on a page,
      with a caption in the dictionary and a help region the help map lists) or a recorded piece of session context;
    - docs/settings-inventory.json is exactly what the window says now - a setting added, renamed, moved or given
      another default without regenerating it fails (`go run ./tools/settingsinv -render`, never a hand edit);
    - the manual says it: the caption of every setting is named in the help copy deck of the surface that owns its
      help region (tools/helpgen/copy), so the generated help pages carry the same caption;
    - a setting that maps to a command-line flag is a row of all three README flag tables, is defined in
      internal/config/flags.go and is really passed by the window's command builder (cmd/doc-html-ui/main.go).

  Exit codes follow CHECK-VERDICT (scripts/lib/verdict.ps1): 0 PASS, 1 FAIL, 2 COULD NOT VERIFY (no Go toolchain, no
  git checkout, an input that cannot be read).
#>
$ErrorActionPreference = "Stop"
. "$PSScriptRoot/lib/verdict.ps1"
. "$PSScriptRoot/lib/gobuildtags.ps1"

$root = (& git rev-parse --show-toplevel 2>$null)
if ($LASTEXITCODE -ne 0 -or -not $root) {
    Write-Host "settings-inventory: not inside a git checkout - the repository root cannot be found."
    Exit-Verdict 'settings-inventory' 2 'no git checkout'
}
Set-Location $root.Trim()
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "settings-inventory: go not found on PATH"
    Exit-Verdict 'settings-inventory' 2 'go toolchain absent'
}

# The tool prints its own banner and findings and ends in its verdict line; its exit code is the verdict's.
$verdictLine = '^settings-inventory: (PASS|FAIL|COULD NOT VERIFY)'
$lines = @(& go run ./tools/settingsinv *>&1 | ForEach-Object { if ("$_" -notmatch $verdictLine) { Write-Host $_ }; "$_" })
$code = $LASTEXITCODE
if ($code -notin 0, 1, 2) {
    Write-Host "settings-inventory: the check ended with exit $code and no verdict"
    Exit-Verdict 'settings-inventory' 2 "the check did not run (exit $code)"
}
$last = @($lines | Where-Object { $_ -match $verdictLine }) | Select-Object -Last 1
# go run exits 1 for any non-zero exit of the program it ran (it prints "exit status 3"), so an advisory
# or a COULD NOT VERIFY would arrive as FAIL: the tool's own verdict word carries the code (CHECK-VERDICT rule 2.5).
$byWord = @{ 'PASS WITH ADVISORIES' = 3; 'COULD NOT VERIFY' = 2; 'FAIL' = 1; 'PASS' = 0 }
if ($last -and $last -match '^settings-inventory: (PASS WITH ADVISORIES|COULD NOT VERIFY|FAIL|PASS)\b') { $code = $byWord[$Matches[1]] }
$detail = if ($last -and $last -match '\((.*)\)\s*$') { $Matches[1] } else { '' }
Exit-Verdict 'settings-inventory' $code $detail
