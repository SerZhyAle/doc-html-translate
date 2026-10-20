# Pointer: DIAGNOSTIC-REPORT

- **Id:** `DIAGNOSTIC-REPORT`
- **Version:** 0.14 draft
- **Home:** the shared contracts catalog, `diagnostic-report/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** producer, `partial` - `internal/report` generates diagnostic zip archives and environment summaries; read at 0.14 on 2026-10-20 (ticket 120)
- **Wire carrier:** `schemaVersion` in metadata or `environment.txt` (`key: value` lines) + sanitized session log bundle

How diagnostic bundles and support packages are assembled and sanitized:
- **Archive format:** `.zip` named `doc-html-translate-logs-<version>-yyyyMMdd-HHmmss.zip` (section 8 A; the version
  segment is left out for an unstamped build, and a name taken in the same second moves to the next second, so a
  report is never overwritten), containing `environment.txt`, `settings.json`, and recent log files under `LogsDir()`.
- **Bounded size and compaction:** archive capped at 15 MB (`MaxArchiveBytes = 15 << 20`); logs are redacted
  newest first and kept while they fit, a whole log that does not fit is left out (never cut), and a log over the cap
  is skipped before it is read into memory. When logs were left out, `environment.txt` ends with the additive line
  `logs left out: N`. A run log stops growing at 5 MiB with a marker line; the store keeps the last 20 run logs
  (20 MiB total).
- **Environment summary:** `environment.txt` provides plain `key: value` fields (`version`, `edition`, `platform`,
  `interface language`, `ocr`, `ocr languages`, `ocr data dir`, `ollama model`, and `logs left out` when due).
- **Redaction invariants:** `Redact` / `RedactBytes` strips API keys, tokens, passwords, signed-CDN parameters,
  credential-in-path and URL userinfo, and personal paths (`%LOCALAPPDATA%`, `%USERPROFILE%`), before any size
  decision (the catalog's structured-value vectors pass; no exchange login, pairing or account file exists here, so
  section 10 has nothing to act on).
- **User consent:** diagnostic bundle generation is strictly user-initiated (UI button / report command); zero
  silent telemetry.

**Deviations, in the catalog registry (section 3).** Temporary, until 2026-12-31: the marker line of a run log cut
at 5 MiB (`[run log cut at N MiB: ..]`) differs from the contract's spelling (ticket 122, item 6), and the
retention of 20 run logs under 20 MiB. The old archive-name exception is closed. Permanent variants, with the
design record [`../PARITY.md`](../PARITY.md) "Contract variants (ticket 120)": the extension has no archive and
no JSON envelope (clipboard text and a saved text file, ADR-4); the environment text is plain `key: value` with
spaced labels and no `schema_version` (absent means 1); document file names stay in run logs and recent paths in
`settings.json`; the path placeholders are `%USERPROFILE%` and `%LOCALAPPDATA%`. Proposals for all four are in the
`diagnostic-report` proposal file of 2026-10-20.

**Conformance.** Package `internal/report` unit tests (`archive_test.go`, `redact_test.go`, `store_test.go`),
`TestRedactStructuredValueVectors`, `TestRedactCatalogVectorFileIfAvailable` (reads the catalog vector) and
`TestParityReportFields` in `tests/parity_test.go` for the labels the two editions share.
