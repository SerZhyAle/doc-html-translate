param(
    [string]$Output = "assets/doc-html-translate.ico"
)

$ErrorActionPreference = "Stop"

Add-Type -AssemblyName PresentationCore
Add-Type -AssemblyName PresentationFramework
Add-Type -AssemblyName WindowsBase

$width = 64
$height = 64

$visual = New-Object System.Windows.Media.DrawingVisual
$ctx = $visual.RenderOpen()

$bg = New-Object System.Windows.Media.SolidColorBrush([System.Windows.Media.Color]::FromRgb(30, 58, 138))
$fg = New-Object System.Windows.Media.SolidColorBrush([System.Windows.Media.Color]::FromRgb(255, 255, 255))

$ctx.DrawRectangle($bg, $null, (New-Object System.Windows.Rect(0, 0, $width, $height)))

$fontFamily = New-Object System.Windows.Media.FontFamily("Segoe UI")
$typeface = New-Object System.Windows.Media.Typeface($fontFamily, [System.Windows.FontStyles]::Normal, [System.Windows.FontWeights]::Bold, [System.Windows.FontStretches]::Normal)

$line1 = New-Object System.Windows.Media.FormattedText(
    "DOC",
    [System.Globalization.CultureInfo]::InvariantCulture,
    [System.Windows.FlowDirection]::LeftToRight,
    $typeface,
    14,
    $fg,
    1.0
)

$line2 = New-Object System.Windows.Media.FormattedText(
    "HTML",
    [System.Globalization.CultureInfo]::InvariantCulture,
    [System.Windows.FlowDirection]::LeftToRight,
    $typeface,
    15,
    $fg,
    1.0
)

$ctx.DrawText($line1, (New-Object System.Windows.Point(7, 14)))
$ctx.DrawText($line2, (New-Object System.Windows.Point(11, 34)))
$ctx.Close()

$rtb = New-Object System.Windows.Media.Imaging.RenderTargetBitmap($width, $height, 96, 96, [System.Windows.Media.PixelFormats]::Pbgra32)
$rtb.Render($visual)

$pngEncoder = New-Object System.Windows.Media.Imaging.PngBitmapEncoder
$pngEncoder.Frames.Add([System.Windows.Media.Imaging.BitmapFrame]::Create($rtb))

$pngStream = New-Object System.IO.MemoryStream
$pngEncoder.Save($pngStream)
$pngBytes = $pngStream.ToArray()
$pngStream.Dispose()

$iconDir = [byte[]](0,0,1,0,1,0)
$entry = New-Object System.IO.MemoryStream
$bw = New-Object System.IO.BinaryWriter($entry)

$bw.Write([byte]$width)
$bw.Write([byte]$height)
$bw.Write([byte]0)
$bw.Write([byte]0)
$bw.Write([UInt16]1)
$bw.Write([UInt16]32)
$bw.Write([UInt32]$pngBytes.Length)
$bw.Write([UInt32]22)
$bw.Flush()
$entryBytes = $entry.ToArray()
$bw.Dispose()
$entry.Dispose()

New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Output) | Out-Null

$outStream = [System.IO.File]::Open($Output, [System.IO.FileMode]::Create, [System.IO.FileAccess]::Write)
$outStream.Write($iconDir, 0, $iconDir.Length)
$outStream.Write($entryBytes, 0, $entryBytes.Length)
$outStream.Write($pngBytes, 0, $pngBytes.Length)
$outStream.Dispose()

Write-Host "Icon generated: $Output"
