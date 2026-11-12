# Pointer: WINDOWS-UI

- **Id:** `WINDOWS-UI`
- **Version:** 0.11 draft
- **Home:** the shared contracts catalog, `desktop-app-ux/WINDOWS-UI.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, `partial` - the GUI launcher (`cmd/doc-html-ui`), a browser-hosted Windows application (adopted 2026-10-05, ticket 93; read at 0.10 on 2026-10-20, ticket 120)
- **Wire carrier:** none - observable Windows UI and UX

0.5-0.9 moved buttons, booleans, groups, navigation and sizing into `BUTTONS`, `BOOLEAN-OPTION`,
`COLLAPSIBLE-GROUP`, `SETTINGS-NAVIGATION` and `UI-SIZING` (sections 11-14), which ticket 119 owns and measures;
[`../UI_PROFILE.md`](../UI_PROFILE.md) holds the product's profile. No longer claimed here: the tinted answer
faces of 0.4 section 11 (BUTTONS 2.0 forbids them) and the fixed 28 / 44 floors (UI-SIZING).

What the GUI holds (the sections the profile binds on this surface):

- **Section 2 - window anatomy.** Native Edge/Chrome `--app` chrome only (caption, drag, Close; no second
  header cross); a start-side rail of seven pages, one level, About last (2.2; 2.6 does not apply to a single
  level); the pane opens with its title and one-line purpose (2.3). Start/end alignment mirrors in RTL and
  technical values keep their data's direction (`TestStyleCarriesNoPhysicalDirectionRules`,
  `TestTechnicalValuesKeepTheirDirection`).
- **Section 3 - groups and remembered context.** Groups never close one another; collapsing hands focus to the
  header; expansion and the pane viewport are remembered by stable id and restoring fires no user-edit save
  (`TestGroupsAndPaneContextAreRemembered`). **3.5 search is omitted** - a permanent variant, below.
- **Section 4 - rows and editors.** Checkbox at the reading start, units next to values, a wheel that never
  edits an unfocused number, a masked key behind an explicit reveal (`TestBooleanRowsFollowTheCheckboxConvention`,
  `TestNumericEditorsScaleDigitsAndSpinners`, `TestUnitsSitNextToTheValues`,
  `TestWheelNeverEditsAnUnfocusedNumber`, `TestSecretRevealsBehindAnExplicitCheckbox`).
- **Section 5 - geometry and DPI.** One token resolver (compact 28 / 42, ordinary 32 / 48); the window fits
  its content at open, remembers its rectangle and clamps to the work area; navigation never resizes it.
  Emulated 150 and 200 percent are measured; the physical mixed-monitor pass is not claimed.
- **Sections 6-7 - theme and feedback.** The `light-dark()` table resolves on every paint (APP-STYLE
  section 10); forced-colors emulation only for high contrast; the log line reads freshly and an unreadable
  store says unknown, not zero (`TestUnknownLogCountIsNotZero`).
- **Section 8 - content viewer.** Not applicable: the GUI opens results in the user's browser.
- **Sections 11-14** are cited to ticket 119. **Section 15** (final-width remeasure, full-width flow, pointer
  profile) holds by construction - CSS flow, the only script-written sizes are `--pane-h` and a stack
  min-width - and was measured on fresh opens only, not in-session wide-narrow-wide.

**Deviations, in the catalog registry (section 3):** a second instance may run (`APP-SETTINGS` rule 1);
the dialog frame budget (500 ms) and the rendered comparison of sections 5 and 9 are not measured; the physical
acceptance of section 9 is undriven (ticket 93); both until 2026-12-31. **Permanent variant:** no settings
search (3.5) - seven pages and about thirty editors, each one rail click away
([`../UI_PROFILE.md`](../UI_PROFILE.md) section 4). The old rail exception is closed.

**Conformance.** `cmd/doc-html-ui/contract_test.go`, `conformance_test.go` and `tools/uicheck`. The
catalog's source-derived reference snapshot is read by `TestWindowsUIReferenceTargetIfAvailable` with
`SZA_CONTRACTS_ROOT`: it compares the declared compact pointer floor of the token resolver with the snapshot's
`MinTarget` (fixed 2026-10-20 - it had grepped a rule ticket 119 removed - and it skips without the catalog).
The snapshot is not a pixel vector or a runtime verdict; no machine runtime vectors exist.

**Re-read 2026-10-29 (ticket E0126) at 0.11.** Section 16: A - the reference snapshot's `MinTarget` is not a floor, so `catalog_reference_test.go` reports it beside the launcher's target instead of failing on it; B - "readily discoverable" is met by seven named pages with each setting on one, so settings search stays omitted without a variant; C - the layout reflows by construction (the script-set log-pane height is a scroll region, not a text block). Sections 5 and 9 stay open (frame budgets, ticket 93).
