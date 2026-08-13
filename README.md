# doc-html-translate

**README languages:** **English** · [Русский](README_RU.md) · [Українська](README_UK.md)

Convert EPUB, PDF, MOBI, AZW3, FB2, RTF, TXT, Markdown, HTML and CBZ/CBR/CB7/CBT comics into clean local HTML on Windows - with optional translation through Google Cloud or a local Ollama model. No cloud account required, no ceremony, and yes, it still runs on plain old Windows in 2026.

Topics: `windows` `windows-app` `desktop` `cli` `golang` `epub` `pdf` `mobi` `fb2` `ebook` `html-converter` `translation` `ollama`

## Project Links
- Website: https://serzhyale.github.io/doc-html-translate/
- Repository: https://github.com/SerZhyAle/doc-html-translate
- Latest release: https://github.com/SerZhyAle/doc-html-translate/releases/latest
- Browser extension: https://chromewebstore.google.com/detail/nmcckamdocainafmmompkbmelkpbnmic
- Universal Agent Kit: https://serzhyale.github.io/universal-agent-kit/
- Author page: https://sza.od.ua
- Email: sza@ukr.net

## Editions

doc-html-translate comes in several forms - pick whichever fits; they all share the same converter:

- **CLI** - `doc-html-translate.exe`, the command-line converter and Windows file-association handler. See [Quick Usage](#quick-usage).
- **GUI desktop app** - `doc-html-ui.exe`, a windowed front-end that exposes every CLI option (file picker, drag & drop, options dialog, a **default-handler toggle** - opt-in, off by default).
- **Microsoft Store app** - the same desktop app (GUI + CLI) shipped as an MSIX package: Store-signed, auto-updating, no manual download. Under MSIX, `-register` is a no-op (file associations come from the package manifest). Packaging details: [`msix/README.md`](msix/README.md).
- **Browser extension** - a Chromium MV3 extension that re-renders documents (PDF, EPUB, MOBI, AZW3, FB2, RTF, TXT, Markdown, local HTML, and CBZ/CBT comics) as clean HTML right in the browser, so the built-in **Translate page** works on them without installing the app. Get it on the [Chrome Web Store](https://chromewebstore.google.com/detail/nmcckamdocainafmmompkbmelkpbnmic); source and docs in [`extension/`](extension/) and [`extension/README.md`](extension/README.md). (Edge Add-ons listing planned.)
- **Website & docs** - the [landing page](https://serzhyale.github.io/doc-html-translate/), multi-language documentation, and a dedicated [extension page](https://serzhyale.github.io/doc-html-translate/extension.html).

The desktop app and the extension are independent and complementary: the app converts a file into a local HTML folder you keep; the extension does the same reflow live inside a browser tab. Both lean on the same "free" idea - hand the browser clean HTML and let its built-in translator do the rest.

## Features

- Convert: EPUB, PDF, TXT, Markdown, FB2, RTF, HTML, MOBI, AZW3
- Read comics: CBZ / CBR / CB7 / CBT comic archives open page by page, with the text in speech bubbles recognized (OCR) and laid over each page as translatable plates - so Chrome's "Translate page" works on the bubbles. OCR is automatic (a comic has no text layer to translate otherwise)
- Translate a standalone image: pass a PNG/JPG/JPEG/WebP/GIF/BMP/TIFF and the app OCRs it and lays translatable text plates over the picture (Chrome's built-in page translation then works in place - the same behaviour as the browser extension). OCR needs a `tesseract` engine (see `-ocr-lang`)
- Local HTML output with generated navigation and TOC
- Real multi-level table of contents: imports the authored EPUB2 `toc.ncx`, EPUB3 `nav.xhtml`, or PDF bookmarks; falls back to scanning headings (`h1`-`h6`) and injecting anchors. Rendered as a collapsible tree with deep links; depth is configurable (`-toc-depth`)
- Optional translation:
  - Google Cloud Translation API (`-google`)
  - Local Ollama (`-ollama`)
  - Hard spending guard for paid engines: `-max-cost N` aborts before sending if the estimated cost in USD exceeds `N`
- Reader experience baked into the output HTML (no server, works on `file://`):
  - Reading themes - Light / Sepia / Dark / Night toggle, remembered across sessions
  - Reading position - scroll is saved per book; `index.html` shows a "Continue reading" link, and the navbar carries a thin progress bar
- Interface in 13 languages: `en ru uk de it es fr pt ar hi bn ur zh` - the `-ui-lang <code>` flag in the CLI, a language selector in the GUI and in the extension. The default follows the system language (the browser's language in the extension). The interface language never changes the document's language: the generated page keeps the book's own `<html lang>`, because otherwise Chrome would stop offering to translate the page
- Send logs to the author when something breaks: the app keeps its recent run logs on disk, and the GUI's **About the program** section (or `-report` on the command line) packs them with an environment summary and the last run's settings into one archive, opens the archive's folder with the file selected and puts its path on the clipboard, and opens a pre-addressed message in your own mail program. **Nothing is ever sent automatically** - you attach the archive and press Send yourself, and you can open the archive first to see exactly what it holds. The Google API key is never in it. The browser extension, which has no log store, offers the same intent as a **Copy diagnostics** button on its options page
- Re-open existing extracted book instantly (idempotent behavior - it remembers, so you don't have to)
- File-type association is **optional and off by default**: the app always adds a **"Convert to HTML" right-click entry** (and an "Open with" entry) for all supported types, and becoming the default handler is a separate opt-in (`-register` / the GUI toggle / a one-time first-run prompt; `-unregister` reverses it)
- MOBI/AZW3: requires [Calibre](https://calibre-ebook.com) installed (non-DRM files only)
- CBR/CB7 comics: require [7-Zip](https://www.7-zip.org) installed (CBZ and CBT need nothing extra)

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

- `doc-html-translate-setup-<version>.exe` - **universal installer** (x86 + x64, per-user, no admin) - the easiest option: installs the GUI + CLI, with optional "Open with" + right-click "Convert to HTML" and browser-extension tasks
- `doc-html-translate-<version>-windows-x64.exe` - command-line tool (portable)
- `doc-html-ui-<version>-windows-x64.exe` - GUI desktop app (portable)
- `doc-html-translate-<version>-windows-x64.zip` - full archive (both binaries + LICENSE + README)

The installer runs on both 32- and 64-bit Windows and needs no administrator rights (it installs into your user profile). The portable exe/zip stay available for a no-install workflow.

Install via winget (portable build):

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

# Opt in to becoming the default handler for supported types (off by default)
doc-html-translate.exe -register

# Undo that - release the default-handler association (keeps the right-click entry + "Open with")
doc-html-translate.exe -unregister
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

Why this workflow is popular (besides the obvious):

- Free (no Google Cloud API billing, no invoices to dread)
- Fast to start (single command, no ceremony required)
- Comfortable reading flow in browser with page navigation

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-register` | `false` | Opt in to becoming the default handler in HKCU for all supported types (off by default - the first run only adds the right-click entry and offers this) |
| `-unregister` | `false` | Release the default-handler association (leaves the "Convert to HTML" right-click entry and "Open with") |
| `-register-openwith` | `false` | Add app to the Windows "Open with" list + the "Convert to HTML" right-click menu, without making it the default handler (the `doc-html-ui` GUI does this automatically on launch) |
| `-notranslate` | `false` | Convert only, skip translation |
| `-noopen` | `false` | Do not open browser after conversion |
| `-google` | `false` | Translate via Google Cloud Translation API |
| `-ollama` | `false` | Translate via local Ollama |
| `-free` | `false` | Alias of `-ollama` |
| `-ollama-model` | `gemma3:12b` | Ollama model name |
| `-ollama-parallel` | `1` | Parallel batch requests |
| `-ollama-ctx` | `8192` | Ollama context size |
| `-max-cost` | `0` | Abort paid translation before sending if estimated cost in USD exceeds N (`0` = no limit) |
| `-ocr` | `false` | OCR text inside document images and overlay it as translatable HTML (needs Tesseract) |
| `-ocr-lang` | (`-src`) | OCR language(s), e.g. `eng` or `eng+rus`. Left empty it defaults from `-src` (else `eng`) and the app checks the page's writing system: it adds a language rather than replacing one (`rus+eng`) where the data is installed, and leaves the page without text plates - naming the pack to install - where it is not. Passing the flag turns that check off |
| `-ocr-langs` | `false` | List installed/available OCR languages and exit |
| `-ocr-download` | empty | Download an OCR language pack (e.g. `-ocr-download rus`) and exit |
| `-split` | `5000` | Split pages at N chars (`0` disables split) |
| `-toc-depth` | `0` | Table-of-contents nesting depth on `index.html` (`0` = unlimited, `1` = chapters only) |
| `-multipage` | `false` | Produce multiple HTML pages with a table of contents instead of the default single page |
| `-folder` | empty | Output parent folder |
| `-force` | `false` | Re-extract and re-translate even if output exists |
| `-ui-lang` | empty | Interface language: `en ru uk de it es fr pt ar hi bn ur zh` (empty = follow the system language) |
| `-v` | `false` | Verbose output |
| `-src` | `en` | Source language |
| `-dst` | `ru` | Target language |
| `-report` | `false` | Pack the recent run logs plus an environment summary into an archive for the author, then exit |
| `-version` | `false` | Print version and exit |

## Google API Key

For `-google`, the key is read from the first available of:

1. `google_api.key` next to the executable (unpackaged build), then
2. `%LOCALAPPDATA%\doc-html-translate\google_api.key` (a writable per-user path that also works under the read-only Microsoft Store/MSIX install directory).

Example file contents:

```text
AIzaSy...your_key_here...
```

In `doc-html-ui`, tick **Google Translate** to reveal a key field - paste your key and click **Save** to write it to the per-user path above (no manual file editing needed).

If no usable key is found, the app logs a warning and skips translation - it would rather say so than guess.

## OCR image overlay (`-ocr`)

Text baked into a document's images (scanned pages, comics, screenshots) is invisible to any text
translator. With `-ocr`, the app recognizes that text and overlays it as real, translatable HTML
positioned over each image, so the app's own translation (`-google` / `-ollama`) or the browser's
"Translate page" translates the pictures too. Works for formats whose images reach the HTML stage (EPUB
and PDF); other formats are unaffected.

- **Engine:** the external **Tesseract** binary. The app finds it via `DOCHT_TESSERACT`, then a
  `tesseract\tesseract.exe` next to the app, then `PATH`. If none is found, conversion still completes
  (without overlays) and logs a hint.
- **Languages:** English (`eng.traineddata`) ships with the app and works offline. Other languages are
  downloaded on demand into the app's `tessdata\` folder:
  - `doc-html-translate.exe -ocr-langs` - list installed and available languages.
  - `doc-html-translate.exe -ocr-download rus` - download Russian (etc.).
  - In `doc-html-ui`, use the **Image OCR** section: tick the toggle, pick the OCR language, and use
    **Download** to add languages.
- **Usage:** `doc-html-translate.exe -ocr -src ja -google "manga.pdf"` (OCR Japanese, then translate).
  `-ocr-lang` overrides the OCR language (accepts Tesseract codes like `eng+rus`); by default it follows
  `-src`.

## Behavior Notes

- Output directory name is derived from input filename and sanitized for Windows compatibility.
- Existing extracted output with `index.html` is reused unless `-force` is set.
- Plain-text (`.txt`) input is decoded by sniffing its leading bytes: a UTF-8/UTF-16 byte-order mark first, then valid UTF-8, then a legacy Cyrillic code page (Windows-1251, KOI8-R, CP866) by detection - so a DOS-era or Notepad "Unicode" `.txt` reads as text, not mojibake.
- An unreadable binary (a `.docx`, `.djvu`, or a comic archive with no 7-Zip) is refused with a named format instead of being converted into a garbage document.
- EPUB table-of-contents snippets are generated correctly even when chapter files live under subfolders such as `OEBPS/`.
- The table of contents prefers the book's authored navigation (EPUB2 `toc.ncx` navMap, EPUB3 `nav.xhtml`, or PDF bookmarks) and renders it as a collapsible multi-level tree with deep links. When a document has no authored TOC, headings (`h1`-`h6`) on each page are scanned and given stable `id` anchors so the generated TOC still links into sections. Use `-toc-depth N` to cap the nesting (`0` = unlimited).
- The generated HTML carries a small reader layer: a theme toggle (Light/Sepia/Dark/Night, stored in `localStorage`) and a reading-position tracker (scroll saved per book, a "Continue reading" link on `index.html`, and a progress bar in the navbar). It is pure client-side JS and works on `file://`. Single-page documents (no navbar) do not get this layer.
- For paid engines the estimated cost is `chars / 1e6 * $20`. `-max-cost N` turns the existing advisory dialog into a hard pre-flight guard: if the estimate exceeds `N`, translation is skipped and the book is still produced untranslated.
- PDF extraction is best-effort and includes fallback flows for difficult files (PDFs have opinions, and they are rarely kind).
- In `doc-html-ui`, `Split Size = 0` now matches the CLI and disables page splitting completely.
- `doc-html-ui` file picker and supported-format hints cover all formats, including MOBI/AZW3 (Calibre required) and CBZ/CBR/CB7/CBT comics (CBR/CB7 need 7-Zip).
- In `doc-html-ui`, Google Translate and Ollama are mutually exclusive, and a Google key can be saved directly from the GUI.
- `doc-html-ui` exposes the full CLI surface, including `-toc-depth` and `-max-cost`, plus a **default-handler toggle** (the GUI equivalent of `-register` / `-unregister`; opt-in, off by default) and a one-time first-run prompt offering it. The toggle and prompt are hidden under the Microsoft Store (MSIX) build, where file associations come from the package manifest instead. The GUI always registers the non-destructive "Convert to HTML" right-click entry + "Open with" on launch. If the converter exe is missing next to the GUI, it shows a warning rather than failing silently on Convert.

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

For documents that are **pictures, not text** - screenshots, manga, photographed or scanned pages, the ones this tool politely cannot read - use
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

---

**About the interface translations.** The interface is available in 13 languages. English, Russian and
Ukrainian are author-proofread; the other ten are machine-translated and unproofread - corrections are
welcome at sza@ukr.net.

