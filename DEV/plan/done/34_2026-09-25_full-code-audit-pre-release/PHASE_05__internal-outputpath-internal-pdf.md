# Phase 05: S05 - internal/outputpath .. internal/pdf

**Slice:** `S05` in [`slices.json`](slices.json) · **Lines:** 2895 in 15 files ·
**Added since `41fbc1b`:** 1355 · **Risk:** 37
**Status:** ✅ Done - findings in FINDINGS.md, tickets 36-47
**Read at commit:** 26605b9 + working tree of 2026-09-26



> Self-contained: this phase carries the whole slice procedure. The findings go to
> [`FINDINGS.md`](FINDINGS.md), the phase state to [`INDEX.md`](INDEX.md).

## Files

| File | Lines | Added since `41fbc1b` |
|------|------:|------:|
| `internal/outputpath/completion.go` | 221 | 221 |
| `internal/outputpath/lock.go` | 92 | 92 |
| `internal/outputpath/outputpath.go` | 75 | 27 |
| `internal/outputpath/ownership.go` | 265 | 265 |
| `internal/outputpath/proc_nonwindows.go` | 21 | 21 |
| `internal/outputpath/proc_windows.go` | 38 | 38 |
| `internal/pdf/extract.go` | 958 | 63 |
| `internal/pdf/images.go` | 488 | 488 |
| `internal/pdf/pdftotext.go` | 46 | 46 |
| `internal/pdf/pdftotext_nonwindows.go` | 30 | 30 |
| `internal/pdf/pdftotext_windows.go` | 46 | 46 |
| `internal/pdf/toc.go` | 95 | 0 |
| `extension/src/pdf-images.js` | 230 | 18 |
| `extension/src/reflow.js` | 231 | 0 |
| `extension/src/toc.js` | 59 | 0 |

None - this slice holds its units whole.

Previous-register ids this slice re-checks: P1, P6, P16, P23, X6, X7, X8, X9, X10, X11, X14, B9.
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

### Step 05.1 - Prescan

- [x] Run the mechanical pass over this slice's own files and paste the verdict lines into the
      handoff notes:
  - Go packages (./internal/outputpath ./internal/pdf): `go vet <pkgs>`, `$env:GOOS='linux'; go vet <pkgs>` (the
    non-Windows twins), `golangci-lint run <pkgs>` (it carries the staticcheck analyzers),
    `go test <pkgs>`.
  - Extension modules (extension/src/pdf-images.js extension/src/reflow.js extension/src/toc.js): every `extension/test/*.test.mjs` that imports one of them -
    `node --test <files>` from `extension/`.
  - Other files ((none)): PowerShell - `Invoke-ScriptAnalyzer` when installed, else a parse
    check (`[System.Management.Automation.Language.Parser]::ParseFile`); a parity pair -
    `./scripts/parity-check.ps1`.
- **Verification:** each command above has its exit code and last line recorded in the handoff
  notes; a red result is either explained as pre-existing (reproduced on the read commit) or becomes
  a finding.

### Step 05.2 - Re-check the previous register

- [x] For each id listed above, find the fixed code and decide: `still fixed`, `regressed`,
      `one edition only` (fixed in Go or JS but not its twin), or `superseded` (the code is gone or
      replaced and the defect cannot occur). Write one line per id under "Re-check" in
      `FINDINGS.md`: `- <id> - <verdict> - S05 - <file:line or test that shows it>`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` no longer lists these ids as missing.
  A `regressed` crit/high becomes a package-1 ticket at priority 90 (Step 05.4).

### Step 05.3 - Read

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

### Step 05.4 - Triage

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

### Step 05.5 - Close

- [x] Inline fixes land with their test run cited (command, exit code, last line); a changed Go file
      is `gofmt`-clean.
- [x] Flip this phase's row in `INDEX.md` to ✅ with the severity counts, and bump `Phases: X/N` in
      `INDEX.md` and the `k/N slices` in the ticket's status line and its `RELEASE_QUEUE.md` line.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` shows `S05` as closed.

## Done criteria

- Every step `[x]`, each with its verification met.
- The findings table below is filled, and every crit/high in it has a ticket or an inline fix.

## Handoff notes

Prescan (run from PowerShell on the read commit + working tree):

- `go vet ./internal/outputpath ./internal/pdf` - exit 0, no output.
- `$env:GOOS='linux'; go vet ./internal/outputpath ./internal/pdf` - exit 0, no output.
- `golangci-lint run --config configs/.golangci.yml ./internal/outputpath/ ./internal/pdf/` - exit 1 before the fix
  (`internal\outputpath\proc_windows.go:18:27: Error return value of windows.CloseHandle is not checked (errcheck)`),
  exit 0 after inline fix S05-1.
- `go test ./internal/outputpath ./internal/pdf` - exit 1, `--- FAIL: TestExtract_Volume3ImagesOnTheirPages: spine has 4 pages, want 10`.
  Pre-existing at the read commit (no working-tree diff in `internal/pdf` or `internal/textutil`); the cause is S05-2.
  `go test ./internal/outputpath` - `ok`, exit 0. `go test -skip TestExtract_Volume3ImagesOnTheirPages ./internal/pdf` - `ok`, exit 0.
- From `extension/`: `node --test test/pdf-images.test.mjs test/reflow.test.mjs test/lifecycle.test.mjs test/viewer.test.mjs` - exit 0, `pass 37, fail 0`.

Re-check: P1, P6, P16, P23, X6, X7, X8, X9, X10, X11, X14, B9 - all still fixed (evidence lines in the orchestrator report).

Files read in full: [x] completion.go [x] lock.go [x] outputpath.go [x] ownership.go [x] proc_nonwindows.go
[x] proc_windows.go [x] extract.go [x] images.go [x] pdftotext.go [x] pdftotext_nonwindows.go
[x] pdftotext_windows.go [x] toc.go [x] pdf-images.js [x] reflow.js [x] toc.js

Findings (local ids S05-n; register ids are assigned in FINDINGS.md): crit 0, high 0, med 3, low 3. No crit/high, so no package-1 ticket.

- S05-1 - low - conf - errcheck red: unchecked `CloseHandle` - `proc_windows.go:18` - inline (behaviour-preserving; lint exit 0, `go test ./internal/outputpath` exit 0)
- S05-2 - med - conf - the ligature filter drops real short text blocks on the pdftotext path and turns `TestExtract_Volume3ImagesOnTheirPages` red - `extract.go:306,386-396`, `reflow.js:48-53,133` - ticket (parity twin)
- S05-3 - med - conf - PDF CMYK rasters are written as `.tif`: Chrome cannot show them, and the file is flipped on disk and again by CSS - `images.go:351,401-437`, `extract.go:426,778,867-873` - ticket (output format)
- S05-4 - med - plaus - the MRC `/Mask` preference exists only in Go; JS keeps the largest raster and PARITY.md does not record the rule - `images.go:249-254` vs `pdf-images.js:141-155` - ticket (parity)
- S05-5 - low - conf - stale-lock takeover race: removal by path can delete a lock another run has just taken - `lock.go:268-271` - register line or ticket
- S05-6 - low - conf - the JPX warning dialog is English-only (no `i18n.S`) and can appear twice per PDF - `images.go:293-303` - ticket (UI strings)
