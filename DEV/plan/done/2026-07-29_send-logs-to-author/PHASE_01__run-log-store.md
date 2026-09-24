# Phase 01 - Run-log store

**Strategic spec:** [`../2026-07-29_send-logs-to-author.md`](../2026-07-29_send-logs-to-author.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** none - foundation phase
**Steps done:** 5 / 5

## Objective

Every conversion run leaves a log file in a bounded per-user store, without changing what the
console or the GUI log pane show and without any way to fail a conversion.

## Prerequisites

- [ ] Working tree clean or on a feature branch.
- [ ] `go build ./...` green before starting.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `internal/report/store.go` | New | ≤ 200 |
| `internal/report/store_test.go` | New | ≤ 200 |
| `internal/logging/log.go` | Modified (114) | ≤ 200 |
| `internal/logging/log_test.go` | New | ≤ 120 |
| `cmd/doc-html-translate/main.go` | Modified (55) | ≤ 100 |

## Steps

### Step 01.1 - Create the per-user store locations

**Files:** `internal/report/store.go`
**Depends on:** - start of phase

**Prompt for developer:**
> Create package `report` with `Dir() string` returning `%LOCALAPPDATA%\doc-html-translate`
> (falling back to `os.TempDir()/doc-html-translate` when `LOCALAPPDATA` is unset), `LogsDir()
> string` returning `Dir()/logs`, and `RunLogPath(at time.Time) string` returning
> `LogsDir()/run-<yyyymmdd-hhmmss>.log`. Take the timestamp as a parameter rather than reading
> the clock inside, so the tests are deterministic. Mirror the existing choice of
> `%LOCALAPPDATA%` documented in `cmd/doc-html-ui/main.go` `settingsPath` - the packaged MSIX
> install directory is read-only. Do not refactor the three existing copies of that path
> computation; they stay as they are.

**Verification:**
- File `internal/report/store.go` exists and declares `package report`.
- `func Dir() string`, `func LogsDir() string` and `func RunLogPath(at time.Time) string` each
  match exactly once as declarations.
- `"logs"` appears in `LogsDir`.

**Status:** `[x]` done - 2026-08-11. `internal/report/store.go` created; all three declarations
match once; `go vet ./internal/report` exit 0.

---

### Step 01.2 - Bound the store

**Files:** `internal/report/store.go`
**Depends on:** Step 01.1

**Prompt for developer:**
> Add `const MaxLogFiles = 20`, `const MaxLogBytes = 20 << 20`, `func Trim() (removed int, err
> error)` and `func ClearLogs() error` to package `report`. `Trim` lists `LogsDir()`, sorts by
> name (the timestamped name sorts chronologically), and deletes oldest-first until both bounds
> hold. `ClearLogs` removes every file in `LogsDir()` and leaves the directory in place. A
> missing directory is not an error for either.

**Verification:**
- `MaxLogFiles = 20` and `MaxLogBytes` declarations match exactly once each.
- `func Trim() (removed int, err error)` and `func ClearLogs() error` each match exactly once.
- `os.IsNotExist` or `errors.Is(err, fs.ErrNotExist)` appears in the file.

**Status:** `[x]` done - 2026-08-11. Both constants and both functions match once;
`errors.Is(.., fs.ErrNotExist)` guards the missing directory; `go vet` exit 0.

---

### Step 01.3 - Tee the log channel to a file

**Files:** `internal/logging/log.go`
**Depends on:** - start of phase

**Prompt for developer:**
> Add a run-log sink to package `logging`: an unexported `runLog io.Writer` guarded by a mutex,
> `func StartRunLog(w io.Writer)` to install it and `func StopRunLog()` to drop it. Route
> `Printf`, `Println`, `Errorf` and `Progress` through a helper that writes the formatted line to
> its current destination (stdout or stderr, unchanged) **and** to `runLog` when one is set.
> Writes to `runLog` ignore their error. In the `runLog` copy of `Progress`, always emit a
> trailing newline and never a carriage return, so the file holds whole lines even in an
> interactive console. Do not touch `stdoutIsTerminal` or `StdoutIsTerminal`.

**Verification:**
- `func StartRunLog(w io.Writer)` and `func StopRunLog()` each match exactly once as
  declarations.
- `sync.Mutex` or `sync.RWMutex` appears in `internal/logging/log.go`.
- `\r` no longer appears inside the branch that writes to `runLog` (grep the file for `runLog`
  and read the four call sites).

**Status:** `[x]` done - 2026-08-11. `StartRunLog`/`StopRunLog` match once, `sync.Mutex` present,
and the carriage return only ever reaches the console argument of `emit` - the run-log copy is
built by `strings.TrimRight(line, " \n") + "\n"`. `go build ./...` exit 0.

---

### Step 01.4 - Prove the tee and the bound

**Files:** `internal/logging/log_test.go`, `internal/report/store_test.go`
**Depends on:** Step 01.2, Step 01.3

**Prompt for developer:**
> Write `TestStartRunLogTeesEveryLevel` in `internal/logging`: install a `bytes.Buffer` sink,
> call `Printf`, `Println`, `Errorf` and `Progress`, and assert all four texts are in the buffer
> and that it contains no `\r`. Write `TestStopRunLogStopsWriting` asserting nothing is appended
> after `StopRunLog`. In `internal/report`, write `TestRunLogPathIsTimestamped`,
> `TestTrimKeepsNewestWithinBounds` (create 25 dated files, assert 20 remain and the removed ones
> are the oldest) and `TestClearLogsEmptiesDirectory`; point `LOCALAPPDATA` at `t.TempDir()` with
> `t.Setenv`. Also write `TestRunLogWriteErrorIsIgnored` in `internal/logging`: install a sink whose
> `Write` always fails, call all four printers, and assert nothing panics and the normal output still
> happened - this pins the "logging can never break a run" rule from strategic §3.2.

**Verification:**
- `go test ./internal/logging ./internal/report` exits 0.
- The six test function names above each match exactly once.

**Status:** `[x]` done - 2026-08-11. `go test ./internal/logging ./internal/report` exit 0; the six
names match once each. Two extra guards added while here: `TestTrimHonoursTheByteBound` and
`TestTrimOnMissingStoreIsNotAnError`.

---

### Step 01.5 - Open the run log at start-up

**Files:** `cmd/doc-html-translate/main.go`
**Depends on:** Step 01.2, Step 01.3

**Prompt for developer:**
> In `main`, right after `logging.AppVersion = Version`, call `report.Trim()`, create
> `report.LogsDir()` with `os.MkdirAll`, open `report.RunLogPath(time.Now())` for append and
> hand it to `logging.StartRunLog`, deferring `logging.StopRunLog` and the file's `Close`.
> **Every error on this path is swallowed** - a store that cannot be written must leave the run
> untouched and print nothing about it. Place the call before `logging.Printf("doc-html-translate
> %s\n", Version)` so the version line is the log's first line.

**Verification:**
- `report.Trim()`, `logging.StartRunLog(` and `logging.StopRunLog()` each appear in
  `cmd/doc-html-translate/main.go`.
- No `os.Exit`, `return`, `fmt.Fprint*` or `logging.Errorf` occurs inside the new block (read it:
  the block has no error reporting at all).
- `go build ./...` exits 0.

**Status:** `[x]` done - 2026-08-11. The three calls are present, the block contains no error
reporting of any kind, `go build ./...` exit 0.

## Phase done criteria

- [x] Every `Step 01.*` is `[x] done`.
- [x] `go test ./internal/logging ./internal/report` exits 0 (targeted test - this phase changes
      logic, not packaging).
- [x] `go build ./...` exits 0.
- [x] Grep for `TODO(phase-01)` returns zero hits (only this file's own criterion line matches).
- [x] Changelog entry added for every file in "Files touched".

## Handoff notes

Established: `report.Dir/LogsDir/RunLogPath/Trim/ClearLogs` and the `logging` tee. Phase 02 reads
`LogsDir()` and must not re-derive the path. The swallow-all-errors rule on the start-up path is
an invariant, not a style choice - phase 05 must not "improve" it into a warning.

## Rollback plan

Revert phase commit(s). Nothing outside the new package and the two touched files depends on it.
