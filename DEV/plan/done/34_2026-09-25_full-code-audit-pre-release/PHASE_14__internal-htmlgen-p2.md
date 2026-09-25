# Phase 14: S14 - internal/htmlgen (part 2 of 2)

**Slice:** `S14` in [`slices.json`](slices.json) · **Lines:** 2179 in 4 files ·
**Added since `41fbc1b`:** 367 · **Risk:** 3.8
**Status:** ✅ Done - findings in FINDINGS.md, tickets 36-47
**Read at commit:** 26605b9 + working tree of 2026-09-26



> Self-contained: this phase carries the whole slice procedure. The findings go to
> [`FINDINGS.md`](FINDINGS.md), the phase state to [`INDEX.md`](INDEX.md).

## Files

| File | Lines | Added since `41fbc1b` |
|------|------:|------:|
| `extension/src/glyphs.js` | 57 | 57 |
| `extension/src/viewer.css` | 297 | 64 |
| `extension/src/viewer.html` | 68 | 12 |
| `extension/src/viewer.js` | 1757 | 234 |

Sibling slices of the same unit, adjacent in the order: S13. Read their files for context only; audit them in their own phase.

Previous-register ids this slice re-checks: B1, B8, B10, B11, B14, B26, B27, B28.
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

### Step 14.1 - Prescan

- [x] Run the mechanical pass over this slice's own files and paste the verdict lines into the
      handoff notes:
  - Go packages ((none)): `go vet <pkgs>`, `$env:GOOS='linux'; go vet <pkgs>` (the
    non-Windows twins), `golangci-lint run <pkgs>` (it carries the staticcheck analyzers),
    `go test <pkgs>`.
  - Extension modules (extension/src/glyphs.js extension/src/viewer.js): every `extension/test/*.test.mjs` that imports one of them -
    `node --test <files>` from `extension/`.
  - Other files (extension/src/viewer.css extension/src/viewer.html): PowerShell - `Invoke-ScriptAnalyzer` when installed, else a parse
    check (`[System.Management.Automation.Language.Parser]::ParseFile`); a parity pair -
    `./scripts/parity-check.ps1`.
- **Verification:** each command above has its exit code and last line recorded in the handoff
  notes; a red result is either explained as pre-existing (reproduced on the read commit) or becomes
  a finding.

### Step 14.2 - Re-check the previous register

- [x] For each id listed above, find the fixed code and decide: `still fixed`, `regressed`,
      `one edition only` (fixed in Go or JS but not its twin), or `superseded` (the code is gone or
      replaced and the defect cannot occur). Write one line per id under "Re-check" in
      `FINDINGS.md`: `- <id> - <verdict> - S14 - <file:line or test that shows it>`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` no longer lists these ids as missing.
  A `regressed` crit/high becomes a package-1 ticket at priority 90 (Step 14.4).

### Step 14.3 - Read

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

### Step 14.4 - Triage

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

### Step 14.5 - Close

- [x] Inline fixes land with their test run cited (command, exit code, last line); a changed Go file
      is `gofmt`-clean.
- [x] Flip this phase's row in `INDEX.md` to ✅ with the severity counts, and bump `Phases: X/N` in
      `INDEX.md` and the `k/N slices` in the ticket's status line and its `RELEASE_QUEUE.md` line.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` shows `S14` as closed.

## Done criteria

- Every step `[x]`, each with its verification met.
- The findings table below is filled, and every crit/high in it has a ticket or an inline fix.

## Handoff notes

**Prescan (2026-09-26, working tree on 26605b9).** Go packages: none in this slice; `internal/htmlgen`
was run for context only.

- `node --test test/viewer.test.mjs test/lifecycle.test.mjs test/content-security.test.mjs test/background.test.mjs test/i18n.test.mjs test/page-ocr.test.mjs` (from `extension/`) - exit 0 - `ℹ duration_ms 2602.9534` (37 pass, 0 fail)
- `node --check src/viewer.js` - exit 0 - (no output); `node --check src/glyphs.js` - exit 0 - (no output)
- `go vet ./internal/htmlgen/` - exit 0; `GOOS=linux go vet ./internal/htmlgen/` - exit 0; `go test ./internal/htmlgen/` - exit 0 - `ok doc-html-translate/internal/htmlgen 5.143s`; `golangci-lint run --config configs/.golangci.yml ./internal/htmlgen/` - exit 0 - (no output)
- `./scripts/parity-check.ps1` - exit 0 - `parity-check: PASS (80 file(s) inspected, 1 acknowledged)`
- viewer.css / viewer.html: no PowerShell analyzer applies; covered by the parity check above.

**Re-check.** B1 still fixed (`teardownCurrent` destroys `pdfTask` and `pdfDoc`, viewer.js:141-142). B8 still
fixed (async `toBlob` one image at a time, canvas emptied, `vSavedLarge` warning, viewer.js:639-661,599-605;
blob URLs held until teardown by the ticket-18 decision). B10 still fixed at its cited sites (load tokens,
viewer.js:164-172,895-910,981,1566) - but see S14-1 for a residual in `renderDocument`. B11 still fixed
(viewer.js:300-328). B14 still fixed (export CSP and escaped attributes, export-html.js `EXPORT_CSP` /
`buildExportHtml`). B26 still fixed on the viewer side (`credentials: "include"`, viewer.js:894);
ocr-overlay.js:117 image fetch is out of this slice. B27 still fixed (diagnostics.js `recordRun` serialized,
format starts a clean run). B28 still fixed (`currentUrl`, viewer.js:502-515,1174,1217).

**Files read in full:** extension/src/glyphs.js (57), viewer.css (297), viewer.html (68), viewer.js (1757).

**Findings (for FINDINGS.md, ids assigned by the orchestrator):**

| # | Sev | Conf | Finding | Site |
|---|---|---|---|---|
| S14-1 | med | conf | `renderDocument` has no load-token check between `collectSample`/`setDocumentLang` and `applyLang`, nor after `await renderChunk()`: a URL PDF superseded by a picked file during language detection writes its `<html lang>` over the new document; a stranded first chunk runs `warnIfNoText` and unhides Export on the new document | viewer.js:1319-1320,1692,1337-1342 |
| S14-2 | med | conf | HTML export re-encodes every blob image as JPEG; a transparent PNG (EPUB diagrams, formulas) is composited on black by spec, so dark line-art becomes a black box | viewer.js:651 |
| S14-3 | med | conf | Chrome state leaks across documents: `renderToc` hides the TOC button and nothing unhides it, so a later document with a TOC has no TOC button; `grp-ocr` is never re-hidden | viewer.js:718-720,218 |
| S14-4 | low | conf | docs/PARITY.md drift: `PAGE_CHUNK = 50`, `CHUNK_LEAD = 2` against 100 / 5 in code; the FAMILIES cite `viewer.js:91-95` is stale (now 186-190) | docs/PARITY.md:1084-1085,286; viewer.js:1295-1296 |
| S14-5 | low | conf | Text-size range/step drift, not recorded in PARITY: viewer 12-40 px step 1 px; desktop 70-300 % (11.2-48 px) step 10 % | viewer.js:1715,1719; navbar.go:428 |
| S14-6 | low | conf | A password prompt records "Password required" as the run's error; a successful unlock never clears it, so diagnostics report a failure for a document that opened | viewer.js:1274,468 |
| S14-7 | low | conf | `<nav id="toc" aria-label="Table of contents">` is English in every interface language (`applyI18n(#toc)` only walks descendants) | viewer.html:59 |

Severity counts: crit 0, high 0, med 3, low 4. Inline fixes: none (each changes behaviour).
