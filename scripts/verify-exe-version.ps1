<#
.SYNOPSIS
  Assert that built executables carry the version they were stamped with (BUILD-EVIDENCE rule 2).

.DESCRIPTION
  A stale or unstamped binary agrees with every check that reads the source, because the source is
  exactly what it was built from. This reads the version out of the artifact itself:
    - the Windows version resource goversioninfo wrote (read only where the OS can read it);
    - the `-X main.Version=..` stamp the linker embedded, found as its literal bytes in the binary
      (works on any OS, so the release workflow can run it on its Linux runner with -Expect);
    - for the CLI on Windows, what `doc-html-translate -version` actually prints.
  It fails when the stamp is not linked in (the binary still says the "dev" default the source
  declares), when the readings disagree, when the executables disagree with each other, or when
  -Expect is given and does not match.

  Exit codes follow CHECK-VERDICT: 0 pass, 1 a version is wrong, 2 nothing could be read (no such
  file; no -Expect on an OS that cannot read the resource). Last line:
  "verify-exe-version: PASS|FAIL|COULD NOT VERIFY (..)".

.PARAMETER Path
  One or more executables.

.PARAMETER Expect
  The stamp every executable must carry, e.g. 26.0923.1430.

.EXAMPLE
  ./scripts/verify-exe-version.ps1 -Path build/doc-html-translate.exe, build/doc-html-ui.exe
#>
param(
    [Parameter(Mandatory = $true)][string[]]$Path,
    [string]$Expect
)

$ErrorActionPreference = "Stop"
. "$PSScriptRoot/lib/verdict.ps1"

$check = 'verify-exe-version'
Write-Subject $check (($Path -join ', ') + $(if ($Expect) { ", expected $Expect" } else { '' }))


$problems = @()
$seen = @{}
foreach ($p in $Path) {
    if (-not (Test-Path -LiteralPath $p -PathType Leaf)) {
        Write-Host "$p does not exist."
        Exit-Verdict $check 2 "no file $p"
    }

    # 1. The version resource goversioninfo wrote (FileVersionInfo reads PE resources on Windows only).
    $resource = $null
    if ($IsWindows) {
        $resource = (Get-Item -LiteralPath $p).VersionInfo.ProductVersion
        if ($resource) { $resource = $resource.Trim() }
    }
    $stamp = if ($Expect) { $Expect } else { $resource }
    if (-not $stamp) {
        Write-Host "${p}: no -Expect given and no version resource readable on this OS - nothing to compare."
        Exit-Verdict $check 2 "no stamp to compare for $p"
    }

    # 2. The linked stamp. -trimpath drops -ldflags from the build info `go version -m` prints, so the
    #    `-X main.Version=..` value is found the only portable way: as its literal ASCII bytes in the
    #    binary. The resource copy is UTF-16 and cannot produce this match.
    $bytes = [System.IO.File]::ReadAllBytes((Resolve-Path -LiteralPath $p).Path)
    $linkedFound = [System.Text.Encoding]::Latin1.GetString($bytes).Contains($stamp)

    # 3. The CLI can say its own version - the strongest reading, where the OS can run it.
    $reported = $null
    if ($IsWindows -and (Split-Path -Leaf $p) -like 'doc-html-translate*') {
        $out = (& $p -version 2>&1 | Out-String).Trim()
        if ($out -match '^doc-html-translate (\S+)$') { $reported = $Matches[1] }
    }

    Write-Host ("  {0}: resource={1} linked-stamp={2} reports={3}" -f $p,
        $(if ($resource) { $resource } elseif ($IsWindows) { '(none)' } else { '(not read on this OS)' }),
        $(if ($linkedFound) { 'present' } else { 'ABSENT' }),
        $(if ($reported) { $reported } else { '(not run)' }))

    if ($IsWindows -and -not $resource) { $problems += "${p}: no version resource" }
    if ($resource -and $Expect -and $resource -ne $Expect) { $problems += "${p}: version resource $resource, expected $Expect" }
    if (-not $linkedFound) { $problems += "${p}: the stamp $stamp is not linked in - the binary reports the source default 'dev' or another build's version" }
    if ($reported -and $reported -ne $stamp) { $problems += "${p}: reports $reported, expected $stamp" }
    $seen[$stamp] = $true
}
if ($seen.Count -gt 1) {
    $problems += "the executables carry different versions: $($seen.Keys -join ', ')"
}

if ($problems) {
    foreach ($x in $problems) { Write-Host "  - $x" -ForegroundColor Red }
    Exit-Verdict $check 1 "$($problems.Count)"
}
Exit-Verdict $check 0 "$($Path.Count) executable(s) at $(@($seen.Keys)[0])"
