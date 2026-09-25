# Strategic spec: 02_2026-09-24_bugfix-shell-open-injection - Open files and pass paths without a shell interpreting them

**Ticket:** 02_2026-09-24_bugfix-shell-open-injection
**Status:** Draft
**Priority:** 90
**Date:** 2026-09-24
**Tier:** Security/Compliance (urgent)
**Tactical plan:** `DEV/plan/02_2026-09-24_bugfix-shell-open-injection/` (created by /spec-tech)
**Findings:** P3 (= G2) P24 G4 G13 (see the [findings register](../research/audit_2026-09-24/README.md))

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
   - **Status:** Open.

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

## 12. Next step
`/spec-tech 02_2026-09-24_bugfix-shell-open-injection`
