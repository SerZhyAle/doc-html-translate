# Strategic spec: 11_2026-09-24_bugfix-external-process-bounds - Bounded helper processes and contained crashes

**Ticket:** 11_2026-09-24_bugfix-external-process-bounds
**Status:** Draft
**Priority:** 75
**Date:** 2026-09-24
**Tier:** Moderate
**Tactical plan:** `DEV/plan/11_2026-09-24_bugfix-external-process-bounds/` (created by /spec-tech)
**Findings:** X7 X9 X10 O5 O7 O9 P15 (see the [findings register](../research/audit_2026-09-24/README.md))

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
   - **Status:** Open.
2. **GUI install button**
   - **Question:** offer an in-app "install Poppler" action, or only link to the instructions?
   - **Status:** Open.

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
