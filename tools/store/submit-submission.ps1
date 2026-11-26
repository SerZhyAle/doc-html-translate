<#
.SYNOPSIS
  The Store submission step of a release: package, listing notes, read-back, commit
  (WINDOWS-STORE section D, rules 19 to 26; ticket E0156).

.DESCRIPTION
  The agent-run half of the Microsoft Store leg of DEV/RELEASE.md step 4. Until E0156 this leg
  was a manual Partner Center upload; the owner pre-authorized the machine channel on 2026-11-17,
  including the commit into certification (ticket E0156 section 0; the rule-22 difference is the
  dated exception row in the catalog registry, with the allowance asked to the contract owner as
  the proposal PROPOSAL-2026-11-17-doc-html-translate-store-commit-preauthorization.md). This step
  is that authorization's one subject: the tag, winget, Chrome and Edge steps keep their own
  confirmations and this script never touches them.

  What it does, in order, each refusal ending the run before the next step runs:

    1. Identity (rule 19): read msix/identity.json; no StoreId means the run stops before it
       talks to the Store.
    2. Channel (rule 21): the msstore CLI must be configured. If it is missing, the run falls back
       to the manual path: build the package and the listing import folder, print the upload
       instructions, record the state, exit 2. The channel never fails a release because it is
       unavailable.
    3. Preflight: the canon's shared read-only tools/Store-Preflight.ps1 (the same third header
       verdict of scripts/release.ps1) is re-run inline and must pass in this same run (A5 of the
       ticket: a verdict printed earlier in the release is not reused). A pending submission it
       refuses is resumed only when the product's own rule-26 record holds that submission id
       (this channel's draft); a pending submission the channel did not create is refused and,
       per rule 23(a), the owner clears it in the console - this channel never deletes or edits it.
    4. Package (rules 1 to 4, 23(c)): msix/build-msix.ps1 -Tag builds the unsigned .msix from the
       tag's exact tree (it refuses a HEAD that is not the tag's commit or a dirty tree, and reads
       the packed archive back), so the green gate evidence that held for the tag covers what is
       uploaded. The run refuses to run while the listing CSV builder writes the same msix/out/
       folder - not enforced here, the release flow orders the steps.
    5. Listing notes (rules 9, 12, 16, 24): the @@ReleaseNotes field of all 13 copy decks
       (tools/store/listing/<code>.txt) - the rule-16 accumulated "What's new", translated in all
       columns - each non-empty and inside the 1500 cap, folded by the release flow before this
       step runs. A language that is missing or over cap refuses the write.
    6. Prepare: msstore publish -i <msix> -id <StoreId> -nc uploads the package and keeps the
       submission in draft; then submission get, only the thirteen ReleaseNotes fields of the
       languages it owns replaced (rule 24: the whole object read, matched by language key, sent
       back whole - for MSIX apps updateMetadata and update are equivalent per the CLI docs), the
       object sent back with msstore submission updateMetadata <id> <json> -s. A Store refusal
       stops the run (rule 23(d)); nothing is retried here.
    7. Read-back (the ticket's criterion (c)): the draft is read again and must hold the package
       version just uploaded, all 13 listing languages and the notes that were sent; any
       disagreement refuses the run before any commit.
    8. Commit (rule 22): only behind -Commit (the distinct switch is kept; the standing
       pre-authorization replaces the per-release typed confirmation, which is the recorded
       rule-22 difference, not a silent one): msstore submission publish <StoreId> sends the
       draft to certification, one status read is recorded, and -Poll additionally runs
       msstore submission poll. Without -Commit the run ends at the draft and prints its identity
       and the console address of the draft (rule 22).
    9. Evidence (rule 26): every run appends one line to DEV/release-verdicts/<msixVersion>.store-submission.
       jsonl and one publication event to the release ledger (DEV/releases/history.jsonl + RELEASES.md)
       through scripts/release-ledger.ps1: Store ID, package version, submission id, the statuses read
       with their time, the listing path used - no account identifier, no credential, no upload address.

  MSIX version remap: the tag v<yy.MMdd.HHmm> becomes <yy.MMdd.HHmm>.0, exactly as scripts/release.ps1
  maps the preflight candidate and msix/build-msix.ps1 stamps the package; change both together.

  UNVERIFIED until the first real run (contract rung 6, the next release; no Store write has ever been
  made): the publish invocation shape of the installed msstore 0.3.9.7, the write field name
  "releaseNotes" and the listings keys, and whether the CLI accepts a whole-draft JSON argument of
  the size a 13-language submission returns (about 330 KB was measured for submission get; the
  documented tip for large metadata is to pass the file's contents). A refusal the checks did not
  predict is a defect in the rules, amended in the catalog first (comply or amend), never here.

  Exit codes: 0 the run completed (draft prepared, or committed when -Commit was given); 1 refused -
  the run names the refusal, writes no Store object, and a prior draft is left exactly as it was;
  2 the channel is unavailable, the run fell back to the manual path and printed the instructions
  (the release continues; CHECK-VERDICT's COULD NOT VERIFY - a missing tool is never a pass).

.PARAMETER Tag
  The app release tag (v<yy.MMdd.HHmm>); the package version and the verdicts file name derive from it.

.PARAMETER Commit
  Commit the prepared draft for certification (rule 22's distinct switch; standing pre-authorization
  E0156). Without it the run stops at the draft and prints its identity.

.PARAMETER CredentialExpiry
  Optional credential expiry date (yyyy-MM-dd) from the owner's expiry list, passed to the preflight.

.PARAMETER Poll
  After a commit, also poll through the CLI (msstore submission poll) until PUBLISHED or FAILED.

.PARAMETER MsstoreCommand
  The msstore CLI to invoke. Default: the one on PATH (the owner's terminal). The test points it at a
  stub that logs every invocation; production runs never pass it.

.PARAMETER PreflightCommand
  The canon store preflight tool (.ps1). Default: the sza plugin's shared tools/Store-Preflight.ps1.
  The test points it at a stub.

.PARAMETER IdentityFile
  The machine-readable identity file (rule 19). Default: msix/identity.json. The test points it at a
  scratch copy; production runs never pass it.

.PARAMETER ListingDir
  The folder of per-language decks whose @@ReleaseNotes this step sends. Default:
  tools/store/listing - the same folder, and the same parameter name, as build-store-listing-csv.ps1.

.PARAMETER PackageBuildScript
  The package build the step runs. Default: msix/build-msix.ps1. The test points it at a stub.

.PARAMETER MsixOut
  The folder the package is expected in. Default: msix/out. The test points it at a scratch folder,
  so a stub never touches a real build artifact.

.PARAMETER VerdictsDir
  The rule-26 evidence folder, one <msixVersion>.store-submission.jsonl per run. Default:
  DEV/release-verdicts. The test points it at a scratch folder.

.EXAMPLE
  # the release step (after the GitHub release is live; the commit is the pre-authorized act):
  ./tools/store/submit-submission.ps1 -Tag v26.1117.1450 -Commit

.EXAMPLE
  # prepare only: leave the draft, prove it, print its identity and the console address:
  ./tools/store/submit-submission.ps1 -Tag v26.1117.1450
#>
param(
    [Parameter(Mandatory)][string]$Tag,
    [switch]$Commit,
    [string]$CredentialExpiry,
    [switch]$Poll,
    [string]$MsstoreCommand,
    [string]$PreflightCommand,
    [string]$IdentityFile,
    [string]$ListingDir,
    [string]$PackageBuildScript,
    [string]$MsixOut,
    [string]$VerdictsDir
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
. (Join-Path $PSScriptRoot 'store-listing-checks.ps1')

# The deck language code -> the Partner Center listing key (the same map the listing builder holds).
$Locale = [ordered]@{
    en = "en-us"; ru = "ru"; uk = "uk"; de = "de"; it = "it"; es = "es"; fr = "fr"
    pt = "pt-br"; ar = "ar"; hi = "hi"; bn = "bn"; ur = "ur"; zh = "zh-hans"
}

function Stop-Refused([string[]]$reasons) {
    foreach ($r in $reasons) { Write-Host "refused: $r" -ForegroundColor Red }
    Write-Host "store-submission: REFUSED (nothing was committed)" -ForegroundColor Red
    exit 1
}

# The copy-deck reader of build-store-listing-csv.ps1, restated here: dot-sourcing the builder would
# run it. Same shape, same tolerance: @@Field sections, lines starting '#' are comments.
function Read-ListingSource([string]$path) {
    $fields = [ordered]@{}
    $name = $null
    $buf = New-Object System.Collections.ArrayList
    foreach ($line in [System.IO.File]::ReadAllLines($path)) {
        if ($line -match '^@@(\w+)\s*$') {
            if ($name) { $fields[$name] = ($buf -join "`n").Trim() }
            $name = $Matches[1]; $buf.Clear()
        } elseif ($name -and $line -notmatch '^#') {
            [void]$buf.Add($line)
        }
    }
    if ($name) { $fields[$name] = ($buf -join "`n").Trim() }
    return $fields
}

# Reading the CLI's output (the contract's rule, measured 2026-11-12): redirected, the payload is
# wrapped at 80 columns even inside \uXXXX escapes, because the real breaks inside values are the
# escaped "\r\n". Removing every raw CR and LF keeps the escaped ones and the payload then parses;
# a raw break is legal JSON only as whitespace, so removing it changes no value.
function ConvertFrom-MsstoreJson([string]$Text, [string]$What) {
    $joined = [regex]::Replace($Text, '\r|\n', '')
    $candidates = @($joined)
    # The CLI prints a status line ahead of the payload; every line that parses as JSON is accepted,
    # the last one winning (the payload follows the line).
    $parsed = $null
    foreach ($line in ($Text -split '\r?\n')) {
        $l = $line.Trim()
        if ($l -match '^[\{\[].*[\}\]]$') {
            $alt = [regex]::Replace($l, '\r|\n', '')
            try { $parsed = $alt | ConvertFrom-Json } catch { }
        }
    }
    if ($null -ne $parsed) { return $parsed }
    try { return ($candidates[0] | ConvertFrom-Json) } catch { }
    Stop-Refused @("the $What could not be read (rule 23: a state the channel cannot read refuses the run)")
}

function Invoke-Msstore {
    param([string[]]$Arguments)
    $output = & $script:Msstore @Arguments 2>&1 | ForEach-Object { "$_" }
    return @{ Text = ($output -join "`n"); Code = $LASTEXITCODE }
}

# Rule 26: the evidence record, one JSON line per event, no account identifier, credential or
# upload address. The verdicts folder name follows StreamsPlayer's release-verdicts precedent.
if (-not $VerdictsDir) { $VerdictsDir = Join-Path $RepoRoot 'DEV/release-verdicts' }
function Write-Verdict([hashtable]$Event) {
    New-Item -ItemType Directory -Force -Path $VerdictsDir | Out-Null
    $row = @{ time = (Get-Date).ToUniversalTime().ToString('o'); storeId = $script:StoreIdOfRecord;
        packageVersion = $script:MsixVersionTag; listingPath = 'json-metadata' }
    foreach ($k in $Event.Keys) { $row[$k] = $Event[$k] }
    $line = $row | ConvertTo-Json -Compress
    Add-Content -LiteralPath (Join-Path $VerdictsDir "$($script:MsixVersionTag).store-submission.jsonl") -Value $line -Encoding utf8
}
function Read-VerdictIds([string]$VerdictsFile) {
    if (-not (Test-Path -LiteralPath $VerdictsFile)) { return @() }
    $ids = @()
    foreach ($line in (Get-Content -LiteralPath $VerdictsFile -Encoding utf8)) {
        if (-not $line.Trim()) { continue }
        try { $r = $line | ConvertFrom-Json } catch { continue }
        if ($r.submissionId) { $ids += [string]$r.submissionId }
    }
    return $ids
}

# -- identity (rule 19) and the tag --
if ($Tag -notmatch '^v\d{2}\.\d{4}\.\d{4}$') {
    Stop-Refused @("-Tag '$Tag' is not a v<yy.MMdd.HHmm> app tag")
}
$stamp = $Tag.TrimStart('v')
$sv = $stamp.Split('.')
$script:MsixVersionTag = '{0}.{1}.{2}.0' -f [int]$sv[0], [int]$sv[1], [int]$sv[2]
if (-not $IdentityFile) { $IdentityFile = Join-Path $RepoRoot 'msix/identity.json' }
if (-not (Test-Path -LiteralPath $IdentityFile)) {
    Stop-Refused @("$IdentityFile is missing; the channel stops before it talks to the Store (rule 19)")
}
try { $identity = Get-Content -LiteralPath $IdentityFile -Raw | ConvertFrom-Json } catch {
    Stop-Refused @("$IdentityFile does not parse (rule 19)")
}
$StoreId = [string]$identity.StoreId
if (-not $StoreId -or $StoreId -notmatch '^[A-Za-z0-9]{12}$') {
    Stop-Refused @("msix/identity.json carries no usable StoreId (rule 19)")
}
$script:StoreIdOfRecord = $StoreId

# -- the channel (rule 21) --
if (-not $MsstoreCommand) {
    $cmd = Get-Command msstore -ErrorAction SilentlyContinue
    if (-not $cmd) { $MsstoreCommand = $null } else { $MsstoreCommand = $cmd.Source }
}
if (-not $PreflightCommand) {
    # The canon's shared preflight, located the same way scripts/release.ps1 locates it.
    $roots = @($env:CLAUDE_PLUGIN_ROOT)
    $record = Join-Path $HOME '.claude/plugins/installed_plugins.json'
    if (Test-Path -LiteralPath $record) {
        try {
            $roots += @((Get-Content -LiteralPath $record -Raw | ConvertFrom-Json).plugins.'sza@sza-unified-rules' | ForEach-Object { $_.installPath })
        } catch { }
    }
    foreach ($root in $roots) {
        if (-not $root) { continue }
        $tool = Join-Path $root 'tools/Store-Preflight.ps1'
        if (Test-Path -LiteralPath $tool) { $PreflightCommand = $tool; break }
    }
}

$listingBuilder = Join-Path $RepoRoot 'tools/store/build-store-listing-csv.ps1'
if (-not $PackageBuildScript) { $PackageBuildScript = Join-Path $RepoRoot 'msix/build-msix.ps1' }
if (-not $ListingDir) { $ListingDir = Join-Path $RepoRoot 'tools/store/listing' }
if (-not $MsixOut) { $MsixOut = Join-Path $RepoRoot 'msix/out' }
$scratchDir = Join-Path $RepoRoot 'temp/E0156'
New-Item -ItemType Directory -Force -Path $scratchDir | Out-Null
$expectedMsix  = Join-Path $MsixOut ('{0}_{1}_x64.msix' -f ($identity.IdentityName -replace '[^A-Za-z0-9._-]', '_'), $script:MsixVersionTag)

# Runs a repo .ps1 and reports @{ Code; Text }. Hash splat, never an array: an array splatted to
# a script binds its "-Name" tokens as positional values (the observed failure mode, the same
# trap the harness fixed with PositionalBinding=$false) - a hashtable binds by name, and a
# refusal inside the child becomes this run's refusal instead of a terminating error.
# *>&1: repo scripts report through Write-Host (the preflight's refusal lines, the builders'
# "refused:" lines), and this run's refusals quote them.
function Invoke-RepoScript {
    param([string]$Path, [hashtable]$Arguments)
    $out = @()
    try { $out = @(& $Path @Arguments *>&1 | ForEach-Object { "$_" }) }
    catch { $out = @("$out", $_.Exception.Message) }
    if (-not $out) { $out = @('') }
    $code = if ($LASTEXITCODE -gt 0) { $LASTEXITCODE } elseif ($?) { 0 } elseif ($LASTEXITCODE -lt 0) { - $LASTEXITCODE } else { 1 }
    return @{ Text = ($out -join "`n"); Code = $code }
}

function Write-ReleaseLedgerPublish {
    param([string]$Version, [string]$State, [string]$Ref, [string]$Note)
    $ledgerScript = Join-Path $RepoRoot 'scripts/release-ledger.ps1'
    if (-not (Test-Path -LiteralPath $ledgerScript)) { return }
    $argv = @('publish', '-Key', 'msstore', '-Version', $Version, '-State', $State)
    if ($Ref)  { $argv += @('-Ref', $Ref) }
    if ($Note) { $argv += @('-Note', $Note) }
    try {
        & $ledgerScript @argv 2>&1 | Out-Null
    } catch {
        Write-Warning "release ledger not updated: $($_.Exception.Message)"
    }
}

function Invoke-ManualFallback([string]$Why, [bool]$Built) {
    # Rule 21: the run builds and verifies the package and the listing exactly as without the
    # channel, ends with the manual instructions, and never fails the release because of the
    # channel. Only the channel-unavailable paths lead here, so the fallback never runs behind a
    # Store refusal.
    Write-Host "the Store channel is unavailable: $Why" -ForegroundColor Yellow
    Write-Host "the Store step is manual; the release goes on (rule 21)" -ForegroundColor Yellow
    if (-not $Built) {
        $build = Invoke-RepoScript $PackageBuildScript @{ Tag = $Tag }
        if ($build.Code -ne 0 -or -not (Test-Path -LiteralPath $expectedMsix)) {
            Stop-Refused @("the package build for $Tag failed; the manual fallback has no package to hand over", (@($build.Text -split '\r?\n') | Select-Object -Last 3))
        }
    }
    # The listing half of the manual path: the folder import with the decks' ReleaseNotes
    # refreshed behind the builder's explicit replace switch (rule 12), the result written to the
    # scratch CSV and copied into the dashboard folder beside the package - the committed
    # listingData.csv (a fresh export the release flow keeps) is not rewritten by the fallback.
    $listing = Invoke-RepoScript $listingBuilder @{ Refresh = @('ReleaseNotes'); Out = (Join-Path $scratchDir 'manual-listing.csv'); ImportFolder = (Join-Path $MsixOut 'store-import') }
    if ($listing.Code -ne 0) {
        Stop-Refused @("the listing builder refused: $(@($listing.Text -split '\r?\n' | Where-Object { $_ -match '^refused:' } | Select-Object -First 3) -join '; ')")
    }
    Write-Host "manual path:" -ForegroundColor Yellow
    Write-Host "  - upload $expectedMsix in Partner Center (Products > $StoreId > Update)"
    Write-Host "  - import the listing from msix/out/store-import/ (folder import)"
    Write-Host "  - submit for certification by hand; record the state with scripts/release-ledger.ps1"
    Write-Verdict @{ mode = 'manual'; note = $Why }
    Write-ReleaseLedgerPublish -Version $stamp -State 'pending' -Note "channel unavailable ($Why); package built from $Tag; listing folder built; owner uploads by hand"
    Write-Host "store-submission: MANUAL (exit 2; nothing was written to the Store)" -ForegroundColor Yellow
    exit 2
}

if (-not $MsstoreCommand -and -not $PreflightCommand) { Invoke-ManualFallback 'the msstore CLI was not found on PATH and the shared preflight tool was not found' $false }
if (-not $MsstoreCommand) { Invoke-ManualFallback 'the msstore CLI was not found on PATH (the owner configures it in the owner terminal)' $false }
if (-not $PreflightCommand) { Invoke-ManualFallback "the canon's shared Store-Preflight.ps1 was not found (the sza plugin is not installed here)" $false }
$script:Msstore = $MsstoreCommand

# -- preflight, re-run inline (ticket A5) --
$pfArgs = @{ ProductName = 'Doc-HTML-Translate'; IdentityFile = $IdentityFile;
    PackageVersion = $script:MsixVersionTag; Json = $true }
if ($CredentialExpiry) { $pfArgs['CredentialExpiry'] = $CredentialExpiry }
$pf = Invoke-RepoScript $PreflightCommand $pfArgs
$pfCode = $pf.Code
$pfText = $pf.Text
$pfJsonLine = @(($pfText -split '\r?\n') | Where-Object { $_ -match '^\{.*\}$' }) | Select-Object -Last 1
$pfJson = $null
if ($pfJsonLine) { try { $pfJson = $pfJsonLine | ConvertFrom-Json } catch { } }

if ($pfCode -eq 2) {
    Invoke-ManualFallback (@(($pfText -split '\r?\n') | Where-Object { $_ -match 'manual|unavailable|expired|refused token' } | Select-Object -First 1) -join ' ') $false
}
if ($pfCode -ne 0) {
    $refusal = if ($pfJson -and $pfJson.refusal) { [string]$pfJson.refusal } else { "the preflight exited $pfCode" }
    $detail  = if ($pfJson -and $pfJson.detail) { [string]$pfJson.detail } else { (@(($pfText -split '\r?\n') | Where-Object { $_ -match '^refused:' }) -join '; ') }
    if ($refusal -eq 'pending-submission' -or $refusal -eq 'mixed-edit') {
        # Rule 23(a) and (f): a pending submission is refused unless this channel's own record
        # (rule 26) holds its id - a draft a previous run of this step left. The channel never
        # deletes or edits a submission it did not itself create; the owner clears it in the console.
        $ids = @(Read-VerdictIds (Join-Path $VerdictsDir "$($script:MsixVersionTag).store-submission.jsonl")) +
               @(Get-ChildItem -LiteralPath $VerdictsDir -Filter '*.store-submission.jsonl' -ErrorAction SilentlyContinue |
                    ForEach-Object { Read-VerdictIds $_.FullName })
        $live = ConvertFrom-MsstoreJson (Invoke-Msstore @('submission', 'get', $StoreId)).Text "pending submission's state"
        $pendingId = if ($pfJson -and $pfJson.submissionId) { [string]$pfJson.submissionId }
                     elseif ($live -and $live.id) { [string]$live.id } else { '' }
        if (-not $pendingId -or $ids -notcontains $pendingId) {
            Write-Verdict @{ mode = 'refused'; refusal = 'pending-submission'; submissionId = $pendingId; note = "not this channel's draft; the owner clears it in the console (rule 23 (a), (f))" }
            Stop-Refused @("a submission is already pending ($pendingId) and the product's own record does not hold it, so this channel will not update, replace or delete it (rule 23 (a)); the owner clears it in the console and runs the step again",
                $detail)
        }
        Write-Host "resuming this channel's own draft $pendingId (recorded in DEV/release-verdicts/)" -ForegroundColor Yellow
    } else {
        Write-Verdict @{ mode = 'refused'; refusal = $refusal }
        Stop-Refused @("the preflight refused this run: $refusal", $detail)
    }
}

# -- package (rules 1 to 4, 23(c); the build holds the step to the tag's exact tree) --
$build = Invoke-RepoScript $PackageBuildScript @{ Tag = $Tag }
if ($build.Code -ne 0 -or -not (Test-Path -LiteralPath $expectedMsix)) {
    Stop-Refused @("msix/build-msix.ps1 -Tag $Tag did not produce $expectedMsix", (@($build.Text -split '\r?\n') | Select-Object -Last 3))
}

# -- listing notes (rules 9, 12, 16, 24): the decks are the single source of the copy --
$notes = @{}
$noteRefusals = @()
foreach ($code in $Locale.Keys) {
    $deck = Join-Path $ListingDir "$code.txt"
    if (-not (Test-Path -LiteralPath $deck)) { $noteRefusals += "no copy deck at $deck (rule 12)"; continue }
    $fields = Read-ListingSource $deck
    if (-not $fields.Contains('ReleaseNotes')) { $noteRefusals += "$deck has no @@ReleaseNotes field (rule 16: the accumulated What's new, refreshed by the release flow)"; continue }
    $value = [string]$fields['ReleaseNotes']
    if (-not $value.Trim()) { $noteRefusals += "$deck @@ReleaseNotes is empty; the release flow refreshes all 13 decks before this step"; continue }
    if ((Get-ListingCap 'ReleaseNotes') -lt (Get-CellLength $value)) { $noteRefusals += "ReleaseNotes over its 1500-character cap in $deck ($(Get-CellLength $value)); fold the text (rule 9, counted after CRLF normalisation)" }
    $notes[$Locale[$code]] = $value
}
if ($noteRefusals) {
    Write-Verdict @{ mode = 'refused'; refusal = 'listing-notes' }
    Stop-Refused $noteRefusals
}

# -- prepare: upload the package into a draft, then send the draft JSON back with only the
#    thirteen ReleaseNotes fields replaced (rule 24) --
$pub = Invoke-Msstore @('publish', '-i', $expectedMsix, '-id', $StoreId, '-nc')
if ($pub.Code -ne 0) {
    Write-Verdict @{ mode = 'store-refused'; verb = 'publish -nc' }
    Stop-Refused @("msstore publish refused the upload (exit $($pub.Code)); the run stops, the draft is left as it was (rule 23 (d))")
}
$draft = ConvertFrom-MsstoreJson (Invoke-Msstore @('submission', 'get', $StoreId)).Text 'draft (submission get)'
if (-not $draft -or -not $draft.id) { Stop-Refused @("the draft did not read back after the upload (rule 26's record follows every run)") }
$submissionId = [string]$draft.id

$packageOk = $false
foreach ($p in @($draft.applicationPackages)) {
    if ($p -and (([string]$p.version) -eq $script:MsixVersionTag)) { $packageOk = $true }
}
if (-not $packageOk) {
    Write-Verdict @{ mode = 'refused'; refusal = 'readback-package'; submissionId = $submissionId }
    Stop-Refused @("the draft does not hold the package version $script:MsixVersionTag this run just uploaded; nothing was committed (criterion (c))")
}

if (-not $draft.listings) {
    Write-Verdict @{ mode = 'refused'; refusal = 'readback-listings'; submissionId = $submissionId }
    Stop-Refused @("the draft read back with no listings object (rule 24: the draft is read whole and sent back whole)")
}
$missingLang = @()
foreach ($key in $notes.Keys) {
    if (-not $draft.listings.$key) { $missingLang += $key }
}
if ($missingLang) {
    Write-Verdict @{ mode = 'refused'; refusal = 'readback-listings'; submissionId = $submissionId }
    Stop-Refused @("the draft has no listing for: $($missingLang -join ', '); a listing the product ships exists in Partner Center first (rule 7), so nothing was written")
}
foreach ($key in $notes.Keys) {
    # Only the fields this channel owns: the thirteen ReleaseNotes cells (rule 24).
    if (-not ($draft.listings.$key.PSObject.Properties.Name -contains 'releaseNotes')) {
        Add-Member -InputObject $draft.listings.$key -Name releaseNotes -Value $notes[$key] -Force
    } else {
        $draft.listings.$key.releaseNotes = $notes[$key]
    }
}

# Send the object back whole. The write field names of the API are UNVERIFIED (rule 24) and the
# payload of a 13-language draft measured about 330 KB, above the Windows command-line length
# limit - both are proven by the first real run (rung 6) and corrected under comply-or-amend,
# never silently; -s skips the CLI's own pre-action polling so this run controls every Store call.
$sentJson = $draft | ConvertTo-Json -Depth 100
$upd = Invoke-Msstore @('submission', 'updateMetadata', $StoreId, $sentJson, '-s')
if ($upd.Code -ne 0) {
    Write-Verdict @{ mode = 'store-refused'; verb = 'submission updateMetadata'; submissionId = $submissionId }
    Stop-Refused @("msstore submission updateMetadata refused the metadata update (exit $($upd.Code)); the draft still holds the uploaded package and no package was committed (rule 23 (d))")
}

# -- read-back proof (criterion (c)): version, 13 languages, the notes sent --
$proof = ConvertFrom-MsstoreJson (Invoke-Msstore @('submission', 'get', $StoreId)).Text 'draft after the metadata update'
$proofRefusals = @()
$packageOk = $false
foreach ($p in @($proof.applicationPackages)) {
    if ($p -and (([string]$p.version) -eq $script:MsixVersionTag)) { $packageOk = $true }
}
if (-not $packageOk) { $proofRefusals += "the read-back draft holds no package version $script:MsixVersionTag" }
if (-not $proof.listings) { $proofRefusals += "the read-back draft has no listings object" }
foreach ($key in $notes.Keys) {
    $have = if ($proof.listings -and $proof.listings.$key) { [string]$proof.listings.$key.releaseNotes } else { $null }
    if ($null -eq $have) { $proofRefusals += "listing $key read back with no releaseNotes" }
    elseif ([regex]::Replace($have, '\r\n|\r|\n', "`n") -ne [regex]::Replace($notes[$key], '\r\n|\r|\n', "`n")) {
        $proofRefusals += "listing $key read back with release notes that differ from the deck"
    }
}
if (@($Locale.Values | Where-Object { -not ($proof.listings -and $proof.listings.$_) }).Count -gt 0) {
    $proofRefusals += "the draft holds fewer than the 13 listing languages: $((@($Locale.Values | Where-Object { -not ($proof.listings -and $proof.listings.$_) })) -join ', ')"
}
if ($proofRefusals) {
    Write-Verdict @{ mode = 'refused'; refusal = 'readback-proof'; submissionId = $submissionId }
    Stop-Refused (@('the draft read-back disagrees with what this run sent (criterion (c)); nothing was committed') + $proofRefusals)
}

Write-Verdict @{ mode = 'prepared'; submissionId = $submissionId }

# -- commit (rule 22: distinct switch; standing pre-authorization, ticket E0156) --
if ($Commit) {
    $cm = Invoke-Msstore @('submission', 'publish', $StoreId)
    if ($cm.Code -ne 0) {
        Write-Verdict @{ mode = 'store-refused'; verb = 'submission publish'; submissionId = $submissionId }
        Stop-Refused @("msstore submission publish refused the commit (exit $($cm.Code)); the draft stands, nothing else changed (rule 23 (d))")
    }
    if ($Poll) {
        $pl = Invoke-Msstore @('submission', 'poll', $StoreId)
        if ($pl.Code -ne 0) {
            Write-Verdict @{ mode = 'poll-failed'; submissionId = $submissionId }
            Stop-Refused @("msstore submission poll failed (exit $($pl.Code)); the commit stands; record the state by hand")
        }
    }
    $st = Invoke-Msstore @('submission', 'status', $StoreId)
    $statusText = (@($st.Text -split '\r?\n') | Where-Object { $_.Trim() } | Select-Object -Last 1)
    Write-Verdict @{ mode = 'committed'; submissionId = $submissionId; status = $statusText }
    Write-ReleaseLedgerPublish -Version $stamp -State 'submitted' -Ref $submissionId -Note "store-submission: committed $script:MsixVersionTag (listing path json-metadata, all 13 ReleaseNotes refreshed, read-back proved); status: $statusText"
    Write-Host "store-submission: COMMITTED $script:MsixVersionTag submission $submissionId; status: $statusText" -ForegroundColor Green
    Write-Host "the certification result and the published state are checked in step 6 of the checklist (Verify)"
    exit 0
}

# Prepare-only end (rule 22): print the draft's identity and the console address of the draft.
Write-Host "store-submission: DRAFT prepared - $script:MsixVersionTag submission $submissionId (PENDING COMMIT)" -ForegroundColor Green
Write-Host "the draft is not a dry run; nobody edits it in the console (rule 23 (f)); commit it with:"
Write-Host "  ./tools/store/submit-submission.ps1 -Tag $Tag -Commit"
Write-Host "console address of the draft: https://partner.microsoft.com/dashboard/products/$StoreId"
Write-ReleaseLedgerPublish -Version $stamp -State 'pending' -Ref $submissionId -Note "store-submission: draft $submissionId of $script:MsixVersionTag prepared (listing path json-metadata, read-back proved), awaiting -Commit"
exit 0
