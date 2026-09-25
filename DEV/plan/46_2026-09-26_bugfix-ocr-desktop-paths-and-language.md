# Desktop OCR fails under a non-ANSI profile, and the GUI never checks the script

**Status:** Draft
**Priority:** 70
**Date:** 2026-09-26

> Filed by the pre-release audit, [ticket 34](done/34_2026-09-25_full-code-audit-pre-release.md). Finding ids
> refer to its register, [`FINDINGS.md`](done/34_2026-09-25_full-code-audit-pre-release/FINDINGS.md).

## 1. Problem

- **O15** - the paths meant to be ASCII-safe for Tesseract (the temp staging and, since O4, the per-user
  `--tessdata-dir`) live under the user profile and are never checked. A profile path outside the ANSI
  code page loses OCR on the upscale, rotate, rescue and screen passes, and on every run once a language
  pack is downloaded.
- **O16** - the GUI always sends `-ocr-lang` (its select falls back to `eng`), so the script check of
  ticket 30 never runs in the GUI, which is the Store entry point.
- **O17** - `-src` codes the catalog lacks pass through or map to packs that do not exist, and the
  advice to download them is refused.
- **O18** - a failed copy of a bundled pack leaves every image failing with an error that names nothing.

## 2. Goals

1. Every path handed to Tesseract is ASCII-safe, or verified to be (short names or an ASCII root).
2. The GUI sends `-ocr-lang` only on an explicit choice.
3. Language mapping and advice agree with the download catalog.
4. A missing pack is named in the error.

## 3. Constraints

- `OCR-INVOCATION` is ours: amend the catalog first if the invocation changes.

## 4. Acceptance

- A test with a non-ASCII temp and data root passes (or is skipped with the reason where one cannot be
  created); the GUI argument test shows no `-ocr-lang` for the default choice.
