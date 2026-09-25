# The desktop window behaves and looks like the rest of the portfolio

**Status:** Draft
**Priority:** 53
**Date:** 2026-09-23

> Contract sync ticket, both directions.
> Contracts: `APP-BEHAVIOUR` 0.9, `APP-STYLE` 0.9 (domain `desktop-app-ux/`, owner StreamsPlayer, drafts).
> No pointer in [`docs/contracts/`](../../docs/contracts/) yet.

## What / why

The domain's scope is "a desktop application with a window, on Windows, shipped to an end user". The GUI
`doc-html-ui` is exactly that - an HTML page in an Edge/Chrome `--app` window backed by a loopback Go
server, the installer's and the MSIX's entry point - yet the contract's declared consumer list (seven
binaries in five repositories) does not name it, and the registry has no row.

In scope: the GUI window, its dialogs, and the CLI's native message boxes that a GUI user meets during a
conversion (a grey zone - see B1). Out of scope: the installer (belongs to `INSTALL-TRUST`), the browser
extension, the CLI as a console tool.

The contract is written in WPF terms (`ShowDialog`, `IsCancel`, resource dictionaries); a browser-hosted
GUI needs each rule translated before it can be judged, and some rules do not survive translation. That is
the direction-B half of this ticket.

Found by reading the code on 2026-09-23; no rule was exercised in a running window.

## Findings that drive the work

Held: rule 2 (text grows the window) mostly; rule 7 (a missing string never ends the process) in effect;
the "send the log to the author" half of rule 6. Not applicable as written: rule 10 (no remembered
geometry) and rule 12 (no Save/Cancel settings window - every control autosaves).

Deviations, most serious first:
- **Rule 4 / 11 - acting without asking.** The GUI writes the HKCU Open-with and right-click entries on
  **every launch**, which silently undoes a user who unticked the installer's `openwith` task
  (`cmd/doc-html-ui/main.go` `ensureRightClickRegistered`; `installer/doc-html-translate.iss`). The CLI's
  no-arg flow registers first and asks afterwards (`internal/app/app.go`). The PDF path runs
  `winget install ossia.poppler` unprompted (`internal/pdf/extract.go` `tryInstallPoppler`).
- **Rule 3 - no cancel.** A conversion cannot be cancelled; the child CLI is started without the request
  context, so closing the window leaves it running and the heartbeat keeps the server alive
  (`handleRun`, `watchHeartbeat`). The OCR-pack download has no progress and no cancel.
- **Rule 1 - the cost question.** The paid-translation confirmation is an unowned `MessageBoxW` with Yes/No,
  default Yes: Enter means "spend money", and a Yes/No box has no Escape path
  (`internal/dialog/dialog_windows.go`, `internal/pipeline/pipeline.go`). It can open behind the window.
- **Rule 6 - raw errors.** About a dozen failure sites show `err.Error()` inside a sentence; a failed
  conversion ends in `Exit: exit status N` with no actions (`ui.html`, `main.go`).
- **Rule 5** - "Clear stored logs" deletes without confirmation and reports success on an empty store.
- **Rule 8** - RTL is declared twice (`i18n.js` `RTL_LANGS`, `internal/i18n` `IsRTL`) with no gate between.
- **Rule 9** - the swap button's accessible name is the glyph `⇄`; the drop zone is a `div` with no role or
  tabindex, so the main "choose a file" action is not reachable from the keyboard; the language select
  has only a `title`.
- **Localization leaks** - byline, "English (default)", WinForms dialog titles, server error strings,
  `Done.` / `Exit:`, and every CLI message box are English-only.
- **`APP-STYLE` rule 2** - dark only: no light theme, no system mode, no `prefers-color-scheme`.
- **`APP-STYLE` rules 3-5** - one palette table, but not per-theme pairs; `var(--fg)` is referenced and
  never declared (a real defect); role names differ from the vocabulary (`--accent` vs `accent`, missing
  `text.disabled`, `link`, `accent.ink`, `control.pressed`); the green console log surface is out-of-theme
  and not declared as such.

## Direction A - the product conforms

1. **Ask before acting** - the GUI registers shell integration only on the user's yes (the existing
   first-run banner is the ask), never on every launch; the CLI no-arg flow asks before it writes; the
   Poppler install becomes a question with a clear "no" path.
2. **Cancel** - a Cancel control for a running conversion that kills the child and keeps finished work;
   closing the window during a run is a defined outcome, not an orphaned process. Progress and cancel for
   the OCR-pack download.
3. **The cost question** - when launched from the GUI, asked in the GUI; in the CLI, an owned box whose
   default and Escape path is "do not spend".
4. **Failures as actions** - named cause + what to do + "send the log"; the raw error goes to the log only.
5. **Confirm the irreversible** - "Clear stored logs" confirms; "nothing to clear" is a message.
6. **Accessibility** - the drop zone is a real button; glyph-only controls carry a localized name; one RTL
   declaration or a gate that keeps two in step.
7. **Localize the leaks** listed above, GUI and CLI dialogs alike.
8. **Theme** - light, dark and system (default system, follows the OS live via `prefers-color-scheme`),
   persisted through the settings file; one palette table of per-theme pairs named by the role
   vocabulary; `--fg` fixed; the log console declared out-of-theme with its own text colour.
9. Tests for what can be pinned statically: palette roles present in both themes, RTL parity, accessible
   names of glyph-only controls in the embedded markup.

## Direction B - what the contracts need from this product

Filed as `PROPOSAL-2026-09-23-<topic>.md` in the catalog's `desktop-app-ux/` folder.

- **B1 scope** - add doc-html-translate to the consumers; say whether a console CLI that raises native
  dialogs is in scope for rule 1.
- **B2 technology-neutral wording** - a browser-hosted GUI does not own its title bar, its native
  `alert`/`confirm` placement or a Windows theme API; `prefers-color-scheme` should count as the live OS
  signal of `APP-STYLE` rule 2.
- **B3** rule 12 has no answer for autosave windows: an autosave surface may hold only reversible value
  edits.
- **B4** rule 4 speaks of "disk outside the app's own state", rule 11 of "outside the machine": is the
  app's own shell registration (HKCU verbs) its own state? Decide once.
- **B5** rule 7: allow "fall back to the source language, then the key" as a documented degradation.
- **B6** `APP-STYLE` rule 4: add `accent.hover`; note the naming gap with the web contract (`--acc` vs
  `--accent`) that `APP-STYLE` rule 6 says must be shared.
- **B7** a moment no rule covers: process lifetime when the window closes during a long operation, and
  single-instance behaviour.
- **B8** a second data point for the open `..` vs `…` menu-ellipsis collision (`Browse..`).

## Done criteria

- [ ] Pointer files `docs/contracts/APP-BEHAVIOUR.md`, `APP-STYLE.md` exist and are listed in the pointer
      README.
- [ ] Registry: this product's consumer rows for both ids with the rule-by-rule result; every deviation
      still open is a dated exception.
- [ ] No registry write and no network install happens without an answer the user gave in that session.
- [ ] A running conversion can be cancelled from the window; closing the window mid-run leaves no orphaned
      CLI process (checked with a running window, not only by reading).
- [ ] The paid-translation question defaults to "no" and has an Escape path.
- [ ] No failure surface shows a raw error string.
- [ ] The GUI offers system / light / dark and follows the OS live; `--fg` is declared.
- [ ] B1-B8 filed or withdrawn in writing here.
- [ ] Every new user-visible string exists in all 13 GUI languages.

## Open questions

1. Registry convention: implements `0.9 (partial)` or `-` (not adopted, as `INSTALL-TRUST` was declared)?
2. Shell integration: make it strictly opt-in, or argue (B4) that registering the app's own verbs is its
   own state?
3. Cancel semantics: what does "finished work kept" mean for a half-written output folder, given the
   invariant that an existing `index.html` is reused unless `-force`?
4. Replace native `alert`/`confirm` with in-page dialogs (owner-centred, safe default) everywhere, or only
   for the destructive and costly ones?

## Notes

Changes the behaviour of a shipped invariant in two places (first-run registration, cost confirmation);
both are user-visible and each needs the docs/site line that describes it updated in the same edit.
`main.go` (1082 lines) and `ui.html` (1226) are the only two files most items touch - split before adding.
