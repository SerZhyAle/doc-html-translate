# Phase 10: S10 - internal/ocr (part 1 of 3)

**Slice:** `S10` in [`slices.json`](slices.json) · **Lines:** 2141 in 8 files ·
**Added since `41fbc1b`:** 762 · **Risk:** 19
**Status:** ✅ Done - findings in FINDINGS.md, tickets 36-47
**Read at commit:** 26605b9 + working tree of 2026-09-26



> Self-contained: this phase carries the whole slice procedure. The findings go to
> [`FINDINGS.md`](FINDINGS.md), the phase state to [`INDEX.md`](INDEX.md).

## Files

| File | Lines | Added since `41fbc1b` |
|------|------:|------:|
| `internal/ocr/boundary.go` | 109 | 109 |
| `internal/ocr/budget.go` | 115 | 115 |
| `internal/ocr/diag.go` | 115 | 15 |
| `internal/ocr/download.go` | 265 | 265 |
| `internal/ocr/exif.go` | 205 | 12 |
| `internal/ocr/overlay.go` | 921 | 190 |
| `internal/ocr/run.go` | 51 | 51 |
| `internal/ocr/screen.go` | 360 | 5 |

Sibling slices of the same unit, adjacent in the order: S11, S12. Read their files for context only; audit them in their own phase.

Previous-register ids this slice re-checks: O5, O6, O7, O8, O9, O10, O11, O12.
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

### Step 10.1 - Prescan

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

### Step 10.2 - Re-check the previous register

- [x] For each id listed above, find the fixed code and decide: `still fixed`, `regressed`,
      `one edition only` (fixed in Go or JS but not its twin), or `superseded` (the code is gone or
      replaced and the defect cannot occur). Write one line per id under "Re-check" in
      `FINDINGS.md`: `- <id> - <verdict> - S10 - <file:line or test that shows it>`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` no longer lists these ids as missing.
  A `regressed` crit/high becomes a package-1 ticket at priority 90 (Step 10.4).

### Step 10.3 - Read

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

### Step 10.4 - Triage

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

### Step 10.5 - Close

- [x] Inline fixes land with their test run cited (command, exit code, last line); a changed Go file
      is `gofmt`-clean.
- [x] Flip this phase's row in `INDEX.md` to ✅ with the severity counts, and bump `Phases: X/N` in
      `INDEX.md` and the `k/N slices` in the ticket's status line and its `RELEASE_QUEUE.md` line.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` shows `S10` as closed.

## Done criteria

- Every step `[x]`, each with its verification met.
- The findings table below is filled, and every crit/high in it has a ticket or an inline fix.

## Handoff notes

Run as one of several parallel auditors: FINDINGS.md, INDEX.md and RELEASE_QUEUE.md are written by
the orchestrator from these notes (Step 10.5's INDEX flip is left to it).

**Prescan** (PowerShell, `P:\WINDOWS\EPUB_2_HTML`):

- `go vet ./internal/ocr/` - exit 0, no output.
- `$env:GOOS='linux'; go vet ./internal/ocr/` - exit 0, no output.
- `golangci-lint run --config configs/.golangci.yml ./internal/ocr/` - exit 0, no output.
- `go test ./internal/ocr/` - exit 0, `ok doc-html-translate/internal/ocr 2.048s`.
- Extension modules / other files: none in this slice.

**Re-check** (sites found by symbol):

- O5 - still fixed - S10 - `run.go:19-31` every Tesseract call goes through `procrun.Run` with a `procrun.Budget` deadline
- O6 - still fixed - S10 - `overlay.go:569-587` header pixel budget before decode; `budget.go:82-115` pool width capped by memory
- O7 - still fixed - S10 - `run.go:41-51` `recognizeSafe` recover per image; pipeline `guard.go:41-49` around phase 3
- O8 - still fixed - S10 - `script.go:122-129` `stageForDetection` stages to an ASCII path
- O9 - still fixed - S10 - `run.go:15` `OMP_THREAD_LIMIT=1`; `run_test.go:41`
- O10 - still fixed - S10 - `overlay.go:205-207` `renderHTMLFile` writes via atomic `fsutil.Write`
- O11 - still fixed - S10 - `overlay.go:333-350` `localImageFile` strips `?`/`#` and percent-decodes
- O12 - still fixed - S10 - `overlay.go:317,824-831,873-879,913-920` marker-based `findInjected` + `isWrapped`

**Files read in full:** boundary.go, budget.go, diag.go, download.go, exif.go, overlay.go, run.go,
screen.go (all 8).

**Findings** (0 crit, 0 high, 0 med, 2 low):

- S10-1 - low - conf - `recognizePaths` evaluated `poolWorkers(paths)` in the loop condition, so
  every image header of the book was re-read once per worker started (up to 17 full passes) before
  recognition began - `overlay.go:407-410` - inline: hoisted into `workers`; `go test -count=1
  ./internal/ocr/` exit 0 (`ok ... 13.634s`), `gofmt -l internal/ocr/` clean, `go vet` exit 0.
- S10-2 - low - plaus - `localImageFile` joins the `<img src>` onto the page dir with no containment,
  so a `../../..` src in book content makes OCR read an image outside the output folder and write its
  text into the output page - `overlay.go:333-350` - ticket (no inline: changes which images are
  recognized; the JS twin loads through the browser and needs its own decision).
