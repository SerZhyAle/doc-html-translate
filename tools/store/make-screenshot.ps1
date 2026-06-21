<#
.SYNOPSIS
  Generate Microsoft Store listing screenshots (>= 1366x768 PNG) for doc-html-translate.

.DESCRIPTION
  Self-contained and deterministic - needs no real book and writes nothing tracked:
    1. Writes a neutral public-domain sample (Alice in Wonderland) to a temp folder.
    2. Converts it with the CLI (no translation, no browser) into a reading view + TOC.
    3. Uses headless Edge to capture the reading view and the table of contents at
       1366x768 into tools/store/*.png (the files you upload to Partner Center).

  Re-run any time; the PNGs are overwritten. The CLI is taken from msix/staging
  (built by build-msix.ps1) or build/, or pass -Cli to point at a specific exe.

.EXAMPLE
  .\tools\store\make-screenshot.ps1
#>
param(
    # Path to doc-html-translate.exe. Default: msix/staging then build/.
    [string]$Cli
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$OutDir   = $PSScriptRoot
$WorkDir  = Join-Path $RepoRoot "temp\store-screenshot"

# ── locate the CLI ───────────────────────────────────────────
if (-not $Cli) {
    foreach ($c in @(
        (Join-Path $RepoRoot "msix\staging\doc-html-translate.exe"),
        (Join-Path $RepoRoot "build\doc-html-translate.exe"))) {
        if (Test-Path $c) { $Cli = $c; break }
    }
}
if (-not $Cli -or -not (Test-Path $Cli)) {
    throw "doc-html-translate.exe not found. Build it first (e.g. .\msix\build-msix.ps1) or pass -Cli."
}

# ── locate Edge ──────────────────────────────────────────────
$Edge = @(
    "$env:ProgramFiles\Microsoft\Edge\Application\msedge.exe",
    "${env:ProgramFiles(x86)}\Microsoft\Edge\Application\msedge.exe"
) | Where-Object { Test-Path $_ } | Select-Object -First 1
if (-not $Edge) { throw "msedge.exe not found (needed for headless capture)." }

# ── neutral public-domain sample (long enough to fill the viewport) ──
$sample = @'
# Alice's Adventures in Wonderland

_by Lewis Carroll_

## Chapter I. Down the Rabbit-Hole

Alice was beginning to get very tired of sitting by her sister on the bank, and of having nothing to do: once or twice she had peeped into the book her sister was reading, but it had no pictures or conversations in it, "and what is the use of a book," thought Alice, "without pictures or conversations?"

So she was considering in her own mind (as well as she could, for the hot day made her feel very sleepy and stupid), whether the pleasure of making a daisy-chain would be worth the trouble of getting up and picking the daisies, when suddenly a White Rabbit with pink eyes ran close by her.

There was nothing so very remarkable in that; nor did Alice think it so very much out of the way to hear the Rabbit say to itself, "Oh dear! Oh dear! I shall be late!" But when the Rabbit actually took a watch out of its waistcoat-pocket, and looked at it, and then hurried on, Alice started to her feet, for it flashed across her mind that she had never before seen a rabbit with either a waistcoat-pocket, or a watch to take out of it, and burning with curiosity, she ran across the field after it, and fortunately was just in time to see it pop down a large rabbit-hole under the hedge.

In another moment down went Alice after it, never once considering how in the world she was to get out again.

The rabbit-hole went straight on like a tunnel for some way, and then dipped suddenly down, so suddenly that Alice had not a moment to think about stopping herself before she found herself falling down a very deep well.

Either the well was very deep, or she fell very slowly, for she had plenty of time as she went down to look about her and to wonder what was going to happen next. First, she tried to look down and make out what she was coming to, but it was too dark to see anything; then she looked at the sides of the well, and noticed that they were filled with cupboards and book-shelves: here and there she saw maps and pictures hung upon pegs.

She took down a jar from one of the shelves as she passed; it was labelled "ORANGE MARMALADE," but to her great disappointment it was empty: she did not like to drop the jar for fear of killing somebody underneath, so managed to put it into one of the cupboards as she fell past it.

"Well!" thought Alice to herself, "after such a fall as this, I shall think nothing of tumbling down stairs! How brave they'll all think me at home! Why, I wouldn't say anything about it, even if I fell off the top of the house!" (Which was very likely true.)

## Chapter II. The Pool of Tears

"Curiouser and curiouser!" cried Alice (she was so much surprised, that for the moment she quite forgot how to speak good English); "now I'm opening out like the largest telescope that ever was! Good-bye, feet!" (for when she looked down at her feet, they seemed to be almost out of sight, they were getting so far off).

"Oh, my poor little feet, I wonder who will put on your shoes and stockings for you now, dears? I'm sure I shan't be able! I shall be a great deal too far off to trouble myself about you: you must manage the best way you can; but I must be kind to them," thought Alice, "or perhaps they won't walk the way I want to go!"

## Chapter III. A Caucus-Race and a Long Tale

They were indeed a queer-looking party that assembled on the bank - the birds with draggled feathers, the animals with their fur clinging close to them, and all dripping wet, cross, and uncomfortable.

The first question of course was, how to get dry again: they had a consultation about this, and after a few minutes it seemed quite natural to Alice to find herself talking familiarly with them, as if she had known them all her life.
'@

Remove-Item $WorkDir -Recurse -Force -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force -Path $WorkDir | Out-Null
$srcMd = Join-Path $WorkDir "Alice in Wonderland.md"
Set-Content -LiteralPath $srcMd -Value $sample -Encoding utf8

Write-Host "Converting sample..." -ForegroundColor Cyan
& $Cli -notranslate -noopen -force "$srcMd"
if ($LASTEXITCODE -ne 0) { throw "conversion failed" }

$bookDir = Join-Path $WorkDir "Alice in Wonderland"
function ConvertTo-FileUrl([string]$p) { "file:///" + ($p -replace '\\','/' -replace ' ','%20') }

# page_002 is Chapter I (text-rich reading view); index.html is the TOC.
$shots = @(
    @{ page = "page_002.html"; out = "reading-view.png" },
    @{ page = "index.html";    out = "table-of-contents.png" }
)
Add-Type -AssemblyName System.Drawing
# A throwaway --user-data-dir forces a fresh, isolated browser process so the
# headless screenshot is never handed off to an already-running Edge instance.
$EdgeProfile = Join-Path $WorkDir "edge-profile"
foreach ($s in $shots) {
    $outPath = Join-Path $OutDir $s.out
    Remove-Item $outPath -ErrorAction SilentlyContinue
    $url = ConvertTo-FileUrl (Join-Path $bookDir $s.page)
    & $Edge --headless=new --disable-gpu --no-first-run --hide-scrollbars `
        --user-data-dir="$EdgeProfile" --force-device-scale-factor=1 `
        --window-size=1366,768 "--screenshot=$outPath" $url 2>$null | Out-Null
    for ($i = 0; $i -lt 50 -and -not (Test-Path $outPath); $i++) { Start-Sleep -Milliseconds 100 }
    if (-not (Test-Path $outPath)) { throw "Edge did not produce $($s.out)" }
    $img = [System.Drawing.Image]::FromFile($outPath)
    Write-Host ("  {0,-22} {1}x{2}" -f $s.out, $img.Width, $img.Height) -ForegroundColor Green
    $img.Dispose()
}
Write-Host "Screenshots written to $OutDir" -ForegroundColor Cyan
