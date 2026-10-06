<#
.SYNOPSIS
  The release-notes check (SITE-STRUCTURE rules 2 and 14): docs/release-notes.json against the release
  tags, and release-notes.html against docs/release-notes.json. Ends in one CHECK-VERDICT line.

.DESCRIPTION
  The notes page is assembled at the release from one source, never edited per ticket. The source is
  docs/release-notes.json: one record per app release tag, newest first, each with the same list of
  bullets in en, ru and uk. The marked block of release-notes.html is rendered from it.

  The source
    - parses, declares schema 1, and holds at least one record;
    - every record has a version of the shape YY.MMDD.HHmm, a date that agrees with the version's
      year, month and day, and bullets in en, ru and uk - the same number in each, none empty, none
      with a long dash, an ellipsis or a three-dot run (the house text style);
    - versions are strictly descending, so a record can be neither duplicated nor misplaced.

  The source against the tags (rule 14: the notes are drawn from the releases, none left out)
    - every `v<version>` tag of the app (the extension's ext-* tags are not releases of the app) has a
      record, and every record has its tag - except the newest record, which may be the release in
      preparation and not yet tagged (the release flow writes the notes before it pushes the tag).

  The page against the source
    - release-notes.html carries exactly one marked block and it equals the render, byte for byte, so a
      hand-edited block, or a record added without rendering, fails here.

  Exit codes follow CHECK-VERDICT (scripts/lib/verdict.ps1): 0 PASS, 1 FAIL, 2 COULD NOT VERIFY
  (no git checkout, no source, no page, a clone with no release tags at all).

.PARAMETER Render
  Rewrite the marked block of release-notes.html from docs/release-notes.json first, then run the check.

.EXAMPLE
  ./scripts/release-notes.ps1            # the check, as scripts/check.ps1 runs it
.EXAMPLE
  ./scripts/release-notes.ps1 -Render    # after adding a record to docs/release-notes.json
#>
param(
    [switch]$Render
)

$ErrorActionPreference = "Stop"
. "$PSScriptRoot/lib/verdict.ps1"

$SourcePath = 'docs/release-notes.json'
$PagePath = 'release-notes.html'
$Block = 'releases'
$ReleasesUrl = 'https://github.com/SerZhyAle/doc-html-translate/releases/tag/'
# The page keys Ukrainian as `ua` (the site's language value space); the source and the sitemap say `uk`.
$Langs = @(@{ source = 'ru'; page = 'ru' }, @{ source = 'en'; page = 'en' }, @{ source = 'uk'; page = 'ua' })

$root = (& git rev-parse --show-toplevel 2>$null)
if ($LASTEXITCODE -ne 0 -or -not $root) {
    Write-Host "release-notes: not inside a git checkout - the tags cannot be listed."
    Exit-Verdict 'release-notes' 2 'no git checkout'
}
$root = $root.Trim()
Set-Location $root
Write-Subject 'release-notes' "$SourcePath and $PagePath against the release tags of $(Get-TreeLabel)"

foreach ($f in @($SourcePath, $PagePath)) {
    if (-not (Test-Path -LiteralPath $f)) {
        Write-Host "release-notes: $f is missing."
        Exit-Verdict 'release-notes' 2 "no $f"
    }
}

$errors = [System.Collections.Generic.List[string]]::new()
function Add-Finding([string]$Message) { $script:errors.Add($Message) }

# ── the source ────────────────────────────────────────────────
try { $data = [IO.File]::ReadAllText((Resolve-Path -LiteralPath $SourcePath), [Text.Encoding]::UTF8) | ConvertFrom-Json }
catch {
    Write-Host "release-notes: $SourcePath is not valid JSON: $($_.Exception.Message)" -ForegroundColor Red
    Exit-Verdict 'release-notes' 1 'source is not JSON'
}
if ($data.schema -ne 1) { Add-Finding "${SourcePath}: schema must be 1" }
foreach ($l in $Langs) { if ([string]::IsNullOrWhiteSpace([string]$data.link.($l.source))) { Add-Finding "${SourcePath}: link.$($l.source) (the label of the GitHub release link) is missing" } }
$records = @($data.releases)
if (-not $records.Count) { Add-Finding "${SourcePath}: no release records" }

$versionRx = '^(\d{2})\.(\d{2})(\d{2})\.(\d{4})$'
$dateRx = '^(\d{4})-(\d{2})-(\d{2})$'
$styleRx = "$([char]0x2014)|$([char]0x2026)|\.\.\."
$previous = $null
foreach ($r in $records) {
    $at = "${SourcePath} $($r.version)"
    $vm = [regex]::Match([string]$r.version, $versionRx)
    if (-not $vm.Success) { Add-Finding "${at}: the version is not of the shape YY.MMDD.HHmm"; continue }
    $dm = [regex]::Match([string]$r.date, $dateRx)
    if (-not $dm.Success) { Add-Finding "${at}: the date is not YYYY-MM-DD" }
    elseif ($dm.Groups[1].Value.Substring(2) -ne $vm.Groups[1].Value -or $dm.Groups[2].Value -ne $vm.Groups[2].Value -or $dm.Groups[3].Value -ne $vm.Groups[3].Value) {
        Add-Finding "${at}: the date $($r.date) disagrees with the version's year, month and day"
    }
    if ($null -ne $previous -and [string]::CompareOrdinal($previous, [string]$r.version) -le 0) {
        Add-Finding "${at}: not strictly descending (after $previous)"
    }
    $previous = [string]$r.version

    $counts = @{}
    foreach ($l in $Langs) {
        $bullets = @($r.notes.($l.source))
        $counts[$l.source] = $bullets.Count
        if (-not $bullets.Count) { Add-Finding "${at}: no $($l.source) notes"; continue }
        foreach ($b in $bullets) {
            if ([string]::IsNullOrWhiteSpace([string]$b)) { Add-Finding "${at}: an empty $($l.source) bullet"; continue }
            if ($b -match $styleRx) { Add-Finding "${at}: a $($l.source) bullet carries a long dash, an ellipsis or three dots (house style: plain hyphen, '..'): $($b.Substring(0, [Math]::Min(60, $b.Length)))" }
        }
    }
    if (($counts.Values | Sort-Object -Unique).Count -gt 1) {
        Add-Finding "${at}: the languages are not mirrored (en $($counts['en']), ru $($counts['ru']), uk $($counts['uk']) bullets)"
    }
}

# ── the tags ──────────────────────────────────────────────────
$tagLines = @(& git tag --list 'v[0-9]*' 2>$null)
if ($LASTEXITCODE -ne 0) { Exit-Verdict 'release-notes' 2 'git tag failed' }
$tags = @($tagLines | ForEach-Object { "$_".Trim() } | Where-Object { $_ -match '^v\d{2}\.\d{4}\.\d{4}$' })
if (-not $tags.Count) {
    Write-Host "release-notes: this clone has no app release tags (v<version>); the notes cannot be held against them. Fetch the tags and run again."
    Exit-Verdict 'release-notes' 2 'no release tags in this clone'
}
$recordVersions = @($records | ForEach-Object { [string]$_.version })
foreach ($t in $tags) {
    if ($t.Substring(1) -notin $recordVersions) { Add-Finding "tag ${t} has no record in $SourcePath (a release must have notes)" }
}
foreach ($i in 0..([Math]::Max($records.Count, 1) - 1)) {
    if ($i -ge $records.Count) { break }
    $v = [string]$records[$i].version
    if ("v$v" -in $tags) { continue }
    if ($i -eq 0) { Write-Host "  $v is the newest record and has no tag yet: read as the release in preparation." -ForegroundColor Yellow }
    else { Add-Finding "$SourcePath $v has no tag v$v and is not the newest record" }
}

# ── the render ────────────────────────────────────────────────
function ConvertTo-HtmlText([string]$s) { return $s.Replace('&', '&amp;').Replace('<', '&lt;').Replace('>', '&gt;').Replace('"', '&quot;') }

function Get-Render {
    $sb = [System.Text.StringBuilder]::new()
    $null = $sb.Append("    <!-- release-notes:begin $Block (rendered from $SourcePath by scripts/release-notes.ps1 -Render; edit the records there) -->`n")
    foreach ($r in $records) {
        $v = [string]$r.version
        $id = 'v' + $v.Replace('.', '-')
        $null = $sb.Append("    <section class=`"release`" id=`"$id`">`n")
        $null = $sb.Append("      <h2>$v <time datetime=`"$($r.date)`">$($r.date)</time></h2>`n")
        foreach ($l in $Langs) {
            $null = $sb.Append("      <ul data-l=`"$($l.page)`">`n")
            foreach ($b in @($r.notes.($l.source))) { $null = $sb.Append("        <li>$(ConvertTo-HtmlText ([string]$b))</li>`n") }
            $null = $sb.Append("      </ul>`n")
        }
        $link = "$ReleasesUrl" + "v$v"
        $null = $sb.Append("      <p class=`"meta`"><a href=`"$link`">" + (($Langs | ForEach-Object { "<span data-l=`"$($_.page)`">$(ConvertTo-HtmlText ([string]$data.link.($_.source)))</span>" }) -join '') + "</a></p>`n")
        $null = $sb.Append("    </section>`n")
    }
    $null = $sb.Append("    <!-- release-notes:end $Block -->`n")
    return $sb.ToString()
}

$pageText = [IO.File]::ReadAllText((Resolve-Path -LiteralPath $PagePath), [Text.Encoding]::UTF8)
$begins = [regex]::Matches($pageText, "(?m)^[ \t]*<!-- release-notes:begin $Block\b[^\n]*-->[ \t]*\n")
$ends = [regex]::Matches($pageText, "(?m)^[ \t]*<!-- release-notes:end $Block -->[ \t]*\n")
if ($begins.Count -ne 1 -or $ends.Count -ne 1 -or $ends[0].Index -lt $begins[0].Index) {
    Add-Finding "${PagePath}: must carry exactly one '<!-- release-notes:begin $Block -->' and one matching end marker, in that order"
} elseif ($records.Count) {
    $start = $begins[0].Index
    $stop = $ends[0].Index + $ends[0].Length
    $expected = Get-Render
    $current = $pageText.Substring($start, $stop - $start)
    if ($current -ne $expected) {
        if ($Render) {
            $new = $pageText.Substring(0, $start) + $expected + $pageText.Substring($stop)
            [IO.File]::WriteAllText((Resolve-Path -LiteralPath $PagePath), $new, [Text.UTF8Encoding]::new($false))
            Write-Host "release-notes: rewrote the $Block block of $PagePath from $SourcePath." -ForegroundColor Cyan
        } else {
            Add-Finding "${PagePath}: the '$Block' block is not the render of $SourcePath (run scripts/release-notes.ps1 -Render; never edit the block by hand)"
        }
    }
}

Write-Host ""
Write-Host "release-notes: $($records.Count) record(s), $($tags.Count) app release tag(s)."
if ($errors.Count) {
    foreach ($m in $errors) { Write-Host "  $m" -ForegroundColor Red }
    Exit-Verdict 'release-notes' 1 "$($errors.Count) finding(s)"
}
Exit-Verdict 'release-notes' 0 ''
