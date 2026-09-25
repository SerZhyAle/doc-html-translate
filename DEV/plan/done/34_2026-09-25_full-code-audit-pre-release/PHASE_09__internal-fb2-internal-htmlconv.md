# Phase 09: S09 - internal/fb2 .. internal/htmlconv

**Slice:** `S09` in [`slices.json`](slices.json) · **Lines:** 1124 in 6 files ·
**Added since `41fbc1b`:** 691 · **Risk:** 24.4
**Status:** ✅ Done - findings in FINDINGS.md, tickets 36-47
**Read at commit:** 26605b9 + working tree of 2026-09-26



> Self-contained: this phase carries the whole slice procedure. The findings go to
> [`FINDINGS.md`](FINDINGS.md), the phase state to [`INDEX.md`](INDEX.md).

## Files

| File | Lines | Added since `41fbc1b` |
|------|------:|------:|
| `internal/fb2/content.go` | 297 | 297 |
| `internal/fb2/extract.go` | 247 | 72 |
| `extension/src/fb2.js` | 218 | 124 |
| `internal/fsutil/atomic.go` | 74 | 74 |
| `internal/htmlconv/extract.go` | 260 | 122 |
| `extension/src/html.js` | 28 | 2 |

None - this slice holds its units whole.

Previous-register ids this slice re-checks: E14, E15, E21, E22, X4, X18, X19, X20.
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

### Step 09.1 - Prescan

- [x] Run the mechanical pass over this slice's own files and paste the verdict lines into the
      handoff notes:
  - Go packages (./internal/fb2 ./internal/fsutil ./internal/htmlconv): `go vet <pkgs>`, `$env:GOOS='linux'; go vet <pkgs>` (the
    non-Windows twins), `golangci-lint run <pkgs>` (it carries the staticcheck analyzers),
    `go test <pkgs>`.
  - Extension modules (extension/src/fb2.js extension/src/html.js): every `extension/test/*.test.mjs` that imports one of them -
    `node --test <files>` from `extension/`.
  - Other files ((none)): PowerShell - `Invoke-ScriptAnalyzer` when installed, else a parse
    check (`[System.Management.Automation.Language.Parser]::ParseFile`); a parity pair -
    `./scripts/parity-check.ps1`.
- **Verification:** each command above has its exit code and last line recorded in the handoff
  notes; a red result is either explained as pre-existing (reproduced on the read commit) or becomes
  a finding.

### Step 09.2 - Re-check the previous register

- [x] For each id listed above, find the fixed code and decide: `still fixed`, `regressed`,
      `one edition only` (fixed in Go or JS but not its twin), or `superseded` (the code is gone or
      replaced and the defect cannot occur). Write one line per id under "Re-check" in
      `FINDINGS.md`: `- <id> - <verdict> - S09 - <file:line or test that shows it>`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` no longer lists these ids as missing.
  A `regressed` crit/high becomes a package-1 ticket at priority 90 (Step 09.4).

### Step 09.3 - Read

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

### Step 09.4 - Triage

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

### Step 09.5 - Close

- [x] Inline fixes land with their test run cited (command, exit code, last line); a changed Go file
      is `gofmt`-clean.
- [x] Flip this phase's row in `INDEX.md` to ✅ with the severity counts, and bump `Phases: X/N` in
      `INDEX.md` and the `k/N slices` in the ticket's status line and its `RELEASE_QUEUE.md` line.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` shows `S09` as closed.

## Done criteria

- Every step `[x]`, each with its verification met.
- The findings table below is filled, and every crit/high in it has a ticket or an inline fix.

## Handoff notes

**Prescan (2026-09-26, all green):**

- `go vet ./internal/fb2 ./internal/fsutil ./internal/htmlconv` - exit 0, no output.
- `$env:GOOS='linux'; go vet` (same packages) - exit 0, no output.
- `golangci-lint run --config configs/.golangci.yml ./internal/fb2/ ./internal/fsutil/ ./internal/htmlconv/` - exit 0, no output.
- `go test ./internal/fb2 ./internal/fsutil ./internal/htmlconv` - exit 0, `ok doc-html-translate/internal/htmlconv (cached)`.
- `node --test test/fb2.test.mjs test/legacy-text.test.mjs` (from `extension/`) - exit 0, `pass 12, fail 0`.

**Files read in full:** [x] `internal/fb2/content.go` [x] `internal/fb2/extract.go` [x] `extension/src/fb2.js`
[x] `internal/fsutil/atomic.go` [x] `internal/htmlconv/extract.go` [x] `extension/src/html.js`

**Re-check:**

- E14 - one edition only - S09 - Go `internal/htmlconv/extract.go:97-110` (BOM, meta, detection); `extension/src/html.js:11` still `TextDecoder("utf-8")`, recorded as an open gap at `docs/PARITY.md:419`, no open ticket
- E15 - still fixed - S09 - `htmlconv/extract.go:143` -> `internal/assets/html.go:9-35,107-138` (img/source srcset)
- E21 - still fixed - S09 - `internal/assets/copier.go:157-185` (case-folded claim + `reserved`), `copier.go:102` (EvalSymlinks containment)
- E22 - still fixed - S09 - `htmlconv/extract.go:184-208` (source lang/dir), `154-178` (head CSS kept); `internal/md/extract.go:42,103,155` (no hardcoded lang, images copied)
- X4 - still fixed - S09 - `internal/fb2/content.go:59-77,86`; `extension/src/fb2.js:21-40`; `fb2.test.mjs` pass
- X18 - still fixed - S09 - `content.go:128-173`; `fb2.js:134-154` (but see S09-1: the extension drops a stanza's own title/subtitle)
- X19 - still fixed - S09 - `internal/fb2/extract.go:183-210` (FNV hash for sanitized ids, dots stripped, case-folded set)
- X20 - still fixed - S09 - `content.go:211-257` (16 KB chunked AppendDecode, single pass)

**Findings (proposed ids S09-n; severities: 0 crit, 0 high, 2 med, 5 low):**

- S09-1 - med - conf - extension drops a `<stanza>`'s own `<title>` and `<subtitle>` (renders only `<v>` children); Go keeps both. Probed on the same FB2: Go wrote `<p>StanzaTitle</p><p class="subtitle">StanzaSub</p>`, the extension only the verse - `extension/src/fb2.js:144-147`, `internal/fb2/content.go:128-173` - ticket (parity)
- S09-2 - med - conf - Go FB2 pages hardcode `<html lang="en">` and ignore `<title-info><lang>`, which `fb2.js:91` reads; a Russian FB2 declares English, and the single-page merge copies it - `internal/fb2/extract.go:129`, `internal/htmlgen/singlepage.go:64` - ticket (the same literal sits in txt/rtf/pdf/img/comic, other slices)
- S09-3 - low - conf - extension never shows the FB2 `<coverpage>` image and silently drops an `<image>` with no binary; Go prepends the cover and writes a visible placeholder. `docs/PARITY.md:1061` says both show them - `fb2.js:100-101,183`, `content.go:178-181`, `extract.go:69-71,148-149` - ticket (parity)
- S09-4 - low - conf - FB2 image names skip the generator's reserved names and take the extension from the id: probed, binary id `page_001.html` was written as an image and then overwritten by the page; `index.html`, `favicon.ico`, `.exe`/`.lnk`/`.ini` ids land the same way. `internal/assets` already has `reserved()` - `internal/fb2/extract.go:183-210,62`, `internal/assets/copier.go:178-185` - register line
- S09-5 - low - conf (Go) / plaus (browser) - a `<binary>` without `=` padding, or with `\f`/`\v` in it, fails Go's strict `StdEncoding` and shows a placeholder, while the extension's `data:` URL decodes it forgivingly (pre-existing since 41fbc1b) - `content.go:229,235,246`, `fb2.js:85` - register line
- S09-6 - low - conf - HTML-input lang: Go also takes `xml:lang` and a `<body lang>` and writes it un-normalized, the extension reads only `<html lang>` and normalizes it - `htmlconv/extract.go:184-208`, `html.js:15` - register line
- S09-7 - low - conf - `docs/PARITY.md:1066` names `copyLocalImages` in htmlconv, a function that no longer exists (now `internal/assets` `Copier`) - `docs/PARITY.md:1066` - docs

No inline fix: every Go finding sits in a package with a JS twin or needs an output change. `internal/fsutil/atomic.go`: no defect found.
