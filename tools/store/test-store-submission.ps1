<#
.SYNOPSIS
  Stub-proven refusals of tools/store/submit-submission.ps1 (WINDOWS-STORE rung 5, stub half).

.DESCRIPTION
  WINDOWS-STORE section D, conformance rung 5: for each of rules 23 (a) to (d) and 25 "a negative
  run against a stub of the Store ends non-zero and writes nothing" - the stub half is owed by
  whoever builds a channel, and this test is that half for ticket E0156. Every case runs the real
  step against stubs of the msstore CLI, the canon preflight and the package build, and asserts
  three things: the exit code, the refusal or completion line, and the stub's call log - a write
  verb that must not have run (the commit, the metadata update, a delete) must not be in it.

  What it proves:

    - a malformed tag and an identity file with no StoreId refuse before anything is invoked;
    - a preflight refusal (rule 23 (b) through its refusal slug) refuses the run and invokes
      nothing at all;
    - an unavailable channel (rule 21) falls back to the manual path: package and listing folder
      built, instructions printed, exit 2, nothing written to the Store;
    - a pending submission the channel's own record does not hold is refused with the
      owner-clears-it line and never deleted or replaced (rule 23 (a), (f)), while a draft the
      channel's own record holds is resumed;
    - a missing deck, and an over-cap @@ReleaseNotes, refuse before any Store call (rules
      9, 12, 16);
    - a Store refusal of the upload or of the metadata update stops the run (rule 23 (d));
    - a draft that reads back without the uploaded package version, a listing language that is
      missing, or notes that differ from the decks refuse the run (criterion (c)) - and a
      read-back that disagrees keeps -Commit from committing;
    - the happy prepare sends the whole draft JSON back with only the thirteen ReleaseNotes fields
      replaced (rule 24: the stub's draft carries a description per language, and it must survive
      unchanged in the payload), leaves the draft without committing (rule 22 without -Commit),
      and writes the rule-26 record and the DEV/releases/history.jsonl + RELEASES.md event;
    - -Commit commits behind the proof, records the status, and writes the submitted ledger event.

  The stubs emulate the measured output defect of msstore 0.3.9.7: every payload is wrapped at 70
  columns and every unicode escape carries a raw line break inside it, exactly the "н\u 043E"
  shape the contract records, so a case only passes through the contract's raw-CR/LF repair
  ("Reading the CLI's output" of WINDOWS-STORE 0.4).

  Isolation: the step's -MsixOut and -VerdictsDir parameters point every case at this scratch
  folder, so no stub write ever touches msix/out or DEV/release-verdicts; the release-ledger files
  a case still appends to (DEV/releases/history.jsonl and RELEASES.md, through scripts/release-ledger.ps1)
  are snapshotted and restored around every case, and the decks it reads are the committed real decks
  (read-only) or scratch copies under -ListingDir.

  Exit codes: 0 every case passed; 1 a case failed. The last line is "cases=<N> failed=<M>".

.PARAMETER Work
  Scratch folder. Default: a new folder under the system temp folder, removed at the end. A caller
  that passes one owns its removal.

.EXAMPLE
  pwsh -NoProfile -File tools\store\test-store-submission.ps1
#>
param([string]$Work)

$ErrorActionPreference = 'Stop'
$Repo = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$Step = Join-Path $PSScriptRoot 'submit-submission.ps1'

$ownsWork = -not $Work
if ($ownsWork) { $Work = Join-Path ([System.IO.Path]::GetTempPath()) ('store-submission-test-' + [guid]::NewGuid().ToString('N').Substring(0, 8)) }
New-Item -ItemType Directory -Force -Path $Work | Out-Null

# -- sandbox: every path the step may write lives under $Work --
$testOut      = Join-Path $Work 'out'
$testVerdicts = Join-Path $Work 'verdicts'
New-Item -ItemType Directory -Force -Path $testOut, $testVerdicts | Out-Null
$expectedMsix = Join-Path $testOut 'SZA.Doc-HTML-Translate_26.1117.1450.0_x64.msix'
$import       = Join-Path $testOut 'store-import'

# The release-ledger files a case appends to (via the real scripts/release-ledger.ps1).
$realHistory      = Join-Path $Repo 'DEV/releases/history.jsonl'
$realReleasesMd   = Join-Path $Repo 'DEV/releases/RELEASES.md'
$historyBackup    = Join-Path $Work 'history.jsonl.bak'
$releasesMdBackup = Join-Path $Work 'RELEASES.md.bak'
if (Test-Path -LiteralPath $realHistory)    { Copy-Item -LiteralPath $realHistory -Destination $historyBackup -Force }
if (Test-Path -LiteralPath $realReleasesMd) { Copy-Item -LiteralPath $realReleasesMd -Destination $releasesMdBackup -Force }

$msstoreLog   = Join-Path $Work 'msstore-log.txt'
$msixLog      = Join-Path $Work 'msix-log.txt'
$metaFile     = Join-Path $Work 'metadata-sent.json'
$cfgFile      = Join-Path $Work 'stub-config.json'
$pfStub       = Join-Path $Work 'stub-preflight.ps1'
$cliStub      = Join-Path $Work 'stub-msstore.ps1'
$buildStub    = Join-Path $Work 'stub-msixbuild.ps1'
$decksDir     = Join-Path $Work 'decks'

# -- the preflight stub: exit 0 / 1 (rule 23 refusal slugs) / 2 (rule 21 manual) --
@'
param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Rest)
$mode = $env:SZA_SUB_TEST_PF
switch ($mode) {
    'pass' { break }
    'version' {
        Write-Output 'refused: version-not-greater'
        Write-Output '{"checkedAt":"2026-11-17T00:00:00.0000000Z","mode":"refused","refusal":"version-not-greater","detail":"package version 26.1117.1450.0 is not greater than the live one"}'
        exit 1
    }
    'pending-foreign' {
        Write-Output 'refused: pending-submission'
        Write-Output '{"checkedAt":"2026-11-17T00:00:00.0000000Z","mode":"refused","refusal":"pending-submission","submissionId":"111111111111","detail":"status PendingCommit"}'
        exit 1
    }
    'pending-own' {
        Write-Output 'refused: pending-submission'
        Write-Output '{"checkedAt":"2026-11-17T00:00:00.0000000Z","mode":"refused","refusal":"pending-submission","submissionId":"222222222222","detail":"status PendingCommit"}'
        exit 1
    }
    'manual' {
        Write-Output 'the Store channel is unavailable (the credential is missing or expired); the Store step is manual and the release goes on'
        exit 2
    }
    default { Write-Output "stub-preflight: unknown mode '$mode'"; exit 9 }
}
Write-Output 'preflight-stub: PASS - last published submission, no pending one, price Free, candidate above every live version'
Write-Output '{"checkedAt":"2026-11-17T00:00:00.0000000Z","mode":"ready","submissionId":"123456789012","status":"Published","pricing":"Free"}'
exit 0
'@ | Set-Content -LiteralPath $pfStub -Encoding utf8

# -- the msstore stub: a call log, per-mode refusals, and the wrapped output defect --
@'
param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Rest)
# Emulates msstore 0.3.9.7's output defect: wrap the payload at 70 columns and break every
# unicode escape with a raw line break, so only the contract's de-wrap repair can parse it.
function Emit-Payload([string]$Json) {
    $broken = $Json -replace '\\u([0-9a-fA-F]{4})', ('\u' + "`n" + 'u$1')
    $out = New-Object System.Text.StringBuilder
    for ($i = 0; $i -lt $broken.Length; $i += 70) {
        [void]$out.Append($broken.Substring($i, [Math]::Min(70, $broken.Length - $i)))
        [void]$out.Append("`n")
    }
    Write-Output $out.ToString()
}
$log = $env:SZA_SUB_TEST_MSSTORE_LOG
$cfg = Get-Content -LiteralPath $env:SZA_SUB_TEST_MSSTORE_CFG -Raw | ConvertFrom-Json
$mode = [string]$cfg.mode
if ($Rest.Count -ge 2 -and $Rest[1] -eq 'updateMetadata') {
    [System.IO.File]::WriteAllText($env:SZA_SUB_TEST_MSSTORE_META, $Rest[3])
    Add-Content -LiteralPath $log -Value 'submission updateMetadata'
    if ($mode -eq 'fail-update') { Write-Output 'error: the submission metadata was refused (mock 422)'; exit 4 }
    Write-Output 'Submission metadata updated.'
    exit 0
}
$verb = if ($Rest[0] -eq 'submission') { "submission $($Rest[1])" } elseif ($Rest[0] -eq 'publish') { 'publish -nc' } else { "$($Rest[0])" }
Add-Content -LiteralPath $log -Value ("$verb " + (@($Rest | Select-Object -Skip 2) -join ' '))
switch ($verb) {
    'publish -nc' {
        if ($mode -eq 'fail-publish') { Write-Output 'error: the upload was refused (mock 409)'; exit 3 }
        Write-Output 'Publishing without commit... Submission created and kept in draft.'
        exit 0
    }
    'submission get' {
        $metaPath = $env:SZA_SUB_TEST_MSSTORE_META
        if (Test-Path -LiteralPath $metaPath) {
            $draft = [System.IO.File]::ReadAllText($metaPath) | ConvertFrom-Json
            if ($mode -eq 'readbackbad') { $draft.listings.de.releaseNotes = 'mutated ' + $draft.listings.de.releaseNotes }
            Emit-Payload ($draft | ConvertTo-Json -Depth 100 -Compress)
            exit 0
        }
        $langs = @('en-us','ru','uk','de','it','es','fr','pt-br','ar','hi','bn','ur','zh-hans') | Where-Object { $_ -ne $cfg.dropLanguage }
        $listings = @{}
        foreach ($l in $langs) { $listings[$l] = @{ description = "keep me $l"; releaseNotes = "старые notes $l line one`r`nline two" } }
        $draft = @{ id = $cfg.submissionId; status = 'PendingCommit'; priceId = 'Free'
            applicationPackages = @(@{ fileName = 'SZA.Doc-HTML-Translate_26.1117.1450.0_x64.msix'; version = $cfg.draftVersion })
            listings = $listings }
        Emit-Payload ($draft | ConvertTo-Json -Depth 100 -Compress)
        exit 0
    }
    'submission publish' {
        if ($mode -eq 'fail-commit') { Write-Output 'error: the commit was refused (mock 409: the current state of the app forbids it)'; exit 5 }
        Write-Output 'Committing the submission for certification...'
        exit 0
    }
    'submission poll' { Write-Output 'Polling... PUBLISHED'; exit 0 }
    'submission status' { Write-Output ([string]$cfg.statusText); exit 0 }
    default { Add-Content -LiteralPath $log -Value "UNEXPECTED-VERB $verb"; Write-Output "stub: unexpected call $verb"; exit 40 }
}
exit 0
'@ | Set-Content -LiteralPath $cliStub -Encoding utf8

# -- the package build stub: writes into the sandboxed msix folder --
@'
param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Rest)
Add-Content -LiteralPath $env:SZA_SUB_TEST_MSIX_LOG -Value ($Rest -join ' ')
if ($env:SZA_SUB_TEST_MSIX -eq 'fail') { Write-Output 'build refused: HEAD is not the tag commit or the tree is dirty'; exit 5 }
[System.IO.File]::WriteAllText($env:SZA_SUB_TEST_MSIX_OUT, 'stub-msix-bytes')
Write-Output 'stub package build OK'
exit 0
'@ | Set-Content -LiteralPath $buildStub -Encoding utf8

$env:SZA_SUB_TEST_PF = 'pass'
$env:SZA_SUB_TEST_MSSTORE_LOG = $msstoreLog
$env:SZA_SUB_TEST_MSSTORE_CFG = $cfgFile
$env:SZA_SUB_TEST_MSSTORE_META = $metaFile
$env:SZA_SUB_TEST_MSIX_LOG = $msixLog
$env:SZA_SUB_TEST_MSIX = 'ok'
$env:SZA_SUB_TEST_MSIX_OUT = $expectedMsix

function Restore-LedgerSnapshot {
    if (Test-Path -LiteralPath $historyBackup)    { Copy-Item -LiteralPath $historyBackup -Destination $realHistory -Force }
    if (Test-Path -LiteralPath $releasesMdBackup) { Copy-Item -LiteralPath $releasesMdBackup -Destination $realReleasesMd -Force }
}
function Reset-Case {
    Remove-Item -LiteralPath $msstoreLog, $msixLog, $metaFile, $cfgFile -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $testOut -Recurse -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $testVerdicts -Recurse -Force -ErrorAction SilentlyContinue
    New-Item -ItemType Directory -Force -Path $testOut, $testVerdicts | Out-Null
    Restore-LedgerSnapshot
}
function Set-StubConfig([string]$Mode, [string]$SubmissionId = '333333333333', [string]$DraftVersion = '26.1117.1450.0', [string]$DropLanguage = '', [string]$StatusText = 'CommitStarted') {
    @{ mode = $Mode; submissionId = $SubmissionId; draftVersion = $DraftVersion; dropLanguage = $DropLanguage; statusText = $StatusText } |
        ConvertTo-Json -Compress | Set-Content -LiteralPath $cfgFile -Encoding utf8
}
function Invoke-Step([hashtable]$Params) {
    # *>&1: the step reports through Write-Host, and the cases assert on the printed lines.
    # -MsixOut and -VerdictsDir are always the sandbox: no case may write to msix/out or
    # DEV/release-verdicts.
    $Params['MsixOut'] = $testOut
    $Params['VerdictsDir'] = $testVerdicts
    $out = @(& $Step @Params *>&1 | ForEach-Object { "$_" })
    return @{ Text = ($out -join "`n"); Code = $LASTEXITCODE }
}
function LogLines {
    if (Test-Path -LiteralPath $msstoreLog) { return @(Get-Content -LiteralPath $msstoreLog | Where-Object { $_.Trim() }) }
    return @()
}
function VerdictText {
    if (Test-Path -LiteralPath $testVerdicts) {
        $rows = @(Get-ChildItem -LiteralPath $testVerdicts -Filter '*.store-submission.jsonl' | ForEach-Object { Get-Content -LiteralPath $_.FullName })
        return ($rows -join "`n")
    }
    return ''
}
function LedgerHistoryText {
    if (Test-Path -LiteralPath $realHistory) { return (Get-Content -LiteralPath $realHistory -Raw) }
    return ''
}
function Read-DeckReleaseNotes([string]$Path) {
    $name = $null; $buf = New-Object System.Collections.ArrayList
    foreach ($line in [System.IO.File]::ReadAllLines($Path)) {
        if ($line -match '^@@(\w+)\s*$') {
            if ($name -eq 'ReleaseNotes') { return ($buf -join "`n").Trim() }
            $name = $Matches[1]; $buf.Clear()
        } elseif ($name -and $line -notmatch '^#') { [void]$buf.Add($line) }
    }
    if ($name -eq 'ReleaseNotes') { return ($buf -join "`n").Trim() }
    return $null
}

$script:total = 0
$script:failed = 0
$script:caseFailed = $false
function Check($Cond, [string]$Why) {
    if ([bool]$Cond) { return }
    $script:caseFailed = $true
    Write-Host "    FAIL: $Why" -ForegroundColor Red
}
function Start-Case([string]$Name) {
    Reset-Case
    $script:total++
    $script:caseFailed = $false
    Write-Host "  case $($script:total): $Name" -ForegroundColor Cyan
}
function End-Case {
    if ($script:caseFailed) { $script:failed++; Write-Host "    -> case $($script:total) FAILED" -ForegroundColor Red }
    Restore-LedgerSnapshot
}

$localeMap = @{ en = 'en-us'; ru = 'ru'; uk = 'uk'; de = 'de'; it = 'it'; es = 'es'; fr = 'fr'; pt = 'pt-br'; ar = 'ar'; hi = 'hi'; bn = 'bn'; ur = 'ur'; zh = 'zh-hans' }

Write-Host 'store-submission stub test (WINDOWS-STORE rung 5, stub half; ticket E0156)' -ForegroundColor Cyan

# -- 1: a malformed tag refuses before anything is invoked --
Start-Case 'malformed tag refuses with nothing invoked'
$r = Invoke-Step @{ Tag = 'vbad'; MsstoreCommand = $cliStub; PreflightCommand = $pfStub; IdentityFile = (Join-Path $Repo 'msix/identity.json') }
Check ($r.Code -eq 1) "exit 1 expected, was $($r.Code)"
Check ($r.Text -match 'is not a v<yy\.MMdd\.HHmm> app tag') 'the refusal names the tag shape'
Check (@(LogLines).Count -eq 0) 'no msstore call at all'
Check (-not (Test-Path -LiteralPath $msixLog)) 'no package build ran'
End-Case

# -- 2: an identity file with no StoreId refuses before anything is invoked (rule 19) --
Start-Case 'identity file with no StoreId refuses (rule 19)'
'{}' | Set-Content -LiteralPath (Join-Path $Work 'identity-noid.json') -Encoding utf8
$r = Invoke-Step @{ Tag = 'v26.1117.1450'; MsstoreCommand = $cliStub; PreflightCommand = $pfStub; IdentityFile = (Join-Path $Work 'identity-noid.json') }
Check ($r.Code -eq 1) "exit 1 expected, was $($r.Code)"
Check ($r.Text -match 'no usable StoreId') 'the refusal names the identity file'
Check (@(LogLines).Count -eq 0) 'no msstore call at all'
Check (-not (Test-Path -LiteralPath $msixLog)) 'no package build ran'
End-Case

# -- 3: a preflight refusal refuses the run and invokes nothing (rule 23 (b) via the slug) --
Start-Case 'preflight refusal (version not greater) invokes nothing'
$env:SZA_SUB_TEST_PF = 'version'
$r = Invoke-Step @{ Tag = 'v26.1117.1450'; MsstoreCommand = $cliStub; PreflightCommand = $pfStub }
$env:SZA_SUB_TEST_PF = 'pass'
Check ($r.Code -eq 1) "exit 1 expected, was $($r.Code)"
Check ($r.Text -match 'the preflight refused this run: version-not-greater') 'the refusal names the preflight refusal'
Check (@(LogLines).Count -eq 0) 'no msstore call at all'
Check (-not (Test-Path -LiteralPath $msixLog)) 'no package build ran'
End-Case

# -- 4: an unavailable channel falls back to the manual path (rule 21) --
Start-Case 'unavailable channel falls back to the manual path (rule 21)'
$env:SZA_SUB_TEST_PF = 'manual'
Set-StubConfig 'ok'
$r = Invoke-Step @{ Tag = 'v26.1117.1450'; MsstoreCommand = $cliStub; PreflightCommand = $pfStub; PackageBuildScript = $buildStub }
$env:SZA_SUB_TEST_PF = 'pass'
Check ($r.Code -eq 2) "exit 2 expected, was $($r.Code); output: $($r.Text)"
Check ($r.Text -match 'the Store step is manual; the release goes on') 'the manual line printed'
Check ($r.Text -match 'upload .* in Partner Center') 'the manual instructions printed'
Check (@(LogLines).Count -eq 0) 'nothing was written to the Store'
Check (Test-Path -LiteralPath $msixLog) 'the package was built for the manual upload'
Check (Test-Path -LiteralPath (Join-Path $import 'listingData.csv')) 'the listing import folder was built with refreshed notes'
Check ((VerdictText) -match '"mode":"manual"') 'the verdict row records the manual fallback'
End-Case

# -- 5: a foreign pending submission refuses and is never replaced (rule 23 (a), (f)) --
Start-Case 'foreign pending submission refuses, never deleted (rule 23 (a), (f))'
$env:SZA_SUB_TEST_PF = 'pending-foreign'
Set-StubConfig 'ok' -SubmissionId '111111111111'
$r = Invoke-Step @{ Tag = 'v26.1117.1450'; MsstoreCommand = $cliStub; PreflightCommand = $pfStub; PackageBuildScript = $buildStub }
$env:SZA_SUB_TEST_PF = 'pass'
Check ($r.Code -eq 1) "exit 1 expected, was $($r.Code)"
Check ($r.Text -match 'the owner clears it in the console') 'the refusal names the owner console action'
Check ((VerdictText) -match '"refusal":"pending-submission"') 'the verdict row records the refusal'
$lines = LogLines
Check (@($lines | Where-Object { $_ -match 'publish|delete|updateMetadata' }).Count -eq 0) 'no write verb was invoked'
Check (@($lines | Where-Object { $_ -match '^submission get' }).Count -ge 1) 'the pending state was read'
Check (-not (Test-Path -LiteralPath $msixLog)) 'no package build ran behind the refusal'
End-Case

# -- 6: a draft the channel's own record holds is resumed --
Start-Case "resumes the channel's own pending draft (rule 23 (a))"
$env:SZA_SUB_TEST_PF = 'pending-own'
Set-StubConfig 'ok' -SubmissionId '222222222222'
'{"time":"2026-11-17T00:00:00.0000000Z","storeId":"9PMHSWQPR6V1","packageVersion":"26.1117.1450.0","submissionId":"222222222222","mode":"prepared","listingPath":"json-metadata"}' |
    Add-Content -LiteralPath (Join-Path $testVerdicts '26.1117.1450.store-submission.jsonl') -Encoding utf8
$r = Invoke-Step @{ Tag = 'v26.1117.1450'; MsstoreCommand = $cliStub; PreflightCommand = $pfStub; PackageBuildScript = $buildStub }
$env:SZA_SUB_TEST_PF = 'pass'
Check ($r.Code -eq 0) "exit 0 expected, was $($r.Code); output: $($r.Text)"
Check ($r.Text -match "resuming this channel's own draft 222222222222") 'the resume line printed'
Check ($r.Text -match 'DRAFT prepared') 'the run prepared the draft'
Check ((LogLines) -match 'submission updateMetadata') 'the metadata was refreshed on the resumed draft'
End-Case

# -- 7: a missing deck and an over-cap ReleaseNotes refuse before any Store call (rules 9, 12, 16) --
Start-Case 'missing deck and over-cap ReleaseNotes refuse before any write'
New-Item -ItemType Directory -Force -Path $decksDir | Out-Null
foreach ($code in @('en','ru','uk','it','es','fr','pt','ar','hi','ur','zh')) {
    [System.IO.File]::WriteAllText((Join-Path $decksDir "$code.txt"), "# deck`n`n@@Title`ntranslate`n`n@@ReleaseNotes`nRelease notes for $code - short`n")
}
[System.IO.File]::WriteAllText((Join-Path $decksDir 'de.txt'), "# deck`n`n@@Title`ntranslate-de`n`n@@ReleaseNotes`n$([string]::new('d', 1600))`n")
# bn.txt deliberately absent
$r = Invoke-Step @{ Tag = 'v26.1117.1450'; MsstoreCommand = $cliStub; PreflightCommand = $pfStub; PackageBuildScript = $buildStub; ListingDir = $decksDir }
Check ($r.Code -eq 1) "exit 1 expected, was $($r.Code)"
Check ($r.Text -match 'no copy deck at .*bn\.txt') 'the refusal names the missing deck'
Check ($r.Text -match 'over its 1500-character cap') 'the refusal names the cap'
Check (@(LogLines).Count -eq 0) 'no msstore call at all'
Check (Test-Path -LiteralPath $msixLog) 'the package build had already run'
Check ((VerdictText) -match '"refusal":"listing-notes"') 'the verdict row records the refusal'
End-Case

# -- 8: a Store refusal of the upload stops the run (rule 23 (d)) --
Start-Case 'Store refuses the upload stops the run (rule 23 (d))'
Set-StubConfig 'fail-publish'
$r = Invoke-Step @{ Tag = 'v26.1117.1450'; MsstoreCommand = $cliStub; PreflightCommand = $pfStub; PackageBuildScript = $buildStub }
Check ($r.Code -eq 1) "exit 1 expected, was $($r.Code); output: $($r.Text)"
Check ($r.Text -match 'publish refused the upload') 'the refusal names the upload'
Check (@(LogLines | Where-Object { $_ -match 'updateMetadata|submission publish|submission get' }).Count -eq 0) 'no metadata write, no commit, no further read'
Check ((VerdictText) -match 'store-refused') 'the verdict row records the refusal'
End-Case

# -- 9: a draft that reads back without the uploaded version refuses (criterion (c)) --
Start-Case 'draft without the uploaded package version refuses (criterion (c))'
Set-StubConfig 'ok' -DraftVersion '9.9.9.9.0'
$r = Invoke-Step @{ Tag = 'v26.1117.1450'; MsstoreCommand = $cliStub; PreflightCommand = $pfStub; PackageBuildScript = $buildStub }
Check ($r.Code -eq 1) "exit 1 expected, was $($r.Code); output: $($r.Text)"
Check ($r.Text -match 'does not hold the package version 26\.1117\.1450\.0') 'the refusal names the version'
Check (@(LogLines | Where-Object { $_ -match 'updateMetadata|submission publish' }).Count -eq 0) 'no metadata write, no commit'
End-Case

# -- 10: a missing listing language refuses before the metadata write (rule 7 via rule 24) --
Start-Case 'missing listing language refuses before any metadata write'
Set-StubConfig 'ok' -DropLanguage 'bn'
$r = Invoke-Step @{ Tag = 'v26.1117.1450'; MsstoreCommand = $cliStub; PreflightCommand = $pfStub; PackageBuildScript = $buildStub }
Check ($r.Code -eq 1) "exit 1 expected, was $($r.Code); output: $($r.Text)"
Check ($r.Text -match 'no listing for: bn') 'the refusal names the language'
Check (@(LogLines | Where-Object { $_ -match 'updateMetadata|submission publish' }).Count -eq 0) 'no metadata write, no commit'
End-Case

# -- 11: a Store refusal of the metadata update stops the run before any commit --
Start-Case 'Store refuses the metadata update stops the run'
Set-StubConfig 'fail-update'
$r = Invoke-Step @{ Tag = 'v26.1117.1450'; MsstoreCommand = $cliStub; PreflightCommand = $pfStub; PackageBuildScript = $buildStub }
Check ($r.Code -eq 1) "exit 1 expected, was $($r.Code); output: $($r.Text)"
Check ($r.Text -match 'refused the metadata update') 'the refusal names the metadata update'
Check ((LogLines) -match 'submission updateMetadata') 'the update was attempted'
Check (@(LogLines | Where-Object { $_ -match 'submission publish' }).Count -eq 0) 'no commit behind a refused update'
End-Case

# -- 12: a read-back that disagrees keeps -Commit from committing (criterion (c)) --
Start-Case 'read-back disagreement blocks -Commit'
Set-StubConfig 'readbackbad'
$r = Invoke-Step @{ Tag = 'v26.1117.1450'; MsstoreCommand = $cliStub; PreflightCommand = $pfStub; PackageBuildScript = $buildStub; Commit = $true }
Check ($r.Code -eq 1) "exit 1 expected, was $($r.Code); output: $($r.Text)"
Check ($r.Text -match 'disagrees with what this run sent') 'the refusal names the disagreement'
Check (@(LogLines | Where-Object { $_ -match 'submission publish' }).Count -eq 0) 'no commit behind a failed proof'
Check ((VerdictText) -match '"refusal":"readback-proof"') 'the verdict row records the failed proof'
End-Case

# -- 13: the happy prepare sends the whole draft with only the 13 ReleaseNotes replaced --
Start-Case 'happy prepare: whole object back, only the 13 ReleaseNotes replaced (rule 24)'
Set-StubConfig 'ok'
$r = Invoke-Step @{ Tag = 'v26.1117.1450'; MsstoreCommand = $cliStub; PreflightCommand = $pfStub; PackageBuildScript = $buildStub }
Check ($r.Code -eq 0) "exit 0 expected, was $($r.Code); output: $($r.Text)"
Check ($r.Text -match 'DRAFT prepared - 26\.1117\.1450\.0 submission 333333333333 \(PENDING COMMIT\)') 'the draft identity printed'
Check ($r.Text -match 'partner\.microsoft\.com/dashboard/products/9PMHSWQPR6V1') 'the console address of the draft printed'
$lines = LogLines
Check (@($lines | Where-Object { $_ -match '^publish -nc' }).Count -eq 1) 'the upload ran exactly once'
Check (@($lines | Where-Object { $_ -match '^submission updateMetadata' }).Count -eq 1) 'the metadata update ran exactly once'
Check (@($lines | Where-Object { $_ -match '^submission get' }).Count -eq 2) 'the draft was read before and after the update'
Check (@($lines | Where-Object { $_ -match 'submission publish|delete|rollout|UNEXPECTED' }).Count -eq 0) 'no commit, no delete, no rollout without -Commit'
Check (Test-Path -LiteralPath $metaFile) 'the stub captured the metadata the step sent'
if (Test-Path -LiteralPath $metaFile) {
    $sent = Get-Content -LiteralPath $metaFile -Raw | ConvertFrom-Json
    Check ([string]$sent.id -eq '333333333333') 'the object went back whole: the id survived'
    Check ([string]$sent.priceId -eq 'Free') 'the object went back whole: the pricing survived untouched'
    Check (@($sent.applicationPackages | Where-Object { [string]$_.version -eq '26.1117.1450.0' }).Count -eq 1) 'the uploaded package survived in the object'
    $deckFailures = @()
    foreach ($code in $localeMap.Keys) {
        $key = $localeMap[$code]
        $deckNotes = Read-DeckReleaseNotes (Join-Path $Repo "tools/store/listing/$code.txt")
        $have = [string]$sent.listings.$key.releaseNotes
        if ($have -ne $deckNotes) { $deckFailures += $key }
    }
    Check ($deckFailures.Count -eq 0) "every listing carries its deck's ReleaseNotes (offenders: $($deckFailures -join ', '))"
    Check ([string]$sent.listings.'en-us'.description -eq 'keep me en-us') 'only the owned field changed: the description survived'
}
Check ((VerdictText) -match '"mode":"prepared"') 'the rule-26 row records the prepared draft'
Check ((LedgerHistoryText) -match 'store-submission: draft 333333333333 of 26\.1117\.1450\.0 prepared') 'the release ledger records the draft'
End-Case

# -- 14: -Commit commits behind the proof and records the status (rule 22 pre-authorization) --
Start-Case 'happy commit: commits, records status and ledger event'
Set-StubConfig 'ok'
$r = Invoke-Step @{ Tag = 'v26.1117.1450'; MsstoreCommand = $cliStub; PreflightCommand = $pfStub; PackageBuildScript = $buildStub; Commit = $true; Poll = $true }
Check ($r.Code -eq 0) "exit 0 expected, was $($r.Code); output: $($r.Text)"
Check ($r.Text -match 'COMMITTED 26\.1117\.1450\.0 submission 333333333333; status: CommitStarted') 'the commit line printed with the status'
$lines = LogLines
Check (@($lines | Where-Object { $_ -match '^submission publish' }).Count -eq 1) 'the commit verb ran exactly once'
Check (@($lines | Where-Object { $_ -match '^submission poll' }).Count -eq 1) 'the poll ran behind -Poll'
Check (@($lines | Where-Object { $_ -match 'delete|rollout|UNEXPECTED' }).Count -eq 0) 'no delete, no rollout'
$v = VerdictText
Check ($v -match '"mode":"committed"') 'the rule-26 row records the commit'
Check ($v -match 'CommitStarted') 'the rule-26 row records the status read'
Check ((LedgerHistoryText) -match 'store-submission: committed 26\.1117\.1450\.0') 'the release ledger records the commit'
End-Case

# -- restore the ledger files the cases appended to, and clean the scratch --
Restore-LedgerSnapshot
if ($ownsWork) { Remove-Item -LiteralPath $Work -Recurse -Force -ErrorAction SilentlyContinue }

. (Join-Path $PSScriptRoot '../../scripts/lib/verdict.ps1')
Write-Host ("cases={0} failed={1}" -f $script:total, $script:failed)
$verdictCode = if ($script:failed -gt 0) { 1 } else { 0 }
Exit-Verdict 'test-store-submission' $verdictCode ("run={0} failed={1}" -f $script:total, $script:failed)
