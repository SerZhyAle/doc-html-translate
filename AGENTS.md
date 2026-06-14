# AGENTS

Purpose: help coding agents become productive in this repository quickly.

## Project Snapshot

- Language: Go (module: doc-html-translate, go 1.25)
- Primary target: Windows desktop usage (CLI + GUI launcher)
- Main binaries:
  - cmd/doc-html-translate (CLI)
  - cmd/doc-html-ui (GUI wrapper for CLI)

## Start Here

1. Read README.md for product behavior and user-facing flags.
2. Prefer script-based workflows from scripts/ instead of ad-hoc commands.
3. For conversion logic changes, start in internal/pipeline/pipeline.go and trace into format-specific packages.

## Build, Test, Lint

Run from repository root in PowerShell.

- Build CLI: ./scripts/build.ps1
- Build UI: ./scripts/build-ui.ps1
- Test: ./scripts/test.ps1
- Lint: ./scripts/lint.ps1
- Full local checks: ./scripts/check.ps1

Tool bootstrap (when missing):

- ./scripts/bootstrap-tools.ps1

Notes:

- scripts/lint.ps1 expects golangci-lint.
- scripts/typo.ps1 expects typos (typos-cli via cargo).

## Architecture Map

- Entry points:
  - cmd/doc-html-translate/main.go
  - cmd/doc-html-ui/main.go
- App wiring:
  - internal/app/app.go
  - internal/config/flags.go
- Pipeline orchestration:
  - internal/pipeline/pipeline.go
- Format extractors:
  - internal/epub, internal/pdf, internal/mobi, internal/fb2, internal/rtf, internal/txt, internal/md, internal/htmlconv
- HTML processing/generation:
  - internal/htmlproc, internal/htmlsplit, internal/htmlgen
- Translation:
  - internal/translator

## Conventions And Behavior To Preserve

- Script-first dev flow: prefer existing scripts in scripts/ for routine tasks.
- Idempotent output reuse: if output index exists, pipeline reuses it unless -force is set.
- Default CLI with no args enters registration flow (not conversion).
- Translation is optional; default run is convert + open without translation engine unless -google or -ollama is passed.
- Windows and non-Windows behavior is split via *_windows.go and *_nonwindows.go files in several packages.

## Pitfalls

- scripts/build.ps1 and scripts/build-ui.ps1 copy artifacts to C:/GD/tc/SZA/_APP. This is environment-specific and may fail on other machines.
- Build scripts rely on goversioninfo for Windows resource embedding.
- MOBI/AZW3 conversion depends on Calibre at runtime.

## Editing Guidance

- Keep changes minimal and package-scoped.
- Add or update tests in the same internal package when behavior changes.
- Avoid changing public CLI flag semantics unless explicitly requested.

## Existing Docs (Link, Do Not Duplicate)

- README.md
- DEV/README.md
- docs/how-i-posted-this-project-to-winget.md
