# Strategic spec: 20_2026-09-24_bugfix-windows-registration-honesty - File association that works, reverts and reports truthfully

**Ticket:** 20_2026-09-24_bugfix-windows-registration-honesty
**Status:** BlockNeedUserTest - implemented and covered by tests (the Windows registry tests run under Wine against a fake hive); needs the manual Windows 11 check of §13: an existing user choice for `.epub`, then register and unregister.
**Priority:** 50
**Date:** 2026-09-24
**Tier:** Easy
**Tactical plan:** none - implemented directly from this spec (see §13)
**Findings:** P12 P13 P14 (see the [findings register](../../research/audit_2026-09-24/README.md))

> **Scope:** STRATEGIC.

---

## 1. Problem
"Make default handler" writes only the per-user class key. Since Windows 8, the user's own choice
record wins over that key, so where such a record exists nothing changes. The app still announces
that double-click now opens files with it, and Explorer is not told to refresh. The GUI toggle
reads the same key and misreports the state. Unregistering deletes the association without
restoring the handler that was there before, and ignores deletion errors. Registration failures on
first run are discarded, so the user sees "done" when nothing happened.

## 2. Goals
1. After "make default", the app either is the handler, or tells the user exactly how to finish the step in Windows Settings.
2. Explorer reflects changes immediately.
3. Unregistering restores whatever per-user handler existed before registration.
4. Every registration outcome, success, partial or failure, is reported accurately in the CLI and the GUI.
5. The GUI's "is default" indicator reflects what Windows actually uses.

**Non-goals:**
- Bypassing the protected user-choice mechanism (not allowed by Windows policy).

## 3. Wishes and constraints
### 3.2 Hard constraints
- **Platform / versions:** Windows 10/11, per-user only (no admin rights); the MSIX build declares associations in its manifest instead.
- **Data compatibility:** existing registrations, made without a saved backup, can only be removed, not restored.
- **Localization:** new messages in 13 languages.

### 3.3 Owner inputs (Approval gate)
- **Related tickets:** none.
- **Copy/tone policy:** the wording for "open Settings > Default apps to finish".
- **Validation level:** a manual check on Windows 11 with an existing user choice for `.epub`.

## 4. Current architecture context
Registration writes a program id, the context-menu verbs, open-with entries and, on request, the
per-extension default value. Unregistration deletes them. The state query reads the default value
only. Errors from several registration calls are discarded at the call site.

## 5. Proposed approach
### 5.1 Pillars / modules
- **Change notification:** Explorer is notified after every write.
- **Effective-handler query:** the user's choice record is read to decide what "is default" means; the UI shows "registered, not default" separately from "default".
- **Guided completion:** when a user choice blocks the change, the app opens the Default apps page for the extension.
- **Backup and restore:** the prior handler is saved at registration and restored on unregistration.
- **Error propagation:** partial and failed registrations are reported with the extension list.

## 6. Open questions / research items
1. **Open the Settings page automatically?**
   - **Question:** launch Default apps automatically, or only on a button?
   - **Status:** Resolved (2026-09-25): only on an explicit action. The GUI shows an "Open Default apps"
     button; the console asks `[y/N]` and only when it is a terminal, because the GUI runs `-register`
     as a child with no one to answer.

## 7. Risks
- **Reading the user-choice keys differs across Windows builds.** Likelihood: low. Impact: a wrong indicator. Mitigation: fall back to "unknown" rather than "yes".

## 8. User impact (docs)
README: how default-app registration works on Windows 10/11.

## 9. Architecture decisions (ADR)
No ADRs. The decision follows the OS-mandated association model.

## 10. Links to other specs
None.

## 11. Done criteria (strategic)
1. With an existing user choice for `.epub`, "make default" does not claim success, and it guides the user.
2. Register, then unregister, restores the previous `.epub` handler.
3. A forced registry write failure on first run is shown to the user.

## 12. Next step
`/spec-tech 20_2026-09-24_bugfix-windows-registration-honesty`

## 13. Implementation record (2026-09-25)

| Finding | Change |
|---|---|
| P12 | `RegisterHandler` returns a `Registration` (`Default` / `Blocked` / `Unknown` / `Failed`) read back from what Windows uses, not from what was written. The CLI prints `DONE` and the double-click line only when every type is default; otherwise `INCOMPLETE`, each list, and the Settings > Default apps step. Explorer gets `SHChangeNotify(SHCNE_ASSOCCHANGED)` after any write that changed a value; the launch-time rewrites skip it when nothing changed. |
| P12 (GUI) | `HandlerStatus` reads `FileExts\<ext>\UserChoiceLatest` (Windows 11 24H2, ProgId on the key or in a `ProgId` subkey) before `UserChoice`; an unreadable choice is `Unknown`, never "yes". `Applications\doc-html-translate.exe` (Open with -> Always) counts as ours. `/api/assoc-status`, `/api/register` and `/api/unregister` return `state` (`default` / `blocked` / `unknown` / `partial` / `none`) and the lists; the page tells "registered, not default" apart and offers the button (`/api/open-default-apps`). |
| P13 | Registering saves the replaced per-user handler in `Software\Classes\<ext>` value `doc-html-translate.previous`; unregistering restores it, or deletes the default when none was saved (nothing there, or a registration made before the backup). A backup for a type someone else took since is dropped. Every write and delete error is collected; the error names the extensions that could not be released. |
| P14 | First run reports a failed "Open with" or right-click registration; `-register` and `-register-openwith` report a failed right-click entry; a failed default-handler registration prints its error and the per-extension result. |

Messages: 14 CLI strings and 6 GUI keys, 13 languages each. The user's own choice is never written or
deleted: Windows protects it with a hash (non-goal).

Tests: `register_windows_test.go` (a blocking user choice, the Windows 11 key, an unreadable choice,
register-then-unregister restores the previous handler, a stale backup, a failed release, no Explorer
notification on an unchanged rewrite) - run under Wine; `internal/app/registration_test.go` (the printed
result never claims success when blocked).

Not done / open:
- Validation level of §3.3: the manual check on Windows 11 with an existing user choice for `.epub`.
- The Default apps page opens at its top; a per-extension deep link needs a `RegisteredApplications`
  capabilities entry, which this app does not write.
