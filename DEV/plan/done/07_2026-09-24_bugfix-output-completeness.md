# Strategic spec: 07_2026-09-24_bugfix-output-completeness - Reuse only complete outputs built with the same options

**Ticket:** 07_2026-09-24_bugfix-output-completeness
**Status:** BlockNeedUserTest - implemented and covered by tests on Linux; needs a Windows hands-on: Ctrl+C in a real console during a translation, and a rebuild while Chrome holds the old output open (page rename over an open file, rewrite of the hidden record).
**Priority:** 90
**Date:** 2026-09-24
**Tier:** Moderate
**Tactical plan:** `DEV/plan/done/07_2026-09-24_bugfix-output-completeness/` (created by /spec-tech)
**Findings:** P4 P5 P8 P9 T5 O10 (see the [findings register](../../research/audit_2026-09-24/README.md))

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
   - **Status:** Decided: translation engine (none/google/ollama) with the target language (and the Ollama model), OCR on/off with the OCR language, single-page vs multipage, split size and TOC depth, plus the source identity (absolute path, size, mtime). The tool version is stored in the record but is not a reuse criterion. Implementation detail: the source language is compared too wherever it steers the result (with an engine on, or as the OCR language when `-ocr-lang` is not given), and options that cannot affect the output are normalized away (the target language without an engine, the split size and TOC depth in single-page mode, `-ocr` for an image or comic, which is OCRed anyway).
2. **Staged directory vs atomic files**
   - **Question:** staging needs double disk space for large PDFs; atomic files plus a record need no swap.
   - **To find out:** measure disk use on the largest test PDF, and check Windows rename behaviour while a browser holds files open.
   - **Status:** Decided: atomic files (a temp file in the same directory, then a rename) plus the completion record, not a staged directory - a Windows rename of a folder open in a browser fails, and staging doubles disk use. Every page write and rewrite in the pipeline path (extraction, navigation injection, OCR overlay re-render, translation, TOC anchors, index regeneration) goes through one shared helper.
3. **Exit code value**
   - **Question:** reuse the existing `4` or add a new "partial" code?
   - **Status:** Decided: the existing `ExitAPI = 4` for a failed or partial translation, with the localized message "partially translated, N of M pages". Ctrl+C is cooperative (checked between pages, batches and OCR images), exits `130`, never calls `os.Exit` from the signal goroutine, and leaves no completion record. The record marks the translation none / full / partial, and a partial one is never reused as a translated book.
4. **Options mismatch and legacy outputs**
   - **Status:** Decided: a mismatch or a missing / incomplete record prints a localized line saying why, then rebuilds - no interactive prompt. A legacy folder that ticket 05 adopts as ours but that has no completion record is rebuilt; ticket 05's rules for foreign and other-document folders are unchanged.
5. **Ollama batches and the GUI**
   - **Status:** Decided: a failed Ollama batch no longer discards the page's successful batches; the untranslated segments stay in the source language and count toward "partial". The GUI reads the CLI's completion record instead of its own option fingerprint.

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

## Implementation

- **Atomic writes:** new `internal/fsutil` (`Write`, `WriteFile`: temp file in the same directory, rename, a short retry for a Windows sharing hold). Used by every page write in the pipeline path: the extractors' page files, `htmlsplit`, `htmlgen` (navigation, favicon link, TOC anchors, index, single page, redirect stub), `htmlproc.RenderToFile` (translation) and the OCR overlay re-render (O10). Test: `internal/fsutil/atomic_test.go` (a failed rewrite keeps the old page, no temp left).
- **Completion record:** `internal/outputpath/completion.go` extends ticket 05's marker with `complete` (time, tool version, options, forced OCR, translation state), written by `MarkComplete` as the run's last step. `CheckReuse` returns the reason an output is not reusable; `OptionsFor` derives the options from the parsed config for both the CLI and the GUI. Tests: `internal/outputpath/completion_test.go`.
- **Pipeline:** `internal/pipeline/pipeline.go` checks the record before reopening and logs the localized reason (`reuse.go`); translation moved to `translate.go` (outcome with pages done / total, stops at the first engine failure, keeps the partial page, title/TOC failures count as partial), the OCR step to `ocrstep.go`. A partial translation still opens the book and returns `ExitAPI` with "partially translated, N of M pages". `RunContext` takes a context; `internal/app` builds it with `signal.NotifyContext` (a second Ctrl+C kills), cancellation returns `ExitInterrupted` (130) and writes no record; `ocr.OverlayBook` takes the context and stops between images and pages; the Ollama model is unloaded on the way out.
- **Translator:** `Client.Translate` takes a context; `PartialError` carries the untranslated slots; the Ollama client keeps the batches that succeeded (T5) and the cache does not store the missing slots.
- **GUI:** `cmd/doc-html-ui` drops its `output-params.json` fingerprint and asks `outputpath.CheckReuse` on the parsed command line it would run.
- **Done criteria:** 1 - `TestInterruptedRunIsRebuilt` (multipage and single-page); 2 - `TestReuseAfterOptionChangeRebuilds`; 3 - `TestTranslationFailureOnPageThreeIsPartial`; 4 - `TestWriteFailureKeepsOriginal` plus the parse check in the interrupted test. Ollama partial pages: `TestOllamaKeepsSuccessfulBatches`, `TestPartialPageKeepsTranslatedSegments`. GUI: `TestOutputStatusReadsCompletionRecord`.
- **Contract note:** exit code `130` is new to the published list in `OCR-INVOCATION` (codes 0-4); it was already returned by the old Ollama interrupt path. The catalog entry needs the amendment; this repo only holds the pointer.
- **Docs:** README (EN/RU/UK) reuse rule and exit codes; AGENTS.md and CLAUDE.md invariant lines.
