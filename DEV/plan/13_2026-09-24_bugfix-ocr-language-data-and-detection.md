# Strategic spec: 13_2026-09-24_bugfix-ocr-language-data-and-detection - Safe language downloads and reliable OCR detection

**Ticket:** 13_2026-09-24_bugfix-ocr-language-data-and-detection
**Status:** Draft
**Priority:** 70
**Date:** 2026-09-24
**Tier:** Moderate
**Tactical plan:** `DEV/plan/13_2026-09-24_bugfix-ocr-language-data-and-detection/` (created by /spec-tech)
**Findings:** O1 O2 O3 O4 O8 O11 O12 (see the [findings register](../research/audit_2026-09-24/README.md))

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
   - **Status:** Open (pinning is simpler; the catalogue is fixed).

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

## 12. Next step
`/spec-tech 13_2026-09-24_bugfix-ocr-language-data-and-detection`
