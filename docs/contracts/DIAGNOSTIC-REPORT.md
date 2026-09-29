# Pointer: DIAGNOSTIC-REPORT

- **Id:** `DIAGNOSTIC-REPORT`
- **Version:** 0.10 draft
- **Home:** the shared contracts catalog, `diagnostic-report/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** producer - `internal/report` generates diagnostic zip archives and environment summaries
- **Wire carrier:** `schemaVersion` in metadata or `environment.txt` (`key: value` lines) + sanitized session log bundle

How diagnostic bundles and support packages are assembled and sanitized:
- **Archive format:** `.zip` named `report_doc-html-translate_<version>_<timestamp>.zip` containing `environment.txt`, `settings.json`, and recent log files under `LogsDir()`.
- **Bounded size & compaction:** archive capped at 15 MB (`MaxArchiveBytes = 15 << 20`); newest logs included first, older logs dropped when cap is reached. A run log stops growing at 5 MiB with a marker line; the store keeps the last 20 run logs (20 MiB total).
- **Environment summary:** `environment.txt` provides plain `key: value` fields (`version`, `edition`, `platform`, `interface language`, `ocr`, `ocr languages`, `ocr data dir`, `ollama model`).
- **Redaction invariants:** `Redact` / `RedactBytes` strips API keys (Google API keys `AIza...`), tokens/passwords, and personal paths (`%LOCALAPPDATA%`, `%USERPROFILE%`).
- **User consent:** diagnostic bundle generation is strictly user-initiated (UI button / report command); zero silent telemetry.

0.10 (2026-09-29) restated rule 4's retention reading - the last 10 session logs in total, current plus 9
closed. This producer keeps 20 run logs under byte bounds and names its archive with the version inside
the file name; both are deliberate and stand as dated exceptions in the catalog's registry (ticket 69).

**Conformance.** Package `internal/report` unit tests (`archive_test.go`, `redact_test.go`, `store_test.go`).
