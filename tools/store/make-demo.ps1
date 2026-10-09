<#
.SYNOPSIS
  Record the silent product demo and encode it for the site (assets/demo.mp4) and the README
  (assets/demo.gif) from one master.

.DESCRIPTION
  1. Builds the CLI into temp/demo with `go build` (not scripts/build.ps1, which copies to a drive
     of the author's machine) and puts the bundled English OCR data next to it.
  2. Runs tools/store/make-demo.mjs: it converts a public-domain sample book and an original comic
     page with that binary and drives the converted pages in headless Chrome, recording a frame
     sequence with exact durations (the master). Nothing on screen is third-party or personal: the
     sources live in temp/demo/work, and the pages show a file name, never a path.
  3. Encodes the master with ffmpeg: demo.mp4 (H.264, yuv420p, 1280x720, no audio track, faststart)
     and demo.gif (800 px wide, at most 15 fps, optimised palette, stepped down until it fits).
  4. Checks both files with ffprobe against the limits below and copies them to -OutDir only if every
     check passes.

  Needs: Go, Node (a global WebSocket, as the other scripts of this repository), ffmpeg + ffprobe,
  Chrome or Edge. Tesseract is optional - without it the comic scene is dropped and said so.

.PARAMETER OutDir
  Where demo.mp4 and demo.gif are written. Default: assets.

.PARAMETER Keep
  Keep temp/demo (the built CLI, the sources, the converted pages, the frames and the master) for
  inspection instead of removing it.

.EXAMPLE
  .\tools\store\make-demo.ps1
  .\tools\store\make-demo.ps1 -Keep -OutDir temp\demo-out
#>
param(
    [string]$OutDir = (Join-Path $PSScriptRoot "..\..\assets"),
    [switch]$Keep
)

$ErrorActionPreference = "Stop"
$RepoRoot = [IO.Path]::GetFullPath((Join-Path $PSScriptRoot "..\.."))
$OutDir   = [IO.Path]::GetFullPath($OutDir)
$Work     = Join-Path $RepoRoot "temp\demo"

# What the site and the README can carry (strategic spec criterion 5; phase 06).
$MaxSeconds = 60
$Mp4MaxBytes = 4MB
$GifMaxBytes = 5MB
$GifWidth = 800
$GifMaxFps = 15

function Invoke-Native([string]$What, [scriptblock]$Command) {
    # ffmpeg and go write progress to stderr; that is not a failure, the exit code is.
    $previous = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try { & $Command } finally { $ErrorActionPreference = $previous }
    if ($LASTEXITCODE -ne 0) { throw "$What failed (exit $LASTEXITCODE)" }
}

function Find-Tool([string]$Name, [string[]]$Fallback = @()) {
    $cmd = Get-Command $Name -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($cmd) { return $cmd.Source }
    foreach ($p in $Fallback) { if ($p -and (Test-Path $p)) { return $p } }
    return $null
}

# ---- tools ----------------------------------------------------------------------------------
$winGetLinks = Join-Path $env:LOCALAPPDATA "Microsoft\WinGet\Links"
$go      = Find-Tool "go"
$node    = Find-Tool "node"
$ffmpeg  = Find-Tool "ffmpeg"  @((Join-Path $winGetLinks "ffmpeg.exe"))
$ffprobe = Find-Tool "ffprobe" @((Join-Path $winGetLinks "ffprobe.exe"))
$tesseract = if ($env:DOCHT_TESSERACT -and (Test-Path $env:DOCHT_TESSERACT)) { $env:DOCHT_TESSERACT }
             else { Find-Tool "tesseract" @("$env:ProgramFiles\Tesseract-OCR\tesseract.exe") }
foreach ($t in @(@("go", $go), @("node", $node), @("ffmpeg", $ffmpeg), @("ffprobe", $ffprobe))) {
    if (-not $t[1]) { throw "$($t[0]) not found on PATH - the demo cannot be recorded or encoded without it." }
}
if (-not $tesseract) { Write-Host "tesseract not found - the comic scene will be dropped." -ForegroundColor Yellow }

# ---- workspace and the real binary ----------------------------------------------------------
Remove-Item $Work -Recurse -Force -ErrorAction SilentlyContinue
$FramesDir = Join-Path $Work "frames"
$OutWork   = Join-Path $Work "out"
New-Item -ItemType Directory -Force -Path $Work, $OutWork | Out-Null
$Cli = Join-Path $Work "doc-html-translate.exe"

Write-Host "Building the CLI into temp\demo.." -ForegroundColor Cyan
$manifest = Join-Path $RepoRoot "winget\SerZhyAle.DocHtmlTranslate.yaml"
$version = "demo"
if (Test-Path $manifest) {
    $m = Select-String -Path $manifest -Pattern '^PackageVersion:\s*"?([^"\s]+)"?' | Select-Object -First 1
    if ($m) { $version = $m.Matches[0].Groups[1].Value }
}
$savedArch, $savedOs = $env:GOARCH, $env:GOOS
Push-Location $RepoRoot
try {
    # amd64 is what the release binary is, and it avoids the 32-bit address-space ceiling of a 386 host.
    $env:GOARCH = "amd64"; $env:GOOS = "windows"
    Invoke-Native "go build" { & $go build -trimpath -ldflags "-s -w -X main.Version=$version" -o $Cli ./cmd/doc-html-translate }
} finally {
    $env:GOARCH = $savedArch; $env:GOOS = $savedOs
    Pop-Location
}
$tessdata = Join-Path $RepoRoot "build\tessdata\eng.traineddata"
if (Test-Path $tessdata) {
    New-Item -ItemType Directory -Force -Path (Join-Path $Work "tessdata") | Out-Null
    Copy-Item $tessdata (Join-Path $Work "tessdata") -Force
} else {
    Write-Host "build\tessdata\eng.traineddata not found - the comic scene will be dropped." -ForegroundColor Yellow
    $tesseract = $null
}

# ---- record the master ----------------------------------------------------------------------
Write-Host "Recording the master.." -ForegroundColor Cyan
$nodeArgs = @((Join-Path $PSScriptRoot "make-demo.mjs"), "--cli", $Cli, "--work", (Join-Path $Work "work"), "--frames", $FramesDir)
if ($tesseract) { $nodeArgs += @("--tesseract", $tesseract) }
Invoke-Native "make-demo.mjs" { & $node @nodeArgs }
$scenes = Get-Content (Join-Path $FramesDir "scenes.json") -Raw | ConvertFrom-Json

# ---- encode ---------------------------------------------------------------------------------
$master = Join-Path $OutWork "master.mkv"
Write-Host "Encoding the lossless master.." -ForegroundColor Cyan
# -t pins the master to the recorded length: the concat demuxer would otherwise hold the last frame
# well past its duration. Lossless RGB H.264 keeps the master exact and, the frames being mostly
# unchanged, small.
$masterSeconds = ([double]$scenes.seconds).ToString("0.000", [Globalization.CultureInfo]::InvariantCulture)
Invoke-Native "master encode" {
    & $ffmpeg -y -hide_banner -loglevel error -f concat -safe 0 -i (Join-Path $FramesDir "frames.txt") `
        -t $masterSeconds -vf "fps=30,format=rgb24" -c:v libx264rgb -qp 0 -preset veryfast -an $master
}

$mp4 = Join-Path $OutWork "demo.mp4"
foreach ($crf in 20, 23, 26, 29, 32) {
    Write-Host "Encoding demo.mp4 (crf $crf).." -ForegroundColor Cyan
    # Limited-range BT.709, tagged, so a player decodes the colours the capture drew.
    Invoke-Native "mp4 encode" {
        & $ffmpeg -y -hide_banner -loglevel error -i $master `
            -vf "scale=1280:720:flags=lanczos:out_color_matrix=bt709:out_range=tv,format=yuv420p" `
            -c:v libx264 -preset slow -crf $crf -profile:v high -pix_fmt yuv420p -r 30 `
            -colorspace bt709 -color_primaries bt709 -color_trc bt709 -color_range tv `
            -movflags +faststart -an $mp4
    }
    if ((Get-Item $mp4).Length -le $Mp4MaxBytes) { break }
}

$gif = Join-Path $OutWork "demo.gif"
# 14 fps first, not 15: a GIF frame delay is a whole number of centiseconds, and the 6 cs a 15 fps
# delay rounds to plays at 16.7 fps, over the limit. Each later plan trades smoothness for size.
$gifPlans = @(@(14, 256), @(12, 256), @(10, 256), @(10, 160), @(8, 128))
foreach ($plan in $gifPlans) {
    $fps, $colors = $plan
    Write-Host "Encoding demo.gif ($fps fps, $colors colours).." -ForegroundColor Cyan
    $vf = "fps=$fps,scale=${GifWidth}:-1:flags=lanczos,split[a][b];" +
          "[a]palettegen=stats_mode=diff:max_colors=$colors[p];" +
          "[b][p]paletteuse=dither=sierra2_4a:diff_mode=rectangle"
    Invoke-Native "gif encode" { & $ffmpeg -y -hide_banner -loglevel error -i $master -vf $vf -loop 0 $gif }
    if ((Get-Item $gif).Length -le $GifMaxBytes) { break }
}

# The landing's <video> poster: one frame of the contents scene (15 s in), shown before the visitor plays.
$poster = Join-Path $OutWork "demo-poster.jpg"
Invoke-Native "poster" { & $ffmpeg -y -hide_banner -loglevel error -ss 15 -i $master -frames:v 1 -q:v 4 $poster }

# ---- verify, then publish to -OutDir --------------------------------------------------------
function Get-Probe([string]$Path) {
    $json = & $ffprobe -v error -show_format -show_streams -of json $Path | Out-String
    $json | ConvertFrom-Json
}

function Test-FastStart([string]$Path) {
    $fs = [IO.File]::OpenRead($Path)
    try {
        $buf = New-Object byte[] 65536
        $n = $fs.Read($buf, 0, $buf.Length)
        $text = [Text.Encoding]::Latin1.GetString($buf, 0, $n)
        $moov = $text.IndexOf("moov"); $mdat = $text.IndexOf("mdat")
        return ($moov -ge 0 -and ($mdat -lt 0 -or $moov -lt $mdat))
    } finally { $fs.Dispose() }
}

$failures = @()
function Assert-That([bool]$Ok, [string]$Message) { if (-not $Ok) { $script:failures += $Message } }

$p = Get-Probe $mp4
$v = $p.streams | Where-Object codec_type -eq "video" | Select-Object -First 1
$mp4Seconds = [double]$p.format.duration
$mp4Bytes = (Get-Item $mp4).Length
Assert-That ($mp4Seconds -lt $MaxSeconds) "mp4 lasts $mp4Seconds s, the limit is $MaxSeconds"
Assert-That (-not ($p.streams | Where-Object codec_type -eq "audio")) "mp4 carries an audio stream"
Assert-That ($v.codec_name -eq "h264" -and $v.pix_fmt -eq "yuv420p") "mp4 is $($v.codec_name)/$($v.pix_fmt), expected h264/yuv420p"
Assert-That ($v.width -eq 1280 -and $v.height -eq 720) "mp4 is $($v.width)x$($v.height), expected 1280x720"
Assert-That ($mp4Bytes -le $Mp4MaxBytes) "mp4 is $mp4Bytes bytes, the limit is $Mp4MaxBytes"
Assert-That (Test-FastStart $mp4) "mp4 has its moov atom after the media data (not faststart)"

$g = Get-Probe $gif
$gv = $g.streams | Where-Object codec_type -eq "video" | Select-Object -First 1
$gifBytes = (Get-Item $gif).Length
$gifSeconds = [double]$g.format.duration
$num, $den = $gv.avg_frame_rate -split "/"
$gifFps = [double]$num / [double]$den
Assert-That ($gv.width -eq $GifWidth) "gif is $($gv.width) px wide, expected $GifWidth"
Assert-That ($gifFps -le $GifMaxFps) "gif runs at $gifFps fps, the limit is $GifMaxFps"
Assert-That ($gifBytes -le $GifMaxBytes) "gif is $gifBytes bytes, the limit is $GifMaxBytes"
Assert-That ($gifSeconds -lt $MaxSeconds) "gif lasts $gifSeconds s, the limit is $MaxSeconds"

Write-Host ""
Write-Host "Scenes: $(($scenes.scenes | ForEach-Object { '{0} {1:N1}s' -f $_.name, $_.seconds }) -join ', ')"
foreach ($d in $scenes.dropped) { Write-Host "Dropped: $($d.name) - $($d.reason)" -ForegroundColor Yellow }
Write-Host ("demo.mp4  {0:N0} bytes  {1:N1} s  {2}x{3} {4}/{5}  audio streams: {6}" -f $mp4Bytes, $mp4Seconds, $v.width, $v.height, $v.codec_name, $v.pix_fmt, @($p.streams | Where-Object codec_type -eq "audio").Count)
Write-Host ("demo.gif  {0:N0} bytes  {1:N1} s  {2}x{3}  {4:N0} fps" -f $gifBytes, $gifSeconds, $gv.width, $gv.height, $gifFps)

if ($failures.Count -gt 0) {
    $failures | ForEach-Object { Write-Host "FAIL: $_" -ForegroundColor Red }
    throw "the demo does not meet its limits; nothing was written to $OutDir (work kept in $Work)"
}

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
Copy-Item $mp4 (Join-Path $OutDir "demo.mp4") -Force
Copy-Item $gif (Join-Path $OutDir "demo.gif") -Force
Copy-Item $poster (Join-Path $OutDir "demo-poster.jpg") -Force
Write-Host "Demo written to $OutDir" -ForegroundColor Cyan

if (-not $Keep) { Remove-Item $Work -Recurse -Force -ErrorAction SilentlyContinue }
else { Write-Host "Frames, master and sources kept in $Work" }
