<#
.SYNOPSIS
  The security posture consistency check (canon SECURITY_AND_PRIVACY section 7): the permission
  and network-surface inventories in docs/security-posture.json against the manifests, the code,
  the dependency set and every public text rendered from them. Ends in one CHECK-VERDICT line.

.DESCRIPTION
  Its subject is the product as a whole, so it runs over the whole tree in scripts/check.ps1 - the
  gate whose evidence scripts/release.ps1 requires before the tag step (section 7 item 6).

  What fails it:
    - a permission declared in extension/manifest.json (permissions, host_permissions and their
      optional forms) or a capability declared in msix/AppxManifest.xml without an inventory row,
      or a row declaring something no manifest declares - both directions;
    - a row whose consumer evidence, or a shown string it cites, is not in the file it names - a
      permission whose consumer cannot be named is removed, not explained (item 1);
    - a source file under a declared root that contains a network primitive (fetch, an HTTP
      request, a listener) and is neither the evidence of a network-surface row nor listed in
      notNetwork with a reason - reverse coverage for item 3;
    - a dependency in go.sum or extension/package-lock.json matching the telemetry denylist while
      the inventory claims none, or a public text that no longer states the claim (item 6);
    - a public form that is not the render of the rows: the blocks in privacy.html,
      extension-privacy.html, extension/store/PRIVACY.md, extension/store/LISTING.md,
      msix/README.md, and all of docs/SECURITY_POSTURE.md (item 5);
    - a missing or future lastReconciled date, a row with no public sentence and no reason, or a
      row no public form renders (item 7).

  Exit codes follow CHECK-VERDICT: 0 PASS, 1 FAIL, 2 COULD NOT VERIFY (no git checkout, no
  inventory).

.PARAMETER Render
  Rewrite every public form from the rows first, then run the check.

.EXAMPLE
  ./scripts/security-posture.ps1 -Render   # after editing docs/security-posture.json
#>
param(
    [switch]$Render
)

$ErrorActionPreference = "Stop"
. "$PSScriptRoot/lib/verdict.ps1"
. "$PSScriptRoot/lib/docregistry.ps1"

$root = (& git rev-parse --show-toplevel 2>$null)
if ($LASTEXITCODE -ne 0 -or -not $root) {
    Write-Host "security-posture: not inside a git checkout."
    Exit-Verdict 'security-posture' 2 'no git checkout'
}
$root = $root.Trim()
Set-Location $root
$invPath = 'docs/security-posture.json'
Write-Subject 'security-posture' "$invPath against $(Get-TreeLabel)"
if (-not (Test-Path -LiteralPath $invPath)) { Exit-Verdict 'security-posture' 2 'no inventory' }
try { $inv = Get-Content -LiteralPath $invPath -Raw -Encoding utf8 | ConvertFrom-Json }
catch { Write-Host "security-posture: ${invPath}: $($_.Exception.Message)"; Exit-Verdict 'security-posture' 1 'inventory is not valid JSON' }
$files = Get-RepoFiles $root
if ($null -eq $files) { Exit-Verdict 'security-posture' 2 'git ls-files failed' }

$errors = [System.Collections.Generic.List[string]]::new()
function Add-Finding([string]$Message) { $script:errors.Add($Message) }
function Get-Words([string]$s) { return @(($s -split '\s+') | Where-Object { $_ }).Count }
function Compress-Space([string]$s) { return ([regex]::Replace($s, '\s+', ' ')).Trim() }

$fileCache = @{}
function Get-FileText([string]$Path) {
    if (-not $fileCache.ContainsKey($Path)) {
        $full = Join-Path $root $Path
        $fileCache[$Path] = if (Test-Path -LiteralPath $full -PathType Leaf) { [System.IO.File]::ReadAllText($full) } else { $null }
    }
    return $fileCache[$Path]
}
function Test-Cites($Cite, [string]$At) {
    $text = Get-FileText $Cite.file
    if ($null -eq $text) { Add-Finding "${At}: cites $($Cite.file), which does not exist"; return }
    if (-not (Compress-Space $text).Contains((Compress-Space $Cite.contains))) {
        Add-Finding "${At}: $($Cite.file) no longer contains '$($Cite.contains)'"
    }
}

# ── rendering ────────────────────────────────────────────────
$langs = @(@{ code = 'ru'; attr = 'ru' }, @{ code = 'en'; attr = 'en' }, @{ code = 'uk'; attr = 'ua' })

function ConvertTo-InlineHtml([string]$s) {
    $s = $s.Replace('&', '&amp;').Replace('<', '&lt;').Replace('>', '&gt;')
    $s = [regex]::Replace($s, '`([^`]+)`', '<code>$1</code>')
    $s = [regex]::Replace($s, '\*\*(.+?)\*\*', '<strong>$1</strong>')
    $s = [regex]::Replace($s, '(?<![\w*])\*(?=\S)(.+?)(?<=\S)\*(?![\w*])', '<em>$1</em>')
    return $s
}

function Format-Wrapped([string]$Text, [string]$First, [string]$Rest, [int]$Width = 105) {
    $lines = [System.Collections.Generic.List[string]]::new()
    $cur = $First
    $fresh = $true
    foreach ($w in ($Text -split ' ')) {
        if ($w -eq '') { continue }
        if (-not $fresh -and ($cur.Length + 1 + $w.Length) -gt $Width) { $lines.Add($cur); $cur = $Rest + $w }
        elseif ($fresh) { $cur += $w; $fresh = $false }
        else { $cur += ' ' + $w }
    }
    $lines.Add($cur)
    return $lines
}

$rowsById = @{}
foreach ($r in @($inv.permissions) + @($inv.networkSurfaces)) { if ($r.id) { $rowsById[[string]$r.id] = $r } }

function Get-BlockBody($R, [string]$Indent) {
    $rows = @($R.rows | ForEach-Object { $rowsById[[string]$_] } | Where-Object { $_ })
    $out = [System.Collections.Generic.List[string]]::new()
    switch ($R.format) {
        'html-lists' {
            foreach ($l in $langs) {
                $out.Add("$Indent<ul data-l=`"$($l.attr)`">")
                foreach ($row in $rows) { $out.Add("$Indent  <li>$(ConvertTo-InlineHtml $row.public.($l.code))</li>") }
                $out.Add("$Indent</ul>")
            }
            if ($R.after) { foreach ($l in $langs) { $out.Add("$Indent<p data-l=`"$($l.attr)`">$(ConvertTo-InlineHtml $R.after.($l.code))</p>") } }
        }
        'md-list' {
            foreach ($row in $rows) { foreach ($x in (Format-Wrapped $row.public.en '- ' '  ')) { $out.Add($x) } }
            if ($R.after) { $out.Add(''); foreach ($x in (Format-Wrapped $R.after.en '' '')) { $out.Add($x) } }
        }
        'md-justifications' {
            foreach ($row in $rows) { foreach ($x in (Format-Wrapped "$($row.storeLabel): $($row.storeJustification)" '- ' '  ')) { $out.Add($x) } }
        }
        'md-fence' {
            $out.Add('```')
            foreach ($row in $rows) { foreach ($x in ([string]$row.storeJustification -split "`n")) { $out.Add($x) } }
            $out.Add('```')
        }
    }
    return ($out -join "`n")
}

function Format-Cell([string]$s) { return $s.Replace('|', '\|').Replace("`n", ' ') }

function Get-PostureDoc {
    $nl = "`n"
    $sb = [System.Text.StringBuilder]::new()
    $w = { param($s) [void]$sb.Append($s + $nl) }
    & $w '# Security posture - what this product can touch and what it can send'
    & $w ''
    & $w "<!-- Rendered from docs/security-posture.json by scripts/security-posture.ps1 -Render. Do not edit by hand. -->"
    & $w ''
    & $w "**Last reconciled:** $($inv.lastReconciled)"
    & $w '**Contract:** canon `SECURITY_AND_PRIVACY` section 7 - the permission and network-surface inventories.'
    & $w '**Checked by:** `scripts/security-posture.ps1`, run by `scripts/check.ps1`, whose evidence `scripts/release.ps1` requires before the tag step.'
    & $w ''
    & $w 'Two editions share no code: the Windows app (CLI, GUI, MSIX) and the browser extension. Every public form of the privacy promise is rendered from these rows - the list below says which - so a row and a sentence cannot disagree without the check going red.'
    & $w ''
    & $w '## 1. Permission inventory'
    & $w ''
    & $w 'One row per declared permission, and, for the app, which has no permission manifest, one row per capability it takes (section 7 item 8). *Shown at request* says whether the sentence is shown at the moment the permission is requested.'
    & $w ''
    & $w '| Row | Edition | Declares | Declared in | Consumers | Shown at request | Where the sentence is shown |'
    & $w '|---|---|---|---|---|---|---|'
    foreach ($r in @($inv.permissions)) {
        $shown = if ($r.shownAtRequest) { 'yes' } else { 'no' }
        & $w ("| ``{0}`` | {1} | ``{2}`` | {3} | {4} | {5} | {6} |" -f $r.id, $r.edition, (Format-Cell $r.declares), (Format-Cell $r.declaredIn), (Format-Cell (@($r.consumers) -join '; ')), $shown, (Format-Cell $r.shownWhere))
    }
    & $w ''
    & $w '## 2. Network-surface inventory'
    & $w ''
    & $w 'One row per surface that opens a listening port, initiates an outbound connection, or hands a file outward.'
    & $w ''
    & $w '| Row | Edition | Kind | Surface | On by default | Turned on by | Lifetime | What leaves, to where |'
    & $w '|---|---|---|---|---|---|---|---|'
    foreach ($r in @($inv.networkSurfaces)) {
        $on = if ($r.defaultOn) { 'yes' } else { 'no' }
        & $w ("| ``{0}`` | {1} | {2} | {3} | {4} | {5} | {6} | {7} |" -f $r.id, $r.edition, $r.kind, (Format-Cell $r.surface), $on, (Format-Cell $r.turnedOnBy), (Format-Cell $r.lifetime), (Format-Cell $r.leaves))
    }
    & $w ''
    & $w 'Every source file under these roots that contains a network primitive is the evidence of a row above, or is listed here as not reaching the network:'
    & $w ''
    foreach ($rt in @($inv.networkCallSites.roots)) { & $w ("- ``{0}`` - {1}" -f $rt.glob, ((@($rt.patterns) | ForEach-Object { "``$_``" }) -join ', ')) }
    & $w ''
    foreach ($n in @($inv.networkCallSites.notNetwork)) { & $w ("- not network: ``{0}`` - {1}" -f $n.file, $n.reason) }
    & $w ''
    & $w '## 3. The "no telemetry" claim'
    & $w ''
    & $w "**Claim:** $($inv.telemetry.claim)"
    & $w ''
    & $w ('It is proven against the dependency set, not against the wording of a page: {0} are read in full and every module or package name is matched against the denylist in the source. The claim is stated in:' -f ((@($inv.telemetry.dependencySets) | ForEach-Object { "``$($_.file)``" }) -join ' and '))
    & $w ''
    foreach ($s in @($inv.telemetry.statedIn)) { & $w ("- ``{0}``" -f $s.file) }
    & $w ''
    & $w '## 4. The public forms rendered from these rows'
    & $w ''
    foreach ($r in @($inv.renders)) {
        if ($r.format -eq 'posture-doc') { continue }
        & $w ("- ``{0}``, block ``{1}`` ({2}): {3}" -f $r.target, $r.block, $r.format, ((@($r.rows) | ForEach-Object { "``$_``" }) -join ', '))
    }
    & $w ''
    & $w '## 5. How this is kept true'
    & $w ''
    & $w '- A permission added to `extension/manifest.json` or a capability added to `msix/AppxManifest.xml` without a row fails the check, and so does a row naming something no manifest declares.'
    & $w '- Every row names its consumers and cites the code that consumes it; a citation the code no longer contains fails the check. A shown string a row cites must still be in the strings file it names.'
    & $w '- A new network call site under the roots above fails the check until a row or a not-network reason covers it.'
    & $w '- An analytics, crash-reporting or advertising dependency entering either dependency set fails the check while section 3 claims none.'
    & $w '- A public form edited by hand, or a row edited without `-Render`, fails the check. The consumers, the lifetimes and the "what leaves" column are authored: no mechanism can derive which features share a permission.'
    return $sb.ToString()
}

$beginRx = { param($b) "(?m)^([ \t]*)<!-- security-posture:begin $([regex]::Escape($b))\b[^\n]*-->[ \t]*\r?\n" }
$endRx = { param($b) "(?m)^[ \t]*<!-- security-posture:end $([regex]::Escape($b)) -->" }

# Returns @{ Current; Expected; Error } for one render.
function Get-RenderState($R) {
    $path = [string]$R.target
    $text = Read-TextNormalized (Join-Path $root $path)
    if ($R.format -eq 'posture-doc') { return @{ Path = $path; Current = $text; Expected = (Get-PostureDoc); Whole = $true } }
    if ($null -eq $text) { return @{ Path = $path; Error = "$path does not exist" } }
    $b = [regex]::Match($text, (& $beginRx $R.block))
    if (-not $b.Success) { return @{ Path = $path; Error = "$path has no '<!-- security-posture:begin $($R.block) -->' marker" } }
    $e = [regex]::Match($text.Substring($b.Index + $b.Length), (& $endRx $R.block))
    if (-not $e.Success) { return @{ Path = $path; Error = "$path has no '<!-- security-posture:end $($R.block) -->' marker" } }
    $bodyStart = $b.Index + $b.Length
    $current = $text.Substring($bodyStart, $e.Index)
    $expected = (Get-BlockBody $R $b.Groups[1].Value) + "`n"
    return @{ Path = $path; Current = $current; Expected = $expected; Whole = $false; Text = $text; Start = $bodyStart; Length = $e.Index }
}

# ── render mode ──────────────────────────────────────────────
if ($Render) {
    # One pass per target file, so several blocks of one file land in one write.
    foreach ($group in (@($inv.renders) | Group-Object { [string]$_.target })) {
        $path = $group.Name
        foreach ($r in $group.Group) {
            $st = Get-RenderState $r
            if ($st.Error) { Write-Host "security-posture: cannot render - $($st.Error)" -ForegroundColor Red; continue }
            if ($st.Whole) { Write-TextLF (Join-Path $root $path) $st.Expected; continue }
            $new = $st.Text.Substring(0, $st.Start) + $st.Expected + $st.Text.Substring($st.Start + $st.Length)
            Write-TextLF (Join-Path $root $path) $new
        }
        Write-Host "security-posture: rendered $path"
    }
}

# ── the inventory itself (items 1, 3, 7) ─────────────────────
if ($inv.shape -ne 1) { Add-Finding "${invPath}: shape is '$($inv.shape)', this check reads shape 1" }
$date = [datetime]::MinValue
if (-not [datetime]::TryParseExact([string]$inv.lastReconciled, 'yyyy-MM-dd', [cultureinfo]::InvariantCulture, 'None', [ref]$date)) {
    Add-Finding "${invPath}: lastReconciled '$($inv.lastReconciled)' is not a yyyy-MM-dd date"
} elseif ($date -gt (Get-Date).Date) {
    Add-Finding "${invPath}: lastReconciled $($inv.lastReconciled) is in the future"
}

$seen = @{}
$evidenceFiles = [System.Collections.Generic.HashSet[string]]::new()
foreach ($kind in 'permissions', 'networkSurfaces') {
    $rows = @($inv.$kind)
    $empty = $inv.($kind + 'EmptyReason')
    if ($rows.Count -eq 0 -and (Get-Words $empty) -lt 4) { Add-Finding "${invPath}: $kind is empty and says nothing about why (item 7)" }
    foreach ($r in $rows) {
        $at = "$invPath $kind/$($r.id)"
        if (-not $r.id) { Add-Finding "${invPath}: a $kind row without an id"; continue }
        if ($seen.ContainsKey($r.id)) { Add-Finding "${at}: id used twice" } else { $seen[$r.id] = $true }
        if ($r.edition -notin 'app', 'extension') { Add-Finding "${at}: edition must be app or extension" }
        $req = if ($kind -eq 'permissions') { 'declares', 'declaredIn', 'shownWhere' } else { 'kind', 'surface', 'turnedOnBy', 'lifetime', 'leaves' }
        foreach ($f in $req) { if (-not $r.$f) { Add-Finding "${at}: '$f' is empty" } }
        if ($kind -eq 'permissions') {
            if (@($r.consumers | Where-Object { $_ }).Count -eq 0) { Add-Finding "${at}: no consumer named - a permission whose consumer cannot be named is removed, not explained" }
            if ($r.shownAtRequest -isnot [bool]) { Add-Finding "${at}: shownAtRequest must be true or false" }
        } else {
            if ($r.kind -notin 'listening', 'outbound', 'hand-off') { Add-Finding "${at}: kind must be listening, outbound or hand-off" }
            if ($r.defaultOn -isnot [bool]) { Add-Finding "${at}: defaultOn must be true or false" }
        }
        if (@($r.evidence).Count -eq 0) { Add-Finding "${at}: no evidence cited" }
        foreach ($c in @($r.evidence)) { Test-Cites $c $at; if ($kind -eq 'networkSurfaces' -and $c.file) { [void]$evidenceFiles.Add([string]$c.file) } }
        foreach ($c in @($r.shownText)) { if ($c) { Test-Cites $c "$at (shown string)" } }
        if ($null -eq $r.public) {
            if ((Get-Words $r.publicNote) -lt 4) { Add-Finding "${at}: no public sentence and no publicNote saying why" }
        } else {
            foreach ($l in 'en', 'ru', 'uk') { if (-not $r.public.$l) { Add-Finding "${at}: public.$l is empty" } }
        }
    }
}

# ── manifests against the inventory, both directions (item 6) ─
$declaredRows = @{}
foreach ($r in @($inv.permissions)) { if ($r.declares -match '^(permission|optional-permission|host|optional-host|msix):') { $declaredRows[[string]$r.declares] = $r } }
$declared = [System.Collections.Generic.List[string]]::new()
$manifest = Get-FileText 'extension/manifest.json'
if ($null -eq $manifest) { Add-Finding 'extension/manifest.json is missing' } else {
    $m = $manifest | ConvertFrom-Json
    foreach ($p in @($m.permissions)) { if ($p) { $declared.Add("permission:$p") } }
    foreach ($p in @($m.optional_permissions)) { if ($p) { $declared.Add("optional-permission:$p") } }
    foreach ($p in @($m.host_permissions)) { if ($p) { $declared.Add("host:$p") } }
    foreach ($p in @($m.optional_host_permissions)) { if ($p) { $declared.Add("optional-host:$p") } }
}
$appx = Get-FileText 'msix/AppxManifest.xml'
if ($null -eq $appx) { Add-Finding 'msix/AppxManifest.xml is missing' } else {
    foreach ($c in [regex]::Matches($appx, '<(?:\w+:)?(?:Device)?Capability\s+Name="([^"]+)"')) { $declared.Add("msix:$($c.Groups[1].Value)") }
}
foreach ($d in $declared) { if (-not $declaredRows.ContainsKey($d)) { Add-Finding "$d is declared in a manifest and has no row in $invPath" } }
foreach ($k in $declaredRows.Keys) { if ($declared -notcontains $k) { Add-Finding "$invPath permissions/$($declaredRows[$k].id): declares $k, which no manifest declares - remove the row" } }
foreach ($r in @($inv.permissions)) {
    if ($r.edition -eq 'extension' -or $r.declares -like 'msix:*') {
        if (-not $r.storeLabel -or (Get-Words $r.storeJustification) -lt 8) { Add-Finding "$invPath permissions/$($r.id): a declared permission needs a storeLabel and a storeJustification for the store form" }
        # Both stores' justification fields stop at 1000 characters; a longer text is cut on paste.
        elseif (([string]$r.storeJustification).Length -gt 1000) { Add-Finding "$invPath permissions/$($r.id): storeJustification is $(([string]$r.storeJustification).Length) characters, over the forms' 1000" }
    }
}

# ── network call sites: reverse coverage (item 3) ─────────────
$notNetwork = @{}
foreach ($n in @($inv.networkCallSites.notNetwork)) {
    if ((Get-Words $n.reason) -lt 4) { Add-Finding "$invPath notNetwork $($n.file): the reason is under four words" }
    $notNetwork[[string]$n.file] = $n
}
$primitiveFiles = [System.Collections.Generic.HashSet[string]]::new()
foreach ($rt in @($inv.networkCallSites.roots)) {
    foreach ($f in (Resolve-RegistryPath ([string]$rt.glob) $files)) {
        if ($rt.skip -and $f -match $rt.skip) { continue }
        $text = Get-FileText $f
        $hit = $false
        foreach ($p in @($rt.patterns)) { if ($text -match $p) { $hit = $true; break } }
        if (-not $hit) { continue }
        [void]$primitiveFiles.Add($f)
        if (-not $evidenceFiles.Contains($f) -and -not $notNetwork.ContainsKey($f)) {
            Add-Finding "$f contains a network primitive and no network-surface row cites it - add a row, or a notNetwork entry with a reason"
        }
    }
}
foreach ($k in $notNetwork.Keys) {
    if (-not $primitiveFiles.Contains($k)) { Add-Finding "$invPath notNetwork ${k}: no network primitive there any more - drop the entry" }
    if ($evidenceFiles.Contains($k)) { Add-Finding "$invPath notNetwork ${k}: also cited as a network surface - pick one" }
}

# ── the telemetry claim against the dependency set (item 6) ──
$depNames = [System.Collections.Generic.HashSet[string]]::new()
foreach ($set in @($inv.telemetry.dependencySets)) {
    $text = Get-FileText $set.file
    if ($null -eq $text) { Add-Finding "$($set.file) is missing - the telemetry claim cannot be checked against it"; continue }
    switch ($set.kind) {
        'go-sum' { foreach ($line in ($text -split "`n")) { $mod = ($line.Trim() -split '\s+')[0]; if ($mod) { [void]$depNames.Add($mod) } } }
        'npm-lock' {
            $lock = $text | ConvertFrom-Json -AsHashtable
            foreach ($k in @($lock['packages'].Keys)) { if ($k) { [void]$depNames.Add(($k -replace '^.*node_modules/', '')) } }
        }
        default { Add-Finding "$invPath telemetry: unknown dependency-set kind '$($set.kind)'" }
    }
}
$flagged = @($depNames | Where-Object { $n = $_; @($inv.telemetry.denylist | Where-Object { $n -match $_ }).Count -gt 0 } | Sort-Object)
foreach ($d in $flagged) { Add-Finding "dependency '$d' is a telemetry, analytics or crash-reporting library, and $invPath claims: $($inv.telemetry.claim)" }
foreach ($s in @($inv.telemetry.statedIn)) { Test-Cites $s "$invPath telemetry (stated claim)" }

# ── the public forms are renders of the rows (item 5) ─────────
$rendered = [System.Collections.Generic.HashSet[string]]::new()
foreach ($r in @($inv.renders)) {
    foreach ($id in @($r.rows)) {
        if (-not $rowsById.ContainsKey([string]$id)) { Add-Finding "$invPath renders $($r.target)#$($r.block): unknown row '$id'" } else { [void]$rendered.Add([string]$id) }
    }
    $st = Get-RenderState $r
    if ($st.Error) { Add-Finding $st.Error; continue }
    if ($st.Current -ne $st.Expected) {
        $what = if ($st.Whole) { $st.Path } else { "$($st.Path) block '$($r.block)'" }
        Add-Finding "$what is not the render of $invPath - run scripts/security-posture.ps1 -Render (never edit the block by hand)"
    }
}
foreach ($id in $rowsById.Keys) { if (-not $rendered.Contains($id)) { Add-Finding "$invPath row '$id' is rendered into no public form" } }

foreach ($m in $errors) { Write-Host "  $m" -ForegroundColor Red }
$summary = "$(@($inv.permissions).Count) permission row(s), $(@($inv.networkSurfaces).Count) network surface(s), $($declared.Count) declaration(s), $($depNames.Count) dependencies, $(@($inv.renders).Count) render(s)"
if ($errors.Count -gt 0) { Exit-Verdict 'security-posture' 1 "$($errors.Count): $summary" }
Exit-Verdict 'security-posture' 0 $summary
