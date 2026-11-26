<#
.SYNOPSIS
  Compose the Chrome Web Store small promotional tile from the product identity and a real reader capture.
.EXAMPLE
  pwsh -NoProfile -File tools/store/make-chrome-promo.ps1
#>
param([string]$Output = (Join-Path $PSScriptRoot '../../extension/store/chrome-promo-small.png'))
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName PresentationCore, PresentationFramework, WindowsBase

function Read-Image([string]$Path) {
    $image = [Windows.Media.Imaging.BitmapImage]::new()
    $image.BeginInit()
    $image.CacheOption = [Windows.Media.Imaging.BitmapCacheOption]::OnLoad
    $image.UriSource = [Uri]::new([IO.Path]::GetFullPath($Path))
    $image.EndInit()
    $image.Freeze()
    return $image
}

$mark = Read-Image (Join-Path $PSScriptRoot '../../assets/identity/out/social-mark-280.png')
$screen = Read-Image (Join-Path $PSScriptRoot '../../extension/store/screenshots/shot1-en.png')
$visual = [Windows.Media.DrawingVisual]::new()
$ctx = $visual.RenderOpen()
$ctx.DrawRectangle([Windows.Media.Brushes]::MidnightBlue, $null, [Windows.Rect]::new(0, 0, 440, 280))
$ctx.DrawImage($screen, [Windows.Rect]::new(0, 135, 440, 275))
$overlay = [Windows.Media.SolidColorBrush]::new([Windows.Media.Color]::FromArgb(175, 10, 15, 20))
$ctx.DrawRectangle($overlay, $null, [Windows.Rect]::new(0, 135, 440, 145))
$ctx.DrawImage($mark, [Windows.Rect]::new(24, 25, 104, 104))
$typeface = [Windows.Media.Typeface]::new([Windows.Media.FontFamily]::new('Segoe UI'), [Windows.FontStyles]::Normal, [Windows.FontWeights]::Bold, [Windows.FontStretches]::Normal)
$text = [Windows.Media.FormattedText]::new('doc-html-translate', [Globalization.CultureInfo]::InvariantCulture, [Windows.FlowDirection]::LeftToRight, $typeface, 29, [Windows.Media.Brushes]::White, 1.0)
$ctx.DrawText($text, [Windows.Point]::new(145, 56))
$ctx.Close()
$bitmap = [Windows.Media.Imaging.RenderTargetBitmap]::new(440, 280, 96, 96, [Windows.Media.PixelFormats]::Pbgra32)
$bitmap.Render($visual)
$encoder = [Windows.Media.Imaging.PngBitmapEncoder]::new()
$encoder.Frames.Add([Windows.Media.Imaging.BitmapFrame]::Create($bitmap))
$stream = [IO.File]::Open([IO.Path]::GetFullPath($Output), [IO.FileMode]::Create, [IO.FileAccess]::Write)
try { $encoder.Save($stream) } finally { $stream.Dispose() }
Write-Host "Chrome promo tile written: $Output (440x280)"
