param(
    [string]$Output = "assets/doc-html-translate.ico",
    [string]$Source = "tools/store/logos/boxart-1x1-1080x1080.png"
)

$ErrorActionPreference = "Stop"

# Build a proper multi-resolution .ico from the canonical project logo so Windows
# renders a crisp icon at every size (taskbar 16/24/32, large icons up to 256) instead
# of scaling a single low-res frame. Each frame is stored PNG-compressed (supported by
# Windows Vista+), which keeps the file small and preserves the alpha edges.

Add-Type -AssemblyName System.Drawing

if (-not (Test-Path $Source)) {
    throw "Source logo not found: $Source"
}

$sizes = @(16, 24, 32, 48, 64, 128, 256)

$src = [System.Drawing.Image]::FromFile((Resolve-Path $Source).Path)
try {
    $frames = @()
    foreach ($s in $sizes) {
        $bmp = New-Object System.Drawing.Bitmap($s, $s, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
        $g = [System.Drawing.Graphics]::FromImage($bmp)
        $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
        $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
        $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
        $g.CompositingQuality = [System.Drawing.Drawing2D.CompositingQuality]::HighQuality
        $g.DrawImage($src, (New-Object System.Drawing.Rectangle(0, 0, $s, $s)))
        $g.Dispose()

        $ms = New-Object System.IO.MemoryStream
        $bmp.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png)
        $bmp.Dispose()
        $frames += , @{ Size = $s; Data = $ms.ToArray() }
        $ms.Dispose()
    }
} finally {
    $src.Dispose()
}

New-Item -ItemType Directory -Force -Path (Split-Path -Parent $Output) | Out-Null

$out = [System.IO.File]::Open($Output, [System.IO.FileMode]::Create, [System.IO.FileAccess]::Write)
$bw = New-Object System.IO.BinaryWriter($out)

# ICONDIR: reserved, type (1 = icon), image count.
$bw.Write([UInt16]0)
$bw.Write([UInt16]1)
$bw.Write([UInt16]$frames.Count)

# Frame data starts right after the directory (6 bytes) + one 16-byte entry per frame.
$offset = 6 + 16 * $frames.Count
foreach ($f in $frames) {
    $dim = if ($f.Size -ge 256) { 0 } else { $f.Size } # 0 means 256 in the ICONDIRENTRY
    $bw.Write([byte]$dim)          # width
    $bw.Write([byte]$dim)          # height
    $bw.Write([byte]0)             # color count (0 = truecolor)
    $bw.Write([byte]0)             # reserved
    $bw.Write([UInt16]1)           # color planes
    $bw.Write([UInt16]32)          # bits per pixel
    $bw.Write([UInt32]$f.Data.Length)
    $bw.Write([UInt32]$offset)
    $offset += $f.Data.Length
}
foreach ($f in $frames) {
    $bw.Write($f.Data)
}

$bw.Flush()
$bw.Dispose()
$out.Dispose()

Write-Host "Icon generated: $Output ($($frames.Count) sizes: $($sizes -join ', '))"
