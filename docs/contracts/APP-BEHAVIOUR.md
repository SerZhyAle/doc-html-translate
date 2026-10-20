# Pointer: APP-BEHAVIOUR

- **Id:** `APP-BEHAVIOUR`
- **Version:** 0.12 draft
- **Home:** the shared contracts catalog, `desktop-app-ux/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - GUI launcher behaviour (`cmd/doc-html-ui`), plus the CLI's native dialogs a GUI user meets
- **Wire carrier:** none - desktop application user interaction

What the GUI holds, rule by rule (the GUI is an HTML page in an Edge/Chrome `--app` window; the WPF
wording of the rules is read through that - see ticket 23, Direction B):

- **Rule 1 - one no-action exit.** Every question and notice is one in-page modal `<dialog>` (no
  `alert`/`confirm`): centred on the window, Escape / close / the cancel button are one path, and the
  recommended answer has the focus. The paid-translation cost question is asked in that dialog when the
  GUI runs the converter (marker protocol, `internal/dialog` `HostStdio`); run from a console, it is an
  owned OK/Cancel box whose default and Escape path is "do not spend".
- **Rule 2 - text grows the window.** Dialogs and rows size to their text; only the output path ellipsizes.
- **Rule 3 - progress, cancel, no second start.** A conversion has Cancel (kills the whole process tree;
  closing the window does too), one run per output folder, and "cancelled" is its own outcome. What was
  already written stays in the output folder and is rebuilt by the next run, because a cancelled run
  writes no completion record. The OCR-pack download shows progress and can be cancelled.
- **Rules 4 and 11 - nothing unasked; "nothing" is an answer.** The GUI writes no registry entry on its
  own: the right-click entry and "Open with" are added on the first-run question's yes or the toggle
  under "Windows integration"; the CLI's no-arg flow asks before it writes. Nothing is installed.
- **Rule 5 - confirm the irreversible, not the empty.** Deleting a previous result and clearing the
  stored logs and rebuilding existing output are confirmed, with Cancel focused; an empty log store is reported as such.
- **Rule 6 - a failure is a set of actions.** The page words each failure as a named cause and what to
  do; the raw error goes to the GUI's own log in the run-log store, which "Send logs to the author" packs.
  A failed conversion offers send logs / try again / close.
- **Rule 7 - rendering never throws.** A missing string falls back to English, then to the key.
- **Rule 8 - direction per language, gated.** `RTL_LANGS` in the GUI dictionary is checked against
  `internal/i18n.IsRTL` by a test.
- **Rule 9 - accessible names, gated.** Glyph-only controls and every select carry a localized
  `aria-label`; the drop zone is a real button. A test fails on a missing name.
- **Rule 10 - window geometry.** The GUI saves and restores its rectangle (since 2026-09-29, ticket 63): the
  restore is defensive (`saneGeometry`, `clampWindowToScreen` - a lost monitor lands on a surviving one) and a
  maximized or full-screen box never overwrites the remembered normal rectangle (`windowFillsScreen`, pinned in
  `session_test.go`, 2026-10-20). The one difference is timing: the rectangle is written after the window
  settles (800 ms after the last move or resize), not on close, because a browser-hosted window has no
  reliable close event - a permanent variant recorded in [`../UI_PROFILE.md`](../UI_PROFILE.md) section 4 and
  in the catalog registry. (The earlier "no saved geometry here" was false since 2026-09-29.)
- **Rule 12** applies to the autosaving settings surface: value edits are reversible; deleting output and logs
  keeps a separate confirmed button.

**Re-checked 2026-10-06 (ticket 95):** a second pass found a run could start twice from a double click
(rule 3 - now blocked synchronously; a cancelled queue item is logged as its own outcome), the persisted
recent-documents history cleared without a confirmation (rule 5 - now confirmed, `danger`, safe answer
focused), a failure worded as an exit code (rule 6 - every converter exit code maps to a named cause,
an unknown one to "send the logs"), a missing format argument rendered as `{k}` (rule 7 - a blank),
captions that named no control and text buttons announcing a different sentence (rule 9), and the
auto-update hint left visible in the MSIX build (rule 11). All fixed and gated in
`cmd/doc-html-ui/conformance_test.go` and `contract_test.go`.

**Open, recorded in the catalog (2026-09-28):** the right-click verb's own caption ("Convert to HTML")
is English in Explorer - a dated exception in the shared registry (until 2026-12-31). The console-dialog and own-state-repair proposals are settled by 0.12: console dialogs stay bound;
repair needs no new consent only within already enabled HKCU registration kinds. This product has no
automatic repair path: the no-argument CLI asks again, and explicit registration flags authorize writes.

**Read again 2026-10-20 (ticket 120)** at 0.12, which the consolidation of 2026-10-17 did not change: rules 1-9,
11 and 12 were not re-read line by line, the whole package passes (`go test ./cmd/doc-html-ui`, and
`ui_dialog_test.mjs` with `help_resolver_test.mjs`), and the dialog guards hold.

**Conformance.** `cmd/doc-html-ui/contract_test.go`, `cmd/doc-html-ui/conformance_test.go`, `cmd/doc-html-ui/hardening_test.go`
(`TestQuestionIsAskedInTheWindow`, the cancel tests), `cmd/doc-html-ui/session_test.go` (the geometry guard),
`internal/dialog` tests, and a headless-Edge run of the real page (ticket 23, "Verification").
