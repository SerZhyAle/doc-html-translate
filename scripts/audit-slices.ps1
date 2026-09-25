<#
.SYNOPSIS
  Cuts the audit scope (class A: shipped code and the release path) into slices small enough to read
  line by line, and measures a sliced audit campaign. Ends in one CHECK-VERDICT line.

.DESCRIPTION
  Slicing (the default) walks the files git would commit, keeps the class-A ones (class B - tests,
  the OCR lab, fixture and screenshot scripts, the GUI's string dictionary - only with
  -IncludeClassB), and groups them into units: a Go package, a family of extension modules (the
  first dash-separated word of the name: ocr-*, page-*, ..), one release-path script or folder.
  A JS module paired with a Go package in configs/parity-map.json joins that package's unit, so the
  two sides are read together. Inside a unit a *_windows.go file and its *_nonwindows.go twin are
  one atom and are never parted.

  A unit over the limits is cut into the fewest contiguous parts whose largest part is smallest;
  its parts are sibling slices, kept adjacent in the order and naming each other. Units that fit
  are merged with their neighbours under the same parent (internal/, extension/src, the release
  path) up to the limits.

  Every slice gets a risk score: weighted hits of the risk signals below per thousand lines, the
  divisor floored at a quarter of -MaxLines so a tiny slice cannot outrank a large one on one hit,
  times (1 + the share of the slice added since -Base). Sibling groups are ordered by their highest
  score; a release-path slice that lands below the median is pinned first. The ids of the previous
  register (-Register) are placed on the slices that hold the files they cite.

  The same tree gives the same manifest, byte for byte.

  -Summary reads the ticket folder instead - slices.json, INDEX.md, FINDINGS.md - and reports
  coverage (class-A files no slice holds, files held twice), slices by state, the re-check lines
  still missing, finding totals by severity, crit/high findings with neither a ticket nor an inline
  fix, and the tickets filed with their status lines.

  Exit codes follow CHECK-VERDICT (scripts/lib/verdict.ps1):
    slicing:  0 manifest computed (and written with -Write / -Tail); 1 a slice broke a limit it
              could have kept (a bug); 2 no git checkout, an unknown -Base, or no parity map
    -Summary: 0 campaign closed; 1 campaign open (open slices, uncovered or doubly held files,
              missing re-checks, untriaged crit/high); 2 no manifest or no INDEX.md

.PARAMETER MaxLines
  Line limit of one slice (default 3000). A single atom over it becomes a slice of its own.

.PARAMETER MaxFiles
  File limit of one slice (default 25).

.PARAMETER Base
  The commit "changed since" is measured against (default 41fbc1b, the commit the 2026-09-24
  register cites).

.PARAMETER Ticket
  The ticket folder that holds slices.json, INDEX.md, FINDINGS.md and PHASE_TEMPLATE.md. The default is
  the first campaign (ticket 34, closed); a new campaign is a new ticket passed here, with its own base.

.PARAMETER Register
  The previous findings register whose ids are placed on slices.

.PARAMETER IncludeClassB
  Slice class B too (tests, the OCR lab, fixture and screenshot scripts). Changes membership only.

.PARAMETER Write
  Write <Ticket>/slices.json; when <Ticket>/INDEX.md does not exist yet, also scaffold INDEX.md
  and one PHASE_NN file per slice from <Ticket>/PHASE_TEMPLATE.md.

.PARAMETER Tail
  Slice only the class-A files the written manifest does not hold (added since slicing) into tail
  slices with the next ids, append them to slices.json, and append their phases to INDEX.md.

.PARAMETER Summary
  Measure the campaign instead of slicing.

.EXAMPLE
  ./scripts/audit-slices.ps1                  # print the slices of the live tree
.EXAMPLE
  ./scripts/audit-slices.ps1 -Write           # write the manifest, scaffold the phases once
.EXAMPLE
  ./scripts/audit-slices.ps1 -Summary         # campaign state; exit 0 only when it is closed
#>
param(
    [int]$MaxLines = 3000,
    [int]$MaxFiles = 25,
    [string]$Base = '41fbc1b',
    [string]$Ticket = 'DEV/plan/done/34_2026-09-25_full-code-audit-pre-release',
    [string]$Register = 'DEV/research/audit_2026-09-24/README.md',
    [switch]$IncludeClassB,
    [switch]$Write,
    [switch]$Tail,
    [switch]$Summary
)

$ErrorActionPreference = 'Stop'
. "$PSScriptRoot/lib/verdict.ps1"
$check = 'audit-slices'

$root = (& git rev-parse --show-toplevel 2>$null)
if ($LASTEXITCODE -ne 0 -or -not $root) {
    Write-Host "${check}: not inside a git checkout - the tree cannot be enumerated."
    Exit-Verdict $check 2 'no git checkout'
}
$root = $root.Trim()
Set-Location $root

$ordinal = [System.StringComparer]::Ordinal
function Sort-Ordinal([string[]]$Items) {
    $a = [string[]]@($Items)
    [Array]::Sort($a, $ordinal)
    return , $a
}
function Get-Dir([string]$Path) {
    $i = $Path.LastIndexOf('/')
    if ($i -lt 0) { return '.' }
    return $Path.Substring(0, $i)
}
function Get-Leaf([string]$Path) { return $Path.Substring($Path.LastIndexOf('/') + 1) }
function Write-Lf([string]$Path, [string]$Text) {
    $dir = Split-Path -Parent $Path
    if ($dir -and -not (Test-Path $dir)) { New-Item -ItemType Directory -Force $dir | Out-Null }
    [System.IO.File]::WriteAllText((Join-Path $root $Path), $Text.Replace("`r`n", "`n"), [System.Text.UTF8Encoding]::new($false))
}

# --- scope -------------------------------------------------------------------------------------

# Class A is shipped code plus the release path; class B is code that ships nothing. Anything else
# (docs, data, binaries, site pages) is not audit scope at all.
function Get-AuditClass([string]$p) {
    if ($p -match '(^|/)(vendor|node_modules|dist)/') { return $null }
    if ($p -match '_test\.go$' -or $p -match '^tests/' -or $p -match '^extension/test/') { return 'B' }
    if ($p -match '^(cmd|internal)/.+\.go$') { return 'A' }
    # The GUI's interface dictionary is strings, which release package 3 owns (ticket 34 non-goals).
    if ($p -eq 'cmd/doc-html-ui/i18n.js') { return 'B' }
    if ($p -match '^cmd/.+\.(html|js|css)$') { return 'A' }
    if ($p -match '^cmd/[^/]+/versioninfo\.json$') { return 'A' }
    if ($p -match '^extension/src/.+\.(js|html|css)$') { return 'A' }
    if ($p -match '^extension/(manifest\.json|package\.json|build\.mjs)$') { return 'A' }
    # _lib.mjs is the publish scripts' .env and HTTP layer, so it is release path too.
    if ($p -match '^extension/scripts/(publish-[^/]+|bump-version|gen-appearance|_lib)\.mjs$') { return 'A' }
    if ($p -match '^scripts/([^/]+|lib/[^/]+)\.ps1$') { return 'A' }
    if ($p -match '^msix/[^/]+\.(ps1|xml)$') { return 'A' }
    if ($p -match '^installer/[^/]+\.iss$') { return 'A' }
    if ($p -match '^winget/[^/]+\.yaml$') { return 'A' }
    if ($p -match '^\.github/workflows/[^/]+\.ya?ml$') { return 'A' }
    if ($p -match '^(tools|extension/scripts)/.+\.(go|js|mjs|ps1)$') { return 'B' }
    return $null
}

function Test-Generated([string]$p, [string[]]$Head) {
    if ($p -match '_gen\.go$') { return $true }
    foreach ($l in $Head) { if ($l -match 'Code generated .* DO NOT EDIT') { return $true } }
    return $false
}

function Get-TreeFiles {
    $list = @(& git -c core.quotepath=off ls-files --cached --others --exclude-standard)
    if ($LASTEXITCODE -ne 0) { Exit-Verdict $check 2 'git ls-files failed' }
    $seen = [System.Collections.Generic.HashSet[string]]::new($ordinal)
    $out = [System.Collections.Generic.List[string]]::new()
    foreach ($f in $list) {
        if (-not $f -or -not $seen.Add($f)) { continue }
        if (-not (Test-Path -LiteralPath (Join-Path $root $f) -PathType Leaf)) { continue }   # deleted in the working tree
        $out.Add($f)
    }
    return , (Sort-Ordinal $out.ToArray())
}

function Get-ScopeFiles([string[]]$Tree, [bool]$WithB) {
    $out = [System.Collections.Generic.List[string]]::new()
    foreach ($f in $Tree) {
        $c = Get-AuditClass $f
        if ($c -eq 'A' -or ($WithB -and $c -eq 'B')) { $out.Add($f) }
    }
    return , $out.ToArray()
}

# --- measurements ------------------------------------------------------------------------------

# Risk signals: [regex, weight]. Counted per line of text, in every file type - a pattern that
# cannot occur in a language simply never hits there.
$signals = @(
    @('\bos\.(RemoveAll|Remove)\(', 3),                                   # deletes
    @('\bos\.(Rename|WriteFile|Create|OpenFile|Truncate)\(|\bfsutil\.', 2), # writes and renames
    @('\bexec\.Command|\bprocrun\.', 3),                                  # process spawns
    @('\bfilepath\.Join\(|\bzip\.|\btar\.|\bfilepath\.Rel\(', 1),        # archive and path joins
    @('\.HandleFunc\(|\bhttp\.(Handle|ListenAndServe|Serve)\b|\bnet\.Listen\(', 3),  # HTTP handlers
    @('^\s*go\s+(func\b|[A-Za-z_][\w.]*\()|\bsync\.(Mutex|RWMutex|Once)\b|\batomic\.', 2),  # goroutines, shared state
    @('\b(innerHTML|outerHTML|insertAdjacentHTML)\b|document\.write\(', 3),  # HTML sinks
    @('\.onMessage(External)?\.addListener|addEventListener\(\s*[''"]message', 2),  # message listeners
    @('\beval\(|new Function\(', 3),
    @('\bfetch\(|\bXMLHttpRequest\b|https?\.request\(', 1),
    @('\bchrome\.(scripting|declarativeNetRequest|offscreen|downloads)\b', 2),
    @('\bInvoke-Expression\b|\biex\s', 3),
    @('\bgit\s+(push|tag)\b|\bgh\s+release\b|\bwingetcreate\b|\bInvoke-(RestMethod|WebRequest)\b', 3),  # publishing
    @('\bRemove-Item\b|\brm\s+-rf?\b|\brmSync\(|\bunlink(Sync)?\(', 2),
    @('\bsecrets\.|\bclient_secret\b|\brefresh_token\b|\bSignTool\b|\bsigntool\b', 2)   # credentials and signing
)

$signalRx = @($signals | ForEach-Object { , @([regex]::new($_[0], 'Compiled'), $_[1]) })

function Get-FileFacts([string]$Path) {
    $text = [System.IO.File]::ReadAllText((Join-Path $root $Path)).Replace("`r`n", "`n")
    $lines = [System.Collections.Generic.List[string]]::new()
    if ($text.Length -gt 0) {
        $lines.AddRange([string[]]$text.Split("`n"))
        if ($text.EndsWith("`n")) { $lines.RemoveAt($lines.Count - 1) }
    }
    $hits = 0
    foreach ($l in $lines) {
        foreach ($s in $signalRx) { if ($s[0].IsMatch($l)) { $hits += $s[1] } }
    }
    $head = @($lines | Select-Object -First 5)
    return [pscustomobject]@{ Lines = $lines.Count; Hits = $hits; Generated = (Test-Generated $Path $head) }
}

# Lines added since the base, per path; a file the base does not have counts whole.
function Get-ChangedMap([string]$BaseRef) {
    & git rev-parse --verify --quiet "$BaseRef^{commit}" *> $null
    if ($LASTEXITCODE -ne 0) {
        Write-Host "${check}: the base commit '$BaseRef' is not in this repository."
        Exit-Verdict $check 2 "unknown base $BaseRef"
    }
    $map = @{}
    foreach ($row in @(& git -c core.quotepath=off diff --numstat --no-renames $BaseRef 2>$null)) {
        $parts = $row -split "`t"
        if ($parts.Count -ge 3 -and $parts[0] -ne '-') { $map[$parts[2]] = [int]$parts[0] }
    }
    $atBase = [System.Collections.Generic.HashSet[string]]::new($ordinal)
    foreach ($f in @(& git -c core.quotepath=off ls-tree -r --name-only $BaseRef)) { [void]$atBase.Add($f) }
    return @{ Added = $map; AtBase = $atBase }
}

# --- units and atoms ---------------------------------------------------------------------------

function Read-ParityPairs {
    $p = Join-Path $root 'configs/parity-map.json'
    if (-not (Test-Path $p)) { return @{} }
    $map = @{}
    $json = Get-Content -Raw $p | ConvertFrom-Json
    foreach ($pair in $json.pairs) {
        $go = @($pair.go)[0]
        $pkg = if ($go.EndsWith('/')) { $go.TrimEnd('/') } else { Get-Dir $go }
        foreach ($js in @($pair.js)) { if (-not $map.ContainsKey($js)) { $map[$js] = $pkg } }
    }
    return $map
}

$releaseParent = 'release path'

function Get-Unit([string]$p, $Pairs) {
    # A page or stylesheet follows the module of the same name (options.html -> options.js).
    $companion = $p -replace '\.(html|css)$', '.js'
    foreach ($q in @($p, $companion)) {
        if ($q -match '^extension/src/' -and $Pairs.ContainsKey($q)) {
            $k = $Pairs[$q]
            return @{ Key = $k; Parent = (Get-Dir $k) }
        }
    }
    if ($p -match '^(cmd|internal|tools)/' -or $p -match '^(tests|extension/test)/') {
        $k = Get-Dir $p
        return @{ Key = $k; Parent = (Get-Dir $k) }
    }
    if ($p -match '^extension/src/') {
        $name = (Get-Leaf $p) -replace '\.[^.]+$', ''
        $fam = ($name -split '-')[0]
        return @{ Key = "extension/src/$fam*"; Parent = 'extension/src' }
    }
    if ($p -match '^scripts/') { return @{ Key = $p; Parent = $releaseParent } }
    if ($p -match '^extension/(manifest\.json|package\.json|build\.mjs|scripts/)') {
        if ((Get-AuditClass $p) -eq 'B') { return @{ Key = 'extension/scripts'; Parent = 'extension' } }
        return @{ Key = 'extension (build + publish)'; Parent = $releaseParent }
    }
    return @{ Key = (Get-Dir $p); Parent = $releaseParent }
}

function Get-AtomKey([string]$p) {
    if ($p -match '^(.*?)(_windows|_nonwindows|_other|_unix|_linux|_darwin|_posix)\.go$') { return $Matches[1] }
    return $p
}

# Splits an ordered atom list into the fewest contiguous parts that keep both limits (an atom over
# a limit is a part of its own), with the largest part as small as possible.
function Split-Atoms($Atoms) {
    $n = $Atoms.Count
    $pre = [int[]]::new($n + 1); $preF = [int[]]::new($n + 1)
    for ($i = 0; $i -lt $n; $i++) { $pre[$i + 1] = $pre[$i] + $Atoms[$i].Lines; $preF[$i + 1] = $preF[$i] + $Atoms[$i].Files }
    $ok = {
        param($a, $b)   # atoms a..b-1
        if ($b - $a -eq 1) { return $true }
        return (($pre[$b] - $pre[$a]) -le $MaxLines) -and (($preF[$b] - $preF[$a]) -le $MaxFiles)
    }
    $total = $pre[$n]
    $kMin = [Math]::Max([Math]::Ceiling($total / $MaxLines), [Math]::Ceiling($preF[$n] / $MaxFiles))
    for ($k = [Math]::Max(1, [int]$kMin); $k -le $n; $k++) {
        # best[j][i]: smallest max-part over atoms 0..i-1 in j parts; cut[j][i]: where part j starts
        $inf = [int]::MaxValue
        $best = New-Object 'int[,]' ($k + 1), ($n + 1)
        $cut = New-Object 'int[,]' ($k + 1), ($n + 1)
        for ($j = 0; $j -le $k; $j++) { for ($i = 0; $i -le $n; $i++) { $best[$j, $i] = $inf } }
        $best[0, 0] = 0
        for ($j = 1; $j -le $k; $j++) {
            for ($i = 1; $i -le $n; $i++) {
                for ($s = $j - 1; $s -lt $i; $s++) {
                    if ($best[($j - 1), $s] -eq $inf) { continue }
                    if (-not (& $ok $s $i)) { continue }
                    $v = [Math]::Max($best[($j - 1), $s], $pre[$i] - $pre[$s])
                    if ($v -lt $best[$j, $i]) { $best[$j, $i] = $v; $cut[$j, $i] = $s }
                }
            }
        }
        if ($best[$k, $n] -eq $inf) { continue }
        $parts = [System.Collections.Generic.List[object]]::new()
        $i = $n
        for ($j = $k; $j -ge 1; $j--) {
            $s = $cut[$j, $i]
            $parts.Insert(0, @($Atoms[$s..($i - 1)]))
            $i = $s
        }
        return , $parts.ToArray()
    }
    return , @(, @($Atoms))
}

function New-Slice($Keys, $Files, [int]$Part, [int]$Parts) {
    $lines = 0; $changed = 0; $hits = 0
    foreach ($f in $Files) { if (-not $f.Generated) { $lines += $f.Lines; $changed += $f.Changed; $hits += $f.Hits } }
    $share = if ($lines -gt 0) { [Math]::Min(1.0, $changed / $lines) } else { 0.0 }
    $risk = [Math]::Round($hits * 1000.0 / [Math]::Max($lines, $MaxLines / 4.0) * (1 + $share), 1)
    $first = $Keys[0]; $last = $Keys[-1]
    $name = if ($Parts -gt 1) { "$first (part $Part of $Parts)" } elseif ($Keys.Count -eq 1) { $first } else { "$first .. $last" }
    return [pscustomobject]@{
        Name = $name; Keys = $Keys; Files = $Files; Lines = $lines; Changed = $changed; Risk = $risk
        Group = $null; Part = $Part; Parts = $Parts; Parent = $null
    }
}

# Builds slices over the given scope files. Returns sibling groups (arrays of slices), unordered.
function Build-Groups([string[]]$Files, $Pairs, $Changed) {
    $units = @{}
    foreach ($f in $Files) {
        $u = Get-Unit $f $Pairs
        if (-not $units.ContainsKey($u.Key)) { $units[$u.Key] = @{ Key = $u.Key; Parent = $u.Parent; Files = [System.Collections.Generic.List[string]]::new() } }
        $units[$u.Key].Files.Add($f)
    }
    $facts = @{}
    foreach ($f in $Files) {
        $x = Get-FileFacts $f
        $c = if ($Changed.Added.ContainsKey($f)) { $Changed.Added[$f] } elseif (-not $Changed.AtBase.Contains($f)) { $x.Lines } else { 0 }
        $facts[$f] = [pscustomobject]@{ Path = $f; Lines = $x.Lines; Hits = $x.Hits; Changed = [Math]::Min($c, $x.Lines); Generated = $x.Generated }
    }
    $groups = [System.Collections.Generic.List[object]]::new()
    $byParent = @{}
    foreach ($k in (Sort-Ordinal @($units.Keys))) {
        $u = $units[$k]
        if (-not $byParent.ContainsKey($u.Parent)) { $byParent[$u.Parent] = [System.Collections.Generic.List[object]]::new() }
        $byParent[$u.Parent].Add($u)
    }
    foreach ($parent in (Sort-Ordinal @($byParent.Keys))) {
        $binKeys = [System.Collections.Generic.List[string]]::new()
        $binFiles = [System.Collections.Generic.List[object]]::new()
        $binLines = 0
        $flush = {
            if ($binKeys.Count -gt 0) {
                $s = New-Slice $binKeys.ToArray() $binFiles.ToArray() 1 1
                $s.Parent = $parent
                $groups.Add(@(, $s))
                $binKeys.Clear(); $binFiles.Clear()
            }
        }
        foreach ($u in $byParent[$parent]) {
            # The unit's own Go files first, then its other own files, then the paired modules.
            $sortedFiles = Sort-Ordinal $u.Files.ToArray()
            $ownDir = $u.Key + '/'
            $ordered = @($sortedFiles | Where-Object { $_.StartsWith($ownDir) -and $_.EndsWith('.go') }) +
                @($sortedFiles | Where-Object { $_.StartsWith($ownDir) -and -not $_.EndsWith('.go') }) +
                @($sortedFiles | Where-Object { -not $_.StartsWith($ownDir) })
            $unitLines = 0; foreach ($f in $ordered) { if (-not $facts[$f].Generated) { $unitLines += $facts[$f].Lines } }
            if ($unitLines -le $MaxLines -and $ordered.Count -le $MaxFiles) {
                if ($binLines + $unitLines -gt $MaxLines -or $binFiles.Count + $ordered.Count -gt $MaxFiles) { & $flush; $binLines = 0 }
                $binKeys.Add($u.Key)
                foreach ($f in $ordered) { $binFiles.Add($facts[$f]) }
                $binLines += $unitLines
                continue
            }
            & $flush; $binLines = 0
            $atomMap = [ordered]@{}
            foreach ($f in $ordered) {
                $ak = Get-AtomKey $f
                if (-not $atomMap.Contains($ak)) { $atomMap[$ak] = [System.Collections.Generic.List[object]]::new() }
                $atomMap[$ak].Add($facts[$f])
            }
            $atoms = foreach ($ak in $atomMap.Keys) {
                $fs = $atomMap[$ak].ToArray()
                $l = 0; foreach ($x in $fs) { if (-not $x.Generated) { $l += $x.Lines } }
                [pscustomobject]@{ Files = $fs.Count; Lines = $l; Facts = $fs }
            }
            $parts = Split-Atoms @($atoms)
            $group = [System.Collections.Generic.List[object]]::new()
            $i = 0
            foreach ($part in $parts) {
                $i++
                $fs = foreach ($a in $part) { $a.Facts }
                $s = New-Slice @($u.Key) @($fs) $i $parts.Count
                $s.Parent = $parent
                $group.Add($s)
            }
            $groups.Add($group.ToArray())
        }
        & $flush
    }
    return , $groups.ToArray()
}

# Orders sibling groups by their highest risk, pins the release path first when the score puts it
# below the median (ticket 34, open question 2), and numbers the slices.
function Set-Order($Groups, [int]$FirstId) {
    $sorted = @($Groups | Sort-Object -Property @{ Expression = { ($_ | Measure-Object Risk -Maximum).Maximum }; Descending = $true }, @{ Expression = { $_[0].Keys[0] }; Descending = $false } -CaseSensitive)
    $flat = @($sorted | ForEach-Object { $_ })
    $pinned = $null
    $rel = @($flat | Where-Object { $_.Parent -eq $releaseParent })
    if ($rel.Count -gt 0 -and $flat.Count -gt 1) {
        $topRel = $rel[0]
        $rank = [Array]::IndexOf($flat, $topRel)
        if ($rank -ge [Math]::Ceiling($flat.Count / 2.0)) {
            $g = $sorted | Where-Object { $_ -contains $topRel } | Select-Object -First 1
            $sorted = @(, $g) + @($sorted | Where-Object { -not ($_ -contains $topRel) })
            $pinned = $topRel
        }
    }
    $id = $FirstId
    $out = [System.Collections.Generic.List[object]]::new()
    foreach ($g in $sorted) {
        $ids = @()
        foreach ($s in $g) { $s | Add-Member -NotePropertyName Id -NotePropertyValue ('S{0:D2}' -f $id) -Force; $s | Add-Member -NotePropertyName Pinned -NotePropertyValue ([bool]($s -eq $pinned)) -Force; $ids += $s.Id; $out.Add($s); $id++ }
        foreach ($s in $g) { $s.Group = @($ids | Where-Object { $_ -ne $s.Id }) }
    }
    return , $out.ToArray()
}

# --- the previous register ---------------------------------------------------------------------

$areaDirs = @{
    P = @('internal/', 'cmd/doc-html-translate/'); G = @('cmd/doc-html-ui/')
    E = @('internal/epub/', 'internal/htmlgen/', 'internal/htmlsplit/', 'internal/htmlconv/', 'internal/md/', 'internal/htmlproc/')
    X = @('internal/pdf/', 'internal/mobi/', 'internal/fb2/', 'internal/rtf/', 'internal/txt/', 'internal/img/', 'internal/comic/')
    T = @('internal/translator/'); O = @('internal/ocr/'); B = @('extension/'); Q = @('')
}

# One entry per id: the tree files its evidence cites, or the folders when the file is gone.
function Read-RegisterIds([string[]]$Tree) {
    $path = Join-Path $root $Register
    if (-not (Test-Path $path)) { return @() }
    $entries = [System.Collections.Generic.List[object]]::new()
    $dups = @{}
    foreach ($line in (Get-Content $path)) {
        if ($line -notmatch '^- ([A-Z])(\d+) - (.*)$') { continue }
        $id = "$($Matches[1])$($Matches[2])"; $area = $Matches[1]; $rest = $Matches[3]
        if ($rest -match '^duplicate of ([A-Z]\d+)') { $dups[$id] = $Matches[1] }
        $files = [System.Collections.Generic.List[string]]::new()
        $dirs = [System.Collections.Generic.List[string]]::new()
        $pending = [System.Collections.Generic.List[string]]::new()
        foreach ($m in [regex]::Matches($rest, '`([^`]+)`')) {
            foreach ($t in [regex]::Matches($m.Groups[1].Value, '[A-Za-z0-9_./-]+\.(go|js|mjs|html|css|ps1|json|iss|ya?ml)\b')) {
                $pending.Add($t.Value)
            }
        }
        foreach ($t in $pending) {
            $c = @($Tree | Where-Object { $_ -eq $t -or $_.EndsWith('/' + $t) })
            if ($c.Count -gt 1) {
                $byArea = @($c | Where-Object { $f = $_; @($areaDirs[$area] | Where-Object { $f.StartsWith($_) }).Count -gt 0 })
                if ($byArea.Count -ge 1) { $c = $byArea }
            }
            if ($c.Count -gt 1 -and $files.Count -gt 0) {
                $near = @($c | Where-Object { $d = Get-Dir $_; @($files | Where-Object { (Get-Dir $_) -eq $d }).Count -gt 0 })
                if ($near.Count -ge 1) { $c = $near }
            }
            if ($c.Count -ge 1) { $files.Add((Sort-Ordinal $c)[0]); continue }
            if ($t.Contains('/')) {
                $d = Get-Dir $t
                $dm = @($Tree | ForEach-Object { Get-Dir $_ } | Where-Object { $_ -eq $d -or $_.EndsWith('/' + $d) } | Select-Object -Unique)
                foreach ($x in $dm) { if (-not $dirs.Contains($x)) { $dirs.Add($x) } }
            }
        }
        $entries.Add([pscustomobject]@{ Id = $id; Files = @($files | Select-Object -Unique); Dirs = $dirs.ToArray() })
    }
    foreach ($e in $entries) {
        if ($dups.ContainsKey($e.Id) -and $e.Files.Count -eq 0 -and $e.Dirs.Count -eq 0) {
            $t = $entries | Where-Object { $_.Id -eq $dups[$e.Id] } | Select-Object -First 1
            if ($t) { $e.Files = $t.Files; $e.Dirs = $t.Dirs }
        }
    }
    return , $entries.ToArray()
}

# --- manifest ----------------------------------------------------------------------------------

function ConvertTo-SliceRecord($s, $Placed) {
    [ordered]@{
        id       = $s.Id
        name     = $s.Name
        units    = @($s.Keys)
        part     = $s.Part
        parts    = $s.Parts
        siblings = @($s.Group)
        pinned   = $s.Pinned
        lines    = $s.Lines
        changed  = $s.Changed
        risk     = $s.Risk
        files    = @($s.Files | ForEach-Object { [ordered]@{ path = $_.Path; lines = $_.Lines; changed = $_.Changed; generated = $_.Generated } })
        recheck  = @(if ($Placed.ContainsKey($s.Id)) { $Placed[$s.Id] })
    }
}

function Get-Placement($Slices, $Entries, [string[]]$Scope) {
    $holder = @{}
    foreach ($s in $Slices) { foreach ($f in $s.Files) { $holder[$f.Path] = $s.Id } }
    $placed = @{}
    foreach ($s in $Slices) { $placed[$s.Id] = [System.Collections.Generic.List[string]]::new() }
    $unplaced = [System.Collections.Generic.List[string]]::new()
    $outside = [System.Collections.Generic.List[string]]::new()
    foreach ($e in $Entries) {
        $ids = [System.Collections.Generic.List[string]]::new()
        foreach ($f in $e.Files) { if ($holder.ContainsKey($f) -and -not $ids.Contains($holder[$f])) { $ids.Add($holder[$f]) } }
        if ($ids.Count -eq 0) {
            foreach ($d in $e.Dirs) {
                foreach ($s in $Slices) {
                    if (@($s.Files | Where-Object { (Get-Dir $_.Path) -eq $d }).Count -gt 0 -and -not $ids.Contains($s.Id)) { $ids.Add($s.Id) }
                }
            }
        }
        if ($ids.Count -gt 0) { foreach ($i in $ids) { $placed[$i].Add($e.Id) } }
        elseif ($e.Files.Count -gt 0) { $outside.Add($e.Id) }      # cites only class-B code (tests, the lab)
        else { $unplaced.Add($e.Id) }                               # cites nothing that is still in the tree
    }
    return @{ Placed = $placed; Unplaced = $unplaced.ToArray(); Outside = $outside.ToArray() }
}

function Get-ManifestText($Slices, $Placement, $Generated) {
    $m = [ordered]@{
        about          = 'Written by scripts/audit-slices.ps1 - do not edit by hand; rerun it. The phase state lives in INDEX.md, the findings in FINDINGS.md.'
        base           = $Base
        maxLines       = $MaxLines
        maxFiles       = $MaxFiles
        includeClassB  = [bool]$IncludeClassB
        register       = $Register
        slices         = @($Slices | ForEach-Object { ConvertTo-SliceRecord $_ $Placement.Placed })
        generated      = @($Generated)
        registerNoCode = @($Placement.Unplaced)
        registerClassB = @($Placement.Outside)
    }
    return ($m | ConvertTo-Json -Depth 8).Replace("`r`n", "`n") + "`n"
}

function Write-SliceTable($Slices) {
    Write-Host ('{0,-4} {1,5} {2,6} {3,6} {4,7}  {5}' -f 'id', 'files', 'lines', 'chg%', 'risk', 'slice')
    foreach ($s in $Slices) {
        $pct = if ($s.Lines -gt 0) { [int](100 * $s.Changed / $s.Lines) } else { 0 }
        $flag = if ($s.Pinned) { '  [pinned: release path]' } else { '' }
        Write-Host ('{0,-4} {1,5} {2,6} {3,6} {4,7}  {5}{6}' -f $s.Id, @($s.Files).Count, $s.Lines, $pct, $s.Risk, $s.Name, $flag)
    }
}

# --- plan scaffolding --------------------------------------------------------------------------

function Get-Slug([string]$Name) {
    $s = ($Name.ToLowerInvariant() -replace '\(part (\d+) of \d+\)', 'p$1' -replace '[^a-z0-9]+', '-').Trim('-')
    if ($s.Length -gt 48) { $s = $s.Substring(0, 48).Trim('-') }
    return $s
}

function Get-PhaseFile($s) { return ('PHASE_{0}__{1}.md' -f $s.Id.Substring(1), (Get-Slug $s.Name)) }

function Get-PhaseText([string]$Template, $s, $Placed) {
    $goPkgs = @($s.Files | Where-Object { $_.Path.EndsWith('.go') } | ForEach-Object { './' + (Get-Dir $_.Path) } | Select-Object -Unique)
    $js = @($s.Files | Where-Object { $_.Path -match '^extension/src/.+\.js$' } | ForEach-Object { $_.Path })
    $other = @($s.Files | Where-Object { -not $_.Path.EndsWith('.go') -and $_.Path -notmatch '^extension/src/.+\.js$' } | ForEach-Object { $_.Path })
    $rows = ($s.Files | ForEach-Object {
            $g = if ($_.Generated) { ' (generated - read at its generator)' } else { '' }
            "| ``$($_.Path)``$g | $($_.Lines) | $($_.Changed) |"
        }) -join "`n"
    $sib = if (@($s.Group).Count -gt 0) { 'Sibling slices of the same unit, adjacent in the order: ' + ((@($s.Group) | ForEach-Object { $_ }) -join ', ') + '. Read their files for context only; audit them in their own phase.' } else { 'None - this slice holds its units whole.' }
    $ids = @(if ($Placed.ContainsKey($s.Id)) { $Placed[$s.Id] })
    $re = if ($ids.Count -gt 0) { ($ids -join ', ') } else { 'none - no 2026-09-24 finding cites these files' }
    $t = $Template
    $vals = [ordered]@{
        '{{PHASE}}' = $s.Id.Substring(1); '{{SLICE}}' = $s.Id; '{{NAME}}' = $s.Name; '{{LINES}}' = "$($s.Lines)"
        '{{FILECOUNT}}' = "$(@($s.Files).Count)"; '{{CHANGED}}' = "$($s.Changed)"; '{{RISK}}' = "$($s.Risk)"; '{{BASE}}' = $Base
        '{{FILES_TABLE}}' = $rows; '{{SIBLINGS}}' = $sib; '{{RECHECK_IDS}}' = $re
        '{{GO_PACKAGES}}' = $(if ($goPkgs.Count) { ($goPkgs -join ' ') } else { '(none)' })
        '{{JS_MODULES}}' = $(if ($js.Count) { ($js -join ' ') } else { '(none)' })
        '{{OTHER_FILES}}' = $(if ($other.Count) { ($other -join ' ') } else { '(none)' })
        '{{PINNED}}' = $(if ($s.Pinned) { 'Pinned first: the release path ranked below the median, so it is read first (ticket 34, open question 2).' } else { '' })
    }
    foreach ($k in $vals.Keys) { $t = $t.Replace($k, $vals[$k]) }
    return $t
}

function Get-IndexRow($s) {
    return ("| {0} | {1} {2} | {3} | {4} | `u{2B1C} Not started | 0/5 | [{5}]({5}) |" -f $s.Id.Substring(1), $s.Id, $s.Name, $s.Lines, $s.Risk, (Get-PhaseFile $s))
}

function Write-Plan($Slices, $Placed, [bool]$Append) {
    $tplPath = Join-Path $root "$Ticket/PHASE_TEMPLATE.md"
    if (-not (Test-Path $tplPath)) {
        Write-Host "${check}: no $Ticket/PHASE_TEMPLATE.md - the phases were not scaffolded."
        return
    }
    $tpl = [System.IO.File]::ReadAllText($tplPath).Replace("`r`n", "`n")
    foreach ($s in $Slices) { Write-Lf "$Ticket/$(Get-PhaseFile $s)" (Get-PhaseText $tpl $s $Placed) }
    $idx = "$Ticket/INDEX.md"
    $rows = ($Slices | ForEach-Object { Get-IndexRow $_ }) -join "`n"
    if ($Append) {
        $text = [System.IO.File]::ReadAllText((Join-Path $root $idx)).Replace("`r`n", "`n")
        $lines = [System.Collections.Generic.List[string]]($text -split "`n")
        $last = -1
        for ($i = 0; $i -lt $lines.Count; $i++) { if ($lines[$i] -match '^\| \d+ \| S\d+ ') { $last = $i } }
        $lines.Insert($last + 1, $rows)
        Write-Lf $idx ($lines -join "`n")
        return
    }
    $text = @"
# Tactical plan: $(Split-Path -Leaf $Ticket)

**Strategic spec:** [``../$(Split-Path -Leaf $Ticket).md``](../$(Split-Path -Leaf $Ticket).md)
**Status:** In Progress
**Phases:** 0 / $($Slices.Count) done
**Last updated:** $(Get-Date -Format 'yyyy-MM-dd')

> Generated by ``scripts/audit-slices.ps1 -Write``; from here on this file is maintained by hand and is
> the authority on phase state. One phase per slice of [``slices.json``](slices.json), in risk order.
> Every phase carries the whole slice procedure; findings go to [``FINDINGS.md``](FINDINGS.md).
> Campaign state: ``./scripts/audit-slices.ps1 -Summary``.

## Phase overview

| # | Phase | Lines | Risk | Status | Steps | File |
|---|-------|------:|-----:|--------|------:|------|
$rows

Legend: `u{2B1C} Not started `u{00B7} `u{1F6A7} In Progress `u{00B7} `u{2705} Done `u{00B7} `u{26D4} Blocked `u{00B7} `u{23ED}`u{FE0F} Skipped

## Completion gate

- [ ] ``./scripts/audit-slices.ps1 -Summary`` ends in ``audit-slices: PASS``.
- [ ] Every crit / high finding in ``FINDINGS.md`` has a package-1 ticket or an inline fix with its run cited.

## Change log

- $(Get-Date -Format 'yyyy-MM-dd') - scaffolded by ``scripts/audit-slices.ps1 -Write``.
"@
    Write-Lf $idx $text
}

# --- summary -----------------------------------------------------------------------------------

function Invoke-Summary {
    $mPath = Join-Path $root "$Ticket/slices.json"
    $iPath = Join-Path $root "$Ticket/INDEX.md"
    if (-not (Test-Path $mPath)) { Exit-Verdict $check 2 "no $Ticket/slices.json - run with -Write first" }
    if (-not (Test-Path $iPath)) { Exit-Verdict $check 2 "no $Ticket/INDEX.md" }
    $m = Get-Content -Raw $mPath | ConvertFrom-Json
    Write-Subject $check "$Ticket ($(@($m.slices).Count) slices, base $($m.base), $(Get-TreeLabel))"
    $open = [System.Collections.Generic.List[string]]::new()

    # coverage
    $tree = Get-TreeFiles
    $scope = Get-ScopeFiles $tree ([bool]$m.includeClassB)
    $count = @{}
    foreach ($s in $m.slices) { foreach ($f in $s.files) { $count[$f.path] = 1 + [int]$count[$f.path] } }
    $scopeSet = [System.Collections.Generic.HashSet[string]]::new([string[]]$scope, $ordinal)
    $uncovered = @($scope | Where-Object { -not $count.ContainsKey($_) })
    $twice = [string[]](Sort-Ordinal @($count.Keys | Where-Object { $count[$_] -gt 1 }))
    $gone = [string[]](Sort-Ordinal @($count.Keys | Where-Object { -not $scopeSet.Contains($_) }))
    Write-Host "coverage: $($scope.Count) class-A files in the tree, $($uncovered.Count) in no slice, $($twice.Count) in two, $($gone.Count) sliced but gone"
    foreach ($f in $uncovered) { Write-Host "  uncovered: $f (cut a tail slice: -Tail)" -ForegroundColor Red }
    foreach ($f in $twice) { Write-Host "  held twice: $f" -ForegroundColor Red }
    foreach ($f in $gone) { Write-Host "  gone since slicing: $f" -ForegroundColor Yellow }
    if ($uncovered.Count) { $open.Add("$($uncovered.Count) uncovered") }
    if ($twice.Count) { $open.Add("$($twice.Count) held twice") }

    # slice state, from INDEX.md
    $state = @{}
    foreach ($line in (Get-Content $iPath -Encoding utf8)) {
        if ($line -match '^\|\s*\d+\s*\|\s*(S\d+)\b') {
            $sid = $Matches[1]
            $state[$sid] = if ($line.Contains("`u{2705}")) { 'done' } elseif ($line.Contains("`u{23ED}")) { 'skipped' } else { 'open' }
        }
    }
    $openSlices = @($m.slices | Where-Object { $state[$_.id] -ne 'done' -and $state[$_.id] -ne 'skipped' } | ForEach-Object { $_.id })
    $noPhase = @($m.slices | Where-Object { -not $state.ContainsKey($_.id) } | ForEach-Object { $_.id })
    $closed = @($m.slices).Count - $openSlices.Count
    Write-Host "slices: $closed closed, $($openSlices.Count) open$(if ($openSlices.Count) { ' (' + ($openSlices -join ' ') + ')' })"
    foreach ($x in $noPhase) { Write-Host "  $x has no phase row in INDEX.md" -ForegroundColor Red }
    if ($openSlices.Count) { $open.Add("$($openSlices.Count) slices open") }

    # findings
    $fPath = Join-Path $root "$Ticket/FINDINGS.md"
    $rechecked = [System.Collections.Generic.HashSet[string]]::new($ordinal)
    $sev = [ordered]@{ crit = 0; high = 0; med = 0; low = 0 }
    $untriaged = [System.Collections.Generic.List[string]]::new()
    $tickets = [System.Collections.Generic.SortedSet[int]]::new()
    $regressed = [System.Collections.Generic.List[string]]::new()
    if (Test-Path $fPath) {
        $section = ''
        foreach ($line in (Get-Content $fPath -Encoding utf8)) {
            if ($line -match '^## (.*)') { $section = $Matches[1]; continue }
            if ($section -like 'Re-check*' -and $line -match '^- ([A-Z]\d+) - (still fixed|regressed|one edition only|superseded) - ') {
                [void]$rechecked.Add($Matches[1])
                if ($Matches[2] -ne 'still fixed' -and $Matches[2] -ne 'superseded') { $regressed.Add("$($Matches[1]) $($Matches[2])") }
                continue
            }
            if ($section -like 'New findings*' -and $line -match '^- ([A-Z]\d+) - (crit|high|med|low) - (conf|plaus)\b') {
                $fid = $Matches[1]; $sv = $Matches[2]
                $sev[$sv]++
                $last = ($line -split ' - ')[-1].Trim()
                # Only the ticket field names tickets - a '#808080' in the finding's text is not one.
                foreach ($t in [regex]::Matches($last, '#(\d+)\b')) { [void]$tickets.Add([int]$t.Groups[1].Value) }
                if (($sv -eq 'crit' -or $sv -eq 'high') -and $last -notmatch '#\d+' -and $last -notmatch '^inline\b') { $untriaged.Add($fid) }
            }
        }
    }
    $want = [System.Collections.Generic.List[string]]::new()
    foreach ($s in $m.slices) { foreach ($i in @($s.recheck)) { if ($i -and -not $want.Contains($i)) { $want.Add($i) } } }
    $missing = @($want | Where-Object { -not $rechecked.Contains($_) })
    Write-Host "re-check: $($want.Count) ids of the previous register in class A, $($want.Count - $missing.Count) re-checked, $($missing.Count) missing"
    if ($missing.Count) { Write-Host "  missing: $($missing -join ' ')" -ForegroundColor Yellow }
    if ($regressed.Count) { Write-Host "  not holding: $($regressed -join ', ')" -ForegroundColor Yellow }
    if (@($m.registerNoCode).Count) { Write-Host "  cite no file still in the tree (re-check by hand): $(@($m.registerNoCode) -join ' ')" -ForegroundColor Yellow }
    if ($missing.Count) { $open.Add("$($missing.Count) re-checks missing") }
    Write-Host "findings: crit $($sev.crit), high $($sev.high), med $($sev.med), low $($sev.low)"
    foreach ($x in $untriaged) { Write-Host "  $x is crit/high with neither a ticket nor an inline fix" -ForegroundColor Red }
    if ($untriaged.Count) { $open.Add("$($untriaged.Count) crit/high untriaged") }

    foreach ($n in $tickets) {
        $pat = '{0:D2}_*.md' -f $n
        $file = @(Get-ChildItem -Path (Join-Path $root 'DEV/plan'), (Join-Path $root 'DEV/plan/done') -Filter $pat -File -ErrorAction SilentlyContinue) | Select-Object -First 1
        if (-not $file) { Write-Host "  ticket #$n - no file" -ForegroundColor Red; $open.Add("ticket #$n missing"); continue }
        $st = (Select-String -Path $file.FullName -Pattern '^\*\*Status:\*\*\s*(.*)$' | Select-Object -First 1)
        $where = if ($file.DirectoryName -like '*done') { 'done/' } else { '' }
        Write-Host ("  ticket #{0:D2} " -f $n) -NoNewline; Write-Host "$where$($file.BaseName) - $(if ($st) { $st.Matches[0].Groups[1].Value } else { 'no status line' })"
    }

    if ($open.Count) { Exit-Verdict $check 1 ('campaign open: ' + ($open -join ', ')) }
    Exit-Verdict $check 0 "campaign closed: $(@($m.slices).Count) slices, $($sev.crit + $sev.high + $sev.med + $sev.low) findings"
}

# --- main --------------------------------------------------------------------------------------

if ($Summary) { Invoke-Summary }

if ($MaxLines -lt 1 -or $MaxFiles -lt 1) { Exit-Verdict $check 2 'the limits must be positive' }
$changed = Get-ChangedMap $Base
$pairs = Read-ParityPairs
$tree = Get-TreeFiles
$scope = Get-ScopeFiles $tree ([bool]$IncludeClassB)
$mPath = "$Ticket/slices.json"

if ($Tail) {
    if (-not (Test-Path (Join-Path $root $mPath))) { Exit-Verdict $check 2 "no $mPath to extend" }
    $old = Get-Content -Raw (Join-Path $root $mPath) | ConvertFrom-Json
    $MaxLines = [int]$old.maxLines; $MaxFiles = [int]$old.maxFiles; $Base = [string]$old.base; $IncludeClassB = [bool]$old.includeClassB
    $changed = Get-ChangedMap $Base
    $scope = Get-ScopeFiles $tree ([bool]$IncludeClassB)
    $held = [System.Collections.Generic.HashSet[string]]::new($ordinal)
    foreach ($s in $old.slices) { foreach ($f in $s.files) { [void]$held.Add($f.path) } }
    $new = @($scope | Where-Object { -not $held.Contains($_) })
    if ($new.Count -eq 0) { Exit-Verdict $check 0 'no uncovered class-A file - no tail slice needed' }
    $next = 1 + (@($old.slices | ForEach-Object { [int]$_.id.Substring(1) }) | Measure-Object -Maximum).Maximum
    $tailSlices = Set-Order (Build-Groups $new $pairs $changed) $next
    foreach ($s in $tailSlices) { $s.Pinned = $false }
    $placement = Get-Placement $tailSlices (Read-RegisterIds $tree) $new
    Write-SliceTable $tailSlices
    $json = Get-Content -Raw (Join-Path $root $mPath) | ConvertFrom-Json -AsHashtable
    $json.slices = @($json.slices) + @($tailSlices | ForEach-Object { ConvertTo-SliceRecord $_ $placement.Placed })
    Write-Lf $mPath ((($json | ConvertTo-Json -Depth 8).Replace("`r`n", "`n")) + "`n")
    if (Test-Path (Join-Path $root "$Ticket/INDEX.md")) { Write-Plan $tailSlices $placement.Placed $true }
    Exit-Verdict $check 0 "$($tailSlices.Count) tail slice(s) appended for $($new.Count) file(s)"
}

$groups = Build-Groups $scope $pairs $changed
$slices = Set-Order $groups 1
$generated = @($slices | ForEach-Object { $_.Files } | Where-Object { $_.Generated } | ForEach-Object { $_.Path })
$entries = Read-RegisterIds $tree
$placement = Get-Placement $slices $entries $scope

Write-Subject $check "$(Get-TreeLabel), class A$(if ($IncludeClassB) { ' + B' }), limits $MaxLines lines / $MaxFiles files, base $Base"
Write-SliceTable $slices
$total = ($slices | Measure-Object Lines -Sum).Sum
Write-Host "total: $($slices.Count) slices, $(@($scope).Count) files, $total lines; $(@($entries).Count) register ids, $(@($placement.Placed.Values | ForEach-Object { $_ } | Select-Object -Unique).Count) placed, $($placement.Outside.Count) class B only, $($placement.Unplaced.Count) cite no file in the tree"

# A slice over a limit that is not a single atom means the partition is wrong.
$bad = @($slices | Where-Object { ($_.Lines -gt $MaxLines -or @($_.Files).Count -gt $MaxFiles) -and @($_.Files | ForEach-Object { Get-AtomKey $_.Path } | Select-Object -Unique).Count -gt 1 })
if ($bad.Count) { Exit-Verdict $check 1 "slice(s) over the limits: $(($bad | ForEach-Object Id) -join ' ')" }

if ($Write) {
    Write-Lf $mPath (Get-ManifestText $slices $placement $generated)
    Write-Host "wrote $mPath"
    if (-not (Test-Path (Join-Path $root "$Ticket/INDEX.md"))) { Write-Plan $slices $placement.Placed $false; Write-Host "scaffolded $Ticket/INDEX.md and $($slices.Count) phase files" }
    else { Write-Host "$Ticket/INDEX.md exists - left as it is (it is the authority on phase state)" }
}
Exit-Verdict $check 0 "$($slices.Count) slices over $(@($scope).Count) files"
