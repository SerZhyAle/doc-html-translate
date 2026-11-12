# Pointer: DIAGNOSTIC-REPORT

- **Id:** `DIAGNOSTIC-REPORT`
- **Version:** 0.15 draft
- **Home:** the shared contracts catalog, `diagnostic-report/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** producer, `partial` - `internal/report` generates diagnostic zip archives and environment summaries, and the extension's `diagnostics.js` the JSON envelope; read at 0.15 on 2026-10-29 (ticket E0126), the owed conformance of the 2026-10-20 proposal done on 2026-10-30 (ticket E0122, Q3)
- **Wire carrier:** `schema_version` in `environment.txt` (`KEY=value` lines) or `schemaVersion` in the JSON envelope + sanitized session log bundle

How diagnostic bundles and support packages are assembled and sanitized:
- **Archive format:** `.zip` named `doc-html-translate-logs-<version>-yyyyMMdd-HHmmss.zip` (section 8 A; the version
  segment is left out for an unstamped build, and a name taken in the same second moves to the next second, so a
  report is never overwritten), containing `environment.txt`, `report.txt`, `settings.json`, and recent log files under `LogsDir()`.
  `settings.json` is the settings copy without the output folder and the recent-documents list (`output`, `recent`).
- **Bounded size and compaction:** archive capped at 15 MB (`MaxArchiveBytes = 15 << 20`); logs are redacted
  newest first and kept while they fit, a whole log that does not fit is left out (never cut), and a log over the cap
  is skipped before it is read into memory. A log left out keeps its entry name and holds only the 0.15 omission
  marker `[Diag] LOG OMITTED | reason=archive_cap | source_bytes=<n>` (the size as read; implemented 2026-10-29,
  ticket 122, ahead of the full 0.15 re-read of ticket 126), and `environment.txt` ends with the additive line
  `logs_left_out=N` (and `report.txt` the line `logs left out: N`). A run log is held to 5 MiB by compaction: its first
  1 MiB is written as it comes, the rest is held in memory by whole lines, and when the run ends the last ~4 MiB follow
  one marker `[Diag] LOG COMPACTED | dropped_middle_bytes=<n> | kept_head_bytes=<n> | kept_tail_bytes=<n>` (written when
  the run ends; a process killed from outside leaves the log at its head; ticket E0122, Q1); a log that fits is untouched.
  The store keeps the last 20 run logs (20 MiB total).
- **Environment summary:** `environment.txt` provides greppable `KEY=value` lines opening with `schema_version=1`
  (`app_version`, `edition`, `os`, `ui_language`, `ocr`, `ocr_languages`, `ocr_data_dir`, `ollama_model`, and
  `logs_left_out` when due); `report.txt` is the same facts as labelled lines for a person (`version`, `edition`,
  `platform`, `interface language`, `ocr`, .., `logs left out`), the labels shared with the extension. The extension's
  clipboard and saved file carry the JSON envelope (`schemaVersion`, `app`, `environment`, `options`, `lastRun`) with the
  same readable lines in `report`; the two schema versions are pinned equal by `TestParityReportFields`.
- **Redaction invariants:** `Redact` / `RedactBytes` strips API keys, tokens, passwords, signed-CDN parameters,
  credential-in-path and URL userinfo, personal paths (`%LOCALAPPDATA%`, `%USERPROFILE%`), and the names and titles
  of the user's documents (a file name becomes `<document>` keeping its extension, a `Title:` line `<document title>`;
  `internal/report/names.go`), before any size decision (the catalog's structured-value vectors pass; no exchange login, pairing or account file exists here, so
  section 10 has nothing to act on).
- **User consent:** diagnostic bundle generation is strictly user-initiated (UI button / report command); zero
  silent telemetry.

**Deviations, in the catalog registry (section 3).** Temporary, until 2026-12-31: the retention of 20 run logs under
20 MiB (the contract's default keeps 10 session logs; a size ceiling may be lower, a count is a default). The run-log
marker row is closed by the head-and-tail compaction above (ticket E0122, Q1), and the three variants the 2026-10-20
proposal asked for are closed by conforming (Q3): the extension's clipboard and file carry the JSON envelope (it
still has no zip archive, ADR-4 in [`../PARITY.md`](../PARITY.md)), `environment.txt` is `KEY=value` with
`schema_version` and the labelled block is its own entry, and document names and titles stay out of logs and the
settings copy. The path placeholders are `%USERPROFILE%` and `%LOCALAPPDATA%`, accepted by 0.15. The old archive-name
exception is closed.

**Conformance.** Package `internal/report` unit tests (`archive_test.go`, `redact_test.go`, `store_test.go`),
`TestRedactStructuredValueVectors`, `TestRedactCatalogVectorFileIfAvailable` (reads the catalog vector) and
`TestParityReportFields` in `tests/parity_test.go` for the labels and the schema version the two editions share,
`names_test.go` for the document names and titles, and `extension/test/diagnostics.test.mjs` for the envelope.

**Re-read 2026-10-29 (ticket E0126) at 0.15.** The placeholder reading is accepted (any placeholder without the profile name), so that variant is closed; `reason=archive_cap` is in the open reason set; an absent family may be omitted. Items 1 and 2 of this product's proposal were not adopted; the owner chose to conform (ticket E0122, Q3, 2026-10-30): see Environment summary and Redaction invariants above.
