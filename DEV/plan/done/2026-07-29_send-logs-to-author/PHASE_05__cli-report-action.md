# Phase 05 - CLI report action

**Strategic spec:** [`../2026-07-29_send-logs-to-author.md`](../2026-07-29_send-logs-to-author.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 02, Phase 03
**Steps done:** 4 / 4

## Objective

`doc-html-translate -report` builds the same archive without the GUI, prints where it landed in the
interface language, and keeps the ui-cli parity guard green.

## Prerequisites

- [ ] Phase 02 is ✅ Done (`report.Build`).
- [ ] Phase 03 is ✅ Done (`/api/report` exists, so the parity allow-list entry is truthful).

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `internal/config/flags.go` | Modified (152) | ≤ 180 |
| `internal/app/app.go` | Modified (190) | ≤ 240 |
| `internal/i18n/i18n_cli.go` | Modified (139) | ≤ 220 |
| `tests/ui_cli_parity_test.go` | Modified (46) | ≤ 60 |
| `internal/config/flags_test.go` | New or Modified | ≤ 120 |

## Steps

### Step 05.1 - Add the flag

**Files:** `internal/config/flags.go`
**Depends on:** - start of phase

**Prompt for developer:**
> Add `Report bool // -report: pack the recent run logs into an archive and exit` to `Config`,
> register `report := fs.Bool("report", false, "pack the recent run logs plus an environment
> summary into an archive for the author, then exit")`, assign it in the `cfg` literal, and add
> `cfg.Report` to the condition at the end of `ParseArgs` that decides an input file is not
> required - alongside `cfg.OCRList` and `cfg.OCRDownload`.

**Verification:**
- `fs.Bool("report"` matches exactly once.
- `Report ` appears in the `Config` struct and `Report:` in the `cfg` literal.
- `cfg.Report` appears in the `if !cfg.Register && ..` input-file condition.

**Status:** `[x]` done - 2026-08-11. All three predicates hold; `go build ./cmd/... ./internal/...`
exit 0.

---

### Step 05.2 - Handle it in the app

**Files:** `internal/app/app.go`
**Depends on:** Step 05.1

**Prompt for developer:**
> In `Run`, next to the `OCRList` branch, add: when `a.cfg.Report` is set, call `report.Build` with
> `AppVersion: logging.AppVersion`, `Packaged: false`, no settings blob and `time.Now()`; on
> success print the localized "Report written to:" line plus the path and, when logs were dropped,
> the localized dropped-count line; return `0, nil`. On failure return `1, err`. Print the path
> raw (not through `i18n.S`) so it can be copied. The CLI does **not** open a mail program - that
> is the GUI's job.

**Verification:**
- `a.cfg.Report` matches exactly once in `internal/app/app.go`.
- `report.Build(` matches exactly once and the file imports `doc-html-translate/internal/report`.
- Neither `mailto` nor `explorer` appears anywhere in `internal/app/app.go`.

**Status:** `[x]` done - 2026-08-11. `a.cfg.Report` and `report.Build(` match once each, the
package is imported, and neither `mailto` nor `explorer` occurs in the file.

---

### Step 05.3 - Localize the two console lines

**Files:** `internal/i18n/i18n_cli.go`
**Depends on:** Step 05.2

**Prompt for developer:**
> Add two `Add(..)` registrations for the strings printed in step 05.2 - "Report written to:" and
> the dropped-logs notice with a `%d` verb - each with exactly twelve translations in
> `Codes[1:]` order (`ru uk de it es fr pt ar hi bn ur zh`). A wrong count panics at init, which
> is the intended guard.

**Verification:**
- `go test ./internal/i18n` exits 0 (`TestEveryKeyHasTwelveNonEmptyTranslations` covers the new
  keys).
- `Report written to:` matches exactly once in `internal/i18n/i18n_cli.go`.
- `go test ./tests -run TestTypography` exits 0.

**Status:** `[x]` done - 2026-08-11. Both `Add(..)` registrations carry twelve translations (a
wrong count panics at init, and `go test ./internal/i18n` exit 0 proves it did not);
`Report written to:` matches once in the dictionary file; typography gate exit 0.

---

### Step 05.4 - Keep the parity guard truthful and pin the flag

**Files:** `tests/ui_cli_parity_test.go`, `internal/config/flags_test.go`
**Depends on:** Step 05.3

**Prompt for developer:**
> Add `"report": "About section \"Send logs to the author\" button + /api/report"` to the
> `guiNative` map in `tests/ui_cli_parity_test.go` - the GUI serves this natively rather than
> forwarding the flag. Then add `TestParseArgsReportNeedsNoInputFile` asserting `-report` parses
> with no positional argument and yields `Report == true`.

**Verification:**
- `"report":` matches exactly once in `tests/ui_cli_parity_test.go`.
- `go test ./tests -run TestParityGUIExposesEveryCLIFlag` exits 0.
- `go test ./internal/config` exits 0 and `TestParseArgsReportNeedsNoInputFile` matches exactly
  once.

**Status:** `[x]` done - 2026-08-11. `"report":` matches once in the allow-list, the parity test
and `./internal/config` both exit 0.

## Phase done criteria

- [x] Every `Step 05.*` is `[x] done`.
- [x] `go test ./internal/config ./internal/i18n ./tests -run 'TestParseArgs|TestEveryKey|TestParity|TestTypography'` exits 0.
- [x] `go build ./cmd/... ./internal/...` exits 0. **Not `go build ./...`:** an untracked
      `tools/ocrlab/` appeared in the working tree at 14:28-14:34 today - another ticket's
      in-flight work, unrelated to this one - and it does not compile (`undefined: cmdSynth`,
      `cmdSeed`; an unused import). It is left exactly as found; the build was scoped to the two
      trees this ticket touches.
- [x] Grep for `TODO(phase-05)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

**Measured, not asserted:** a freshly built CLI run against a temp `%LOCALAPPDATA%` printed
`Report written to:` + the path (and `Отчёт записан в:` under `-ui-lang ru`), exit 0 both times,
and the archive it wrote contained `environment.txt` plus `logs/run-20260811-143516.log` - so
phase 01's store and this action meet end to end.

## Handoff notes

Established: `-report` as a no-input-file action, its two localized lines, and the parity
allow-list entry. Phase 07 documents the flag in the README flag table and the docs trio.

## Rollback plan

Revert phase commit(s). Removing the flag also requires removing the `guiNative` entry, or the
parity test passes for a flag that no longer exists (harmless but stale).
