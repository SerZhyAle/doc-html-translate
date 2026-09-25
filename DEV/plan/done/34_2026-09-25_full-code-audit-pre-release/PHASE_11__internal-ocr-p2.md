# Phase 11: S11 - internal/ocr (part 2 of 3)

**Slice:** `S11` in [`slices.json`](slices.json) · **Lines:** 2072 in 4 files ·
**Added since `41fbc1b`:** 466 · **Risk:** 22.5
**Status:** ✅ Done - findings in FINDINGS.md, tickets 36-47
**Read at commit:** 26605b9 + working tree of 2026-09-26



> Self-contained: this phase carries the whole slice procedure. The findings go to
> [`FINDINGS.md`](FINDINGS.md), the phase state to [`INDEX.md`](INDEX.md).

## Files

| File | Lines | Added since `41fbc1b` |
|------|------:|------:|
| `internal/ocr/script.go` | 166 | 23 |
| `internal/ocr/strength.go` | 32 | 0 |
| `internal/ocr/tessdata.go` | 255 | 117 |
| `internal/ocr/tesseract.go` | 1619 | 326 |

Sibling slices of the same unit, adjacent in the order: S10, S12. Read their files for context only; audit them in their own phase.

Previous-register ids this slice re-checks: O1, O2, O3, O4, O5, O6, O8.
The register is [`DEV/research/audit_2026-09-24/README.md`](../../../research/audit_2026-09-24/README.md);
its line numbers cite commit `41fbc1b`, so find each site by its symbol, not by its line.

## Rules for this phase

- **Scope.** Audit only the files above. A sibling slice's file may be read for context; a finding
  in it is written down for that slice, not fixed here.
- **Inline fix only when the proof needs no owner machine:** (a) a Go change with a test that runs
  here, run after the fix with its exit code cited, or a JS change covered by `extension/test`;
  (b) a change that preserves behaviour by construction - dead code, a missing `defer Close`, an
  unchecked error now returned, a comment. Anything touching the registry, the Windows shell, the
  MSIX or installer, a real browser, or a publishing script is a ticket. A fix in one edition whose
  code has a Go/JS twin (`configs/parity-map.json`, `docs/PARITY.md`) is a ticket, never a one-side
  fix.
- **No output-format or completion-record change, no new UI strings** inside the phase - a finding
  that needs one is a ticket.
- **Never run a publishing step.** Release scripts are read; only their local, read-only gates and
  `-WhatIf` modes may run. No tag, no push, no upload, no store call.
- **Evidence.** `conf` only when the code was read at the cited line or the behaviour was run;
  anything that depends on the OS or a third party is `plaus`. Cite `file:line` at the read commit.
- **Toolchain.** Go runs from PowerShell; the toolchain is windows/386, so `-race` is not available
  and a `go test ./tests/` out-of-memory death is rerun, not reported.

## Steps

### Step 11.1 - Prescan

- [x] Run the mechanical pass over this slice's own files and paste the verdict lines into the
      handoff notes:
  - Go packages (./internal/ocr): `go vet <pkgs>`, `$env:GOOS='linux'; go vet <pkgs>` (the
    non-Windows twins), `golangci-lint run <pkgs>` (it carries the staticcheck analyzers),
    `go test <pkgs>`.
  - Extension modules ((none)): every `extension/test/*.test.mjs` that imports one of them -
    `node --test <files>` from `extension/`.
  - Other files ((none)): PowerShell - `Invoke-ScriptAnalyzer` when installed, else a parse
    check (`[System.Management.Automation.Language.Parser]::ParseFile`); a parity pair -
    `./scripts/parity-check.ps1`.
- **Verification:** each command above has its exit code and last line recorded in the handoff
  notes; a red result is either explained as pre-existing (reproduced on the read commit) or becomes
  a finding.

### Step 11.2 - Re-check the previous register

- [x] For each id listed above, find the fixed code and decide: `still fixed`, `regressed`,
      `one edition only` (fixed in Go or JS but not its twin), or `superseded` (the code is gone or
      replaced and the defect cannot occur). Write one line per id under "Re-check" in
      `FINDINGS.md`: `- <id> - <verdict> - S11 - <file:line or test that shows it>`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` no longer lists these ids as missing.
  A `regressed` crit/high becomes a package-1 ticket at priority 90 (Step 11.4).

### Step 11.3 - Read

- [x] Read every file line by line through the layers the gates cannot see:
  - **data safety:** deletes, overwrites and renames of anything the run did not create; path
    containment; reuse of stale output;
  - **untrusted input:** archives, HTML, RTF, XML, filenames reaching a shell, the GUI's local API,
    messages into the extension;
  - **resource bounds:** process lifetime and timeouts, memory and size budgets on the 386 build,
    handles;
  - **concurrency:** shared state, goroutine and listener lifetimes, cancellation;
  - **honesty:** exit codes, messages that say done when it was not, errors swallowed
    (`DEV/research/CODE_QUALITY.md` pattern 2);
  - **platform twins and Go/JS parity:** a `*_windows.go` against its `*_nonwindows.go`; a paired
    module against its twin per `docs/PARITY.md` - drift is a finding of its own;
  - **release path only:** frozen anchors (winget `SerZhyAle.DocHtmlTranslate`, MSIX Identity
    `SZA.Doc-HTML-Translate`, Inno AppId, Go module path, the distinct Chrome / Edge ids), version
    derivation (`YY.MMDD.HHmm`, never hand-bumped), the pre-flight verdict gate, secrets, and every
    step that publishes - can it run without the owner having asked for that exact release.
- **Verification:** every file of the slice is ticked in the handoff notes as read; each finding
  has severity (crit / high / med / low), confidence (`conf` / `plaus`) and a `file:line`.

### Step 11.4 - Triage

- [x] Dedupe first: search `DEV/plan/`, `DEV/plan/done/` and the previous register for the symptom;
      a match gets a link, not a duplicate (a regression of an old id is a new ticket citing it).
- [x] (orchestrator) crit / high: a ticket in package 1 of `DEV/plan/RELEASE_QUEUE.md` (next number from its
      `next-ticket-number`, which is then bumped), or an inline fix under the rule above.
      med: inline under the rule, else a ticket in the package its area belongs to.
      low: inline, or a register line with no ticket.
- [x] (orchestrator) Write each finding under "New findings" in `FINDINGS.md`, ids continuing per area letter
      (P CLI/pipeline/support, G GUI, E EPUB/HTML, X extractors, T translation, O OCR,
      B extension, R release path, Q static checks) from the highest id in both registers:
      `- <id> - <sev> - <conf|plaus> - <finding> - <file:line> - <#NN | inline: <proof> | ->`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` reports no crit/high as untriaged.

### Step 11.5 - Close

- [x] Inline fixes land with their test run cited (command, exit code, last line); a changed Go file
      is `gofmt`-clean.
- [x] (orchestrator) Flip this phase's row in `INDEX.md` to ✅ with the severity counts, and bump `Phases: X/N` in
      `INDEX.md` and the `k/N slices` in the ticket's status line and its `RELEASE_QUEUE.md` line.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` shows `S11` as closed.

## Done criteria

- Every step `[x]`, each with its verification met.
- The findings table below is filled, and every crit/high in it has a ticket or an inline fix.

## Handoff notes

Run as one of several parallel auditors: `FINDINGS.md`, `INDEX.md`, `RELEASE_QUEUE.md` and tickets are
written by the orchestrator from this phase's report; this file records the slice's own state.

**Prescan** (PowerShell, repo root):

- `go vet ./internal/ocr/` - exit 0, no output.
- `$env:GOOS='linux'; go vet ./internal/ocr/` - exit 0, no output.
- `golangci-lint run --config configs/.golangci.yml ./internal/ocr/` - exit 0, no output.
- `go test -count=1 ./internal/ocr/` - exit 0, `ok doc-html-translate/internal/ocr 2.146s`.
- Extension modules / other files: none in this slice.

**Re-check** (sites found by symbol; S10/S12 files cited where the fix lives):

- O1 - still fixed - `download.go:94-107,124` `CheckLang` gate; `TestDownloadRefusesTraversalCode`.
- O2 - still fixed - `download.go:135-138,174` per-code lock + `os.CreateTemp`; `TestConcurrentDownloadsInstallOnePack`.
- O3 - still fixed - `download.go:32-46,170,190-199,251-265`; `TestDownloadRefusesChecksumMismatch`, `TestDownloadRefusesOversizeBody`, `TestDownloadRemovesOnlyStaleTemps`.
- O4 - still fixed - `tessdata.go:59-65` per-user `userDataDir`, `tessdata.go:94-119` `DataDir` union; `TestLayeredDataDirs`.
- O5 - still fixed - `run.go:19-31` every Tesseract call via `procrun` with `Tesseract`/`TesseractProbe` budgets (`tesseract.go:139,276`, `script.go:97`).
- O6 - still fixed - `overlay.go:576` pixel budget in `decodeImage`, `budget.go:91-115` memory-capped pool; `TestOverBudgetImageIsNotDecoded`, `TestMemoryWorkers`.
- O8 - still fixed - `script.go:91,122-129` `stageForDetection`; `TestDetectScriptStagesANonASCIIPath`.

**Files read in full:** [x] `script.go` (166) [x] `strength.go` (32) [x] `tessdata.go` (255) [x] `tesseract.go` (1619).
Context read: `download.go`, `run.go`, `budget.go`, `boundary.go`, `overlay.go` (OverlayBook, decodeImage),
`internal/pipeline/ocrstep.go`, `internal/pipeline/guard.go`, `cmd/doc-html-ui/main.go:914-959`, `cmd/doc-html-ui/ui.html:707-724,1150-1191`.

**Findings** (0 crit, 0 high, 2 med, 3 low):

| id | sev | conf | finding | where | disposition |
|----|-----|------|---------|-------|-------------|
| S11-1 | med | plaus | Every "ASCII path" handed to Tesseract lives under the user profile: temp stagings via `os.CreateTemp("")` and, since the O4 fix, `--tessdata-dir` under `%LOCALAPPDATA%`. Nothing checks those are ASCII, so a profile path outside the ANSI code page breaks the upscale/rotate/rescue passes and every run after a pack download. | `tesseract.go:686,713,297-298`, `tessdata.go:59-65`, `script.go:128` | ticket, package 1 |
| S11-2 | med | conf | The GUI always sends `-ocr-lang`, so `langFixed` is always true and the script check in `script.go` never runs from the GUI (the MSIX/Store entry point). | `cmd/doc-html-ui/main.go:916-917`, `ui.html:1157-1168`, `internal/pipeline/ocrstep.go:54`, `script.go:139` | ticket, package 1 |
| S11-3 | low | conf | `TessLang` passes through `-src` codes it cannot map (`cs`, `sv`, `hi`, `zh-CN`, `pt-BR`) and maps `nl`/`tr`/`ar` to packs outside the catalog. OCR is then skipped with the advice `-ocr-download <code>`, which `CheckLang` refuses - a dead end (CLI only). | `tessdata.go:194-211`, `ocrstep.go:36-39`, `download.go:124` | -> |
| S11-4 | low | plaus | Tesseract runs under `context.Background()`, so a cancelled OCR run waits for every in-flight pass (up to 5 per image, each budgeted up to 10 min) before `ctx` is next checked. Known deferral of ticket 11. | `run.go:24`, `tesseract.go:276`, `script.go:97` | -> |
| S11-5 | low | conf | When `DataDir` fails to stage a bundled pack into the per-user folder, `hasLangFile` drops `--tessdata-dir` and the engine falls back to its own data. `MissingLangs` already counted the pack as installed, and the staging failure only goes to the run log, so every image fails. | `tessdata.go:114-118`, `tesseract.go:297`, `tesseract.go:175-177` | -> |
