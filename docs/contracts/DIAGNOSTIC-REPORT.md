# Pointer: DIAGNOSTIC-REPORT

- **Id:** `DIAGNOSTIC-REPORT`
- **Version:** 0.12 draft
- **Home:** the shared contracts catalog, `diagnostic-report/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** producer - `internal/report` generates diagnostic zip archives and environment summaries
- **Wire carrier:** `schemaVersion` in metadata or `environment.txt` (`key: value` lines) + sanitized session log bundle

How diagnostic bundles and support packages are assembled and sanitized:
- **Archive format:** `.zip` named `report_doc-html-translate_<version>_<timestamp>.zip` containing `environment.txt`, `settings.json`, and recent log files under `LogsDir()`.
- **Bounded size & compaction:** archive capped at 15 MB (`MaxArchiveBytes = 15 << 20`); newest logs included first, older logs dropped when cap is reached. A run log stops growing at 5 MiB with a marker line; the store keeps the last 20 run logs (20 MiB total).
- **Environment summary:** `environment.txt` provides plain `key: value` fields (`version`, `edition`, `platform`, `interface language`, `ocr`, `ocr languages`, `ocr data dir`, `ollama model`).
- **Redaction invariants:** `Redact` / `RedactBytes` strips API keys (Google API keys `AIza...`), tokens/passwords, and personal paths (`%LOCALAPPDATA%`, `%USERPROFILE%`).
- **User consent:** diagnostic bundle generation is strictly user-initiated (UI button / report command); zero silent telemetry.

**Re-read 2026-10-02 at 0.12.** A version in the archive name is now allowed, but this product's
`report_` prefix still differs from the `-logs-` pattern. Its run-log retention/capping exception stays.
The environment has a per-product field set (absent schema key means 1).
**Conformed 2026-10-02 (ticket 91):** Redactor in `internal/report/redact.go` conforms to sections 7 and 8 C
(structured JSON values, signed-CDN query parameters with `&amp;`, credential-in-path, URL userinfo, short
secrets with `[REDACTED]` marker); all catalog vectors pass in `TestRedactStructuredValueVectors` and
`TestRedactCatalogVectorFileIfAvailable`. The redaction exception is closed. The archive name and run-log
retention deviations remain explicitly dated in the registry.

**Conformance.** Package `internal/report` unit tests (`archive_test.go`, `redact_test.go`, `store_test.go`).
