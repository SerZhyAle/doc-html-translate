# Phase 15: S15 - internal/epub

**Slice:** `S15` in [`slices.json`](slices.json) · **Lines:** 2277 in 7 files ·
**Added since `41fbc1b`:** 1158 · **Risk:** 13.9
**Status:** ✅ Done - findings in FINDINGS.md, tickets 36-47
**Read at commit:** 26605b9 + working tree of 2026-09-26



> Self-contained: this phase carries the whole slice procedure. The findings go to
> [`FINDINGS.md`](FINDINGS.md), the phase state to [`INDEX.md`](INDEX.md).

## Files

| File | Lines | Added since `41fbc1b` |
|------|------:|------:|
| `internal/epub/charset.go` | 101 | 101 |
| `internal/epub/epub.go` | 412 | 117 |
| `internal/epub/links.go` | 264 | 264 |
| `internal/epub/normalize.go` | 393 | 393 |
| `internal/epub/resolve.go` | 130 | 130 |
| `internal/epub/toc.go` | 409 | 19 |
| `extension/src/epub.js` | 568 | 134 |

None - this slice holds its units whole.

Previous-register ids this slice re-checks: E1, E5, E6, E10, E11, E12, E16, E18, E20, B16, B17, B19, B20, B23.
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

### Step 15.1 - Prescan

- [x] Run the mechanical pass over this slice's own files and paste the verdict lines into the
      handoff notes:
  - Go packages (./internal/epub): `go vet <pkgs>`, `$env:GOOS='linux'; go vet <pkgs>` (the
    non-Windows twins), `golangci-lint run <pkgs>` (it carries the staticcheck analyzers),
    `go test <pkgs>`.
  - Extension modules (extension/src/epub.js): every `extension/test/*.test.mjs` that imports one of them -
    `node --test <files>` from `extension/`.
  - Other files ((none)): PowerShell - `Invoke-ScriptAnalyzer` when installed, else a parse
    check (`[System.Management.Automation.Language.Parser]::ParseFile`); a parity pair -
    `./scripts/parity-check.ps1`.
- **Verification:** each command above has its exit code and last line recorded in the handoff
  notes; a red result is either explained as pre-existing (reproduced on the read commit) or becomes
  a finding.

### Step 15.2 - Re-check the previous register

- [x] For each id listed above, find the fixed code and decide: `still fixed`, `regressed`,
      `one edition only` (fixed in Go or JS but not its twin), or `superseded` (the code is gone or
      replaced and the defect cannot occur). Write one line per id under "Re-check" in
      `FINDINGS.md`: `- <id> - <verdict> - S15 - <file:line or test that shows it>`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` no longer lists these ids as missing.
  A `regressed` crit/high becomes a package-1 ticket at priority 90 (Step 15.4).

### Step 15.3 - Read

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

### Step 15.4 - Triage

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

### Step 15.5 - Close

- [x] Inline fixes land with their test run cited (command, exit code, last line); a changed Go file
      is `gofmt`-clean.
- [x] Flip this phase's row in `INDEX.md` to ✅ with the severity counts, and bump `Phases: X/N` in
      `INDEX.md` and the `k/N slices` in the ticket's status line and its `RELEASE_QUEUE.md` line.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` shows `S15` as closed.

## Done criteria

- Every step `[x]`, each with its verification met.
- The findings table below is filled, and every crit/high in it has a ticket or an inline fix.

## Handoff notes

Steps 15.1-15.3 done; 15.4/15.5 bookkeeping (FINDINGS.md, INDEX.md, RELEASE_QUEUE.md, tickets) is the
coordinator's - this auditor may not edit those files. No inline fixes.

Prescan (2026-09-26, PowerShell, repo root):
- `go vet ./internal/epub/` - exit 0 - (no output)
- `$env:GOOS='linux'; go vet ./internal/epub/` - exit 0 - (no output)
- `golangci-lint run --config configs/.golangci.yml ./internal/epub/` - exit 0 - (no issues)
- `go test ./internal/epub/` - exit 0 - `ok doc-html-translate/internal/epub 1.482s`
- `gofmt -l internal/epub` - exit 0 - (clean)
- extension: `node --test test/epub-dom.test.mjs test/epub.test.mjs test/limits.test.mjs` - exit 0 - `pass 34, fail 0`;
  `node --check src/epub.js` - exit 0

Files read in full: charset.go, epub.go, links.go, normalize.go, resolve.go, toc.go, extension/src/epub.js
(plus docs/PARITY.md "EPUB TOC parsing", "EPUB href resolution", "EPUB and HTML content fidelity", "Input limits").

Re-check: E1, E5, E10, E11, E16, E18, E20, B16, B17, B19, B20, B23 still fixed; E6 and E12 one edition only
(fixed in Go, still open in extension/src/epub.js - PARITY.md records an open gap that no open ticket carries).

Findings (0 crit, 0 high, 2 med, 7 low):
- S15-1 - med - conf - the XHTML -> HTML rename has no collision check: `a.xhtml` next to `a.html` (or `x.xhtm` +
  `x.xhtml`) writes over the other chapter, and both manifest items end on one href (one chapter lost, one shown
  twice) - normalize.go:92-101 - ticket (output naming). Proof: throwaway package test, manifest after
  normalizeContent `[{h a.html} {x a.html}]`, a.html holds only the XHTML chapter's text; test file removed.
- S15-2 - med - conf - extension EPUB content-fidelity gaps have no carrier: XHTML parsed as text/html (E12:
  `<script src/>`, `<title/>`, `<a id/>` swallow chapter text), UTF-8-only decode, SVG cover with text replaced
  (E6) - epub.js:456, 183-185, 422-424 - ticket (cross-edition follow-up of done/06).
- S15-3 - low - conf - a chapter whose normalization fails keeps its .xhtml name, but other chapters' links were
  already redirected to the .html that is never written - normalize.go:99-101 vs 57-59.
- S15-4 - low - conf - Go TOC hrefs bypass resolveBookPath: root-relative `/x`, backslash and `?query` targets are
  dropped in Go and resolved in JS; not listed in PARITY - toc.go:311-319 vs epub.js:347-349.
- S15-5 - low - conf - Go link rewrite skips root-relative `/..` links, so an .xhtml rename is not followed and the
  link points at the drive root under file://; JS resolves them to the book root - links.go:125 vs epub.js:446.
- S15-6 - low - plaus - extraction containment is a prefix check only; entry names with a DOS device segment or an
  NTFS stream colon are not refused (resolveBookPath refuses them only for OPF names) - epub.go:270-276.
- S15-7 - low - conf - PARITY "Input limits" says symlink entries are skipped from the listing; JS keeps them as
  files, Go counts them in the total and fails them per entry - epub.js:65-83, epub.go:255-262,281.
- S15-8 - low - plaus - JS convertSvgImage in a multi-image SVG inserts an HTML `<img>` inside `<svg>`, which is not
  rendered - epub.js:424.
- S15-9 - low - conf - JS parseContainer falls back to the first rootfile of any type, Go fails the book -
  epub.js:208 vs epub.go:324-334.
