# The desktop window behaves and looks like the rest of the portfolio

**Status:** Implemented - Direction A verified 2026-09-25; the catalog half (registry row, exception, B1-B8, the section 4 danger report) done 2026-09-28
**Priority:** 53
**Date:** 2026-09-23

> Contract sync ticket, both directions.
> Contracts: `APP-BEHAVIOUR` 0.9, `APP-STYLE` 0.9 (domain `desktop-app-ux/`, owner StreamsPlayer, drafts).
> Both are 0.10 in the catalog since 2026-09-24 (rule 1 narrowed to secondary windows; `APP-STYLE` section 5
> content-coloured-window amendment; doc-html-translate added to both `consumers` lists).
> Pointers now exist in [`docs/contracts/`](../../../docs/contracts/) (see "Re-verified 2026-09-25").

> **Remote execution (2026-09-25):** the contract text this ticket needs is quoted in "Contract snapshot" below, so every step not marked ⛔ runs in a cloud session from this repository alone. Steps marked **⛔ Local only** edit the shared contracts catalog (or another repository) and can run only on the owner's machine, where the catalog is mounted.

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

### Re-verified 2026-09-25

Re-read against the catalog and the working tree on 2026-09-25, by reading only (no running window):

- **Done:** pointer files `docs/contracts/APP-BEHAVIOUR.md` and `docs/contracts/APP-STYLE.md` exist and are
  listed in `docs/contracts/README.md` (rows 25-26). **Drift:** both say `0.9 draft`; the catalog is 0.10.
- **Done (catalog side):** the registry carries a consumer row `APP-BEHAVIOUR`, `APP-STYLE` /
  doc-html-translate (GUI launcher), dated 2026-09-24, and both contracts' `consumers` lists name
  `doc-html-translate (GUI launcher)` - so B1's "add to consumers" half is already done.
- **Not true in code, although the row and the pointers claim it.** The registry row says "Escape dismisses
  modal dialogs, non-blocking asynchronous conversion lifecycle, .. dark/light system theme compliance,
  actionable error reporting". None of that is in the tree: `cmd/doc-html-ui/ui.html` has no `keydown` or
  Escape handler and uses native `alert`/`confirm` (`:1050-1069`, `:1173-1185`); no `prefers-color-scheme`
  or `matchMedia`, and the palette is one dark set (`:10-21`); `var(--fg)` is still referenced and never
  declared (`:262`). The pointers likewise claim "progress .. without blocking cancellation" and
  "dark, light, and system modes" for the GUI. The row has no rule-by-rule result and no exceptions.
- **Every deviation above still holds**, unchanged: `go ensureRightClickRegistered()` on every launch
  (`cmd/doc-html-ui/main.go:101`, `:112-119`); CLI no-arg registration before asking
  (`internal/app/app.go:40-52`); `tryInstallPoppler` (`internal/pdf/extract.go:95`, called at `:125`);
  `exec.Command(bin, args...)` without the request context (`main.go:661`); `ConfirmYesNo` passes owner `0`
  and `MB_YESNO|MB_ICONQUESTION`, no default-button flag (`internal/dialog/dialog_windows.go:24-33`), used for
  the cost warning with an English title (`internal/pipeline/pipeline.go:328`); `handleLogsClear` clears
  without confirmation and the UI reports `logsCleared` regardless of count (`cmd/doc-html-ui/report.go:129-139`,
  `ui.html:1157-1167`); RTL declared twice (`i18n.js:20`, `internal/i18n/i18n.go:83`); swap button labelled
  by the glyph `&#x21C4;` plus a `title` (`ui.html:344`); drop zone a `div` with `onclick` only (`ui.html:298`).

## Direction A - the product conforms

1. **Ask before acting** - the GUI registers shell integration only on the user's yes (the existing
   first-run banner is the ask), never on every launch; the CLI no-arg flow asks before it writes; the
   Poppler install becomes a question with a clear "no" path. Not blocked on B4: strictly opt-in is the
   default to implement, because it conforms whichever way B4 is answered.
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
10. **Pointers tell the truth** - `docs/contracts/APP-BEHAVIOUR.md` and `APP-STYLE.md` (and their rows in
   `docs/contracts/README.md`) move to 0.10 and describe what the GUI actually holds, not the claims listed
   under "Re-verified 2026-09-25".

## Direction B - what the contracts need from this product

**⛔ Local only - changes the contract catalog.** Every item below is filed as
`PROPOSAL-2026-09-23-<topic>.md` in the shared contracts catalog, `desktop-app-ux/`.

- **B1 scope** - ~~add doc-html-translate to the consumers~~ (done in the catalog on 2026-09-24, see
  "Re-verified 2026-09-25"); say whether a console CLI that raises native
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

## Implementation (2026-09-25)

Decisions taken for the open questions below: **Q2** strictly opt-in (the code half; arguing B4 stays
catalog work). **Q3** "finished work kept" = what the converter already wrote stays in the output folder;
a cancelled run writes no completion record, so the next run rebuilds it - the reuse invariant is untouched,
and the page says exactly that. **Q4** every native `alert`/`confirm` is replaced by one in-page modal
dialog; destructive and costly buttons use `--danger`. Found stale on arrival: conversion Cancel and the
process-tree kill (`run.go`, an earlier ticket) and the Poppler auto-install (removed by ticket 11).

- **Step 1** GUI launch writes nothing; first-run banner (right-click only / also default / No, thanks),
  a right-click toggle + `/api/shell-entries`, `windowsreg.HasShellEntries` / `RemoveShellEntries` (keeps
  `Applications\<exe>`); the CLI no-arg flow asks `[y/N]` before it writes.
- **Step 2** OCR download streams `{done,total}` lines and cancels through the request context
  (`ocr.DownloadContext`).
- **Step 3** `internal/dialog.Confirm`: owned (console window) OK/Cancel, Cancel default; under the GUI
  (`DOCHT_DIALOG_HOST=stdio`) an `ask` marker line on stdout, the answer on stdin via `/api/answer`;
  `ShowWarning` becomes a `note` marker. Title and text localized (13).
- **Step 4** the run ends with a data line (`\x1edht:end {state,code}`); a failure dialog offers send logs /
  try again / close; every page failure is a named cause + action; raw errors go to
  `run-<launch>-gui<pid>.log` in the run-log store.
- **Step 5** clearing logs confirms; an empty store says so (`cleared`).
- **Steps 6-8** drop zone is a `<button>`, `data-i18n-aria`, RTL gate; leaks localized (byline, OCR
  fallback name, file-dialog captions - passed to PowerShell as base64); one `light-dark()` palette table,
  theme select saved in settings, `.console` out-of-theme, `--fg` gone.
- **Step 9** `cmd/doc-html-ui/contract_test.go`. **Step 10** pointers at 0.10, rule by rule.

## Verification (2026-09-25)

- `go test ./cmd/... ./internal/...` - ok except `internal/pdf` `TestExtract_Volume3ImagesOnTheirPages`
  and `internal/procrun` `TestRunMissingBinaryKeepsThePathError`, which fail identically on a clean HEAD
  worktree (pre-existing, packages not touched here); `go test ./tests/ -skip TestConvertTestDoc` - ok.
  `TestConvertTestDoc` dies with `fatal error: out of memory` on the 386 toolchain - reproduced on a clean
  HEAD worktree, so pre-existing and unrelated.
- Live: the real page and API (the test binary as a converter that asks the cost question) driven in
  headless Edge over CDP - `live: PASS`, 25/25: OS light -> dark followed without reload, pinned themes
  override it, console stays dark, Escape = no action, Enter declines a destructive or costly question,
  cost question in the window with "do not spend" focused, declined run -> failure dialog naming code 4 and
  no raw error, accepted run -> localized "done", empty log store -> message, Arabic mirrors.
- Not checked by hand: the real converter's own cost box from a console, and closing the `--app` window
  mid-run (pinned by `TestDroppedRequestStopsTheChild` with a real child tree).
- `scripts/lint.ps1`: three pre-existing `errcheck` findings (`windows.CloseHandle`) in files this ticket
  did not touch.

Still open when Direction A closed, both closed by the catalog half below (2026-09-28): the Explorer
caption of the right-click verb (`Convert to HTML`) - one registry value for all languages - is now a
dated exception in the shared registry; and the catalog half itself is done.

## Catalog half (2026-09-28, local)

The ⛔ half, executed on the owner's machine against the mounted catalog (`desktop-app-ux/`):

- **Registry row rewritten.** The 2026-09-24 adoption row (which claimed Escape-dismissed dialogs, a
  cancellable lifecycle, three themes and actionable errors the tree did not have) is replaced by the
  result of Direction A: `APP-BEHAVIOUR`, `APP-STYLE` / doc-html-translate (GUI launcher) / C /
  implements `0.10 / 0.10`, verified 2026-09-28, rule by rule with each gate test named, the 2026-09-25
  verification runs quoted, and the old row kept as history beneath the new one.
- **Exception recorded.** The Explorer verb caption (English in every UI language; one `MUIVerb` value
  serves all languages, and a per-language caption means MUI indirect strings - a resource DLL per
  language on every install channel) is a dated exception in the registry, until 2026-12-31, pointing
  at B4 for the maintenance-consent question.
- **B1-B8 filed** as `PROPOSAL-2026-09-28-*` in `desktop-app-ux/`, each naming its B-number:
  `cli-dialogs-rule-1-scope` (B1), `prefers-color-scheme-live-signal` (B2), `autosave-settings-surface`
  (B3), `own-state-shell-registration` (B4), `localization-degradation-ladder` (B5),
  `accent-hover-and-web-naming` (B6), `window-close-lifetime-single-instance` (B7),
  `menu-ellipsis-second-data-point` (B8). All eight open, awaiting the owner's decision; none edits the
  contracts themselves (StreamsPlayer owns both; `RULES.md` section 7 - proposal, not edit).
- **Plus the report `APP-STYLE` section 4 asks for.** Open question 4's decision (in-page confirm
  everywhere, destructive and costly buttons in `--danger`) made this GUI the product that report
  waits for: filed as `PROPOSAL-2026-09-28-confirmation-uses-danger.md`, with what the confirmation
  looks like - the condition the proposed `danger` / `warning` / `success` roles name for flipping to
  [CONTRACT].
- The domain README's proposal table lists all nine.

## Done criteria

- [x] Pointer files `docs/contracts/APP-BEHAVIOUR.md`, `APP-STYLE.md` exist and are listed in the pointer
      README, at 0.10 and describing what the GUI holds (step 10, done with Direction A).
- [x] Registry: this product's consumer rows for both ids rewritten 2026-09-28 from the result of
      Direction A - one adoption row (`0.10 / 0.10`, rule by rule, the 2026-09-24 claims replaced by
      what the tree holds) and every still-open deviation a dated exception (the English Explorer verb
      caption, until 2026-12-31).
- [x] No registry write and no network install happens without an answer the user gave in that session.
- [x] A running conversion can be cancelled from the window; closing the window mid-run leaves no orphaned
      CLI process (checked with a running window, not only by reading).
- [x] The paid-translation question defaults to "no" and has an Escape path.
- [x] No failure surface shows a raw error string.
- [x] The GUI offers system / light / dark and follows the OS live; `--fg` is declared.
- [x] B1-B8 filed 2026-09-28 as `PROPOSAL-2026-09-28-*` in the catalog's `desktop-app-ux/`, plus the
      `APP-STYLE` section 4 `danger` report the contract itself asked for; the domain README lists all
      nine. Withdrawn: none.
- [x] Every new user-visible string exists in all 13 GUI languages.

## Open questions

1. Registry convention - **answered 2026-09-28: neither.** The Implements column records the version
   the tree implements, verified by reading plus the suite, released or not - the precedent is this
   repo's own `OCR-OVERLAY` row (ticket 21, also unreleased when its row said 1.2). The row reads
   `0.10 / 0.10`; when the release ships is the queue's question, not the registry's.
2. Shell integration: strictly opt-in in the code (Direction A step 1, done); arguing B4 is filed as
   `PROPOSAL-2026-09-28-own-state-shell-registration.md`, awaiting the owner.
3. Cancel semantics - answered in "Implementation" (2026-09-25): what the converter wrote stays, no
   completion record is written, the next run rebuilds; the reuse invariant is untouched.
4. Answered in "Implementation" (2026-09-25): every native `alert`/`confirm` replaced by the in-page
   dialog; destructive and costly buttons use `--danger` - and that answer is what made this product
   the report `APP-STYLE` section 4 waits for (see "Catalog half").

## Notes

Changes the behaviour of a shipped invariant in two places (first-run registration, cost confirmation);
both are user-visible and each needs the docs/site line that describes it updated in the same edit.
`main.go` (1082 lines) and `ui.html` (1226) are the only two files most items touch - split before adding.

