# Phase 19: S19 - internal/config .. internal/dialog

**Slice:** `S19` in [`slices.json`](slices.json) · **Lines:** 1327 in 10 files ·
**Added since `41fbc1b`:** 233 · **Risk:** 7.1
**Status:** ✅ Done - findings in FINDINGS.md, tickets 36-47
**Read at commit:** 26605b9 + working tree of 2026-09-26



> Self-contained: this phase carries the whole slice procedure. The findings go to
> [`FINDINGS.md`](FINDINGS.md), the phase state to [`INDEX.md`](INDEX.md).

## Files

| File | Lines | Added since `41fbc1b` |
|------|------:|------:|
| `internal/config/flags.go` | 271 | 51 |
| `internal/config/lang.go` | 25 | 0 |
| `extension/src/background.js` | 271 | 48 |
| `extension/src/options.html` | 109 | 5 |
| `extension/src/options.js` | 247 | 8 |
| `extension/src/popup.html` | 73 | 4 |
| `extension/src/popup.js` | 170 | 2 |
| `internal/dialog/dialog_nonwindows.go` | 32 | 11 |
| `internal/dialog/dialog_windows.go` | 78 | 53 |
| `internal/dialog/host.go` | 51 | 51 |

None - this slice holds its units whole.

Previous-register ids this slice re-checks: P10, P18, P19, P20, B24, B25.
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

### Step 19.1 - Prescan

- [x] Run the mechanical pass over this slice's own files and paste the verdict lines into the
      handoff notes:
  - Go packages (./internal/config ./internal/dialog): `go vet <pkgs>`, `$env:GOOS='linux'; go vet <pkgs>` (the
    non-Windows twins), `golangci-lint run <pkgs>` (it carries the staticcheck analyzers),
    `go test <pkgs>`.
  - Extension modules (extension/src/background.js extension/src/options.js extension/src/popup.js): every `extension/test/*.test.mjs` that imports one of them -
    `node --test <files>` from `extension/`.
  - Other files (extension/src/options.html extension/src/popup.html): PowerShell - `Invoke-ScriptAnalyzer` when installed, else a parse
    check (`[System.Management.Automation.Language.Parser]::ParseFile`); a parity pair -
    `./scripts/parity-check.ps1`.
- **Verification:** each command above has its exit code and last line recorded in the handoff
  notes; a red result is either explained as pre-existing (reproduced on the read commit) or becomes
  a finding.

### Step 19.2 - Re-check the previous register

- [x] For each id listed above, find the fixed code and decide: `still fixed`, `regressed`,
      `one edition only` (fixed in Go or JS but not its twin), or `superseded` (the code is gone or
      replaced and the defect cannot occur). Write one line per id under "Re-check" in
      `FINDINGS.md`: `- <id> - <verdict> - S19 - <file:line or test that shows it>`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` no longer lists these ids as missing.
  A `regressed` crit/high becomes a package-1 ticket at priority 90 (Step 19.4).

### Step 19.3 - Read

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

### Step 19.4 - Triage

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

### Step 19.5 - Close

- [x] Inline fixes land with their test run cited (command, exit code, last line); a changed Go file
      is `gofmt`-clean.
- [x] Flip this phase's row in `INDEX.md` to ✅ with the severity counts, and bump `Phases: X/N` in
      `INDEX.md` and the `k/N slices` in the ticket's status line and its `RELEASE_QUEUE.md` line.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` shows `S19` as closed.

## Done criteria

- Every step `[x]`, each with its verification met.
- The findings table below is filled, and every crit/high in it has a ticket or an inline fix.

## Handoff notes

Prescan (26605b9 + working tree of 2026-09-26):
- `go vet ./internal/config ./internal/dialog` - exit 0 - no output
- `GOOS=linux go vet ./internal/config ./internal/dialog` - exit 0 - no output
- `golangci-lint run --config configs/.golangci.yml ./internal/config/ ./internal/dialog/` - exit 0 - no issues
- `go test ./internal/config ./internal/dialog` - exit 0 - `ok doc-html-translate/internal/dialog`
- `node --test test/background.test.mjs` (the only test importing background/options/popup) - exit 0 - `fail 0` (12 pass)
- `node --check` background.js / options.js / popup.js - exit 0 each
- options.html / popup.html: tag balance 36/36 and 27/27; `./scripts/parity-check.ps1` - exit 0 -
  `parity-check: PASS (85 file(s) inspected, 2 acknowledged)`

Files read in full: [x] flags.go [x] lang.go [x] background.js [x] options.html [x] options.js
[x] popup.html [x] popup.js [x] dialog_nonwindows.go [x] dialog_windows.go [x] host.go

Re-check: P10 still fixed (a set limit pre-approves, pipeline/cost.go:27-34; under the GUI the question
is asked in-window, host.go:42-46); P18 still fixed (flags.go:24,118; cmd/doc-html-translate/main.go:26
`errors.Is`); P19 still fixed (flags.go:139-160, notices printed at app.go:144); P20 still fixed
(flags.go:123-125); B24 still fixed (background.js:22-27); B25 still fixed (background.js:51-60, 94-109).

Findings: 0 crit, 0 high, 3 med, 5 low; no inline fix (each needs a real browser, the Windows shell or
new UI strings, so each is a ticket or a register line):
- S19-1 med - popup "On this site" host label is wiped by applyI18n (popup.html:59, popup.js:11,110,131)
- S19-2 med - per-site switch keys the active tab's host, DNR excludes the request's domain (popup.js:100-106,147-155; background.js:77-78)
- S19-3 med - ShowWarning is a blocking MessageBox on console runs, batch/-noopen included (dialog_windows.go:71-78)
- S19-4..S19-8 low - see the coordinator report
