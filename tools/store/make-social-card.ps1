<#
.SYNOPSIS
  Render the social card of doc-html-translate: Open Graph (1200x630) and the GitHub repository
  social preview (1280x640), from one layout.

.DESCRIPTION
  Brand mark (white bold "DOC / HTML" on the brand blue #1E3A8A plate, as in make-logos.ps1), the product
  name and the one-sentence job statement of docs/POSITIONING.md, on the site's dark palette (assets/sza-kit.css
  tokens). WPF in PowerShell, Segoe UI from Windows, no network, no external font; the same input renders the
  same pixels. Output: assets/social-card-1200x630.png and assets/social-card-1280x640.png.

  Social platforms crop the edges, so every glyph sits inside the central $SafeW x $SafeH box. Both sizes use
  that one box, only the canvas around it grows.

.EXAMPLE
  .\tools\store\make-social-card.ps1
#>
param([string]$OutDir = (Join-Path $PSScriptRoot "..\..\assets"))
$ErrorActionPreference = "Stop"
$OutDir = [IO.Path]::GetFullPath($OutDir)
New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

Add-Type -AssemblyName PresentationCore, PresentationFramework, WindowsBase

# Wording: docs/POSITIONING.md "What the product is"; the accent range is the part that file calls free.
$Name    = "doc-html-translate"
$Tagline = "Turn any book, document or comic into a local web page, then translate it with your browser - free, no key, no account."
$Accent  = "free, no key, no account"

# Layout, in pixels of the safe box.
$SafeW = 1100; $SafeH = 520
$Plate = 280; $Gap = 64; $Radius = 44
$TaglineSize = 38; $TaglineLead = 1.3

function New-Brush([int]$Rgb, [int]$Alpha = 255) {
    $c = [Windows.Media.Color]::FromArgb($Alpha, (($Rgb -shr 16) -band 255), (($Rgb -shr 8) -band 255), ($Rgb -band 255))
    $b = New-Object Windows.Media.SolidColorBrush $c
    $b.Freeze()
    $b
}

# A soft elliptical glow; relative coordinates so it scales with the canvas.
function New-Glow([int]$Rgb, [int]$Alpha, [double]$Cx, [double]$Cy, [double]$Rx, [double]$Ry) {
    $r = ($Rgb -shr 16) -band 255; $g = ($Rgb -shr 8) -band 255; $b = $Rgb -band 255
    $br = New-Object Windows.Media.RadialGradientBrush
    $br.Center = New-Object Windows.Point($Cx, $Cy)
    $br.GradientOrigin = $br.Center
    $br.RadiusX = $Rx; $br.RadiusY = $Ry
    [void]$br.GradientStops.Add((New-Object Windows.Media.GradientStop([Windows.Media.Color]::FromArgb($Alpha, $r, $g, $b), 0.0)))
    [void]$br.GradientStops.Add((New-Object Windows.Media.GradientStop([Windows.Media.Color]::FromArgb(0, $r, $g, $b), 1.0)))
    $br.Freeze()
    $br
}

function New-Typeface([Windows.FontWeight]$Weight) {
    New-Object Windows.Media.Typeface((New-Object Windows.Media.FontFamily("Segoe UI")),
        [Windows.FontStyles]::Normal, $Weight, [Windows.FontStretches]::Normal)
}

function New-Text([string]$Text, [double]$Size, [Windows.FontWeight]$Weight, $Brush) {
    New-Object Windows.Media.FormattedText($Text, [Globalization.CultureInfo]::InvariantCulture,
        [Windows.FlowDirection]::LeftToRight, (New-Typeface $Weight), $Size, $Brush, 1.0)
}

# The mark: lines centred by their ink (cap height), not by the font's line box, so the block sits in the
# middle of the plate whatever Segoe UI's ascent and descent are.
function Add-Mark($ctx, [double]$Left, [double]$Top, [double]$Size) {
    $white = New-Brush 0xFFFFFF
    $fill = New-Brush 0x1E3A8A
    $edge = New-Object Windows.Media.Pen((New-Brush 0xFFFFFF 70), 3)
    $edge.Freeze()
    $ctx.DrawRoundedRectangle($fill, $edge, (New-Object Windows.Rect($Left, $Top, $Size, $Size)), $Radius, $Radius)

    $lines = "DOC", "HTML"
    $ref = 100.0
    $widest = ($lines | ForEach-Object { (New-Text $_ $ref ([Windows.FontWeights]::Bold) $white).BuildGeometry((New-Object Windows.Point(0, 0))).Bounds.Width } | Measure-Object -Maximum).Maximum
    $fs = $ref * ($Size * 0.74) / $widest
    $gapPx = $fs * 0.14
    $inks = foreach ($l in $lines) {
        $t = New-Text $l $fs ([Windows.FontWeights]::Bold) $white
        [pscustomobject]@{ Text = $t; Ink = $t.BuildGeometry((New-Object Windows.Point(0, 0))).Bounds }
    }
    $total = ($inks | ForEach-Object { $_.Ink.Height } | Measure-Object -Sum).Sum + $gapPx * ($inks.Count - 1)
    $inkTop = $Top + ($Size - $total) / 2
    foreach ($i in $inks) {
        $inkLeft = $Left + ($Size - $i.Ink.Width) / 2
        $ctx.DrawText($i.Text, (New-Object Windows.Point(($inkLeft - $i.Ink.X), ($inkTop - $i.Ink.Y))))
        $inkTop += $i.Ink.Height + $gapPx
    }
}

function New-CardPng([string]$Path, [int]$W, [int]$H) {
    $visual = New-Object Windows.Media.DrawingVisual
    $ctx = $visual.RenderOpen()
    $full = New-Object Windows.Rect(0, 0, $W, $H)

    # Site tokens: --bg #0a0f0a, --acc #3fb950, --gold #e3b341, --text #f1f5ee, --text-2 #c3ccbc.
    $ctx.DrawRectangle((New-Brush 0x0A0F0A), $null, $full)
    $ctx.DrawRectangle((New-Glow 0x3FB950 90 0.05 0.0 0.75 1.1), $null, $full)
    $ctx.DrawRectangle((New-Glow 0xE3B341 60 1.0 1.0 0.6 0.9), $null, $full)

    $ox = ($W - $SafeW) / 2; $oy = ($H - $SafeH) / 2
    $colX = $ox + $Plate + $Gap; $colW = $SafeW - $Plate - $Gap

    $text = New-Brush 0xF1F5EE
    $probe = New-Text $Name 100 ([Windows.FontWeights]::Bold) $text
    # 4% slack: the glyph's right side bearing and the safe-box edge must not meet.
    $nameSize = [Math]::Min(92.0, 100.0 * $colW * 0.96 / $probe.Width)
    $title = New-Text $Name $nameSize ([Windows.FontWeights]::Bold) $text

    $tag = New-Text $Tagline $TaglineSize ([Windows.FontWeights]::SemiBold) (New-Brush 0xC3CCBC)
    $tag.MaxTextWidth = $colW
    $tag.LineHeight = $TaglineSize * $TaglineLead
    $tag.SetForegroundBrush((New-Brush 0xE3B341), $Tagline.IndexOf($Accent), $Accent.Length)

    $rule = 6; $gapA = 14; $gapB = 28
    $blockH = $title.Height + $gapA + $rule + $gapB + $tag.Height
    $blockTop = $oy + ($SafeH - $blockH) / 2
    $ctx.DrawText($title, (New-Object Windows.Point($colX, $blockTop)))
    $ruleY = $blockTop + $title.Height + $gapA
    $ctx.DrawRoundedRectangle((New-Brush 0x3FB950), $null, (New-Object Windows.Rect($colX, $ruleY, 96, $rule)), 3, 3)
    $ctx.DrawText($tag, (New-Object Windows.Point($colX, ($ruleY + $rule + $gapB))))

    Add-Mark $ctx $ox ($oy + ($SafeH - $Plate) / 2) $Plate
    $ctx.Close()

    $rtb = New-Object Windows.Media.Imaging.RenderTargetBitmap($W, $H, 96, 96, [Windows.Media.PixelFormats]::Pbgra32)
    $rtb.Render($visual)
    $enc = New-Object Windows.Media.Imaging.PngBitmapEncoder
    $enc.Frames.Add([Windows.Media.Imaging.BitmapFrame]::Create($rtb))
    $fs = [IO.File]::Open($Path, [IO.FileMode]::Create, [IO.FileAccess]::Write)
    try { $enc.Save($fs) } finally { $fs.Dispose() }
    Write-Host ("  {0,-28} {1}x{2}  {3:N0} bytes" -f (Split-Path $Path -Leaf), $W, $H, (Get-Item $Path).Length) -ForegroundColor Green
}

Write-Host "Rendering the social card..." -ForegroundColor Cyan
New-CardPng (Join-Path $OutDir "social-card-1200x630.png") 1200 630
New-CardPng (Join-Path $OutDir "social-card-1280x640.png") 1280 640
Write-Host "Cards written to $OutDir" -ForegroundColor Cyan
