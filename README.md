# doc-html-translate

Convert EPUB, PDF, MOBI, AZW3, FB2, RTF, TXT, Markdown and HTML documents into clean local HTML on Windows - with optional translation through Google Cloud or a local Ollama model.

Topics: `windows` `windows-app` `desktop` `cli` `golang` `epub` `pdf` `mobi` `fb2` `ebook` `html-converter` `translation` `ollama`

## Project Links
- Website: https://serzhyale.github.io/doc-html-translate/
- Repository: https://github.com/SerZhyAle/doc-html-translate
- Latest release: https://github.com/SerZhyAle/doc-html-translate/releases/latest
- Universal Agent Kit: https://serzhyale.github.io/universal-agent-kit/
- Author page: https://sza.od.ua
- Email: sza@ukr.net

## Features

- Convert: EPUB, PDF, TXT, Markdown, FB2, RTF, HTML, MOBI, AZW3
- Local HTML output with generated navigation and TOC
- Real multi-level table of contents: imports the authored EPUB2 `toc.ncx`, EPUB3 `nav.xhtml`, or PDF bookmarks; falls back to scanning headings (`h1`–`h6`) and injecting anchors. Rendered as a collapsible tree with deep links; depth is configurable (`-toc-depth`)
- Optional translation:
  - Google Cloud Translation API (`-google`)
  - Local Ollama (`-ollama`)
  - Hard spending guard for paid engines: `-max-cost N` aborts before sending if the estimated cost in USD exceeds `N`
- Reader experience baked into the output HTML (no server, works on `file://`):
  - Reading themes - Light / Sepia / Dark / Night toggle, remembered across sessions
  - Reading position - scroll is saved per book; `index.html` shows a "Continue reading" link, and the navbar carries a thin progress bar
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

# Cap paid (Google) translation: skip if the estimate exceeds $2.00
doc-html-translate.exe -google -max-cost 2 "book.epub"

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
| `-max-cost` | `0` | Abort paid translation before sending if estimated cost in USD exceeds N (`0` = no limit) |
| `-split` | `5000` | Split pages at N chars (`0` disables split) |
| `-toc-depth` | `0` | Table-of-contents nesting depth on `index.html` (`0` = unlimited, `1` = chapters only) |
| `-folder` | empty | Output parent folder |
| `-force` | `false` | Re-extract and re-translate even if output exists |
| `-v` | `false` | Verbose output |
| `-src` | `en` | Source language |
| `-dst` | `ru` | Target language |
| `-version` | `false` | Print version and exit |

## Google API Key

For `-google`, the key is read from the first available of:

1. `google_api.key` next to the executable (unpackaged build), then
2. `%LOCALAPPDATA%\doc-html-translate\google_api.key` (a writable per-user path that also works under the read-only Microsoft Store/MSIX install directory).

Example file contents:

```text
AIzaSy...your_key_here...
```

In `doc-html-ui`, tick **Google Translate** to reveal a key field — paste your key and click **Save** to write it to the per-user path above (no manual file editing needed).

If no usable key is found, the app logs a warning and skips translation.

## Behavior Notes

- Output directory name is derived from input filename and sanitized for Windows compatibility.
- Existing extracted output with `index.html` is reused unless `-force` is set.
- EPUB table-of-contents snippets are generated correctly even when chapter files live under subfolders such as `OEBPS/`.
- The table of contents prefers the book's authored navigation (EPUB2 `toc.ncx` navMap, EPUB3 `nav.xhtml`, or PDF bookmarks) and renders it as a collapsible multi-level tree with deep links. When a document has no authored TOC, headings (`h1`–`h6`) on each page are scanned and given stable `id` anchors so the generated TOC still links into sections. Use `-toc-depth N` to cap the nesting (`0` = unlimited).
- The generated HTML carries a small reader layer: a theme toggle (Light/Sepia/Dark/Night, stored in `localStorage`) and a reading-position tracker (scroll saved per book, a "Continue reading" link on `index.html`, and a progress bar in the navbar). It is pure client-side JS and works on `file://`. Single-page documents (no navbar) do not get this layer.
- For paid engines the estimated cost is `chars / 1e6 * $20`. `-max-cost N` turns the existing advisory dialog into a hard pre-flight guard: if the estimate exceeds `N`, translation is skipped and the book is still produced untranslated.
- PDF extraction is best-effort and includes fallback flows for difficult files.
- In `doc-html-ui`, `Split Size = 0` now matches the CLI and disables page splitting completely.
- `doc-html-ui` file picker and supported-format hints cover all formats, including MOBI/AZW3 (Calibre required).
- In `doc-html-ui`, Google Translate and Ollama are mutually exclusive, and a Google key can be saved directly from the GUI.

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

## Companion App: FastMediaSorter LITE

For documents that are **pictures, not text** - screenshots, manga, photographed or scanned pages - use
**FastMediaSorter LITE**, a free Windows app for opening and sorting images and videos with built-in **OCR + on-image translation**.
Press `T` on any image to recognize the text and overlay the translation in your language (local Ollama or
LibreTranslate). It complements doc-html-translate, which targets ebook and text formats.

Also available via winget and GitHub:

```powershell
winget install SerZhyAle.FastMediaSorter
```

- Repository: https://github.com/SerZhyAle/FastMediaSorter_Lite
- Latest release: https://github.com/SerZhyAle/FastMediaSorter_Lite/releases/latest

This project is also listed in the Universal Agent Kit collection:

- https://serzhyale.github.io/universal-agent-kit/

## License

This project is licensed under the MIT License. See `LICENSE` for details.

