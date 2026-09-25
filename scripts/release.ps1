<#
.SYNOPSIS
  "Релиз" (release) - prints the complete, ordered release checklist. RUNS NOTHING.

.DESCRIPTION
  This is intentionally a checklist-only helper (see DEV/RELEASE.md for the
  canonical definition). It performs ONLY read-only git/file inspection to fill
  in the current state, then prints every release step with the exact command to
  run by hand. It NEVER pushes a tag, submits to winget, uploads to the Store, or
  publishes the extension - so running it can never trigger a paid GitHub CI run
  or an external publish. You execute each step yourself.

  Its header carries the two verdicts the tag step takes as input: the gate evidence of the last
  scripts/check.ps1 run (the full default plan, every child passing, on HEAD's tree) and the contract gate (scripts/contract-gate.ps1,
  read-only; FAIL or UNVERIFIED blocks). Exit code: 0 when both allow the tag, 1 while either
  blocks it.

  Steps that cost money / are public are marked [PAID] / [PUBLIC].

.PARAMETER Version
  Optional version stamp (yy.MMdd.HHmm) to inline into the printed commands.
  Default: the suggested stamp computed from the current time. Anything that is not a real
  YY.MMDD.HHmm stamp stops the script with exit code 2 before the checklist prints.

.EXAMPLE
  ./scripts/release.ps1
#>
param(
    [string]$Version
)

$ErrorActionPreference = "Stop"

function Get-GitValue([string[]]$GitArgs) {
    try {
        $out = & git @GitArgs 2>$null
        if ($LASTEXITCODE -ne 0) { return "" }
        return ($out | Out-String).Trim()
    } catch { return "" }
}

# ── read-only state ──────────────────────────────────────────
$branch   = Get-GitValue @("branch", "--show-current")
$dirty    = Get-GitValue @("status", "--porcelain")
$lastVer  = Get-GitValue @("describe", "--tags", "--abbrev=0", "--match", "v*")
$lastExt  = Get-GitValue @("describe", "--tags", "--abbrev=0", "--match", "ext-*-v*")

if (-not $Version) {
    $now = Get-Date
    $Version = "{0}.{1:D2}{2:D2}.{3:D4}" -f [int]$now.ToString('yy'), [int]$now.ToString('MM'), [int]$now.ToString('dd'), [int]$now.ToString('HHmm')
}
# The stamp becomes the tag, the file version and the winget version; release.yml rejects any
# other shape, so a malformed one would only surface after the tag is pushed.
if (-not ($Version -match '^\d{2}\.(\d{2})(\d{2})\.(\d{2})(\d{2})$') -or
    [int]$Matches[1] -lt 1 -or [int]$Matches[1] -gt 12 -or [int]$Matches[2] -lt 1 -or [int]$Matches[2] -gt 31 -or
    [int]$Matches[3] -gt 23 -or [int]$Matches[4] -gt 59) {
    Write-Host "-Version '$Version' is not a YY.MMDD.HHmm stamp (e.g. 26.0715.2003)" -ForegroundColor Red
    exit 2
}
$tag = "v$Version"

# ── gate evidence (BUILD-EVIDENCE) ───────────────────────────
# CI rebuilds the release binaries from the tag and runs no tests, so the only thing binding
# them to a tested state is this: the last scripts/check.ps1 run passed on exactly the tree
# HEAD holds (the tree hash, not the commit - build-local commits after the gate ran), with the
# full default plan, every child at PASS or PASS WITH ADVISORIES.
#
# The one path allowed to differ between the gated tree and HEAD's is the build log
# scripts/build-local.ps1 appends after its commit (it needs the commit's hash, so it cannot exist
# before the gate): an append-only ledger no check reads for content.
$postGatePaths = @('DEV/COMMIT_LOG.md')

# The full plan, from the placement record rather than from the evidence's own say-so: every
# check recorded as class gate for runner scripts/check.ps1 (tests/placement_test.go holds
# check.ps1's default plan equal to this set).
function Get-GatePlan {
    $recs = @(Get-Content -LiteralPath 'configs/check-placement.jsonl' -Encoding utf8 |
        Where-Object { $_.Trim() } | ForEach-Object { $_ | ConvertFrom-Json })
    return @($recs | Where-Object { $_.class -eq 'gate' -and $_.runner -eq 'scripts/check.ps1' } | ForEach-Object check | Sort-Object)
}
function ConvertTo-PlanEntry([string]$p) { ($p -replace '\\', '/') -replace '^\./', '' }

$headTree = Get-GitValue @("rev-parse", "HEAD^{tree}")
$evidencePath = "temp/logs/gate-evidence.json"
$gateOk = $false
if (Test-Path -LiteralPath $evidencePath) {
    try {
        $ev = Get-Content -LiteralPath $evidencePath -Raw | ConvertFrom-Json
        $fullPlan = Get-GatePlan
        $ranPlan = @($ev.plan | ForEach-Object { ConvertTo-PlanEntry $_ } | Sort-Object)
        $missing = @($fullPlan | Where-Object { $_ -notin $ranPlan })
        $extra = @($ranPlan | Where-Object { $_ -notin $fullPlan })
        $badChildren = @($ev.children | Where-Object { $_.code -notin 0, 3 })
        $treeDiff = @()
        if ($ev.tree -and $headTree -and $ev.tree -ne $headTree) {
            $treeDiff = @(Get-GitValue @("diff-tree", "-r", "--name-only", $ev.tree, $headTree) -split "`n" | ForEach-Object { $_.Trim() } | Where-Object { $_ })
        }
        if ($ev.code -notin 0, 3) {
            $gateLine = "BLOCKED - the last gate did not pass: $($ev.verdict) ($($ev.time))"
        } elseif ($fullPlan.Count -eq 0) {
            $gateLine = "BLOCKED - configs/check-placement.jsonl records no gate check for scripts/check.ps1, so the full plan is unknown"
        } elseif ($missing -or $extra -or @($ev.children).Count -ne $fullPlan.Count) {
            $gateLine = "BLOCKED - the last gate ran a partial plan ($(@($ev.children).Count) child check(s)$(if ($missing) { '; missing ' + ($missing -join ', ') })$(if ($extra) { '; not in the gate ' + ($extra -join ', ') })); run scripts/check.ps1 with no -Plan"
        } elseif ($badChildren) {
            $gateLine = "BLOCKED - a gate child did not pass: $(($badChildren | ForEach-Object verdict) -join '; ')"
        } elseif (-not $ev.tree) {
            $gateLine = "BLOCKED - the last gate recorded no tree$(if ($ev.treeBefore -ne $ev.treeAfter) { ' (the working tree changed while it ran)' }); re-run scripts/build-local.ps1"
        } elseif ($ev.tree -ne $headTree -and (-not $treeDiff -or @($treeDiff | Where-Object { $_ -notin $postGatePaths }))) {
            $gateLine = "BLOCKED - the last passing gate ran on tree $($ev.tree), HEAD is tree $headTree$(if ($treeDiff) { " (differs in $(@($treeDiff).Count) path(s))" }); re-run scripts/build-local.ps1"
        } elseif ($dirty) {
            $gateLine = "BLOCKED - the gate matches HEAD, but the working tree has changes the tag would not contain"
        } else {
            $gateOk = $true
            $gateLine = if ($ev.tree -eq $headTree) { "$($ev.verdict) on tree $headTree = HEAD ($($ev.time))" }
                        else { "$($ev.verdict) on tree $($ev.tree) = HEAD but for $($treeDiff -join ', ') ($($ev.time))" }
        }
    } catch {
        $gateLine = "BLOCKED - $evidencePath is unreadable: $($_.Exception.Message)"
    }
} else {
    $gateLine = "BLOCKED - no gate evidence ($evidencePath); run scripts/build-local.ps1 first"
}

# ── contract gate (CONTRACTS section 6) ──────────────────────
# Read-only: scripts/contract-gate.ps1 reads docs/contracts/ and the catalog's registry. PASS and
# WARN let the tag step go (a WARN names what to re-verify); FAIL blocks it, and so does UNVERIFIED -
# an unreachable catalog proves nothing, and a release is cut on the machine that has it.
$contractOk = $false
$pwsh = (Get-Process -Id $PID).Path
$gateOut = @(& $pwsh -NoProfile -File "$PSScriptRoot/contract-gate.ps1" *>&1 | ForEach-Object { "$_" })
$gateCode = $LASTEXITCODE
$gateLast = @($gateOut | Where-Object { $_ -match '^contract-gate: (PASS|FAIL|COULD NOT VERIFY)' }) | Select-Object -Last 1
$contractWord = switch ($gateCode) { 0 { 'PASS' } 3 { 'WARN' } 2 { 'UNVERIFIED' } default { 'FAIL' } }
$contractFindings = @($gateOut | Where-Object { $_ -match '^\s+(FAIL|WARN)\s' } | ForEach-Object { $_.Trim() })
if ($gateCode -in 0, 3) {
    $contractOk = $true
    $contractLine = "$contractWord - $gateLast"
} else {
    $contractLine = "BLOCKED ($contractWord) - $(if ($gateLast) { $gateLast } else { "contract-gate exited $gateCode" })"
}

$extVer = ""
$extPkg = "extension/package.json"
if (Test-Path $extPkg) {
    try { $extVer = (Get-Content -LiteralPath $extPkg -Raw | ConvertFrom-Json).version } catch { $extVer = "" }
}

# ── report ───────────────────────────────────────────────────
Write-Host ""
Write-Host "================ RELEASE CHECKLIST (nothing is executed) ================" -ForegroundColor Cyan
Write-Host ""
Write-Host "  branch          : $branch"
Write-Host ("  working tree    : {0}" -f $(if ($dirty) { "DIRTY - commit/clean first (run scripts/build-local.ps1)" } else { "clean" }))
Write-Host "  last app tag    : $(if ($lastVer) { $lastVer } else { '(none)' })"
Write-Host "  last ext tag    : $(if ($lastExt) { $lastExt } else { '(none)' })"
Write-Host "  app version now : $Version   ->  tag $tag"
Write-Host "  ext version now : $(if ($extVer) { $extVer } else { '(unknown)' })   (bump via: cd extension; npm run version:bump)"
Write-Host ("  gate evidence   : " + $gateLine) -ForegroundColor $(if ($gateOk) { 'Green' } else { 'Red' })
if (-not $gateOk) {
    Write-Host "                    Do NOT push the tag in step 2 until this line is green." -ForegroundColor Red
}
Write-Host ("  contract gate   : " + $contractLine) -ForegroundColor $(if ($gateCode -eq 0) { 'Green' } elseif ($contractOk) { 'Yellow' } else { 'Red' })
foreach ($f in $contractFindings) { Write-Host "                    $f" -ForegroundColor $(if ($f.StartsWith('FAIL')) { 'Red' } else { 'Yellow' }) }
if (-not $contractOk) {
    $why = if ($contractWord -eq 'UNVERIFIED') { 'run the release where the contracts catalog is reachable (AGENTS.md names it)' } else { 'fix what scripts/contract-gate.ps1 names, in the catalog first' }
    Write-Host "                    Do NOT push the tag in step 2: $why." -ForegroundColor Red
}
Write-Host ""
Write-Host "  Legend: [PAID] uses paid GitHub Actions minutes   [PUBLIC] publishes to a store/index" -ForegroundColor DarkGray
Write-Host ""

function Step([string]$n, [string]$title) { Write-Host "[$n] $title" -ForegroundColor Yellow }
function Cmd([string]$c)                  { Write-Host "      $c" -ForegroundColor Gray }
function Note([string]$c)                 { Write-Host "      - $c" -ForegroundColor DarkGray }

Step "0" "Preflight (local, free)"
Note "Working tree must be clean and all checks green."
Cmd  "./scripts/build-local.ps1 -Message ""..."""
Note "Optional real-install smoke test of the Store package:"
Cmd  "./msix/build-msix.ps1 -SelfSign"
Write-Host ""

Step "1" "Docs & site (local commit, free)"
Note "Update versioned/dated content, then commit:"
Note "README.md, docs.html / docs.ru.html / docs.uk.html, index.html, extension.html,"
Note "extension/store/LISTING.md, extension/README.md, DEV/CHANGELOG.md"
Cmd  "./scripts/build-local.ps1 -Message ""docs: release $Version"""
Write-Host ""

Step "2" "GitHub Release - app binaries  [PAID]"
Note "Precondition: the 'gate evidence' line above is green (the tag's tree is the tree the gate passed on),"
Note "and the 'contract gate' line is PASS or WARN - never FAIL, never UNVERIFIED."
Note "Pushing a v* tag triggers .github/workflows/release.yml (builds exes + GitHub Release)."
Cmd  "git tag -a $tag -m ""Release $tag"""
Cmd  "git push origin $tag"
Cmd  "gh run watch   # or: gh release view $tag --json assets"
Note "Then prove the shipped exes carry the tag's stamp (free, local):"
Cmd  "gh release download $tag -p ""*.exe"" -D temp/release-$Version"
Cmd  "./scripts/verify-exe-version.ps1 -Path (Get-ChildItem temp/release-$Version/*.exe).FullName -Expect $Version"
Note "Attach the universal installer (CI does not build it): build locally from the tag, then upload."
Note "-Tag takes the version from the tag and refuses unless HEAD is $tag and the tree is clean:"
Cmd  "./scripts/build-installer.ps1 -Tag $tag"
Cmd  "gh release upload $tag dist/doc-html-translate-setup-$Version.exe"
Write-Host ""

Step "3" "winget (Microsoft community index)  [PUBLIC]"
Note "Needs the GitHub Release from step 2 to exist (stable URL + SHA256)."
Note "ALWAYS local-install-test the manifest first (downloads the zip, verifies SHA256 - the best gate):"
Cmd  "winget install --manifest winget   # one-time: winget settings --enable LocalManifestFiles"
Cmd  "wingetcreate update SerZhyAle.DocHtmlTranslate --version $Version ``"
Cmd  "  --urls https://github.com/SerZhyAle/doc-html-translate/releases/download/$tag/doc-html-translate-$Version-windows-x64.zip ``"
Cmd  "  --submit"
Note "To ALSO change description/tags: edit winget/ then 'wingetcreate submit winget' (update copies old metadata forward)."
Note "Then sign the CLA on the PR if prompted: gh pr comment <PR#> --repo microsoft/winget-pkgs --body ""@microsoft-github-policy-service agree"""
Note "Replace the empty auto-generated PR body (wingetcreate leaves an unchecked template):"
Cmd  "gh pr edit <PR#> --repo microsoft/winget-pkgs --body-file <notes>   # version, release URL, 'SHA256 verified via winget install', tick the checklist"
Note "Details: docs/how-i-posted-this-project-to-winget.md"
Write-Host ""

Step "4" "Windows Store - MSIX  [PUBLIC]"
Note "Build the UNSIGNED package from the tag (Microsoft re-signs at certification), then upload by hand."
Note "The identity defaults to the reserved SZA.Doc-HTML-Translate; -Tag refuses unless HEAD is $tag and the tree is clean."
Note "Not while the installer build of step 2 is running - both rewrite cmd/*/resource.syso."
Cmd  "./msix/build-msix.ps1 -Tag $tag"
Note "Upload msix/out/*.msix in Partner Center (no API for create/listing). Details: msix/README.md"
Write-Host ""

Step "5" "Chrome / Edge extension  [PAID] [PUBLIC]"
Note "Chrome and Edge publish INDEPENDENTLY: separate tags, separate build-time versions."
Note "Chrome -> ext-cws-v* (publish-cws.yml); Edge -> ext-edge-v* (publish-edge.yml). Each CI run does its own npm run build (fresh stamp)."
Cmd  "git tag ext-cws-v<label>; git push origin ext-cws-v<label>    # Chrome only"
Cmd  "git tag ext-edge-v<label>; git push origin ext-edge-v<label>  # Edge only"
Note "Details: extension/PUBLISHING.md"
Write-Host ""

Step "6" "Verify"
Cmd  "gh release view $tag --json assets"
Cmd  "winget search SerZhyAle.DocHtmlTranslate   # appears ~30-60 min after winget PR merge"
Note "Confirm Store listing updated and extension live in the Chrome/Edge dashboards."
Note "Once per release, after the site is live - never per edit - tell search engines the sitemap changed:"
Note "resubmit https://serzhyale.github.io/doc-html-translate/sitemap.xml in Google Search Console and Bing Webmaster Tools."
Write-Host ""
Write-Host "========================================================================" -ForegroundColor Cyan
Write-Host "Nothing above was executed. Copy a command to run it. Full doc: DEV/RELEASE.md" -ForegroundColor Green

# Exit 1 while the tag step is blocked (gate evidence not green, or the contract gate at FAIL or
# UNVERIFIED), so a caller reading the exit code cannot mistake a blocked checklist for a ready one;
# 0 when both lines allow the tag. The read-only git probes above leave their own codes behind.
if ($gateOk -and $contractOk) { exit 0 }
exit 1
