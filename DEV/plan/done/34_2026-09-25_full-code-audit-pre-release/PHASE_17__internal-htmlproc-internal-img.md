# Phase 17: S17 - internal/htmlproc .. internal/img

**Slice:** `S17` in [`slices.json`](slices.json) · **Lines:** 2937 in 18 files ·
**Added since `41fbc1b`:** 2149 · **Risk:** 12.4
**Status:** ✅ Done - findings in FINDINGS.md, tickets 36-47
**Read at commit:** 26605b9 + working tree of 2026-09-26



> Self-contained: this phase carries the whole slice procedure. The findings go to
> [`FINDINGS.md`](FINDINGS.md), the phase state to [`INDEX.md`](INDEX.md).

## Files

| File | Lines | Added since `41fbc1b` |
|------|------:|------:|
| `internal/htmlproc/htmlproc.go` | 104 | 3 |
| `internal/htmlsplit/chunk.go` | 312 | 312 |
| `internal/htmlsplit/links.go` | 195 | 195 |
| `internal/htmlsplit/split.go` | 127 | 23 |
| `internal/i18n/i18n.go` | 127 | 0 |
| `internal/i18n/i18n_cli.go` | 423 | 269 |
| `internal/i18n/i18n_epub.go` | 49 | 49 |
| `internal/i18n/i18n_limits.go` | 119 | 119 |
| `internal/i18n/i18n_ocr.go` | 62 | 62 |
| `internal/i18n/i18n_pipeline.go` | 163 | 163 |
| `internal/i18n/i18n_reader.go` | 100 | 25 |
| `internal/i18n/i18n_tools.go` | 77 | 77 |
| `internal/iconart/contrast.go` | 27 | 27 |
| `internal/iconart/mark.go` | 107 | 107 |
| `internal/iconart/outputs.go` | 230 | 230 |
| `internal/iconart/path.go` | 226 | 226 |
| `internal/iconart/raster.go` | 176 | 176 |
| `internal/img/extract.go` | 313 | 86 |

None - this slice holds its units whole.

Previous-register ids this slice re-checks: P8, E1, E7, E8, E9, X11, X12, X13, X23, T2, Q3.
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

### Step 17.1 - Prescan

- [x] Run the mechanical pass over this slice's own files and paste the verdict lines into the
      handoff notes:
  - Go packages (./internal/htmlproc ./internal/htmlsplit ./internal/i18n ./internal/iconart ./internal/img): `go vet <pkgs>`, `$env:GOOS='linux'; go vet <pkgs>` (the
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

### Step 17.2 - Re-check the previous register

- [x] For each id listed above, find the fixed code and decide: `still fixed`, `regressed`,
      `one edition only` (fixed in Go or JS but not its twin), or `superseded` (the code is gone or
      replaced and the defect cannot occur). Write one line per id under "Re-check" in
      `FINDINGS.md`: `- <id> - <verdict> - S17 - <file:line or test that shows it>`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` no longer lists these ids as missing.
  A `regressed` crit/high becomes a package-1 ticket at priority 90 (Step 17.4).

### Step 17.3 - Read

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

### Step 17.4 - Triage

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

### Step 17.5 - Close

- [x] Inline fixes land with their test run cited (command, exit code, last line); a changed Go file
      is `gofmt`-clean.
- [x] Flip this phase's row in `INDEX.md` to ✅ with the severity counts, and bump `Phases: X/N` in
      `INDEX.md` and the `k/N slices` in the ticket's status line and its `RELEASE_QUEUE.md` line.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` shows `S17` as closed.

## Done criteria

- Every step `[x]`, each with its verification met.
- The findings table below is filled, and every crit/high in it has a ticket or an inline fix.

## Handoff notes

Steps 17.1-17.3 done here; 17.4 dedupe done, the FINDINGS.md / ticket / INDEX.md writes of 17.4-17.5
are the orchestrator's (this auditor may not edit those files).

**Prescan** (PowerShell, pkgs = ./internal/htmlproc ./internal/htmlsplit ./internal/i18n ./internal/iconart ./internal/img):

- `go vet <pkgs>` - exit 0 - (no output)
- `$env:GOOS='linux'; go vet <pkgs>` - exit 0 - (no output)
- `go test <pkgs>` - exit 0 - `ok doc-html-translate/internal/img`
- `golangci-lint run --config configs/.golangci.yml <pkgs>` - exit 0 - (no output)
- After the inline fixes: `go test <pkgs>` exit 0; lint on htmlproc/htmlsplit/iconart exit 0; `gofmt -l` clean.
- Extra (not a gate): a throwaway verb-parity test over `i18n.All()` - every translation carries the
  key's verbs (hi/ur use indexed `%[2]d`/`%[1]d`, valid); no key is registered twice (80 `Add` calls,
  80 distinct keys).

**Re-check** (for FINDINGS.md):

- P8 - still fixed (htmlproc half) - S17 - `htmlproc/htmlproc.go:99-104` RenderToFile goes through `fsutil.WriteFile` (temp + rename); the Ctrl+C half is S08's
- E1 - still fixed - S17 - `htmlsplit/split.go:63,89` join only manifest hrefs that passed `resolveBookPath` (`epub/epub.go:363-368`); part names stay in the source's folder
- E7 - still fixed (gap closed inline, S17-1) - S17 - `htmlsplit/links.go:71-90` rewriteTOC, `:89-126` rewriteContentLinks; `TestSplitSingleWrapperRewritesTOC`, `TestSplitRewritesCrossFileAndInFileLinks`
- E8 - still fixed - S17 - `htmlsplit/chunk.go:77-97` buildPage shallow-clones `<html>` and `<body>` with their attributes; `TestSplitKeepsRootAndBodyAttributes`
- E9 - still fixed - S17 - `chunk.go:126-144` descends a sole block wrapper, `:112` counts runes; `TestSplitCountsCharactersNotBytes`
- X11 - still fixed (img half) - S17 - `img/extract.go:241-249` `limits.TIFFFrameSize` + `CheckPixels` before `tiff.Decode`; x/image v0.45 bounds strip reads (`blockMaxDataSize`); the PDF-flip half is S05's
- X12 - still fixed - S17 - `img/extract.go:195,203` compare offsets in int64
- X13 - still fixed - S17 - `img/extract.go:223-249` frameView reads through the file, no per-frame copy
- X23 - still fixed - S17 - `img/extract.go:310` `html.EscapeString(epub.URLPath(imgName))`
- T2 - still fixed - S17 - htmlproc stores decoded DOM text (`htmlproc.go:56`), escape/unescape round trip at `translator/google.go:101-127`
- Q3 - still fixed - S17 - `i18n/i18n_cli.go:383` spells the mark `‏`; golangci-lint (ST1018) clean

**Findings:**

| id | sev | conf | finding | site | resolution |
|----|-----|------|---------|------|------------|
| S17-1 | low | conf | rewriteTOC looked a TOC fragment up only as written; a percent-encoded fragment (`#%D0%B3` for `id="г"`, as nav docs may write it) never matched the decoded id, so the entry kept pointing at part 1 | `htmlsplit/links.go:78` (pre-fix) | inline: decode before lookup, write back as written (`links.go:80-84`); new `toc_fragment_test.go` failed before, passes after |
| S17-2 | low | conf | ReplaceTexts restored only ASCII edge whitespace while ExtractTexts trims every Unicode space: `Chapter&nbsp;<b>1</b>` translated came out glued (`Глава<b>1</b>`) | `htmlproc/htmlproc.go:84-93` (pre-fix) | inline: `TrimLeftFunc/TrimRightFunc(unicode.IsSpace)` (`htmlproc.go:86-87`); new `whitespace_test.go` failed before, passes after |
| S17-3 | med | conf | translation runs after nav injection and htmlproc skips only script/style/code/pre, so every page's navbar chrome (source file name, title, "Previous page"/"Next page"/"Table of contents", theme/font `<option>` labels, version label, "3 / 10") is sent to the paid engine and replaced by target-language text inside a bar still marked `lang=<ui>` | `htmlproc/htmlproc.go:19-24,51-68`, `pipeline/pipeline.go:295,311`, `htmlgen/navbar.go:580-621` | ticket (translation package): skip `.dht-navbar` / `translate="no"` subtrees; touches htmlgen output and the cost estimate |
| S17-4 | low | conf | parsePath looped forever on numbers after a closepath (`Z 5 5`): the `z` case never consumed input; OOM in ~1 s on 386. Input is the vendored glyphs (dev tool only) | `iconart/path.go:50-54` (pre-fix) | inline: refuse a non-command after `Z` (`path.go:54-58`); `robust_test.go`; RepoOutputs byte-identical to the committed ICO/PNG files, MSIXOutputs still builds |
| S17-5 | low | conf | DecodeICO sliced the directory without checking its length and summed `off+size` in uint32 (wraps) - a panic, not an error, on a short or lying ICO (test/tool input only) | `iconart/outputs.go:214-221` (pre-fix) | inline: directory-length check + 64-bit sum (`outputs.go:215,224`); `robust_test.go` |
| S17-6 | low | conf | a split part is written to `<base>_sN.html` with id `<id>_sN` without checking the manifest: a book that already ships `ch_s2.html` has that chapter overwritten by part 2 of `ch.html` before it is read | `htmlsplit/split.go:84-90` | register line (extends ticket 01's "out of scope" note on case-only clashes) |
| S17-7 | low | plaus | image input pages declare `<html lang="en">` whatever the scan's language; Chrome may weigh the declared language against its detector and not offer "Translate page" for a non-English scan (same pattern in comic/pdf/txt/rtf/fb2; E22 covered md/html only) | `img/extract.go:298` | register line; output-format change, cross-extractor |

Files read in full: htmlproc.go, chunk.go, links.go, split.go, i18n.go, i18n_cli.go / i18n_epub.go /
i18n_limits.go / i18n_ocr.go / i18n_pipeline.go / i18n_reader.go / i18n_tools.go (mechanics: `init`
+ `Add` only, no other code; wording not reviewed), contrast.go, mark.go, outputs.go, path.go,
raster.go, extract.go - all 18 ticked.

Note: `internal/iconart/` is untracked in the working tree (owner's uncommitted ticket-32 work); the
S17-4/S17-5 edits and `robust_test.go` sit in that untracked folder.
