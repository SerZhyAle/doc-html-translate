# doc-html-translate

## Project Links
- GitHub Pages: https://serzhyale.github.io/doc-html-translate/
- GitHub Repository: https://github.com/SerZhyAle/doc-html-translate

Windows-focused Go CLI that converts documents to local HTML and optionally translates extracted text.

## Features

- Convert: EPUB, PDF, TXT, Markdown, FB2, RTF, HTML
- Local HTML output with generated navigation and TOC
- Optional translation:
  - Google Cloud Translation API (`-google`)
  - Local Ollama (`-ollama`)
- Re-open existing extracted book instantly (idempotent behavior)
- Optional Windows file association registration (`-register`)

## Installation

Build from source:

```powershell
go build -o build/doc-html-translate.exe ./cmd/doc-html-translate
```

Or use project scripts:

```powershell
./scripts/build.ps1
```

## Download Application

Prebuilt application files are published in the repository build folder:

- https://github.com/SerZhyAle/doc-html-translate/tree/master/build

Direct expected binary path:

- https://github.com/SerZhyAle/doc-html-translate/blob/master/build/doc-html-translate.exe

## Quick Usage

```powershell
# Convert only (no translation)
doc-html-translate.exe -notranslate "book.epub"

# Convert + Google translation
doc-html-translate.exe -google "book.epub"

# Convert + Ollama translation
doc-html-translate.exe -ollama -ollama-model gemma3:12b "book.epub"

# Specify language direction
doc-html-translate.exe -src en -dst ru "book.epub"

# Put output under a custom folder
doc-html-translate.exe -folder "D:\out" "book.pdf"

# Force full rebuild even if output already exists
doc-html-translate.exe -force "book.epub"

# Register as handler in current user registry
doc-html-translate.exe -register
```

## Fastest Free Workflow (Recommended)

The most convenient scenario for many users is:

1. Convert book to local HTML without API translation:

```powershell
doc-html-translate.exe -notranslate "book.epub"
```

or

```powershell
doc-html-translate.exe -notranslate "book.pdf"
```

2. Let the tool open `index.html` in Chrome.
3. Use Chrome built-in page translation to your language.

Why this workflow is popular:

- Free (no Google Cloud API billing)
- Fast to start (single command)
- Comfortable reading flow in browser with page navigation

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-register` | `false` | Register app as document handler in HKCU |
| `-notranslate` | `false` | Convert only, skip translation |
| `-noopen` | `false` | Do not open browser after conversion |
| `-google` | `false` | Translate via Google Cloud Translation API |
| `-ollama` | `false` | Translate via local Ollama |
| `-free` | `false` | Alias of `-ollama` |
| `-ollama-model` | `gemma3:12b` | Ollama model name |
| `-ollama-parallel` | `1` | Parallel batch requests |
| `-ollama-ctx` | `8192` | Ollama context size |
| `-split` | `5000` | Split pages at N chars (`0` disables split) |
| `-folder` | empty | Output parent folder |
| `-force` | `false` | Re-extract and re-translate even if output exists |
| `-v` | `false` | Verbose output |
| `-src` | `en` | Source language |
| `-dst` | `ru` | Target language |
| `-version` | `false` | Print version and exit |

## Google API Key

For `-google`, place `google_api.key` next to the executable.

Example file contents:

```text
AIzaSy...your_key_here...
```

If the file is missing/empty, the app logs a warning and skips translation.

## Behavior Notes

- Output directory name is derived from input filename and sanitized for Windows compatibility.
- Existing extracted output with `index.html` is reused unless `-force` is set.
- PDF extraction is best-effort and includes fallback flows for difficult files.

## Development

```powershell
./scripts/test.ps1
./scripts/lint.ps1
./scripts/check.ps1
```

Main entry points:

- `cmd/doc-html-translate/main.go`
- `internal/pipeline/pipeline.go`
- `internal/pdf/extract.go`
- `internal/translator/translator.go`

## License

This project is licensed under the MIT License. See `LICENSE` for details.

