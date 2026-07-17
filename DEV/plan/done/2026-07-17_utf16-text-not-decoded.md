# UTF-16 text is not decoded; Cyrillic comes out as mojibake

**Status:** Implemented - 2026-07-17; both editions, both modes, verified by reading the output
**Priority:** 2
**Date:** 2026-07-17

> **What landed.** `decodeText` in [`internal/txt/extract.go`](../../../internal/txt/extract.go) and its
> twin in [`extension/src/txt.js`](../../../extension/src/txt.js): BOM first (UTF-8, UTF-16 LE/BE), then
> `utf8.Valid`, then a `decodeLegacy` seam that P3 fills. The decode order is now a recorded shared
> invariant in [`docs/PARITY.md`](../../../docs/PARITY.md).
>
> **The extension was broken too, and this was measured rather than assumed** (the ticket only asked
> that it be checked). `TextDecoder("utf-8")` on UTF-16 bytes yields `U+FFFD U+FFFD` plus 75 NULs -
> the same defect. But the **UTF-8 BOM leak is Go-only**: `TextDecoder` strips a BOM unless `ignoreBOM`
> is set, so the extension never had that half. Recorded in PARITY.md as a difference that is not drift.
>
> **P4 is unblocked.** The ordering constraint below is satisfied: the BOM is now tested before
> anything counts NULs, so the sniff can land without turning UTF-16 from mangled into refused.

> **One implementation pass with
> [`2026-07-17_txt-legacy-encoding-detection`](2026-07-17_txt-legacy-encoding-detection.md) (P3).** Both
> tickets change the same decision - how `internal/txt` (and `extension/src/txt.js`) decides the source
> encoding before decoding - and that ticket's own owner decision already says "a BOM (UTF-8/UTF-16
> LE/BE) is authoritative when present", which *is* this fix. They stay as two files because the evidence
> and the urgency differ (this one is a live corruption bug; that one adds legacy code pages), but they
> want one tactical plan: BOM first, then the code-page heuristic. Implementing them separately means
> touching the same function twice.
>
> **Ordering constraint for [`binary-input-becomes-a-garbage-document`](2026-07-17_binary-input-becomes-a-garbage-document.md)
> (P4):** that ticket's NUL sniff refuses any input with NUL bytes in the first 4 KB, and UTF-16 text is
> full of them (75 NULs in a two-line ASCII file). If the sniff lands first, UTF-16 files start being
> *refused* instead of merely mangled. This fix must land first, or the sniff must check the BOM before
> counting NULs. That is why this sits at P2 and the sniff at P4.

## What / why

`internal/txt` reads its input as if every byte were a character. A UTF-16 file - what Notepad writes
when you pick **"Unicode"** in its Save-as encoding list - is therefore never decoded. The bytes are
passed straight through into the output.

For **Cyrillic** the result is unreadable in the default mode, in `index.html`, which is the file the
reader opens. Measured 2026-07-17 on a two-line UTF-16LE file:

| | |
|---|---|
| Source | `Это обычное предложение на русском языке.` |
| What the reader gets | `-B> >1KG=>5 ?@54;>65=85 =0 @CAA:>< O7K` |

Those are the low bytes of each UTF-16 code unit read as single-byte characters. Nothing in the run
says anything is wrong: `exit=0`, `Done.`

This is not an exotic input for this app's audience. A Russian-language `.txt` saved from Notepad as
"Unicode" is an ordinary thing a real user has on disk, and the conversion turns it into confident
garbage - which a translator will then happily translate.

### The default mode masks it for ASCII, which is why it has gone unnoticed

For **ASCII** text in UTF-16 the same run looks fine, but only by accident:

| Mode | What the reader sees |
|---|---|
| default (single-page) | `The Project Gutenberg eBook of something.` - correct |
| `-multipage` | `T h e   P r o j e c t   G u t e n b e r g   e B o o k   o f   s o m e t h i n g .` |

The mechanism, confirmed by byte counts rather than inference:

| File | Bytes | NUL bytes |
|---|---|---|
| source `utf16le-bom.txt` | 152 | 75 |
| `page_001.html` | 599 | **75** - passed through untouched |
| `index.html` (single-page) | 16,296 | **0** - stripped by the merge |

`txt.Extract` writes the raw bytes into `page_001.html`, NULs and all. `htmlgen.GenerateSinglePage`
merges the spine and drops the control characters on the way, so `T\0h\0e` becomes `The`. That is
NUL-stripping, not decoding - it works for ASCII because ASCII-in-UTF-16 is exactly "every character
followed by a NUL". `-multipage` skips the merge (`index.html` becomes a redirect to `page_001.html`)
and hands the reader the raw bytes.

Cyrillic has no NUL to strip - `П` is `1F 04` - so there is nothing for the accident to remove and
**both** modes are broken. The same file therefore reads correctly, or as spaced-out letters, or as
mojibake, depending on the alphabet and on a flag. Any fix must go in the decoder; do not be tempted
to "fix" the merge.

BOM handling is worth settling in the same pass: the UTF-8-with-BOM case currently leaks its BOM into
the first paragraph (`﻿The Project Gutenberg..`), which is cosmetic but the same root gap - nothing
looks at the first bytes to decide an encoding.

## Interaction with the binary sniff

[`2026-07-17_binary-input-becomes-a-garbage-document`](2026-07-17_binary-input-becomes-a-garbage-document.md)
proposes refusing any input with a NUL byte in the first 4 KB. UTF-16 text is exactly that - 75 NULs
in a two-line file - so a naive sniff would **refuse a legitimate `.txt`**. The two tickets have to
agree: check the UTF-16 BOM (`FF FE` / `FE FF`) before the NUL rule, and decode rather than refuse.
Landing either one alone risks trading this bug for the other.

## Where to look

- [`internal/txt`](../../../internal/txt) - the extractor. Needs BOM detection (`FF FE`, `FE FF`,
  `EF BB BF`) and a real decode; Go's `golang.org/x/text/encoding/unicode` covers it.
- [`internal/htmlgen/singlepage.go`](../../../internal/htmlgen/singlepage.go) - `GenerateSinglePage`,
  where the accidental NUL-stripping happens. Do not build the fix here.
- Extension: it decodes fetched bytes for `parseText` in `extension/src/viewer.js` - check whether it
  handles UTF-16 or shares this gap before assuming a Go-side fix covers it.

## Done criteria

- [x] A UTF-16LE and a UTF-16BE `.txt` convert to correct text, Cyrillic included. `TestExtractDecodesUTF16`.
- [x] Correct in **both** default and `-multipage` - the mode must not change the text. Verified on the
      four real fixtures in both modes: all eight read correctly.
- [x] A UTF-8 BOM does not leak into the first paragraph. `TestExtractStripsUTF8BOM`.
- [x] Plain UTF-8 and the existing cp1251-era fixtures still convert unchanged. `TestExtractPlainUTF8Unchanged`;
      `Лицензионное Соглашение.txt` produces a **byte-identical** `index.html` before and after (same SHA-256).
- [x] Verified by opening the output and reading it, not by counting bytes - after two harness misreads
      (a swallowed CLI log, and `-folder` being the parent dir) were caught and corrected.
- [x] Extension checked, not assumed - measured broken, fixed, `npm test` 43/43 green.
- [x] Tests / gate green (`scripts/test.ps1`) - Go + lint green except the pre-existing `TestExtract_NoCalibre`.
- [x] Changelog entry in `DEV/CHANGELOG.md`.

## Found while verifying, handed to P3

`test_doc/Лицензионное Соглашение.txt` is **real Windows-1251** (3873 bytes, 76% high-bit, not valid
UTF-8), so it converts to mojibake today and will until P3 lands. P3's claim that the corpus had no
non-UTF-8 fixture was wrong; corrected there, with the candidate-decode evidence.
