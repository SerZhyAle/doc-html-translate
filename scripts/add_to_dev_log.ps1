<#
.SYNOPSIS
  Add one row to DEV/CHANGELOG.md, newest first. Timestamp and the changed-file
  list are filled in automatically; you supply only the human parts
  (Target + Description).

.DESCRIPTION
  The changelog format is a fixed 4-column table:

      | Timestamp | Path | Target | Description |

  Three of those four columns are mechanical:
    - Timestamp : now (yyyy-MM-dd HH:mm:ss)
    - Path      : the changed files (staged first, else unstaged working-tree)
    - Target    : the commit subject you are about to use
  Only Description is prose - a factual, evidence-rich summary of the change in
  project typography (short hyphens, Russian ё, ".." not "..."). See CLAUDE.md.

  This is the script behind the `/changelog` skill and the `a log` alias. It used
  to require -Path by hand and wrote to the wrong (lowercase) path, so it went
  unused; it now derives Path from git and writes to the real DEV/CHANGELOG.md,
  splicing the row under the table rule because the table is newest-first.

.PARAMETER Target
  The commit subject (e.g. "PDF: stop showing a page thumbnail as the page").

.PARAMETER Description
  The prose cell. May be a long, multi-sentence string with inline Markdown.
  Newlines are collapsed to spaces (a Markdown table row is one physical line);
  literal '|' are escaped so they don't split the table.

.PARAMETER Path
  Optional override of the changed-files column. Default: staged files, or the
  unstaged working-tree diff if nothing is staged. Pass a comma-separated list to
  override (e.g. when logging a change that is already committed).

.EXAMPLE
  ./scripts/add_to_dev_log.ps1 -Target "fix: navbar position" -Description "The navbar .."

.EXAMPLE
  # multi-line prose via a here-string:
  ./scripts/add_to_dev_log.ps1 -Target "perf: lazy page extraction" -Description @'
  Pages are now extracted on demand .. Chrome-checked: total=1 render=1 broken=0.
  '@
#>
param(
    [Parameter(Mandatory = $true)][string]$Target,
    [Parameter(Mandatory = $true)][string]$Description,
    [string]$Path
)

$ErrorActionPreference = "Stop"

# Run from the repo root so DEV/CHANGELOG.md resolves regardless of caller CWD.
$repoRoot = (& git rev-parse --show-toplevel 2>$null)
if ($LASTEXITCODE -eq 0 -and $repoRoot) { Set-Location ($repoRoot.Trim()) }

$logFile = "DEV/CHANGELOG.md"
if (-not (Test-Path $logFile)) {
    "# DEV CHANGELOG`n`n| Timestamp | Path | Target | Description |`n|---|---|---|---|" |
        Out-File -FilePath $logFile -Encoding utf8
}

# ── Path column: explicit override, else staged, else unstaged ───────
if ([string]::IsNullOrWhiteSpace($Path)) {
    $files = git diff --cached --name-only
    if (-not $files) { $files = git diff --name-only }
    $Path = (@($files) | Where-Object { $_ } ) -join ", "
    if (-not $Path) { $Path = "(no changed files detected)" }
}

# ── one physical line, pipes escaped so the table stays intact ───────
function Format-Cell([string]$s) {
    ($s -replace "`r?`n", " " -replace "\|", "\|").Trim()
}

$ts   = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
$row  = "| $ts | $(Format-Cell $Path) | $(Format-Cell $Target) | $(Format-Cell $Description) |"

# ── newest first: insert under the header rule, never at the end ─────
# The table is read top-down and its first body row is the latest change, so an
# appended row lands at the *oldest* end and reads as history. Splice it in right
# after the "|---|---|---|---|" rule instead.
$lines = @(Get-Content -Path $logFile -Encoding utf8)
$rule  = ($lines | Select-String -SimpleMatch -Pattern "|---|---|---|---|" | Select-Object -First 1)
if ($null -eq $rule) {
    throw "$logFile has no '|---|---|---|---|' table rule - refusing to guess where the row goes."
}
$at = $rule.LineNumber   # 1-based, so this is already the index after the rule
$out = @()
if ($at -gt 0) { $out += $lines[0..($at - 1)] }
$out += $row
if ($at -lt $lines.Count) { $out += $lines[$at..($lines.Count - 1)] }
$out | Set-Content -Path $logFile -Encoding utf8

Write-Host "Inserted changelog row ($ts)" -ForegroundColor Green
Write-Host "  Target: $Target" -ForegroundColor DarkGray
Write-Host "  Path  : $Path" -ForegroundColor DarkGray
