# Pointer: WINDOWS-UI

- **Id:** `WINDOWS-UI`
- **Version:** 0.3 draft
- **Home:** the shared contracts catalog, `desktop-app-ux/WINDOWS-UI.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - the GUI launcher (`cmd/doc-html-ui`), a browser-hosted Windows application (adopted 2026-10-05, ticket 93)
- **Wire carrier:** none - observable Windows UI and UX

What the GUI holds (the sections the profile binds on this surface):

- **Section 2 - window anatomy.** Native Edge/Chrome `--app` chrome only (caption, drag, Close; no second
  header cross); the whole settings inventory is one pane with independently collapsible groups, About last.
  Structural start/end alignment mirrors in RTL - the stylesheet carries no physical left/right rules
  (`TestStyleCarriesNoPhysicalDirectionRules`).
- **Section 3 - groups and remembered context.** Groups never close one another; collapsing hands focus to the
  header (`TestGroupsAndPaneContextAreRemembered`); expansion is remembered by stable id and the pane viewport
  as a stable anchor plus offset, both riding the settings blob, and restoring fires no user-edit save.
  Search is omitted under section 3.5's compact-utility allowance - every setting is visible in the one pane.
- **Section 4 - rows and editors.** Checkbox at the reading start with the caption beside it and the description
  below at the caption's inset (`TestBooleanRowsFollowTheCheckboxConvention`); numeric editors scale their digits
  and native spinners (16px / 110px / 32px, `TestNumericEditorsScaleDigitsAndSpinners`); units sit immediately
  next to the value (`TestUnitsSitNextToTheValues`); the wheel never edits an unfocused number
  (`TestWheelNeverEditsAnUnfocusedNumber`); the Google key stays masked until its explicit reveal checkbox
  (`TestSecretRevealsBehindAnExplicitCheckbox`).
- **Section 5 - geometry and DPI.** The window fits its content at open, remembers its rectangle and clamps to
  the work area; toggling a group no longer resizes the window (the one-way grow of ticket 63 was superseded
  by this profile); the web view scales per-monitor DPI natively - emulated 150/200% verified in the driven
  run, the physical mixed-monitor pass is not claimed.
- **Sections 6-7 - theme and feedback.** The `light-dark()` palette table resolves on every paint, so every
  nested control, the dialog included, follows the live theme (APP-STYLE section 10); the about/log cache line
  keeps usage and Clear together, read freshly. High contrast: forced-colors emulation verified in the driven
  run only; a physical OS high-contrast pass is an open exception.
- **Section 8 - content viewer.** Not applicable to the desktop edition: the GUI opens results in the user's
  browser rather than hosting a media canvas.

**Second pass, 2026-10-06 (ticket 95).** Section 2.3: the pane gained its one-line purpose under the title.
Section 2.5: the command-line preview, the log, the result path and the path, model and key fields keep
left-to-right in Arabic and Urdu - technical values keep their data's direction. Section 3.4: restoring the
groups and the viewport fired their saves as later tasks and could store an empty OCR-language choice before
the catalog arrived, and every launch re-opened the Ollama group over a remembered collapse - a restoring
guard, a pending-value read and a user-only engine expand fix them. Section 6.1: static headings no longer
take the interaction accent. Section 6.4: a selected list row under the pointer measured 4.33:1 in dark; the
hover and selected fills are now in the contrast gate (`TestListRowStatesMeetWCAGAA`). Section 6.5: the
mask-drawn group chevron vanished under forced colors and is drawn in `CanvasText` there. Section 7: an
unreadable log store showed "0 logs" and "nothing to clear"; it now says unknown, with the reason (only a
store not created yet is empty). Gates: `cmd/doc-html-ui/conformance_test.go`.

**Dated registry exceptions:** a second instance may run (no activation IPC; APP-SETTINGS rule 1 /
APP-BEHAVIOUR rule 3 documented decision), and the single pane records no vertical navigation list (there are
no destination pages to navigate; About is last). Both are in the catalog registry with reasons and until
dates.

**Conformance.** `cmd/doc-html-ui/contract_test.go` (the ticket-93 gates named above) and the driven CDP run
of 2026-10-05 (`temp/ui93run/driven.log`: 14/14 - both themes, Light->Dark->Light cycle, trusted-Escape
dialog, safe-answer focus, wheel guard, group/scroll restoration over a fresh load, Arabic RTL,
forced-colors and 150/200% emulation, screenshots beside the log).

**Review 2026-10-07 (ticket 108).** Section 2.6 does not apply: this compact launcher has no grouped navigation list. Sections 5 and 9 add first/repeat/dialog frame budgets; those timings and a visual comparison are not measured, explicitly excepted until 2026-12-31. Existing layout and physical acceptance evidence is retained, not promoted to a new driven pass.

The catalog's source-derived reference snapshot is read by
`TestWindowsUIReferenceTargetIfAvailable` with `SZA_CONTRACTS_ROOT`: its common MinTarget is
compared with the launcher. The snapshot is not a pixel vector or a runtime verdict;
the new frame budgets remain unverified. No machine runtime vectors exist.
