# Shared by every gate script: the CHECK-VERDICT exit-code vocabulary, the verdict line and the
# subject banner (BUILD-EVIDENCE rule 1). Dot-source it: . "$PSScriptRoot/lib/verdict.ps1"
#
#   0 PASS                  something was inspected and nothing was wrong
#   1 FAIL                  something was inspected and judged defective
#   2 COULD NOT VERIFY      nothing was inspected: a missing tool, a missing input, a git failure
#   3 PASS WITH ADVISORIES  it ran, found something, and cannot charge it to the caller's change
#
# No other code carries a verdict, and 2 is never a pass. Every script ends in exactly one line
# "<check>: <WORD> [(detail)]" on the success path and on the failure path alike.
#
# Write-Host only - never Write-Error: under $ErrorActionPreference = 'Stop' a Write-Error is a
# terminating error, so the `exit N` after it dies as 1 while the message still prints
# (CHECK-VERDICT rule 3).

function Get-VerdictWord([int]$Code) {
    switch ($Code) {
        0 { 'PASS' }
        1 { 'FAIL' }
        2 { 'COULD NOT VERIFY' }
        3 { 'PASS WITH ADVISORIES' }
        default { 'FAIL' }   # an unknown code is read as not-passed, never as a pass
    }
}

function Write-VerdictLine([string]$Check, [int]$Code, [string]$Detail) {
    $line = "${Check}: $(Get-VerdictWord $Code)"
    if ($Detail) { $line += " ($Detail)" }
    $color = switch ($Code) { 0 { 'Green' } 3 { 'Yellow' } 2 { 'Magenta' } default { 'Red' } }
    Write-Host $line -ForegroundColor $color
}

# Prints the verdict line and exits the calling script with its code.
function Exit-Verdict([string]$Check, [int]$Code, [string]$Detail) {
    Write-VerdictLine $Check $Code $Detail
    exit $Code
}

# The subject banner: what this run judged, printed before anything else so a result is never
# quoted as evidence about something it did not look at.
function Write-Subject([string]$Check, [string]$Subject) {
    Write-Host "${Check}: subject = $Subject" -ForegroundColor Cyan
}

# "tree 41fbc1b" or "tree 41fbc1b + uncommitted changes" - the working tree a check read.
function Get-TreeLabel {
    $head = (& git rev-parse --short HEAD 2>$null)
    if ($LASTEXITCODE -ne 0 -or -not $head) { return 'tree (not a git checkout)' }
    $dirty = (& git status --porcelain 2>$null)
    if ($dirty) { return "tree $($head.Trim()) + uncommitted changes" }
    return "tree $($head.Trim())"
}
