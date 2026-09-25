# Strategic spec: 2026-09-24_bugfix-shell-open-injection - Open files and pass paths without a shell interpreting them

**Ticket:** 2026-09-24_bugfix-shell-open-injection
**Status:** BlockNeedUserTest - implemented and covered by tests (the Windows open test compiles on Linux, runs only on Windows); needs the manual open check on Windows with Chrome and with Edge as the default browser, and a paste of the GUI command into PowerShell 5.1 and 7.
**Priority:** 90
**Date:** 2026-09-24
**Tier:** Security/Compliance (urgent)
**Tactical plan:** `DEV/plan/2026-09-24_bugfix-shell-open-injection/` (created by /spec-tech)
**Findings:** P3 (= G2) P24 G4 G13 (see the [findings register](../../research/audit_2026-09-24/README.md))

> **Scope:** STRATEGIC.

---

## 1. Problem
The final "open in browser" step, run by default after every conversion and from the GUI's open
buttons, hands the output path to the Windows command interpreter. The interpreter treats `&`, `|`,
`^`, `%VAR%` and similar characters as syntax. A book downloaded as `x&calc&.epub` can therefore
run a command, and an innocent folder name like `Q&A` just fails to open. Separately, the GUI passes
the input path to the CLI in a way that lets a path starting with `-` be read as a flag. The command
line it shows the user for copy-paste is also not valid in PowerShell or cmd for some paths.

## 2. Goals
1. Opening an output never goes through a command interpreter; any legal file name opens correctly.
2. The GUI-to-CLI hand-off always treats the input as a file, whatever its first character.
3. The command line the GUI displays can be pasted into PowerShell and runs as shown.
4. Launched opener processes are released, so they leave no handles behind.

**Non-goals:**
- Securing the GUI's local API (ticket `bugfix-gui-local-api-hardening`).

## 3. Wishes and constraints
### 3.2 Hard constraints
- **Platform / versions:** Windows 10/11, both the 32-bit and 64-bit builds. Must open in the user's default browser, as today.
- **Data compatibility:** n/a.
- **Localization:** n/a.

### 3.3 Owner inputs (Approval gate)
- **Related tickets:** `bugfix-gui-local-api-hardening` (flag injection is only remotely reachable while CSRF is open).
- **Platform constraints:** the fix must keep the MSIX and Store builds working, because shell APIs are available there.
- **Validation level:** a test with a filename set covering `& ^ % ! ( ) , ;`, spaces, a trailing backslash, and Cyrillic, plus a manual open check on Windows.
- **Owner sign-off:** not required beyond review.

## 4. Current architecture context
Opening is a thin Windows-only wrapper that starts the command interpreter with the target as an
argument and never waits. The GUI builds the CLI argument list itself and appends the input last,
without an end-of-flags marker, even though the CLI parser supports one. A separate helper quotes
arguments for display only.

## 5. Proposed approach
### 5.1 Pillars / modules
- **Direct shell-open:** open the target through the OS "open document" facility, which takes the path as data, not as a command line. The non-Windows side stays symmetric.
- **End-of-flags marker:** the GUI always separates the flags from the input.
- **Display quoting:** one quoting routine that is correct for PowerShell. It handles an embedded quote, a trailing backslash inside quotes, and the call operator before a quoted executable.
- **Process hygiene:** any helper process that is launched is released or waited on.

### 5.2 Data & event flows
Pipeline or GUI -> open request (a path as data) -> OS shell -> default browser.

## 6. Open questions / research items
1. **Shell API choice**
   - **Question:** ShellExecute via the system library, or the URL file protocol handler?
   - **To find out:** check MSIX behaviour and the behaviour of a file URL with a fragment.
   - **Status:** Resolved (2026-09-25). `ShellExecute` from `golang.org/x/sys/windows`, verb `open`, the path as the file argument and no parameters. The opener never passes a fragment, so the file-URL question does not arise; full-trust MSIX apps may call it.

## 7. Risks
- **Some browsers ignore the default-app association when opened via ShellExecute.** Likelihood: low. Impact: the wrong browser opens. Mitigation: a manual check with Chrome as default and with Edge as default.

## 8. User impact (docs)
No changes to user docs.

## 9. Architecture decisions (ADR)
**ADR-1: no interpreter in the open path.** Alternatives: escaping for cmd. Why: cmd escaping rules are context-dependent and have repeatedly been bypassed; removing the interpreter removes the class of bug.

## 10. Links to other specs
`bugfix-gui-local-api-hardening`.

## 11. Done criteria (strategic)
1. An EPUB named `a&calc&b.epub` converts and opens, and no other program starts.
2. `Q&A.epub` and `50%.epub` open correctly.
3. A GUI input path starting with `-` is converted as a file.
4. The command line copied from the GUI runs unchanged in PowerShell.

## Implementation notes (2026-09-25)
- `internal/browser` opens through `ShellExecute`; no `cmd.exe`, and no child process to wait on or release. The GUI's default-browser fallback goes through the same call. `internal/browser/browser_windows_test.go` stubs the shell and checks that `& ^ % ! ( ) , ;`, a space, Cyrillic and a trailing backslash reach it verbatim.
- The GUI puts `--` before the input and drops a trailing `\` from the input and output folder (a root like `C:\` keeps it), so the argument means the same and no trailing backslash sits inside quotes. Covered by `cmd/doc-html-ui/cmdline_test.go`, which parses the GUI's own arguments.
- The displayed command is PowerShell: `& ` first, single quotes around anything PowerShell would interpret, a quote (including typographic ones) doubled, and `--` quoted because some PowerShell versions drop a bare one. It is no longer valid for cmd.exe, which the spec does not ask for.
- The GUI's Edge/Chrome app window, the macOS/Linux opener and the Explorer reveal release their process handle right after start (`startDetached`).

## 12. Next step
Manual check on Windows: done criteria 1 and 2 with Chrome and with Edge as default, and criterion 4 in PowerShell 5.1 and 7.
