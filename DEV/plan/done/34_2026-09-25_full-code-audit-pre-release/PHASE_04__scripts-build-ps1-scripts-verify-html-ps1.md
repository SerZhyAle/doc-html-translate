# Phase 04: S04 - scripts/build.ps1 .. scripts/verify-html.ps1

**Slice:** `S04` in [`slices.json`](slices.json) · **Lines:** 2564 in 20 files ·
**Added since `41fbc1b`:** 1747 · **Risk:** 37.4
**Status:** ✅ Done - findings in FINDINGS.md, tickets 36-47
**Read at commit:** 26605b9 + working tree of 2026-09-26



> Self-contained: this phase carries the whole slice procedure. The findings go to
> [`FINDINGS.md`](FINDINGS.md), the phase state to [`INDEX.md`](INDEX.md).

## Files

| File | Lines | Added since `41fbc1b` |
|------|------:|------:|
| `scripts/build.ps1` | 123 | 7 |
| `scripts/check.ps1` | 134 | 128 |
| `scripts/commit-push.ps1` | 68 | 0 |
| `scripts/commit_after_build.ps1` | 18 | 0 |
| `scripts/contract-gate.ps1` | 214 | 214 |
| `scripts/doc-query.ps1` | 81 | 81 |
| `scripts/doc-registry.ps1` | 249 | 249 |
| `scripts/generate-icon.ps1` | 39 | 26 |
| `scripts/lib/docregistry.ps1` | 192 | 192 |
| `scripts/lib/verdict.ps1` | 52 | 52 |
| `scripts/lint.ps1` | 20 | 9 |
| `scripts/parity-check.ps1` | 146 | 62 |
| `scripts/release-state.ps1` | 142 | 0 |
| `scripts/release.ps1` | 203 | 72 |
| `scripts/security-posture.ps1` | 368 | 368 |
| `scripts/test-extension.ps1` | 51 | 51 |
| `scripts/test.ps1` | 127 | 114 |
| `scripts/typo.ps1` | 20 | 9 |
| `scripts/verify-exe-version.ps1` | 93 | 93 |
| `scripts/verify-html.ps1` | 224 | 20 |

None - this slice holds its units whole.

Previous-register ids this slice re-checks: none - no 2026-09-24 finding cites these files.
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

### Step 04.1 - Prescan

- [x] Run the mechanical pass over this slice's own files and paste the verdict lines into the
      handoff notes:
  - Go packages ((none)): `go vet <pkgs>`, `$env:GOOS='linux'; go vet <pkgs>` (the
    non-Windows twins), `golangci-lint run <pkgs>` (it carries the staticcheck analyzers),
    `go test <pkgs>`.
  - Extension modules ((none)): every `extension/test/*.test.mjs` that imports one of them -
    `node --test <files>` from `extension/`.
  - Other files (scripts/build.ps1 scripts/check.ps1 scripts/commit-push.ps1 scripts/commit_after_build.ps1 scripts/contract-gate.ps1 scripts/doc-query.ps1 scripts/doc-registry.ps1 scripts/generate-icon.ps1 scripts/lib/docregistry.ps1 scripts/lib/verdict.ps1 scripts/lint.ps1 scripts/parity-check.ps1 scripts/release-state.ps1 scripts/release.ps1 scripts/security-posture.ps1 scripts/test-extension.ps1 scripts/test.ps1 scripts/typo.ps1 scripts/verify-exe-version.ps1 scripts/verify-html.ps1): PowerShell - `Invoke-ScriptAnalyzer` when installed, else a parse
    check (`[System.Management.Automation.Language.Parser]::ParseFile`); a parity pair -
    `./scripts/parity-check.ps1`.
- **Verification:** each command above has its exit code and last line recorded in the handoff
  notes; a red result is either explained as pre-existing (reproduced on the read commit) or becomes
  a finding.

### Step 04.2 - Re-check the previous register

- [x] For each id listed above (none for this slice), find the fixed code and decide: `still fixed`, `regressed`,
      `one edition only` (fixed in Go or JS but not its twin), or `superseded` (the code is gone or
      replaced and the defect cannot occur). Write one line per id under "Re-check" in
      `FINDINGS.md`: `- <id> - <verdict> - S04 - <file:line or test that shows it>`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` no longer lists these ids as missing.
  A `regressed` crit/high becomes a package-1 ticket at priority 90 (Step 04.4).

### Step 04.3 - Read

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

### Step 04.4 - Triage

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

### Step 04.5 - Close

- [x] Inline fixes land with their test run cited (command, exit code, last line); a changed Go file
      is `gofmt`-clean.
- [x] Flip this phase's row in `INDEX.md` to ✅ with the severity counts, and bump `Phases: X/N` in
      `INDEX.md` and the `k/N slices` in the ticket's status line and its `RELEASE_QUEUE.md` line.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` shows `S04` as closed.

## Done criteria

- Every step `[x]`, each with its verification met.
- The findings table below is filled, and every crit/high in it has a ticket or an inline fix.

## Handoff notes

**Prescan (2026-09-26, PowerShell 7).** No Go packages or extension modules in this slice.

- `Parser::ParseFile` over all 20 files - 0 parse errors each.
- `Invoke-ScriptAnalyzer` - not run: the PSScriptAnalyzer module is not installed here.
- `scripts/doc-registry.ps1` - exit 0 - `doc-registry: PASS (39 record(s), 230 document file(s) covered, 18 page(s) announced)`
- `scripts/security-posture.ps1` - exit 0 - `security-posture: PASS (13 permission row(s), 9 network surface(s), 7 declaration(s), 57 dependencies, 9 render(s))`
- `scripts/parity-check.ps1` - exit 0 - `parity-check: PASS (74 file(s) inspected, no one-sided change)`
- `scripts/typo.ps1` - exit 1 - `typo: FAIL`. This red result was already there before the audit: 85 of its 89 errors are in
  files that match HEAD (`internal/i18n/i18n_limits.go`, `i18n_pipeline.go`, `i18n_tools.go`, `i18n_ocr.go`,
  `i18n_epub.go`, `internal/translator/google.go`, `internal/rtf/*`, `extension/src/rtf.js`, ..): non-English copy that
  `configs/.typos.toml` does not exclude (it excludes `i18n_cli.go` / `i18n_reader.go` only). The cause is outside this
  slice. It is recorded as S04-16 because it keeps `check.ps1` at FAIL, so the gate evidence can't go green.
- Not run (per the rules): build*.ps1, release.ps1, commit*.ps1, generate-icon.ps1, check.ps1, contract-gate.ps1.

**Files read in full:** all 20 - build, check, commit-push, commit_after_build, contract-gate, doc-query,
doc-registry, generate-icon, lib/docregistry, lib/verdict, lint, parity-check, release-state, release,
security-posture, test-extension, test, typo, verify-exe-version, verify-html. For context only, I also read `scripts/build-local.ps1`,
`tests/verdict_scripts_test.go`, `.github/workflows/release.yml` (grep) and `.gitignore`.

**Findings (proposed ids S04-n; the orchestrator assigns the register ids R/Q and files the tickets):**

| id | sev | conf | finding | where |
|----|-----|------|---------|-------|
| S04-1 | high | conf | `check.ps1 -Plan <subset>` writes release-grade `gate-evidence.json` (code 0, the real tree hash). `release.ps1` checks only the code and the tree, never the plan or the children, so a one-child run turns the tag line green | check.ps1:50,121-131; release.ps1:63-71 |
| S04-2 | med | conf | The documented flow can never turn the gate evidence green: build-local runs check.ps1 first. It then rebuilds the tracked `build/*.exe`, stamped to the minute (`.gitignore:8`), and amends the commit with the tracked `DEV/COMMIT_LOG.md`, so HEAD's tree never equals the gated tree. The BLOCKED remedy "re-run scripts/build-local.ps1" loops | release.ps1:56,66,77; build.ps1:69; build-local.ps1:41,53,56,71,99-100 |
| S04-3 | med | conf | The evidence tree is hashed after every child has finished, so an edit made during the run (a parallel agent, the owner) is recorded as tested | check.ps1:125 (children at 59-72) |
| S04-4 | med | plaus | commit-push.ps1 says it "stays local + free", but `a c` on main runs `git add -A` (every untracked, non-ignored file) and pushes to origin main with no gate. GitHub Pages serves main, so the push publishes the site | commit-push.ps1:13-15,47,65 |
| S04-5 | low | conf | A failed goversioninfo or go build throws before cleanup, which leaves `resource.syso` and `versioninfo.generated.json` behind. Neither is gitignored, so the next `git add -A` commits them, and the amd64 syso breaks a default 386 build of the package | build.ps1:55-56,70-76 |
| S04-6 | low | conf | build.ps1 sets `$env:GOARCH/GOOS` in the caller's session and never restores them | build.ps1:67-68 |
| S04-7 | low | conf | The eng.traineddata fallback downloads from tessdata_fast `main` with no digest check. A file that is already there is never re-verified, so a partial download sticks (already noted as "Not done" in #13) | build.ps1:108,116 |
| S04-8 | low | plaus | `DEV/private/google_api.key` is copied into the deploy folder `C:\GD\tc\SZA\_APP`, which is probably synced to the cloud | build.ps1:90-93 |
| S04-9 | low | conf | The "blank-render" check only tests that the screenshot is at least 200x200 px and never looks at its pixels, so a white page passes | verify-html.ps1:193-201 |
| S04-10 | low | conf | Render and sitemap drift is compared with case-insensitive `-ne`, so drift that differs only in case passes | doc-registry.ps1:241; security-posture.ps1:358 |
| S04-11 | low | conf | contract-gate returns PASS when it checked 0 contracts (every pointer not applicable, or none at all) - a vacuous pass instead of 2 | contract-gate.ps1:153-156,211-214 |
| S04-12 | low | plaus | Product rows are matched by prefix (`-notlike "$Product*"`), so a sibling product's rows (for example `doc-html-translate-<x>`) would be read as this product's | contract-gate.ps1:112,124 |
| S04-13 | low | conf | `-Version` is not validated against YY.MMDD.HHmm, and the default is recomputed from the clock on every run, so two runs print different tags and URLs | release.ps1:27-50 |
| S04-14 | low | conf | The working-tree fallback (`git diff --name-only`) leaves out untracked files, so a new one-sided file is never seen | parity-check.ps1:81-84 |
| S04-15 | low | conf | A pipe character in -Note or -Ref is written into the table unescaped and shifts the columns on the next read | release-state.ps1:81,119 |
| S04-16 | med | conf | typo gate FAIL at HEAD (see prescan) keeps check.ps1 at 1, so the release gate is blocked; cause is `configs/.typos.toml` (outside the slice) | configs/.typos.toml extend-exclude |
| S04-17 | low | conf | When `-version` crashes or prints something unexpected, the result is "(not run)", not a problem | verify-exe-version.ps1:69-72,82 |

Counts: crit 0, high 1, med 4, low 12. No inline fixes. The high and med findings need tickets, and the scripts are on the release path. I made no edit to FINDINGS.md, INDEX.md or RELEASE_QUEUE.md (this was a parallel-auditor run), so steps 04.4 and 04.5 remain open.
