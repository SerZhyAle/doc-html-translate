<#
.SYNOPSIS
  The negative and positive cases of build-store-listing-csv.ps1 (WINDOWS-STORE conformance rungs 1 and 2).

.DESCRIPTION
  One PowerShell process runs every case: the builder is called in-process, each case on its own small
  synthetic export and synthetic decks in a scratch folder, so the whole run stays inside the fast tier
  (5 s or less). Three cases read the committed export, its emptied fixture and the real decks.

  What it proves:
    - a shipped language with no column, a column with no deck, a cell over any cap (deck value or a cell
      the export already held), the 21 words over the search terms, a mixed in-cell break, a record break
      that is not the export's, a lone carriage return and an open last cell each end with exit 1, name the
      language and the field, and write nothing (a previous output, or the export itself on an in-place run,
      is left as it was);
    - the writer keeps the export's BOM or its absence, its record break, its in-cell break and its trailing
      newline; a value exactly at each cap is accepted; -Refresh lists every cell it replaces with the length;
    - the -FillNothing round trip is byte-identical on a synthetic export, on the committed fixture and on
      the committed export.

  Exit codes: 0 every case passed; 1 a case failed. The last line is "cases=<N> failed=<M>".

.PARAMETER Work
  Scratch folder. Default: a new folder under the system temp folder, removed at the end. A caller that
  passes one owns its removal.

.EXAMPLE
  pwsh -NoProfile -File tools\store\test-store-listing-csv.ps1
#>
param([string]$Work)

$ErrorActionPreference = 'Stop'
$Builder = Join-Path $PSScriptRoot 'build-store-listing-csv.ps1'
$Repo = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
. (Join-Path $PSScriptRoot 'store-listing-checks.ps1')

$ownsWork = -not $Work
if ($ownsWork) { $Work = Join-Path ([System.IO.Path]::GetTempPath()) ('store-listing-test-' + [guid]::NewGuid().ToString('N').Substring(0, 8)) }
New-Item -ItemType Directory -Force -Path $Work | Out-Null

$Utf8 = [System.Text.UTF8Encoding]::new($false)
$Tags = @('en-us', 'ru', 'uk', 'de', 'it', 'es', 'fr', 'pt-br', 'ar', 'hi', 'bn', 'ur', 'zh-hans')
$Codes = @('en', 'ru', 'uk', 'de', 'it', 'es', 'fr', 'pt', 'ar', 'hi', 'bn', 'ur', 'zh')
$AssetType = 'Relative path (or URL to file in Partner Center)'

# -- fixtures --
# A small export in the shape Partner Center writes: header, one row per field, one column per language, blank
# rows at the end. Options: Bom, Rec (record break), Trailing, DropColumn, Cells ('tag|Field' -> value),
# Mutate (a scriptblock over the finished text).
function Write-Export([string]$path, [hashtable]$o = @{}) {
    $bom = if ($o.ContainsKey('Bom')) { [bool]$o.Bom } else { $true }
    $rec = if ($o.Rec) { $o.Rec } else { "`r`n" }
    $cells = if ($o.Cells) { $o.Cells } else { @{} }
    $tags = @($Tags | Where-Object { $_ -ne $o.DropColumn })
    $rows = New-Object System.Collections.ArrayList
    [void]$rows.Add(@('Field', 'ID', 'Type (Type)', 'default') + $tags)
    $spec = @(
        @('Description', 2, 'Text'), @('ReleaseNotes', 3, 'Text'), @('Title', 4, 'Text'), @('ShortDescription', 8, 'Text'),
        @('DesktopScreenshot1', 100, $AssetType), @('Feature1', 700, 'Text'), @('Feature2', 701, 'Text'),
        @('SearchTerm1', 900, 'Text'), @('SearchTerm2', 901, 'Text'), @('SearchTerm3', 902, 'Text'), @('SearchTerm4', 903, 'Text'),
        @('SearchTerm5', 904, 'Text'), @('SearchTerm6', 905, 'Text'), @('SearchTerm7', 906, 'Text'))
    foreach ($s in $spec) {
        $row = @($s[0], [string]$s[1], $s[2], '')
        foreach ($t in $tags) {
            $k = "$t|$($s[0])"
            $row += $(if ($cells.ContainsKey($k)) { $cells[$k] } elseif ($s[0] -eq 'DesktopScreenshot1') { "https://example.invalid/$t/asset" } else { '' })
        }
        [void]$rows.Add($row)
    }
    $blank = @('', '', '', '') + @($tags | ForEach-Object { '' })
    [void]$rows.Add($blank); [void]$rows.Add($blank)
    $text = Format-Csv $rows $rec ([bool]$o.Trailing)
    if ($o.Mutate) { $text = & $o.Mutate $text }
    $body = $Utf8.GetBytes($text)
    [System.IO.File]::WriteAllBytes($path, $(if ($bom) { [byte[]](@(0xEF, 0xBB, 0xBF) + $body) } else { $body }))
}

# Thirteen small decks. Search terms are 7 cells of 3 words, exactly the 21-word cap. $o overrides 'code|Field'.
function Write-Decks([string]$dir, [hashtable]$o = @{}) {
    New-Item -ItemType Directory -Force -Path $dir | Out-Null
    foreach ($code in $Codes) {
        $f = [ordered]@{
            Title = "Title-$code"; ShortDescription = "Short $code"; Description = "Line one $code`nLine two $code"
            Feature1 = "Feature one $code"; Feature2 = "Feature two $code"
        }
        foreach ($i in 1..7) { $f["SearchTerm$i"] = "w${i}a w${i}b w${i}c" }
        $f['ReleaseNotes'] = "Notes $code"
        foreach ($k in @($f.Keys)) { if ($o.ContainsKey("$code|$k")) { $f[$k] = $o["$code|$k"] } }
        $text = "# deck $code`n"
        foreach ($k in $f.Keys) { $text += "`n@@$k`n$($f[$k])`n" }
        [System.IO.File]::WriteAllText((Join-Path $dir "$code.txt"), $text, $Utf8)
    }
}

# The builder, in-process: exit code and everything it printed.
function Invoke-Builder([hashtable]$p) {
    $global:LASTEXITCODE = 0
    $log = ''
    try { $log = (& $Builder @p *>&1 | Out-String) }
    catch { $log = 'EXCEPTION: ' + $_.Exception.Message; $global:LASTEXITCODE = 1 }
    return [pscustomobject]@{ Code = $LASTEXITCODE; Log = $log }
}

# One fill run on a fresh synthetic export and decks. -Extra adds or overrides builder parameters, -InPlace omits -Out.
function Invoke-Fill([hashtable]$Export = @{}, [hashtable]$Decks = @{}, [hashtable]$Extra = @{}, [string]$DropDeck, [switch]$InPlace, [switch]$PreOut) {
    $d = Join-Path $Work "c$($script:total)"
    New-Item -ItemType Directory -Force -Path $d | Out-Null
    $csv = Join-Path $d 'export.csv'
    $out = Join-Path $d 'out.csv'
    Write-Export $csv $Export
    Write-Decks (Join-Path $d 'listing') $Decks
    if ($DropDeck) { Remove-Item (Join-Path $d "listing/$DropDeck.txt") }
    if ($PreOut) { [System.IO.File]::WriteAllBytes($out, $Utf8.GetBytes('SENTINEL')) }
    $p = @{ Csv = $csv; ListingDir = (Join-Path $d 'listing'); ImportFolder = '' }
    if (-not $InPlace) { $p.Out = $out }
    foreach ($k in $Extra.Keys) { $p[$k] = $Extra[$k] }
    $before = [System.IO.File]::ReadAllBytes($csv)
    $r = Invoke-Builder $p
    return [pscustomobject]@{ Dir = $d; Csv = $csv; Out = $out; Before = $before; Code = $r.Code; Log = $r.Log }
}

function Get-FileText([string]$path) {
    $b = [System.IO.File]::ReadAllBytes($path)
    $skip = if ($b.Length -ge 3 -and $b[0] -eq 0xEF -and $b[1] -eq 0xBB -and $b[2] -eq 0xBF) { 3 } else { 0 }
    return $Utf8.GetString($b, $skip, $b.Length - $skip)
}
function Test-FileBom([string]$path) {
    $b = [System.IO.File]::ReadAllBytes($path)
    return ($b.Length -ge 3 -and $b[0] -eq 0xEF -and $b[1] -eq 0xBB -and $b[2] -eq 0xBF)
}
function Test-SameBytes([byte[]]$a, [byte[]]$b) { return [Convert]::ToBase64String($a) -ceq [Convert]::ToBase64String($b) }

# -- the case runner --
$script:total = 0
$script:failed = 0
function Case([string]$name, [scriptblock]$body) {
    $script:total++
    try { & $body; Write-Host "PASS $name" -ForegroundColor Green }
    catch { $script:failed++; Write-Host "FAIL $name - $($_.Exception.Message)" -ForegroundColor Red }
}
function Assert-That([bool]$cond, [string]$msg) { if (-not $cond) { throw $msg } }

# A refusal: exit 1, the reason in the log, and no output - none created, a previous one untouched, the export untouched.
function Assert-Refused($r, [string[]]$reasons, [switch]$InPlace) {
    Assert-That ($r.Code -eq 1) "expected exit 1, got $($r.Code): $($r.Log)"
    foreach ($x in $reasons) { Assert-That ($r.Log -match $x) "log lacks /$x/: $($r.Log)" }
    Assert-That ($r.Log -match 'refused: nothing written') 'log lacks "refused: nothing written"'
    if ($InPlace) { Assert-That (Test-SameBytes $r.Before ([System.IO.File]::ReadAllBytes($r.Csv))) 'the export was changed by a refused run' }
    elseif (Test-Path $r.Out) { Assert-That ((Get-FileText $r.Out) -ceq 'SENTINEL') 'a refused run wrote the output' }
}

# -- positive: the dialect is kept --
Case 'fill keeps the export dialect and fills every deck cell' {
    $r = Invoke-Fill
    Assert-That ($r.Code -eq 0) "exit $($r.Code): $($r.Log)"
    Assert-That ($r.Log -match 'filled 169 empty cells') "log: $($r.Log)"
    Assert-That (Test-FileBom $r.Out) 'BOM lost'
    $t = Get-FileText $r.Out
    Assert-That ($t.Contains("`"Line one en`nLine two en`"")) 'Description lost its in-cell LF'
    Assert-That ($t.Contains('Title,4,Text,,Title-en,Title-ru,Title-uk')) 'Title row not filled in column order'
    Assert-That ($t.Contains('https://example.invalid/ru/asset')) 'an asset URL was touched'
    $m = Measure-CsvBreaks $t
    Assert-That ($m.RecordLf -eq 0 -and $m.RecordCrLf -eq 16 -and $m.CellCrLf -eq 0 -and $m.CellLf -eq 13 -and -not $m.EndsWithBreak) "breaks: $($m | ConvertTo-Json -Compress)"
}
Case 'a CRLF in-cell export is written with CRLF in-cell breaks' {
    $r = Invoke-Fill -Export @{ Cells = @{ 'en-us|Description' = "a`r`nb" } }
    Assert-That ($r.Code -eq 0) "exit $($r.Code): $($r.Log)"
    $t = Get-FileText $r.Out
    $m = Measure-CsvBreaks $t
    Assert-That ($t.Contains("`"Line one ru`r`nLine two ru`"")) 'deck LF was not turned into the export''s CRLF'
    Assert-That ($m.CellLf -eq 0 -and $m.CellCrLf -ge 13) "breaks: $($m | ConvertTo-Json -Compress)"
}
Case 'a BOM-less export stays BOM-less' {
    $r = Invoke-Fill -Export @{ Bom = $false }
    Assert-That ($r.Code -eq 0) "exit $($r.Code): $($r.Log)"
    Assert-That (-not (Test-FileBom $r.Out)) 'a BOM was added'
}
Case 'an LF record break export stays LF' {
    $r = Invoke-Fill -Export @{ Rec = "`n" }
    Assert-That ($r.Code -eq 0) "exit $($r.Code): $($r.Log)"
    $m = Measure-CsvBreaks (Get-FileText $r.Out)
    Assert-That ($m.RecordCrLf -eq 0 -and $m.RecordLf -eq 16) "breaks: $($m | ConvertTo-Json -Compress)"
}
Case 'a trailing newline in the export is kept' {
    $r = Invoke-Fill -Export @{ Trailing = $true }
    Assert-That ($r.Code -eq 0) "exit $($r.Code): $($r.Log)"
    Assert-That ((Get-FileText $r.Out).EndsWith("`r`n")) 'the trailing newline was lost'
}
Case 'a default run fills only empty cells' {
    $r = Invoke-Fill -Export @{ Cells = @{ 'en-us|Title' = 'Keep me'; 'ru|ReleaseNotes' = 'old notes' } }
    Assert-That ($r.Code -eq 0) "exit $($r.Code): $($r.Log)"
    $t = Get-FileText $r.Out
    Assert-That ($t.Contains('Keep me') -and $t.Contains('old notes') -and -not $t.Contains('Title-en,')) 'a filled cell was replaced without -Refresh'
}
Case 'every value exactly at its cap is accepted (ReleaseNotes after CRLF normalisation, 21 words)' {
    $r = Invoke-Fill -Decks @{
        'en|Description' = ('d' * 10000); 'en|ShortDescription' = ('s' * 1000); 'en|Feature1' = ('f' * 200)
        'en|ReleaseNotes' = (('x' * 1497) + "`ny")
    }
    Assert-That ($r.Code -eq 0) "exit $($r.Code): $($r.Log)"
}
Case '-Refresh replaces and lists every cell with the length it replaced' {
    $cells = @{}; foreach ($t in $Tags) { $cells["$t|ReleaseNotes"] = 'old notes' }
    $r = Invoke-Fill -Export @{ Cells = $cells } -Extra @{ Refresh = @('ReleaseNotes') }
    Assert-That ($r.Code -eq 0) "exit $($r.Code): $($r.Log)"
    Assert-That (([regex]::Matches($r.Log, 'replaced \S+ ReleaseNotes: 9 -> 8 characters')).Count -eq 13) "log: $($r.Log)"
    Assert-That ($r.Log -match 'replaced en-us ReleaseNotes: 9 -> 8 characters') 'no line for en-us'
    Assert-That ($r.Log -match 'refreshed 13') 'count missing'
    Assert-That (-not (Get-FileText $r.Out).Contains('old notes')) 'old notes remain'
}
Case '-FillNothing is a byte-identical round trip' {
    $r = Invoke-Fill -Extra @{ FillNothing = $true }
    Assert-That ($r.Code -eq 0 -and $r.Log -match 'byte-identical') "exit $($r.Code): $($r.Log)"
    Assert-That (Test-SameBytes $r.Before ([System.IO.File]::ReadAllBytes($r.Out))) 'bytes differ'
}
Case '-FillNothing needs no deck' {
    $r = Invoke-Fill -DropDeck 'fr' -Extra @{ FillNothing = $true }
    Assert-That ($r.Code -eq 0 -and $r.Log -match 'byte-identical') "exit $($r.Code): $($r.Log)"
}
Case '-FillNothing never touches the import folder' {
    $imp = Join-Path $Work 'imp-fillnothing'
    $r = Invoke-Fill -Extra @{ FillNothing = $true; ImportFolder = $imp }
    Assert-That ($r.Code -eq 0) "exit $($r.Code): $($r.Log)"
    Assert-That (-not (Test-Path $imp)) 'the import folder was created'
}

# -- negative: rule 7, a language and its column --
Case 'a shipped language with no column is refused' {
    $r = Invoke-Fill -Export @{ DropColumn = 'ur' } -PreOut
    Assert-Refused $r @('no column for ur')
}
Case 'a column with no deck is refused' {
    $r = Invoke-Fill -DropDeck 'fr' -PreOut
    Assert-Refused $r @('no deck for fr')
}

# -- negative: rule 9, the caps --
Case 'Description over 10000 is refused, naming language, field, length and deck' {
    $r = Invoke-Fill -Decks @{ 'de|Description' = ('d' * 10001) } -PreOut
    Assert-Refused $r @('cap: de Description: 10001 characters, cap 10000', 'de\.txt')
}
Case 'ShortDescription over 1000 is refused' {
    $r = Invoke-Fill -Decks @{ 'ru|ShortDescription' = ('s' * 1001) } -PreOut
    Assert-Refused $r @('ru ShortDescription: 1001 characters, cap 1000')
}
Case 'Feature over 200 is refused' {
    $r = Invoke-Fill -Decks @{ 'uk|Feature2' = ('f' * 201) } -PreOut
    Assert-Refused $r @('uk Feature2: 201 characters, cap 200')
}
Case 'ReleaseNotes is counted after CRLF normalisation' {
    $r = Invoke-Fill -Decks @{ 'it|ReleaseNotes' = (('x' * 1498) + "`ny") } -PreOut
    Assert-Refused $r @('it ReleaseNotes: 1501 characters, cap 1500')
}
Case 'the 22nd search-term word is refused' {
    $r = Invoke-Fill -Decks @{ 'en|SearchTerm7' = 'w7a w7b w7c w7d' } -PreOut
    Assert-Refused $r @('en-us SearchTerm1\.\.7: 22 words, cap 21')
}
Case 'a cell the export already held is checked as well' {
    $r = Invoke-Fill -Export @{ Cells = @{ 'en-us|Feature1' = ('f' * 201) } } -PreOut
    Assert-Refused $r @('en-us Feature1: 201 characters, cap 200')
}
Case 'every cap breach is listed in one run' {
    $r = Invoke-Fill -Decks @{ 'de|Description' = ('d' * 10001); 'ru|ShortDescription' = ('s' * 1001) } -PreOut
    Assert-Refused $r @('de Description: 10001', 'ru ShortDescription: 1001')
}
Case 'a refused in-place run leaves the export as it was' {
    $r = Invoke-Fill -Decks @{ 'de|Description' = ('d' * 10001) } -InPlace
    Assert-Refused $r @('cap: de Description') -InPlace
}

# -- negative: rule 8, the dialect --
Case 'mixed in-cell breaks are refused' {
    $r = Invoke-Fill -Export @{ Cells = @{ 'en-us|Description' = "a`r`nb"; 'ru|Description' = "a`nb" } } -PreOut
    Assert-Refused $r @('in-cell breaks mix CRLF')
}
Case 'a record break that is not the export''s is refused (the 2026-08-07 bare LF)' {
    $r = Invoke-Fill -Export @{ Mutate = { param($t) $i = $t.IndexOf("`r`n"); $t.Substring(0, $i) + "`n" + $t.Substring($i + 2) } } -PreOut
    Assert-Refused $r @('record breaks mix CRLF')
}
Case 'a lone carriage return is refused' {
    $r = Invoke-Fill -Export @{ Mutate = { param($t) $i = $t.IndexOf("`r`n"); $t.Substring(0, $i) + "`r" + $t.Substring($i + 2) } } -PreOut
    Assert-Refused $r @('lone carriage return')
}
Case 'an open last cell is refused' {
    $r = Invoke-Fill -Export @{ Mutate = { param($t) $t + ',"abc' } } -PreOut
    Assert-Refused $r @('left open')
}
Case 'a refused -FillNothing writes nothing as well' {
    $r = Invoke-Fill -Export @{ DropColumn = 'ur' } -PreOut -Extra @{ FillNothing = $true }
    Assert-Refused $r @('no column for ur')
}

# -- negative: rules 12 and 13, the replace and the round trip --
Case '-Refresh of an asset field is refused' {
    $r = Invoke-Fill -Extra @{ Refresh = @('DesktopScreenshot1') } -PreOut
    Assert-Refused $r @('-Refresh DesktopScreenshot1 would overwrite a listing-asset')
}
Case '-Refresh of a field the export lacks is refused' {
    $r = Invoke-Fill -Extra @{ Refresh = @('ReleaseNote') } -PreOut
    Assert-Refused $r @('-Refresh ReleaseNote names no field')
}
Case '-FillNothing over its own input is refused' {
    $d = Join-Path $Work "c$($script:total)"
    New-Item -ItemType Directory -Force -Path $d | Out-Null
    Write-Export (Join-Path $d 'export.csv')
    $before = [System.IO.File]::ReadAllBytes((Join-Path $d 'export.csv'))
    $r = Invoke-Builder @{ Csv = (Join-Path $d 'export.csv'); Out = (Join-Path $d 'export.csv'); FillNothing = $true; ImportFolder = '' }
    Assert-That ($r.Code -eq 1 -and $r.Log -match 'compares the file with itself') "exit $($r.Code): $($r.Log)"
    Assert-That (Test-SameBytes $before ([System.IO.File]::ReadAllBytes((Join-Path $d 'export.csv')))) 'the export was changed'
}

# -- the checks themselves, on crafted bytes (rung 2: the writer's refusals) --
$model = "Field,ID`r`nTitle,4`r`nDescription,`"a`nb`""
$dialect = Get-CsvDialect $model $true
function Get-ModelBytes([string]$text, [bool]$bom = $true) {
    $b = $Utf8.GetBytes($text)
    if ($bom) { return [byte[]](@(0xEF, 0xBB, 0xBF) + $b) }
    return $b
}
Case 'Test-CsvOutput accepts the export''s own dialect' {
    Assert-That ($dialect.Problems.Count -eq 0 -and $dialect.RecordBreak -eq "`r`n" -and $dialect.CellBreak -eq "`n" -and -not $dialect.Trailing) 'dialect of the model'
    Assert-That (@(Test-CsvOutput (Get-ModelBytes $model) $dialect).Count -eq 0) 'the model itself was refused'
}
Case 'Test-CsvOutput refuses a lost BOM and an added BOM' {
    Assert-That ((@(Test-CsvOutput (Get-ModelBytes $model $false) $dialect) -join ';') -match 'no BOM') 'lost BOM not refused'
    $noBom = Get-CsvDialect $model $false
    Assert-That ((@(Test-CsvOutput (Get-ModelBytes $model $true) $noBom) -join ';') -match 'starts with a BOM') 'added BOM not refused'
}
Case 'Test-CsvOutput refuses a bare LF record break' {
    Assert-That ((@(Test-CsvOutput (Get-ModelBytes "Field,ID`r`nTitle,4`nDescription,`"a`nb`"") $dialect) -join ';') -match 'bare LF') 'not refused'
}
Case 'Test-CsvOutput refuses an in-cell CRLF in an LF dialect' {
    Assert-That ((@(Test-CsvOutput (Get-ModelBytes "Field,ID`r`nTitle,4`r`nDescription,`"a`r`nb`"") $dialect) -join ';') -match 'in-cell break') 'not refused'
}
Case 'Test-CsvOutput refuses an open last cell' {
    Assert-That ((@(Test-CsvOutput (Get-ModelBytes "Field,ID`r`nTitle,4`r`nDescription,`"a`nb") $dialect) -join ';') -match 'left open') 'not refused'
}
Case 'Test-CsvOutput refuses a lone carriage return' {
    Assert-That ((@(Test-CsvOutput (Get-ModelBytes "Field,ID`rTitle,4`r`nDescription,`"a`nb`"") $dialect) -join ';') -match 'lone carriage return') 'not refused'
}
Case 'Test-CsvOutput refuses a trailing newline the export lacks' {
    Assert-That ((@(Test-CsvOutput (Get-ModelBytes ($model + "`r`n")) $dialect) -join ';') -match 'ends with a newline') 'not refused'
}
Case 'Get-CellLength counts code points and CRLF' {
    Assert-That ((Get-CellLength "a`nb") -eq 4 -and (Get-CellLength "a`r`nb") -eq 4) 'line breaks are two characters'
    Assert-That ((Get-CellLength ([char]::ConvertFromUtf32(0x1F600))) -eq 1) 'a surrogate pair is one character'
}

# -- the committed files --
$sample = Join-Path $Repo 'tools/store/listingData.sample.csv'
$export = Join-Path $Repo 'tools/store/listingData.csv'
Case 'the committed fixture round-trips byte for byte' {
    $r = Invoke-Builder @{ Csv = $sample; Out = (Join-Path $Work 'rt-sample.csv'); FillNothing = $true; ImportFolder = '' }
    Assert-That ($r.Code -eq 0 -and $r.Log -match 'round trip: byte-identical') "exit $($r.Code): $($r.Log)"
}
Case 'the committed export round-trips byte for byte' {
    $r = Invoke-Builder @{ Csv = $export; Out = (Join-Path $Work 'rt-export.csv'); FillNothing = $true; ImportFolder = '' }
    Assert-That ($r.Code -eq 0 -and $r.Log -match 'round trip: byte-identical') "exit $($r.Code): $($r.Log)"
}
Case 'the real decks fill the fixture inside every cap' {
    $r = Invoke-Builder @{ Csv = $sample; Out = (Join-Path $Work 'filled-sample.csv'); ImportFolder = '' }
    Assert-That ($r.Code -eq 0 -and $r.Log -match 'filled 312 empty cells') "exit $($r.Code): $($r.Log)"
}

if ($ownsWork) { Remove-Item -Recurse -Force $Work }
Write-Host ("cases={0} failed={1}" -f $script:total, $script:failed)
exit $(if ($script:failed) { 1 } else { 0 })
