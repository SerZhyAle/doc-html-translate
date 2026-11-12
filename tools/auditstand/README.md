# auditstand - the deep dynamic audit's measuring stand

The instrument of ticket [113](../../DEV/plan/E0113_dynamic-behaviour-audit.md): it
measures what the built desktop program *does* to the user, the UI thread and the rest of Windows
while a person uses it - responsiveness, wrapped errors, no system blocking, leaks,
understandability. The strategic spec owns the why and the budgets; the tactical folder
([INDEX.md](../../DEV/plan/113_2026-10-11_dynamic-behaviour-audit/INDEX.md)) owns the phases and
the step lists; this tool is the only way a scene runs.

Like the ocrlab precedent: a developer tool. No build script compiles it, it ships in no package,
and nothing here is linked into the app. The shipped program is never modified to make it
measurable - the instrument is external.

## Use

From the repository root (PowerShell or a shell with `go` on PATH):

    go run ./tools/auditstand -SelfTest
    go run ./tools/auditstand -Fixtures
    go run ./tools/auditstand -Control [-declared-load qemu-system-x86_64.exe]
    go run ./tools/auditstand -BrowserCheck [-allow-key-inject]
    go run ./tools/auditstand -Scene S03 -isolation B-full
    go run ./tools/auditstand -Summary
    go run ./tools/auditstand -Plan

- **-SelfTest** - matrix integrity (90 cells, budgets, the n/a register), capability coverage
  re-derived from CLAUDE.md/AGENTS.md against the pinned inventory (fails in both directions: a
  capability no scene covers, a pin whose phrase vanished), fixture presence. Missing fixtures
  are `2` (the input is absent - the ocrlab precedent), a fixture that no longer matches its
  manifest hash is `1`.
- **-Fixtures** - generates the synthetic corpus deterministically into
  `temp/auditstand/fixtures/` (byte-identical across runs; `manifest.json` carries the SHA-256s).
  `-Big` adds the multi-GB fixtures behind a free-space check.
- **-Control** - runs every available probe on the idle desktop with no app started and writes
  `control.json`. Exit `2` when a probe's own jitter p95 exceeds 20% of its budget's absolute
  bar, a foreign process exceeds 10% of one core for five consecutive seconds, or the desktop
  locks/the validity sampler stalls: the machine is too noisy for scenes that day. Invalid
  attempts persist; a newer invalid control supersedes the earlier valid one that day. Probes that are unavailable are named
  advisories (the window ping needs the app's window; the keyboard-hook probe is behind
  `-allow-key-inject` and lands with phase 02). `-declared-load <image[,image..]>` names a
  background load by executable file name (the owner's qemu VM is the case it exists for): a
  declared image above the 10% window does not invalidate the control, the record carries
  `declared_load` and an advisory note, and the verdict is PASS WITH ADVISORIES. Every scene run
  against that control inherits the declaration and the advisory. A path, a wildcard or the app
  under test's own image is refused; the flag is refused on any other action.
- **-BrowserCheck** - step 00.08's verification: pre-launch the instrumented browser (it touches
  the real browser profile - the owner's approval names this step), observe the marker page
  through the debugging port, time an injected key when `-allow-key-inject` is passed,
  cross-check the capture-diff fallback on the same window, kill the instance and verify the
  port closed. Exit `2` when a running browser absorbed the launch (the fallback carries the
  timing then and the DOM-level cells become n/a).
- **-Scene** - runs a scene's implemented steps. Refuses (exit `2`) without a same-day valid
  control, without fixtures, and, for registry-touching scenes, without the Level-B named-key
  backup (phase 0 step 05 - no backup, no run). A subset run exits `3` and closes no cell.
- **-Summary** - the campaign state: every cell measured / n-a / open, open P0/P1 findings from
  FINDINGS.md. Exit 0 only when the campaign is closed by the script.

Every action ends in the shared CHECK-VERDICT vocabulary (`docs/contracts/CHECK-VERDICT.md`);
the last line is the verdict.

## Isolation

Scenes start every app process with `LOCALAPPDATA`, `APPDATA`, `TEMP`, `TMP` and `USERPROFILE`
redirected under the run's artifact directory (Level B's environment half), and the CLI's own
log store follows, so temp-residue and log-growth readings see the run and nothing else. The
registry half is implemented with `-isolation B-full`: typed snapshots, hashed exports and
recovery manifest, refusal before launch, surgical restore and verification in finally. Owned
profiles are removed; exports remain until restore confirmation. Failed registry restoration
records P0/T and halts the campaign. `B-env` is environment-only; registry-touching scenes refuse.
Level A refuses until disposable-user verification exists.

The keyboard-hook probe (its full implementation lands with S14) injects keys only when
`-allow-key-inject` is passed: a stand run uninvoked never types at the owner.

## Records

`temp/auditstand/artifacts/<run-id>/record.json` - the format is
[RUN_RECORD.md](../../DEV/plan/113_2026-10-11_dynamic-behaviour-audit/RUN_RECORD.md) (field
names are a contract; bump `format` together with it). Every run records the SHA-256 of the
binaries it started, the isolation it used, the per-probe distributions, the P4 samples, the P7
outcome checks and the cell results with their two-form budget verdicts (absolute hypothesis
**and** at most 2x the same probe's control p95). Artifacts live under gitignored `temp/` and
are deleted at campaign end - the ticket folder gets verdicts and aggregates only.

## Layout

    main.go         flag contract and verdicts
    matrix.go       scene catalog, coverage matrix, budgets H1-H14, the n/a register
    capabilities.go the pinned inventory and the two-direction check
    selftest.go     -SelfTest
    control.go      -Control and the jitter gate
    probes.go       the probe shape, the registry, the platform-neutral citizen probes
    probes_windows.go / probes_other.go  the platform half (P1, P6, P4 sampler, host ping)
    fixtures.go     the synthetic corpus and its manifest
    runenv.go       the redirected environment, output capture, the P4 sampler loop
    scenes.go       the runner discipline (control, fixtures, isolation, verdict folding)
    scene_s03.go / scene_s08.go / scene_s15.go / scene_s16.go  the implemented CLI scenes
    browserlaunch.go       the instrumented-browser prelaunch, marker server and session (step 00.08)
    browsercheck.go        -BrowserCheck: step 00.08's verification action
    capturediff_windows.go / capturediff_other.go  P3's capture-diff fallback timing
    shellnotify_windows.go  P6 shell-notification listener (H11)
    stateprobe.go   P8 state-diff primitives and the H14 toggle verdict
    outputs.go      shared readers of what a CLI run left on disk
    validity.go     the foreign-load validity rule (names the offending images, tolerates a declared load)
    slope.go        P5: OLS slope with its 95% confidence band
    record.go       the run-record format
    summary.go      -Summary

## Current increment and remaining work

S11's `-dry-run` prints the backup inventory without executing the scene. `-Recover <manifest>`
and read-only `-VerifyRecovery <manifest>` require `-recovery-sha256 <hash from record.json>`;
for renamed binaries supply the same `-app`/`-ui` names used for backup. Registry restore does
not remove retained exports or clear a campaign halt automatically.

Records are format 2. Closure requires verified full isolation, tree stability, all declared
budget evidence and an approved same-day control. Baseline sign-off recording is still part of
Phase 0's pending work; controls default to unsigned. Empty and subset scenes remain advisory.
No implemented scene currently closes the whole scene matrix.

`browser_probe.go` provides a loopback-only CDP transport, DOM/heap counters and a Tab focus-change
measurement helper. `browserlaunch.go` (step 00.08, owner-approved) pre-launches the browser the
app's own `openAppWindow` will pick, with a debugging port and a stand-owned marker page on
loopback; the session owns only what it started (an absorbed launch reports the fallback sentinel
and kills nothing), and closing verifies the port shut. `capturediff_windows.go` is P3's
capture-diff fallback timing. `-BrowserCheck` is the step's verification action - its live run
needs the same quiet owner-present window as the scenes. Synthetic target and stub-browser tests
verify transport, counters, prelaunch, the merged-launch sentinel and the port-close discipline.
Instrumented-browser desktop wiring into the GUI scenes, the keyboard-hook negative control, UI
Automation and the GUI scenes themselves remain pending in the tactical plan.

The third increment adds the CLI scenes S08 (translate-guards) and S15 (big-inputs), two more S16
members (`ocr-image-failure`, `missing-filedo`), the shell-notification listener
(`P6.shell-notify`), the P8 state-diff primitives, and names the foreign images in a rejected
control. The scenes are tested against a stub CLI: the test binary doubles as the app
(`stubcli_test.go`, switched on by `AUDITSTAND_STUB_CLI=1`), so what the stand calls a pass, a
leak or an English message under `-ui-lang ru` is pinned without a real binary or desktop.
`-Control` refuses to run while any `doc-html-ui.exe` / `doc-html-translate.exe` is active - the
owner's installed build included; the stand never closes a program it did not start.

Clipboard control measures reader contention without retaining or replacing content. Clipboard
writer coverage requires a disposable desktop and reports unavailable on the owner's desktop.
The fsync control overwrites one bounded scratch file and is paced, including during scenes.

The second increment adds multi-frame/over-cap TIFF fixtures, fixes both PDF metadata dates and
catalog ordering, protects concurrent stdout/stderr and citizen-probe capture, checks fixture
manifest hashes, and preserves invalid control evidence. See the tactical folder's BASELINE.md
for the authorized control attempt: invalid due to foreign CPU load, not a product finding.
