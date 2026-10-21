<#
.SYNOPSIS
  The documentation registry check of the local, git-ignored engineering corpus (DOC-INTERNAL-QUALITY 0.10, ticket
  122): DEV/DOCUMENT_REGISTRY.jsonl against DEV/plan and DEV/private, in both directions, and the links of those
  documents. Ends in one CHECK-VERDICT line.

.DESCRIPTION
  scripts/doc-registry.ps1 holds the registry of everything a clone can open. The ticket store and the private notes
  are git-ignored, so the tracked registry names none of them (it fails if it does) and this check holds their own
  registry true:

    - DEV/DOCUMENT_REGISTRY.jsonl is itself git-ignored; every record is valid JSON in record shape 1, with a unique
      kebab-case id, non-empty product_areas and update_triggers, published and indexable false, and notes of four
      words or more;
    - every explicit path exists, every record resolves to a file, no file has two home records, and no path leaves the
      local trees (a document a clone can open belongs to the tracked registry);
    - every .md and .html file under DEV/plan and DEV/private is claimed by a record (reverse coverage);
    - every relative link and heading anchor of those documents resolves (go test ./tests -run TestLocalDocLinks, the
      same scanner the committed corpus is held to, here judging the links into the local trees as well).

  The corpus exists only on the owner's machine and is edited by many sessions at once, so this is a hand-run check
  (configs/check-placement.jsonl): on a clone without DEV/plan it can only say COULD NOT VERIFY. Run it after adding,
  moving or renaming a ticket, a research note or a phase file.

  Exit codes follow CHECK-VERDICT (scripts/lib/verdict.ps1): 0 PASS, 1 FAIL, 2 COULD NOT VERIFY (no git checkout, no
  local corpus, no local registry, no Go toolchain).

.PARAMETER SkipLinks
  Judge the registry only; do not run the link scan.

.EXAMPLE
  ./scripts/doc-registry-local.ps1
#>
param(
    [switch]$SkipLinks
)

$ErrorActionPreference = "Stop"
. "$PSScriptRoot/lib/verdict.ps1"
. "$PSScriptRoot/lib/docregistry.ps1"

$root = (& git rev-parse --show-toplevel 2>$null)
if ($LASTEXITCODE -ne 0 -or -not $root) {
    Write-Host "doc-registry-local: not inside a git checkout - the ignore rules cannot be read."
    Exit-Verdict 'doc-registry-local' 2 'no git checkout'
}
$root = $root.Trim()
Set-Location $root
Write-Subject 'doc-registry-local' "$DocRegistryLocalPath against $($DocRegistryLocalTrees -join ', ') on disk"

$trees = @($DocRegistryLocalTrees | Where-Object { Test-Path -LiteralPath $_ -PathType Container })
if ($trees.Count -eq 0) {
    Write-Host "doc-registry-local: none of $($DocRegistryLocalTrees -join ', ') exists here - the local corpus is the owner's machine only."
    Exit-Verdict 'doc-registry-local' 2 'no local corpus'
}
if (-not (Test-Path -LiteralPath $DocRegistryLocalPath)) {
    Write-Host "doc-registry-local: $DocRegistryLocalPath is missing, and the local corpus holds documents it should index."
    Exit-Verdict 'doc-registry-local' 2 'no local registry'
}

$errors = [System.Collections.Generic.List[string]]::new()
function Add-Finding([string]$Message) { $script:errors.Add($Message) }

# The registry and every tree it indexes must be git-ignored, or they belong to the tracked registry.
foreach ($ignored in (@($DocRegistryLocalPath) + $trees)) {
    & git check-ignore -q -- $ignored 2>$null
    if ($LASTEXITCODE -ne 0) { Add-Finding "$ignored is not git-ignored - a document a clone can open is indexed by $DocRegistryPath, not here" }
}

# The files on disk under the local trees (the documents the reverse coverage judges, and every target a path may name).
$files = [System.Collections.Generic.List[string]]::new()
foreach ($tree in $trees) {
    foreach ($f in (Get-ChildItem -LiteralPath $tree -Recurse -File)) {
        $files.Add($f.FullName.Substring($root.Length + 1).Replace('\', '/'))
    }
}

$read = Read-JsonLines $DocRegistryLocalPath
foreach ($e in $read.Errors) { Add-Finding $e }
$records = @($read.Entries | ForEach-Object Record)
$required = @('id', 'title', 'category', 'audience', 'paths', 'published', 'indexable', 'product_areas', 'update_triggers', 'generated')
$known = $required + @('notes')
$kebab = '^[a-z0-9]+(?:-[a-z0-9]+)*$'

$ids = @{}
$owner = @{}
foreach ($entry in $read.Entries) {
    $r = $entry.Record
    $at = "${DocRegistryLocalPath}:$($entry.Line) ($($r.id))"
    $fields = Get-FieldNames $r
    foreach ($f in $required) { if ($fields -notcontains $f) { Add-Finding "${at}: missing field '$f'" } }
    foreach ($f in $fields) { if ($known -notcontains $f) { Add-Finding "${at}: field '$f' is not in the local record shape (shape $DocRegistryShape without the site fields)" } }
    if ([string]$r.id -notmatch $kebab) { Add-Finding "${at}: id is not kebab-case" }
    if ($ids.ContainsKey([string]$r.id)) { Add-Finding "${at}: id repeats line $($ids[[string]$r.id])" } else { $ids[[string]$r.id] = $entry.Line }
    foreach ($facet in 'product_areas', 'update_triggers') {
        $vals = @($r.$facet | Where-Object { $_ })
        if ($vals.Count -eq 0) { Add-Finding "${at}: $facet is empty - it is one of the two query facets" }
        foreach ($v in $vals) { if ([string]$v -notmatch $kebab) { Add-Finding "${at}: $facet value '$v' is not kebab-case" } }
    }
    if ($r.published -or $r.indexable) { Add-Finding "${at}: a local record is neither published nor indexable - an index of files a clone cannot open is not to be published" }
    $words = @(([string]$r.notes -split '\s+') | Where-Object { $_ })
    if ($words.Count -lt 4) { Add-Finding "${at}: notes do not say what the record holds (four words or more)" }

    $recordFiles = [System.Collections.Generic.List[string]]::new()
    foreach ($p in @($r.paths)) {
        $p = [string]$p
        if ($p -match '(^|/)\.\.(/|$)' -or $p.StartsWith('/') -or $p -match '^[A-Za-z]:') { Add-Finding "${at}: path escapes the repository: $p"; continue }
        $inTree = $false
        foreach ($tree in $DocRegistryLocalTrees) { if ($p -eq $tree -or $p.StartsWith($tree + '/')) { $inTree = $true } }
        if (-not $inTree) { Add-Finding "${at}: path $p is outside $($DocRegistryLocalTrees -join ', ') - $DocRegistryPath indexes what a clone can open"; continue }
        $hits = Resolve-RegistryPath $p $files
        if ($hits.Count -eq 0 -and -not (Test-IsGlob $p)) { Add-Finding "${at}: no file at '$p'"; continue }
        foreach ($h in $hits) { if (-not $recordFiles.Contains($h)) { $recordFiles.Add($h) } }
    }
    if ($recordFiles.Count -eq 0) { Add-Finding "${at}: no path of the record matches a file" }
    foreach ($f in $recordFiles) {
        if ($owner.ContainsKey($f) -and $owner[$f] -ne $r.id) { Add-Finding "${at}: $f is already claimed by record '$($owner[$f])' - one home per document" }
        else { $owner[$f] = $r.id }
    }
}

# reverse coverage: every document of the local trees is a record's
$covered = 0
foreach ($f in $files) {
    if ($f -notmatch '\.(md|html?)$') { continue }
    if (-not $owner.ContainsKey($f)) { Add-Finding "$f is in no record - register it in $DocRegistryLocalPath (a new record or a glob on an existing one)"; continue }
    $covered++
}

# the links of those documents: the scanner of the committed corpus, judging the links into the local trees as well
$linkNote = 'links skipped'
if (-not $SkipLinks) {
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Write-Host "doc-registry-local: go not found on PATH - the link scan cannot run."
        Exit-Verdict 'doc-registry-local' 2 'go toolchain absent'
    }
    . "$PSScriptRoot/lib/gobuildtags.ps1"
    $env:DOCHT_LOCAL_CORPUS = '1'
    $out = @(& go test ./tests -run 'TestLocalDocLinks$' -count=1 -v 2>&1 | ForEach-Object { "$_" })
    $goCode = $LASTEXITCODE
    Remove-Item Env:DOCHT_LOCAL_CORPUS -ErrorAction SilentlyContinue
    $ran = @($out | Where-Object { $_ -match '^--- PASS: TestLocalDocLinks ' }).Count -gt 0
    if ($goCode -ne 0) {
        $linkNote = 'link scan found problems'
        foreach ($l in ($out | Where-Object { $_ -match 'doc_links_local_test\.go:\d+:' -and $_ -notmatch 'local document\(s\) read' })) { Add-Finding ($l.Trim() -replace '^doc_links_local_test\.go:\d+:\s*', '') }
        if (-not ($out | Where-Object { $_ -match 'doc_links_local_test\.go:\d+:' })) { Add-Finding "the link scan did not run: $($out | Select-Object -Last 3 | ForEach-Object { $_.Trim() })" }
    } elseif (-not $ran) {
        Write-Host "doc-registry-local: the link scan was skipped, so it inspected nothing."
        Exit-Verdict 'doc-registry-local' 2 'link scan skipped'
    } else {
        $m = $out | Where-Object { $_ -match '(\d+) local document\(s\) read' } | Select-Object -First 1
        $linkNote = if ($m -and $m -match '(\d+) local document\(s\) read') { "links of $($Matches[1]) documents resolve" } else { 'links resolve' }
    }
}

foreach ($m in $errors) { Write-Host "  $m" -ForegroundColor Red }
$summary = "$($records.Count) record(s), $covered document file(s) covered, $linkNote"
if ($errors.Count -gt 0) { Exit-Verdict 'doc-registry-local' 1 "$($errors.Count): $summary" }
Exit-Verdict 'doc-registry-local' 0 $summary
