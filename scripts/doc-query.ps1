<#
.SYNOPSIS
  "What must I read before touching this?" - queries docs/DOCUMENT_REGISTRY.jsonl by product area,
  change trigger, or file.

.DESCRIPTION
  The two query facets of canon DOCUMENTATION_CONCEPT section 6 item 2. Each filter narrows the
  result; with no filter, every record is listed. The vocabularies are this project's own - run
  with -List to print every value in use.

  Exit codes: 0 answered (an empty answer included), 2 the registry cannot be read. This is a
  query, not a check; scripts/doc-registry.ps1 is the check.

.PARAMETER Area
  A product area, e.g. ocr, extension, privacy, release.

.PARAMETER Trigger
  A change trigger, e.g. user-feature, permission, contract-change, ticket.

.PARAMETER Path
  A repository path; lists the record that is that file's home.

.PARAMETER List
  Print the product-area and change-trigger vocabularies in use.

.EXAMPLE
  ./scripts/doc-query.ps1 -Area privacy -Trigger network-surface
.EXAMPLE
  ./scripts/doc-query.ps1 -Path docs/PARITY.md
#>
param(
    [string]$Area,
    [string]$Trigger,
    [string]$Path,
    [switch]$List
)

$ErrorActionPreference = "Stop"
. "$PSScriptRoot/lib/docregistry.ps1"

$root = (& git rev-parse --show-toplevel 2>$null)
if ($LASTEXITCODE -ne 0 -or -not $root) { Write-Host "doc-query: not inside a git checkout"; exit 2 }
Set-Location $root.Trim()
if (-not (Test-Path -LiteralPath $DocRegistryPath)) { Write-Host "doc-query: $DocRegistryPath is missing"; exit 2 }

$read = Read-JsonLines $DocRegistryPath
if ($read.Errors.Count -gt 0) { $read.Errors | ForEach-Object { Write-Host "doc-query: $_" }; exit 2 }
$records = @($read.Entries | ForEach-Object Record)

if ($List) {
    Write-Host "product areas : $((@($records | ForEach-Object { $_.product_areas }) | Sort-Object -Unique) -join ', ')"
    Write-Host "change triggers: $((@($records | ForEach-Object { $_.update_triggers }) | Sort-Object -Unique) -join ', ')"
    exit 0
}

$files = $null
if ($Path) {
    $files = Get-RepoFiles (Get-Location).Path
    if ($null -eq $files) { Write-Host "doc-query: git ls-files failed"; exit 2 }
    $Path = $Path.Replace('\', '/')
    if ($Path.StartsWith('./')) { $Path = $Path.Substring(2) }
}

$hits = @($records | Where-Object {
        (-not $Area -or @($_.product_areas) -contains $Area) -and
        (-not $Trigger -or @($_.update_triggers) -contains $Trigger) -and
        (-not $Path -or (Resolve-RecordFiles $_ $files) -contains $Path)
    })

foreach ($r in $hits) {
    $role = if ($r.generated) { 'render - regenerate, never edit' } else { 'source' }
    $pub = if ($r.published -and $r.indexable) { "published, announced at $($r.url)" } elseif ($r.published) { 'published, not in the sitemap' } else { 'not published' }
    Write-Host "$($r.id) - $($r.title)" -ForegroundColor Cyan
    Write-Host "    paths   : $(@($r.paths) -join ', ')"
    Write-Host "    role    : $role; $pub"
    Write-Host "    areas   : $(@($r.product_areas) -join ', ')"
    Write-Host "    triggers: $(@($r.update_triggers) -join ', ')"
    if ($r.notes) { Write-Host "    notes   : $($r.notes)" }
}
Write-Host "doc-query: $($hits.Count) record(s)"
exit 0
