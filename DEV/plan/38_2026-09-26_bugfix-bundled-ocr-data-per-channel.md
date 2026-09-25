# English OCR data is promised in every channel and verified in none

**Status:** Draft
**Priority:** 70
**Date:** 2026-09-26

> Filed by the pre-release audit, [ticket 34](done/34_2026-09-25_full-code-audit-pre-release.md). Finding ids
> refer to its register, [`FINDINGS.md`](done/34_2026-09-25_full-code-audit-pre-release/FINDINGS.md).

## 1. Problem

- **R20, R8** - when the extension's vendored `eng.traineddata` is missing, `build.ps1`, `build-ui.ps1`
  and `build-installer.ps1` download tessdata_fast from `raw/main` with no version pin, no digest and no
  timeout, and a failed download still builds an installer with no English data. The extension build
  and `internal/ocr/download.go` already pin the 4.0.0 digest; done/13 left the desktop scripts under
  "Not done".
- **R21** - neither the MSIX staging nor the CI zip carries `tessdata/eng.traineddata`, although
  AGENTS.md and `tessdata.go` say English data ships next to the exe.
- **R34, R35** - all 13 winget locales say "OCR - English bundled", but the winget zip ships neither
  Tesseract nor the data; the listing also leaves out comic archives and image input.

## 2. Goals

1. Every desktop build script provisions `eng.traineddata` verified against the pinned digest, with a
   timeout, and fails instead of shipping without it.
2. Decide per channel (setup.exe, MSIX, zip / winget) what OCR ships, make the build match, and make
   the listing text say exactly that in every locale.

## 3. Constraints

- The tessdata pin is a both-editions parity invariant (docs/PARITY.md "OCR").
- Listing text lands in every locale in one edit; the Store listing is regenerated, not hand-edited.

## 4. Acceptance

- A build with a tampered or missing cached model fails the digest check (scratch test or `-WhatIf`).
- The winget listing makes no OCR claim the zip does not meet.
