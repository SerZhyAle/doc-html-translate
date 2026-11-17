# Pointer: APP-ACTIVATION

- **Id:** `APP-ACTIVATION`
- **Version:** 0.16 (draft; owner CyrFlip)
- **Home:** the shared contracts catalog, `app-activation/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - single-instance lock and local command pipe of the desktop launcher (`cmd/doc-html-ui`), adopted 2026-11-16 (ticket E0144)

What it covers: how `doc-html-ui.exe` holds one instance per user session (rule 2), transports `APP-COMMANDS` request-response lines over a local named pipe (rule 3, section 11), validates payloads strictly (rule 4) and restores its browser window without attaching input queues (rule 5, section 10 item A).

**Where this product stands (2026-11-16, ticket E0144).**

- **Transport:** named pipe (`\\.\pipe\DocHtmlTranslate_Command_{userSid}_{sessionId}`, rule 3 third shape; new name, no frozen anchor) and per-session mutex `Local\DocHtmlTranslateSingleInstance` (rule 2). No URI scheme is registered (rule 1 is not applicable).
- **Foreground choice (rule 5, section 10 item A):** the second process hands foreground rights over with `AllowSetForegroundWindow` (never `AttachThreadInput`) and bounded pipe I/O (`FILE_FLAG_OVERLAPPED`, 1 s delivery bound); the first instance raises its browser `--app` window by title (`EnumWindows`, `ShowWindow(SW_RESTORE)`, `BringWindowToTop`, `SetForegroundWindow`). Windows foreground refusal is never a failure (`APP-COMMANDS` 0.2 item N.2).
- **Profile (section 11):** `APP-COMMANDS` commands wait for the response line and exit with the answer's exit code; a bare shortcut launch (`""`) or file launch (`<path>`) forwards `show` / `open` to the running instance and exits without console output (`answers: false`).
- **Bypass for automation:** `DOCHT_GUI_NO_WINDOW` skips the single-instance lock so `tools/helpgen`, `tools/uicheck` and test servers can run side by side.
