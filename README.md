# doc-html-translate

Convert EPUB, PDF, MOBI, AZW3, FB2, RTF, TXT, Markdown and HTML documents into clean local HTML on Windows - with optional translation through Google Cloud or a local Ollama model.

Topics: `windows` `windows-app` `desktop` `cli` `golang` `epub` `pdf` `mobi` `fb2` `ebook` `html-converter` `translation` `ollama`

## Project Links
- Website: https://serzhyale.github.io/doc-html-translate/
- Repository: https://github.com/SerZhyAle/doc-html-translate
- Latest release: https://github.com/SerZhyAle/doc-html-translate/releases/latest

## Features

- Convert: EPUB, PDF, TXT, Markdown, FB2, RTF, HTML, MOBI, AZW3
- Local HTML output with generated navigation and TOC
- Optional translation:
  - Google Cloud Translation API (`-google`)
  - Local Ollama (`-ollama`)
- Re-open existing extracted book instantly (idempotent behavior)
- Optional Windows file association registration (`-register`)
- MOBI/AZW3: requires [Calibre](https://calibre-ebook.com) installed (non-DRM files only)

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

Prebuilt Windows x64 binaries are published on the Releases page:

- https://github.com/SerZhyAle/doc-html-translate/releases/latest

Each release contains:

- `doc-html-translate-<version>-windows-x64.exe` - command-line tool
- `doc-html-ui-<version>-windows-x64.exe` - GUI desktop app
- `doc-html-translate-<version>-windows-x64.zip` - full archive (both binaries + LICENSE + README)

Install via winget:

```powershell
winget install SerZhyAle.DocHtmlTranslate
```

## Quick Usage

```powershell
# Default open flow: convert + open in browser (no translation unless -google or -ollama is set)
doc-html-translate.exe "book.epub"

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

1. Open the file with the app or run the default command:

```powershell
doc-html-translate.exe "book.epub"
```

or

```powershell
doc-html-translate.exe "book.pdf"
```

2. Let the tool open `index.html` in Chrome.
3. Use Chrome built-in page translation to your language.

`-notranslate` is still available, but it is only the explicit form of the default non-API flow.

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

