# The app breaks its own typography rule in the output it generates

**Status:** Implemented
**Priority:** 13
**Date:** 2026-07-17

> **Decision (2026-07-18):** the `<title>Book — Page N</title>` em dash **stays** - it is conventional
> book typography, not app chatter. Recorded in [CLAUDE.md](../../../CLAUDE.md) as the rule's one exception
> and enforced by the guard, which allows an em dash only inside a `<title>` literal. Everything else
> (logs, dialogs, errors, the excerpt ellipsis, the extension's `_locales` messages) was swept to `-` and
> `..`. Guard: [`tests/typography_test.go`](../../../tests/typography_test.go).

> Found while implementing [`2026-07-17_cli-diagnostics-honesty`](2026-07-17_cli-diagnostics-honesty.md)
> (P1), whose fix touched two of these strings. Filed rather than fixed by stealth: it is a
> product-wide sweep with one product decision in it, not a drive-by.

## What / why

`CLAUDE.md` states the rule without qualification, and says where it applies:

> **Typography:** use short hyphens (no long dashes), Russian **ё** where applicable, and ".." not "...".
> Applies to **generated output**, docs, and commit messages.

The generated output breaks it in about **42 places across 11 files**. Measured 2026-07-17 by scanning
every Go string literal outside `vendor/` and `_test.go`.

The one that matters most is not a log line. It is in the product:

```go
"  <title>%s — Page %d</title>\n"
```

That is a **long dash in the page title of every page of every converted book**, and it is duplicated
across five extractors - `internal/fb2`, `internal/md`, `internal/pdf` (twice), `internal/rtf`,
`internal/txt`. The rule's own first clause, in the app's most user-visible string.

The rest fall into two groups:

| Shape | Count | Examples |
|---|---|---|
| `...` where the rule says `..` | ~20 | `"[1/4] Extracting PDF..."`, `"[2/4] Building HTML structure..."`, `"Loading model %s into VRAM...\n"`, `"  Installing Poppler via winget...\n"` |
| A long dash in user-visible text | ~15 | `"  Single-page mode — all content merged, TOC skipped."`, `"Calibre not found — install Calibre from .."`, `"[3/4] Google Translate skipped — API key not available.\n"`, `"PDF Quality Reduced — pdftotext Unavailable"` |

Plus one literal ellipsis character (`…`) in `internal/htmlgen/htmlgen.go`.

Two data points on why this is drift rather than a decision:

- `internal/app` already writes `"Press Enter to close.. (we both know you'll close the window anyway)"`
  with the correct two dots, while `cmd/doc-html-translate` wrote `"Press Enter to close..."` with three.
  Two spellings of the same sentence in one binary. (Both now say `..` - P1 touched them.)
- `internal/pipeline/pipeline.go:182` manages **both** violations in a single generated line:
  `"[1/4] Unknown format %q — treating as plain text...\n"`. It is quoted verbatim in
  [`binary-input-becomes-a-garbage-document`](2026-07-17_binary-input-becomes-a-garbage-document.md)
  (P4), which rewrites that dispatch anyway.

## The decision this needs

**Does the rule mean the `<title>` dash too, or only the app's own chatter?**

"Alice's Adventures in Wonderland — Page 5" is conventional book typography, and an em dash there is a
deliberate-looking choice rather than an obvious slip. But the rule as written has no exception, and
this is generated output by any reading. Settle it before the sweep, because it decides whether this is
a log-only cleanup or a change to what every converted book looks like.

The `...` -> `..` half needs no decision - the rule is unambiguous and those are all app chatter.

## Scope notes

- **Not the extension.** This ticket is the Go side; `extension/src` needs the same scan before this
  can be called done, since its strings are a separate codebase (and `_locales` carries three
  languages).
- **`→` is not a dash.** `"  Converting MOBI → EPUB via Calibre ebook-convert..."` uses an arrow; the
  rule does not mention arrows. Its `...` is in scope, its arrow is not.
- **Cyrillic ё** is the rule's third clause and was not surveyed here. Worth folding into the same pass.
- **Comments and docs are out of scope** - the rule covers generated output, and rewriting every `—`
  in a comment would bury the change that matters.
- **P4 overlaps by one line** (`pipeline.go:182`) and is already recorded there as a free fix.

## Done criteria

- [x] The `<title>` question is answered and recorded, either way. (Keep the em dash - see Decision above.)
- [x] No user-visible Go string literal outside `vendor/` carries `...` or a long dash, except the
      `<title>` em dash. Swept across `internal/{comic,pdf,pipeline,mobi,translator,app,htmlgen}`.
- [x] The extension scanned to the same standard, en/ru/uk. Only the three `_locales/*/messages.json`
      "right-click, Translate to .." strings carried `...`; fixed. No user-visible `—`/`…` in `extension/src`.
- [x] A guard so this does not drift back. `typos` is a spell-checker and cannot ban `—`/`...`, so the
      guard is a source-scanning test (`tests/typography_test.go`) that AST-walks Go string literals and
      the `_locales` messages - it lives beside the existing repo-scan guards in `tests/parity_test.go`.
- [x] Tests / gate green (`scripts/test.ps1`).
- [x] Changelog entry in `DEV/CHANGELOG.md`.
