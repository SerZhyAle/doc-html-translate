# Shared by scripts/doc-registry.ps1 (the check and the sitemap generator) and scripts/doc-query.ps1
# (the area / change-trigger query). Dot-source it: . "$PSScriptRoot/lib/docregistry.ps1"
#
# The registry is docs/DOCUMENT_REGISTRY.jsonl, one record per maintained document or document
# group, in record shape 1 (the stamp declares it as docRegistryShape; canon DOCUMENTATION_CONCEPT
# section 6). Shape 1 is the reference implementation's shape: id, title, category, audience, paths,
# published, indexable, url, languages, localized_urls, product_areas, update_triggers, generated,
# sitemap_exclude, notes. `generated` is the record's role in the source-of-truth chain - true is a
# render (regenerated, never hand-edited), false a source.
#
# Paths are globs over repo-relative paths with forward slashes: `*` stays inside one directory, `**/`
# spans any depth (including none), `?` is one character, `[..]` a character class. An explicit path
# keeps its place in the record's order; a glob expands sorted. That order is the sitemap's order.

$script:DocRegistryShape = 1
$script:DocRegistryPath = 'docs/DOCUMENT_REGISTRY.jsonl'
$script:DocRegistryExclusionsPath = 'configs/doc-registry-exclusions.jsonl'

function ConvertTo-GlobRegex([string]$Glob) {
    $sb = [System.Text.StringBuilder]::new('^')
    $i = 0
    while ($i -lt $Glob.Length) {
        $c = $Glob[$i]
        if ($Glob.Substring($i).StartsWith('**/')) { [void]$sb.Append('(?:.*/)?'); $i += 3; continue }
        if ($Glob.Substring($i).StartsWith('**')) { [void]$sb.Append('.*'); $i += 2; continue }
        switch ($c) {
            '*' { [void]$sb.Append('[^/]*') }
            '?' { [void]$sb.Append('[^/]') }
            '[' {
                $end = $Glob.IndexOf(']', $i + 1)
                if ($end -lt 0) { [void]$sb.Append('\[') }
                else { [void]$sb.Append($Glob.Substring($i, $end - $i + 1)); $i = $end }
            }
            default { [void]$sb.Append([regex]::Escape([string]$c)) }
        }
        $i++
    }
    [void]$sb.Append('$')
    return $sb.ToString()
}

function Test-IsGlob([string]$Path) { return $Path -match '[\*\?\[]' }

# Every file git would commit from this working tree - tracked plus untracked-not-ignored - that
# exists on disk (a tracked file deleted in the working tree is gone as far as a reader is
# concerned). Returns $null when git cannot answer, which callers turn into COULD NOT VERIFY.
function Get-RepoFiles([string]$Root) {
    $prev = $ErrorActionPreference
    $ErrorActionPreference = 'SilentlyContinue'
    try {
        $out = & git -C $Root -c core.quotepath=false ls-files -co --exclude-standard 2>$null
        if ($LASTEXITCODE -ne 0) { return $null }
    } finally { $ErrorActionPreference = $prev }
    $files = [System.Collections.Generic.List[string]]::new()
    foreach ($f in @($out)) {
        if (-not $f) { continue }
        if (Test-Path -LiteralPath (Join-Path $Root $f) -PathType Leaf) { $files.Add($f.Replace('\', '/')) }
    }
    return , $files
}

# Expands one record path against the file list: an explicit path matches itself, a glob matches
# every file its regex accepts, sorted ordinally so the order does not depend on the machine.
function Resolve-RegistryPath([string]$Pattern, $Files) {
    if (-not (Test-IsGlob $Pattern)) {
        if ($Files -contains $Pattern) { return , @($Pattern) }
        return , @()
    }
    $rx = ConvertTo-GlobRegex $Pattern
    $hits = [System.Collections.Generic.List[string]]::new()
    foreach ($f in $Files) { if ($f -cmatch $rx) { $hits.Add($f) } }
    $arr = $hits.ToArray()
    [Array]::Sort($arr, [System.StringComparer]::Ordinal)
    return , $arr
}

function Resolve-RecordFiles($Record, $Files) {
    $seen = [System.Collections.Generic.HashSet[string]]::new()
    $out = [System.Collections.Generic.List[string]]::new()
    foreach ($p in @($Record.paths)) {
        foreach ($f in (Resolve-RegistryPath ([string]$p) $Files)) {
            if ($seen.Add($f)) { $out.Add($f) }
        }
    }
    return , $out.ToArray()
}

# Reads a JSON Lines file into a list of @{ Line; Record } entries plus a list of parse errors.
function Read-JsonLines([string]$Path) {
    $entries = [System.Collections.Generic.List[object]]::new()
    $errors = [System.Collections.Generic.List[string]]::new()
    $n = 0
    foreach ($line in (Get-Content -LiteralPath $Path -Encoding utf8)) {
        $n++
        if (-not $line.Trim()) { continue }
        try { $entries.Add([pscustomobject]@{ Line = $n; Record = ($line | ConvertFrom-Json) }) }
        catch { $errors.Add("${Path}:${n}: invalid JSON - $($_.Exception.Message)") }
    }
    return [pscustomobject]@{ Entries = $entries; Errors = $errors }
}

function Get-FieldNames($Record) { return @($Record.PSObject.Properties.Name) }

# The site's public base URL, read from the one place that already states it: robots.txt names the
# sitemap by its absolute address. Returns $null when robots.txt has no Sitemap line.
function Get-SiteBase([string]$Root) {
    $robots = Join-Path $Root 'robots.txt'
    if (-not (Test-Path -LiteralPath $robots)) { return $null }
    foreach ($line in (Get-Content -LiteralPath $robots -Encoding utf8)) {
        if ($line -match '^\s*Sitemap:\s*(\S+)/sitemap\.xml\s*$') { return $Matches[1] }
    }
    return $null
}

# A page declares its own address: <link rel="canonical"> is this site's equivalent of a permalink
# (static HTML, no front matter). The hreflang links are the page's own language cluster.
function Get-PageAddresses([string]$FullPath) {
    $html = [System.IO.File]::ReadAllText($FullPath)
    $canonical = $null
    $m = [regex]::Match($html, '<link\s+rel="canonical"\s+href="([^"]+)"', 'IgnoreCase')
    if ($m.Success) { $canonical = $m.Groups[1].Value }
    $alts = [System.Collections.Generic.List[object]]::new()
    foreach ($a in [regex]::Matches($html, '<link\s+rel="alternate"\s+hreflang="([^"]+)"\s+href="([^"]+)"', 'IgnoreCase')) {
        $alts.Add([pscustomobject]@{ Lang = $a.Groups[1].Value; Href = $a.Groups[2].Value })
    }
    return [pscustomobject]@{ Canonical = $canonical; Alternates = $alts.ToArray(); Html = $html }
}

function Get-AddressFile([string]$Href) { return ($Href -split '[?#]', 2)[0] }

# The addresses one page file serves: its canonical first, then every hreflang alternate that is the
# same file with a query (index.html?l=ru), in the page's own order, without repeats.
function Get-PageOwnAddresses($Page) {
    $own = [System.Collections.Generic.List[string]]::new()
    if (-not $Page.Canonical) { return , $own.ToArray() }
    $own.Add($Page.Canonical)
    $file = Get-AddressFile $Page.Canonical
    foreach ($a in $Page.Alternates) {
        if ($a.Lang -eq 'x-default') { continue }
        if ((Get-AddressFile $a.Href) -ne $file) { continue }
        if (-not $own.Contains($a.Href)) { $own.Add($a.Href) }
    }
    return , $own.ToArray()
}

function Get-ExcludedPaths($Record) {
    return @(@($Record.sitemap_exclude) | Where-Object { $_ -and $_.path } | ForEach-Object { ([string]$_.path).Replace('\', '/') })
}

function Test-IsAnnounced($Record) { return [bool]($Record.published -and $Record.indexable) }

function ConvertTo-XmlText([string]$s) { return $s.Replace('&', '&amp;').Replace('<', '&lt;').Replace('>', '&gt;').Replace('"', '&quot;') }

# The sitemap, rendered from the records that announce pages, in record order and path order. Each
# address a page serves gets its own <url>, repeating the page's whole hreflang cluster including a
# self-reference - a version listed only as someone else's alternate is routinely left unindexed.
function Get-SitemapText([string]$Root, $Records, $Files) {
    $nl = "`n"
    $sb = [System.Text.StringBuilder]::new()
    [void]$sb.Append('<?xml version="1.0" encoding="UTF-8"?>' + $nl)
    [void]$sb.Append('<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9" xmlns:xhtml="http://www.w3.org/1999/xhtml">' + $nl)
    $announced = [System.Collections.Generic.HashSet[string]]::new()
    foreach ($r in $Records) {
        if (-not (Test-IsAnnounced $r)) { continue }
        $excluded = Get-ExcludedPaths $r
        foreach ($f in (Resolve-RecordFiles $r $Files)) {
            if ($excluded -contains $f) { continue }
            if ($f -notmatch '\.html?$') { continue }
            $page = Get-PageAddresses (Join-Path $Root $f)
            foreach ($loc in (Get-PageOwnAddresses $page)) {
                if (-not $announced.Add($loc)) { continue }
                [void]$sb.Append('  <url>' + $nl)
                [void]$sb.Append("    <loc>$(ConvertTo-XmlText $loc)</loc>" + $nl)
                foreach ($a in $page.Alternates) {
                    [void]$sb.Append("    <xhtml:link rel=`"alternate`" hreflang=`"$(ConvertTo-XmlText $a.Lang)`" href=`"$(ConvertTo-XmlText $a.Href)`"/>" + $nl)
                }
                [void]$sb.Append('  </url>' + $nl)
            }
        }
    }
    [void]$sb.Append('</urlset>' + $nl)
    return $sb.ToString()
}

function Read-TextNormalized([string]$Path) {
    if (-not (Test-Path -LiteralPath $Path)) { return $null }
    return ([System.IO.File]::ReadAllText($Path)).Replace("`r`n", "`n")
}

function Write-TextLF([string]$Path, [string]$Text) {
    [System.IO.File]::WriteAllText($Path, $Text.Replace("`r`n", "`n"), [System.Text.UTF8Encoding]::new($false))
}
