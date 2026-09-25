# Phase 08: S08 - internal/pipeline .. internal/syslocale

**Slice:** `S08` in [`slices.json`](slices.json) · **Lines:** 2969 in 21 files ·
**Added since `41fbc1b`:** 2144 · **Risk:** 26.1
**Status:** ✅ Done - findings in FINDINGS.md, tickets 36-47
**Read at commit:** 26605b9 + working tree of 2026-09-26



> Self-contained: this phase carries the whole slice procedure. The findings go to
> [`FINDINGS.md`](FINDINGS.md), the phase state to [`INDEX.md`](INDEX.md).

## Files

| File | Lines | Added since `41fbc1b` |
|------|------:|------:|
| `internal/pipeline/cost.go` | 89 | 89 |
| `internal/pipeline/guard.go` | 49 | 49 |
| `internal/pipeline/ocrstep.go` | 113 | 113 |
| `internal/pipeline/outputdir.go` | 81 | 81 |
| `internal/pipeline/pipeline.go` | 379 | 137 |
| `internal/pipeline/reuse.go` | 25 | 25 |
| `internal/pipeline/translate.go` | 348 | 348 |
| `internal/procrun/budget.go` | 83 | 83 |
| `internal/procrun/procrun.go` | 184 | 184 |
| `internal/procrun/tree_nonwindows.go` | 38 | 38 |
| `internal/procrun/tree_windows.go` | 67 | 67 |
| `internal/report/archive.go` | 124 | 19 |
| `internal/report/environment.go` | 70 | 2 |
| `internal/report/redact.go` | 79 | 0 |
| `internal/report/store.go` | 164 | 48 |
| `internal/rtf/codepage.go` | 141 | 141 |
| `internal/rtf/extract.go` | 127 | 6 |
| `internal/rtf/parse.go` | 382 | 382 |
| `extension/src/rtf.js` | 367 | 332 |
| `internal/syslocale/locale_nonwindows.go` | 6 | 0 |
| `internal/syslocale/locale_windows.go` | 53 | 0 |

None - this slice holds its units whole.

Previous-register ids this slice re-checks: P1, P2, P4, P5, P7, P8, P9, P10, P11, P17, P22, E2, X1, X2, X3, X22, T1, T5, T8, B21.
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

### Step 08.1 - Prescan

- [x] Run the mechanical pass over this slice's own files and paste the verdict lines into the
      handoff notes:
  - Go packages (./internal/pipeline ./internal/procrun ./internal/report ./internal/rtf ./internal/syslocale): `go vet <pkgs>`, `$env:GOOS='linux'; go vet <pkgs>` (the
    non-Windows twins), `golangci-lint run <pkgs>` (it carries the staticcheck analyzers),
    `go test <pkgs>`.
  - Extension modules (extension/src/rtf.js): every `extension/test/*.test.mjs` that imports one of them -
    `node --test <files>` from `extension/`.
  - Other files ((none)): PowerShell - `Invoke-ScriptAnalyzer` when installed, else a parse
    check (`[System.Management.Automation.Language.Parser]::ParseFile`); a parity pair -
    `./scripts/parity-check.ps1`.
- **Verification:** each command above has its exit code and last line recorded in the handoff
  notes; a red result is either explained as pre-existing (reproduced on the read commit) or becomes
  a finding.

### Step 08.2 - Re-check the previous register

- [x] For each id listed above, find the fixed code and decide: `still fixed`, `regressed`,
      `one edition only` (fixed in Go or JS but not its twin), or `superseded` (the code is gone or
      replaced and the defect cannot occur). Write one line per id under "Re-check" in
      `FINDINGS.md`: `- <id> - <verdict> - S08 - <file:line or test that shows it>`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` no longer lists these ids as missing.
  A `regressed` crit/high becomes a package-1 ticket at priority 90 (Step 08.4).

### Step 08.3 - Read

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

### Step 08.4 - Triage

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

### Step 08.5 - Close

- [x] Inline fixes land with their test run cited (command, exit code, last line); a changed Go file
      is `gofmt`-clean.
- [x] Flip this phase's row in `INDEX.md` to ✅ with the severity counts, and bump `Phases: X/N` in
      `INDEX.md` and the `k/N slices` in the ticket's status line and its `RELEASE_QUEUE.md` line.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` shows `S08` as closed.

## Done criteria

- Every step `[x]`, each with its verification met.
- The findings table below is filled, and every crit/high in it has a ticket or an inline fix.

## Handoff notes

Steps 08.1-08.3 done here; 08.4-08.5 (FINDINGS.md, INDEX.md, queue, tickets) are the
orchestrator's - this auditor ran as one of several parallel slices and does not edit those files.

**Prescan** (windows/386 toolchain):
- `go vet` on the 5 packages - exit 0.
- `GOOS=linux go vet` on the 5 packages - exit 0.
- `golangci-lint run --config configs/.golangci.yml` on the 5 packages - exit 0, no issues.
- `go test` on the 5 packages - first run exit 1: `TestRunMissingBinaryKeepsThePathError` failed
  (`exec: ... executable file not found in %PATH%, want an *os.PathError`). Pre-existing since ecf42e5
  (already noted as red in tickets 31 and 32); a test defect on Windows, fixed inline (S08-1). After the
  fix: exit 0 (`ok` pipeline, procrun, report, rtf; syslocale has no tests); procrun also exit 0 with
  GOARCH=amd64.
- `node --test test/rtf.test.mjs test/legacy-text.test.mjs` (extension/) - exit 0, 35 pass, 0 fail.
- `scripts/parity-check.ps1` - exit 0, PASS (2 acknowledged one-sided changes outside this slice).

**Files read in full:** all 21 - cost.go, guard.go, ocrstep.go, outputdir.go, pipeline.go, reuse.go,
translate.go, budget.go, procrun.go, tree_nonwindows.go, tree_windows.go, archive.go, environment.go,
redact.go, store.go, codepage.go, extract.go, parse.go, extension/src/rtf.js, locale_nonwindows.go,
locale_windows.go.

**Re-check:** P1 P2 P4 P5 P7 P8 P9 P10 P11 P17 P22 E2 X1 X2 X3 X22 T1 T5 T8 B21 - all still fixed
(evidence lines in the orchestrator report).

**Findings (0 crit / 0 high / 0 med / 5 low):**
- S08-1 - low - conf - procrun test asserted *os.PathError for a missing binary; on Windows the start
  error is *exec.Error, so the package test was red on the product platform - inline (test only).
- S08-2 - low - conf (executed) - RTF `\binN` with a 10-digit N overflows `int` on the 386 build
  (shipped in the x86 installer): `r.pos` goes negative and the reader panics (index out of range);
  the panic guard turns it into "internal error", exit 3 - `internal/rtf/parse.go:29,155-158,134-135` - ticket
  (Go/JS twin; JS is unaffected).
- S08-3 - low - conf - `-google` with no API key converts, prints "Done." and exits 0, while an
  unreachable Ollama exits ExitAPI 4 - `internal/pipeline/translate.go:99-107`, `pipeline.go:354-358` - ticket
  (exit-code contract).
- S08-4 - low - plaus - Windows helper is assigned to its job object only after `Start`; a child
  spawned in that window escapes the tree kill - `internal/procrun/tree_windows.go:24-49`, `procrun.go:125-131` - register.
- S08-5 - low - conf/plaus - an extraction that fails after Ctrl+C is reported as a parse error (exit 3)
  rather than 130: `ctx` is checked only after a successful extraction - `internal/pipeline/pipeline.go:150-249` - register.
