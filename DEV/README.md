# doc-html-translate (DEV Notes)

## Project Links
- GitHub Pages: https://serzhyale.github.io/doc-html-translate/
- GitHub Repository: https://github.com/SerZhyAle/doc-html-translate

Developer-oriented README for local builds and maintenance.

## Current CLI Snapshot

```powershell
doc-html-translate.exe "book.epub"
doc-html-translate.exe "book.mobi"           # requires Calibre
doc-html-translate.exe "book.azw3"           # requires Calibre
doc-html-translate.exe -google "book.epub"
doc-html-translate.exe -ollama -ollama-model gemma3:12b "book.epub"
doc-html-translate.exe -src en -dst ru "book.epub"
doc-html-translate.exe -folder "D:\out" "book.pdf"
doc-html-translate.exe -force "book.epub"
doc-html-translate.exe -version
```

## Download Source For App Binary

Use GitHub build folder as the download source:

- https://github.com/SerZhyAle/doc-html-translate/tree/master/build
- https://github.com/SerZhyAle/doc-html-translate/blob/master/build/doc-html-translate.exe

## Most Convenient User Scenario

Default support recommendation for end users:

1. Open the file with the app or run the default command:

```powershell
doc-html-translate.exe "book.epub"
```

or

```powershell
doc-html-translate.exe "book.pdf"
```

2. Open result in Chrome.
3. Translate the page in Chrome to the user's target language.

`-notranslate` remains available as an explicit convert-only flag, but it matches the default flow unless `-google` or `-ollama` is provided.

Benefits:

- Free and quick (no key setup, no paid API calls)
- Good UX for long-form reading with generated navigation

## Key Flags

The full, current flag table lives in [../README.md](../README.md#flags) (single source of truth -
this file used to duplicate it and drifted). The canonical defaults are also pinned in
[../docs/PARITY.md](../docs/PARITY.md#settings-defaults). `internal/config/flags.go` is the code of
record.

## Google API Key (Current Behavior)

Google translation reads key from `google_api.key` in the executable directory.

- Missing or empty key file does not abort conversion.
- Translation step is skipped with a warning.

Development key location used in this repo:

- `DEV/private/google_api.key`

## Recent Behavior Notes

- Output folder names are sanitized to avoid invalid Windows paths.
- Existing extracted books are reused when `index.html` exists (unless `-force`).
- PDF extraction uses best-effort fallback logic for problematic files.

## Dev Commands

```powershell
./scripts/build.ps1
./scripts/test.ps1
./scripts/lint.ps1
./scripts/check.ps1
./scripts/build-installer.ps1        # universal setup.exe (x86 + x64, per-user; needs Inno Setup / ISCC)
```

Short aliases are available via `./a.ps1 <cmd>` (e.g. `a ch` = check gate, `a bl` = gated local build,
`a inst` = build the installer, `a c` / `a cc` = quick commit +/- push, `a id` = local MSIX reinstall).
Run `./a.ps1 help` for the full list.

## Core Code Areas

- `cmd/doc-html-translate/main.go`
- `internal/pipeline/pipeline.go`
- `internal/pdf/extract.go`
- `internal/translator/translator.go`

