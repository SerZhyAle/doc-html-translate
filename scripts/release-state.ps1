<#
.SYNOPSIS
  Track the multi-channel publish state of a release in a file (not in prose memory).

.DESCRIPTION
  A release fans out to 5 independent channels that go live at different times:

      GitHub  ->  winget  ->  Store  ->  Chrome  ->  Edge

  The session logs show this state being carried in prose ("запиши себе что мы
  отрелизились и ждём подтверждений") and re-asked later. This helper keeps it in
  DEV/RELEASE_STATE.md - a durable, greppable table - so the state survives across
  sessions and is never guessed.

  It RUNS NOTHING external: it only reads and rewrites the state file. Publishing is
  still the manual, per-step flow in scripts/release.ps1 / DEV/RELEASE.md. Use this to
  RECORD what you already did.

.PARAMETER Channel
  One of: GitHub, winget, Store, Chrome, Edge  (case-insensitive).
  Omit to just print the current table.

.PARAMETER Status
  Free text, but prefer: pending | submitted | live | blocked | n/a.

.PARAMETER Version
  The version/tag for this release (e.g. 26.0717.2200 or v26.0717.2200). Sets the
  release-wide version header when given.

.PARAMETER Ref
  Channel-specific reference: winget PR#, git tag, Store product id, CWS/Edge dashboard note.

.PARAMETER Note
  Short free-text note (e.g. "awaiting cert", "InProgressSubmission from prior cert").

.EXAMPLE
  ./scripts/release-state.ps1                                   # print current state
.EXAMPLE
  ./scripts/release-state.ps1 -Version 26.0717.2200 -Channel GitHub -Status live -Ref v26.0717.2200
.EXAMPLE
  ./scripts/release-state.ps1 -Channel Edge -Status blocked -Note "InProgressSubmission from prior cert"
#>
param(
    [ValidateSet('GitHub', 'winget', 'Store', 'Chrome', 'Edge', IgnoreCase = $true)]
    [string]$Channel,
    [string]$Status,
    [string]$Version,
    [string]$Ref,
    [string]$Note
)

$ErrorActionPreference = "Stop"

$repoRoot = (& git rev-parse --show-toplevel 2>$null)
if ($LASTEXITCODE -eq 0 -and $repoRoot) { Set-Location ($repoRoot.Trim()) }

$stateFile = "DEV/RELEASE_STATE.md"
$channels  = @('GitHub', 'winget', 'Store', 'Chrome', 'Edge')

# Canonical channel casing regardless of how the user typed it.
if ($Channel) { $Channel = $channels | Where-Object { $_ -ieq $Channel } | Select-Object -First 1 }

# ── load or seed the model ───────────────────────────────────
# NB: local var must NOT be named $version - PowerShell variable names are
# case-insensitive, so $version would alias the -Version parameter and wipe it.
$relVersion = ''
$rows = [ordered]@{}
foreach ($c in $channels) { $rows[$c] = @{ Status = 'pending'; Ref = ''; Note = ''; Updated = '' } }

if (Test-Path $stateFile) {
    foreach ($line in Get-Content -LiteralPath $stateFile -Encoding utf8) {
        if ($line -match '^\s*-\s*\*\*Version\*\*\s*:\s*(.+?)\s*$') {
            $v = $Matches[1]
            if ($v -notmatch '^\(') { $relVersion = $v }   # ignore a stored "(unset)"/"(none)"
            continue
        }
        # table row: | Channel | Status | Ref | Note | Updated |
        if ($line -match '^\s*\|\s*([A-Za-z]+)\s*\|(.*)\|\s*$') {
            $name = $channels | Where-Object { $_ -ieq $Matches[1].Trim() } | Select-Object -First 1
            if ($name) {
                $cells = $Matches[2] -split '\|' | ForEach-Object { $_.Trim() }
                $rows[$name] = @{
                    Status  = if ($cells.Count -ge 1) { $cells[0] } else { 'pending' }
                    Ref     = if ($cells.Count -ge 2) { $cells[1] } else { '' }
                    Note    = if ($cells.Count -ge 3) { $cells[2] } else { '' }
                    Updated = if ($cells.Count -ge 4) { $cells[3] } else { '' }
                }
            }
        }
    }
}

# ── apply an update, if requested ────────────────────────────
$changed = $false
if ($Version) { $relVersion = ($Version.TrimStart('v')); $changed = $true }
if ($Channel) {
    if ($Status) { $rows[$Channel].Status = $Status }
    if ($PSBoundParameters.ContainsKey('Ref'))  { $rows[$Channel].Ref  = $Ref }
    if ($PSBoundParameters.ContainsKey('Note')) { $rows[$Channel].Note = $Note }
    $rows[$Channel].Updated = Get-Date -Format "yyyy-MM-dd HH:mm"
    $changed = $true
}

# ── write the file back when something changed ───────────────
if ($changed) {
    $sb = [System.Collections.Generic.List[string]]::new()
    $sb.Add("# Release publish-state")
    $sb.Add("")
    $sb.Add("Durable state of the current release across all 5 channels. Updated via")
    $sb.Add("``scripts/release-state.ps1`` (alias ``a rs``). This file records what already happened;")
    $sb.Add("publishing itself is the manual flow in [RELEASE.md](RELEASE.md) / ``scripts/release.ps1``.")
    $sb.Add("")
    $sb.Add("- **Version** : $(if ($relVersion) { $relVersion } else { '(unset)' })")
    $sb.Add("")
    $sb.Add("| Channel | Status | Ref | Note | Updated |")
    $sb.Add("|---|---|---|---|---|")
    foreach ($c in $channels) {
        $r = $rows[$c]
        $sb.Add("| $c | $($r.Status) | $($r.Ref) | $($r.Note) | $($r.Updated) |")
    }
    $sb.Add("")
    $sb.Add("Status vocabulary: ``pending`` -> ``submitted`` -> ``live`` (or ``blocked`` / ``n/a``).")
    ($sb -join "`n") | Out-File -FilePath $stateFile -Encoding utf8
}

# ── always print the current state ───────────────────────────
Write-Host ""
Write-Host "=============== RELEASE STATE  (version: $(if ($relVersion) { $relVersion } else { 'unset' })) ===============" -ForegroundColor Cyan
foreach ($c in $channels) {
    $r = $rows[$c]
    $color = switch -Regex ($r.Status) {
        'live'      { 'Green' }
        'submitted' { 'Yellow' }
        'blocked'   { 'Red' }
        default     { 'Gray' }
    }
    $extra = @($r.Ref, $r.Note) | Where-Object { $_ } | ForEach-Object { $_ }
    $suffix = if ($extra) { "  (" + ($extra -join " · ") + ")" } else { "" }
    Write-Host ("  {0,-8} {1,-10}{2}" -f $c, $r.Status, $suffix) -ForegroundColor $color
}
Write-Host ""
Write-Host "File: $stateFile   (a rs -Channel Edge -Status live  to update)" -ForegroundColor DarkGray
