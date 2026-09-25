# Security posture - what this product can touch and what it can send

<!-- Rendered from docs/security-posture.json by scripts/security-posture.ps1 -Render. Do not edit by hand. -->

**Last reconciled:** 2026-09-25
**Contract:** canon `SECURITY_AND_PRIVACY` section 7 - the permission and network-surface inventories.
**Checked by:** `scripts/security-posture.ps1`, run by `scripts/check.ps1`, whose evidence `scripts/release.ps1` requires before the tag step.

Two editions share no code: the Windows app (CLI, GUI, MSIX) and the browser extension. Every public form of the privacy promise is rendered from these rows - the list below says which - so a row and a sentence cannot disagree without the check going red.

## 1. Permission inventory

One row per declared permission, and, for the app, which has no permission manifest, one row per capability it takes (section 7 item 8). *Shown at request* says whether the sentence is shown at the moment the permission is requested.

| Row | Edition | Declares | Declared in | Consumers | Shown at request | Where the sentence is shown |
|---|---|---|---|---|---|---|
| `ext-declarativenetrequest` | extension | `permission:declarativeNetRequest` | extension/manifest.json, permissions | the dynamic rules that send a page load of a registered document format, over http, https or file, to the bundled viewer; the session rule that lets Open original load the file once in the browser's own viewer | no | the browser's own install prompt; the sentence is on the store privacy forms and the privacy pages |
| `ext-host-all-urls` | extension | `host:<all_urls>` | extension/manifest.json, host_permissions | the viewer fetches the opened document's bytes, from whatever origin serves it, with the reader's cookies for that site; the OCR actions fetch the image the user picked, or the pictures of the page the user started them on; the popup reads the active tab's host name for the per-site on/off switch | no | the browser's own install prompt (read and change data on all sites); the sentence is on the store privacy forms and the privacy pages |
| `ext-scripting` | extension | `permission:scripting` | extension/manifest.json, permissions | OCR every image on this page: on that click one script and its stylesheet are injected into that one tab, and removed when the user stops | no | the browser's own install prompt; the sentence is on the store privacy forms and the privacy pages |
| `ext-offscreen` | extension | `permission:offscreen` | extension/manifest.json, permissions | the OCR engine for OCR every image on this page runs in one offscreen document, created for a run and closed when it ends | no | nowhere at request time - the browser shows no prompt for it; the sentence is on the store privacy forms and the privacy pages |
| `ext-contextmenus` | extension | `permission:contextMenus` | extension/manifest.json, permissions | OCR & translate this image, on an image; OCR every image on this page, on a page; Convert with doc-html-translate, on a document link or page | no | the browser shows no prompt for it; the menu items themselves are the visible use |
| `ext-storage` | extension | `permission:storage` | extension/manifest.json, permissions | the settings: the viewer on or off, globally and per site; reading preferences; the interface language; whether remote images may load; the list of downloaded OCR languages; the summary of the most recent document that Copy diagnostics reads | no | the browser shows no prompt for it; the sentence is on the store privacy forms and the privacy pages |
| `app-msix-runfulltrust` | app | `msix:runFullTrust` | msix/AppxManifest.xml, Capabilities (Store package only) | the whole desktop app runs as a full-trust Win32 program: every capability row below depends on it | no | the Store shows Windows' own line for it on the listing; the justification goes to Partner Center certification |
| `app-read-documents` | app | `capability:read-opened-documents` | no permission manifest: the Win32 process reads what the user opens (MSIX: under runFullTrust) | every format extractor reads the file the user opens or drops on the app | no | no prompt: the user opens the file |
| `app-open-browser` | app | `capability:open-browser` | no permission manifest: ShellExecute of the converted index.html, and the GUI's app window | the converted book opens in the default browser; the desktop GUI shows its window as a browser app window | no | no prompt: opening the result is the command's visible outcome |
| `app-run-helpers` | app | `capability:run-helper-programs` | no permission manifest: child processes through internal/procrun | the bundled pdftotext for PDF text; Tesseract for OCR, Calibre for MOBI and AZW3, 7-Zip for CBR and CB7, ffmpeg or ImageMagick for some PDF images - each only where the user installed it | no | no prompt: a missing helper is named in the run's own message |
| `app-explorer-registration` | app | `capability:explorer-registration` | no permission manifest: per-user keys under HKCU\Software\Classes (internal/windowsreg); the MSIX declares its own file types instead | the default-handler association for the supported document types; the right-click Convert to HTML verb and the Open with entry | yes | the no-argument registration prompts in the console, and the Windows integration section of the GUI |
| `app-write-output` | app | `capability:write-output-folder` | no permission manifest: the output folder beside the source or chosen with -folder | the converted pages, their assets and index.html; the completion record that decides whether an unchanged book is reopened or rebuilt | no | no prompt: the run prints the output path |
| `app-user-folder` | app | `capability:per-user-app-folder` | no permission manifest: %LOCALAPPDATA%\doc-html-translate (redirected into the package container under MSIX) | the GUI settings and the optional saved Google API key; the recent run logs and the report archives; downloaded OCR language data; the unpacked copy of the bundled pdftotext | no | no prompt: the GUI names the key's path when it saves it |

## 2. Network-surface inventory

One row per surface that opens a listening port, initiates an outbound connection, or hands a file outward.

| Row | Edition | Kind | Surface | On by default | Turned on by | Lifetime | What leaves, to where |
|---|---|---|---|---|---|---|---|
| `net-app-gui-server` | app | listening | the GUI's HTTP server on 127.0.0.1, an ephemeral port | yes | launching doc-html-ui - the GUI window is the session, and the server is how the window talks to the program | the GUI window: a heartbeat watchdog shuts the server down and exits when the window stops pinging | nothing leaves the machine; the loopback address is never advertised, and every call needs the per-launch token baked into the page and passes a Host and Origin guard |
| `net-app-google` | app | outbound | HTTPS to translation.googleapis.com (Google Cloud Translation API v2) | no | -google, or Google Cloud as the engine in the GUI, with the user's own API key; a set -max-cost is checked before any request, and without one a dialog asks above 1000 characters | the conversion run | the extracted text of the book (pages, title, table-of-contents labels), to Google, with the user's key in the X-Goog-Api-Key header |
| `net-app-ollama` | app | outbound | HTTP to localhost:11434, the Ollama server the user runs | no | -ollama, or Ollama (local) as the engine in the GUI | the conversion run | the extracted text, to the user's own machine only; the address is fixed at localhost |
| `net-app-ocr-languages` | app | outbound | HTTPS GET to github.com/tesseract-ocr/tessdata_fast (raw files, tag 4.0.0) | no | -ocr-download with a language code, or Download beside a language in the GUI; catalogue codes only | one download; the file is verified against a pinned SHA-256 digest and kept in the per-user folder | a request naming one language data file; no user content |
| `net-app-report-mail` | app | hand-off | a mailto: link opened in the user's mail program, with the archive's path on the clipboard | no | Send logs to the author in the GUI; -report on the command line only writes the archive | one action | nothing by itself: the mail program shows an unsent message to the author; the redacted archive stays on disk until the user attaches it and presses Send |
| `net-ext-document` | extension | outbound | a fetch of the opened document from its own URL (http, https or file) | yes | opening a supported document while the viewer is on for that site | the viewer tab | an ordinary request for that document to the site that serves it, with the reader's cookies for that site |
| `net-ext-images` | extension | outbound | fetches of the pictures the reader asks the extension to recognize | no | OCR & translate this image, or OCR every image on this page, from the right-click menu | that recognition run | requests for those images to the sites that serve them; the recognition itself runs on the device |
| `net-ext-remote-content` | extension | outbound | images and media a rendered document itself points at on the internet | no | Load them in the viewer, for one document, or the Load remote images in documents option | that document view, or until the option is turned off | requests to the hosts the document names, which tells them the document was opened |
| `net-ext-ocr-languages` | extension | outbound | HTTPS to tessdata.projectnaptha.com (tessdata_fast 4.0.0), through the bundled Tesseract engine | no | Download beside an OCR language in the options | one download, then cached in the browser's storage for reuse | a request naming one language data file; no user content |

Every source file under these roots that contains a network primitive is the evidence of a row above, or is listed here as not reaching the network:

- `cmd/**/*.go` - `\bhttp\.(Get|Post|Head|PostForm|NewRequest|NewRequestWithContext)\(`, `\bnet\.(Listen|Dial|DialTimeout|ListenPacket)\(`, `&http\.Client\{`
- `internal/**/*.go` - `\bhttp\.(Get|Post|Head|PostForm|NewRequest|NewRequestWithContext)\(`, `\bnet\.(Listen|Dial|DialTimeout|ListenPacket)\(`, `&http\.Client\{`
- `extension/src/*.js` - `\bfetch\(`, `XMLHttpRequest`, `new WebSocket\(`, `sendBeacon\(`, `new EventSource\(`
- `cmd/doc-html-ui/*.html` - `\bfetch\(`, `XMLHttpRequest`, `new WebSocket\(`, `sendBeacon\(`, `mailto:`

- not network: `extension/src/i18n.js` - fetches the extension's own bundled _locales file through chrome.runtime.getURL; no request leaves the browser
- not network: `extension/src/ebook.js` - fetches a blob: URL of a book section already unpacked in memory; no request leaves the browser

## 3. The "no telemetry" claim

**Claim:** No telemetry, analytics, crash reporting, advertising or remote code in either edition.

It is proven against the dependency set, not against the wording of a page: `go.sum` and `extension/package-lock.json` are read in full and every module or package name is matched against the denylist in the source. The claim is stated in:

- `privacy.html`
- `extension-privacy.html`
- `extension/store/PRIVACY.md`
- `extension/store/LISTING.md`

## 4. The public forms rendered from these rows

- `privacy.html`, block `app-access` (html-lists): `app-read-documents`, `app-open-browser`, `net-app-gui-server`, `net-app-google`, `net-app-ollama`, `net-app-ocr-languages`, `app-explorer-registration`, `app-run-helpers`, `net-app-report-mail`
- `privacy.html`, block `app-writes` (html-lists): `app-write-output`, `app-user-folder`
- `extension-privacy.html`, block `ext-network` (html-lists): `net-ext-document`, `net-ext-images`, `net-ext-remote-content`, `net-ext-ocr-languages`
- `extension-privacy.html`, block `ext-permissions` (html-lists): `ext-declarativenetrequest`, `ext-host-all-urls`, `ext-scripting`, `ext-offscreen`, `ext-contextmenus`, `ext-storage`
- `extension/store/PRIVACY.md`, block `ext-network` (md-list): `net-ext-document`, `net-ext-images`, `net-ext-remote-content`, `net-ext-ocr-languages`
- `extension/store/PRIVACY.md`, block `ext-permissions` (md-list): `ext-declarativenetrequest`, `ext-host-all-urls`, `ext-scripting`, `ext-offscreen`, `ext-contextmenus`, `ext-storage`
- `extension/store/LISTING.md`, block `ext-justifications` (md-justifications): `ext-declarativenetrequest`, `ext-host-all-urls`, `ext-scripting`, `ext-offscreen`, `ext-contextmenus`, `ext-storage`
- `msix/README.md`, block `msix-justification` (md-fence): `app-msix-runfulltrust`

## 5. How this is kept true

- A permission added to `extension/manifest.json` or a capability added to `msix/AppxManifest.xml` without a row fails the check, and so does a row naming something no manifest declares.
- Every row names its consumers and cites the code that consumes it; a citation the code no longer contains fails the check. A shown string a row cites must still be in the strings file it names.
- A new network call site under the roots above fails the check until a row or a not-network reason covers it.
- An analytics, crash-reporting or advertising dependency entering either dependency set fails the check while section 3 claims none.
- A public form edited by hand, or a row edited without `-Render`, fails the check. The consumers, the lifetimes and the "what leaves" column are authored: no mechanism can derive which features share a permission.
