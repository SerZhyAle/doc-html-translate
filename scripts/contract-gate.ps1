<#
.SYNOPSIS
  The contract gate on the release path (canon RELEASE_AND_DISTRIBUTION section 2 "Contract gate",
  CONTRACTS section 6): every shared contract this product produces or consumes, read against the
  catalog's registry. Ends in one CHECK-VERDICT line; scripts/release.ps1 prints it before the tag
  step and blocks the step on FAIL or UNVERIFIED.

.DESCRIPTION
  The contracts this product holds are the pointers under docs/contracts/ (REPO-LAYOUT rule 2): id,
  version, role. Each is read against the catalog's _meta/REGISTRY.md:

    FAIL  - the catalog does not list the contract;
          - no adoption row for this product (section 6 item 1), or its verification is `pending`
            or older than the last release (the last v* tag);
          - the pointer claims a version the catalog does not have yet - the contract change comes
            first, never after (item 4);
          - two or more MAJOR versions behind (item 3);
          - neither implemented nor read, and no open exception covers it (item 2: absent is not
            allowed);
          - an open exception for this product past its `until` date.
    WARN  - behind the catalog within one MAJOR (allowed with a dated reason in the row);
          - the registry row and the pointer disagree on the version;
          - not adopted, but covered by an open, dated exception;
          - a registry row for this product with no pointer in docs/contracts/.

  Item 5 (the conformance vectors ran in this suite) is reported, not judged: the catalog's
  conformance column is printed per contract, and the run to cite is the gate evidence of
  scripts/check.ps1, which the release step already requires.

  Verdict, in the canon's words and in CHECK-VERDICT codes:
    PASS        exit 0      WARN        exit 3 (PASS WITH ADVISORIES)
    FAIL        exit 1      UNVERIFIED  exit 2 (COULD NOT VERIFY) - the catalog is not reachable,
                                        which is the normal state of a clone, or no contract was
                                        checked at all; never a PASS.

  Where the catalog is: -Catalog, else $env:SZA_CONTRACTS_CATALOG, else the path AGENTS.md states -
  the one tracked file allowed to name it. This script names no path of its own.

.PARAMETER Catalog
  The catalog's root folder (the one holding _meta/REGISTRY.md).

.PARAMETER Product
  The product name as the registry's Product column spells it.

.PARAMETER Today
  Test hook: the date to judge exception expiry against (yyyy-MM-dd). Default: today.

.EXAMPLE
  ./scripts/contract-gate.ps1
#>
param(
    [string]$Catalog,
    [string]$Product = 'doc-html-translate',
    [string]$Today
)

$ErrorActionPreference = "Stop"
. "$PSScriptRoot/lib/verdict.ps1"

$root = (& git rev-parse --show-toplevel 2>$null)
if ($LASTEXITCODE -ne 0 -or -not $root) {
    Write-Host "contract-gate: not inside a git checkout."
    Write-Host "contract-gate: verdict = UNVERIFIED"
    Exit-Verdict 'contract-gate' 2 'no git checkout'
}
Set-Location $root.Trim()

# ── where the catalog is ─────────────────────────────────────
$source = ''
if (-not $Catalog -and $env:SZA_CONTRACTS_CATALOG) { $Catalog = $env:SZA_CONTRACTS_CATALOG; $source = 'SZA_CONTRACTS_CATALOG' }
if (-not $Catalog -and (Test-Path -LiteralPath 'AGENTS.md')) {
    $m = [regex]::Match((Get-Content -LiteralPath 'AGENTS.md' -Raw -Encoding utf8), 'The shared contracts catalog is at `([^`]+)`')
    if ($m.Success) { $Catalog = $m.Groups[1].Value; $source = 'AGENTS.md' }
}
if ($Catalog -and -not $source) { $source = '-Catalog' }
$registryPath = if ($Catalog) { Join-Path $Catalog '_meta/REGISTRY.md' } else { $null }
Write-Subject 'contract-gate' "docs/contracts/*.md against the catalog registry ($(if ($Catalog) { "from $source" } else { 'no catalog named' })), $(Get-TreeLabel)"
if (-not $registryPath -or -not (Test-Path -LiteralPath $registryPath)) {
    Write-Host "contract-gate: the catalog's registry is not reachable here, so nothing binding this release to its contracts was checked."
    Write-Host "contract-gate: verdict = UNVERIFIED"
    Exit-Verdict 'contract-gate' 2 'catalog not reachable'
}

$todayDate = if ($Today) { [datetime]::ParseExact($Today, 'yyyy-MM-dd', [cultureinfo]::InvariantCulture) } else { (Get-Date).Date }

# ── the registry's tables ────────────────────────────────────
function Split-Row([string]$Line) {
    $cells = $Line.Replace('\|', "`u{1}").Trim().Trim('|') -split '\|'
    return @($cells | ForEach-Object { $_.Replace("`u{1}", '|').Trim() })
}
function Get-Ids([string]$Cell) { return @([regex]::Matches($Cell, '`([A-Z0-9][A-Z0-9-]*)`') | ForEach-Object { $_.Groups[1].Value }) }
function Get-Version([string]$s) { $m = [regex]::Match($s, '\d+(?:\.\d+)+'); if ($m.Success) { return $m.Value } return $null }
function Compare-Version([string]$a, [string]$b) {
    $x = @($a.Split('.') | ForEach-Object { [int]$_ }); $y = @($b.Split('.') | ForEach-Object { [int]$_ })
    for ($i = 0; $i -lt [Math]::Max($x.Count, $y.Count); $i++) {
        $p = if ($i -lt $x.Count) { $x[$i] } else { 0 }; $q = if ($i -lt $y.Count) { $y[$i] } else { 0 }
        if ($p -ne $q) { return [Math]::Sign($p - $q) }
    }
    return 0
}

$section = 0
$contracts = @{}
$adoption = [System.Collections.Generic.List[object]]::new()
$exceptions = [System.Collections.Generic.List[object]]::new()
foreach ($line in (Get-Content -LiteralPath $registryPath -Encoding utf8)) {
    if ($line -match '^## (\d+)\.') { $section = [int]$Matches[1]; continue }
    if ($line -notmatch '^\|\s*`') { continue }
    $c = Split-Row $line
    switch ($section) {
        1 { foreach ($id in (Get-Ids $c[0])) { $contracts[$id] = [pscustomobject]@{ Version = $c[2]; Status = $c[3]; Artifacts = $c[6] } } }
        2 {
            if ($c.Count -lt 7 -or $c[1] -notlike "$Product*") { continue }
            $ids = @(Get-Ids $c[0])
            $impl = @($c[3] -split '\s*/\s*'); $reads = @($c[4] -split '\s*/\s*')
            for ($i = 0; $i -lt $ids.Count; $i++) {
                $adoption.Add([pscustomobject]@{
                        Id = $ids[$i]; Role = $c[2]; Verified = $c[5]
                        Implements = if ($impl.Count -eq $ids.Count) { $impl[$i] } else { $impl[0] }
                        Reads = if ($reads.Count -eq $ids.Count) { $reads[$i] } else { $reads[0] }
                    })
            }
        }
        3 {
            if ($c.Count -lt 5 -or $c[1] -notlike "$Product*") { continue }
            $closed = $c[2].StartsWith('~~') -or $c[4] -match '^(closed|-)$'
            foreach ($id in (Get-Ids $c[0])) { $exceptions.Add([pscustomobject]@{ Id = $id; Until = $c[4]; Closed = $closed; Deviation = $c[2] }) }
        }
    }
}

# ── the last release ─────────────────────────────────────────
$lastTag = (& git describe --tags --abbrev=0 --match 'v*' 2>$null)
$lastRelease = $null
if ($LASTEXITCODE -eq 0 -and $lastTag) {
    $d = (& git log -1 --format=%cs $lastTag.Trim() 2>$null)
    if ($d) { $lastRelease = [datetime]::ParseExact($d.Trim(), 'yyyy-MM-dd', [cultureinfo]::InvariantCulture) }
}
$global:LASTEXITCODE = 0

# ── the pointers this repository holds ───────────────────────
$fails = [System.Collections.Generic.List[string]]::new()
$warns = [System.Collections.Generic.List[string]]::new()
$pointers = @{}
foreach ($f in (Get-ChildItem -LiteralPath 'docs/contracts' -Filter '*.md' -File | Where-Object Name -NE 'README.md' | Sort-Object Name)) {
    $text = Get-Content -LiteralPath $f.FullName -Raw -Encoding utf8
    $ids = @([regex]::Matches($text, '(?m)^- \*\*Id:\*\*\s*(.+)$') | ForEach-Object { Get-Ids $_.Groups[1].Value } | ForEach-Object { $_ })
    $ver = [regex]::Match($text, '(?m)^- \*\*Version:\*\*\s*(.+)$')
    $role = [regex]::Match($text, '(?m)^- \*\*Role:\*\*\s*(.+)$')
    if ($ids.Count -eq 0 -or -not $ver.Success) { $fails.Add("docs/contracts/$($f.Name): no '- **Id:**' or '- **Version:**' line - a pointer names its contract and version"); continue }
    foreach ($id in $ids) { $pointers[$id] = [pscustomobject]@{ File = "docs/contracts/$($f.Name)"; Version = (Get-Version $ver.Groups[1].Value); Role = $role.Groups[1].Value } }
}

$checked = 0
foreach ($id in ($pointers.Keys | Sort-Object)) {
    $p = $pointers[$id]
    if ($p.Role -match '^not applicable') { Write-Host "  $id - not applicable here ($($p.File)); skipped"; continue }
    $checked++
    $cat = $contracts[$id]
    if (-not $cat) { $fails.Add("${id}: $($p.File) points at a contract the catalog's registry does not list"); continue }
    $current = Get-Version $cat.Version
    $row = $adoption | Where-Object Id -EQ $id | Select-Object -First 1
    $open = @($exceptions | Where-Object { $_.Id -eq $id -and -not $_.Closed })
    Write-Host ("  {0,-22} pointer {1,-6} catalog {2,-6} {3}" -f $id, $p.Version, $current, $(if ($row) { "row: implements $($row.Implements), reads $($row.Reads), verified $($row.Verified)" } else { 'row: none' }))
    if ($cat.Artifacts -and $cat.Artifacts -notmatch '^none') { Write-Host "      conformance artifacts in the catalog: $($cat.Artifacts) - cite this release's run of them" }

    if (-not $row) { $fails.Add("${id}: no adoption row for $Product in the catalog registry (CONTRACTS section 6 item 1)"); continue }
    $verified = [datetime]::MinValue
    if (-not [datetime]::TryParseExact($row.Verified, 'yyyy-MM-dd', [cultureinfo]::InvariantCulture, 'None', [ref]$verified)) {
        $fails.Add("${id}: the registry row is not verified ('$($row.Verified)')")
    } elseif ($lastRelease -and $verified -lt $lastRelease) {
        $fails.Add("${id}: verified $($row.Verified), before the last release $lastTag ($($lastRelease.ToString('yyyy-MM-dd'))) - re-verify the row")
    }

    $held = Get-Version $row.Implements
    if (-not $held) { $held = Get-Version $row.Reads }
    if (-not $held) {
        if ($open.Count -gt 0) { $warns.Add("${id}: neither implemented nor read; covered by an open exception until $($open[0].Until)") }
        else { $fails.Add("${id}: neither implemented nor read, and no open exception covers it (absent is not allowed)") }
    } elseif ($p.Version -and (Compare-Version $held $p.Version) -ne 0) {
        $warns.Add("${id}: the registry row says $held, $($p.File) says $($p.Version) - one of them is stale")
    }

    if ($p.Version -and $current) {
        $cmp = Compare-Version $p.Version $current
        if ($cmp -gt 0) { $fails.Add("${id}: $($p.File) claims $($p.Version), the catalog is at $current - the contract change goes into the catalog first") }
        elseif ($cmp -lt 0) {
            $majors = [int]$current.Split('.')[0] - [int]$p.Version.Split('.')[0]
            if ($majors -ge 2) { $fails.Add("${id}: $majors MAJOR versions behind the catalog ($($p.Version) against $current)") }
            else { $warns.Add("${id}: behind the catalog ($($p.Version) against $current) - allowed with a dated reason in the row") }
        }
    }
}

foreach ($e in ($exceptions | Where-Object { -not $_.Closed })) {
    $until = [datetime]::MinValue
    if ([datetime]::TryParseExact($e.Until, 'yyyy-MM-dd', [cultureinfo]::InvariantCulture, 'None', [ref]$until)) {
        if ($until -lt $todayDate) { $fails.Add("$($e.Id): an open exception for $Product expired on $($e.Until) - an expired row is a violation") }
    } else {
        $fails.Add("$($e.Id): an open exception for $Product has no until date ('$($e.Until)')")
    }
}
foreach ($r in $adoption) {
    if ($r.Role -eq '-') { continue }
    if (-not $pointers.ContainsKey($r.Id)) { $warns.Add("$($r.Id): the registry has a $Product row, docs/contracts/ has no pointer for it") }
}

Write-Host ""
foreach ($m in $fails) { Write-Host "  FAIL  $m" -ForegroundColor Red }
foreach ($m in $warns) { Write-Host "  WARN  $m" -ForegroundColor Yellow }
$summary = "$checked contract(s), $(@($exceptions | Where-Object { -not $_.Closed }).Count) open exception(s), last release $(if ($lastTag) { $lastTag.Trim() } else { 'none' })"
if ($fails.Count -gt 0) { Write-Host "contract-gate: verdict = FAIL"; Exit-Verdict 'contract-gate' 1 "$($fails.Count) failing, $($warns.Count) warning: $summary" }
# Nothing read against the registry is nothing proven: a gate that checked zero contracts - no
# pointers, or only not-applicable ones - is never a PASS.
if ($checked -eq 0) {
    Write-Host "contract-gate: no contract in docs/contracts/ was checked against the registry."
    Write-Host "contract-gate: verdict = UNVERIFIED"
    Exit-Verdict 'contract-gate' 2 "0 contracts checked$(if ($warns.Count) { ", $($warns.Count) warning" }): $summary"
}
if ($warns.Count -gt 0) { Write-Host "contract-gate: verdict = WARN"; Exit-Verdict 'contract-gate' 3 "$($warns.Count) warning: $summary" }
Write-Host "contract-gate: verdict = PASS"
Exit-Verdict 'contract-gate' 0 $summary
