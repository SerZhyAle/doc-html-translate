# Phase 12: S12 - internal/ocr (part 3 of 3)

**Slice:** `S12` in [`slices.json`](slices.json) · **Lines:** 1999 in 7 files ·
**Added since `41fbc1b`:** 483 · **Risk:** 0.6
**Status:** ✅ Done - findings in FINDINGS.md, tickets 36-47
**Read at commit:** 26605b9 + working tree of 2026-09-26



> Self-contained: this phase carries the whole slice procedure. The findings go to
> [`FINDINGS.md`](FINDINGS.md), the phase state to [`INDEX.md`](INDEX.md).

## Files

| File | Lines | Added since `41fbc1b` |
|------|------:|------:|
| `internal/ocr/text.go` | 87 | 0 |
| `extension/src/ocr-cluster.js` | 606 | 161 |
| `extension/src/ocr-lang.js` | 137 | 0 |
| `extension/src/ocr-overlay.css` | 101 | 45 |
| `extension/src/ocr-overlay.js` | 718 | 149 |
| `extension/src/ocr-plates.js` | 312 | 128 |
| `extension/src/ocr-text.js` | 38 | 0 |

Sibling slices of the same unit, adjacent in the order: S10, S11. Read their files for context only; audit them in their own phase.

Previous-register ids this slice re-checks: B2, B4, B7, B16, B26.
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

### Step 12.1 - Prescan

- [x] Run the mechanical pass over this slice's own files and paste the verdict lines into the
      handoff notes:
  - Go packages (./internal/ocr): `go vet <pkgs>`, `$env:GOOS='linux'; go vet <pkgs>` (the
    non-Windows twins), `golangci-lint run <pkgs>` (it carries the staticcheck analyzers),
    `go test <pkgs>`.
  - Extension modules (extension/src/ocr-cluster.js extension/src/ocr-lang.js extension/src/ocr-overlay.js extension/src/ocr-plates.js extension/src/ocr-text.js): every `extension/test/*.test.mjs` that imports one of them -
    `node --test <files>` from `extension/`.
  - Other files (extension/src/ocr-overlay.css): PowerShell - `Invoke-ScriptAnalyzer` when installed, else a parse
    check (`[System.Management.Automation.Language.Parser]::ParseFile`); a parity pair -
    `./scripts/parity-check.ps1`.
- **Verification:** each command above has its exit code and last line recorded in the handoff
  notes; a red result is either explained as pre-existing (reproduced on the read commit) or becomes
  a finding.

### Step 12.2 - Re-check the previous register

- [x] For each id listed above, find the fixed code and decide: `still fixed`, `regressed`,
      `one edition only` (fixed in Go or JS but not its twin), or `superseded` (the code is gone or
      replaced and the defect cannot occur). Write one line per id under "Re-check" in
      `FINDINGS.md`: `- <id> - <verdict> - S12 - <file:line or test that shows it>`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` no longer lists these ids as missing.
  A `regressed` crit/high becomes a package-1 ticket at priority 90 (Step 12.4).

### Step 12.3 - Read

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

### Step 12.4 - Triage

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

### Step 12.5 - Close

- [x] Inline fixes land with their test run cited (command, exit code, last line); a changed Go file
      is `gofmt`-clean.
- [x] Flip this phase's row in `INDEX.md` to ✅ with the severity counts, and bump `Phases: X/N` in
      `INDEX.md` and the `k/N slices` in the ticket's status line and its `RELEASE_QUEUE.md` line.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` shows `S12` as closed.

## Done criteria

- Every step `[x]`, each with its verification met.
- The findings table below is filled, and every crit/high in it has a ticket or an inline fix.

## Handoff notes

Run as one of several parallel auditors: `FINDINGS.md`, `INDEX.md`, tickets and `RELEASE_QUEUE.md` are
the orchestrator's to write; the lines below are what was handed over. Step 12.5's second box (INDEX flip)
is the orchestrator's.

**Prescan** (PowerShell, repo root unless noted):

- `go vet ./internal/ocr/` - exit 0, no output
- `$env:GOOS='linux'; go vet ./internal/ocr/` - exit 0, no output
- `golangci-lint run --config configs/.golangci.yml ./internal/ocr/` - exit 0, no output (0 issues)
- `go test -count=1 ./internal/ocr/` - exit 0 - `ok  doc-html-translate/internal/ocr 1.848s`
- from `extension/`: `node --test test/ocr-cluster.test.mjs test/ocr-plates.test.mjs test/ocr-text.test.mjs test/ocr-screen.test.mjs test/content-security.test.mjs test/lifecycle.test.mjs` - exit 0 - `pass 65, fail 0`
- `node --check` on ocr-cluster/ocr-lang/ocr-overlay/ocr-plates/ocr-text.js - exit 0 each
- `./scripts/parity-check.ps1` - exit 0 - `parity-check: PASS (78 file(s) inspected, 1 acknowledged)`
- `ocr-overlay.css`: no PowerShell/parse gate applies; its generated region was not hand-checked (owned by internal/appearance + tests/appearance_parity_test.go).

**Re-check:**

- B2 - still fixed - S12 - `extension/src/ocr-plates.js:96,103-115,182-188` (buildOverlay keeps the stop in `liveFits`, `releaseOverlays`), callers `viewer.js:137,314,1013`, page agent keeps `a.stopFit` (`page-agent.js:294`)
- B4 - still fixed - S12 - `extension/src/ocr-overlay.js:65-70` (a rejected start is forgotten)
- B7 - still fixed - S12 - `extension/src/ocr-overlay.js:139-141` (size-only bitmap closed at once)
- B16 - still fixed - S12 - parked remote images are skipped for OCR at `extension/src/viewer.js:342` (`REMOTE_MARK`, url-policy.js)
- B26 - still fixed (viewer site) - S12 - `extension/src/viewer.js:894` `credentials: "include"`; the register's second site `ocr-overlay.js` fetchToBlob was never changed - see S12-2

**Findings:**

- S12-1 - med - conf - Go/JS drift: the extension parks lines for column regrouping at the ordinary floor 50 on every pass, while the desktop parks at the pass's own floor (80 on the rescue ladder, screen rescue and screen sweep); a 50-80 line that the rescue clustering will drop still decides where a column is in the extension, which is the exact chaining `orderColumns` exists to prevent - `extension/src/ocr-overlay.js:377` (`orderColumns(out)`, callers at 539, 583, 622) vs `internal/ocr/tesseract.go:1347,390,437,496` - ticket: package 1, "OCR: pass the pass floor to orderColumns in the extension"
- S12-2 - low - plaus - context-menu OCR of a login-gated remote image: `fetchToBlob` fetches with the default `same-origin` credentials from the extension origin, so the cookie is not sent and the page shows "Image fetch failed: 401/403" (the B26 site the fix did not cover) - `extension/src/ocr-overlay.js:117` - ->
- S12-3 - low - conf - ImageBitmaps are not closed on the error path: `sampleColors` closes only after every block succeeded, `measureScreenPitch` closes after `getImageData` (the call that throws on a huge canvas), `greyCanvas`/`strokePlane` leak when the context or draw throws - GC reclaims them eventually - `extension/src/ocr-overlay.js:301-308,466-473,644-651,667-669` - ->
- S12-4 - low - conf - Go/JS drift in `isTranslatable`'s CJK class: halfwidth katakana, Hangul compatibility jamo and astral Han (CJK Ext B+) count as CJK on the desktop (`unicode.Katakana/Hangul/Han`) and as plain letters in the extension (BMP ranges, no `u` flag), so e.g. `ｶﾀｶﾅ`, `ㄱㄴ`, `𠀀𠀁` are kept by Go and rejected by JS (probed both sides) - `extension/src/ocr-text.js:8` vs `internal/ocr/text.go:17-20` - ->

**Dedupe:** no match for S12-1..4 in `DEV/plan/*.md`, `DEV/plan/done/*.md` or the previous register;
S12-2 is the unaddressed half of B26 (ticket 19 row for B26 covers `viewer.js` only). Pixel budget for
the extension's full-size decodes is a documented intentional divergence (PARITY.md "Input limits",
"Full image decode (desktop only)"), not a finding. The `getComputedStyle(b).minHeight` read in
`fitPlate` is identical on both editions and lab-measured, not a finding.

**Files read in full:** internal/ocr/text.go (87), extension/src/ocr-cluster.js (606),
ocr-lang.js (137), ocr-overlay.css (101), ocr-overlay.js (718), ocr-plates.js (312), ocr-text.js (38).
