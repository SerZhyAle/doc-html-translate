# English OCR data is promised in every channel and verified in none

**Status:** BlockNeedUserTest - implemented 2026-09-26; the next release must show `tessdata/eng.traineddata` inside the CI zip, the MSIX and setup.exe
**Priority:** 70
**Date:** 2026-09-26

> Filed by the pre-release audit, [ticket 34](34_2026-09-25_full-code-audit-pre-release.md). Finding ids
> refer to its register, [`FINDINGS.md`](34_2026-09-25_full-code-audit-pre-release/FINDINGS.md).

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

## 5. Outcome (2026-09-26)

**Channel decision.** No channel ships Tesseract itself (`internal/ocr` `Locate` finds it through
`DOCHT_TESSERACT`, `<exe>/tesseract/`, or PATH). Every channel ships the English language data:

| Channel | English data | Tesseract |
|---|---|---|
| setup.exe (`scripts/build-installer.ps1`) | `tessdata/eng.traineddata`, as before, now verified | installed separately |
| MSIX (`msix/build-msix.ps1`) | `tessdata/eng.traineddata` - **new** | installed separately |
| zip / winget (`.github/workflows/release.yml`) | `tessdata/eng.traineddata` - **new** | installed separately |
| local build (`scripts/build.ps1`, `build-ui.ps1`) | `build/` and the deploy folder, now verified | - |

**Build.** One helper, `scripts/lib/tessdata.ps1` `Install-EngTessdata`, is dot-sourced by all five
paths. Sources, each checked against the pinned size + SHA-256 before use: the extension's vendored
copy (a mismatch is an error), a download cache under `temp/tessdata-cache` (a mismatch is discarded),
then the `raw/4.0.0` URL with a 300 s timeout. Any failure throws, so no package is built without the
data. The pin equals `internal/ocr/download.go` `packDigests["eng"]` and `extension/build.mjs`
`ENG_TRAINEDDATA_SHA256`.

**Listing.** All 13 winget locales now say OCR runs through a separately installed Tesseract with
the English language data bundled, and gain one sentence on CBZ/CBR/CB7/CBT comic archives and
standalone images (CBR/CB7 need 7-Zip). `winget validate --manifest winget` passes. The Store listing's
"English bundled" is now true for the MSIX and was left as is.

**Evidence.** `tests/tessdata_pin_test.go`: `TestEngTessdataPinAgrees` (the three pins and the
`cdnBase` URL agree; no packaging path fetches tessdata on its own) and `TestInstallEngTessdataOutcomes`
(against a local server: failed download, tampered download, tampered vendored copy all exit 1 and
leave nothing behind; a tampered cache is re-fetched once; a verified cache is reused with no request;
a tampered destination is replaced). `go test ./tests/ -count=1` -> `ok` (on the 386 toolchain it passed once
and died three times on the known 2 GB out-of-memory ceiling; with `GOARCH=amd64` -> `ok`, 339 s);
`go test ./internal/ocr/` -> `ok`.

**Left for the release.** The zip and MSIX changes run only on the next tag / Store build; that run
has to show `tessdata/eng.traineddata` in both artifacts before this becomes `Verified`.
