<#
.SYNOPSIS
  The pure checks behind build-store-listing-csv.ps1 (WINDOWS-STORE rules 7 to 9, conformance rung 2).

.DESCRIPTION
  Functions only, no side effects, so the builder can dot-source it and test-store-listing-csv.ps1 can
  call each check on crafted input without running a build. Nothing here reads or writes a file.

  Read-Csv / Format-Csv  the reader and writer of the export (minimal quoting, so a no-op run is byte-identical).
  Get-CsvDialect       measures an export and says which dialect it is in, or why it has none.
  Test-CsvOutput       reads the bytes a writer is about to write and lists every way they differ from that dialect.
  Get-CapViolations    lists every cell over a Partner Center cap, per language.
  Get-CellLength       the unit the caps are counted in.
  ConvertTo-CellBreak  rewrites a deck value's line breaks to the dialect's in-cell break.

  The caps are the numbers WINDOWS-STORE rule 9 records on the dates it gives; Partner Center's own message is
  the authority, and a refusal the table did not predict is amended in the contract first.
#>

# A cap is checked in the unit Partner Center counts: characters (Unicode code points), after the line breaks the
# file will carry, so every break counts as the two characters of CRLF. The contract says so for ReleaseNotes; the
# same reading is taken for every prose field because it is the conservative one.
function Get-CellLength([string]$value) {
    $crlf = [regex]::Replace($value, '\r\n|\r|\n', "`r`n")
    return $crlf.Length - [regex]::Matches($crlf, '[\uDC00-\uDFFF]').Count
}

# Field name -> cap in characters, or $null for a field that has none (assets, IDs, search terms).
function Get-ListingCap([string]$field) {
    switch -Regex ($field) {
        '^ShortDescription$' { return 1000 }
        '^Description$'      { return 10000 }
        '^ReleaseNotes$'     { return 1500 }
        '^Feature\d+$'       { return 200 }
    }
    return $null
}

# Every quoted cell, a quote that never closes, and every break outside quotes, counted by kind. The quote
# group is atomic so an unterminated cell stays unterminated instead of "closing" on one half of a doubled quote.
function Measure-CsvBreaks([string]$text) {
    $m = [pscustomobject]@{
        RecordCrLf = 0; RecordLf = 0; RecordCr = 0
        CellCrLf = 0; CellLf = 0; CellCr = 0
        OpenCell = $false; EndsWithBreak = ($text.EndsWith("`n") -or $text.EndsWith("`r"))
    }
    foreach ($t in [regex]::Matches($text, '"(?>(?:[^"]+|"")*)"|"|\r\n|\n|\r')) {
        $v = $t.Value
        if ($v -eq '"') { $m.OpenCell = $true; break }
        if ($v[0] -eq '"') {
            $m.CellCrLf += [regex]::Matches($v, '\r\n').Count
            $m.CellLf   += [regex]::Matches($v, '(?<!\r)\n').Count
            $m.CellCr   += [regex]::Matches($v, '\r(?!\n)').Count
        }
        elseif ($v -eq "`r`n") { $m.RecordCrLf++ }
        elseif ($v -eq "`n")   { $m.RecordLf++ }
        else                   { $m.RecordCr++ }
    }
    return $m
}

# The dialect of an export (0.3 item B): its BOM or none, its record break, the one break it uses inside cells
# and whether its last record ends with a break. An export that has none of these uniform - mixed record breaks,
# mixed in-cell breaks, a lone carriage return, a last cell left open - has no dialect to keep, and Problems says
# why. A file with no in-cell break at all takes LF, which is what a deck line break is.
function Get-CsvDialect([string]$text, [bool]$hasBom) {
    $m = Measure-CsvBreaks $text
    $problems = New-Object System.Collections.ArrayList
    if ($m.OpenCell) { [void]$problems.Add('the last quoted cell is left open') }
    if ($m.RecordCr -gt 0) { [void]$problems.Add("$($m.RecordCr) lone carriage return(s) between records") }
    if ($m.CellCr -gt 0) { [void]$problems.Add("$($m.CellCr) lone carriage return(s) inside cells") }
    if ($m.RecordCrLf -gt 0 -and $m.RecordLf -gt 0) {
        [void]$problems.Add("record breaks mix CRLF ($($m.RecordCrLf)) and bare LF ($($m.RecordLf))")
    }
    if ($m.CellCrLf -gt 0 -and $m.CellLf -gt 0) {
        [void]$problems.Add("in-cell breaks mix CRLF ($($m.CellCrLf)) and bare LF ($($m.CellLf))")
    }
    return [pscustomobject]@{
        Bom         = $hasBom
        RecordBreak = $(if ($m.RecordLf -gt 0) { "`n" } else { "`r`n" })
        CellBreak   = $(if ($m.CellCrLf -gt 0) { "`r`n" } else { "`n" })
        Trailing    = $m.EndsWithBreak
        Problems    = $problems.ToArray()
    }
}

# The bytes a writer is about to write, against the dialect it promised. Returns every violation, empty when the
# bytes are the dialect: a BOM the export does not carry (or the loss of the export's), a record that ends with a
# break other than the dialect's, an in-cell break of the other style, a lone carriage return, a last quoted cell
# left open, a trailing newline the export did not have. The 2026-08-07 failure - a bare LF closing the last row,
# which swallows the next record and makes Partner Center name an innocent language - is the second of these.
function Test-CsvOutput([byte[]]$bytes, $dialect) {
    $v = New-Object System.Collections.ArrayList
    $hasBom = $bytes.Length -ge 3 -and $bytes[0] -eq 0xEF -and $bytes[1] -eq 0xBB -and $bytes[2] -eq 0xBF
    if ($hasBom -and -not $dialect.Bom) { [void]$v.Add('the output starts with a BOM and the export has none') }
    if (-not $hasBom -and $dialect.Bom) { [void]$v.Add("the output has no BOM and the export's BOM would be lost") }
    $skip = if ($hasBom) { 3 } else { 0 }
    $text = [System.Text.UTF8Encoding]::new($false, $true).GetString($bytes, $skip, $bytes.Length - $skip)
    $m = Measure-CsvBreaks $text
    if ($m.OpenCell) { [void]$v.Add('the last quoted cell is left open') }
    if ($m.RecordCr -gt 0) { [void]$v.Add("$($m.RecordCr) record(s) end with a lone carriage return") }
    if ($m.CellCr -gt 0) { [void]$v.Add("$($m.CellCr) lone carriage return(s) inside cells") }
    $recordName = if ($dialect.RecordBreak -eq "`n") { 'LF' } else { 'CRLF' }
    $other = if ($dialect.RecordBreak -eq "`n") { $m.RecordCrLf } else { $m.RecordLf }
    if ($other -gt 0) {
        [void]$v.Add("$other record(s) end with a break that is not the export's $recordName" +
            $(if ($dialect.RecordBreak -eq "`r`n") { ' (a bare LF)' } else { ' (a CRLF)' }))
    }
    $cellName = if ($dialect.CellBreak -eq "`n") { 'LF' } else { 'CRLF' }
    $otherCell = if ($dialect.CellBreak -eq "`n") { $m.CellCrLf } else { $m.CellLf }
    if ($otherCell -gt 0) { [void]$v.Add("$otherCell in-cell break(s) are not the export's $cellName") }
    if ($m.EndsWithBreak -ne $dialect.Trailing) {
        [void]$v.Add($(if ($dialect.Trailing) { 'the output has lost the export''s trailing newline' } else { 'the output ends with a newline the export does not have' }))
    }
    return $v.ToArray()
}

function ConvertTo-CellBreak([string]$value, [string]$cellBreak) {
    return [regex]::Replace($value, '\r\n|\r|\n', $cellBreak)
}

# Every cell, in every language column named in $tags, that is over its cap: Feature 200, ShortDescription 1000,
# Description 10000, ReleaseNotes 1500, at most 7 search-term cells and 21 words over them (a word is a run of
# non-whitespace; the 21 is the portfolio's accepted conservative cap of the 2026-10-17 amendment). The check reads
# the cells that will reach the file, whether the deck wrote them or the export already held them - a retained cell
# over its cap loses the language just as a filled one does. $deckOf maps a column name to the deck file to name.
function Get-CapViolations($rows, $header, [int]$fieldCol, [string[]]$tags, $deckOf) {
    $found = New-Object System.Collections.ArrayList
    foreach ($tag in $tags) {
        $c = [array]::IndexOf($header, $tag)
        if ($c -lt 0) { continue }
        $words = 0
        $terms = 0
        for ($r = 1; $r -lt $rows.Count; $r++) {
            $row = $rows[$r]
            if ($c -ge $row.Count -or $fieldCol -ge $row.Count) { continue }
            $field = $row[$fieldCol]
            $value = $row[$c]
            if (-not $field -or $value -eq '') { continue }
            $cap = Get-ListingCap $field
            if ($cap) {
                $len = Get-CellLength $value
                if ($len -gt $cap) { [void]$found.Add("$tag $($field): $len characters, cap $cap ($($deckOf[$tag]))") }
            } elseif ($field -match '^SearchTerm\d+$') {
                $terms++
                $words += [regex]::Matches($value, '\S+').Count
            }
        }
        if ($terms -gt 7) { [void]$found.Add("$tag SearchTerm: $terms term cells, at most 7 ($($deckOf[$tag]))") }
        if ($words -gt 21) { [void]$found.Add("$tag SearchTerm1..7: $words words, cap 21 ($($deckOf[$tag]))") }
    }
    return $found.ToArray()
}

# -- CSV reader / writer (minimal quoting, so a no-op run is byte-identical) --
function Read-Csv([string]$text) {
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

# $trailing says whether the file ends with a newline after its last row. It is taken from the
# export rather than assumed, because the -FillNothing round-trip guard compares bytes: Partner
# Center exports without a final newline, and appending one made an otherwise byte-identical
# rewrite report "round trip DIFFERS", which is the one signal that makes a patching run
# trustworthy. A guard that cries wolf over two bytes teaches a reader to ignore it.
function Format-Csv($rows, [string]$newline, [bool]$trailing) {
    $sb = New-Object System.Text.StringBuilder
    for ($i = 0; $i -lt $rows.Count; $i++) {
        [void]$sb.Append((($rows[$i] | ForEach-Object { Format-Cell $_ }) -join ','))
        if ($i -lt $rows.Count - 1 -or $trailing) { [void]$sb.Append($newline) }
    }
    return $sb.ToString()
}
