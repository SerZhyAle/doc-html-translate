# Phase 18: S18 - internal/textutil .. internal/windowsreg

**Slice:** `S18` in [`slices.json`](slices.json) · **Lines:** 2748 in 16 files ·
**Added since `41fbc1b`:** 1589 · **Risk:** 8.6
**Status:** ✅ Done - findings in FINDINGS.md, tickets 36-47
**Read at commit:** 26605b9 + working tree of 2026-09-26



> Self-contained: this phase carries the whole slice procedure. The findings go to
> [`FINDINGS.md`](FINDINGS.md), the phase state to [`INDEX.md`](INDEX.md).

## Files

| File | Lines | Added since `41fbc1b` |
|------|------:|------:|
| `internal/textutil/codec.go` | 109 | 109 |
| `internal/textutil/lines.go` | 41 | 3 |
| `internal/textutil/utf8.go` | 119 | 119 |
| `internal/translator/cache.go` | 88 | 30 |
| `internal/translator/google.go` | 272 | 272 |
| `internal/translator/ollama.go` | 459 | 201 |
| `internal/translator/retry.go` | 98 | 98 |
| `internal/translator/translator.go` | 89 | 21 |
| `internal/txt/decode.go` | 77 | 77 |
| `internal/txt/extract.go` | 234 | 41 |
| `internal/txt/legacy.go` | 90 | 19 |
| `internal/txt/sniff.go` | 113 | 5 |
| `extension/src/txt.js` | 226 | 103 |
| `internal/windowsreg/register_nonwindows.go` | 54 | 25 |
| `internal/windowsreg/register_windows.go` | 633 | 420 |
| `internal/windowsreg/status.go` | 46 | 46 |

None - this slice holds its units whole.

Previous-register ids this slice re-checks: P12, P13, P19, P23, X5, X21, X22, T1, T2, T3, T4, T5, T6, T7, T9.
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

### Step 18.1 - Prescan

- [x] Run the mechanical pass over this slice's own files and paste the verdict lines into the
      handoff notes:
  - Go packages (./internal/textutil ./internal/translator ./internal/txt ./internal/windowsreg): `go vet <pkgs>`, `$env:GOOS='linux'; go vet <pkgs>` (the
    non-Windows twins), `golangci-lint run <pkgs>` (it carries the staticcheck analyzers),
    `go test <pkgs>`.
  - Extension modules (extension/src/txt.js): every `extension/test/*.test.mjs` that imports one of them -
    `node --test <files>` from `extension/`.
  - Other files ((none)): PowerShell - `Invoke-ScriptAnalyzer` when installed, else a parse
    check (`[System.Management.Automation.Language.Parser]::ParseFile`); a parity pair -
    `./scripts/parity-check.ps1`.
- **Verification:** each command above has its exit code and last line recorded in the handoff
  notes; a red result is either explained as pre-existing (reproduced on the read commit) or becomes
  a finding.

### Step 18.2 - Re-check the previous register

- [x] For each id listed above, find the fixed code and decide: `still fixed`, `regressed`,
      `one edition only` (fixed in Go or JS but not its twin), or `superseded` (the code is gone or
      replaced and the defect cannot occur). Write one line per id under "Re-check" in
      `FINDINGS.md`: `- <id> - <verdict> - S18 - <file:line or test that shows it>`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` no longer lists these ids as missing.
  A `regressed` crit/high becomes a package-1 ticket at priority 90 (Step 18.4).

### Step 18.3 - Read

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

### Step 18.4 - Triage

- [x] Dedupe first: search `DEV/plan/`, `DEV/plan/done/` and the previous register for the symptom;
      a match gets a link, not a duplicate (a regression of an old id is a new ticket citing it).
- [x] crit / high: a ticket in package 1 of `DEV/plan/RELEASE_QUEUE.md` (next number from its
      `next-ticket-number`, which is then bumped), or an inline fix under the rule above.
      med: inline under the rule, else a ticket in the package its area belongs to.
      low: inline, or a register line with no ticket.
- [x] (coordinator: AUDITOR.md forbids this phase to edit FINDINGS.md) Write each finding under "New findings" in `FINDINGS.md`, ids continuing per area letter
      (P CLI/pipeline/support, G GUI, E EPUB/HTML, X extractors, T translation, O OCR,
      B extension, R release path, Q static checks) from the highest id in both registers:
      `- <id> - <sev> - <conf|plaus> - <finding> - <file:line> - <#NN | inline: <proof> | ->`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` reports no crit/high as untriaged.

### Step 18.5 - Close

- [x] Inline fixes land with their test run cited (command, exit code, last line); a changed Go file
      is `gofmt`-clean.
- [x] (coordinator: AUDITOR.md forbids this phase to edit INDEX.md / RELEASE_QUEUE.md) Flip this phase's row in `INDEX.md` to ✅ with the severity counts, and bump `Phases: X/N` in
      `INDEX.md` and the `k/N slices` in the ticket's status line and its `RELEASE_QUEUE.md` line.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` shows `S18` as closed.

## Done criteria

- Every step `[x]`, each with its verification met.
- The findings table below is filled, and every crit/high in it has a ticket or an inline fix.

## Handoff notes

Prescan (step 18.1):

- `go vet ./internal/textutil ./internal/translator ./internal/txt ./internal/windowsreg` - exit 0 - no output
- `$env:GOOS='linux'; go vet <same pkgs>` - exit 0 - no output
- `go test <same pkgs>` - exit 0 - `ok doc-html-translate/internal/windowsreg (cached)`
- `golangci-lint run --config configs/.golangci.yml <same pkgs>` - exit 0 - no output
- `node --check src/txt.js` - exit 0; `node --test test/txt.test.mjs test/legacy-text.test.mjs` - exit 0 - `pass 22, fail 0`
- Other files: none in this slice.

Files read in full: all 16 (textutil codec/lines/utf8; translator cache/google/ollama/retry/translator;
txt decode/extract/legacy/sniff; extension/src/txt.js; windowsreg register_nonwindows/register_windows/status).

Re-check (step 18.2):

- P12 - still fixed - S18 - register_windows.go:62-63 SHChangeNotify via changeTracker; extState:538-560 reads UserChoice/UserChoiceLatest; app.go:180-198 prints DONE only on Registration.Complete()
- P13 - still fixed - S18 - registerExtension:614-633 saves the previous handler; releaseExtension:469-495 restores it, errors reach Unregister:462-465
- P19 - still fixed - S18 - flags.go:149-160 rejects negative -ollama-parallel and notices clamps; ollama.go:95-109 clamps kept as a floor only
- P23 - still fixed - S18 - textutil/lines.go:24-26 uses DecodeUTF8 (U+FFFD per subpart), not ToValidUTF8(.., "")
- X5 - still fixed - S18 - txt/decode.go:65-77 acceptAsUTF8 == txt.js:187-192 acceptAsUtf8; legacy-text fixtures pass in both editions
- X21 - still fixed - S18 - txt/decode.go:25-54 sniffUTF16 == txt.js:119-133 sniffUtf16, before the UTF-8 step in both ladders
- X22 - still fixed (TXT half) - S18 - txt/extract.go:84-95 falls back to windows-1252 via LookupCodec, txt.js:220 likewise; the RTF half is another slice's
- T1 - still fixed - S18 - google.go:161 key in X-Goog-Api-Key header, transportCause:228-234 strips *url.Error, scrub:238-243; TestAPIKeyTravelsInHeaderOnly, TestNetworkErrorNeverShowsKey
- T2 - still fixed - S18 - google.go:101-105 html.EscapeString out, :125-127 UnescapeString back; TestEntityRoundTrip
- T3 - still fixed - S18 - google.go:25,247-272 128 q per request; 429/5xx/rate-limit 403 retried :175-176; TestThreeHundredSegments
- T4 - still fixed - S18 - google.go:114-123 reply count checked, cache.go:57-59 length check; TestShortReplyIsAnError, TestCacheRejectsWrongLength
- T5 - still fixed - S18 - ollama.go:200-217 returns PartialError with the successful batches; residual gaps are S18-2 and S18-4
- T6 - still fixed - S18 - retry.go:53-67 capped backoff with jitter, Retry-After honoured, 60 s budget; TestRetryHonoursRetryAfterSeconds, TestForbiddenRateLimitIsRetried
- T7 - still fixed - S18 - ollama.go:121-139 multi-line text sent alone, parseNumberedResponse:413-425 first answer wins; TestOllama1984LineKeepsItsSlot (one residual parser bug fixed inline, S18-1)
- T9 - still fixed - S18 - ollama.go:369-391 per-request timeout under ctx, 15 min until the model has answered once; TestOllamaLoadTimeoutOnlyUntilReady

Findings (steps 18.3-18.4):

| id | sev | conf | finding | evidence | disposition |
|----|-----|------|---------|----------|-------------|
| S18-1 | low | conf (run) | Ollama numbered-reply regex used `\s*` after "N.", which crosses a line break: an empty "2." answer took "3. text" as slot 2's translation and left slot 3 empty | translator/ollama.go:409 (was :407) | inline: `[ \t]*` instead of `\s*`, new internal/translator/ollama_parse_test.go; `go test ./internal/translator/` exit 0 `ok`; gofmt and golangci-lint clean |
| S18-2 | med | conf | Ollama slot still empty after the echo retries (model skipped the number, or the retry request itself failed - its error is dropped at :248-250) is returned as a success: the page counts as done, the run reports "Translation complete", and the cache stores "" for that text, so the segment silently stays in the source language | translator/ollama.go:242-261, cache.go:70-78, pipeline/translate.go:224-240 | ticket: pkg translation, "Ollama: report unfilled slots as PartialError" (changes the partial state of the completion record, so not inline) |
| S18-3 | med | conf (run) | TXT paragraphing drifts between editions: Go normalizes \f, \v, U+0085, U+2028, U+2029 to \n before splitting (textutil/lines.go:28-38), txt.js only \r\n and \r. "l1\nl2\n\f\nl3\nl4" gives Go 2 paragraphs (blank-line mode), JS 4; "a b\nc" gives Go 3, JS 2. Not listed in docs/PARITY.md | txt/extract.go:163, extension/src/txt.js:11 | ticket: pkg extension/parity, "txt: one line-separator set in both editions" (Go/JS twin - no one-side fix) |
| S18-4 | low | conf | GoogleClient.Translate drops every finished batch of a page when a later batch fails (returns nil, err), while Ollama returns a PartialError with them; the book stops either way, the Google page just loses the batches it had | translator/google.go:83-89 | - |

Dedupe: S18-2 is the gap left by ticket 07 (done/07, "cache does not store the missing slots") - only slots of a
failed batch are marked missing; no open ticket covers it. S18-3, S18-4: no match in DEV/plan, DEV/plan/done or the
previous register. Proposed register ids: S18-1 -> T10, S18-2 -> T11, S18-4 -> T12, S18-3 -> X25.

Per AUDITOR.md this phase did not edit FINDINGS.md, INDEX.md or RELEASE_QUEUE.md and filed no ticket file; the
coordinator carries the lines above into them. windowsreg: no new defect found; it is ticket-only in any case.
