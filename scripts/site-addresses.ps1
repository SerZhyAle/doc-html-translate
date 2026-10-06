<#
.SYNOPSIS
  The held-addresses check (SITE-STRUCTURE rule 8): every address of this site that a surface outside
  the site holds is listed in configs/site-held-addresses.jsonl, and every listed address still answers.
  Ends in one CHECK-VERDICT line.

.DESCRIPTION
  The list is the one place that says which program, extension message, README, listing source or
  manifest holds which page of the site (docs/SITE_ADDRESSES.md is the policy). This check holds it
  true in both directions:

  List -> tree
    - every record is valid JSON of one of two kinds: `held` (holder, surface, addresses) or `ignore`
      (a glob and a reason of four words or more); no holder is listed twice;
    - every holder exists, still carries each address it is listed with, and is not also ignored;
    - every address starts with the site base (the one robots.txt names), resolves to a file the
      repository serves (a directory address to its index.html), carries only an `l=en|ru|uk` query on
      an html page, and - when it names a section - an id that exists in that page;
    - every ignore glob still matches a file.

  Tree -> list
    - every file git would commit that is not ignored is read, and every address of this site in it
      must be in that file's record. A holder with no record, or an address missing from its record,
      is a finding: add the entry (-Suggest prints the records) or remove the address.

  Why a moved page is caught: a page that moves or is retired leaves a forwarder at the old address
  (docs/SITE_ADDRESSES.md), and a forwarder is a file; an entry whose file is gone fails here until the
  forwarder exists or the holder stops using the address.

  Exit codes follow CHECK-VERDICT (scripts/lib/verdict.ps1): 0 PASS, 1 FAIL, 2 COULD NOT VERIFY
  (no git checkout, no list, no site base).

.PARAMETER Suggest
  After the check, print a ready-to-paste `held` record for every holder that is missing or incomplete.

.EXAMPLE
  ./scripts/site-addresses.ps1            # the check, as scripts/check.ps1 runs it
.EXAMPLE
  ./scripts/site-addresses.ps1 -Suggest   # also print the records a new holder needs
#>
param(
    [switch]$Suggest
)

$ErrorActionPreference = "Stop"
. "$PSScriptRoot/lib/verdict.ps1"
. "$PSScriptRoot/lib/docregistry.ps1"

$ListPath = 'configs/site-held-addresses.jsonl'
$Surfaces = @('program', 'extension', 'readme', 'store-listing', 'manifest', 'script', 'test', 'developer-doc')
$Languages = @('en', 'ru', 'uk')
$BinaryExt = '\.(png|jpe?g|webp|gif|ico|exe|dll|zip|gz|pdf|svg|woff2?|ttf|msix|traineddata|mp[34]|webm)$'

$root = (& git rev-parse --show-toplevel 2>$null)
if ($LASTEXITCODE -ne 0 -or -not $root) {
    Write-Host "site-addresses: not inside a git checkout - the tree cannot be enumerated."
    Exit-Verdict 'site-addresses' 2 'no git checkout'
}
$root = $root.Trim()
Set-Location $root
Write-Subject 'site-addresses' "$ListPath against $(Get-TreeLabel)"

if (-not (Test-Path -LiteralPath $ListPath)) {
    Write-Host "site-addresses: $ListPath is missing."
    Exit-Verdict 'site-addresses' 2 'no list'
}
$base = Get-SiteBase $root
if (-not $base) {
    Write-Host "site-addresses: robots.txt names no Sitemap address, so the site base is unknown."
    Exit-Verdict 'site-addresses' 2 'no site base'
}
$files = Get-RepoFiles $root
if ($null -eq $files) { Exit-Verdict 'site-addresses' 2 'git ls-files failed' }
$fileSet = [System.Collections.Generic.HashSet[string]]::new([string[]]@($files), [System.StringComparer]::Ordinal)

$errors = [System.Collections.Generic.List[string]]::new()
function Add-Finding([string]$Message) { $script:errors.Add($Message) }

# ── the list ──────────────────────────────────────────────────
$parsed = Read-JsonLines $ListPath
foreach ($e in $parsed.Errors) { Add-Finding $e }
$held = @{}
$ignores = [System.Collections.Generic.List[object]]::new()
foreach ($entry in $parsed.Entries) {
    $r = $entry.Record
    $where = "${ListPath}:$($entry.Line)"
    switch ($r.kind) {
        'held' {
            $extra = @(Get-FieldNames $r | Where-Object { $_ -notin 'kind', 'holder', 'surface', 'addresses' })
            if ($extra) { Add-Finding "$where`: unknown field(s) $($extra -join ', ')" }
            if (-not $r.holder) { Add-Finding "$where`: a held record needs a holder"; continue }
            if ($r.surface -notin $Surfaces) { Add-Finding "$where`: surface '$($r.surface)' is not one of $($Surfaces -join ', ')" }
            if (-not @($r.addresses).Count) { Add-Finding "$where`: $($r.holder) lists no address" }
            if ($held.ContainsKey($r.holder)) { Add-Finding "$where`: $($r.holder) is listed twice" }
            else { $held[$r.holder] = $r }
            if (@($r.addresses).Count -ne @($r.addresses | Select-Object -Unique).Count) { Add-Finding "$where`: $($r.holder) lists an address twice" }
        }
        'ignore' {
            $extra = @(Get-FieldNames $r | Where-Object { $_ -notin 'kind', 'path', 'reason' })
            if ($extra) { Add-Finding "$where`: unknown field(s) $($extra -join ', ')" }
            if (-not $r.path) { Add-Finding "$where`: an ignore record needs a path"; continue }
            if (@(([string]$r.reason).Split(' ', [System.StringSplitOptions]::RemoveEmptyEntries)).Count -lt 4) {
                Add-Finding "$where`: the ignore of $($r.path) needs a reason of four words or more"
            }
            $ignores.Add([pscustomobject]@{ Path = $r.path; Regex = (ConvertTo-GlobRegex $r.path); Line = $entry.Line })
        }
        default { Add-Finding "$where`: kind '$($r.kind)' is not held or ignore" }
    }
}

function Test-Ignored([string]$File) {
    foreach ($i in $ignores) { if ($File -cmatch $i.Regex) { return $i } }
    return $null
}

foreach ($i in $ignores) {
    if (-not @($files | Where-Object { $_ -cmatch $i.Regex }).Count) {
        Add-Finding "${ListPath}:$($i.Line): the ignore '$($i.Path)' matches no file - remove it"
    }
}

# ── one address against the page set ──────────────────────────
# Returns a problem text, or $null when the address answers with the page it meant.
function Resolve-HeldAddress([string]$Address) {
    $baseNoSlash = $base.TrimEnd('/')
    if ($Address -ne $baseNoSlash -and -not $Address.StartsWith("$baseNoSlash/")) {
        return "is not under the site base $baseNoSlash"
    }
    $rest = $Address.Substring($baseNoSlash.Length)
    $fragment = $null
    $query = $null
    if ($rest.Contains('#')) { $i = $rest.IndexOf('#'); $fragment = $rest.Substring($i + 1); $rest = $rest.Substring(0, $i) }
    if ($rest.Contains('?')) { $i = $rest.IndexOf('?'); $query = $rest.Substring($i + 1); $rest = $rest.Substring(0, $i) }
    $path = $rest.TrimStart('/')
    if ($path -eq '' -or $path.EndsWith('/')) { $path += 'index.html' }
    if (-not $fileSet.Contains($path)) { return "answers with no file ($path is not in the tree) - restore the page or leave a forwarder there" }
    if ($null -ne $query) {
        if ($path -notmatch '\.html$') { return "carries a query on a file that is not a page" }
        if ($query -notmatch '^l=([a-z]+)$' -or $Matches[1] -notin $Languages) { return "carries the query '$query'; only l=$($Languages -join '|') is part of the address scheme" }
    }
    if ($fragment) {
        if ($path -notmatch '\.html$') { return "names the section '$fragment' of a file that is not a page" }
        $text = [System.IO.File]::ReadAllText((Join-Path $root $path))
        $esc = [regex]::Escape($fragment)
        if ($text -notmatch "(?i)\b(?:id|name)\s*=\s*[""']$esc[""']") { return "names the section '$fragment', which $path no longer has - a section anchor an outside surface links is never renamed" }
    }
    return $null
}

# ── list -> tree ──────────────────────────────────────────────
$resolved = 0
foreach ($h in $held.Values) {
    if (-not $fileSet.Contains($h.holder)) { Add-Finding "$($h.holder): listed as a holder but the file does not exist"; continue }
    if (Test-Ignored $h.holder) { Add-Finding "$($h.holder): is both listed as a holder and ignored" }
    $text = [System.IO.File]::ReadAllText((Join-Path $root $h.holder))
    foreach ($a in @($h.addresses)) {
        if (-not $text.Contains($a)) { Add-Finding "$($h.holder): is listed with $a but no longer holds it - remove the entry" }
        $problem = Resolve-HeldAddress $a
        $resolved++
        if ($problem) { Add-Finding "$($h.holder): $a $problem" }
    }
}

# ── tree -> list ──────────────────────────────────────────────
$addressRx = [regex]::new('(?i)(?:https?://)?' + [regex]::Escape(([uri]$base).Host) + [regex]::Escape(([uri]$base).AbsolutePath.TrimEnd('/')) + '(?:[/?#][^\s"''<>)\]\\`,;|]*)?')
$suggested = @{}
$scanned = 0
foreach ($f in $files) {
    if ($f -match $BinaryExt) { continue }
    if (Test-Ignored $f) { continue }
    $text = $null
    try { $text = [System.IO.File]::ReadAllText((Join-Path $root $f)) } catch { continue }
    if ($text.Contains([char]0)) { continue }
    $scanned++
    $found = [System.Collections.Generic.List[string]]::new()
    foreach ($m in $addressRx.Matches($text)) {
        $v = $m.Value.TrimEnd('.', ':')
        if (-not $found.Contains($v)) { $found.Add($v) }
    }
    if (-not $found.Count) { continue }
    if (-not $held.ContainsKey($f)) {
        Add-Finding "${f}: holds $($found.Count) address(es) of the site and is not in $ListPath ($($found -join ' '))"
        $suggested[$f] = $found
        continue
    }
    $listed = @($held[$f].addresses)
    $missing = @($found | Where-Object { $_ -notin $listed })
    if ($missing) {
        Add-Finding "${f}: holds $($missing -join ' ') which $ListPath does not list for it"
        $suggested[$f] = $found
    }
}

if ($Suggest -and $suggested.Count) {
    Write-Host ""
    Write-Host "records to paste into ${ListPath} (set the surface; replace a holder's existing record):"
    foreach ($f in ($suggested.Keys | Sort-Object)) {
        [ordered]@{ kind = 'held'; holder = $f; surface = '?'; addresses = @($suggested[$f]) } | ConvertTo-Json -Compress | Write-Host
    }
}

Write-Host ""
Write-Host "site-addresses: $($held.Count) holder(s), $resolved listed address(es) resolved against the page set, $scanned file(s) scanned for unlisted ones."
if ($errors.Count) {
    foreach ($m in $errors) { Write-Host "  $m" -ForegroundColor Red }
    Exit-Verdict 'site-addresses' 1 "$($errors.Count) finding(s)"
}
Exit-Verdict 'site-addresses' 0 ''
