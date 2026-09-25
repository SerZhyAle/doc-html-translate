# The desktop window behaves and looks like the rest of the portfolio

**Status:** Draft
**Priority:** 53
**Date:** 2026-09-23

> Contract sync ticket, both directions.
> Contracts: `APP-BEHAVIOUR` 0.9, `APP-STYLE` 0.9 (domain `desktop-app-ux/`, owner StreamsPlayer, drafts).
> Both are 0.10 in the catalog since 2026-09-24 (rule 1 narrowed to secondary windows; `APP-STYLE` section 5
> content-coloured-window amendment; doc-html-translate added to both `consumers` lists).
> Pointers now exist in [`docs/contracts/`](../../docs/contracts/) (see "Re-verified 2026-09-25").

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

## Done criteria

- [x] Pointer files `docs/contracts/APP-BEHAVIOUR.md`, `APP-STYLE.md` exist and are listed in the pointer
      README. (Verified 2026-09-25; their text still needs Direction A step 10.)
- [ ] **⛔ Local only - changes the contract catalog.** Registry: this product's consumer rows for both ids with the rule-by-rule result; every deviation
      still open is a dated exception. The 2026-09-24 row exists but claims conformance the code does not
      have; it is rewritten from the result of Direction A.
- [ ] No registry write and no network install happens without an answer the user gave in that session.
- [ ] A running conversion can be cancelled from the window; closing the window mid-run leaves no orphaned
      CLI process (checked with a running window, not only by reading).
- [ ] The paid-translation question defaults to "no" and has an Escape path.
- [ ] No failure surface shows a raw error string.
- [ ] The GUI offers system / light / dark and follows the OS live; `--fg` is declared.
- [ ] **⛔ Local only - changes the contract catalog.** B1-B8 filed or withdrawn in writing here.
- [ ] Every new user-visible string exists in all 13 GUI languages.

## Open questions

1. **⛔ Local only - changes the contract catalog.** Registry convention: implements `0.9 (partial)` or `-` (not adopted, as `INSTALL-TRUST` was declared)?
2. Shell integration: make it strictly opt-in, or argue (B4) that registering the app's own verbs is its
   own state? The code half does not wait: implement strictly opt-in (Direction A step 1); arguing B4 is
   **⛔ Local only - changes the contract catalog.**
3. Cancel semantics: what does "finished work kept" mean for a half-written output folder, given the
   invariant that an existing `index.html` is reused unless `-force`?
4. Replace native `alert`/`confirm` with in-page dialogs (owner-centred, safe default) everywhere, or only
   for the destructive and costly ones?

## Notes

Changes the behaviour of a shipped invariant in two places (first-run registration, cost confirmation);
both are user-visible and each needs the docs/site line that describes it updated in the same edit.
`main.go` (1082 lines) and `ui.html` (1226) are the only two files most items touch - split before adding.

## Contract snapshot (2026-09-25)

Quoted from the catalog on 2026-09-25 - `APP-BEHAVIOUR` 0.10, `APP-STYLE` 0.10. A working copy for executing this ticket without the catalog, not a second source: the catalog stays authoritative, and this section is deleted when the ticket moves to done/.

What is quoted: each rule's **Rule** and **Why** paragraphs verbatim, plus the rule 1 amendment and the rule 2
Windows note. The **Evidence** paragraphs cite the owner's WPF code (StreamsPlayer) and are omitted; so are
`APP-BEHAVIOUR` rules 10 and 12, which do not apply here (see "Findings"). **Palette values: the catalog
prescribes none** - "The values are each product's own" (`APP-STYLE` rule 4). The existing dark set is
`cmd/doc-html-ui/ui.html:10-21`; the light set is this product's choice, named by the role vocabulary below.

### `APP-BEHAVIOUR` 0.10

**1. One no-action exit from every secondary window [CONTRACT]**

> **Rule.** Every secondary window - one that asks, collects or confirms something and then goes away - is
> modal, is owned, and opens centred on its owner. It
> offers exactly one path that leaves without changing anything, and Escape, the close box and the Cancel
> button are that same path - not three implementations of it. Enter confirms the recommended answer.
>
> **Why.** A user who does not know what a dialog will do needs to be able to get out of it before they
> find out. Where the three exits are written separately they drift, and the one that drifts is usually
> Escape, which is the one a hesitant user reaches for first.
>
> **Amended 2026-09-24 (0.10): what a secondary window is.** The first version said "every window other
> than the main one". The owning product never held that, and should not: its player window and its
> compact panel are **companion surfaces** - windows whose purpose is to stay open beside the main one
> while the user keeps using it - and making either modal would lock the main window for as long as
> something plays. A companion surface is outside this rule. Everything that asks, collects or confirms
> is a secondary window and is bound in full. The narrowing makes no shipped product wrong.
>
> **What a second implementation shows.** Open every secondary window; press Escape in each; nothing
> changed and the window is gone.

**2. Growing text grows the window [CONTRACT]**

> **Rule.** A surface whose text can change length sizes itself to its content. Truncation with an ellipsis
> is not an answer to translation.
>
> **Why.** The same sentence in thirteen languages is thirteen lengths, and the longest is routinely twice
> the shortest. A layout fixed to the language it was designed in fails in the languages nobody on the team
> reads, which is exactly where nobody will notice.
>
> **What a second implementation shows.** The longest shipped language, on every dialog, with nothing cut.

**3. A long operation: progress, cancellation, and no second start [CONTRACT]**

> **Rule.** An operation the user waits for reports real progress, can be cancelled, and blocks its own
> re-entry while it runs. **Cancelling is an outcome, not a failure**: it gets its own message, and work
> already completed is kept rather than discarded. The wait cursor is shown only when the operation cannot
> be cancelled.
>
> **Why.** The last clause is the one that is got wrong. A wait cursor beside a live Cancel button tells the
> user the application is not listening, at the exact moment they are trying to tell it something.
>
> **What a second implementation shows.** Start the longest operation, cancel it halfway: a message that is
> not an error, partial work retained, and the operation startable again.

**4. Network and disk only because the user asked [CONTRACT]**

> **Rule.** Nothing is downloaded, uploaded or written outside the application's own state because the
> application decided to. Consent is asked each time it is needed. **A record that consent was once given
> is a record of what happened, never a licence to stop asking.**
>
> **Why.** The second sentence is the one that erodes. A stored "the user agreed last time" becomes a
> background fetch within two releases, and by then nobody can point at the decision that made it one.
>
> **What a second implementation shows.** A packet capture over a cold start that touches nothing until the
> user acts.

**5. Confirm the irreversible; do not confirm the empty [CONTRACT]**

> **Rule.** An action that cannot be undone is confirmed before it happens. An action with nothing to do
> reports that, and does not ask for confirmation of nothing.
>
> **Why.** Confirmations work only while they are rare and always mean something. A dialog that appears
> when nothing is at stake trains the user to dismiss the one that matters.
>
> **Note.** This product has no undo anywhere, which is why confirmation carries the whole weight here. A
> product that does have undo may legitimately confirm less; that is a narrowing to record in its registry
> row, not a deviation.

**6. A failure is a set of actions [CONTRACT]**

> **Rule.** When something the user asked for fails, they are offered what to do next, not told what went
> wrong. The raw exception goes to the log; the user gets a named cause and actions. The application ships
> a way for the user to send that log to its author.
>
> **Why.** "An error occurred" ends the user's session. Retry, report, remove, leave continues it, and
> three of those four are things only the user can decide.

**7. Localized rendering cannot end the process [CONTRACT]**

> **Rule.** Formatting a user-facing string never throws. Each shortfall has a documented degradation: a
> missing translation renders the key, a missing argument renders a blank, an unparseable template renders
> itself.
>
> **Why.** These strings are formatted inside event handlers that cannot return an error to anybody. An
> exception there is not a bad message - it is the application closing while the user watches.
>
> **What a second implementation shows.** Delete a key from a shipped language and run: the interface is
> ugly and alive.

**8. Layout direction belongs to the language, and is gated [CONTRACT]**

> **Rule.** Whether the interface runs right-to-left is declared once, beside the language, and read from
> there by every window. It is verified by an automated gate against the language registry, not by looking.
> **Media-transport glyphs are pinned left-to-right.**
>
> **Why.** Direction set per window is direction that is wrong in one window. And a mirrored Play triangle
> does not read as Play pointing the other way - it reads as Rewind, which is a different function.
>
> **Note.** A surface that leaves the window's visual tree - an overlay hosted outside it - does not inherit
> direction and must set it on itself. That is a consequence, not a second rule.

**9. A glyph-only control carries an accessible name, and it is gated [CONTRACT]**

> **Rule.** Any control whose only label is an icon carries an accessible name. The name follows the
> control's current role, and survives a language change. An automated gate fails the build when a name is
> missing or disagrees with the visible caption.
>
> **Why.** Two reasons, and the second is the one that gets this rule implemented. A screen reader needs it.
> And **without accessible names nothing outside the process can drive the interface**, so run-and-observe
> verification - the evidence standard this portfolio holds itself to for a changed GUI action - becomes
> impossible to automate.
>
> **[PROPOSED]** Tab order, Windows text scaling and high-contrast mode. No product in this portfolio has
> evidence of any of the three, and inventing a rule for them here would be writing a wish. The honest
> state is that this contract covers naming and nothing else about accessibility.

**11. First run asks, and "nothing" is a real answer [CONTRACT]**

> **Rule.** The first launch asks before doing anything that reaches outside the machine, and one of the
> offered answers is to do nothing. That answer leaves a working application. **A control that cannot work
> in this build is hidden, not disabled.**
>
> **Why.** A first launch that starts downloading has already broken rule 4 before the user has seen the
> product. And a greyed-out control with no explanation is not caution - it is the first impression that
> something is broken, from a user who has no way to find out otherwise.

### `APP-STYLE` 0.10

**2. Three themes, switched without a restart [CONTRACT]**

> **Rule.** The application offers **system, light and dark**, defaulting to system. A change applies
> immediately, without a restart. While the preference is system, the application follows the operating
> system's own setting **and reacts to it changing while the application is running**.
>
> **Why.** The last clause is what separates a theme setting from a theme. An application that reads the
> system preference once at start is dark until tomorrow, in the middle of a system that went light an hour
> ago.
>
> **Windows.** The preference is read from the Personalize key's `AppsUseLightTheme` value and the change
> is observed through the system's user-preference notification.

(For this browser-hosted GUI the live OS signal is `prefers-color-scheme` - Direction A step 8; that it
counts is what B2 asks the catalog to say.)

**3. One palette table, and only dynamic references [CONTRACT]**

> **Rule.** Every themed colour is declared once, in a single named table, as a pair - one value per theme.
> Every use of it resolves **dynamically**, so that replacing the table reaches every element already on
> screen. A reference resolved once at load is a defect, not a style choice: it removes that element from
> the theme permanently, and the removal is invisible until somebody switches theme and looks at that one
> control.
>
> **Why.** This is the rule whose violation is hardest to find later, because the element keeps working and
> simply stops changing. It has to be a rule rather than a habit for the same reason.

**4. The role vocabulary [CONTRACT]**

> **Rule.** These are the names. A product that has such a surface calls it this; a product that does not
> have one owes nothing. **The values are each product's own** - this vocabulary exists so that two
> implementations can be read against each other, not so that they look identical.
>
> | Role | What it is for | StreamsPlayer's spelling |
> | --- | --- | --- |
> | `surface.window` | the window's own background | `WindowBackground` |
> | `surface.raised` | a panel or card sitting on the window | `SurfaceBrush`, `CardBrush` |
> | `surface.sunken` | a grouped or inset region | `SectionBrush` |
> | `surface.selected` | the raised surface, selected | `CardSelectedBrush` |
> | `control` | an interactive control's own face | `ControlBrush` |
> | `control.hover` | the same face under the pointer | `ControlHoverBrush` |
> | `control.pressed` | the same face while pressed | `ControlPressedBrush` |
> | `border` | every separating line | `BorderBrush` |
> | `text.primary` | body text | `TextBrush` |
> | `text.muted` | secondary text that must still be read | `MutedBrush` |
> | `text.disabled` | text of an unavailable control | `DisabledTextBrush` |
> | `accent` | the primary action's fill | `AccentBrush` |
> | `accent.ink` | text drawn on the accent fill | `AccentTextBrush` |
> | `link` | a hyperlink | `LinkBrush` |
> | `info` | a neutral informational mark | `InfoBrush` |
>
> **[PROPOSED] - `danger`, `warning`, `success`.** No desktop product in this portfolio has any of the
> three, and the gap is not cosmetic: `APP-BEHAVIOUR` rules 5 and 6 - confirm the irreversible, offer a
> failure as actions - are today carried by wording alone, with a destructive button that looks exactly
> like the button beside it. The web contract already has all three (`--danger`, `--warn`, `--ok`), which
> is the argument that they belong in the vocabulary rather than in each product's taste. What closes this:
> one product adding them and reporting what the confirmation dialog looked like afterwards.

(Since 2026-09-24 the owner uses all three for state marks; they stay proposed "until a product that draws
its own confirmation uses `danger` there" - an in-page confirm here, Open question 4, would be that product.
The catalog's `APP-STYLE` rule 6 maps the shared names to the web page's CSS: `surface.window` = `--bg`,
`text.primary` = `--text`, `text.muted` = `--muted`, `accent` / `accent.ink` = `--acc` / `--acc-ink`,
`border` = `--border`.)

**5. A surface deliberately outside the theme declares itself [CONTRACT]**

> **Rule.** A surface that must not follow the theme says so where it is defined, and carries its own text
> colour with it. It is never left to inherit and quietly diverge.
>
> **Why.** There is always at least one: something drawn over live content, or over an image, where the
> window's own background is not behind it and a themed text colour would be unreadable half the time. The
> rule is not that this is forbidden - it is that it is declared, so the next reader knows the exception was
> a decision.

(The 0.10 amendment to this section - a whole window may be content-coloured - covers content viewers and
does not apply to `doc-html-ui`; the green log console is a surface under the rule above.)
