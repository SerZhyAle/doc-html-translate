# Phase 16: S16 - extension/src/defaults* .. extension/src/url*

**Slice:** `S16` in [`slices.json`](slices.json) · **Lines:** 2082 in 15 files ·
**Added since `41fbc1b`:** 491 · **Risk:** 12.5
**Status:** ✅ Done - findings in FINDINGS.md, tickets 36-47
**Read at commit:** 26605b9 + working tree of 2026-09-26



> Self-contained: this phase carries the whole slice procedure. The findings go to
> [`FINDINGS.md`](FINDINGS.md), the phase state to [`INDEX.md`](INDEX.md).

## Files

| File | Lines | Added since `41fbc1b` |
|------|------:|------:|
| `extension/src/defaults.js` | 22 | 5 |
| `extension/src/diagnostics.js` | 86 | 15 |
| `extension/src/export-html.js` | 48 | 48 |
| `extension/src/i18n.js` | 118 | 5 |
| `extension/src/lang.js` | 97 | 0 |
| `extension/src/ocr-host.html` | 15 | 1 |
| `extension/src/ocr-host.js` | 89 | 27 |
| `extension/src/ocr-screen.js` | 229 | 8 |
| `extension/src/ocr.html` | 29 | 0 |
| `extension/src/ocr.js` | 175 | 8 |
| `extension/src/page-agent.js` | 509 | 126 |
| `extension/src/page-ocr.js` | 376 | 96 |
| `extension/src/page-overlay.css` | 108 | 0 |
| `extension/src/sanitize.js` | 35 | 6 |
| `extension/src/url-policy.js` | 146 | 146 |

None - this slice holds its units whole.

Previous-register ids this slice re-checks: B3, B5, B6, B12, B13, B14, B15, B17, B18, B27.
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

### Step 16.1 - Prescan

- [x] Run the mechanical pass over this slice's own files and paste the verdict lines into the
      handoff notes:
  - Go packages ((none)): `go vet <pkgs>`, `$env:GOOS='linux'; go vet <pkgs>` (the
    non-Windows twins), `golangci-lint run <pkgs>` (it carries the staticcheck analyzers),
    `go test <pkgs>`.
  - Extension modules (extension/src/defaults.js extension/src/diagnostics.js extension/src/export-html.js extension/src/i18n.js extension/src/lang.js extension/src/ocr-host.js extension/src/ocr-screen.js extension/src/ocr.js extension/src/page-agent.js extension/src/page-ocr.js extension/src/sanitize.js extension/src/url-policy.js): every `extension/test/*.test.mjs` that imports one of them -
    `node --test <files>` from `extension/`.
  - Other files (extension/src/ocr-host.html extension/src/ocr.html extension/src/page-overlay.css): PowerShell - `Invoke-ScriptAnalyzer` when installed, else a parse
    check (`[System.Management.Automation.Language.Parser]::ParseFile`); a parity pair -
    `./scripts/parity-check.ps1`.
- **Verification:** each command above has its exit code and last line recorded in the handoff
  notes; a red result is either explained as pre-existing (reproduced on the read commit) or becomes
  a finding.

### Step 16.2 - Re-check the previous register

- [x] For each id listed above, find the fixed code and decide: `still fixed`, `regressed`,
      `one edition only` (fixed in Go or JS but not its twin), or `superseded` (the code is gone or
      replaced and the defect cannot occur). Write one line per id under "Re-check" in
      `FINDINGS.md`: `- <id> - <verdict> - S16 - <file:line or test that shows it>`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` no longer lists these ids as missing.
  A `regressed` crit/high becomes a package-1 ticket at priority 90 (Step 16.4).

### Step 16.3 - Read

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

### Step 16.4 - Triage

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

### Step 16.5 - Close

- [x] Inline fixes land with their test run cited (command, exit code, last line); a changed Go file
      is `gofmt`-clean.
- [x] Flip this phase's row in `INDEX.md` to ✅ with the severity counts, and bump `Phases: X/N` in
      `INDEX.md` and the `k/N slices` in the ticket's status line and its `RELEASE_QUEUE.md` line.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` shows `S16` as closed.

## Done criteria

- Every step `[x]`, each with its verification met.
- The findings table below is filled, and every crit/high in it has a ticket or an inline fix.

## Handoff notes

Prescan (no Go packages in this slice):
- `node --test test/content-security.test.mjs test/diagnostics.test.mjs test/ocr-screen.test.mjs test/page-ocr.test.mjs test/reflow.test.mjs test/sanitize.test.mjs` (from extension/) - exit 0 - `pass 55, fail 0`.
- `node --check src/<f>.js` for all 12 modules - exit 0 each.
- `./scripts/parity-check.ps1` - exit 0 - `parity-check: PASS (85 file(s) inspected, 2 acknowledged)`.

Re-check: B3 B5 B6 B12 B13 B14 B15 B17 B18 B27 - all still fixed.

Files read in full: defaults.js, diagnostics.js, export-html.js, i18n.js, lang.js, ocr-host.html, ocr-host.js,
ocr-screen.js (against internal/ocr/screen.go), ocr.html, ocr.js, page-agent.js, page-ocr.js, page-overlay.css,
sanitize.js, url-policy.js.

Findings (1 high, 2 med, 6 low; no inline fixes; repro scripts in temp/audit34/s16/):
- S16-1 - high - plaus - page OCR collects any `<img>` src (file:, intranet http, never-loaded images) with no origin check and returns the OCR text into the page's own DOM; the page can trigger rescans by synthetic clicks - page-agent.js:93-96,101-108,349; page-ocr.js:182 - ticket pkg 1
- S16-2 - med - conf - the agent's onMessage listener survives teardown; Remove + Start leaves a zombie agent that answers `collect` first, two bars - page-agent.js:452-474 - ticket
- S16-3 - med - conf - ids are namespaced but SVG `url(#id)`, usemap / map name, label for, aria-* references are not - inline SVG gradients/clips and image maps break - url-policy.js:89-90,94,127 - ticket
- S16-4 - low - plaus - remote-content parking misses SVG presentation `url()` (cursor/filter/mask/clip-path) and `<template>` content (live again in the export via shadowrootmode) - url-policy.js:18,26-28 - register
- S16-5 - low - conf - detectLang returns zh for kanji-dense Japanese - lang.js:24-26,79-82 - register
- S16-6 - low - conf - ocr.js / page-agent.js / page-ocr.js ignore the interface-language override - ticket (UI strings)
- S16-7 - low - conf - a host frame that never announced itself stays in the page - page-ocr.js:141,279-283 - register
- S16-8 - low - conf - parity-map OCR pair omits ocr-screen.js - configs/parity-map.json:14 - register
- S16-9 - low - plaus - urlKind treats `/\host` and `\/host` as relative; in the file: export they are network-path links - url-policy.js:39 - register
