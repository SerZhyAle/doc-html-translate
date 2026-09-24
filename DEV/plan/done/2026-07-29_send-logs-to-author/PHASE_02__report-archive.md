# Phase 02 - Report archive

**Strategic spec:** [`../2026-07-29_send-logs-to-author.md`](../2026-07-29_send-logs-to-author.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 01
**Steps done:** 5 / 5

## Objective

One call produces one zip archive - environment summary, redacted settings, recent run logs -
capped in size, with a test that a secret cannot reach it.

## Prerequisites

- [ ] Phase 01 is ✅ Done.
- [ ] Strategic §6 items 1-3 are Resolved (they are - owner decision 2026-07-29).

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `internal/report/redact.go` | New | ≤ 160 |
| `internal/report/environment.go` | New | ≤ 180 |
| `internal/report/archive.go` | New | ≤ 240 |
| `internal/report/redact_test.go` | New | ≤ 200 |
| `internal/report/archive_test.go` | New | ≤ 220 |

## Steps

### Step 02.1 - Write the redaction rules

**Files:** `internal/report/redact.go`
**Depends on:** - start of phase

**Prompt for developer:**
> Add `func Redact(s string) string` to package `report`, applying an ordered, package-level
> `redactions` slice of `struct{ re *regexp.Regexp; with string }` so a future rule is one entry:
> replace the current user's profile directory (from `os.UserHomeDir`, case-insensitive) with
> `%USERPROFILE%`; replace `%LOCALAPPDATA%`'s value likewise; replace any run of 20 or more
> characters from `[A-Za-z0-9_\-]` that follows `key`, `token`, `secret` or `password`
> (case-insensitive, optional `=`/`:`/whitespace between) with `<redacted>`; replace a bare
> Google-API-key-shaped token (`AIza` followed by 35 URL-safe characters) with `<redacted>`
> wherever it appears. Keep document **file names** intact - only directory components above the
> file are reduced. Export `func RedactBytes(b []byte) []byte` as a thin wrapper.

**Verification:**
- `func Redact(s string) string` and `func RedactBytes(b []byte) []byte` each match exactly once.
- `AIza` and `%USERPROFILE%` both appear in the file.
- `var redactions = []` matches exactly once.

**Status:** `[x]` done - 2026-08-11. All five predicates match once each; `go vet` exit 0. Note:
the two path rules are built per call (`pathRedactions`) because their locations come from the
environment, and `%LOCALAPPDATA%` is applied before `%USERPROFILE%` since it sits inside it.

---

### Step 02.2 - Generate the environment summary

**Files:** `internal/report/environment.go`
**Depends on:** - start of phase

**Prompt for developer:**
> Add `type Env struct { AppVersion, Edition, OS, UILang, Tesseract, OllamaModel string; OCRLangs
> []string }` and `func Environment(appVersion string, packaged bool) string` returning a plain
> English `key: value` block, one field per line, ending with a newline. Fill: app version; edition
> (`packaged (MSIX)` or `portable`); Windows version from `runtime.GOOS` plus `os.Getenv("OS")` and
> the build number if cheaply available; resolved interface language via
> `i18n.Resolve("", "", syslocale.Lang())`; tesseract presence and the installed OCR languages via
> `ocr.Installed()` and `ocr.DataDir()`; the Ollama default model name. **The summary is always
> English** regardless of interface language. Pass the result through `Redact` before returning.

**Verification:**
- `func Environment(appVersion string, packaged bool) string` matches exactly once.
- The file imports `doc-html-translate/internal/ocr` and `doc-html-translate/internal/i18n`.
- `Redact(` appears in the return path.

**Status:** `[x]` done - 2026-08-11. All three predicates hold; `go vet` exit 0. The Windows build
number is not read - no cheap source exists without a registry call and a `_windows.go` split,
so the platform line carries `runtime.GOOS/GOARCH` plus `%OS%`. The Ollama default model is a
local constant mirroring `internal/translator` (the repo already keeps three copies of it).

---

### Step 02.3 - Build the archive

**Files:** `internal/report/archive.go`
**Depends on:** Step 02.1, Step 02.2

**Prompt for developer:**
> Add `const MaxArchiveBytes = 15 << 20`, `type BuildOptions struct { AppVersion string; Packaged
> bool; SettingsJSON []byte; At time.Time }` and `func Build(opts BuildOptions) (path string,
> droppedLogs int, err error)`. Write
> `Dir()/reports/report_doc-html-translate_<version>_<yyyymmdd-hhmm>.zip` using `archive/zip`:
> `environment.txt` from `Environment`, `settings.json` from `RedactBytes(opts.SettingsJSON)` when
> non-empty, then the log files from `LogsDir()` newest-first under `logs/`, each passed through
> `RedactBytes`, stopping before the running total would exceed `MaxArchiveBytes` and counting the
> skipped ones into `droppedLogs`. Create the `reports` directory as needed and return the absolute
> path. Never include anything from `Dir()` other than the log files - in particular never
> `google_api.key`.

**Verification:**
- `func Build(opts BuildOptions) (path string, droppedLogs int, err error)` matches exactly once.
- `MaxArchiveBytes` declaration matches exactly once.
- `"environment.txt"`, `"settings.json"` and `"logs/"` each appear.
- `google_api.key` does not appear as an included name anywhere in the file.

**Status:** `[x]` done - 2026-08-11. `Build` and `MaxArchiveBytes` match once, the three entry
names are present, `google_api.key` appears nowhere in the file (the archive only ever reads
`LogsDir()`); `go vet` exit 0.

---

### Step 02.4 - Prove no secret can reach the archive

**Files:** `internal/report/redact_test.go`
**Depends on:** Step 02.1

**Prompt for developer:**
> Write `TestRedactRemovesGoogleAPIKeyShapes` (a bare `AIza..` token, `key=..`, `token: ..`,
> `Password ..` - each must not survive), `TestRedactShortensUserProfilePaths` (a path under the
> home directory becomes `%USERPROFILE%\..` while the file's own name survives verbatim) and
> `TestRedactKeepsDocumentFileNames` (a Cyrillic document name survives). Set `USERPROFILE` and
> `LOCALAPPDATA` with `t.Setenv` so the test is machine-independent.

**Verification:**
- `go test ./internal/report -run TestRedact` exits 0.
- The three test names above each match exactly once.

**Status:** `[x]` done - 2026-08-11. `go test ./internal/report -run TestRedact` exit 0, four tests
pass; the fourth (`TestRedactPrefersTheMoreSpecificLocation`) pins the `%LOCALAPPDATA%`-before-
`%USERPROFILE%` order.

---

### Step 02.5 - Prove the archive's shape and cap

**Files:** `internal/report/archive_test.go`
**Depends on:** Step 02.3

**Prompt for developer:**
> With `LOCALAPPDATA` pointed at `t.TempDir()`, write `TestBuildProducesReadableArchive` (assert
> the returned path exists, `zip.OpenReader` succeeds, and the entry set is exactly
> `environment.txt`, `settings.json` and at least one `logs/` entry),
> `TestBuildDropsOldestLogsOverCap` (write log files that together exceed `MaxArchiveBytes`,
> assert `droppedLogs > 0` and that the newest log is present while the oldest is not),
> `TestBuildRedactsSettings` (feed a `SettingsJSON` containing an `AIza..` token, assert the
> archived `settings.json` does not contain it) and `TestBuildNeverArchivesTheAPIKeyFile` (create
> `google_api.key` in `Dir()`, assert no archive entry mentions it).

**Verification:**
- `go test ./internal/report` exits 0.
- The four test names above each match exactly once.

**Status:** `[x]` done - 2026-08-11. `go test ./internal/report` exit 0; the four names match once
each. The cap test also asserts the file on disk is under `MaxArchiveBytes`.

## Phase done criteria

- [x] Every `Step 02.*` is `[x] done`.
- [x] `go test ./internal/report` exits 0 (targeted test - changed logic, no packaging impact).
- [x] Grep for `TODO(phase-02)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Handoff notes

Established: `report.Build`, `report.Redact`, `report.Environment`, `report.MaxArchiveBytes`.
Phases 03 and 05 both call `Build` and neither may compose a zip of its own. The redaction gate
lives inside `Build`; a caller must never write an archive by another route.

## Rollback plan

Revert phase commit(s). Phase 01 stands alone without this phase.
