# Strategic spec: 07_2026-09-24_bugfix-output-completeness - Reuse only complete outputs built with the same options

**Ticket:** 07_2026-09-24_bugfix-output-completeness
**Status:** Draft
**Priority:** 90
**Date:** 2026-09-24
**Tier:** Moderate
**Tactical plan:** `DEV/plan/07_2026-09-24_bugfix-output-completeness/` (created by /spec-tech)
**Findings:** P4 P5 P8 P9 T5 O10 (see the [findings register](../research/audit_2026-09-24/README.md))

> **Scope:** STRATEGIC.

---

## 1. Problem
The "already converted, just open it" shortcut trusts the mere existence of the entry page.
That page is written early, before OCR and translation. A run that is interrupted, crashes, or
fails mid-translation therefore leaves an output the next run treats as finished. The shortcut
also ignores the options: a later run that asks for translation or OCR silently opens the earlier
untranslated result. A translation failure is reported as success (exit 0) with a message that
says the book is untranslated, while part of it is already translated. Ctrl+C can cut a page in
half, because files are rewritten in place.

## 2. Goals
1. An output is reused only if it was completed, from the same source, with options that affect the result.
2. When the options differ, the user is told, and the output is rebuilt (or reused, if the user chooses).
3. No page file can ever be left half-written: every rewrite is all-or-nothing.
4. Ctrl+C stops the run cleanly: the exit code signals interruption and nothing partial is marked complete.
5. A translation failure yields a distinct non-zero exit code and an accurate message ("partially translated, N of M pages").
6. For Ollama, batches that already succeeded are kept when a sibling batch fails.

**Non-goals:**
- Resuming a partial translation from where it stopped (a possible follow-up).

## 3. Wishes and constraints
### 3.1 Owner wishes
- A partially translated output could be offered for resume later.

### 3.2 Hard constraints
- **Platform / versions:** Windows file-replace semantics (open handles block renames).
- **Data compatibility:** legacy outputs have no completion record; handle them by the policy chosen in `hotfix-output-dir-ownership` §6.1.
- **Localization:** new messages in 13 languages.

### 3.3 Owner inputs (Approval gate)
- **Related tickets:** `hotfix-output-dir-ownership` (it provides the record this ticket extends with completion and options); `bugfix-gui-local-api-hardening` (the GUI already keeps its own option fingerprint, which must agree with this).
- **Data compatibility:** which options count as "affecting the result" (§6.1).
- **Validation level:** tests for an interrupted run, reuse after an option change, and the exit code on a translation failure.
- **Owner sign-off:** exit-code semantics are public, so owner sign-off is needed for the new code.

## 4. Current architecture context
The pipeline writes the entry page in its generation step, then runs OCR, translation and index
regeneration in place. Early returns after generation do no cleanup. The interrupt handler exits
the process from a background goroutine. The GUI separately remembers option fingerprints per
output, but the CLI knows nothing about them.

## 5. Proposed approach
### 5.1 Pillars / modules
- **Completion record:** written as the very last step of a successful run. It holds the source identity, the result-affecting options, the tool version, and a translation state (none / full / partial). Reuse requires it.
- **Staged build:** the run builds into a sibling staging location and swaps it into place on success; or it writes every file atomically and relies on the completion record. Choose one in §6.2.
- **Cooperative cancellation:** an interrupt request is checked between pages and batches; the run then stops, cleans the staging area and returns the interruption code.
- **Honest translation outcome:** a distinct exit code and message for partial or failed translation, and the record marks the output partial so it is not silently reused as a translated book.
- **Batch preservation (Ollama):** a failed batch no longer discards the page's successful batches; untranslated segments stay in the source language and are counted.
- **One source of truth for the GUI:** the GUI reads the CLI's record instead of keeping its own fingerprint.

### 5.2 Data & event flows
Run -> staging -> extract -> generate -> OCR -> translate -> index -> record -> swap into place.
Reuse check -> record present, complete, and matching -> open; otherwise rebuild (or prompt).

## 6. Open questions / research items
1. **Result-affecting options**
   - **Question:** exactly which flags invalidate reuse?
   - **Options:** engine, target language, OCR on/off plus OCR language, single-page/multipage, split size, TOC depth.
   - **Status:** Open.
2. **Staged directory vs atomic files**
   - **Question:** staging needs double disk space for large PDFs; atomic files plus a record need no swap.
   - **To find out:** measure disk use on the largest test PDF, and check Windows rename behaviour while a browser holds files open.
   - **Status:** Open.
3. **Exit code value**
   - **Question:** reuse the existing `4` or add a new "partial" code?
   - **Status:** Open.

## 7. Risks
- **A browser holding the old output open blocks the swap on Windows.** Likelihood: medium. Impact: the rebuild fails. Mitigation: prefer atomic files plus the record, or retry the rename with a clear message.
- **The new exit code breaks user scripts.** Likelihood: low. Impact: scripts treat partial as failure. Mitigation: document it in the README.

## 8. User impact (docs)
README: the reuse rule ("reused only when complete and the options match") and the exit code list.

## 9. Architecture decisions (ADR)
**ADR-1: the completion record is authoritative for reuse.** Alternatives: checking for the entry page. Why: the entry page exists long before the run is complete.

## 10. Links to other specs
`hotfix-output-dir-ownership`, `bugfix-translation-engine-correctness`, `bugfix-gui-local-api-hardening`.

## 11. Done criteria (strategic)
1. Kill the process during translation; the next run rebuilds instead of opening the partial output.
2. Convert with no translation, then run with `-ollama -dst de`; the output is translated (or the user is asked).
3. A simulated translation failure on page 3 exits non-zero and says "partially translated".
4. Interrupting at any point leaves no page that fails to parse.

## 12. Next step
`/spec-tech 07_2026-09-24_bugfix-output-completeness`
