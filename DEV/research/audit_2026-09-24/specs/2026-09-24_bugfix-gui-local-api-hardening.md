# Strategic spec: 2026-09-24_bugfix-gui-local-api-hardening - GUI local server: authenticated, cancellable, leak-free

**Ticket:** 2026-09-24_bugfix-gui-local-api-hardening
**Status:** Draft
**Priority:** 90
**Date:** 2026-09-24
**Tier:** Security/Compliance (urgent)
**Tactical plan:** `DEV/plan/2026-09-24_bugfix-gui-local-api-hardening/` (created by /spec-tech)
**Findings:** G1 G3 G5 G6 G7 G8 G10 G12 G14 G15 G16 G17 G18 G19 (see `../README.md`)

> **Scope:** STRATEGIC.

---

## 1. Problem
The GUI is a local web server on a random loopback port, with a page that drives it. Any web page
the user visits can send requests to that server. Some endpoints act on a plain GET, and the JSON
endpoints accept any content type. A hostile page can therefore run conversions, delete output
folders, write large files, replace the saved API key, or change file associations. No token, no
Origin check and no Host check stand in the way. Beyond security, the conversion stream has two
writers racing on one response and can hang on one long output line. A conversion cannot be
cancelled, it may open a stray console window, a minimized GUI can shut itself down, and several
form fields forward values the CLI rejects.

## 2. Goals
1. Only the GUI's own page can call its API. Cross-site requests and DNS-rebinding requests are refused.
2. State-changing endpoints accept only the intended method and content type.
3. The conversion log stream is well-formed under any stdout/stderr interleaving and any line length.
4. A running conversion can be cancelled from the GUI, and closing the GUI stops its child.
5. One output location has at most one conversion at a time from the GUI.
6. No console window appears during a GUI conversion.
7. A GUI left minimized for any length of time is still alive when the user returns.
8. Settings survive a crash mid-save, and concurrent instances do not lose each other's updates.
9. Only values the CLI accepts are forwarded. Empty fields mean their documented GUI default, and engine-specific fields are sent only for the selected engine.
10. The UI tells the user when a multi-file drop used only one file, and when a run request failed.

**Non-goals:**
- The open-path injection and flag separator (ticket `bugfix-shell-open-injection`).
- Delete ownership and drop naming (ticket `hotfix-output-dir-ownership`).

## 3. Wishes and constraints
### 3.1 Owner wishes
- Batch conversion of a multi-file drop (later; for now only a warning).

### 3.2 Hard constraints
- **Platform / versions:** Windows; the page is served to the user's default browser (Chrome/Edge).
- **Performance:** n/a.
- **Data compatibility:** the settings file format is unchanged; only the write becomes atomic.
- **Localization:** new UI strings in all 13 languages (the GUI dictionary coverage test enforces this).
- **Accessibility:** a cancel control reachable by keyboard.

### 3.3 Owner inputs (Approval gate)
- **Related tickets:** `bugfix-shell-open-injection`, `hotfix-output-dir-ownership`, `bugfix-ocr-language-data-and-detection` (the language-code validation reachable via the GUI).
- **Localization:** new strings for cancel, "only the first file was used" and the request-failure line.
- **Validation level:** automated tests for token/Origin rejection, the stream race (`-race`), a long line, and cancellation; a manual Windows check for the console window and the minimized-heartbeat case.
- **Owner sign-off:** required (security).

## 4. Current architecture context
One process serves the static page and about 25 JSON endpoints on an unauthenticated mux. A
conversion spawns the CLI as a child and relays its two output pipes line by line into a streaming
response, from two goroutines. Liveness is a page heartbeat with a short server-side grace period.
Settings and output history are small JSON files rewritten in place.

## 5. Proposed approach
### 5.1 Pillars / modules
- **Request authentication:** a per-launch secret delivered only inside the served page and required on every API call. Host must be the loopback address and port, and Origin must be the app origin. State changes require POST with a JSON content type.
- **Single-writer log relay:** both child streams feed one ordered channel with one writer. No line-length limit, and draining continues after any read error so the child never blocks.
- **Run lifecycle:** the child is tied to the request and to a cancel action. The child's process tree is terminated on cancel, on a dropped request and on GUI exit. There is one active run per output location.
- **No console:** the child is created without a console window.
- **Liveness:** a signal that survives background-tab throttling (for example a long-lived connection whose closing means the page is gone), or a grace period well above the throttled timer interval.
- **Durable settings:** atomic replace on write, serialized updates, and a corrupt file is reported, not silently reset.
- **Argument hygiene:** per-field validation. Engine-specific fields are sent only when that engine is chosen, and an empty value maps to the GUI default.
- **Page robustness:** the saved engine is matched against the known set without building selectors, a failed response is shown as an error, and the log follows new output only when the user is already at the bottom.
- **Parity test depth:** the UI-to-CLI test also feeds empty and non-numeric field values through the real CLI parser.

### 5.2 Data & event flows
Page (holds the secret) -> authenticated API -> run manager (one per output) -> child process ->
single-writer relay -> page. Cancel / close -> run manager -> process-tree kill.

## 6. Open questions / research items
1. **Liveness mechanism**
   - **Question:** a long-lived connection close, or a longer grace period?
   - **To find out:** check Chrome's intensive throttling behaviour against the chosen approach.
   - **Status:** Open.
2. **Process-tree kill on Windows**
   - **Question:** use a job object, or kill the child recursively?
   - **Status:** Open.

## 7. Risks
- **The token breaks the GUI when the page is reloaded from history.** Likelihood: medium. Impact: the GUI stops working until restart. Mitigation: the page fetches the token from a same-origin bootstrap that checks Host and Origin.
- **Cancelling mid-write leaves a partial output.** Likelihood: high. Impact: a broken book. Mitigation: depends on `bugfix-output-completeness` (atomic output).

## 8. User impact (docs)
README (GUI section): a Cancel button, and a note that multi-file drop uses the first file.

## 9. Architecture decisions (ADR)
**ADR-1: a per-launch secret plus Origin and Host checks.** Alternatives: Origin only, which is weak against rebinding and non-browser local callers. Why: defence in depth at negligible cost.

## 10. Links to other specs
`bugfix-shell-open-injection`, `hotfix-output-dir-ownership`, `bugfix-output-completeness`.

## 11. Done criteria (strategic)
1. A request from another origin, or without the secret, to any state-changing endpoint is refused, and nothing changes.
2. A child that writes interleaved stdout/stderr and a 1 MB line produces a complete log, and the run finishes.
3. Pressing Cancel stops the conversion within seconds, and no converter process remains.
4. No console window appears during a GUI conversion.
5. The GUI still responds after being minimized for 10 minutes.
6. Clearing the Ollama fields or the Split field and converting with Google works.

## 12. Next step
`/spec-tech 2026-09-24_bugfix-gui-local-api-hardening`
