<#
.SYNOPSIS
  Fill a Partner Center listing export with the per-language copy from tools/store/listing/*.txt.

.DESCRIPTION
  Partner Center round-trips a listing as a CSV: one row per field, one column per locale. The rule
  this script exists to enforce is "patch, never regenerate" - it fills **only empty cells** and leaves
  everything else, in particular the listing-asset URLs Partner Center puts in the screenshot and logo
  rows, exactly as exported. Those URLs point at images already uploaded to the dashboard; rewriting
  them detaches the assets and the listing loses its images.

  tools/store/listingData.csv is the authoritative listing file (AGENTS.md, WINDOWS-STORE rule 12):
  the product's own export, patched in place. msix/out/store-import/ holds a gitignored copy of the
  result for the dashboard's folder import, never a second source. tools/store/listingData.sample.csv
  is the same export with every copy cell emptied, the fixture the round trip is proven on.

  Column order is never changed and no column is invented: an import cannot create a language, and a
  locale the Store does not yet know about is dropped on import without an error message. So a shipped
  language with no column in the export is refused here - add it in Partner Center, export again, then
  run this.

  The file is written in the dialect of the export it patches (WINDOWS-STORE 0.3 item B): its BOM or its
  absence, its record break and its one in-cell break. Partner Center's own export is UTF-8 with a BOM
  and CRLF between records and LF inside cells; without the BOM the dashboard reads Cyrillic, Arabic and
  CJK cells as mojibake and the import "succeeds" with ruined text.

  Nothing is written until every check has passed, and a refusal leaves the previous output as it was.
  A run is refused - exit 1, one "refused:" line per reason, nothing written - for:
    - a shipped language with no column in the export, or a column with no deck (rule 7);
    - an export or an output whose record break is not one break, whose in-cell breaks are mixed, that holds a
      lone carriage return, or whose last quoted cell is left open (rule 8, item B);
    - a cell over its cap, whether the deck filled it or the export already held it: Feature 200,
      ShortDescription 1000, Description 10000, ReleaseNotes 1500 (counted after CRLF normalisation), at
      most 7 search terms and 21 words over SearchTerm1..7 (rule 9);
    - a -Refresh of a field that does not exist or is not prose (rule 12).
  Exit codes: 0 written (or the round trip is byte-identical); 1 refused, or the round trip differs, or an
  error stopped the run. A missing file stops with a message and exit 1 as well.

.PARAMETER Csv
  The export to patch. Default: tools/store/listingData.csv.

.PARAMETER Out
  Where to write. Default: in place; with -FillNothing, temp/store-roundtrip.csv, because a round trip
  written over its own input compares a file with itself.

.PARAMETER ImportFolder
  Copy the result next to the per-locale screenshots, as the folder you hand to the dashboard.
  Defaults to `msix/out/store-import`, which puts it beside the MSIX the same upload session needs -
  the two halves of a Partner Center submission live in one place instead of two, and neither is in
  the repository (`msix/out/` is gitignored, like the `temp/` it used to sit in). An empty value skips
  the copy; -FillNothing never copies.

.PARAMETER ListingDir
  The folder of per-language decks (<code>.txt). Default: tools/store/listing. The test points it at
  synthetic decks.

.PARAMETER SkipFields
  Field names to leave alone even when empty (e.g. ReleaseNotes during a copy-only edit).

.PARAMETER Refresh
  Field names to overwrite even when the cell already has a value. `ReleaseNotes` is the one field
  that is rewritten every release, so it needs this; everything else is fill-only on purpose. Every
  cell it replaces is listed with the length it replaced. Never pass a screenshot or logo field - those
  cells hold Partner Center's own asset URLs; the run refuses one.

.PARAMETER FillNothing
  Parse and rewrite without filling anything. A byte-identical result proves the reader and writer
  round-trip this export, which is what makes the patching runs trustworthy.

.EXAMPLE
  .\tools\store\build-store-listing-csv.ps1 -FillNothing -Out temp\roundtrip.csv
.EXAMPLE
  .\tools\store\build-store-listing-csv.ps1 -ImportFolder msix\out\store-import
.EXAMPLE
  .\tools\store\build-store-listing-csv.ps1 -Refresh ReleaseNotes
#>
param(
    [string]$Csv,
    [string]$Out,
    [string]$ImportFolder = "msix/out/store-import",
    [string]$ListingDir,
    # Fill empty DesktopScreenshotN cells with the per-locale image file names from -ImportFolder,
    # so a locale that has no assets yet gets them attached by the import. Cells that already hold a
    # value are never touched: Partner Center puts its own asset URLs there, and rewriting one
    # detaches an image already uploaded to the dashboard. Off by default - a listing pass that only
    # edits copy has no business near the asset rows.
    #
    # A file name is valid ONLY through the dashboard's folder import, where the images travel with
    # the CSV. Importing the CSV on its own answers "The value you provided is not valid
    # (reading-view-de.png)" and refuses every language in the file, including the ones that were
    # already correct - measured 2026-08-13. So the output of a -Screenshots run is the folder (or
    # its zip), never the bare CSV.
    [switch]$Screenshots,
    [string[]]$SkipFields = @(),
    [string[]]$Refresh = @(),
    [switch]$FillNothing
)

$ErrorActionPreference = "Stop"
$RepoRoot   = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
. (Join-Path $PSScriptRoot "store-listing-checks.ps1")

# .NET file APIs resolve relative paths against the process directory, which PowerShell does not
# keep in step with the session location - so make every path absolute before anything writes.
# (Join-Path onto an already absolute path builds an invalid one, hence the rooted test.)
function Resolve-FullPath([string]$p) {
    if (-not [System.IO.Path]::IsPathRooted($p)) { $p = Join-Path (Get-Location) $p }
    return [System.IO.Path]::GetFullPath($p)
}

if (-not $ListingDir) { $ListingDir = Join-Path $PSScriptRoot "listing" }
$ListingDir = Resolve-FullPath $ListingDir
if (-not $Csv) { $Csv = Join-Path $PSScriptRoot "listingData.csv" }
if (-not (Test-Path $Csv)) { throw "listing export not found: $Csv" }
$Csv = (Resolve-Path $Csv).Path
if ($Out) { $Out = Resolve-FullPath $Out }
elseif ($FillNothing) { $Out = Resolve-FullPath (Join-Path $RepoRoot "temp/store-roundtrip.csv") }
else { $Out = $Csv }
if ($ImportFolder) { $ImportFolder = Resolve-FullPath $ImportFolder }
if ($FillNothing -and $Out -ieq $Csv) {
    Write-Host "refused: -FillNothing with -Out equal to -Csv compares the file with itself; name another -Out" -ForegroundColor Red
    exit 1
}

# language code -> Partner Center column header
$Locale = [ordered]@{
    en = "en-us"; ru = "ru"; uk = "uk"; de = "de"; it = "it"; es = "es"; fr = "fr"
    pt = "pt-br"; ar = "ar"; hi = "hi"; bn = "bn"; ur = "ur"; zh = "zh-hans"
}

# -- read the per-language sources --
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

# A refusal: one line per reason, nothing written, exit 1 (see .DESCRIPTION).
function Stop-Refused([string[]]$reasons) {
    foreach ($r in $reasons) { Write-Host "refused: $r" -ForegroundColor Red }
    Write-Host "refused: nothing written" -ForegroundColor Red
    exit 1
}

# decks, named by the path a reader can open
$deckShown = if ($ListingDir.StartsWith($RepoRoot, [System.StringComparison]::OrdinalIgnoreCase)) {
    $ListingDir.Substring($RepoRoot.Length).TrimStart('\', '/').Replace('\', '/')
} else { $ListingDir }
$source = @{}
$deckOf = @{}
foreach ($code in $Locale.Keys) {
    $p = Join-Path $ListingDir "$code.txt"
    $deckOf[$Locale[$code]] = "$deckShown/$code.txt"
    if (Test-Path $p) { $source[$Locale[$code]] = Read-ListingSource $p }
}

# -- read the export and its dialect --
$bytes = [System.IO.File]::ReadAllBytes($Csv)
$hasBom = $bytes.Length -ge 3 -and $bytes[0] -eq 0xEF -and $bytes[1] -eq 0xBB -and $bytes[2] -eq 0xBF
$skip = if ($hasBom) { 3 } else { 0 }
# Strict: an export that is not valid UTF-8 would be rewritten with replacement characters.
$raw = [System.Text.UTF8Encoding]::new($false, $true).GetString($bytes, $skip, $bytes.Length - $skip)
$dialect = Get-CsvDialect $raw $hasBom
$newline = $dialect.RecordBreak
$rows = Read-Csv $raw
if ($rows.Count -lt 2) { throw "export has no data rows: $Csv" }

$header = $rows[0]
$fieldCol = [array]::IndexOf($header, "Field")
if ($fieldCol -lt 0) { throw "no 'Field' column in $Csv" }
$typeCol = -1
for ($i = 0; $i -lt $header.Count; $i++) { if ($header[$i] -match '^Type\b') { $typeCol = $i; break } }

$present = @()
$missing = @()
foreach ($tag in $Locale.Values) {
    if ($header -contains $tag) { $present += $tag } else { $missing += $tag }
}

# -- refuse what cannot be patched safely, all reasons at once --
$refusals = New-Object System.Collections.ArrayList
foreach ($p in $dialect.Problems) { [void]$refusals.Add("export dialect: $p") }
foreach ($tag in $missing) {
    [void]$refusals.Add("no column for $tag in $(Split-Path -Leaf $Csv): the language is shipped, an import cannot create a column and would drop its copy - add it in Partner Center, export again")
}
if (-not $FillNothing) {
    foreach ($tag in $present) {
        if (-not $source.ContainsKey($tag)) {
            [void]$refusals.Add("no deck for $tag ($($deckOf[$tag])): the column would stay empty")
        }
    }
}

# Asset rows must never be refreshed: their cells hold the URLs of images already uploaded to the
# dashboard, and rewriting one detaches the asset. A prose row is a Text row; anything else (a
# relative path or URL to a file in Partner Center) is not ours to rewrite.
$fieldRows = @{}
for ($r = 1; $r -lt $rows.Count; $r++) {
    $name = $rows[$r][$fieldCol]
    if ($name) { $fieldRows[$name] = $rows[$r] }
}
foreach ($f in $Refresh) {
    if (-not $fieldRows.ContainsKey($f)) {
        [void]$refusals.Add("-Refresh $f names no field of the export")
    } elseif ($f -match 'Screenshot|Logo|Trailer' -or ($typeCol -ge 0 -and $typeCol -lt $fieldRows[$f].Count -and $fieldRows[$f][$typeCol] -ne 'Text')) {
        [void]$refusals.Add("-Refresh $f would overwrite a listing-asset URL or file name, which is not prose")
    }
}
if ($refusals.Count) { Stop-Refused $refusals.ToArray() }

# -- patch --
$filled = 0
$replaced = New-Object System.Collections.ArrayList
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
            # A deck holds LF; the file holds the export's own in-cell break throughout.
            $value = ConvertTo-CellBreak $value $dialect.CellBreak
            # -ceq, not -eq: PowerShell compares strings case-insensitively, so a correction that
            # differs only in case - "doc-html-translate" to the reserved "Doc-HTML-Translate" - was
            # silently treated as "already up to date" and never written.
            if ($row[$c] -ceq $value) { continue }
            if ($row[$c] -ne "") {
                [void]$replaced.Add([pscustomobject]@{ Tag = $tag; Field = $field; Old = (Get-CellLength $row[$c]); New = (Get-CellLength $value) })
            } else { $filled++ }
            $row[$c] = $value
        }
        $rows[$r] = $row
    }
}

# Screenshots are attached by file name, and only where the locale has none. Partner Center's import
# resolves a bare name against the images sitting beside the CSV in the folder it is given, which is
# how a locale that was added with the package gets its first assets; a cell that already holds a
# value holds the dashboard's own URL for an image already uploaded, and overwriting it detaches the
# image. Order is the order of $shotKinds - the same three views in every language.
$shotKinds = @("reading-view", "table-of-contents", "gui")
$shots = 0
if ($Screenshots -and -not $FillNothing) {
    $dir = if ($ImportFolder -and (Test-Path $ImportFolder)) { $ImportFolder } else { (Join-Path (Get-Location) "tools/store") }
    for ($r = 1; $r -lt $rows.Count; $r++) {
        $row = $rows[$r]
        if ($row[$fieldCol] -notmatch '^DesktopScreenshot(\d+)$') { continue }
        $n = [int]$Matches[1]
        if ($n -lt 1 -or $n -gt $shotKinds.Count) { continue }
        foreach ($tag in $present) {
            $c = [array]::IndexOf($header, $tag)
            if ($c -lt 0 -or $c -ge $row.Count -or $row[$c] -ne "") { continue }
            $name = "{0}-{1}.png" -f $shotKinds[$n - 1], $tag
            if (-not (Test-Path (Join-Path $dir $name))) { continue }
            $row[$c] = $name
            $shots++
        }
        $rows[$r] = $row
    }
}

# -- check what will be written, then write it --
$refusals.Clear()
foreach ($v in @(Get-CapViolations $rows $header $fieldCol $present $deckOf)) { [void]$refusals.Add("cap: $v") }
$text = Format-Csv $rows $newline $dialect.Trailing
$body = [System.Text.UTF8Encoding]::new($false).GetBytes($text)
$outBytes = if ($dialect.Bom) { [byte[]](@(0xEF, 0xBB, 0xBF) + $body) } else { $body }
foreach ($v in @(Test-CsvOutput $outBytes $dialect)) { [void]$refusals.Add("output dialect: $v") }
if ($refusals.Count) { Stop-Refused $refusals.ToArray() }

New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Out) | Out-Null
[System.IO.File]::WriteAllBytes($Out, $outBytes)

if ($FillNothing) {
    $same = [Convert]::ToBase64String([System.IO.File]::ReadAllBytes($Csv)) -ceq [Convert]::ToBase64String([System.IO.File]::ReadAllBytes($Out))
    Write-Host ("round trip: {0}" -f $(if ($same) { "byte-identical" } else { "DIFFERS - do not trust a patching run" })) `
        -ForegroundColor $(if ($same) { "Green" } else { "Red" })
    if (-not $same) { exit 1 }
} else {
    # A replace is never silent (WINDOWS-STORE rule 12): every cell that stood before, with its length.
    if ($replaced.Count) {
        Write-Host "-Refresh replaced $($replaced.Count) cells:" -ForegroundColor Yellow
        foreach ($x in $replaced) { Write-Host ("  replaced {0} {1}: {2} -> {3} characters" -f $x.Tag, $x.Field, $x.Old, $x.New) -ForegroundColor Yellow }
    }
    Write-Host "filled $filled empty cells, refreshed $($replaced.Count), attached $shots screenshots, across $($present.Count) locales -> $Out" -ForegroundColor Green
}

if ($ImportFolder -and -not $FillNothing) {
    New-Item -ItemType Directory -Force -Path $ImportFolder | Out-Null
    Copy-Item $Out (Join-Path $ImportFolder "listingData.csv") -Force
    $pngs = Get-ChildItem $PSScriptRoot -Filter *.png -File
    foreach ($s in $pngs) { Copy-Item $s.FullName (Join-Path $ImportFolder $s.Name) -Force }
    Write-Host "import folder ready: $ImportFolder ($($pngs.Count) images)" -ForegroundColor Cyan
}
