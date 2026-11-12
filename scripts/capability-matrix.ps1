<#
.SYNOPSIS
  The capability matrix check (SITE-REPRESENTATION rules 3 and 9): docs/capabilities.json against
  the facts the app itself states, and the three rendered pages against the source, byte for byte.
  Ends in one CHECK-VERDICT line.

.DESCRIPTION
  The matrix is generated, never hand-written (ticket E0130, question 5a): docs/capabilities.json
  holds the operating modes and the input formats with their runtime dependencies in en, ru and uk,
  and this script renders capability-matrix.html, capability-matrix.ru.html and
  capability-matrix.uk.html from it. The whole page is the render - there is no hand-written shell -
  so a capability cannot be typed onto the site by hand.

  The source against the app (a capability question is answered from the matrix, so the matrix may
  not claim what the app denies):
    - README.md states the same dependency facts (Calibre for MOBI/AZW3, 7-Zip for CBR/CB7 and not
      for CBZ/CBT, Tesseract for OCR, the bundled JPEG XL decoder);
    - docs/security-posture.json declares the network surfaces the translation row names
      (translation.googleapis.com, localhost:11434);
    - internal/jxl exists, so the in-process JPEG XL decoder claim has code behind it;
    - every display string obeys the house text style (no long dash, no ellipsis).

  The pages against the source
    - each page equals its render byte for byte, so a hand-edited page or a stale render fails here.

  Exit codes follow CHECK-VERDICT (scripts/lib/verdict.ps1): 0 PASS, 1 FAIL, 2 COULD NOT VERIFY
  (no git checkout, no source).

.PARAMETER Render
  Rewrite the three pages from docs/capabilities.json first, then run the check.

.EXAMPLE
  ./scripts/capability-matrix.ps1            # the check, as scripts/check.ps1 runs it
.EXAMPLE
  ./scripts/capability-matrix.ps1 -Render    # after changing docs/capabilities.json
#>
param(
    [switch]$Render
)

$ErrorActionPreference = "Stop"
. "$PSScriptRoot/lib/verdict.ps1"

$SourcePath = 'docs/capabilities.json'
$Base = 'https://serzhyale.github.io/doc-html-translate'
$Pages = [ordered]@{
    'en' = 'capability-matrix.html'
    'ru' = 'capability-matrix.ru.html'
    'uk' = 'capability-matrix.uk.html'
}

$root = (& git rev-parse --show-toplevel 2>$null)
if ($LASTEXITCODE -ne 0 -or -not $root) {
    Write-Host "capability-matrix: not inside a git checkout - the source cannot be located."
    Exit-Verdict 'capability-matrix' 2 'no git checkout'
}
$root = $root.Trim()
Set-Location $root
Write-Subject 'capability-matrix' "$SourcePath against the app's own facts, and the three pages against it ($(Get-TreeLabel))"

if (-not (Test-Path -LiteralPath $SourcePath)) {
    Write-Host "capability-matrix: $SourcePath is missing."
    Exit-Verdict 'capability-matrix' 2 "no $SourcePath"
}

$errors = [System.Collections.Generic.List[string]]::new()
function Add-Finding([string]$Message) { $script:errors.Add($Message) }

# ── the source ────────────────────────────────────────────────
try { $data = [IO.File]::ReadAllText((Resolve-Path -LiteralPath $SourcePath), [Text.Encoding]::UTF8) | ConvertFrom-Json }
catch {
    Write-Host "capability-matrix: $SourcePath is not valid JSON: $($_.Exception.Message)" -ForegroundColor Red
    Exit-Verdict 'capability-matrix' 1 'source is not JSON'
}
if ($data.schema -ne 1) { Add-Finding "${SourcePath}: schema must be 1" }

$modes = @($data.modes)
$formats = @($data.formats)
if (-not $modes.Count) { Add-Finding "${SourcePath}: no mode rows" }
if (-not $formats.Count) { Add-Finding "${SourcePath}: no format rows" }

# the house text style over every display string (DOCUMENTATION_CONCEPT section 5)
$styleRx = "$([char]0x2014)|$([char]0x2026)|\.\.\."
foreach ($row in $modes) {
    foreach ($field in @('name', 'audience', 'capabilities', 'dependencies')) {
        foreach ($lang in $Pages.Keys) {
            $v = [string]$row.$field.$lang
            if ([string]::IsNullOrWhiteSpace($v)) { Add-Finding "${SourcePath} mode $($row.id): $field.$lang is empty" }
            elseif ($v -match $styleRx) { Add-Finding "${SourcePath} mode $($row.id): $field.$lang breaks the house text style" }
        }
    }
}
foreach ($row in $formats) {
    foreach ($lang in $Pages.Keys) {
        $n = [string]$row.name.$lang
        $d = [string]$row.dependency.$lang
        if ([string]::IsNullOrWhiteSpace($n)) { Add-Finding "${SourcePath} format $($row.id): name.$lang is empty" }
        elseif ($n -match $styleRx) { Add-Finding "${SourcePath} format $($row.id): name.$lang breaks the house text style" }
        if ([string]::IsNullOrWhiteSpace($d)) { Add-Finding "${SourcePath} format $($row.id): dependency.$lang is empty" }
        elseif ($d -match $styleRx) { Add-Finding "${SourcePath} format $($row.id): dependency.$lang breaks the house text style" }
    }
    if (-not @($row.extensions).Count) { Add-Finding "${SourcePath} format $($row.id): no extensions" }
}

# ── the source against the app ────────────────────────────────
# A capability question is answered from the matrix, so the matrix must not claim a dependency the
# app's own surfaces deny - and every dependency it names must be stated there too.
$readmePath = 'README.md'
if (Test-Path -LiteralPath $readmePath) {
    $readme = [IO.File]::ReadAllText((Resolve-Path -LiteralPath $readmePath), [Text.Encoding]::UTF8)
    $need = [ordered]@{
        'Calibre for MOBI/AZW3'          = 'MOBI/AZW3: requires \[Calibre\]'
        '7-Zip for CBR/CB7'              = 'CBR/CB7 comics: require \[7-Zip\]'
        'CBZ and CBT need nothing extra' = 'CBZ and CBT need nothing extra'
        'bundled JPEG XL decoder'        = 'JPEG XL \(`\.jxl`\) pictures are decoded locally with the bundled codec'
        'external Tesseract for OCR'     = 'OCR needs a `tesseract` engine'
    }
    foreach ($k in $need.Keys) {
        if ($readme -notmatch $need[$k]) { Add-Finding "${SourcePath}: the '$k' fact is not stated in README.md any more - agree the two in one edit" }
    }
} else {
    Add-Finding "$readmePath is missing - the matrix facts cannot be held against the app"
}

$posturePath = 'docs/security-posture.json'
if (Test-Path -LiteralPath $posturePath) {
    $postureText = [IO.File]::ReadAllText((Resolve-Path -LiteralPath $posturePath), [Text.Encoding]::UTF8)
    foreach ($surface in @('translation.googleapis.com', 'localhost:11434')) {
        if (-not $postureText.Contains($surface)) { Add-Finding "${SourcePath}: the translation row names $surface, which docs/security-posture.json does not declare" }
    }
} else {
    Add-Finding "$posturePath is missing - the translation row's network claims cannot be held"
}

if (-not (Test-Path -LiteralPath 'internal/jxl')) {
    Add-Finding "${SourcePath}: the JPEG XL row says the decoder is bundled with the app, but internal/jxl does not exist"
}

# ── the render ────────────────────────────────────────────────
$Chrome = @{
    en = @{
        title       = 'doc-html-translate capability matrix'
        desc        = 'The operating modes of doc-html-translate against what they run on, and every input format against the helper it needs - if it needs one.'
        h1          = 'Capability matrix'
        lead        = 'The operating modes against what they run on, and every input format against the helper it needs - if it needs one. The page is generated from the app&rsquo;s own facts and is held against them by the build&rsquo;s checks.'
        modesH2     = 'Operating modes and their dependencies'
        modesHead   = @('Mode', 'What it is for', 'What it does', 'What it runs on')
        formatsH2   = 'Formats and the helpers they need'
        formatsHead = @('Format', 'File extensions', 'What it needs')
        notesTag    = 'Good to know'
        docsLink    = 'Full documentation'
        navDocs     = 'Docs'
        navAria     = 'Main navigation'
        skipLink    = 'Skip to content'
        themeLabel  = 'Switch theme'
        topLabel    = 'Scroll to top'
        htmlLang    = 'en'
        dataLang    = 'en'
        docHref     = 'docs.html'
    }
    ru = @{
        title       = 'Матрица возможностей doc-html-translate'
        desc        = 'Режимы работы doc-html-translate и то, на чём они работают; каждый входной формат и вспомогательная программа, которая ему нужна.'
        h1          = 'Матрица возможностей'
        lead        = 'Режимы работы против того, на чём они работают, и каждый входной формат против программы, которая ему нужна - если нужна. Страница собирается из собственных фактов приложения и удерживается против них проверками сборки.'
        modesH2     = 'Режимы работы и их зависимости'
        modesHead   = @('Режим', 'Для чего он', 'Что умеет', 'На чём работает')
        formatsH2   = 'Форматы и нужные им программы'
        formatsHead = @('Формат', 'Расширения файлов', 'Что нужно')
        notesTag    = 'Полезно знать'
        docsLink    = 'Полная документация'
        navDocs     = 'Справка'
        navAria     = 'Основная навигация'
        skipLink    = 'К содержимому'
        themeLabel  = 'Переключить тему'
        topLabel    = 'В начало страницы'
        htmlLang    = 'ru'
        dataLang    = 'ru'
        docHref     = 'docs.ru.html'
    }
    uk = @{
        title       = 'Матриця можливостей doc-html-translate'
        desc        = 'Режими роботи doc-html-translate і те, на чому вони працюють; кожен вхідний формат і допоміжна програма, яка йому потрібна.'
        h1          = 'Матриця можливостей'
        lead        = 'Режими роботи проти того, на чому вони працюють, і кожен вхідний формат проти програми, яка йому потрібна - якщо потрібна. Сторінка збирається з власних фактів застосунку й утримується проти них перевірками збірки.'
        modesH2     = 'Режими роботи та їхні залежності'
        modesHead   = @('Режим', 'Для чого він', 'Що вміє', 'На чому працює')
        formatsH2   = 'Формати та потрібні їм програми'
        formatsHead = @('Формат', 'Розширення файлів', 'Що потрібно')
        notesTag    = 'Корисно знати'
        docsLink    = 'Повна документація'
        navDocs     = 'Довідка'
        navAria     = 'Основна навігація'
        skipLink    = 'До змісту'
        themeLabel  = 'Змінити тему'
        topLabel    = 'На початок сторінки'
        htmlLang    = 'uk'
        dataLang    = 'ua'
        docHref     = 'docs.uk.html'
    }
}

$SiteStrings = @{
    en = @{ copied = 'Copied'; toLight = 'Switch to light theme'; toDark = 'Switch to dark theme'; toTop = 'Scroll to top' }
    ru = @{ copied = 'Скопировано'; toLight = 'Светлая тема'; toDark = 'Тёмная тема'; toTop = 'В начало страницы' }
    uk = @{ copied = 'Скопійовано'; toLight = 'Світла тема'; toDark = 'Темна тема'; toTop = 'На початок сторінки' }
}

function ConvertTo-HtmlText([string]$s) { return $s.Replace('&', '&amp;').Replace('<', '&lt;').Replace('>', '&gt;') }

function Build-Page([string]$Lang) {
    $c = $Chrome[$Lang]
    $url = "$Base/$($Pages[$Lang])"

    $modeRows = ""
    foreach ($m in $modes) {
        $modeRows += "          <tr><th scope=`"row`">$(ConvertTo-HtmlText ([string]$m.name.$Lang))</th><td>$(ConvertTo-HtmlText ([string]$m.audience.$Lang))</td><td>$(ConvertTo-HtmlText ([string]$m.capabilities.$Lang))</td><td>$(ConvertTo-HtmlText ([string]$m.dependencies.$Lang))</td></tr>`n"
    }
    $formatRows = ""
    foreach ($f in $formats) {
        $exts = (@($f.extensions) | ForEach-Object { "<code dir=`"ltr`">$(ConvertTo-HtmlText ([string]$_))</code>" }) -join ', '
        $formatRows += "          <tr><th scope=`"row`">$(ConvertTo-HtmlText ([string]$f.name.$Lang))</th><td>$exts</td><td>$(ConvertTo-HtmlText ([string]$f.dependency.$Lang))</td></tr>`n"
    }

    $alt = ""
    foreach ($l in $Pages.Keys) { $alt += "  <link rel=`"alternate`" hreflang=`"$l`" href=`"$Base/$($Pages[$l])`">`n" }
    $alt += "  <link rel=`"alternate`" hreflang=`"x-default`" href=`"$Base/$($Pages['en'])`">"

    $head = @('en', 'ru', 'uk') | ForEach-Object {
        $loc = @{ en = 'en_US'; ru = 'ru_RU'; uk = 'uk_UA' }[$_]
        if ($_ -eq $Lang) { "  <meta property=`"og:locale`" content=`"$loc`">" } else { "  <meta property=`"og:locale:alternate`" content=`"$loc`">" }
    }

    $strings = $SiteStrings[$Lang]
    $jsonLdDescription = ConvertTo-HtmlText $c.desc
    $jsonLdTitle = ConvertTo-HtmlText $c.title

return @"
<!doctype html>
<html lang="$($c.htmlLang)" data-lang="$($c.dataLang)" data-theme="dark">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <meta name="theme-color" content="#0a0f0a">
  <link rel="canonical" href="$url">
$alt
  <meta property="og:type" content="article">
  <meta property="og:site_name" content="doc-html-translate">
$($head -join "`n")
  <meta property="og:url" content="$url">
  <meta property="og:title" content="$jsonLdTitle">
  <meta property="og:description" content="$jsonLdDescription">
  <meta property="og:image" content="$Base/assets/social-card-1200x630.png">
  <meta property="og:image:width" content="1200">
  <meta property="og:image:height" content="630">
  <meta property="og:image:alt" content="doc-html-translate logo and the line: turn any book, document or comic into a local web page, then translate it with your browser">
  <meta name="twitter:card" content="summary_large_image">
  <meta name="twitter:title" content="$jsonLdTitle">
  <meta name="twitter:description" content="$jsonLdDescription">
  <meta name="twitter:image" content="$Base/assets/social-card-1200x630.png">
  <script type="application/ld+json">
  {
    "@context": "https://schema.org",
    "@type": "TechArticle",
    "headline": "$jsonLdTitle",
    "url": "$url",
    "inLanguage": "$($c.htmlLang)",
    "description": "$jsonLdDescription",
    "image": "$Base/assets/social-card-1200x630.png",
    "author": { "@type": "Person", "name": "Serhii Zhyhunenko", "url": "https://sza.od.ua/" },
    "isPartOf": { "@type": "WebSite", "name": "doc-html-translate", "url": "$Base/" }
  }
  </script>
  <link rel="icon" href="assets/doc-html-translate.ico">
  <link rel="preconnect" href="https://fonts.googleapis.com"><link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@400;500;600;700;800&family=Plus+Jakarta+Sans:wght@300;400;500;600;700&display=swap" rel="stylesheet">
  <script>
(function(){try{
  var t=localStorage.getItem('sza-theme');
  if(t!=='dark'&&t!=='light')t=matchMedia('(prefers-color-scheme: light)').matches?'light':'dark';
  document.documentElement.setAttribute('data-theme',t);
  var l=localStorage.getItem('sza-lang');
  if(l==='uk')l='ua';
  if(l!=='ru'&&l!=='en'&&l!=='ua'){var n=(navigator.language||'en').toLowerCase();l=n.indexOf('ru')==0?'ru':n.indexOf('uk')==0?'ua':'en';}
  document.documentElement.setAttribute('data-lang',l);
}catch(e){}})();
  </script>
  <script>document.documentElement.setAttribute('data-lang','$($c.dataLang)');</script>
  <link rel="stylesheet" href="assets/sza-kit.css">
  <link rel="stylesheet" href="assets/site.css">
  <title>$jsonLdTitle</title>
  <meta name="description" content="$jsonLdDescription">
</head>
<body>
  <a class="skip-link" href="#top">$($c.skipLink)</a>
  <div class="bg-blobs" aria-hidden="true"><i class="blob blob-1"></i><i class="blob blob-2"></i><i class="blob blob-3"></i></div>
  <header class="site-header">
    <a class="brand" href="#top" aria-label="doc-html-translate"><span class="brand-mark" aria-hidden="true"><span>DOC<br>HTML</span></span><span dir="ltr">doc-html-translate<span class="dot">.</span></span></a>
    <button class="theme-btn" id="themeBtn" type="button" aria-label="$($c.themeLabel)">◐</button>
    <nav class="site-nav" aria-label="$($c.navAria)"><a class="nav-link" href="$($c.docHref)">$($c.navDocs)</a></nav>
  </header>

  <main class="container" id="top" tabindex="-1">
    <h1>$($c.h1)</h1>
    <p class="lead">$($c.lead)</p>

    <section class="section" id="modes">
      <h2 class="section-title"><span class="h">#</span> $($c.modesH2)</h2>
      <div class="table-wrap">
        <table>
          <thead>
            <tr><th scope="col">$($c.modesHead[0])</th><th scope="col">$($c.modesHead[1])</th><th scope="col">$($c.modesHead[2])</th><th scope="col">$($c.modesHead[3])</th></tr>
          </thead>
          <tbody>
$modeRows        </tbody>
        </table>
      </div>
    </section>

    <section class="section" id="formats">
      <h2 class="section-title"><span class="h">#</span> $($c.formatsH2)</h2>
      <div class="table-wrap">
        <table>
          <thead>
            <tr><th scope="col">$($c.formatsHead[0])</th><th scope="col">$($c.formatsHead[1])</th><th scope="col">$($c.formatsHead[2])</th></tr>
          </thead>
          <tbody>
$formatRows        </tbody>
        </table>
      </div>
    </section>

    <div class="note"><div class="tag">$($c.notesTag)</div><p>$(ConvertTo-HtmlText ([string]$data.notes.$Lang))</p></div>

    <p><a class="btn btn-ghost btn-sm" href="$($c.docHref)">$($c.docsLink)</a></p>
  </main>

  <footer class="site-footer"><div class="container"><h2>More tools by SZA</h2><div class="tools-grid"><a href="https://serzhyale.github.io/FastMediaSorter_mob_v2/" target="_blank" rel="noopener"><b>Fast Media Sorter &amp; Organizer</b><span>Android media sorter</span></a><a href="https://serzhyale.github.io/FastMediaSorter_Lite/" target="_blank" rel="noopener"><b>Fast Media Sorter &amp; Sharing for Windows</b><span>Windows media and file manager</span></a><a href="https://serzhyale.github.io/CyrFlip/" target="_blank" rel="noopener"><b>CyrFlip</b><span>Windows layout fixer</span></a><a href="https://serzhyale.github.io/FileDO/" target="_blank" rel="noopener"><b>FileDO</b><span>Windows storage CLI</span></a><a href="https://serzhyale.github.io/StreamsPlayer/" target="_blank" rel="noopener"><b>STREAMS Player</b><span>Windows stream player</span></a><a href="https://serzhyale.github.io/OneClickRunner/" target="_blank" rel="noopener"><b>OneClickRunner</b><span>Windows tray launcher</span></a><a href="https://serzhyale.github.io/universal-agent-kit/" target="_blank" rel="noopener"><b>Universal Agent Kit</b><span>AI-dev methodology</span></a><a href="https://sza.od.ua" target="_blank" rel="noopener"><b>SZA</b><span>Portfolio</span></a></div><div class="footer-bottom"><span>© 2026 <a href="https://sza.od.ua">Serhii Zhyhunenko</a></span><span><a href="https://sza.od.ua">sza.od.ua</a> · <a href="https://github.com/SerZhyAle">SerZhyAle on GitHub</a> · <a href="mailto:sza@ukr.net">sza@ukr.net</a> · <a href="https://github.com/SerZhyAle/doc-html-translate/issues">GitHub Issues</a> · <a href="support.html">Support</a></span></div></div></footer>
  <button class="to-top" id="toTop" type="button" aria-label="$($c.topLabel)"><svg class="glyph" viewBox="0 0 24 24" width="20" height="20" aria-hidden="true" focusable="false"><path fill="currentColor" d="M8,11h3v10h2V11h3l-4,-4 -4,4zM4,3v2h16V3H4z"/></svg></button>
  <script>window.SITE={strings:{page:{"title": "$jsonLdTitle", "desc": "$jsonLdDescription", "copied": "$($strings.copied)", "toLight": "$($strings.toLight)", "toDark": "$($strings.toDark)", "toTop": "$($strings.toTop)"}}};</script>
  <script src="assets/site.js"></script>
</body>
</html>
"@
}

$renders = @{}
foreach ($l in $Pages.Keys) { $renders[$l] = (Build-Page $l) -replace "`r`n", "`n" }

if ($Render) {
    foreach ($l in $Pages.Keys) {
        $p = $Pages[$l]
        [IO.File]::WriteAllText((Join-Path $root $p), $renders[$l], [Text.UTF8Encoding]::new($false))
        Write-Host "capability-matrix: rendered $p." -ForegroundColor Cyan
    }
}

foreach ($l in $Pages.Keys) {
    $p = $Pages[$l]
    if (-not (Test-Path -LiteralPath $p)) { Add-Finding "$p does not exist - run scripts/capability-matrix.ps1 -Render"; continue }
    $current = [IO.File]::ReadAllText((Resolve-Path -LiteralPath $p), [Text.Encoding]::UTF8).Replace("`r`n", "`n")
    if ($current -ne $renders[$l]) { Add-Finding "$p differs from the render of $SourcePath (run scripts/capability-matrix.ps1 -Render; never edit the page by hand)" }
}

if ($errors.Count) {
    foreach ($e in $errors) { Write-Host "capability-matrix: $e" -ForegroundColor Red }
    Exit-Verdict 'capability-matrix' 1 "$($errors.Count) finding(s)"
}
Exit-Verdict 'capability-matrix' 0 ''
