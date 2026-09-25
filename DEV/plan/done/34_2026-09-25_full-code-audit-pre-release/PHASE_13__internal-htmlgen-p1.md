# Phase 13: S13 - internal/htmlgen (part 1 of 2)

**Slice:** `S13` in [`slices.json`](slices.json) · **Lines:** 2305 in 9 files ·
**Added since `41fbc1b`:** 604 · **Risk:** 14.8
**Status:** ✅ Done - findings in FINDINGS.md, tickets 36-47
**Read at commit:** 26605b9 + working tree of 2026-09-26



> Self-contained: this phase carries the whole slice procedure. The findings go to
> [`FINDINGS.md`](FINDINGS.md), the phase state to [`INDEX.md`](INDEX.md).

## Files

| File | Lines | Added since `41fbc1b` |
|------|------:|------:|
| `internal/htmlgen/encode.go` | 13 | 13 |
| `internal/htmlgen/favicon.go` | 58 | 8 |
| `internal/htmlgen/glyphs.go` | 54 | 54 |
| `internal/htmlgen/htmlgen.go` | 452 | 82 |
| `internal/htmlgen/merge.go` | 287 | 287 |
| `internal/htmlgen/navbar.go` | 819 | 62 |
| `internal/htmlgen/reader_key.go` | 31 | 31 |
| `internal/htmlgen/singlepage.go` | 303 | 61 |
| `internal/htmlgen/toc_scan.go` | 288 | 6 |

Sibling slices of the same unit, adjacent in the order: S14. Read their files for context only; audit them in their own phase.

Previous-register ids this slice re-checks: E1, E2, E3, E4, E13, E14, E17, E18, E19, E21, E22, E23, E24.
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

### Step 13.1 - Prescan

- [x] Run the mechanical pass over this slice's own files and paste the verdict lines into the
      handoff notes:
  - Go packages (./internal/htmlgen): `go vet <pkgs>`, `$env:GOOS='linux'; go vet <pkgs>` (the
    non-Windows twins), `golangci-lint run <pkgs>` (it carries the staticcheck analyzers),
    `go test <pkgs>`.
  - Extension modules ((none)): every `extension/test/*.test.mjs` that imports one of them -
    `node --test <files>` from `extension/`.
  - Other files ((none)): PowerShell - `Invoke-ScriptAnalyzer` when installed, else a parse
    check (`[System.Management.Automation.Language.Parser]::ParseFile`); a parity pair -
    `./scripts/parity-check.ps1`.
- **Verification:** each command above has its exit code and last line recorded in the handoff
  notes; a red result is either explained as pre-existing (reproduced on the read commit) or becomes
  a finding.

### Step 13.2 - Re-check the previous register

- [x] For each id listed above, find the fixed code and decide: `still fixed`, `regressed`,
      `one edition only` (fixed in Go or JS but not its twin), or `superseded` (the code is gone or
      replaced and the defect cannot occur). Write one line per id under "Re-check" in
      `FINDINGS.md`: `- <id> - <verdict> - S13 - <file:line or test that shows it>`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` no longer lists these ids as missing.
  A `regressed` crit/high becomes a package-1 ticket at priority 90 (Step 13.4).

### Step 13.3 - Read

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

### Step 13.4 - Triage

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

### Step 13.5 - Close

- [x] Inline fixes land with their test run cited (command, exit code, last line); a changed Go file
      is `gofmt`-clean.
- [x] Flip this phase's row in `INDEX.md` to ✅ with the severity counts, and bump `Phases: X/N` in
      `INDEX.md` and the `k/N slices` in the ticket's status line and its `RELEASE_QUEUE.md` line.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` shows `S13` as closed.

## Done criteria

- Every step `[x]`, each with its verification met.
- The findings table below is filled, and every crit/high in it has a ticket or an inline fix.

## Handoff notes

**Prescan (2026-09-26, all green):**

- `go vet ./internal/htmlgen/` - exit 0, no output.
- `$env:GOOS='linux'; go vet ./internal/htmlgen/` - exit 0, no output.
- `golangci-lint run --allow-serial-runners --config configs/.golangci.yml ./internal/htmlgen/` - exit 0, no output
  (a first run without the flag exited 3, "parallel golangci-lint is running" - another auditor's lock, not a result).
- `go test ./internal/htmlgen/` - exit 0, `ok doc-html-translate/internal/htmlgen 5.345s`.
- Twin gate for glyphs.go / the palette: `go test ./tests/ -run 'Iconograph|Glyph|Palette|Typography' -count=1` -
  exit 0, `ok doc-html-translate/tests 0.221s` (12 tests ran, all PASS).
- No extension module and no other file type in this slice.

**Files read in full:** [x] `encode.go` [x] `favicon.go` [x] `glyphs.go` [x] `htmlgen.go` [x] `merge.go`
[x] `navbar.go` (1-819) [x] `reader_key.go` [x] `singlepage.go` [x] `toc_scan.go`

**Re-check:**

- E1 - still fixed - S13 - containment moved upstream to the one gate `internal/epub/resolve.go:32-81` (`resolveBookPath`, applied to every manifest href at `epub.go:361-369`); `singlepage.go:162-169` deletes only resolved spine hrefs and never the merged file (`EqualFold` guard); `containment_test.go` passes
- E2 - still fixed - S13 - `reader_key.go:17-31`; key set once at `pipeline.go:274` before translation; `htmlgen.go:120`, `navbar.go:688`, `singlepage.go:142` all read `readerKey(book)`
- E3 - still fixed - S13 - `merge.go:160-216` rebases link attrs, style attrs and body `<style>` from the chapter folder; `TestSinglePageMergeSigilLayout`
- E4 - still fixed - S13 - `merge.go:85-129` (only colliding ids renamed `cN-`), `merge.go:175-178,219-234` (spine links become in-page anchors, `dht-ch-N` marker)
- E13 - still fixed - S13 - `navbar.go:543` (`!location.hash` guards the restore)
- E14 - still fixed - S13 - htmlgen reads pages already transcoded to UTF-8 by `internal/epub/normalize.go:127` (`decodeToUTF8`, `charset.go:24-55`); HTML input by `htmlconv/extract.go:97-110`
- E17 - still fixed - S13 - `singlepage.go:114,116` `html.EscapeString(lang)`; `TestSinglePageLangIsEscaped`
- E18 - still fixed - S13 - `htmlgen.go:246-253` (`epub.ExternalHref`: external kept as written without the base prefix, a non-clickable scheme leaves only the label); `epub/toc.go:307-309`
- E19 - still fixed - S13 - `epub.URLPath` at `navbar.go:708,711,724`, `htmlgen.go:49,185,269`, `toc_scan.go:45,141`, `singlepage.go:101,174`; script values only via `jsString` (`encode.go:10`)
- E21 - still fixed - S13 - `favicon.go:21` exports `FaviconName`, reserved by `internal/assets/copier.go:183`
- E22 - still fixed - S13 - the index no longer hardcodes `lang="en"`: `htmlgen.go:72,135-149` takes the first page's lang/dir (the residual "en" fallback for a page that declares none is S13-1)
- E23 - still fixed - S13 - `navbar.go:746` (reader marker makes injection idempotent; `TestInjectNavBarsIsIdempotent`); xml:lang honoured at `singlepage.go:261`, `htmlgen.go:168`
- E24 - still fixed - S13 - `reader_key.go:17-21` (source name + size + title + page count, 64-bit FNV)

**Findings (proposed ids S13-n; severities: 0 crit, 0 high, 2 med, 2 low):**

- S13-1 - med - plaus - the single-page merge and the TOC index fall back to `<html lang="en">` when the first page declares no language, while `internal/md/extract.go:152-154` deliberately declares none ("a wrong `<html lang>` can stop Chrome offering Translate page"): a multi-page Markdown book, and any EPUB whose chapters carry no `lang`/`xml:lang` (OPF `dc:language` is never read), is labelled English in the default flow. Plaus: the impact depends on how Chrome treats a mis-declared `en` - `singlepage.go:51`, `htmlgen.go:136` - ticket: package 1, "no guessed lang=en on the merged page and the index" (htmlgen side of S09-2)
- S13-2 - med - conf - the single-page merge (the default mode, `internal/config/flags.go:175`) keeps only each page's `<body>` children, so every page's `<head>` `<style>` (and its `<body>` attributes) is dropped: PDF pages lose `.pdf-flip-y { transform: scaleY(-1) }` (a flipped image renders mirrored) and the side-by-side float layout, FB2 its stanza/subtitle/author styles, EPUB chapters their inline head CSS. Probed with a throwaway test (deleted afterwards): two pages carrying the flip rule in `<head>`, merged -> `class kept=true rule kept=false` - `singlepage.go:72-79,98-103`; rule source `internal/pdf/extract.go:426,778` - ticket: package 1, "single-page merge drops per-page head styles"
- S13-3 - low - conf - the TOC index's "Chapters: N" line is an interface string (`i18n.S`) inside a page whose `<html lang>` is the book's, with no `lang`/`dir` of its own, unlike the toolbar right below it (`htmlgen.go:113`) - `htmlgen.go:111` - register line
- S13-4 - low - conf - the index reads `dir` from `<html>` only (`pageRootLang`) while the merge falls back to `<body dir>` (`htmlDir`): an RTL book declaring direction on `<body>` gets an LTR TOC index in multi-page mode but an RTL merged page - `htmlgen.go:170` vs `singlepage.go:275-288` - register line

No inline fix: all four change emitted HTML (output format), and navbar.go / glyphs.go have JS twins.
Dedupe: S13-2 is not in the 2026-09-24 register, `done/09` (which rebased body `<style>` blocks only) or any open
ticket; S13-1 is the htmlgen half of S09-2 and the residue of E22.
