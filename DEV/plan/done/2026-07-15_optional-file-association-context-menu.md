# Optional file-type association + always-on right-click entry

**Status:** Implemented
**Priority:** 2
**Date:** 2026-07-15

## What / why

Today the app and the extension make themselves the **default** handler for supported document types
out of the box: the unpackaged first-run (no-arg) flow sets this app as the HKCU default for all ten
extensions, and the browser extension auto-intercepts every PDF/EPUB/RTF/FB2/MOBI/AZW3 it sees
(`enabledByDefault: true`). That is intrusive - users lose their existing PDF/EPUB viewers on the first
run without asking.

New behaviour: **being the default handler becomes an explicit opt-in, off by default, for all ten
types.** Nothing is auto-associated on install/first-run. Instead the app is always reachable through a
right-click entry - a dedicated **"Convert to HTML"** verb (plus the existing "Open with" entry) in
Windows Explorer, and a **"Convert with doc-html-translate"** context-menu item in Chrome/Edge - for
every supported file type. The user can turn on default-handler association whenever they want, via an
in-app toggle and a one-time first-run prompt. Since there is no classic installer with checkboxes
(Store / winget / portable exe), "optional at install" is realised as *don't grab defaults + offer an
opt-in inside the app*.

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[x] Done` | No-arg first-run no longer auto-sets defaults: it registers the non-destructive **"Convert to HTML"** shell verb (`RegisterContextMenu`) + "Open with" for all types, then interactively offers the opt-in ("Make default handler? [y/N]"). `-register` stays the explicit "become default" opt-in; new `-unregister` clears it. `FirstRun` config replaces the old implicit `Register`. |
| GUI (`doc-html-ui`) | `[x] Done` | Launch registers verb + "Open with" (`ensureRightClickRegistered`), not the default handler. Settings **toggle** bound to live state (`/api/assoc-status` -> `IsDefaultHandler`; on -> `/api/register`, off -> `/api/unregister`). One-time first-run banner offering the opt-in (dismissal persisted in ui-settings `assocPrompted`). Hidden when packaged. |
| MSIX Store app | `[x] Declined` | HKCU is virtualized under MSIX, so a runtime shell verb won't persist. The manifest `fileTypeAssociation` already cannot steal a default (Windows always prompts) and only surfaces the app in "Open with", so it is opt-in by OS design and stays as-is. GUI hides the toggle/prompt when packaged. Native `IExplorerCommand` handler deferred (needs a COM component). Recorded in docs/PARITY.md intentional divergences. |
| Browser extension | `[x] Done` | `enabledByDefault` flipped `true` -> `false` (no auto-interception). New "Convert with doc-html-translate" context-menu on `link` + `page` for supported document URLs (`targetUrlPatterns`/`documentUrlPatterns`); click opens the reflow viewer via `?file=<encoded>`. Popup toggle now starts off; added a note + `convertDocMenu` i18n (en/ru/uk). |
| Website / docs | `[x] Done` | README, docs.{html,ru,uk} flag tables (+`-unregister`, opt-in wording), extension.html "ways"/limitations (en/ru/uk) reworked for off-by-default + right-click, CHANGELOG, PARITY.md. |

## Validation

- `go build ./...`, `go vet ./...`, `go test ./...` green (updated `flags_test.go`: no-arg -> `FirstRun`; added `-unregister` test; parity guard allow-lists `-unregister`).
- Extension `npm test` green (39/39). Locale JSON + `background.js`/`defaults.js` syntax-checked; ui.html inline JS syntax-checked.
- Live registry round-trip verified in isolation against a fake extension: `RegisterContextMenuFor` writes the verb+command and `MUIVerb`; seeded default -> `IsDefaultHandler` true; `Unregister` releases it (idempotent, empty on re-run); keys cleaned up. Real associations untouched.

## Shared invariants touched

- **`enabledByDefault` default flips `true` -> `false`** (docs/PARITY.md "Settings defaults" and the
  PDF/EPUB interception note). Update PARITY.md.
- **New cross-edition invariant:** no edition auto-associates itself as the default handler; default
  association is always an explicit user opt-in. Every edition provides an always-available right-click
  entry instead. Add to PARITY.md.

## Cross-references

- Go `internal/windowsreg/register_windows.go` (`RegisterHandler` = become-default opt-in;
  `RegisterOpenWith*` + new `RegisterContextMenu`/`Unregister`) <-> extension `src/background.js`
  (`enabledByDefault` interception + `contextMenus`).
- Go first-run opt-in flag (persisted "asked already") <-> extension `enabledByDefault` stored option.
- Splash / first-run copy: `internal/app/app.go` <-> extension popup/options strings.

## Done criteria

- [x] Every edition row above is `Done` or `Declined (reason)`.
- [x] docs/PARITY.md updated (defaults table + new no-auto-default invariant + MSIX divergence).
- [x] Go build/vet/tests green and `npm test` green (extension).
- [x] Changelog entry in `DEV/CHANGELOG.md`.

## Tactical outline (to be expanded via /spec-tech)

1. **Go registry layer** (`internal/windowsreg`): add `RegisterContextMenu()` writing
   `Software\Classes\SystemFileAssociations\<ext>\shell\dochtmltranslate.convert\command` = `"<exe>" "%1"`
   with a friendly `MUIVerb` ("Convert to HTML") + `Icon` for all `SupportedExtensions`; add
   `Unregister()` that removes the default-handler ProgID assignment (leaving the verb + Open-with). Keep
   `RegisterHandler` as the become-default path. Unit-testable key-path builders where possible.
2. **CLI flow** (`internal/config/flags.go`, `internal/app/app.go`): no-arg first-run -> register context
   menu + open-with (non-destructive) and prompt for the default-handler opt-in; add `-unregister`.
   Preserve the "no-arg = registration flow" invariant.
3. **GUI** (`cmd/doc-html-ui`): launch-time context-menu + open-with registration; `/api/unregister`;
   Settings toggle bound to current association state; one-time first-run modal.
4. **Extension** (`extension/src`): `defaults.js` flip; `background.js` context-menu on link/page for
   supported doc URLs -> open viewer; verify popup/options copy.
5. **Docs**: PARITY.md, README, site, extension listing (en/ru/uk), CHANGELOG.
