# Pointer: OCR-INVOCATION

- **Id:** `OCR-INVOCATION`
- **Version:** 1.0
- **Home:** the shared contracts catalog, `ocr-overlay/integration-image-translate.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** producer and owner - this repo ships the executable the contract describes
- **Declared consumer:** FastMediaSorter_Lite ("Translate image" button)

How another program hands one image to `doc-html-translate.exe` and gets a browser page whose OCR'd text
is real, translatable HTML: the positional argument, the flags that matter, the output location, the exit
codes, how the Tesseract engine is located, how to detect the app and how to install it.

**What this repo owes it**

- The call shape is frozen for a consumer that shipped against it. A renamed flag, a changed default, a
  moved output path or a re-used exit code is a **breaking** change to this contract: amend the catalog
  document first, with a version bump, then change the code.
- The surfaces the contract pins, and where they live here:
  - flags `-ocr-lang` / `-src` / `-folder` / `-force` / `-noopen` / `-ocr-langs` / `-ocr-download` -
    [`../../internal/config/flags.go`](../../internal/config/flags.go);
  - exit codes `0` ok, `1` bad arguments, `2` I/O, `3` parse, `4` translation API -
    [`../../internal/pipeline/pipeline.go`](../../internal/pipeline/pipeline.go);
  - engine lookup order `DOCHT_TESSERACT` -> `tesseract/` next to the exe -> `PATH` -
    [`../../internal/ocr/tesseract.go`](../../internal/ocr/tesseract.go) (`Locate`);
  - the accepted image extensions - [`../../internal/img/extract.go`](../../internal/img/extract.go);
  - the winget package id and the Store product id, which are frozen anchors of the release flow.
- The default for an omitted `-ocr-lang` (derive from `-src`, else `eng`) is part of the contract, not a
  local decision: change it there first.

**Conformance.** No vectors in the catalog. The observable surfaces above are covered by this repo's own
tests ([`../../internal/img/`](../../internal/img/), [`../../tests/`](../../tests/)); an end-to-end call
from a consumer is not exercised here.
