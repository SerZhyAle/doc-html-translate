---
paths:
  - "cmd/doc-html-ui/**"
  - "tools/helpgen/**"
  - "tools/helpcheck/**"
  - "internal/helpcheck/**"
  - "help/**"
  - "tools/uicheck/**"
  - "tools/settingsinv/**"
  - "internal/settingsinv/**"
  - "docs/settings-inventory.json"
  - "docs/UI_PROFILE.md"
  - "scripts/help-check.ps1"
  - "scripts/settings-inventory.ps1"
  - "internal/htmlgen/**"
  - "extension/src/viewer.*"
---

# GUI launcher, in-app help, settings, UI contracts

Moved verbatim from `AGENTS.md` / `CLAUDE.md` (2026-11-25); the short core there still binds. A section
name quoted below is a section of `AGENTS.md` unless it says otherwise.

## GUI window structure

- GUI window structure (ticket 119): `cmd/doc-html-ui/ui.html` is one page of three layers. The **token block** (`html { --ui .. --tx .. --tgt .. }`, `html[data-input=touch]`, `data-size`, `data-text`) is the one resolver for every covered component (never hard-code a px size in a component; use the tokens and `var(--s)`); the **components** are `button` (ranks `.primary` / `.secondary` / `.danger`, rows `.btn-row`, stateful `.state-stack`), `.opt` (a boolean option; `.opt-help-wrap` for local help), `.group` (a `<details>` group) and `.rail-item` (the page rail); the **pages** are seven `<section class="page" id="page-..">` shown one at a time by `showPage()` (`convert`, `ollama`, `ocr`, `windows`, `interface`, `advanced`, `about`; ids are stable and saved). The dialog answers read Cancel first, the affirmative last (BUTTONS 4.4). `/api/resources` is the About page's local, no-network resource lookup.

## In-app help

- In-app help (ticket 118, `CONTEXT-HELP`; pointer [docs/contracts/CONTEXT-HELP.md](../../docs/contracts/CONTEXT-HELP.md)):
  - `cmd/doc-html-ui/help/help-map.json` is the **help map** (CONTEXT-HELP rule 17): one row per surface of the GUI launcher, its form, its help id and the `none` / `inherit` reasons. It ships inside the exe with the generated pages (`go:embed all:help`, served under `/help/`) and the gate reads the same file.
  - `cmd/doc-html-ui/help.js` is the one resolver: F1 (bare, on the page only) and every `?` button call it. A region of `ui.html` is marked `data-help="<surface>.<region>"`, a dialog names its `surface:`; the gate counts both against the map.
  - `tools/helpgen` generates the pages, the screenshots with their pins and `help/` (the site copy, the same bytes) from the copy deck `tools/helpgen/copy/*.json`, the GUI dictionary `i18n.js` and the real build. **Never edit `cmd/doc-html-ui/help/` or `help/` by hand**; change the window, the dictionary or the copy deck, run `node tools/helpgen/repage.mjs` from PowerShell for the wording (the same edit), and leave `node tools/helpgen/generate.mjs` - the screenshots and pin places, about two minutes with the GUI build - to the release (dated exception on CONTEXT-HELP rule 18, ticket E0125; see [tools/helpgen/README.md](../../tools/helpgen/README.md)). `node tools/helpgen/walk.mjs` is the automated F1 / button walk.
  - `internal/helpcheck` is the gate (`scripts/help-check.ps1`, also `tests/help_test.go`): surfaces against the map, pages against the locale set, links, rule 12, text style, the two copies, and the freshness of the set against the digests of its inputs (an advisory in the fast tier, a failure under `-Release` in the full gate and the release pre-flight: `go run ./tools/helpcheck -strict-freshness`). The locales the GUI ships and the help has no pages in are the set baseline `configs/help-baseline.txt` - it may fall, never rise.
- A change that alters what the user sees in the GUI launcher (a control, a label, a region, a dialog) changes its help in the same edit (CONTEXT-HELP rule 18): the window's own strings are the help's names and one-line summaries, the explanation is in `tools/helpgen/copy/`, and the screenshots are regenerated, not edited. A new region gets `data-help`, a new dialog a `surface:` and a row in the help map, or the gate fails.

## In-app help commands

- In-app help: ./scripts/help-check.ps1 (the gate child, no browser; the full gate and the release pre-flight run it with `-Release`, where a set older than its inputs FAILS, and the fast tier only advises); node tools/helpgen/repage.mjs [--check] is how a wording change of the copy deck, help.css or the renderer reaches both copies of the set in the same edit: it re-renders the pages from the committed captures with no build and no browser (0.07 s on 2026-10-29), touches no screenshot and exits 2, writing nothing, when a window input (ui.html, i18n.js, the capture code) changed after the last capture; node tools/helpgen/generate.mjs regenerates the screenshots, the pin places and the pages after a change of what the window shows (needs the Go toolchain and tools/sitecheck's Playwright Chromium; it builds the GUI and its capture alone took 106 s on 2026-10-28) and **it belongs to the release, not to the window edit**: CONTEXT-HELP rule 18 carries a dated exception for the screenshot half (ticket E0125, owner decision 2026-10-30, expires 2026-12-31), so between a window change and the release the committed screenshots lag the window and `help-check` names the moved input as an advisory; a ticket that changes the window records `node tools/helpgen/generate.mjs` as a `RELEASE` item, and the release pre-flight runs it, then `--check`; node tools/helpgen/walk.mjs drives F1 and the button on every surface in five locales and both themes

## Settings inventory

- Settings inventory (ticket 122; `APP-SETTINGS` rule 11): ./scripts/settings-inventory.ps1 (a gate child) holds `docs/settings-inventory.json` - every key `collectSettings()` persists, with its control, page, caption, default, help region and command-line flags - equal to `ui.html`, the help map, the help copy deck and the README flag tables; after adding, renaming or moving a setting run `go run ./tools/settingsinv -render` (never edit the JSON), and name the new caption in `tools/helpgen/copy/` before regenerating the help set

## UI contracts

- UI contracts (ticket 119; `UI-SIZING`, `BUTTONS`, `BOOLEAN-OPTION`, `COLLAPSIBLE-GROUP`, `SETTINGS-NAVIGATION`): `node tools/uicheck/run.mjs` measures the real GUI window in headless Chromium (geometry, contrast, keyboard, negative cases, page-switch timings with every raw sample; `--quick`, `--only <scenario>`; last line is the CHECK-VERDICT line; see [tools/uicheck/README.md](../../tools/uicheck/README.md)); the product's size-profile record, permanent variants and what is still owed are in [docs/UI_PROFILE.md](../../docs/UI_PROFILE.md). It builds the GUI and runs for minutes, so it belongs to the release pre-flight: run it there when `ui.html` changed since the last release, and `--only reader` when the generated reader chrome (`internal/htmlgen` `chromeUICSS`, `navbar.go`, `search.go`) or the extension viewer's toolbar (`extension/src/viewer.*`) changed. A ticket that changes them records the run as a `RELEASE` item; the help screenshots wait for the release too (CONTEXT-HELP rule 18 exception, ticket E0125 - `node tools/helpgen/generate.mjs` is another `RELEASE` item, and `node tools/helpgen/repage.mjs` carries any wording change in the same edit) - the window's regions are the help's inputs
