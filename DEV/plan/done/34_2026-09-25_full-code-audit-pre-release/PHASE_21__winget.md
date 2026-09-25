# Phase 21: S21 - winget

**Slice:** `S21` in [`slices.json`](slices.json) · **Lines:** 605 in 15 files ·
**Added since `41fbc1b`:** 0 · **Risk:** 0
**Status:** ✅ Done - findings in FINDINGS.md, tickets 36-47
**Read at commit:** 26605b9 + working tree of 2026-09-26



> Self-contained: this phase carries the whole slice procedure. The findings go to
> [`FINDINGS.md`](FINDINGS.md), the phase state to [`INDEX.md`](INDEX.md).

## Files

| File | Lines | Added since `41fbc1b` |
|------|------:|------:|
| `winget/SerZhyAle.DocHtmlTranslate.installer.yaml` | 18 | 0 |
| `winget/SerZhyAle.DocHtmlTranslate.locale.ar-SA.yaml` | 43 | 0 |
| `winget/SerZhyAle.DocHtmlTranslate.locale.bn-BD.yaml` | 44 | 0 |
| `winget/SerZhyAle.DocHtmlTranslate.locale.de-DE.yaml` | 45 | 0 |
| `winget/SerZhyAle.DocHtmlTranslate.locale.en-US.yaml` | 51 | 0 |
| `winget/SerZhyAle.DocHtmlTranslate.locale.es-ES.yaml` | 45 | 0 |
| `winget/SerZhyAle.DocHtmlTranslate.locale.fr-FR.yaml` | 45 | 0 |
| `winget/SerZhyAle.DocHtmlTranslate.locale.hi-IN.yaml` | 44 | 0 |
| `winget/SerZhyAle.DocHtmlTranslate.locale.it-IT.yaml` | 45 | 0 |
| `winget/SerZhyAle.DocHtmlTranslate.locale.pt-BR.yaml` | 45 | 0 |
| `winget/SerZhyAle.DocHtmlTranslate.locale.ru-RU.yaml` | 45 | 0 |
| `winget/SerZhyAle.DocHtmlTranslate.locale.uk-UA.yaml` | 44 | 0 |
| `winget/SerZhyAle.DocHtmlTranslate.locale.ur-PK.yaml` | 44 | 0 |
| `winget/SerZhyAle.DocHtmlTranslate.locale.zh-CN.yaml` | 40 | 0 |
| `winget/SerZhyAle.DocHtmlTranslate.yaml` | 7 | 0 |

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

### Step 21.1 - Prescan

- [x] Run the mechanical pass over this slice's own files and paste the verdict lines into the
      handoff notes:
  - Go packages ((none)): `go vet <pkgs>`, `$env:GOOS='linux'; go vet <pkgs>` (the
    non-Windows twins), `golangci-lint run <pkgs>` (it carries the staticcheck analyzers),
    `go test <pkgs>`.
  - Extension modules ((none)): every `extension/test/*.test.mjs` that imports one of them -
    `node --test <files>` from `extension/`.
  - Other files (winget/SerZhyAle.DocHtmlTranslate.installer.yaml winget/SerZhyAle.DocHtmlTranslate.locale.ar-SA.yaml winget/SerZhyAle.DocHtmlTranslate.locale.bn-BD.yaml winget/SerZhyAle.DocHtmlTranslate.locale.de-DE.yaml winget/SerZhyAle.DocHtmlTranslate.locale.en-US.yaml winget/SerZhyAle.DocHtmlTranslate.locale.es-ES.yaml winget/SerZhyAle.DocHtmlTranslate.locale.fr-FR.yaml winget/SerZhyAle.DocHtmlTranslate.locale.hi-IN.yaml winget/SerZhyAle.DocHtmlTranslate.locale.it-IT.yaml winget/SerZhyAle.DocHtmlTranslate.locale.pt-BR.yaml winget/SerZhyAle.DocHtmlTranslate.locale.ru-RU.yaml winget/SerZhyAle.DocHtmlTranslate.locale.uk-UA.yaml winget/SerZhyAle.DocHtmlTranslate.locale.ur-PK.yaml winget/SerZhyAle.DocHtmlTranslate.locale.zh-CN.yaml winget/SerZhyAle.DocHtmlTranslate.yaml): PowerShell - `Invoke-ScriptAnalyzer` when installed, else a parse
    check (`[System.Management.Automation.Language.Parser]::ParseFile`); a parity pair -
    `./scripts/parity-check.ps1`.
- **Verification:** each command above has its exit code and last line recorded in the handoff
  notes; a red result is either explained as pre-existing (reproduced on the read commit) or becomes
  a finding.

### Step 21.2 - Re-check the previous register

- [x] For each id listed above, find the fixed code and decide: `still fixed`, `regressed`,
      `one edition only` (fixed in Go or JS but not its twin), or `superseded` (the code is gone or
      replaced and the defect cannot occur). Write one line per id under "Re-check" in
      `FINDINGS.md`: `- <id> - <verdict> - S21 - <file:line or test that shows it>`.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` no longer lists these ids as missing.
  A `regressed` crit/high becomes a package-1 ticket at priority 90 (Step 21.4).

### Step 21.3 - Read

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

### Step 21.4 - Triage

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

### Step 21.5 - Close

- [x] Inline fixes land with their test run cited (command, exit code, last line); a changed Go file
      is `gofmt`-clean.
- [x] Flip this phase's row in `INDEX.md` to ✅ with the severity counts, and bump `Phases: X/N` in
      `INDEX.md` and the `k/N slices` in the ticket's status line and its `RELEASE_QUEUE.md` line.
- **Verification:** `./scripts/audit-slices.ps1 -Summary` shows `S21` as closed.

## Done criteria

- Every step `[x]`, each with its verification met.
- The findings table below is filled, and every crit/high in it has a ticket or an inline fix.

## Handoff notes

Prescan (26605b9 + working tree of 2026-09-26; read only, no winget/wingetcreate run):
- No Go packages, no extension modules. Invoke-ScriptAnalyzer not installed; not PowerShell.
- `python yaml.safe_load` over winget/*.yaml - exit 0 - `yaml ok 15`; one PackageIdentifier
  (SerZhyAle.DocHtmlTranslate), one PackageVersion (26.0912.2026, matches YY.MMDD.HHmm), one ManifestVersion (1.12.0).
- `git ls-files --eol winget` - all 15 top-level files i/lf w/lf, no BOM.
- InstallerSha256 8AB7..2507 equals the CI sidecar `temp/imgcheck/rel/doc-html-translate-26.0912.2026-windows-x64.zip.sha256` (nothing downloaded).
- `./scripts/audit-slices.ps1 -Summary` - exit 1 - campaign-level FAIL (21 slices open, FINDINGS not yet written) - expected.

Re-check: none for this slice.

Files read in full: [x] all 15 winget/*.yaml (version, installer, 13 locales). Context: release.yml:41-56,153-190, DEV/RELEASE.md:75-103.

Findings: med 1 (S21-1 "English OCR bundled" claim false for the winget zip), low 2 (S21-2 listing omits comics/image input;
S21-3 winget/manifests/ archive breaks the documented `--manifest winget` gate). crit 0, high 0.
