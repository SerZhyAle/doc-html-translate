<#
.SYNOPSIS
  Fill a Partner Center listing export with the per-language copy from tools/store/listing/*.txt.

.DESCRIPTION
  Partner Center round-trips a listing as a CSV: one row per field, one column per locale. The rule
  this script exists to enforce is "patch, never regenerate" - it fills **only empty cells** and leaves
  everything else, in particular the listing-asset URLs Partner Center puts in the screenshot and logo
  rows, exactly as exported. Those URLs point at images already uploaded to the dashboard; rewriting
  them detaches the assets and the listing loses its images.

  Column order is never changed and no column is invented: a locale the Store does not yet know about
  is dropped on import without an error message, so adding one here would look like it worked and
  quietly do nothing. Add the locale in Partner Center, export again, then run this.

  Encoding is UTF-8 **with** BOM, matching what Partner Center itself exports. Without the BOM the
  dashboard reads Cyrillic, Arabic and CJK cells as mojibake and the import "succeeds" with ruined text.

.PARAMETER Csv
  The export to patch. Default: tools/store/listingData.csv.

.PARAMETER Out
  Where to write. Default: in place.

.PARAMETER ImportFolder
  Copy the result next to the per-locale screenshots, as the folder you hand to the dashboard.

.PARAMETER SkipFields
  Field names to leave alone even when empty (e.g. ReleaseNotes during a copy-only edit).

.PARAMETER Refresh
  Field names to overwrite even when the cell already has a value. `ReleaseNotes` is the one field
  that is rewritten every release, so it needs this; everything else is fill-only on purpose. Never
  pass a screenshot or logo field - those cells hold Partner Center's own asset URLs.

.PARAMETER FillNothing
  Parse and rewrite without filling anything. A byte-identical result proves the reader and writer
  round-trip this export, which is what makes the patching runs trustworthy.

.EXAMPLE
  .\tools\store\build-store-listing-csv.ps1 -FillNothing -Out temp\roundtrip.csv
.EXAMPLE
  .\tools\store\build-store-listing-csv.ps1 -ImportFolder temp\store-import
#>
param(
    [string]$Csv,
    [string]$Out,
    [string]$ImportFolder,
    [string[]]$SkipFields = @(),
    [string[]]$Refresh = @(),
    [switch]$FillNothing
)

$ErrorActionPreference = "Stop"
$RepoRoot   = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$ListingDir = Join-Path $PSScriptRoot "listing"
if (-not $Csv) { $Csv = Join-Path $PSScriptRoot "listingData.csv" }
if (-not (Test-Path $Csv)) { throw "listing export not found: $Csv" }
# .NET file APIs resolve relative paths against the process directory, which PowerShell does not
# keep in step with the session location - so make every path absolute before anything writes.
$Csv = (Resolve-Path $Csv).Path
if (-not $Out) { $Out = $Csv } else { $Out = [System.IO.Path]::GetFullPath((Join-Path (Get-Location) $Out)) }
if ($ImportFolder) { $ImportFolder = [System.IO.Path]::GetFullPath((Join-Path (Get-Location) $ImportFolder)) }

# language code -> Partner Center column header
$Locale = [ordered]@{
    en = "en-us"; ru = "ru"; uk = "uk"; de = "de"; it = "it"; es = "es"; fr = "fr"
    pt = "pt-br"; ar = "ar"; hi = "hi"; bn = "bn"; ur = "ur"; zh = "zh-hans"
}

# ── CSV reader / writer (minimal quoting, so a no-op run is byte-identical) ──
function Read-Csv([string]$path) {
    $text = [System.IO.File]::ReadAllText($path, [System.Text.UTF8Encoding]::new($false))
    $rows = New-Object System.Collections.ArrayList
    $row  = New-Object System.Collections.ArrayList
    $sb   = New-Object System.Text.StringBuilder
    $inQuotes = $false
    for ($i = 0; $i -lt $text.Length; $i++) {
        $ch = $text[$i]
        if ($inQuotes) {
            if ($ch -eq '"') {
                if ($i + 1 -lt $text.Length -and $text[$i + 1] -eq '"') { [void]$sb.Append('"'); $i++ }
                else { $inQuotes = $false }
            } else { [void]$sb.Append($ch) }
            continue
        }
        switch ($ch) {
            '"'  { $inQuotes = $true }
            ','  { [void]$row.Add($sb.ToString()); $sb.Clear() | Out-Null }
            "`r" { }
            "`n" { [void]$row.Add($sb.ToString()); $sb.Clear() | Out-Null; [void]$rows.Add($row.ToArray()); $row.Clear() }
            default { [void]$sb.Append($ch) }
        }
    }
    if ($sb.Length -gt 0 -or $row.Count -gt 0) { [void]$row.Add($sb.ToString()); [void]$rows.Add($row.ToArray()) }
    return $rows
}

function Format-Cell([string]$v) {
    if ($v -match '[",\r\n]') { return '"' + $v.Replace('"', '""') + '"' }
    return $v
}

function Write-Csv($rows, [string]$path, [string]$newline) {
    $sb = New-Object System.Text.StringBuilder
    foreach ($r in $rows) {
        [void]$sb.Append((($r | ForEach-Object { Format-Cell $_ }) -join ','))
        [void]$sb.Append($newline)
    }
    # UTF-8 *with* BOM - Partner Center's own export encoding; see the note in .DESCRIPTION.
    [System.IO.File]::WriteAllText($path, $sb.ToString(), [System.Text.UTF8Encoding]::new($true))
}

# ── read the per-language sources ────────────────────────────────────────────
function Read-ListingSource([string]$path) {
    $fields = [ordered]@{}
    $name = $null
    $buf = New-Object System.Collections.ArrayList
    foreach ($line in [System.IO.File]::ReadAllLines($path)) {
        if ($line -match '^@@(\w+)\s*$') {
            if ($name) { $fields[$name] = ($buf -join "`n").Trim() }
            $name = $Matches[1]; $buf.Clear()
        } elseif ($name -and $line -notmatch '^#') {
            [void]$buf.Add($line)
        }
    }
    if ($name) { $fields[$name] = ($buf -join "`n").Trim() }
    return $fields
}

$source = @{}
foreach ($code in $Locale.Keys) {
    $p = Join-Path $ListingDir "$code.txt"
    if (Test-Path $p) { $source[$Locale[$code]] = Read-ListingSource $p }
}

# ── patch ────────────────────────────────────────────────────────────────────
$raw = [System.IO.File]::ReadAllText($Csv, [System.Text.UTF8Encoding]::new($false))
$newline = if ($raw -match "`r`n") { "`r`n" } else { "`n" }
$rows = Read-Csv $Csv
if ($rows.Count -lt 2) { throw "export has no data rows: $Csv" }

$header = $rows[0]
$fieldCol = [array]::IndexOf($header, "Field")
if ($fieldCol -lt 0) { throw "no 'Field' column in $Csv" }

$present = @()
$missing = @()
foreach ($tag in $Locale.Values) {
    if ($header -contains $tag) { $present += $tag } else { $missing += $tag }
}

# Asset rows must never be refreshed: their cells hold the URLs of images already uploaded to the
# dashboard, and rewriting one detaches the asset.
$assetFields = $rows | ForEach-Object { $_[$fieldCol] } | Where-Object { $_ -match 'Screenshot|Logo' }
foreach ($f in $Refresh) {
    if ($assetFields -contains $f) { throw "-Refresh $f would overwrite a listing-asset URL; refuse" }
}

$filled = 0
$refreshed = 0
if (-not $FillNothing) {
    for ($r = 1; $r -lt $rows.Count; $r++) {
        $row = $rows[$r]
        $field = $row[$fieldCol]
        if (-not $field -or $SkipFields -contains $field) { continue }
        foreach ($tag in $present) {
            $c = [array]::IndexOf($header, $tag)
            if ($c -lt 0 -or $c -ge $row.Count) { continue }
            $isRefresh = $Refresh -contains $field
            if ($row[$c] -ne "" -and -not $isRefresh) { continue }  # asset URLs live in these cells
            $value = $source[$tag][$field]
            if (-not $value) { continue }
            if ($row[$c] -eq $value) { continue }
            if ($row[$c] -ne "") { $refreshed++ } else { $filled++ }
            $row[$c] = $value
        }
        $rows[$r] = $row
    }
}

New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Out) | Out-Null
Write-Csv $rows $Out $newline

if ($FillNothing) {
    $same = ([System.IO.File]::ReadAllBytes($Csv) -join ',') -eq ([System.IO.File]::ReadAllBytes($Out) -join ',')
    Write-Host ("round trip: {0}" -f $(if ($same) { "byte-identical" } else { "DIFFERS - do not trust a patching run" })) `
        -ForegroundColor $(if ($same) { "Green" } else { "Red" })
    if (-not $same) { exit 1 }
} else {
    Write-Host "filled $filled empty cells, refreshed $refreshed, across $($present.Count) locales -> $Out" -ForegroundColor Green
}

if ($missing.Count) {
    Write-Host ("locales not in this export (add them in Partner Center, then export again): {0}" -f ($missing -join ' ')) -ForegroundColor Yellow
}

if ($ImportFolder) {
    New-Item -ItemType Directory -Force -Path $ImportFolder | Out-Null
    Copy-Item $Out (Join-Path $ImportFolder "listingData.csv") -Force
    $shots = Get-ChildItem $PSScriptRoot -Filter *.png -File
    foreach ($s in $shots) { Copy-Item $s.FullName (Join-Path $ImportFolder $s.Name) -Force }
    Write-Host "import folder ready: $ImportFolder ($($shots.Count) images)" -ForegroundColor Cyan
}
