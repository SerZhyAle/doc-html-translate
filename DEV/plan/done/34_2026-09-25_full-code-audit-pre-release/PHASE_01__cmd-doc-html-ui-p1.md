# Phase 01: S01 - cmd/doc-html-ui (part 1 of 2)

**Slice:** `S01` in [`slices.json`](slices.json) · **Lines:** 2170 in 10 files ·
**Added since `41fbc1b`:** 1276 · **Risk:** 124.4
**Status:** ✅ Done - findings in FINDINGS.md, tickets 36-47
**Read at commit:** 26605b9 + working tree of 2026-09-26



> Self-contained: this phase carries the whole slice procedure. The findings go to
> [`FINDINGS.md`](FINDINGS.md), the phase state to [`INDEX.md`](INDEX.md).

## Files

| File | Lines | Added since `41fbc1b` |
|------|------:|------:|
| `cmd/doc-html-ui/failure.go` | 43 | 43 |
| `cmd/doc-html-ui/guard.go` | 142 | 142 |
| `cmd/doc-html-ui/hide_other.go` | 7 | 0 |
| `cmd/doc-html-ui/hide_windows.go` | 17 | 6 |
| `cmd/doc-html-ui/liveness.go` | 89 | 89 |
| `cmd/doc-html-ui/main.go` | 1223 | 483 |
| `cmd/doc-html-ui/proc_other.go` | 23 | 23 |
| `cmd/doc-html-ui/proc_windows.go` | 61 | 61 |
| `cmd/doc-html-ui/report.go` | 149 | 13 |
| `cmd/doc-html-ui/run.go` | 416 | 416 |

Sibling slices of the same unit, adjacent in the order: S02. Read their files for context only; audit them in their own phase.

Previous-register ids this slice re-checks: G1, G2, G3, G4, G5, G6, G7, G8, G9, G10, G11, G12, G13, G14, G16.
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

### Step 01.1 - Prescan

- [x] Run the mechanical pass over this slice's own files and paste the verdict lines into the
      handoff notes:
  - Go packages (./cmd/doc-html-ui): `go vet <pkgs>`, `$env:GOOS='linux'; go vet <pkgs>` (the
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

### Step 01.2 - Re-check the previous register

- [x] For each id listed above, find the fixed code and decide: `still fixed`, `regressed`,
      `one edition only` (fixed in Go or JS but not its twin), or `superseded` (the code is gone or
      replaced and the defect cannot occur). Write one line per id under "Re-check" in
      `FINDINGS.md`: `- <id> - <verdict> - S01 - <file:line or test that shows it>`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` no longer lists these ids as missing.
  A `regressed` crit/high becomes a package-1 ticket at priority 90 (Step 01.4).

### Step 01.3 - Read

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

### Step 01.4 - Triage

- [x] Dedupe first: search `DEV/plan/`, `DEV/plan/done/` and the previous register for the symptom;
      a match gets a link, not a duplicate (a regression of an old id is a new ticket citing it).
- [x] crit / high: a ticket in package 1 of `DEV/plan/RELEASE_QUEUE.md` (next number from its
      `next-ticket-number`, which is then bumped), or an inline fix under the rule above.
      med: inline under the rule, else a ticket in the package its area belongs to.
      low: inline, or a register line with no ticket.
- [x] Write each finding under "New findings" in `FINDINGS.md`, ids continuing per area letter
      (P CLI/pipeline/support, G GUI, E EPUB/HTML, X extractors, T translation, O OCR,
      B extension, R release path, Q static checks) from the highest id in both registers:
      `- <id> - <sev> - <conf|plaus> - <finding> - <file:line> - <#NN | inline: <proof> | ->`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` reports no crit/high as untriaged.

### Step 01.5 - Close

- [x] Inline fixes land with their test run cited (command, exit code, last line); a changed Go file
      is `gofmt`-clean.
- [x] Flip this phase's row in `INDEX.md` to ✅ with the severity counts, and bump `Phases: X/N` in
      `INDEX.md` and the `k/N slices` in the ticket's status line and its `RELEASE_QUEUE.md` line.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` shows `S01` as closed.

## Done criteria

- Every step `[x]`, each with its verification met.
- The findings table below is filled, and every crit/high in it has a ticket or an inline fix.

## Handoff notes

Prescan (before fixes):

- `go vet ./cmd/doc-html-ui/` - exit 0, no output
- `GOOS=linux go vet ./cmd/doc-html-ui/` - exit 0, no output
- `golangci-lint run --config configs/.golangci.yml ./cmd/doc-html-ui/` - exit 1, errcheck at
  `proc_windows.go:37:27` and `proc_windows_test.go:12:27` (`windows.CloseHandle` unchecked; pre-existing,
  both files match HEAD) -> S01-1
- `go test ./cmd/doc-html-ui/` - exit 0, `ok doc-html-translate/cmd/doc-html-ui`
- Extension modules / other files: none in this slice.

After the inline fixes: `go test -count=1 ./cmd/doc-html-ui/` exit 0 (`ok .. 7.231s`); both vets exit 0;
`gofmt -l cmd/doc-html-ui/` clean; lint exit 1 only on `proc_windows_test.go:12:27` (not a slice file, not edited).

Files read in full: failure.go, guard.go, hide_other.go, hide_windows.go, liveness.go, main.go (1-1224),
proc_other.go, proc_windows.go, report.go, run.go. Context only: store.go, ui.html (grep), internal/browser,
internal/config/flags.go ParseArgs, internal/ocr/download.go progress.

Re-check: G1 G3 G4 G5 G6 G7 G8 G10 G11 G13 G14 G16 still fixed; G2 still fixed (ShellExecute via
internal/browser); G9 still fixed with ticket 05's documented no-retention deviation; G12 still fixed for
settings (store.go), the history file is gone.

Findings (all low): S01-1 lint errcheck (inline, slice half); S01-2 a late Cancel reported a finished run as
cancelled (inline: run.go `outcome` + new `outcome_test.go`); S01-3 relay memory bound is 256 x 8 MiB;
S01-4 no browser fallback when Edge fails to start; S01-5 delete-output ignores the GUI's own run registry.
No tickets filed from this phase (the orchestrator merges FINDINGS/INDEX).
