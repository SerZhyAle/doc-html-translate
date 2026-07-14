# Integration contract: translate an image with doc-html-translate

Audience: developers embedding a **"Translate image"** button (e.g. FastMediaSorter Lite).
Goal: given a local image file, show the picture in the browser with its text OCR'd and laid
over the image as real, translatable HTML, so the browser's built-in **"Translate to.."**
(Chrome / Edge) works in place. No API key, no account.

This is the same mechanism the browser extension uses, exposed to any program as a plain
command-line call to `doc-html-translate.exe`.

---

## 1. The call

```
doc-html-translate.exe [flags] <path-to-image>
```

- The image path is a **positional argument** (not a flag). Quote it if it contains spaces.
- Supported image types: `.png .jpg .jpeg .webp .gif .bmp .tif .tiff`.
- For an image, **OCR + overlay is automatic** - you do not need `-ocr`.
- With no translation flag (`-google` / `-ollama`), the app does **not** translate itself; it
  just opens the page, and the browser translates. That is the intended "free" flow here.

### Recommended invocation

```
doc-html-translate.exe -ocr-lang <langs> "C:\path\to\photo.jpg"
```

### Flags that matter for this use case

| Flag | Purpose | Recommendation |
|------|---------|----------------|
| `-ocr-lang <codes>` | Tesseract language(s) of the text in the image, e.g. `eng`, `rus`, `eng+rus`, `jpn`. Drives OCR accuracy. | Set it to the expected language. If omitted it derives from `-src`, else `eng`. |
| `-src <lang>` | Source language hint (also the fallback for `-ocr-lang`). | Optional; `-ocr-lang` is more direct. |
| `-folder <dir>` | Output directory. Default: a subfolder next to the image, named after it. | Point at a temp/cache dir if you don't want files next to the user's images. |
| `-force` | Re-OCR even if a previous result exists. Without it, a repeat call on the same image just re-opens the cached page (fast, no re-OCR). | Add `-force` only if you want a fresh pass every click. |
| `-noopen` | Convert without opening the browser (batch). | **Do not** use for the button - you want the browser to open. |

Everything else (`-google`, `-ollama`, `-split`, `-toc-depth`, ..) is irrelevant to the image flow.

### What happens

1. The image is copied into the output folder and wrapped in `index.html`.
2. Tesseract OCRs it; recognized text becomes opaque, position-matched plates over the picture.
3. `index.html` is opened via the OS default handler (`cmd /c start "" <index.html>`) - i.e. the
   **default browser**. For the translation offer to appear, the default browser should be
   **Chrome or Edge**.

> The plates are real HTML text nodes, so page-translate rewrites them in place. If Tesseract is
> unavailable, the page still opens and shows the image unchanged (overlay is best-effort, never fatal).

### Output location

For input `C:\pics\cat.png` (no `-folder`): output is `C:\pics\cat\index.html` (+ `page_001.html`
+ the copied image). Re-running reuses it unless `-force` is passed.

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success |
| `1` | Bad arguments (e.g. no input file) |
| `2` | I/O error |
| `3` | Parse/extract error |
| `4` | Translation API error (not reachable in the free image flow) |

---

## 2. Dependency: the Tesseract OCR engine

The portable / winget build **does not bundle Tesseract or language data** - only the two exes.
The app locates the engine in this order:

1. Env var **`DOCHT_TESSERACT`** = full path to `tesseract.exe` (most reliable for embedding).
2. A `tesseract\tesseract.exe` folder **next to** `doc-html-translate.exe`.
3. `tesseract` on **PATH**.

Language data (`*.traineddata`) comes from that Tesseract install's own `tessdata`. To add a
language into the app's own data dir on demand:

```
doc-html-translate.exe -ocr-download rus      # downloads rus.traineddata, then exits
doc-html-translate.exe -ocr-langs             # lists installed + available languages, then exits
```

**For the button to produce plates, ensure Tesseract is installed and findable** (bundle it and
set `DOCHT_TESSERACT`, or install e.g. the UB-Mannheim Tesseract build). Without it the user still
sees the image, just no translatable text.

---

## 3. Is the app installed? (detection)

The winget/portable install registers command aliases on PATH. Detect with:

```powershell
# PowerShell
$exe = (Get-Command doc-html-translate -ErrorAction SilentlyContinue).Source
```

```cmd
:: cmd
where doc-html-translate
```

If found, call it by that name (or full path). If not found, install it (next section).

> If you ship your own copy of `doc-html-translate.exe` alongside FastMediaSorter, skip detection
> entirely and call it by its absolute path - that is the most robust option.

---

## 4. If absent: initialize installation

### Preferred - winget (silent, registers the `doc-html-translate` command)

```
winget install --id SerZhyAle.DocHtmlTranslate -e --source winget --accept-package-agreements --accept-source-agreements
```

- Installs the portable package and puts `doc-html-translate` / `doc-html-ui` aliases in
  `%LOCALAPPDATA%\Microsoft\WinGet\Links` (on PATH).
- **PATH note:** a process started *before* the install won't see the new alias. After installing,
  either call the exe by full path, or re-resolve PATH / relaunch, or restart your process.

### Fallback - direct download (no winget, no PATH dependency)

Download and extract the latest release zip, then call the exe by absolute path:

- Latest release page: `https://github.com/SerZhyAle/doc-html-translate/releases/latest`
- Asset name pattern: `doc-html-translate-<version>-windows-x64.zip`
  (contains `doc-html-translate.exe`, `doc-html-ui.exe`, `LICENSE`, `README.md`).

This is the recommended path if you want zero external state - bundle or fetch the exe and invoke it
directly.

### Microsoft Store - NOT suitable for programmatic launch

The Store (MSIX) build is the same app but is **GUI-first**: it exposes **no command-line alias**
and its file associations **do not include image types**. Use it only for human install, not for a
button. Product page (for a "Get the app" link): `https://apps.microsoft.com/detail/9PMHSWQPR6V1`.

---

## 5. Suggested button logic (pseudocode)

```text
onTranslateImageClicked(imagePath):
    exe = resolveDocHtmlTranslate()          # bundled abs path, else `where doc-html-translate`
    if exe is null:
        if userAcceptsInstall():
            run: winget install --id SerZhyAle.DocHtmlTranslate -e --source winget \
                     --accept-package-agreements --accept-source-agreements
            exe = resolveDocHtmlTranslate()   # re-resolve (or use full path from WinGet\Links)
        else:
            openInBrowser("https://apps.microsoft.com/detail/9PMHSWQPR6V1")   # let the user install
            return

    lang = mapUiLanguageToTesseract()         # e.g. "eng", "rus", "eng+rus"
    exitCode = run(exe, ["-ocr-lang", lang, imagePath])   # opens the browser on success
    if exitCode != 0:
        showError("Conversion failed (code " + exitCode + ")")
```

Notes:
- Run it non-blocking; it returns after launching the browser.
- Set `DOCHT_TESSERACT` in the child process environment if you bundle Tesseract.
- One click = one call. No stdin, no server, no ports.

---

## Quick copy-paste example

```
set DOCHT_TESSERACT=C:\Program Files\Tesseract-OCR\tesseract.exe
doc-html-translate.exe -ocr-lang eng+rus "C:\Users\me\Pictures\scan.jpg"
```

Opens `C:\Users\me\Pictures\scan\index.html` in the default browser with translatable text plates
over the scan.
