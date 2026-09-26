# Translation sends the reader chrome to the paid engine and hides partial results

**Status:** Implemented
**Priority:** 85
**Date:** 2026-09-26

> Filed by the pre-release audit, [ticket 34](done/34_2026-09-25_full-code-audit-pre-release.md). Finding ids
> refer to its register, [`FINDINGS.md`](done/34_2026-09-25_full-code-audit-pre-release/FINDINGS.md).

## 1. Problem

- **T13 (high)** - translation runs after the navbar is injected, and htmlproc skips only
  script/style/code/pre. Every page's reader chrome - file name, title, Previous / Next / Contents, the
  theme and font option labels, the version, "N / M" - is sent to the engine, billed on Google, and
  comes back translated inside a bar still marked with the interface language.
- **T15** - an Ollama slot still empty after its echo retries counts as success: the run says
  "Translation complete", the cache stores "" and the segment silently stays untranslated.
- **T11** - when a later batch of a page fails, the Google client discards the batches already paid
  for; nothing is cached and a rebuild bills them again.
- **T10** - `-google` with no key converts and exits 0, while an unreachable Ollama exits 4.

## 2. Goals

1. The reader chrome is never sent for translation, and the cost estimate counts only what is sent.
2. An unfilled slot is a partial result, reported and recorded as partial, never cached as "".
3. Batches that succeeded are kept and cached when a later batch fails, on both engines.
4. One exit code for "the requested engine is unavailable", whichever engine.

## 3. Constraints

- The exit codes are part of the `OCR-INVOCATION` contract: amend it in the catalog first if they change.
- The completion record's partial state is the existing one; no new state.

## 4. Acceptance

- A stub-engine test sees no navbar text in the request.
- A stub Ollama that skips a number yields a partial run and no cached "".

## Implementation (2026-09-26)

All four goals are met; no item was declined or blocked.

- **T13** - the reader chrome is left out where the text is collected, so it is neither sent nor
  counted. `htmlgen.IsReaderChrome` (new `internal/htmlgen/chrome.go`) recognizes the roots of the
  chrome htmlgen writes (`dht-navbar`: chapter bar and single-page header; `dht-toolbar`: index
  toolbar); `htmlproc.ExtractTexts` takes a skip predicate that drops such a subtree; the pipeline
  passes it (`internal/pipeline/translate.go`). The cost estimate is built from the same segments,
  so it now counts only what is sent. No markup changes: the page output is identical except that
  the chrome stays in the interface language. Tests: `TestReaderChromeIsNotSentForTranslation`
  (paged and single-page, a recording stub engine sees no navbar text and no page carries a
  translated chrome word), `TestChromeIsNotBilled`, `TestExtractTextsSkipsCallerSubtree`.
- **T15** - `OllamaClient.runJob` now reports the slots still empty after the echo retries and no
  longer drops a failed retry request's error. `Translate` returns them in a `*PartialError`
  whose cause is the new `translator.ErrNoTranslation` when every request succeeded (a failed
  retry is the cause otherwise, and stops new batches like any engine failure). `CachingClient`
  never caches "" for a non-empty text and reports an unflagged empty answer the same way, whatever
  the engine. The pipeline treats an `ErrNoTranslation` page as not fully translated but goes on
  with the book (the engine is working); the run ends `partial` (the existing state) with exit 4.
  Tests: `TestOllamaSkippedNumberIsPartialAndNotCached` (stub Ollama that skips a number: partial,
  no cached ""), `TestOllamaRetryFailureIsReported`, `TestCacheRefusesUnflaggedEmptyAnswer`,
  `TestEmptySlotIsPartialNotComplete` (pipeline: `partially translated, 4 of 5 pages`, record
  `partial`).
- **T11** - `GoogleClient.Translate` keeps the batches already answered when a later batch fails
  and returns them with a `*PartialError` naming the rest, so `CachingClient` caches them and the
  page keeps them. The Ollama half was already in place (`TestOllamaKeepsSuccessfulBatches`). The
  TOC-label step also uses the labels of a partial reply instead of dropping all of them. Test:
  `TestGoogleKeepsAnsweredBatches`.
- **T10** - conforms to the `OCR-INVOCATION` pointer as it stands (`docs/contracts/OCR-INVOCATION.md`:
  `4` = translation API): no code is added or re-used for another meaning, so no catalog amendment
  was needed. `-google` with no key now ends with exit 4 like an unreachable Ollama; the book is
  still produced and recorded `none` (so the next `-google` run rebuilds). Exit-code wording updated
  in `README.md`, `README_RU.md`, `README_UK.md`. Test: `TestGoogleWithoutKeyExitsLikeOllamaDown`.

Not changed: the Go/JS parity map (engine translation is Go-only, and the chrome markup is
untouched), the completion record (no format constant to bump; `partial` is the existing state).
For the owner on Windows: nothing platform-specific; a real `-google` run with the key file
removed should now exit 4 (`echo %ERRORLEVEL%`).
