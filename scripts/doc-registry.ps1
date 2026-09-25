<#
.SYNOPSIS
  The documentation registry check (canon DOCUMENTATION_CONCEPT section 6): the registry against the
  tree, the tree against the registry, and the sitemap against the registry. Ends in one
  CHECK-VERDICT line.

.DESCRIPTION
  docs/DOCUMENT_REGISTRY.jsonl declares every maintained document, record shape 1 (the stamp's
  docRegistryShape). This check holds it true in both directions:

  Registry -> tree (items 1-3, 5)
    - every record is valid JSON with the shape-1 fields, a unique kebab-case id, and non-empty
      product_areas and update_triggers - the two query facets scripts/doc-query.ps1 answers;
    - every explicit path exists, every record resolves to at least one file, and no file has two
      home records;
    - a record that is not published says why in `notes`;
    - a record that announces pages (published + indexable) has a `url`, and it and every
      `localized_urls` value are addresses one of the record's pages actually serves;
    - the stamp declares docRegistryShape = 1 and docRegistryFile = this registry.

  Tree -> registry (item 4, reverse coverage)
    - every .md and .html file git would commit is claimed by a record or by a row of
      configs/doc-registry-exclusions.jsonl (each with a reason of four words or more, each still
      matching a file); a file claimed by both is a contradiction;
    - Pages serves the repository root, so every .html file is publishable: under an announcing
      record it declares its own address (<link rel="canonical">, this site's permalink), under any
      record it may instead sit in `sitemap_exclude` with a reason, and it is never left silent.

  The site half (items 6-7)
    - every announced page carries the SEO block of canon section 3 (title, description,
      canonical, Open Graph with an image, a Twitter card, JSON-LD, one h1, and an hreflang cluster
      where the record has more than one language);
    - sitemap.xml equals the sitemap rendered from the records. Item 8, notifying search engines,
      is a release step in DEV/RELEASE.md, once per release.

  Exit codes follow CHECK-VERDICT (scripts/lib/verdict.ps1): 0 PASS, 1 FAIL, 2 COULD NOT VERIFY
  (no git checkout, no registry).

.PARAMETER Generate
  Rewrite sitemap.xml from the registry first, then run the check.

.EXAMPLE
  ./scripts/doc-registry.ps1              # the check, as scripts/check.ps1 runs it
.EXAMPLE
  ./scripts/doc-registry.ps1 -Generate    # after adding or renaming a site page
#>
param(
    [switch]$Generate
)

$ErrorActionPreference = "Stop"
. "$PSScriptRoot/lib/verdict.ps1"
. "$PSScriptRoot/lib/docregistry.ps1"

$root = (& git rev-parse --show-toplevel 2>$null)
if ($LASTEXITCODE -ne 0 -or -not $root) {
    Write-Host "doc-registry: not inside a git checkout - the tree cannot be enumerated."
    Exit-Verdict 'doc-registry' 2 'no git checkout'
}
$root = $root.Trim()
Set-Location $root
Write-Subject 'doc-registry' "$DocRegistryPath against $(Get-TreeLabel)"

if (-not (Test-Path -LiteralPath $DocRegistryPath)) {
    Write-Host "doc-registry: $DocRegistryPath is missing."
    Exit-Verdict 'doc-registry' 2 'no registry'
}
$files = Get-RepoFiles $root
if ($null -eq $files) { Exit-Verdict 'doc-registry' 2 'git ls-files failed' }

$errors = [System.Collections.Generic.List[string]]::new()
function Add-Finding([string]$Message) { $script:errors.Add($Message) }

# ── the stamp declares the shape (item 5) ─────────────────────
try {
    $stamp = Get-Content -LiteralPath '.sza-canon.json' -Raw | ConvertFrom-Json
    if ($stamp.docRegistryShape -ne $DocRegistryShape) {
        Add-Finding ".sza-canon.json: docRegistryShape is '$($stamp.docRegistryShape)', this check reads shape $DocRegistryShape"
    }
    if ($stamp.docRegistryFile -ne $DocRegistryPath) {
        Add-Finding ".sza-canon.json: docRegistryFile is '$($stamp.docRegistryFile)', the registry is $DocRegistryPath"
    }
} catch {
    Add-Finding ".sza-canon.json: unreadable - $($_.Exception.Message)"
}

# ── read the registry ────────────────────────────────────────
$read = Read-JsonLines $DocRegistryPath
foreach ($e in $read.Errors) { Add-Finding $e }
$records = @($read.Entries | ForEach-Object Record)
$required = @('id', 'title', 'category', 'audience', 'paths', 'published', 'indexable', 'product_areas', 'update_triggers', 'generated')
$known = $required + @('url', 'languages', 'localized_urls', 'sitemap_exclude', 'notes')
$kebab = '^[a-z0-9]+(?:-[a-z0-9]+)*$'

$siteBase = Get-SiteBase $root
if (-not $siteBase) { Add-Finding "robots.txt: no 'Sitemap: <base>/sitemap.xml' line, so the site's base address is unknown" }

$ids = @{}
$owner = @{}          # file -> the id of the record that claims it
$announcedPages = 0
foreach ($entry in $read.Entries) {
    $r = $entry.Record
    $at = "${DocRegistryPath}:$($entry.Line) ($($r.id))"
    $fields = Get-FieldNames $r
    foreach ($f in $required) { if ($fields -notcontains $f) { Add-Finding "${at}: missing field '$f'" } }
    foreach ($f in $fields) { if ($known -notcontains $f) { Add-Finding "${at}: field '$f' is not in record shape $DocRegistryShape" } }
    if ([string]$r.id -notmatch $kebab) { Add-Finding "${at}: id is not kebab-case" }
    if ($ids.ContainsKey([string]$r.id)) { Add-Finding "${at}: id repeats line $($ids[[string]$r.id])" } else { $ids[[string]$r.id] = $entry.Line }
    foreach ($facet in 'product_areas', 'update_triggers') {
        $vals = @($r.$facet | Where-Object { $_ })
        if ($vals.Count -eq 0) { Add-Finding "${at}: $facet is empty - it is one of the two query facets" }
        foreach ($v in $vals) { if ([string]$v -notmatch $kebab) { Add-Finding "${at}: $facet value '$v' is not kebab-case" } }
    }
    if (-not $r.published) {
        $words = @(([string]$r.notes -split '\s+') | Where-Object { $_ })
        if ($words.Count -lt 4) { Add-Finding "${at}: not published, and notes do not say why (four words or more)" }
    }
    if ($r.indexable -and -not $r.published) { Add-Finding "${at}: indexable but not published" }

    # paths: registry -> tree, and one home per file
    $recordFiles = [System.Collections.Generic.List[string]]::new()
    foreach ($p in @($r.paths)) {
        $p = [string]$p
        if ($p -match '(^|/)\.\.(/|$)' -or $p.StartsWith('/') -or $p -match '^[A-Za-z]:') { Add-Finding "${at}: path escapes the repository: $p"; continue }
        $hits = Resolve-RegistryPath $p $files
        # An explicit path names a document, so it must exist. A glob declares a group's naming and
        # may match nothing yet (the RESEARCH_ scheme before its first note); the record as a whole
        # must still point at a file.
        if ($hits.Count -eq 0 -and -not (Test-IsGlob $p)) { Add-Finding "${at}: no file at '$p'"; continue }
        foreach ($h in $hits) { if (-not $recordFiles.Contains($h)) { $recordFiles.Add($h) } }
    }
    if ($recordFiles.Count -eq 0) { Add-Finding "${at}: no path of the record matches a file" }
    foreach ($f in $recordFiles) {
        if ($owner.ContainsKey($f) -and $owner[$f] -ne $r.id) { Add-Finding "${at}: $f is already claimed by record '$($owner[$f])' - one home per document" }
        else { $owner[$f] = $r.id }
    }

    # sitemap_exclude: each names a file of this record, with a reason
    $excluded = Get-ExcludedPaths $r
    foreach ($x in @($r.sitemap_exclude)) {
        if (-not $x) { continue }
        if (-not $x.path) { Add-Finding "${at}: sitemap_exclude entry without a path"; continue }
        $xp = ([string]$x.path).Replace('\', '/')
        if (-not $recordFiles.Contains($xp)) { Add-Finding "${at}: sitemap_exclude names $xp, which is not one of this record's files" }
        $words = @(([string]$x.reason -split '\s+') | Where-Object { $_ })
        if ($words.Count -lt 4) { Add-Finding "${at}: sitemap_exclude reason for $xp is under four words" }
    }

    # pages: Pages serves the repository root, so every .html here is publishable
    $served = [System.Collections.Generic.HashSet[string]]::new()
    foreach ($f in $recordFiles) {
        if ($f -notmatch '\.html?$' -or $excluded -contains $f) { continue }
        if (-not (Test-IsAnnounced $r)) {
            Add-Finding "${at}: $f is served by Pages but its record announces nothing - add it to sitemap_exclude with a reason, or publish the record"
            continue
        }
        $page = Get-PageAddresses (Join-Path $root $f)
        if (-not $page.Canonical) { Add-Finding "${at}: $f declares no address (<link rel=`"canonical`">) and is not in sitemap_exclude"; continue }
        if ($siteBase -and -not $page.Canonical.StartsWith($siteBase + '/')) { Add-Finding "${at}: $f canonical $($page.Canonical) is outside the site $siteBase" }
        foreach ($a in (Get-PageOwnAddresses $page)) { [void]$served.Add($a) }
        $announcedPages++

        # the SEO block every listed page carries (canon DOCUMENTATION_CONCEPT section 3)
        $h = $page.Html
        $need = [ordered]@{
            '<title>'                = '<title>[^<]+</title>'
            'meta description'       = '<meta\s+name="description"\s+content="[^"]+"'
            'canonical'              = '<link\s+rel="canonical"\s+href="[^"]+"'
            'og:title'               = '<meta\s+property="og:title"\s+content="[^"]+"'
            'og:description'         = '<meta\s+property="og:description"\s+content="[^"]+"'
            'og:image'               = '<meta\s+property="og:image"\s+content="https?://[^"]+"'
            'og:url'                 = '<meta\s+property="og:url"\s+content="[^"]+"'
            'twitter:card'           = '<meta\s+name="twitter:card"\s+content="summary_large_image"'
            'JSON-LD'                = '<script\s+type="application/ld\+json">'
        }
        foreach ($k in $need.Keys) {
            $n = [regex]::Matches($h, $need[$k], 'IgnoreCase').Count
            if ($n -ne 1) { Add-Finding "${at}: $f carries $n '$k' (the SEO block wants exactly one)" }
        }
        $h1 = [regex]::Matches($h, '<h1[\s>]', 'IgnoreCase').Count
        if ($h1 -ne 1) { Add-Finding "${at}: $f has $h1 <h1> (the SEO block wants exactly one)" }
        if (@($r.languages).Count -gt 1 -and $page.Alternates.Count -lt 2) { Add-Finding "${at}: $f has no hreflang cluster, and its record declares $(@($r.languages).Count) languages" }
    }

    # a record that announces pages has an address, and every address it names is served
    if (Test-IsAnnounced $r) {
        if (-not $r.url) { Add-Finding "${at}: announces pages but declares no url" }
        $declared = @()
        if ($r.url) { $declared += [string]$r.url }
        if ($r.localized_urls) { $declared += @($r.localized_urls.PSObject.Properties | ForEach-Object { [string]$_.Value }) }
        foreach ($d in $declared) {
            if ($siteBase -and -not $served.Contains($siteBase + $d)) { Add-Finding "${at}: declared address $d is served by none of the record's pages" }
        }
        if ($r.localized_urls) {
            foreach ($lang in $r.localized_urls.PSObject.Properties.Name) {
                if (@($r.languages) -notcontains $lang) { Add-Finding "${at}: localized_urls names '$lang', which languages does not list" }
            }
        }
    }
}

# ── tree -> registry (item 4) ────────────────────────────────
$exclusions = @()
if (Test-Path -LiteralPath $DocRegistryExclusionsPath) {
    $xr = Read-JsonLines $DocRegistryExclusionsPath
    foreach ($e in $xr.Errors) { Add-Finding $e }
    $exclusions = @($xr.Entries)
}
$excludedBy = @{}
foreach ($x in $exclusions) {
    $at = "${DocRegistryExclusionsPath}:$($x.Line)"
    $p = [string]$x.Record.path
    $words = @(([string]$x.Record.reason -split '\s+') | Where-Object { $_ })
    if (-not $p) { Add-Finding "${at}: no path"; continue }
    if ($words.Count -lt 4) { Add-Finding "${at}: the reason for $p is under four words" }
    $hits = Resolve-RegistryPath $p $files
    if ($hits.Count -eq 0) { Add-Finding "${at}: $p matches no file - drop the row" }
    foreach ($h in $hits) { $excludedBy[$h] = $p }
}
$covered = 0
foreach ($f in $files) {
    if ($f -notmatch '\.(md|html?)$') { continue }
    $inRecord = $owner.ContainsKey($f)
    $inExclusion = $excludedBy.ContainsKey($f)
    if ($inRecord -and $inExclusion) { Add-Finding "$f is claimed by record '$($owner[$f])' and excluded by '$($excludedBy[$f])' - pick one"; continue }
    if (-not $inRecord -and -not $inExclusion) {
        Add-Finding "$f is in no record - register it in $DocRegistryPath (a new record or a glob on an existing one), or exclude it in $DocRegistryExclusionsPath with a reason"
        continue
    }
    $covered++
}

# ── the sitemap is generated (item 7) ────────────────────────
if ($read.Errors.Count -eq 0 -and $siteBase) {
    $rendered = Get-SitemapText $root $records $files
    if ($Generate) {
        Write-TextLF (Join-Path $root 'sitemap.xml') $rendered
        Write-Host "doc-registry: sitemap.xml written ($(([regex]::Matches($rendered, '<loc>')).Count) url(s))"
    }
    $current = Read-TextNormalized (Join-Path $root 'sitemap.xml')
    if ($current -ne $rendered) {
        Add-Finding "sitemap.xml differs from the sitemap the registry renders - run scripts/doc-registry.ps1 -Generate (never edit it by hand)"
    }
}

foreach ($m in $errors) { Write-Host "  $m" -ForegroundColor Red }
$summary = "$($records.Count) record(s), $covered document file(s) covered, $announcedPages page(s) announced"
if ($errors.Count -gt 0) { Exit-Verdict 'doc-registry' 1 "$($errors.Count): $summary" }
Exit-Verdict 'doc-registry' 0 $summary
