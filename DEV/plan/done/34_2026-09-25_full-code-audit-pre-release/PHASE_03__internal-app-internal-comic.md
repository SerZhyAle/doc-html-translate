# Phase 03: S03 - internal/app .. internal/comic

**Slice:** `S03` in [`slices.json`](slices.json) · **Lines:** 2588 in 20 files ·
**Added since `41fbc1b`:** 1789 · **Risk:** 49
**Status:** ✅ Done - findings in FINDINGS.md, tickets 36-47
**Read at commit:** 26605b9 + working tree of 2026-09-26



> Self-contained: this phase carries the whole slice procedure. The findings go to
> [`FINDINGS.md`](FINDINGS.md), the phase state to [`INDEX.md`](INDEX.md).

## Files

| File | Lines | Added since `41fbc1b` |
|------|------:|------:|
| `internal/app/app.go` | 320 | 137 |
| `internal/app/splash.go` | 37 | 0 |
| `internal/appearance/appearance.go` | 238 | 238 |
| `internal/assets/copier.go` | 290 | 290 |
| `internal/assets/css.go` | 125 | 125 |
| `internal/assets/html.go` | 197 | 197 |
| `internal/browser/browser.go` | 2 | 0 |
| `internal/browser/browser_nonwindows.go` | 10 | 0 |
| `internal/browser/browser_windows.go` | 76 | 62 |
| `internal/bundledtools/cache.go` | 139 | 139 |
| `internal/bundledtools/doc.go` | 9 | 9 |
| `internal/bundledtools/pdftotext_nonwindows.go` | 9 | 9 |
| `internal/bundledtools/pdftotext_windows.go` | 49 | 49 |
| `internal/comic/container.go` | 66 | 66 |
| `internal/comic/extract.go` | 261 | 105 |
| `internal/comic/natural.go` | 73 | 0 |
| `internal/comic/readers_sevenzip.go` | 265 | 206 |
| `internal/comic/readers_tar.go` | 66 | 36 |
| `internal/comic/readers_zip.go` | 30 | 13 |
| `extension/src/comic.js` | 326 | 108 |

None - this slice holds its units whole.

Previous-register ids this slice re-checks: P3, P12, P14, P15, P24, X9, X15, X16, X17, X24, B22, B23, Q1, Q2.
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

### Step 03.1 - Prescan

- [x] Run the mechanical pass over this slice's own files and paste the verdict lines into the
      handoff notes:
  - Go packages (./internal/app ./internal/appearance ./internal/assets ./internal/browser ./internal/bundledtools ./internal/comic): `go vet <pkgs>`, `$env:GOOS='linux'; go vet <pkgs>` (the
    non-Windows twins), `golangci-lint run <pkgs>` (it carries the staticcheck analyzers),
    `go test <pkgs>`.
  - Extension modules (extension/src/comic.js): every `extension/test/*.test.mjs` that imports one of them -
    `node --test <files>` from `extension/`.
  - Other files ((none)): PowerShell - `Invoke-ScriptAnalyzer` when installed, else a parse
    check (`[System.Management.Automation.Language.Parser]::ParseFile`); a parity pair -
    `./scripts/parity-check.ps1`.
- **Verification:** each command above has its exit code and last line recorded in the handoff
  notes; a red result is either explained as pre-existing (reproduced on the read commit) or becomes
  a finding.

### Step 03.2 - Re-check the previous register

- [x] For each id listed above, find the fixed code and decide: `still fixed`, `regressed`,
      `one edition only` (fixed in Go or JS but not its twin), or `superseded` (the code is gone or
      replaced and the defect cannot occur). Write one line per id under "Re-check" in
      `FINDINGS.md`: `- <id> - <verdict> - S03 - <file:line or test that shows it>`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` no longer lists these ids as missing.
  A `regressed` crit/high becomes a package-1 ticket at priority 90 (Step 03.4).

### Step 03.3 - Read

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

### Step 03.4 - Triage

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

### Step 03.5 - Close

- [x] Inline fixes land with their test run cited (command, exit code, last line); a changed Go file
      is `gofmt`-clean.
- [x] Flip this phase's row in `INDEX.md` to ✅ with the severity counts, and bump `Phases: X/N` in
      `INDEX.md` and the `k/N slices` in the ticket's status line and its `RELEASE_QUEUE.md` line.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` shows `S03` as closed.

## Done criteria

- Every step `[x]`, each with its verification met.
- The findings table below is filled, and every crit/high in it has a ticket or an inline fix.

## Handoff notes

Run as one of several parallel auditors: `FINDINGS.md`, `INDEX.md` and `RELEASE_QUEUE.md` are written by
the orchestrator from the lines below, not by this phase.

**Prescan** (all green):

- `go vet ./internal/app ./internal/appearance ./internal/assets ./internal/browser ./internal/bundledtools ./internal/comic` - exit 0, no output.
- `$env:GOOS='linux'; go vet <same pkgs>` - exit 0, no output.
- `golangci-lint run --config configs/.golangci.yml <same pkgs>` - exit 0, no output; with `GOOS=linux` on `./internal/browser/ ./internal/bundledtools/` - exit 0, no output.
- `go test <same pkgs>` - exit 0, last line `ok doc-html-translate/internal/comic (cached)`.
- `node --test test/comic.test.mjs test/limits.test.mjs` (from `extension/`) - exit 0, `pass 18`, `fail 0`.

**Re-check:**

- P3 - still fixed - S03 - `internal/browser/browser_windows.go:25-36` (ShellExecute, no cmd.exe); `TestOpenPassesTargetVerbatim`
- P12 - still fixed - S03 - `internal/app/app.go:180-198` reports Default/Blocked/Unknown/Failed; `windowsreg/register_windows.go:63` SHChangeNotify; `TestDefaultHandlerResultIsHonest`
- P14 - still fixed - S03 - `internal/app/app.go:42-44,60-63,69-72,234-241` (every integration error printed)
- P15 - still fixed - S03 - `internal/bundledtools/cache.go:62-116` (content-hash folder, temp+rename); `TestExtractSetConcurrent`
- P24 - superseded - S03 - `internal/browser/browser_windows.go:32` (ShellExecute starts no child process to wait on)
- X9 - still fixed - S03 - `internal/comic/readers_sevenzip.go:70-77,209-216` (procrun with `procrun.SevenZip` budget)
- X15 - still fixed - S03 - `internal/comic/readers_sevenzip.go:88-107,188-224` (listing budget first, only chosen pages unpacked, `-spd`); `TestSevenZipBombRefusedFromListing`
- X16 - still fixed - S03 - `internal/comic/readers_sevenzip.go:229-246` (containment + Lstat regular-file check)
- X17 - still fixed - S03 - `internal/comic/extract.go:131-149,188-198` (pages streamed, capped); `TestExtractCBZStreamsPages`
- X24 - still fixed - S03 - `internal/comic/container.go:32-66`; `TestSniffContainer` (JS fallback drift: S03-5)
- B22 - still fixed - S03 - `extension/src/comic.js:267-317`; `test/comic.test.mjs` (precedence drift when both L and PAX path are set: S03-4)
- B23 - still fixed - S03 - `extension/src/comic.js:140-150,216-218`; `test/limits.test.mjs`
- Q1 - still fixed - S03 - `normalizeTarget` now lives in `browser_windows.go:40`; GOOS=linux lint exit 0
- Q2 - still fixed - S03 - `internal/comic/readers_sevenzip.go:251-265`; golangci-lint exit 0

**Files read in full:** app.go, splash.go, appearance.go, copier.go, css.go, html.go, browser.go,
browser_nonwindows.go, browser_windows.go, cache.go, doc.go, pdftotext_nonwindows.go, pdftotext_windows.go,
container.go, extract.go, natural.go, readers_sevenzip.go, readers_tar.go, readers_zip.go, comic.js - all 20.

**Findings** (0 crit, 0 high, 0 med, 6 low):

- S03-1 - low - conf - a stylesheet whose write fails after its own url()/@import files were copied dropped the *last* `infos` entry instead of its own, so a later reference to the same sheet by another spelling resolved (by file identity) to the name of a copy never written - `internal/assets/copier.go:122,133` - inline: remove by index; new `internal/assets/copier_test.go` `TestFailedSheetIsForgottenNotItsLastImport` failed before the fix (exit 1), `go test -count=1 ./internal/assets/` exit 0 after; `./internal/htmlconv/ ./internal/md/` exit 0; gofmt/vet/lint clean
- S03-2 - low - conf - `askYes` reads with `fmt.Scanln`, which leaves the rest of a multi-word line in stdin for the next prompt: "no thanks" silently answers the default-handler prompt with "hanks", and "n yy" answers it "y" (registry write the user never confirmed); reproduced with a probe program - `internal/app/app.go:296-306` (prompts at 59, 68) - ->
- S03-3 - low - conf - the extension's ZIP lister does not skip symlink entries (external attributes never read), while Go skips `!Mode().IsRegular()` and PARITY.md states "Symlink entries are skipped"; a CBZ with a symlink named `*.jpg` numbers its pages differently per edition - `extension/src/comic.js:191-222`, `internal/comic/readers_zip.go:20`, `docs/PARITY.md:510` - ->
- S03-4 - low - conf - TAR name precedence drift: Go applies the GNU `L` long name after PAX `path=` (GOROOT `archive/tar/reader.go:136-141`), JS lets PAX `path=` win - `extension/src/comic.js:299-302` - ->
- S03-5 - low - conf - with no known signature Go falls back to the extension's container (`.cbz` -> ZIP), JS always falls back to TAR, so a ZIP with leading data (SFX, prefixed) named `.cbz` converts on the desktop and fails as "no page images" in the extension; PARITY.md:512-514 says the extension is the fallback in both - `extension/src/comic.js:111-116`, `internal/comic/container.go:57-61` - ->
- S03-6 - low - plaus - `pruneOldSets` removes every other hash folder, including one a concurrently running different build (e.g. winget and Store installs) is using; that instance's cache check stats only `pdftotext.exe` (not its DLLs), so a half-removed folder passes the check and the run fails. Same cache check misses an AV-quarantined DLL - `internal/bundledtools/cache.go:80,126-138`, `internal/bundledtools/pdftotext_windows.go:35-38` - -> (area of tickets 33 / 35)

Tickets filed: none (no crit/high/med).
