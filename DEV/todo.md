# TASK: DOC-HTML-UI — GUI Frontend for doc-html-translate

## Project Links
- GitHub Pages: https://serzhyale.github.io/doc-html-translate/
- GitHub Repository: https://github.com/SerZhyAle/doc-html-translate

**Date:** 2026-03-07  
**Status:** Research required before implementation

---

## Objective

Design and implement a standalone Windows GUI application (`DOC-HTML-UI.EXE`) that serves as a user-friendly frontend for `doc-html-translate`. The UI program assembles CLI flags/values and launches the core Go binary with the configured parameters.

---

## Requirements

### Application Identity
- Executable name: `DOC-HTML-UI.EXE`
- Displays current version of both itself and the underlying `doc-html-translate` binary.

### Input
- **Drag & Drop**: Accept file(s) dropped onto the application window.
- **File picker**: Standard open-file dialog as a fallback for file selection.

### Output
- **Output folder selector**: Browse-dialog for choosing the output/save directory.
- All fields pre-filled with **sensible defaults** on startup.

### Flags / Options
- Render **every CLI flag** of `doc-html-translate` as a labelled **checkbox** (boolean flags) or **input field** (value flags).
- Default values must match the defaults defined in `doc-html-translate`.

### Execution
- On "Run" / "Convert": build the CLI command from the current UI state and execute `doc-html-translate` with those arguments.
- Show stdout/stderr output in a scrollable log panel within the UI.

---

## Technology Choice

**Preferred:** Go (e.g., `fyne`, `walk`, or `webview`).  
**Fallback:** Any platform-suitable technology (.NET WinForms / WPF, etc.) — the UI is just a launcher, so the language doesn't matter as long as it ships as a native Windows executable.

**Research gate:** Before implementation, evaluate Go GUI options (Fyne, Walk, Lorca/webview) for effort vs. feasibility. Document findings in `DEV/research/`.

---

## Out of Scope

- No business logic — all processing stays in `doc-html-translate`.
- No macOS / Linux build required at this stage.

