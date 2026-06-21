<#
.SYNOPSIS
  Generate Microsoft Store logo / display images for doc-html-translate.

.DESCRIPTION
  Renders the same brand mark as the package logos (brand blue #1E3A8A, white
  bold "DOC / HTML") at every size the Store listing offers, auto-fitting the
  text to each aspect ratio. Output: tools/store/logos/*.png.

  All of these are OPTIONAL (the Store falls back to the package tiles). The
  9:16 poster + 1:1 box art are only needed if you opt into Xbox display.

.EXAMPLE
  .\tools\store\make-logos.ps1
#>
$ErrorActionPreference = "Stop"
$OutDir = Join-Path $PSScriptRoot "logos"
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

Add-Type -AssemblyName PresentationCore, PresentationFramework, WindowsBase

# Brand mark: white bold "DOC"/"HTML" on #1E3A8A, auto-sized to fit WxH.
function New-LogoPng([string]$Path, [int]$W, [int]$H, [string[]]$Lines) {
    $bg = New-Object Windows.Media.SolidColorBrush ([Windows.Media.Color]::FromRgb(30,58,138))
    $fg = New-Object Windows.Media.SolidColorBrush ([Windows.Media.Color]::FromRgb(255,255,255))
    $tf = New-Object Windows.Media.Typeface(
        (New-Object Windows.Media.FontFamily("Segoe UI")),
        [Windows.FontStyles]::Normal, [Windows.FontWeights]::Bold, [Windows.FontStretches]::Normal)

    # Measure at a reference size, then scale so the block fits ~80% of each axis.
    $ref = 100.0
    $mk = {
        param($txt, $sz)
        New-Object Windows.Media.FormattedText($txt,
            [Globalization.CultureInfo]::InvariantCulture, [Windows.FlowDirection]::LeftToRight,
            $tf, $sz, $fg, 1.0)
    }
    $refTexts = foreach ($l in $Lines) { & $mk $l $ref }
    $maxW = ($refTexts | Measure-Object Width -Maximum).Maximum
    $gapRef = $ref * 0.05
    $totalH = ($refTexts | Measure-Object Height -Sum).Sum + ($gapRef * ($refTexts.Count - 1))
    $sizeByW = $ref * ($W * 0.80) / $maxW
    $sizeByH = $ref * ($H * 0.80) / $totalH
    $size = [Math]::Max(8, [int][Math]::Min($sizeByW, $sizeByH))

    $visual = New-Object Windows.Media.DrawingVisual
    $ctx = $visual.RenderOpen()
    $ctx.DrawRectangle($bg, $null, (New-Object Windows.Rect(0,0,$W,$H)))
    $texts = foreach ($l in $Lines) { & $mk $l $size }
    $gap = $size * 0.05
    $blockH = ($texts | Measure-Object Height -Sum).Sum + ($gap * ($texts.Count - 1))
    $y = ($H - $blockH) / 2
    foreach ($t in $texts) {
        $x = ($W - $t.Width) / 2
        $ctx.DrawText($t, (New-Object Windows.Point($x, $y)))
        $y += $t.Height + $gap
    }
    $ctx.Close()
    $rtb = New-Object Windows.Media.Imaging.RenderTargetBitmap($W, $H, 96, 96, [Windows.Media.PixelFormats]::Pbgra32)
    $rtb.Render($visual)
    $enc = New-Object Windows.Media.Imaging.PngBitmapEncoder
    $enc.Frames.Add([Windows.Media.Imaging.BitmapFrame]::Create($rtb))
    $fs = [IO.File]::Open($Path, [IO.FileMode]::Create, [IO.FileAccess]::Write)
    try { $enc.Save($fs) } finally { $fs.Dispose() }
    Write-Host ("  {0,-28} {1}x{2}" -f (Split-Path $Path -Leaf), $W, $H) -ForegroundColor Green
}

$mark = @("DOC","HTML")
Write-Host "Generating Store logos..." -ForegroundColor Cyan
# 9:16 Poster art (Xbox) - provide the larger of the two accepted sizes.
New-LogoPng (Join-Path $OutDir "poster-9x16-1440x2160.png") 1440 2160 $mark
# 1:1 Box art (Xbox) - larger accepted size.
New-LogoPng (Join-Path $OutDir "boxart-1x1-2160x2160.png")  2160 2160 $mark
# 1:1 Store display images.
New-LogoPng (Join-Path $OutDir "tile-300x300.png")          300  300  $mark
New-LogoPng (Join-Path $OutDir "tile-150x150.png")          150  150  $mark
New-LogoPng (Join-Path $OutDir "tile-71x71.png")            71   71   $mark
Write-Host "Logos written to $OutDir" -ForegroundColor Cyan
