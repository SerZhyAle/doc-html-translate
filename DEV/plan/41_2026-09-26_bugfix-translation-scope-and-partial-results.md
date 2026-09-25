# Translation sends the reader chrome to the paid engine and hides partial results

**Status:** Draft
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
