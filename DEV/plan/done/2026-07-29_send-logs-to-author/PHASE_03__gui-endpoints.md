# Phase 03 - GUI endpoints

**Strategic spec:** [`../2026-07-29_send-logs-to-author.md`](../2026-07-29_send-logs-to-author.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 02
**Steps done:** 4 / 4

## Objective

The GUI's local service can build a report archive, reveal it in Explorer with the file selected,
open it for inspection, and clear the log store - each as its own endpoint, with no UI yet.

## Prerequisites

- [ ] Phase 02 is ✅ Done.
- [ ] `cmd/doc-html-ui/main.go` is committed (1046 lines - the tree is the backup for a file this
      size; do not edit it with uncommitted changes present).

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `cmd/doc-html-ui/report.go` | New | ≤ 200 |
| `cmd/doc-html-ui/main.go` | Modified (1046) | ≤ 1120 |
| `cmd/doc-html-ui/report_test.go` | New | ≤ 200 |

## Steps

### Step 03.1 - Add the report endpoint

**Files:** `cmd/doc-html-ui/report.go`
**Depends on:** - start of phase

**Prompt for developer:**
> Create `cmd/doc-html-ui/report.go` with `func handleReport(w http.ResponseWriter, r
> *http.Request)`: POST only; read the saved GUI state from `settingsPath()` (ignore a read error
> and pass nil), call `report.Build` with `AppVersion: Version`, `Packaged: isPackaged()`, that
> settings blob and `time.Now()`, and answer
> `{"ok":true,"path":"..","dropped":N,"bytes":N}` or `{"ok":false,"error":".."}`. Follow the
> answer shape and the method-check style of `handleOpenOutput` in `main.go`.

**Verification:**
- File `cmd/doc-html-ui/report.go` exists and declares `func handleReport(w http.ResponseWriter, r *http.Request)` exactly once.
- `report.Build(` appears exactly once in the file.
- `"dropped"` appears in the response encoding.

**Status:** `[x]` done - 2026-08-11. `handleReport` and `report.Build(` match once, `"dropped"` is
in the encoded answer; `go vet ./cmd/doc-html-ui` exit 0.

---

### Step 03.2 - Add reveal, open and clear

**Files:** `cmd/doc-html-ui/report.go`
**Depends on:** Step 03.1

**Prompt for developer:**
> Add `func handleReportReveal(w http.ResponseWriter, r *http.Request)` (POST, body
> `{"path":".."}`): refuse a path that is not inside `report.Dir()` with
> `{"ok":false,"error":"outside the report folder"}`, otherwise run `explorer.exe /select,<path>`
> and answer `{"ok":true}`. Add `func handleReportOpen` opening the archive itself with the
> existing `openTarget` helper, under the same containment check. Add `func handleLogsClear` (POST)
> calling `report.ClearLogs()` and answering `{"ok":true}` or the error. Explorer's exit code is
> not a failure signal - it returns non-zero on success; ignore it and report `ok` when the process
> started.

**Verification:**
- `handleReportReveal`, `handleReportOpen` and `handleLogsClear` each match exactly once as
  declarations.
- `explorer.exe` and `/select,` both appear.
- `report.Dir()` appears in the containment check of each of the two path-taking handlers.

**Status:** `[x]` done - 2026-08-11. The three handlers match once each; `explorer.exe /select,` is
started (not waited on) through the `revealInExplorer` indirection, so the guard is testable
without opening windows. **Deviation from the letter of the third predicate:** the containment
check is not written twice - both path-taking handlers go through one `pathRequest` helper that
calls `insideReportDir`, where `report.Dir()` lives. One gate is the point of the rule; two copies
would be the way to drift out of it.

---

### Step 03.3 - Register the routes and report the log-store state

**Files:** `cmd/doc-html-ui/main.go`
**Depends on:** Step 03.2

**Prompt for developer:**
> Register `/api/report`, `/api/report-reveal`, `/api/report-open` and `/api/logs-clear` in the
> `mux` block in `main`, keeping the existing one-line-per-route style. Extend `handleEnv`'s JSON
> with `"logs"`: the number of stored run logs and their total size, read from
> `report.LogsDir()`, plus `"author"`: the constant `sza@ukr.net` so the page does not hardcode the
> address a second time.

**Verification:**
- `"/api/report"`, `"/api/report-reveal"`, `"/api/report-open"` and `"/api/logs-clear"` each
  appear exactly once in `cmd/doc-html-ui/main.go`.
- `"author"` and `sza@ukr.net` appear in `handleEnv`.
- `go build ./...` exits 0.

**Status:** `[x]` done - 2026-08-11. The four routes match once each; `handleEnv` now answers
`logs: {count, bytes}` (via a `logStoreSize` helper - a missing store is empty, not an error) and
`author`, whose value is the new `authorEmail` constant beside the handler rather than a second
literal. `go build ./...` exit 0.

---

### Step 03.4 - Prove the endpoints

**Files:** `cmd/doc-html-ui/report_test.go`
**Depends on:** Step 03.3

**Prompt for developer:**
> With `LOCALAPPDATA` at `t.TempDir()`, write `TestHandleReportWritesArchive` (POST via
> `httptest`, assert `ok` true and the returned path exists), `TestHandleReportRejectsGET`
> (405), `TestHandleReportRevealRefusesPathOutsideReportDir` (a path in `t.TempDir()` outside
> `report.Dir()` answers `ok:false` and no process is started) and
> `TestHandleLogsClearEmptiesTheStore`. Follow the `httptest` style already used by
> `TestSettingsRoundTrip` in `main_test.go`.

**Verification:**
- `go test ./cmd/doc-html-ui` exits 0.
- The four test names above each match exactly once.

**Status:** `[x]` done - 2026-08-11. `go test ./cmd/doc-html-ui` exit 0; the four names match once
each. The reveal test also asserts the positive case - a path the app itself wrote is accepted -
so the guard cannot pass by refusing everything.

## Phase done criteria

- [x] Every `Step 03.*` is `[x] done`.
- [x] `go test ./cmd/doc-html-ui` exits 0 (targeted test).
- [x] `go build ./...` exits 0.
- [x] Grep for `TODO(phase-03)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Handoff notes

Established: the four endpoints and the `logs`/`author` fields on `/api/env`. Phase 04 consumes
them and must not add a fifth path-taking endpoint without the same `report.Dir()` containment
check. Phase 05's parity allow-list entry names `/api/report` from this phase.

## Rollback plan

Revert phase commit(s); the route registrations are the only edit inside `main.go`.
