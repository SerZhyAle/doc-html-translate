# Pointer: APP-BEHAVIOUR

- **Id:** `APP-BEHAVIOUR`
- **Version:** 0.10 draft
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
  stored logs are confirmed; an empty log store is reported as such.
- **Rule 6 - a failure is a set of actions.** The page words each failure as a named cause and what to
  do; the raw error goes to the GUI's own log in the run-log store, which "Send logs to the author" packs.
  A failed conversion offers send logs / try again / close.
- **Rule 7 - rendering never throws.** A missing string falls back to English, then to the key.
- **Rule 8 - direction per language, gated.** `RTL_LANGS` in the GUI dictionary is checked against
  `internal/i18n.IsRTL` by a test.
- **Rule 9 - accessible names, gated.** Glyph-only controls and every select carry a localized
  `aria-label`; the drop zone is a real button. A test fails on a missing name.
- **Rules 10 and 12** do not apply as written: the window keeps no geometry, and every control saves
  itself (no Save/Cancel settings window).

**Open, recorded in ticket 23:** the right-click verb's own caption ("Convert to HTML") is English in
Explorer; B1-B8 proposals to the catalog.

**Conformance.** `cmd/doc-html-ui/contract_test.go`, `cmd/doc-html-ui/hardening_test.go`
(`TestQuestionIsAskedInTheWindow`, the cancel tests), `internal/dialog` tests, and a headless-Edge run of
the real page (ticket 23, "Verification").
