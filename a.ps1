#!/usr/bin/env pwsh
<#
.SYNOPSIS
    doc-html-translate project scripts launcher (analogue of the Android a.ps1).
.DESCRIPTION
    Quick alias launcher for common project scripts. The alias VOCABULARY is inherited
    from the Android project's a.ps1 (c / cc / ch / d / db / nd / nl / id / b ..) so the
    muscle memory carries across both repos; each alias is mapped to the nearest operation
    in this repo. Everything you type after the command is forwarded verbatim to the target
    script (so `a cc -Message ".."` reaches build-local's -Message).

    Flavor mapping: this repo frames the desktop app and the browser extension as two
    "editions" of one product (see docs/PARITY.md), which is the analogue of the Android
    standard / noLegal flavors:
      standard flavor  -> desktop app (Go, CLI + GUI + MSIX)   ->  d / db / b / id
      alternate flavor -> browser extension (JS)               ->  nd / nl

    Nothing here publishes. All commands are local + free; even `c` only pushes a BRANCH
    (every CI workflow fires on tags, so a branch push costs nothing). Publishing
    ("релиз") stays a manual, per-step flow - see 'r'.

      COMMIT
        c    - quick commit + push branch (auto timestamp msg)       (scripts/commit-push.ps1)
        cc   - quick commit, no push (auto timestamp msg)            (scripts/commit-push.ps1 -NoPush)
        bl   - gated "сборка": test+lint+typo + build + commit       (scripts/build-local.ps1)   [-Message ".."]

      CHECK
        ch   - check: test + lint + typo (the gate)                  (scripts/check.ps1)
        fc   - alias for ch (fast code check)
        fu   - fast unit-test suite (go test ./...)                  (scripts/test.ps1)
        t    - alias for fu (tests)
        l    - golangci-lint only                                    (scripts/lint.ps1)
        ty   - typos spell-check only                                (scripts/typo.ps1)

      BUILD (desktop app = standard flavor)
        d    - fast build: both exes + stamp extension -> _APP       (build.ps1)
        db   - fast build, desktop only (no extension)               (build.ps1 -NoExtension)
        bd   - alias for db
        dc   - gated build (runs the gate first)                     (build.ps1 -Gate)
        cd   - alias for dc
        b    - build all: exes + extension + store zip               (build.ps1 -ExtZip)
        bp   - alias for b
        cli  - build the CLI exe only                                (scripts/build.ps1)
        ui   - build the GUI exe only                                (scripts/build-ui.ps1)
        inst - build universal setup.exe (Inno Setup, x86+x64)       (scripts/build-installer.ps1)

      EXTENSION (alternate edition = noLegal analog)
        nd   - extension dev build (stamp + test + vendor)           (scripts/build-extension.ps1)
        nl   - extension store zip (release of the extension)        (scripts/build-extension.ps1 -Zip)
        ext  - alias for nd
        extz - alias for nl

      INSTALL / MSIX
        id   - install locally: build MSIX -> uninstall -> install -> launch   (reinstall.ps1)
        ind  - install the last MSIX, skip rebuild                   (reinstall.ps1 -SkipBuild)
        ivn  - alias for ind
        msix - build the UNSIGNED Store package                      (msix/build-msix.ps1)
        sign - build the SELF-SIGNED local package                  (msix/build-msix.ps1 -SelfSign)

      RELEASE / TOOLING
        r    - print the release checklist (RUNS NOTHING)            (scripts/release.ps1)
        s    - setup dev tools (golangci-lint + typos)              (scripts/bootstrap-tools.ps1)
        icon - regenerate assets/doc-html-translate.ico             (scripts/generate-icon.ps1)
        log  - append a DEV/CHANGELOG.md row (-Path -Target -Description)
.EXAMPLE
    .\a.ps1 ch
.EXAMPLE
    .\a.ps1 cc                                        # quick commit (auto timestamp message), no push
.EXAMPLE
    .\a.ps1 cc "fix: navbar position"                # quick commit with a message, no push
.EXAMPLE
    .\a.ps1 c                                         # quick commit AND push the branch
.EXAMPLE
    .\a.ps1 ind                                       # reinstall the last MSIX, no rebuild
.EXAMPLE
    .\a.ps1 id -NoLaunch                              # any target flag forwards: install without launching
#>

param(
    [string]$Command = ''
)

# Everything after <Command> is forwarded verbatim to the target script (after its
# preset Args), e.g. `.\a.ps1 cc -Message ".."` or `.\a.ps1 msix -IdentityName X`.
# Reading the automatic $args (simple-script mode) rather than a declared
# ValueFromRemainingArguments parameter is deliberate: the automatic $args re-splats
# with -flag tokens INTACT (a literal array would splat positionally and swallow flags),
# so forwarded parameters still bind by name.
$Rest = $args

$ErrorActionPreference = "Stop"

# Repo root = this script's folder. Every target script uses paths relative to the repo
# root (./scripts/.., configs/.., temp/logs), so we run from here regardless of caller CWD.
$ProjectRoot = $PSScriptRoot

# Command table - single source of truth for BOTH dispatch and the help text, so the two
# can never drift (the Android original duplicates its help list by hand).
#
# Args MUST be a hashtable, not a string array: `& $script @hashtableArgs` always binds by
# name, while a string-array splat would leak `-foo` presets positionally. Empty hashtable
# means "no preset args". Each value is $true for a switch, or a string/int for a typed param.
$scripts = [ordered]@{
    # --- commit ---
    'c'    = @{ Group = 'Commit';        Path = 'scripts/commit-push.ps1';      Args = @{};                   Desc = 'quick commit + push branch (auto timestamp msg)' }
    'cc'   = @{ Group = 'Commit';        Path = 'scripts/commit-push.ps1';      Args = @{ NoPush = $true };   Desc = 'quick commit, no push (auto timestamp msg)' }
    'bl'   = @{ Group = 'Commit';        Path = 'scripts/build-local.ps1';      Args = @{};                   Desc = 'gated "сборка": gate + build + commit (needs -Message)' }

    # --- check ---
    'ch'   = @{ Group = 'Check';         Path = 'scripts/check.ps1';            Args = @{};                   Desc = 'check: test + lint + typo (the gate)' }
    'fc'   = @{ Group = 'Check';         Path = 'scripts/check.ps1';            Args = @{};                   Desc = 'alias for ch' }
    'fu'   = @{ Group = 'Check';         Path = 'scripts/test.ps1';             Args = @{};                   Desc = 'fast unit-test suite (go test ./...)' }
    't'    = @{ Group = 'Check';         Path = 'scripts/test.ps1';             Args = @{};                   Desc = 'alias for fu (tests)' }
    'l'    = @{ Group = 'Check';         Path = 'scripts/lint.ps1';             Args = @{};                   Desc = 'golangci-lint only' }
    'ty'   = @{ Group = 'Check';         Path = 'scripts/typo.ps1';             Args = @{};                   Desc = 'typos spell-check only' }

    # --- build: desktop app (standard flavor) ---
    'd'    = @{ Group = 'Build (app)';   Path = 'build.ps1';                    Args = @{};                   Desc = 'fast build: both exes + stamp extension -> _APP' }
    'db'   = @{ Group = 'Build (app)';   Path = 'build.ps1';                    Args = @{ NoExtension = $true }; Desc = 'fast build, desktop only (no extension)' }
    'bd'   = @{ Group = 'Build (app)';   Path = 'build.ps1';                    Args = @{ NoExtension = $true }; Desc = 'alias for db' }
    'dc'   = @{ Group = 'Build (app)';   Path = 'build.ps1';                    Args = @{ Gate = $true };     Desc = 'gated build (runs the gate first)' }
    'cd'   = @{ Group = 'Build (app)';   Path = 'build.ps1';                    Args = @{ Gate = $true };     Desc = 'alias for dc' }
    'b'    = @{ Group = 'Build (app)';   Path = 'build.ps1';                    Args = @{ ExtZip = $true };   Desc = 'build all: exes + extension + store zip' }
    'bp'   = @{ Group = 'Build (app)';   Path = 'build.ps1';                    Args = @{ ExtZip = $true };   Desc = 'alias for b' }
    'cli'  = @{ Group = 'Build (app)';   Path = 'scripts/build.ps1';            Args = @{};                   Desc = 'build the CLI exe only' }
    'ui'   = @{ Group = 'Build (app)';   Path = 'scripts/build-ui.ps1';         Args = @{};                   Desc = 'build the GUI exe only' }
    'inst' = @{ Group = 'Build (app)';   Path = 'scripts/build-installer.ps1';  Args = @{};                   Desc = 'build universal setup.exe (Inno Setup, x86+x64)' }

    # --- extension (alternate edition = noLegal analog) ---
    'nd'   = @{ Group = 'Extension';     Path = 'scripts/build-extension.ps1';  Args = @{};                   Desc = 'extension dev build (stamp + test + vendor)' }
    'nl'   = @{ Group = 'Extension';     Path = 'scripts/build-extension.ps1';  Args = @{ Zip = $true };      Desc = 'extension store zip (release of the extension)' }
    'ext'  = @{ Group = 'Extension';     Path = 'scripts/build-extension.ps1';  Args = @{};                   Desc = 'alias for nd' }
    'extz' = @{ Group = 'Extension';     Path = 'scripts/build-extension.ps1';  Args = @{ Zip = $true };      Desc = 'alias for nl' }

    # --- install / MSIX ---
    'id'   = @{ Group = 'Install / MSIX';Path = 'reinstall.ps1';                Args = @{};                   Desc = 'install locally: build MSIX -> uninstall -> install -> launch' }
    'ind'  = @{ Group = 'Install / MSIX';Path = 'reinstall.ps1';                Args = @{ SkipBuild = $true };Desc = 'install the last MSIX, skip rebuild' }
    'ivn'  = @{ Group = 'Install / MSIX';Path = 'reinstall.ps1';                Args = @{ SkipBuild = $true };Desc = 'alias for ind' }
    'msix' = @{ Group = 'Install / MSIX';Path = 'msix/build-msix.ps1';          Args = @{};                   Desc = 'build the UNSIGNED Store package' }
    'sign' = @{ Group = 'Install / MSIX';Path = 'msix/build-msix.ps1';          Args = @{ SelfSign = $true }; Desc = 'build the SELF-SIGNED local package' }

    # --- release / tooling ---
    'r'    = @{ Group = 'Release / tool';Path = 'scripts/release.ps1';          Args = @{};                   Desc = 'print the release checklist (RUNS NOTHING)' }
    's'    = @{ Group = 'Release / tool';Path = 'scripts/bootstrap-tools.ps1';  Args = @{};                   Desc = 'setup dev tools (golangci-lint + typos)' }
    'icon' = @{ Group = 'Release / tool';Path = 'scripts/generate-icon.ps1';    Args = @{};                   Desc = 'regenerate assets/doc-html-translate.ico' }
    'log'  = @{ Group = 'Release / tool';Path = 'scripts/add_to_dev_log.ps1';   Args = @{};                   Desc = 'append a DEV/CHANGELOG.md row (-Path -Target -Description)' }
}

function Show-Help {
    Write-Host ""
    Write-Host "doc-html-translate launcher - short aliases (inherited from the Android a.ps1)" -ForegroundColor Cyan
    Write-Host ""
    $lastGroup = ''
    foreach ($key in $scripts.Keys) {
        $entry = $scripts[$key]
        if ($entry.Group -ne $lastGroup) {
            Write-Host ""
            Write-Host "  $($entry.Group)" -ForegroundColor Yellow
            $lastGroup = $entry.Group
        }
        Write-Host ("    {0,-5} - {1}" -f $key, $entry.Desc) -ForegroundColor Cyan
    }
    Write-Host ""
    Write-Host "Usage:   .\a.ps1 <command> [extra args forwarded to the script]" -ForegroundColor Gray
    Write-Host "Example: .\a.ps1 ch" -ForegroundColor Gray
    Write-Host "Example: .\a.ps1 cc                       # quick commit, no push (auto message)" -ForegroundColor Gray
}

# Explicit help request (also: unknown / empty command prints help below).
if ($Command -in @('h', 'help', '-h', '--help', '?', '/?')) {
    Show-Help
    exit 0
}

# Validate command
if (-not $scripts.Contains($Command)) {
    if ($Command) {
        Write-Host "Unknown command: $Command" -ForegroundColor Red
    } else {
        Write-Host "No command given." -ForegroundColor Red
    }
    Show-Help
    exit 1
}

# Resolve script path
$scriptEntry = $scripts[$Command]
$scriptPath  = Join-Path $ProjectRoot $scriptEntry.Path
$scriptArgs  = $scriptEntry.Args

if (-not (Test-Path $scriptPath)) {
    Write-Host "Script not found: $scriptPath" -ForegroundColor Red
    exit 1
}

# Human-readable echo of what is about to run
$argsDisplay = ($scriptArgs.GetEnumerator() | ForEach-Object {
        if ($_.Value -is [bool]) { "-$($_.Key)" } else { "-$($_.Key) $($_.Value)" }
    }) -join ' '
Write-Host "Executing: $($scriptEntry.Path) $argsDisplay $($Rest -join ' ')".TrimEnd() -ForegroundColor Green
Write-Host ""

# Run from the repo root so the target script's relative paths resolve.
Set-Location $ProjectRoot

# Two splats in one call: preset hashtable (bind by name) + forwarded automatic-$args
# array (keeps -flags). See the $Rest note above for why the automatic $args is required.
& $scriptPath @scriptArgs @Rest

# Return the exit code from the executed script.
exit $LASTEXITCODE
