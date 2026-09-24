# Pointer: APP-BEHAVIOUR

- **Id:** `APP-BEHAVIOUR`
- **Version:** 0.9 draft
- **Home:** the shared contracts catalog, `desktop-app-ux/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - GUI launcher behaviour (`cmd/doc-html-ui`)
- **Wire carrier:** none - desktop application user interaction

Twelve shared moments and desktop UX invariants:
- **Secondary dialogs:** centered on owner window with single no-action exit (Escape/Cancel).
- **Progress:** clear conversion progress indication without blocking cancellation.
- **Data safety:** non-destructive operations; local file conversion does not overwrite existing output unless explicitly forced.
- **Localization:** UI text adapts dynamically across 13 supported languages with RTL mirroring for Arabic and Urdu.
- **Failures:** non-fatal error presentation with actionable messages; logs preserved for user-initiated diagnostic reports.

**Conformance.** `cmd/doc-html-ui` tests and smoke tests.
