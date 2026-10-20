# Pointer: APP-SETTINGS

- **Id:** `APP-SETTINGS`
- **Version:** 0.7 draft
- **Home:** the shared contracts catalog, `desktop-app-ux/APP-SETTINGS.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, `partial` - the GUI launcher's settings surface (`cmd/doc-html-ui`), adopted 2026-10-05 (ticket 93), read at 0.7 on 2026-10-20 (ticket 120)
- **Wire carrier:** none - a shipped user-facing surface (the GUI's saved blob is this product's own store, wire: none)

0.4 and the practice of 2026-10-15 are CyrFlip-private and bind no other consumer. 0.5-0.7 delegate the
group, navigation and sizing rules to `COLLAPSIBLE-GROUP`, `SETTINGS-NAVIGATION` and `UI-SIZING`
([the pointers beside this one](README.md)); ticket 119 owns and measures those, and
[`../UI_PROFILE.md`](../UI_PROFILE.md) holds the product's profile, so nothing is restated here.

The GUI's whole window is its settings surface, in the no-Commit shape `APP-BEHAVIOUR` rule 12 names:
every value control applies on touch and can be touched back; anything irreversible (delete result, clear
logs, clear the recent-documents history, rebuild over a changed result) keeps its own button and its own
confirmation with the safe answer focused (section 9). The two Windows registrations are reversible
toggles, not operations. What the rules find here:

- **Rules 1-3:** one surface with a rail of seven pages, About last, opening on the page the user left; the
  first page carries the interface language and the theme as labelled fields; every setting has its caption,
  control and muted hint, and each page opens with its title and one-line purpose.
- **Rule 4:** the declared autosave shape above. **Rules 5-6:** the language selector offers all 13 endonyms
  and applies live, including layout direction; the theme offers system, light and dark (default system),
  and a forced-colors palette wins.
- **Rules 7-8 and 10-12:** the sizing profile of ticket 119 holds 100-200 percent (emulated; the physical
  mixed-monitor pass is open), the window fits its content at open and navigation never resizes it, focus
  is visible and every control is named (`TestFocusRingIsNeverSuppressed`,
  `TestEveryFieldIsNamedAndEveryLabelNamesOne`), and About carries the version, the MIT licence, the support
  bundle and the privacy link (`TestAboutCarriesLicenceAndPrivacy`).
- **Section 8:** page, group expansion and the pane viewport are remembered by stable id, and restoring fires
  no user-edit save; the Windows layout detail is [WINDOWS-UI](WINDOWS-UI.md).

**Deviations, all recorded in the catalog registry (section 3):**

- **Rule 1** - a second launch runs a second instance (no activation IPC); dated, until 2026-12-31.
- **Rule 9 - Escape** steps back one layer (the help bubble or the page drawer, then a run) and never closes
  the window, because the settings surface *is* the main window. Permanent variant, recorded in
  [`../UI_PROFILE.md`](../UI_PROFILE.md) section 4.
- **Section 8 - search.** No settings search: seven pages, about thirty editors, each page one rail click
  away. Permanent variant, same record.
- **Rule 11** - no settings inventory aligned with the documentation (the GUI mirrors the CLI flags
  `README.md` documents); temporary, ticket 122 item 4, until 2026-12-31.

**Conformance.** The gates of [WINDOWS-UI](WINDOWS-UI.md) plus `cmd/doc-html-ui/conformance_test.go`, the
dictionary and markup gates, and `tools/uicheck` (ticket 119). The stored UI context follows the compatibility
law: keys by name, an unknown key ignored, an unknown enum to its default, a corrupt file set aside and not
overwritten, bounded geometry. No catalog vectors exist.
