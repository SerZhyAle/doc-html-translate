# Pointer: APP-COMMANDS

- **Id:** `APP-COMMANDS`
- **Version:** 0.3 (draft; owner shared)
- **Home:** the shared contracts catalog, `app-commands/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, `partial` - the command line of both executables and the launcher first instance, adopted 2026-11-14 (ticket E0143), aligned with 0.2 and 0.3 item V on 2026-11-16 (ticket E0149), and completed for the 16 ship-first verbs on 2026-11-16 (ticket E0144); the 0.1 research was ticket E0142
- **Wire carrier:** local named pipe `\\.\pipe\DocHtmlTranslate_Command_{userSid}_{sessionId}` (`APP-ACTIVATION` rule 3, third shape; mutex `Local\DocHtmlTranslateSingleInstance`), carrying one JSON Lines request and one JSON Lines response per connection (rules 6 to 10) for `status`, `show`, `quit` and `open`; the remaining 12 verbs are `forward: never` (0.2 item C)

What it covers: how a script, a scheduler or a development agent asks this product something by command line and
learns what exists - the grammar (rules 1 and 3), the answer vocabulary (rule 11), the manifest (rule 18) and the
reference rendered from it (rule 20).

**Where this product stands (2026-11-16, tickets E0149 and E0144).**

- Held: [`../cli-commands.json`](../cli-commands.json) is the manifest (`cliVersion` 1.0, 16 verbs); `internal/commands` embeds
  a copy and a test fails unless the two are byte-equal; `commands --json` prints those bytes;
  [`../CLI.md`](../CLI.md) is rendered from the manifest by `tools/clidoc` and held by the gate child `scripts/cli-doc.ps1`;
  both executables call `commands.Front` before touching a window, setting, worker or log; an unknown option or bad value
  exits 10 and runs nothing; an unreserved first word routes to `convert` (`doc-html-translate.exe`) or `open` (`doc-html-ui.exe`, `defaultFor`, 0.2 item A.2); a request over
  64 KiB or an argument over 32 KiB is refused, never cut. Option aliases (`free` on `convert` `ollama`) are retained
  for `covered()` / C13 only and ignored by canonical verb option parsing (0.2 item E.5).
- The launcher group (ticket E0144): `doc-html-ui.exe` holds the per-session mutex `Local\DocHtmlTranslateSingleInstance`
  and listens on `\\.\pipe\DocHtmlTranslate_Command_{userSid}_{sessionId}` (`APP-ACTIVATION` rule 3, third shape; skipped under
  `DOCHT_GUI_NO_WINDOW`). `status` (answers version, pid, `windowOpen`, `activeRuns`, `acceptsRestrictedCommands: false`; `coldStart: refuse` -> exit 11 `not-running`)
  and `quit [--cancel-runs]` (`coldStart: oneshot` -> exit 0 when not running; `busy` exit 11 while a conversion runs unless `--cancel-runs`)
  are forwarded by either executable; `show` (`coldStart: start`, raises the `"DOC-HTML-UI"` browser app window via `EnumWindows` and `AllowSetForegroundWindow` or opens a window when none is alive; Windows foreground refusal is never a failure, 0.2 item N.2)
  and `open --in <path>` (repeatable, `coldStart: start`, pushes paths into the live page queue over `/api/events` or pre-fills `/api/initial`, `defaultFor: "doc-html-ui.exe"`)
  carry `"exe": "doc-html-ui.exe"`. Bare `doc-html-ui.exe` is the `""` alias of `show` (`answers: false`).
- The convert group (ticket E0145): `convert <path> [options] [--result <path>]` is the verb of the product's one call and
  the bare call `doc-html-translate.exe [flags] <path>` is its frozen default alias (`defaultFor`); both run the same code.
  The verb never waits on a person (rule 4): `--google` needs a positive finite `--max-cost` and a `.fd-sec` input needs
  `--fdsec-password-env` (the name of a variable, never the secret, rule 5), else exit 10; the bare spelling keeps its cost
  dialog and password prompt. `--result` writes the data of the run whole or not at all (rule 9). `output status --in <path>`
  answers whether a finished result exists, is complete and would be reused, by the same `outputpath` code the window asks.
  Both are served by `doc-html-translate.exe` only (`forward: never`); `doc-html-ui.exe` says so and exits 10. The manifest
  members `forward`, `defaultFor` and `positional` are beyond the 0.1 text (rule 18 lets a reader ignore them).
- The inspection group (ticket E0146): `doctor [--ollama [--ollama-model <m>]]` says what this machine can do as data (the
  helper tools found with a version number, the input formats and the tool each needs, the installed OCR languages, whether
  a Google key is stored, and with `--ollama` only whether the local server answers and has the model; never a path, a
  document name or a key, and no request to any service without `--ollama`); `input check --in <path> [options]` is the
  window's pre-flight for one document as data (`state` and `issues`; a missing path exits 11, reason `target-missing`);
  `ocr langs` lists the catalogue with the installed ones marked and prints the listing the frozen alias `-ocr-langs`
  prints. The check moved from the window into `internal/readiness`, which the window and the verb both call. All three are
  `forward: never` and are served by both executables.
- The maintenance group (ticket E0147; folder `--out` aligned with 0.3 item V in ticket E0149): `ocr download --lang <code>` fetches one recognition language pack into the per-user
  folder, checked against its pinned digest (`downloaded: false` and no request when the verified pack is already there);
  `report build [--out <path>] [--overwrite]` packs the recent run logs and the environment summary and answers the archive's
  path (when `--out` names an existing folder or ends in a separator, it writes the `DIAGNOSTIC-REPORT` 8A file inside it;
  a missing folder is refused exit 11 `target-missing` and never created; an existing target file is refused exit 11
  reason `exists` unless `--overwrite`, which skips only that refusal; exit 13 when the disk is full; no document name is
  in the archive); `integration add` adds the "Open with" entry and
  the right-click "Convert to HTML" entry, `default-handler set` also becomes the default handler, `default-handler release`
  hands the association back and leaves the other two. All five are `forward: never`, never prompt, offer Settings or wait for
  Enter (rule 4), and are served by `doc-html-translate.exe` only (the launcher says so, exit 10). Under a Store package the three
  registry verbs are refused, exit 11, reason `packaged` (the product's own token, rule 12), and change nothing. The shipped
  flags keep their prompts exactly (`internal/app/app.go` is not edited).
- Frozen aliases (rule 2, 0.2 items E.1-E.4, F.1): all 12 pre-adoption console spellings (`-h`,
  `-help`, `--help` for `commands`; `-version`, `--version` for `version`; `-ocr-langs` for `ocr langs`; `-ocr-download`
  (`answers: false`) for `ocr download`; `-report` for `report build`; `""` (`answers: false`) and
  `-register-openwith` (`answers: false`) for `integration add`; `-register` (`answers: false`) for `default-handler set`;
  `-unregister` for `default-handler release`) plus the bare `""` launch (`exe: "doc-html-ui.exe"`, `answers: false`) for `show`.
- Open (dated exceptions in the catalog registry): rule 16 (verbs write no structured app-commands log); rule 18 member
  `positional` (unnamed in 0.2/0.3).

Everything the catalog accepted from `PROPOSAL-2026-11-14-doc-html-translate-research-gaps` and
`PROPOSAL-2026-11-15-doc-html-translate-bundle-folder-name` is in `APP-COMMANDS` 0.2 and 0.3 section 8 item V (`report
build --out <folder>` and `reason=exists`). Remaining unit-test runs in `internal/commands` and `internal/report` wait
for the release tier.
