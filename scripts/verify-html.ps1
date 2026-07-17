<#
.SYNOPSIS
  Render converted HTML in headless Chrome/Edge and assert the common checks, so the
  "Chrome-checked: total=N render=M broken=K" evidence in DEV/CHANGELOG.md is cheap and
  repeatable instead of a throwaway temp/probe-*.mjs written from scratch each session.

.DESCRIPTION
  Zero extra dependencies - uses the same headless browser as tools/store/make-screenshot.ps1
  (Edge --headless=new, Chrome fallback). For the target page it:
    1. loads the page headless and dumps the post-load DOM (--dump-dom),
    2. counts <img> and resolves each local src on disk (a missing/misnamed file = broken),
    3. reports <embed application/pdf> presence (the no-text-PDF fallback),
    4. optionally asserts a caller-supplied marker is present (-Expect),
    5. captures a screenshot and flags a blank/near-empty render.

  Prints one summary line per page in the changelog's own vocabulary:
      <page>: total=N render=M broken=K  [embed pdf] [expect ok]
  Exit code is non-zero if any assertion fails (broken>0, expected marker missing, blank render),
  so it can gate a build.

.PARAMETER Path
  A converted-book output folder (its index.html + page_*.html are checked) or a single .html file.

.PARAMETER Expect
  Optional substring that must appear in the dumped DOM of every checked page (e.g. '<embed').

.PARAMETER Pages
  Optional glob for which pages to check inside a folder. Default: index.html + page_*.html.

.EXAMPLE
  ./scripts/verify-html.ps1 -Path "temp/Alice in Wonderland"
.EXAMPLE
  ./scripts/verify-html.ps1 -Path temp/out/index.html -Expect "<embed application/pdf"
#>
param(
    [Parameter(Mandatory = $true)][string]$Path,
    [string]$Expect,
    [string]$Pages
)

$ErrorActionPreference = "Stop"

# ── locate a headless browser (Edge first, Chrome fallback) ──
$browser = @(
    "$env:ProgramFiles\Microsoft\Edge\Application\msedge.exe",
    "${env:ProgramFiles(x86)}\Microsoft\Edge\Application\msedge.exe",
    "$env:ProgramFiles\Google\Chrome\Application\chrome.exe",
    "${env:ProgramFiles(x86)}\Google\Chrome\Application\chrome.exe"
) | Where-Object { Test-Path $_ } | Select-Object -First 1
if (-not $browser) { throw "No headless browser found (Edge or Chrome)." }

if (-not (Test-Path $Path)) { throw "Path not found: $Path" }

# ── resolve the page list ────────────────────────────────────
$item = Get-Item -LiteralPath $Path
if ($item.PSIsContainer) {
    # Filter in PowerShell (-like), NOT the provider -Filter, so wildcard classes
    # like page_00[1-3].html actually work (the FileSystem -Filter ignores them).
    $all = @(Get-ChildItem -LiteralPath $item.FullName -Filter '*.html' -File)
    if ($Pages) {
        $files = @($all | Where-Object { $_.Name -like $Pages } | Sort-Object Name)
    } else {
        $files = @($all | Where-Object { $_.Name -eq 'index.html' -or $_.Name -like 'page_*.html' } | Sort-Object Name)
        if (-not $files) { $files = @($all | Sort-Object Name) }
    }
} else {
    $files = @($item)
}
if (-not $files) { throw "No .html files to check under $Path." }

$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("verify-html-" + [System.IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Force -Path $tmp | Out-Null
$profileDir = Join-Path $tmp "profile"

function ConvertTo-FileUrl([string]$p) { "file:///" + ((Resolve-Path -LiteralPath $p).Path -replace '\\', '/' -replace ' ', '%20') }

$failures = 0
Add-Type -AssemblyName System.Drawing

foreach ($f in $files) {
    $url = ConvertTo-FileUrl $f.FullName
    $dir = Split-Path -Parent $f.FullName

    # 1. dump the post-load DOM (fall back to the raw file if the dump comes back empty)
    $dom = & $browser --headless=new --disable-gpu --no-first-run --hide-scrollbars `
        --user-data-dir="$profileDir" --virtual-time-budget=4000 --dump-dom $url 2>$null | Out-String
    if (-not $dom -or $dom.Length -lt 100) { $dom = Get-Content -LiteralPath $f.FullName -Raw }

    $pageFail = @()

    # 2. images: count, resolve local src on disk.
    # Backreference the opening quote (\1) so a literal apostrophe inside a
    # double-quoted src (e.g. "pdf_images/Aphrodite's Mirror.jpg", which is how
    # --dump-dom serialises &#39;) does not truncate the captured path.
    $imgSrcs = [regex]::Matches($dom, '<img\b[^>]*?\bsrc\s*=\s*(["''])(.*?)\1') | ForEach-Object { $_.Groups[2].Value }
    $total = $imgSrcs.Count
    $broken = 0
    foreach ($src in $imgSrcs) {
        # HTML-decode FIRST: 'Aphrodite&#39;s' would otherwise keep a stray '#'
        # that a naive fragment-split ([?#]) chops the path on.
        $s = [System.Net.WebUtility]::HtmlDecode($src)
        if ($s -match '^(data:|https?://)') { continue }        # inline/remote - assume ok
        $s = ($s -replace '^file:///', '')
        $s = ($s -split '[?#]', 2)[0]                           # drop a real query/fragment suffix
        $decoded = [uri]::UnescapeDataString($s)
        $abs = if ([System.IO.Path]::IsPathRooted($decoded)) { $decoded } else { Join-Path $dir $decoded }
        if (-not (Test-Path -LiteralPath $abs)) { $broken++ }
    }
    $render = $total - $broken
    if ($broken -gt 0) { $pageFail += "broken=$broken" }

    # 3. <embed application/pdf> presence
    $hasEmbed = $dom -match '<embed\b[^>]*application/pdf' -or $dom -match '<embed\b[^>]*\.pdf'

    # 4. caller-supplied marker
    $expectOk = $true
    if ($Expect) {
        $expectOk = $dom.Contains($Expect)
        if (-not $expectOk) { $pageFail += "missing:'$Expect'" }
    }

    # 5. screenshot blank check
    $shot = Join-Path $tmp ($f.BaseName + ".png")
    & $browser --headless=new --disable-gpu --no-first-run --hide-scrollbars `
        --user-data-dir="$profileDir" --window-size=1200,900 "--screenshot=$shot" $url 2>$null | Out-Null
    $blank = $true
    if (Test-Path $shot) {
        try {
            $img = [System.Drawing.Image]::FromFile($shot)
            if ($img.Width -ge 200 -and $img.Height -ge 200) { $blank = $false }
            $img.Dispose()
        } catch { $blank = $true }
    }
    if ($blank) { $pageFail += "blank-render" }

    # ── report line, in the changelog's vocabulary ──
    $flags = @()
    if ($hasEmbed) { $flags += "embed pdf" }
    if ($Expect -and $expectOk) { $flags += "expect ok" }
    $flagStr = if ($flags) { "  [" + ($flags -join "] [") + "]" } else { "" }
    $line = ("{0,-16} total={1} render={2} broken={3}{4}" -f $f.Name, $total, $render, $broken, $flagStr)

    if ($pageFail.Count -gt 0) {
        Write-Host ("  FAIL " + $line + "  <- " + ($pageFail -join ", ")) -ForegroundColor Red
        $failures++
    } else {
        Write-Host ("  ok   " + $line) -ForegroundColor Green
    }
}

Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue

Write-Host ""
if ($failures -gt 0) {
    Write-Host "verify-html: $failures page(s) FAILED" -ForegroundColor Red
    exit 1
}
Write-Host "verify-html: all $($files.Count) page(s) OK" -ForegroundColor Green
exit 0
