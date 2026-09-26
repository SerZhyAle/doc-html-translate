# Desktop OCR fails under a non-ANSI profile, and the GUI never checks the script

**Status:** BlockNeedUserTest - implemented 2026-09-26; left: non-ANSI Windows profile, 8.3 names on and off, OCR after `-ocr-download rus`
**Priority:** 70
**Date:** 2026-09-26

> Filed by the pre-release audit, [ticket 34](34_2026-09-25_full-code-audit-pre-release.md). Finding ids
> refer to its register, [`FINDINGS.md`](34_2026-09-25_full-code-audit-pre-release/FINDINGS.md).

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

## Implementation (2026-09-26)

**Contract check.** `OCR-INVOCATION` 1.1 pins the flags, exit codes, engine lookup and "an omitted
`-ocr-lang` derives from `-src`, else `eng`". Nothing here changes a flag, an exit code or the lookup;
the `-src` mapping now yields `eng` where it used to yield a pack the catalog lacks, which is the
contract's own "else `eng`". The GUI is not part of the invocation. No catalog amendment needed.

**O15 - ASCII-safe Tesseract paths.**
- New [`internal/ocr/asciipath.go`](../../../internal/ocr/asciipath.go) (portable) with
  `asciipath_windows.go` / `asciipath_nonwindows.go`. A path goes to the engine as is when ASCII, else as
  its 8.3 short name (`GetShortPathNameW`, Windows only) when that is ASCII. Temp images are written under
  a staging root resolved once per process: the system temp folder, then `%PUBLIC%` / `%ProgramData%`
  (Windows) or `/tmp` / `/var/tmp` (elsewhere), then a folder next to the exe, each under
  `doc-html-translate-ocr`, taking the first that exists, has an ASCII form and is writable.
- `--tessdata-dir`: `PrepareEngine` returns the data folder, its short name, or a mirror of its packs under
  `<staging root>/doc-html-translate-tessdata` (copied once, reused at the same size).
- [`internal/pipeline/ocrstep.go`](../../../internal/pipeline/ocrstep.go) calls it once per book; when no ASCII
  root exists OCR is skipped with a line naming every folder tried.
- `writeTempPNG` / `stageASCIIPath` / `isASCIIPath` moved out of `tesseract.go` into the new file, which
  covers the upscale, rotate, rescue, screen and script-detection passes.
- Tests: `asciipath_test.go` - `TestStagingRootSkipsANonASCIITempFolder`,
  `TestStagedImagesAvoidANonASCIITempFolder` (TMPDIR set to a Cyrillic folder),
  `TestPrepareEngineMirrorsANonASCIIDataFolder`, `TestPrepareEngineNamesTheFolderWhenNothingIsUsable`.
  Each skips with the reason if the file system cannot hold a Cyrillic name.
- **Owner to verify on Windows:** a profile named outside the ANSI code page, on a volume with and
  without 8.3 names (`fsutil 8dot3name query C:`). Run an OCR conversion that upscales, then one after
  `-ocr-download rus`. Check that plates appear, and that `%PUBLIC%\doc-html-translate-ocr` is only
  used when 8.3 names are off. The short-name call itself is compiled and vetted
  (`GOOS=windows go vet`) but has not run on Windows here.

**O16 - GUI sends `-ocr-lang` only on an explicit choice.**
- [`cmd/doc-html-ui/ui.html`](../../../cmd/doc-html-ui/ui.html): the OCR select now starts with an
  **Automatic (source language)** entry (value `""`, i18n key `ocrLangAuto` replacing the now unused
  `ocrEngDefault` in all 13 dictionaries of `i18n.js`). The `eng` fallback and the copy of the `-src`
  language into the select are gone. `syncOcrLangToSource` now only reports a missing `-src` pack and
  preselects its download.
- The choice is saved as `ocrLangChoice`. An older saved `ocrLang` equal to `eng`, or to the `-src`
  language the page filled in by itself, falls back to Automatic. Any other saved value was a real pick
  and is kept (`applyOcrWant`).
- Tests: `TestAssembleArgsSendsNoOCRLangForTheAutomaticChoice` (the GUI argument test: `-ocr` with no
  `-ocr-lang`, and the CLI parses an empty `OCRLang`) and `TestUIDefaultsTheOCRLanguageToAutomatic`
  (markup) in [`cmd/doc-html-ui/main_test.go`](../../../cmd/doc-html-ui/main_test.go).
- Also checked in headless Chromium: the page was served with stubbed APIs and its `/api/preview` body
  recorded for fresh, legacy and new settings. Pre-fix: 5 of 6 cases sent an explicit language. Post-fix:
  all 6 as intended.
- **Owner decision to confirm:** keeping a saved legacy pick only when the page could not have made it
  itself is a judgement call (a user who deliberately chose `eng` goes back to Automatic once).

**O17 - mapping and advice agree with the catalog.**
- [`internal/ocr/tessdata.go`](../../../internal/ocr/tessdata.go): `iso2tess` lists only catalog languages
  (`nld` / `tur` / `ara` dropped). `TessLang` maps a region subtag to its language (`pt-BR` -> `por`,
  `zh-CN` / `zh-TW` -> `chi_sim`) and passes a catalog Tesseract name as is. Anything else gives `eng`,
  which the script check can still correct. A `+`-joined value keeps the parts that derive a pack.
- `MissingAdvice` offers `-ocr-download` only for a catalog code. For any other code it says the app has
  no download for it.
- Tests: `TestTessLangDerivesOnlyCatalogLanguages` checks that every derived code passes `CheckLang`.
  `TestMissingAdviceOffersOnlyCatalogDownloads` covers the advice.
- [`docs/PARITY.md`](../../../docs/PARITY.md) ("Default OCR language rule") and `README.md` are updated.

**O18 - a pack that could not be staged is named.**
- `DataDir()` is replaced by `DataDirFor(lang)`. It still stages the bundled packs into the per-user folder
  when both hold data, and it records each failed copy. A failure matters only if `lang` needs that pack:
  the bundled folder is used when it serves `lang` alone. Otherwise the error names the pack, both folders
  and the cause, and `ocrstep` skips OCR with that line instead of failing every image.
- `hasLangFile` now accepts only a regular file, so a folder named like a pack no longer counts as one.
- `tools/ocrlab` follows the new signature.
- Tests: `TestDataDirForNamesAPackThatCouldNotBeStaged`; `TestLayeredDataDirs` is updated.

**Checks:** `go build ./...`, `go vet ./...`, `GOOS=windows go vet ./...` and `go test ./...` all pass
(41 packages ok). `npm test` in `extension/`: 271 pass, 0 fail.
