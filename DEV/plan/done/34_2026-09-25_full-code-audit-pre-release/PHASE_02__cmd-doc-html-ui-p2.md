# Phase 02: S02 - cmd/doc-html-ui (part 2 of 2)

**Slice:** `S02` in [`slices.json`](slices.json) · **Lines:** 1838 in 3 files ·
**Added since `41fbc1b`:** 743 · **Risk:** 22.9
**Status:** ✅ Done - findings in FINDINGS.md, tickets 36-47
**Read at commit:** 26605b9 + working tree of 2026-09-26



> Self-contained: this phase carries the whole slice procedure. The findings go to
> [`FINDINGS.md`](FINDINGS.md), the phase state to [`INDEX.md`](INDEX.md).

## Files

| File | Lines | Added since `41fbc1b` |
|------|------:|------:|
| `cmd/doc-html-ui/store.go` | 123 | 123 |
| `cmd/doc-html-ui/ui.html` | 1679 | 618 |
| `cmd/doc-html-ui/versioninfo.json` | 36 | 2 |

Sibling slices of the same unit, adjacent in the order: S01. Read their files for context only; audit them in their own phase.

Previous-register ids this slice re-checks: G5, G10, G15, G17, G18.
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

### Step 02.1 - Prescan

- [x] Run the mechanical pass over this slice's own files and paste the verdict lines into the
      handoff notes:
  - Go packages (./cmd/doc-html-ui): `go vet <pkgs>`, `$env:GOOS='linux'; go vet <pkgs>` (the
    non-Windows twins), `golangci-lint run <pkgs>` (it carries the staticcheck analyzers),
    `go test <pkgs>`.
  - Extension modules ((none)): every `extension/test/*.test.mjs` that imports one of them -
    `node --test <files>` from `extension/`.
  - Other files (cmd/doc-html-ui/ui.html cmd/doc-html-ui/versioninfo.json): PowerShell - `Invoke-ScriptAnalyzer` when installed, else a parse
    check (`[System.Management.Automation.Language.Parser]::ParseFile`); a parity pair -
    `./scripts/parity-check.ps1`.
- **Verification:** each command above has its exit code and last line recorded in the handoff
  notes; a red result is either explained as pre-existing (reproduced on the read commit) or becomes
  a finding.

### Step 02.2 - Re-check the previous register

- [x] For each id listed above, find the fixed code and decide: `still fixed`, `regressed`,
      `one edition only` (fixed in Go or JS but not its twin), or `superseded` (the code is gone or
      replaced and the defect cannot occur). Write one line per id under "Re-check" in
      `FINDINGS.md`: `- <id> - <verdict> - S02 - <file:line or test that shows it>`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` no longer lists these ids as missing.
  A `regressed` crit/high becomes a package-1 ticket at priority 90 (Step 02.4).

### Step 02.3 - Read

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

### Step 02.4 - Triage

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

### Step 02.5 - Close

- [x] Inline fixes land with their test run cited (command, exit code, last line); a changed Go file
      is `gofmt`-clean.
- [x] Flip this phase's row in `INDEX.md` to ✅ with the severity counts, and bump `Phases: X/N` in
      `INDEX.md` and the `k/N slices` in the ticket's status line and its `RELEASE_QUEUE.md` line.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` shows `S02` as closed.

## Done criteria

- Every step `[x]`, each with its verification met.
- The findings table below is filled, and every crit/high in it has a ticket or an inline fix.

## Handoff notes

Run as one of several parallel auditors: FINDINGS.md, INDEX.md and RELEASE_QUEUE.md are left to the
orchestrator, which takes the lines below from this phase's report.

**Prescan (2026-09-26)**

- `go vet ./cmd/doc-html-ui/` - exit 0, no output.
- `GOOS=linux go vet ./cmd/doc-html-ui/` - exit 0, no output.
- `go test ./cmd/doc-html-ui/` - exit 0, `ok doc-html-translate/cmd/doc-html-ui 5.239s` (rerun after the inline fix).
- `golangci-lint run --config configs/.golangci.yml ./cmd/doc-html-ui/` - exit 1: two `errcheck` on
  `defer windows.CloseHandle(h)` at `proc_windows.go:37` and `proc_windows_test.go:12`. Both files belong
  to S01 (not this slice); handed to S01. Nothing in `store.go`.
- `ui.html`: inline `<script>` extracted and `node --check` - exit 0; `i18n.js` `node --check` - exit 0;
  every i18n key `ui.html` uses exists in `en`, and every language has the same `{placeholders}` as `en`
  (script `temp/audit_s02/keys.cjs`).
- `versioninfo.json`: `ConvertFrom-Json` - ok. The comma-separated `IconPath` is supported by goversioninfo
  (v1.7.0 `addIcon` splits on `,`). Invoke-ScriptAnalyzer not installed; no PowerShell file in the slice.
- No extension module and no parity pair in the slice.

**Re-check**

- G5 - still fixed - S02 - `main.go:875-886` (Ollama fields only under `req.Ollama`, via `intField`); the page still sends them raw (`ui.html:1282-1284`), which is now harmless.
- G10 - still fixed - S02 - `ui.html:743-752` (`/api/alive` held open, reopened on drop) + `liveness.go:29,41-74` (90 s grace, stream counts as alive).
- G15 - still fixed - S02 - `ui.html:797-806,816,825` (`dropFirstOnly` status names the file used).
- G17 - still fixed - S02 - `ui.html:1105` (radio matched by value, no selector splice).
- G18 - still fixed - S02 - `ui.html:1593-1597` (`resp.ok` checked, 409 worded), `ui.html:1541,1552` (follows only at bottom).

**Findings** (all low; none needs package 1)

| id | sev | conf | finding | where | disposition |
|----|-----|------|---------|-------|-------------|
| S02-1 | low | conf | Convert has no re-entry guard before its awaits: a second activation during the google-key / output-status fetches posts a second run, gets 409, and its `finally` re-enables Convert, hides Cancel and nulls `runAbort`/`runTarget` while the first run still streams - that run can no longer be cancelled from the window | `ui.html:1505-1536,1587,1593-1615` | ticket, package 4 |
| S02-2 | low | conf | a failed settings save is silent: the page ignores the response and swallows errors; the server's 500 (lock busy, or the rename refused because another handle has the file open - Go opens without FILE_SHARE_DELETE) is only logged | `ui.html:1127-1133`, `store.go:46`, `main.go:302-305` | ticket, package 4 (needs a UI string) |
| S02-3 | low | conf | `readSettings` drops the lock error and a failed set-aside: the corrupt file stays, the page gets `{}` with no header, and the next save overwrites it - the silent reset G12 removed, on a narrower trigger | `store.go:114-118` | register line |
| S02-4 | low | plaus | the saved OCR language is only stored in `dataset.want`; when `/api/ocr-langs` answers before `/api/settings`, nothing applies it, so a source language without installed OCR data shows the first installed pack and the next save persists that | `ui.html:1117,1157,1169,1140-1143` | register line |
| S02-5 | low | conf | a run's cost question queued behind another dialog is not withdrawn when the run ends (`dlgOpen` covers only the dialog on screen); it appears after the run and its answer goes nowhere | `ui.html:659-687,1612,1636` | register line |
| S02-6 | low | conf | `decodeURIComponent` in `fileUriToPath` throws on a malformed `%` escape in dropped `text/plain`, rejecting `handleDrop` before the `dt.files` upload fallback, with no status | `ui.html:774,788,822-828` | register line |
| S02-7 | low | conf | stale header comment in `store.go` described an output history and a read-modify-write cycle that no longer exist | `store.go:14-19` | inline: comment rewritten; `gofmt -l` empty, `go vet` exit 0, `go test ./cmd/doc-html-ui/` exit 0 |

**Files read in full:** `cmd/doc-html-ui/store.go` (123 lines + 1 from the fix), `cmd/doc-html-ui/ui.html`
(1679), `cmd/doc-html-ui/versioninfo.json` (36). Context from S01: `main.go` (settings, google-key, preview,
env, ocr handlers, `assembleArgs`), `run.go`, `liveness.go`.
