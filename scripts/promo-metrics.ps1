<#
.SYNOPSIS
  Reads the free-promotion campaign counters (ticket 111) and writes one dated Markdown checkpoint.
  Read-only; ends in one CHECK-VERDICT line.

.DESCRIPTION
  Sources, all public reads (plain GETs, plus one read-only GraphQL query that gh can only send as POST):
    GitHub     repository, releases with per-asset download_count, traffic (needs push access),
               usesCustomOpenGraphImage, and the winget-pkgs folder of this package
    Extensions the Chrome Web Store detail page and the Edge Add-ons product-details endpoint
    Store      the public Microsoft Store product endpoint

  A value the script cannot read is the word `unknown` - never an estimate or a range. The consoles
  that only the owner can open (Search Console, Partner Center acquisitions, indexed pages) are always
  `unknown` here. Nothing is sent anywhere: no analytics, no tracking parameter, no third-party service.

  The checkpoint is <OutDir>/<Date>.md: the section 9 table of the ticket in its column order, then a
  detail table. Downloads exclude checksum, notices and third-party files. A same-day rerun replaces the file.

  Exit codes follow CHECK-VERDICT (scripts/lib/verdict.ps1): 0 PASS when GitHub was read (sources that
  could not be read are named in the detail and are `unknown` in the file), 2 COULD NOT VERIFY when gh is
  missing, GitHub could not be read, or the script failed unexpectedly (no file is written then), 1 FAIL on
  a malformed -Date.

.PARAMETER OutDir
  Folder of the checkpoint file; relative paths resolve against the repository root.

.PARAMETER Date
  Label and file name of the checkpoint (yyyy-MM-dd), default today. The numbers are always read now.

.PARAMETER Print
  Print the checkpoint instead of writing it.

.EXAMPLE
  ./scripts/promo-metrics.ps1
.EXAMPLE
  ./scripts/promo-metrics.ps1 -Print
#>
param(
    [string]$OutDir = 'DEV/plan/111_2026-10-09_free-promotion-campaign/metrics',
    [string]$Date = (Get-Date -Format 'yyyy-MM-dd'),
    [switch]$Print
)

$ErrorActionPreference = 'Stop'
. "$PSScriptRoot/lib/verdict.ps1"

$Check = 'promo-metrics'
$Owner = 'SerZhyAle'
$Repo = 'doc-html-translate'
$Unknown = 'unknown'
$ChromeUrl = 'https://chromewebstore.google.com/detail/nmcckamdocainafmmompkbmelkpbnmic?hl=en'
$EdgeUrl = 'https://microsoftedge.microsoft.com/addons/getproductdetailsbycrxid/anokfnnfiboaccbbpfkdaphejnkkhajh'
$StoreUrl = 'https://storeedgefd.dsx.mp.microsoft.com/v9.0/products/9PMHSWQPR6V1?market=US&locale=en-us&deviceFamily=Windows.Desktop'
$WingetApi = 'repos/microsoft/winget-pkgs/contents/manifests/s/SerZhyAle/DocHtmlTranslate'
$GraphQlQuery = 'query($o:String!,$n:String!){repository(owner:$o,name:$n){usesCustomOpenGraphImage}}'
# Checksums, the notices file and the third-party list are not downloads of the product.
$ExcludedAsset = 'checksum|sha256|notices|third-party'

$read = [System.Collections.Generic.List[string]]::new()
$unread = [System.Collections.Generic.List[string]]::new()
$script:LastError = ''

function Register-Source([string]$Source, $Value, [string]$Reason = $script:LastError) {
    if ($null -ne $Value) { $script:read.Add($Source) } else { $script:unread.Add("$Source ($Reason)") }
}

# The unary comma keeps an empty JSON array an empty array instead of letting the pipeline unroll it
# into $null, which the callers read as "could not be read".
function ConvertFrom-JsonOrNull([string]$Json) {
    try { $parsed = $Json | ConvertFrom-Json -NoEnumerate }
    catch [System.ArgumentException] { $script:LastError = 'response is not JSON'; $parsed = $null }
    return , $parsed
}

function Invoke-GhApi([string[]]$ApiArgs) {
    $out = & gh api @ApiArgs 2>$null
    if ($LASTEXITCODE -ne 0 -or -not $out) { $script:LastError = "gh api exit $LASTEXITCODE"; return $null }
    $parsed = ConvertFrom-JsonOrNull ($out -join "`n")
    return , $parsed
}

function Get-WebText([string]$Url, [hashtable]$Headers = @{}) {
    try { return (Invoke-WebRequest -Uri $Url -Headers $Headers -UseBasicParsing -TimeoutSec 30).Content }
    catch [System.Net.Http.HttpRequestException], [System.OperationCanceledException] {
        $script:LastError = $_.Exception.Message
        return $null
    }
}

function ConvertTo-VisibleText([string]$Html) {
    $start = $Html.IndexOf('<body')
    if ($start -lt 0) { return '' }
    $text = [regex]::Replace($Html.Substring($start), '<(script|style)\b.*?</\1>', ' ', 'Singleline,IgnoreCase')
    $text = [regex]::Replace($text, '<[^>]+>', ' ')
    return [regex]::Replace([System.Net.WebUtility]::HtmlDecode($text), '\s+', ' ')
}

function Get-AssetKind([string]$Name) {
    if ($Name -match $ExcludedAsset) { return $null }
    if ($Name -match '\.zip$') { return 'zip' }
    if ($Name -match '-setup-.*\.exe$') { return 'setup' }
    if ($Name -match '^doc-html-ui-.*\.exe$') { return 'GUI exe' }
    if ($Name -match '^doc-html-translate-.*\.exe$') { return 'CLI exe' }
    return 'other'
}

function Get-ReleaseDownloads($Release) {
    $kinds = [ordered]@{ 'zip' = 0; 'CLI exe' = 0; 'setup' = 0; 'GUI exe' = 0; 'other' = 0 }
    foreach ($asset in $Release.assets) {
        $kind = Get-AssetKind $asset.name
        if ($kind) { $kinds[$kind] += [int]$asset.download_count }
    }
    $total = ($kinds.Values | Measure-Object -Sum).Sum
    $parts = foreach ($k in $kinds.Keys) { if ($k -ne 'other' -or $kinds[$k] -gt 0) { "$k $($kinds[$k])" } }
    [pscustomobject]@{
        Tag       = $Release.tag_name
        Published = ([datetime]$Release.published_at).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')
        Total     = $total
        Split     = "$total ($($parts -join ', '))"
    }
}

function Format-Cell($Value) {
    if ($null -eq $Value -or "$Value" -eq '') { return $Unknown }
    return ("$Value" -replace '\|', '\|') -replace '\s*[\r\n]+\s*', ' '
}

function Write-Checkpoint([string]$Content) {
    if ($Print) { Write-Host $Content; return 'printed, nothing written' }
    New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
    $path = Join-Path $OutDir "$Date.md"
    Set-Content -LiteralPath $path -Value $Content -Encoding utf8NoBOM -NoNewline
    return "wrote $path"
}

try {
    if ($Date -notmatch '^\d{4}-\d{2}-\d{2}$') { Exit-Verdict $Check 1 "-Date must be yyyy-MM-dd, got '$Date'" }
    Set-Location (Split-Path $PSScriptRoot -Parent)
    Write-Subject $Check "public counters of $Owner/$Repo and its store listings, read now"
    if (-not (Get-Command gh -ErrorAction SilentlyContinue)) { Exit-Verdict $Check 2 'gh is not on PATH' }

    # --- GitHub: nothing is written unless the repository and its releases were read ---
    $repoInfo = Invoke-GhApi @("repos/$Owner/$Repo")
    Register-Source 'GitHub repository' $repoInfo
    $pages = Invoke-GhApi @("repos/$Owner/$Repo/releases", '--paginate', '--slurp')
    Register-Source 'GitHub releases' $pages
    if ($null -eq $repoInfo -or $null -eq $pages) {
        Write-Host "${Check}: unread: $($unread -join '; ')"
        Exit-Verdict $Check 2 'GitHub could not be read, no checkpoint written'
    }

    $releases = @(foreach ($page in $pages) { foreach ($rel in @($page)) { if (-not $rel.draft) { $rel } } })
    $perRelease = @($releases | Sort-Object { [datetime]$_.published_at } -Descending | ForEach-Object { Get-ReleaseDownloads $_ })
    $allDownloads = ($perRelease | Measure-Object -Property Total -Sum).Sum
    if ($null -eq $allDownloads) { $allDownloads = 0 }
    $latest = $perRelease | Select-Object -First 1

    $gql = Invoke-GhApi @('graphql', '-f', "query=$GraphQlQuery", '-f', "o=$Owner", '-f', "n=$Repo")
    $ogCustom = if ($null -ne $gql.data.repository) { "$($gql.data.repository.usesCustomOpenGraphImage)".ToLower() } else { $null }
    Register-Source 'GitHub GraphQL' $ogCustom

    $views = Invoke-GhApi @("repos/$Owner/$Repo/traffic/views")
    Register-Source 'traffic views' $views
    $clones = Invoke-GhApi @("repos/$Owner/$Repo/traffic/clones")
    Register-Source 'traffic clones' $clones
    $referrers = Invoke-GhApi @("repos/$Owner/$Repo/traffic/popular/referrers")
    Register-Source 'traffic referrers' $referrers

    $dirs = Invoke-GhApi @($WingetApi)
    $wingetNewest = $null
    if ($null -ne $dirs) {
        $wingetNewest = @($dirs | Where-Object { $_.type -eq 'dir' -and $_.name -match '^\d+(\.\d+){1,3}$' } |
            Sort-Object { [version]$_.name } -Descending | ForEach-Object { $_.name }) | Select-Object -First 1
        $script:LastError = 'no version folder'
    }
    Register-Source 'winget-pkgs folder' $wingetNewest

    # --- Chrome Web Store: the page text; SOCS is Google's consent cookie, without it a EU-region
    #     request is answered with the consent interstitial instead of the listing ---
    $chromeUsers = $chromeVersion = $chromeRatings = $null
    $html = Get-WebText $ChromeUrl @{ Cookie = 'SOCS=CAI' }
    if ($html) {
        $text = ConvertTo-VisibleText $html
        if ($text -match '(\d[\d,.]*[KkMm]?\+?)\s+users?\b') { $chromeUsers = $Matches[1] }
        if ($text -match '\bVersion\s+(\d+(\.\d+)+)') { $chromeVersion = $Matches[1] }
        if ($text -match '\bNo ratings\b') { $chromeRatings = 'none shown' }
        $script:LastError = 'page text has neither a user count nor a version'
    }
    Register-Source 'Chrome Web Store' ($chromeUsers ?? $chromeVersion)

    $edge = $null
    $json = Get-WebText $EdgeUrl
    if ($json) { $edge = ConvertFrom-JsonOrNull $json }
    Register-Source 'Edge Add-ons' $edge

    $store = $null
    $json = Get-WebText $StoreUrl
    if ($json) { $store = (ConvertFrom-JsonOrNull $json).Payload }
    $storeUpdated = if ($json -match '"LastUpdateDateUtc"\s*:\s*"([^"]+)"') { $Matches[1] } else { $null }
    Register-Source 'Microsoft Store' $store

    # --- compose ---
    $viewsCell = if ($views) { "$($views.count) ($($views.uniques))" } else { $null }
    $referrerText = if ($null -eq $referrers) { $null } elseif (@($referrers).Count -eq 0) { 'none' } else {
        (@($referrers) | ForEach-Object { "$($_.referrer) $($_.count) ($($_.uniques))" }) -join '; '
    }
    $storeVersion = if ($store -and $store.Version) { $store.Version } else { $null }
    $latestCell = if ($latest) { "$($latest.Total) ($($latest.Tag))" } else { $null }

    $cells = @($Date, $repoInfo.stargazers_count, $repoInfo.forks_count, $allDownloads, $latestCell, $viewsCell,
        $chromeUsers, $edge.activeInstallCount, $null, $null, $null) | ForEach-Object { Format-Cell $_ }
    $detail = [System.Collections.Generic.List[object]]::new()
    function Add-Detail([string]$Item, $Value) { $script:detail.Add(@($Item, (Format-Cell $Value))) }
    Add-Detail 'Read at (UTC)' ([datetime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ'))
    Add-Detail 'Watchers' $repoInfo.subscribers_count
    Add-Detail 'Open issues' $repoInfo.open_issues_count
    Add-Detail 'Discussions enabled' "$($repoInfo.has_discussions)".ToLower()
    Add-Detail 'Description' $repoInfo.description
    Add-Detail 'Homepage' $repoInfo.homepage
    Add-Detail "Topics ($(@($repoInfo.topics).Count))" (@($repoInfo.topics) -join ', ')
    Add-Detail 'usesCustomOpenGraphImage' $ogCustom
    Add-Detail 'Releases (published)' $perRelease.Count
    foreach ($r in $perRelease) { Add-Detail "Downloads $($r.Tag) ($($r.Published))" $r.Split }
    Add-Detail 'Repo views 14 d (count, unique)' $(if ($views) { "$($views.count), $($views.uniques)" })
    Add-Detail 'Repo clones 14 d (count, unique; includes automation, not a reach figure)' $(if ($clones) { "$($clones.count), $($clones.uniques)" })
    Add-Detail 'Referrers 14 d (count, unique)' $referrerText
    Add-Detail 'Chrome Web Store version' $chromeVersion
    Add-Detail 'Chrome Web Store ratings' $chromeRatings
    Add-Detail 'Edge Add-ons version' $edge.version
    Add-Detail 'Edge Add-ons rating count' $edge.ratingCount
    Add-Detail 'Microsoft Store LastUpdateDateUtc' $storeUpdated
    Add-Detail 'Microsoft Store RatingCount' $store.RatingCount
    Add-Detail 'Microsoft Store served version (Payload.Version)' $storeVersion
    Add-Detail 'winget upstream newest version' $wingetNewest
    Add-Detail 'Sources not read' $(if ($unread.Count) { $unread -join '; ' } else { 'none' })

    $md = [System.Collections.Generic.List[string]]::new()
    $md.Add("# Promotion metrics checkpoint $Date")
    $md.Add('')
    $md.Add('Written by `scripts/promo-metrics.ps1`. Every value is a number read when the script ran or the word `unknown`.')
    $md.Add('Search Console, indexed pages and Store acquisitions sit in owner-held consoles and stay `unknown`.')
    $md.Add('')
    $md.Add('| Date | Stars | Forks | Downloads (all, no checksums) | Latest release downloads | Repo views 14 d (unique) | Chrome users | Edge installs | Store acquisitions | GSC impressions / clicks | Indexed pages |')
    $md.Add('| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |')
    $md.Add("| $($cells -join ' | ') |")
    $md.Add('')
    $md.Add('| Detail | Value |')
    $md.Add('| --- | --- |')
    foreach ($d in $detail) { $md.Add("| $($d[0]) | $($d[1]) |") }
    $where = Write-Checkpoint (($md -join "`n") + "`n")

    $unreadNote = if ($unread.Count) { "; unread: $($unread -join ', ')" } else { '' }
    Exit-Verdict $Check 0 "$where; $($read.Count) of $($read.Count + $unread.Count) sources read$unreadNote"
}
catch {
    Exit-Verdict $Check 2 "unexpected error: $($_.Exception.Message)"
}
