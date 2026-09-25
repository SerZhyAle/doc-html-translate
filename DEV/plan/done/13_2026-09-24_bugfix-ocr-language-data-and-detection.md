# Strategic spec: 13_2026-09-24_bugfix-ocr-language-data-and-detection - Safe language downloads and reliable OCR detection

**Ticket:** 13_2026-09-24_bugfix-ocr-language-data-and-detection
**Status:** BlockNeedUserTest - implemented and covered by tests on Linux; needs done criterion 3 by hand: in the Store (MSIX) build, download `deu` from the GUI and run OCR with it, then confirm the pack landed under `%LOCALAPPDATA%\doc-html-translate\tessdata` (or the package's redirected LocalCache) and Tesseract loads it
**Priority:** 70
**Date:** 2026-09-24
**Tier:** Moderate
**Tactical plan:** `DEV/plan/done/13_2026-09-24_bugfix-ocr-language-data-and-detection/` (created by /spec-tech)
**Findings:** O1 O2 O3 O4 O8 O11 O12 (see the [findings register](../../research/audit_2026-09-24/README.md))

> **Scope:** STRATEGIC.

---

## 1. Problem
OCR language packs are downloaded on demand, and the language code goes unchecked into both the
download URL and the file path, so a crafted code writes outside the data folder. Two downloads
of one language race on one temporary file, and a corrupted pack can be installed; the code checks
neither size nor checksum. In the Store build the data folder sits inside the read-only install
directory, so no download can ever succeed there. Script detection, the feature meant for
non-Latin books, skips the path workaround the rest of OCR uses, so it silently fails exactly when
the book's folder has a Cyrillic name. Images whose names are percent-encoded or carry a query are
skipped silently. Running the overlay twice nests the plates.

## 2. Goals
1. Only catalogue language codes are accepted, from every entry point (CLI, GUI).
2. A downloaded language pack is complete and verified before it becomes visible. Concurrent downloads cannot corrupt it.
3. In every build flavour, including MSIX, language packs can be downloaded to a writable per-user location, and bundled data is still found.
4. Script detection works for any book path, including non-ASCII paths.
5. Every image referenced by a page is considered for OCR, whatever the encoding of its reference.
6. The overlay is idempotent on a page.

**Non-goals:**
- OCR quality tuning (tracked by the OCR lab).

## 3. Wishes and constraints
### 3.2 Hard constraints
- **Platform / versions:** the MSIX sandbox; the Windows ANSI code page limitation of the Tesseract binary.
- **Data compatibility:** existing packs in the current location keep working.
- **Localization:** download error messages in 13 languages.

### 3.3 Owner inputs (Approval gate)
- **Related tickets:** `bugfix-gui-local-api-hardening` (the GUI entry point), `bugfix-external-process-bounds` (Tesseract timeout).
- **Platform constraints:** cross-edition. The extension uses the same catalogue and download host (docs/PARITY.md). Checksums pinned for one edition should be shared.
- **Validation level:** a test for a traversal code, a concurrent download test, a checksum mismatch test, and a Cyrillic-path detection test; a manual Store-build download.
- **Owner sign-off:** not required beyond review.

## 4. Current architecture context
The data folder is resolved next to the executable only. Downloads stream to a fixed temporary name
and are renamed into place. Recognition stages images through an ASCII-only temporary path, but
the separate detection call does not. The overlay appends its style and script and wraps images
without checking for an earlier pass.

## 5. Proposed approach
### 5.1 Pillars / modules
- **Catalogue gate:** a language code is valid only if it is in the fixed catalogue.
- **Verified installs:** a unique temporary file per download, a size bound, a checksum against pinned digests, then an atomic rename. A per-language guard serializes concurrent requests.
- **Layered data locations:** first the writable per-user folder, then the bundled folder; downloads always go to the writable one.
- **Uniform path staging:** every Tesseract invocation goes through the same ASCII staging and orientation step.
- **Reference normalization:** image references are decoded, and query and fragment are stripped, before they are resolved.
- **Idempotency markers:** a page already carrying the overlay is skipped or refreshed, not stacked.

## 6. Open questions / research items
1. **Checksum source**
   - **Question:** pin digests in code, or fetch a signed manifest?
   - **Status:** Decided: pin SHA-256 digests and exact sizes in code for every catalogue language
     (`internal/ocr/download.go` `packDigests`), taken from the 4.0.0 files on 2026-09-25. The code's
     URL (`github.com/.../raw/4.0.0`) is not reachable from the build sandbox, so the digests were
     computed from its redirect target `raw.githubusercontent.com/tesseract-ocr/tessdata_fast/4.0.0/`,
     which serves the same bytes. Not shared with the extension: tesseract.js fetches a gzipped build
     from projectnaptha and never exposes the bytes; the difference is recorded in docs/PARITY.md.
2. **Catalogue gate** - Decided: a code is valid only if it is in `Available`; enforced by
   `ocr.CheckLang` in `-ocr-download` (internal/app), the GUI's `/api/ocr-download` and `ocr.Download`.
3. **Verified installs** - Decided: unique temp file per download in the target folder, body bounded
   to the pinned size, SHA-256 checked, atomic rename; a per-language in-process mutex, and a pack
   another process installed is accepted when it verifies. Temp files older than a day are removed
   on the next download.
4. **Layered data locations** - Decided: per-user `os.UserCacheDir()/doc-html-translate/tessdata`
   (`%LOCALAPPDATA%` on Windows, the translator's precedent) first, then `<exe>/tessdata`. Downloads
   go to the per-user folder. When both folders hold packs, the bundled ones are copied into the
   per-user folder once and that folder is passed to Tesseract, so a split `rus+eng` loads.
5. **Overlay idempotency (O12)** - Decided: the injected style and script carry `data-dht-ocr` and are
   refreshed, not stacked (an unmarked copy from an older page is recognized by its content); an
   image already inside `.ocr-fig` is not wrapped again.
6. **Localization** - Decided: the four new download messages are registered with `i18n.Add` in all
   12 translations; the GUI sends its page language so the error comes back in it. No new
   `i18n.js` strings were needed.

## 7. Risks
- **Upstream traineddata files change, so the pinned checksums fail.** Likelihood: low (the version is pinned in the URL). Impact: the download is refused. Mitigation: a clear message; update the digests on a release.

## 8. User impact (docs)
No changes to user docs.

## 9. Architecture decisions (ADR)
No ADRs. The decision follows established project patterns (the translator already uses a per-user fallback location).

## 10. Links to other specs
`bugfix-gui-local-api-hardening`, `bugfix-external-process-bounds`.

## 11. Done criteria (strategic)
1. Requesting the language `../../x` is refused, and no file is written.
2. Two simultaneous downloads of `rus` produce one valid pack.
3. The Store build downloads `deu` successfully.
4. A Russian comic in a folder named in Cyrillic gets script detection applied.

## Implementation
Built without a tactical plan.
- **O1** `internal/ocr/download.go` `CheckLang`, called from `internal/app/app.go` (`-ocr-download`),
  `cmd/doc-html-ui/main.go` `handleOCRDownload` and `Download`. Criterion 1:
  `TestDownloadRefusesTraversalCode` (no server hit, nothing written), `TestOCRDownloadRefusesATraversalCode`
  (CLI, exit 1), `TestHandleOCRDownloadRefusesATraversalCode` (GUI, refusal in the page language).
- **O2 / O3** `Download` / `fetchPack` / `verifyPack` / `removeStaleTemps`. Criterion 2:
  `TestConcurrentDownloadsInstallOnePack` (one transfer, one verified pack, no temp file). Also
  `TestDownloadRefusesChecksumMismatch`, `TestDownloadRefusesOversizeBody` (with and without
  Content-Length), `TestDownloadAcceptsAnInstalledVerifiedPack`, `TestDownloadRemovesOnlyStaleTemps`,
  `TestEveryCataloguePackHasADigest`.
- **O4** `internal/ocr/tessdata.go` `UserDataDir` / `DataDirs` / `DataDir` (staging) / `Installed` /
  `IsInstalled`; `-ocr-langs`, the report summary and "Installed into" name the new folders.
  `TestLayeredDataDirs`. Criterion 3 needs the Store build (see Status).
- **O8** `internal/ocr/script.go` `stageForDetection`: orientation and ASCII staging as in
  `prepareForOCR`, without the upscale (the confidence floor was measured on unscaled images), run
  through `runTesseract`. Criterion 4: `TestDetectScriptStagesANonASCIIPath` (a stand-in engine that
  fails on a non-ASCII path, image under `Книга/`).
- **O11** already fixed by ticket 09 (`localImageFile` decodes the src and drops `?`/`#`). The encoded
  case was tested; `TestCollectBookImagesStripsQueryAndFragment` adds the query and fragment case.
- **O12** `internal/ocr/overlay.go` `findInjected` / `isWrapped` and the `data-dht-ocr` marker.
  `TestOverlayTwiceEqualsOnce`, `TestEnsureAssetsRecognizesAnUnmarkedEarlierPass`.
- Localization: `internal/i18n/i18n_ocr.go`.
- Docs: README (download folder), AGENTS.md, docs/PARITY.md "OCR" (download integrity is desktop-only).
- Not done: `scripts/build.ps1` still provisions the bundled eng without checking it against
  `packDigests` (Windows build script, outside this ticket's entry points).

## 12. Next step
The Store-build check in Status, then `/spec-check`.
