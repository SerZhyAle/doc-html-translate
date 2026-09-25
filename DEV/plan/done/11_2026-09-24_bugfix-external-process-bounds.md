# Strategic spec: 11_2026-09-24_bugfix-external-process-bounds - Bounded helper processes and contained crashes

**Ticket:** 11_2026-09-24_bugfix-external-process-bounds
**Status:** BlockNeedUserTest - on Windows: (a) a hung helper leaves no process behind after its deadline (job object path), (b) after an app upgrade the new bundled pdftotext folder is used and the old one removed
**Priority:** 75
**Date:** 2026-09-24
**Tier:** Moderate
**Tactical plan:** `DEV/plan/done/11_2026-09-24_bugfix-external-process-bounds/` (created by /spec-tech)
**Findings:** X7 X9 X10 O5 O7 O9 P15 (see the [findings register](../../research/audit_2026-09-24/README.md))

> **Scope:** STRATEGIC.

---

## 1. Problem
The converter shells out to pdftotext, Tesseract, Calibre, 7-Zip, ffmpeg/ImageMagick and winget.
None of these calls has a time limit, so one pathological input hangs the CLI and the GUI forever.
When the bundled pdftotext is blocked, the converter silently starts a system-wide winget install
with auto-accepted agreements. The cached copy of pdftotext is never refreshed after an app upgrade,
can be half-written, and races between instances. Library panics during PDF image extraction or
OCR decoding kill the whole process, even though OCR is documented as best-effort. Tesseract's
internal threading on top of a CPU-sized worker pool oversubscribes the machine.

## 2. Goals
1. Every helper process has a deadline suited to its tool and input size. On expiry the process tree is killed, and the stage fails or degrades with a clear message.
2. Nothing is installed on the machine without the user's explicit consent.
3. The bundled helper cache is versioned, written atomically, and safe across concurrent instances.
4. A panic inside a third-party library during an optional or per-item stage is contained to that item, and the conversion continues where the design says it is best-effort.
5. OCR parallelism uses the CPU without oversubscription.

**Non-goals:**
- Memory budgets (ticket `bugfix-resource-budgets`).

## 3. Wishes and constraints
### 3.2 Hard constraints
- **Platform / versions:** Windows process-tree semantics; the MSIX sandbox (its cache location is under the user profile).
- **Performance:** deadlines must not fire on legitimately large inputs (§6.1).
- **Localization:** new messages (timeout, install advice) in 13 languages.

### 3.3 Owner inputs (Approval gate)
- **Related tickets:** `bugfix-gui-local-api-hardening` (cancellation reuses the same kill path).
- **Performance budget:** a per-tool timeout policy (§6.1).
- **Copy/tone policy:** the "pdftotext unavailable" advice without auto-install.
- **Validation level:** tests with a stub helper that sleeps or panics; a manual Windows check that no process remains after a timeout.
- **Owner sign-off:** required, because removing auto-install changes behaviour.

## 4. Current architecture context
Each format package calls its helper directly with no context. The PDF path falls back to a winget
install when the bundled binary cannot be executed. The helper cache is keyed only by file name.
Panic guards exist around some PDF library calls, but not around image extraction, repair or the
OCR workers, and the pipeline has none.

## 5. Proposed approach
### 5.1 Pillars / modules
- **Process runner:** one shared way to run a helper, with a deadline, cancellation, a process-tree kill, bounded captured output, and a uniform error.
- **Consent-only installs:** the auto-install becomes advice text (CLI) or an explicit button (GUI).
- **Versioned helper cache:** the cache location includes the bundled content hash; files are written to a temporary name, then renamed. Old versions are cleaned opportunistically.
- **Containment boundaries:** recovery at each best-effort unit (a PDF image pass, a PDF repair attempt, one OCR image), turning a panic into a recorded per-item failure. A top-level recovery in the pipeline reports an internal error with a clean exit code, instead of a crash dump.
- **Thread discipline:** the OCR worker processes run single-threaded internally.

### 5.2 Data & event flows
Stage -> process runner (deadline, cancel) -> helper -> result or timeout -> stage decision.

## 6. Open questions / research items
1. **Timeout policy**
   - **Question:** fixed per tool, or scaled by input size or page count?
   - **Status:** Decided: a per-tool base deadline scaled by input size and clamped to a ceiling - pdftotext 2 min + 1 min per 50 MB (max 30 min), Tesseract 2 min per image + 10 s per MB (max 10 min), its short probes 30 s, Calibre 10 min + 30 s per MB (max 60 min), 7-Zip 2 min + 6 s per MB (max 30 min), ffmpeg/ImageMagick 2 min. The environment variable `DOCHT_TOOL_TIMEOUT_SCALE` (a float multiplier, applied after the ceiling) is the override; no new CLI flag.
2. **GUI install button**
   - **Question:** offer an in-app "install Poppler" action, or only link to the instructions?
   - **Status:** Decided: no install anywhere. The automatic winget install is removed; the CLI prints localized advice on installing Poppler manually, and the GUI only shows that advice (the converter's warning dialog), with no install button.

## 7. Risks
- **Timeouts fire on slow machines with huge PDFs.** Likelihood: medium. Impact: the conversion fails. Mitigation: size-scaled deadlines and an override flag.
- **Recovering from a panic hides real bugs.** Likelihood: low. Impact: silent degradation. Mitigation: always log the panic with its stack to the run log.

## 8. User impact (docs)
README: pdftotext is never auto-installed; how to install Poppler manually.

## 9. Architecture decisions (ADR)
**ADR-1: one process runner.** Why: six call sites with six ad-hoc behaviours is how the missing timeouts happened.

## 10. Links to other specs
`bugfix-resource-budgets`, `bugfix-gui-local-api-hardening`.

## 11. Done criteria (strategic)
1. A stub pdftotext that never exits makes the conversion fall back or fail within the deadline, and no process remains.
2. A blocked pdftotext never triggers a winget install.
3. After upgrading the app, the new pdftotext is used.
4. An image that panics the decoder is reported as one OCR failure, and the book still converts.

## 12. Next step
`/spec-tech 11_2026-09-24_bugfix-external-process-bounds`

## Implementation

- **ADR-1 - one runner.** New package `internal/procrun`: `Run(ctx, Cmd) (Result, error)` with a deadline (`Cmd.Timeout`), a process-tree kill on expiry or cancel (Windows: a job object with `KILL_ON_JOB_CLOSE`, `taskkill /T /F` fallback; elsewhere: its own process group killed by `-pgid`), `cmd.WaitDelay` so a grandchild holding a pipe cannot hang `Wait`, capped stdout/stderr, and one `*procrun.Error` that names the tool and says timeout (localized), matching `errors.Is(err, procrun.ErrTimeout)`. Leftovers are killed after a normal exit too, so no helper outlives its call. Per-tool policy in `procrun.Budget` (`For`, `ForFile`).
- **X9 / O5 - deadlines at every helper call.** pdftotext and the JPX converter (`internal/pdf`), Calibre (`internal/mobi`), 7-Zip (`internal/comic`), and all three Tesseract calls (`internal/ocr`: recognition, `--list-langs`, script detection) go through `procrun`. A timed-out pdftotext falls back to the pure-Go reader; a timed-out Tesseract image is one OCR failure. A pdftotext output over 256 MB, or a Tesseract TSV over the cap, is treated as a failure rather than parsed truncated. Call sites pass `context.Background()` for now; the cancellation path is in place for `bugfix-gui-local-api-hardening` to thread a real context.
- **X10 - no installs.** `tryInstallPoppler` and every winget call are gone. A blocked bundled pdftotext retries a Poppler the user installed, then falls back and prints/shows localized advice (`winget install ossia.poppler`; `poppler-utils` / `brew install poppler` elsewhere). The old advice to exempt the cache folder from antivirus scanning is dropped: INSTALL-TRUST forbids telling users to weaken a protection. The GUI never offered an install and needed no change.
- **P15 - versioned cache.** `internal/bundledtools`: the bundled set is unpacked into `<UserCacheDir>/doc-html-translate/pdftotext-<sha256[:12]>/`, hashed over file names and contents; each file is written to a temp name in that folder and renamed, a file already holding the right bytes is left alone (it may be running in another instance), and older `pdftotext` / `pdftotext-<hash>` folders are removed best-effort. A build without the vendored exe reports `ErrNotBundled` instead of a path to nothing.
- **X7 / O7 - containment.** `recover` at: each page of the PDF image pass and the pass as a whole, the pdfcpu repair attempt (`optimizeSafe`), each OCR image in the worker pool (`recognizeSafe`, a panic becomes that image's failure), and the OCR stage as a whole (`overlayImagesSafe`). `pipeline.Runner.Run` has a top-level guard: a panic becomes a localized "internal error" and exit code 3 (`ExitInternal = ExitParse`) - reused, not new, because the exit codes are an OCR-INVOCATION contract and an escaped panic is in practice a parser failing on this document. Stacks go to the run log only (`logging.RunLogf`).
- **O9.** Every Tesseract process runs with `OMP_THREAD_LIMIT=1`.
- Docs: README (EN/RU/UK) behavior notes on no auto-install, manual Poppler install, the versioned cache and `DOCHT_TOOL_TIMEOUT_SCALE`; AGENTS.md architecture map.

Tests:

- Done 1: `TestExtractFallsBackWhenPDFToTextHangs` (`internal/pdf`, non-Windows) - a stub pdftotext that never exits and starts a child: the conversion falls back within the deadline and the child is gone. `internal/procrun`: `TestRunKillsTheTreeAtTheDeadline`, `TestRunKillsTheTreeOnCancel`, `TestRunDoesNotHangOnAnInheritedPipe`, `TestRunBoundsCapturedOutput`. The Windows job-object path is compiled and vetted but needs a hands-on check (status).
- Done 2: `tests/no_auto_install_test.go` fails on any `"winget"` program literal or winget agreement flag in shipped Go code.
- Done 3: `internal/bundledtools/cache_test.go` - content-keyed folder, upgrade lands in a new folder and prunes the old one, a partial file is repaired with no temp left, eight concurrent extractions all see a complete copy. The real upgrade on an installed app needs a hands-on check (status).
- Done 4: `TestRecognizePathsContainsAPanickingImage` (`internal/ocr`) - one image panics, it is one recorded failure, the others are recognized.
