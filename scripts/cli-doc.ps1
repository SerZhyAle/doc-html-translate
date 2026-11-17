<#
.SYNOPSIS
  The command-line reference check (APP-COMMANDS rules 18 and 20; ticket E0143): the manifest docs/cli-commands.json is
  valid, the executable's embedded copy is byte-equal to it, and docs/CLI.md is byte-equal to its render. Ends in one
  CHECK-VERDICT line.

.DESCRIPTION
  The judging is internal/commands, run through tools/clidoc. What it holds, in the contract's words:
    - docs/cli-commands.json has the shape of rule 18: schemaVersion, product, exe, cliVersion, and for every command its
      name (one or two lower-case words), aliases, summary, synopsis, args, returns, risk, coldStart, wait, timeoutMs,
      interactive, idempotent, exitCodes, examples and since, each from its closed list, and the base commands of rule 17
      that need no running window (commands, version);
    - internal/commands/cli-commands.json, the copy both executables embed, is the same bytes - `commands --json` prints
      the shipped copy and is never rewritten from memory;
    - docs/CLI.md is the render of the manifest, never edited by hand (rule 20): after changing the manifest run
      `go run ./tools/clidoc -render`, which rewrites the embedded copy and the page.

  Exit codes follow CHECK-VERDICT (scripts/lib/verdict.ps1): 0 PASS, 1 FAIL, 2 COULD NOT VERIFY (no Go toolchain, no git
  checkout, a manifest that cannot be read).
#>
$ErrorActionPreference = "Stop"
. "$PSScriptRoot/lib/verdict.ps1"
. "$PSScriptRoot/lib/gobuildtags.ps1"

$root = (& git rev-parse --show-toplevel 2>$null)
if ($LASTEXITCODE -ne 0 -or -not $root) {
    Write-Host "cli-doc: not inside a git checkout - the repository root cannot be found."
    Exit-Verdict 'cli-doc' 2 'no git checkout'
}
Set-Location $root.Trim()
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "cli-doc: go not found on PATH"
    Exit-Verdict 'cli-doc' 2 'go toolchain absent'
}

# The tool prints its own banner and findings and ends in its verdict line; its exit code is the verdict's.
$verdictLine = '^cli-doc: (PASS|FAIL|COULD NOT VERIFY)'
$lines = @(& go run ./tools/clidoc *>&1 | ForEach-Object { if ("$_" -notmatch $verdictLine) { Write-Host $_ }; "$_" })
$code = $LASTEXITCODE
if ($code -notin 0, 1, 2) {
    Write-Host "cli-doc: the check ended with exit $code and no verdict"
    Exit-Verdict 'cli-doc' 2 "the check did not run (exit $code)"
}
$last = @($lines | Where-Object { $_ -match $verdictLine }) | Select-Object -Last 1
# go run exits 1 for any non-zero exit of the program it ran (it prints "exit status 3"), so a COULD NOT VERIFY would
# arrive as FAIL: the tool's own verdict word carries the code (CHECK-VERDICT rule 2.5).
$byWord = @{ 'PASS WITH ADVISORIES' = 3; 'COULD NOT VERIFY' = 2; 'FAIL' = 1; 'PASS' = 0 }
if ($last -and $last -match '^cli-doc: (PASS WITH ADVISORIES|COULD NOT VERIFY|FAIL|PASS)\b') { $code = $byWord[$Matches[1]] }
$detail = if ($last -and $last -match '\((.*)\)\s*$') { $Matches[1] } else { '' }
Exit-Verdict 'cli-doc' $code $detail
