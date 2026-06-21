# Microsoft Store (MSIX) packaging - doc-html-translate

Packaging artifacts for shipping **doc-html-translate** to the Microsoft Store via MSIX,
adapted from the reusable CyrFlip publishing playbook. The Store path is **free** (no
developer-account fee, no code-signing certificate to buy - Microsoft re-signs the MSIX
during certification) and a Store-signed build also reduces antivirus false positives.

| File | Role |
| --- | --- |
| [`AppxManifest.xml`](AppxManifest.xml) | Package manifest **template** - `runFullTrust`, file-type associations, visual assets. `{{...}}` placeholders are filled by the build script. |
| [`build-msix.ps1`](build-msix.ps1) | go build (CLI + GUI) → version remap → generate logos → fill manifest → `makeappx pack` → optional self-sign. |
| `staging/`, `out/` | Generated (git-ignored). `out/*.msix` is what you upload. |

## What's in the package

Both binaries ship side by side, plus the manifest and generated logos:

```
doc-html-ui.exe            ← Application entry point (the launchable GUI)
doc-html-translate.exe     ← the CLI the GUI spawns (findCLI() looks next to itself)
AppxManifest.xml
Assets\StoreLogo.png  Square44x44Logo.png  Square71x71Logo.png  Square150x150Logo.png  Wide310x150Logo.png
```

The GUI is the entry point because a Store app needs a launchable window. Double-clicking an
associated document launches `doc-html-ui.exe` with the file as `argv[1]`; it pre-fills the path
and lets the user pick options before converting (see `cmd/doc-html-ui/main.go`).

## Prerequisites

```powershell
winget install Microsoft.WindowsSDK.10.0.26100         # makeappx.exe + signtool.exe
go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest
```

(Go itself is already required to build the app.)

## 1. Build & verify locally

```powershell
.\msix\build-msix.ps1 -SelfSign
```

This builds, packs, self-signs, and prints two commands. To install on this machine:

```powershell
# once, as Administrator - trust the self-signed test cert:
Import-Certificate -FilePath 'msix\out\doc-html-translate-test.cer' -CertStoreLocation Cert:\LocalMachine\Root
# install (does NOT launch - start it from the Start menu afterwards):
Add-AppxPackage 'msix\out\SerZhyAle.DocHtmlTranslate_<version>_x64.msix'
```

Smoke test under the MSIX container: launch from Start, convert a sample `.epub`/`.pdf`, confirm the
browser opens the result; double-click an associated file in Explorer and confirm the GUI opens with
it pre-filled. Uninstall with `Get-AppxPackage *DocHtmlTranslate* | Remove-AppxPackage`.

## Publisher identity (SZA account - portable across products)

These two values are **account-wide** for the SZA publisher and identical for every SZA product,
so they are already baked into `build-msix.ps1` as defaults - you do not pass them per build:

| Value | Setting | Note |
| --- | --- | --- |
| `CN=F98ACEDB-1E22-4C39-AF63-F9FCFE807DCD` | `Package/Identity/Publisher` (`-Publisher`) | tied to the account; same for all SZA products |
| `SZA` | `Package/Properties/PublisherDisplayName` (`-PublisherDisplayName`) | same for all SZA products |

Only `Package/Identity/Name` (`-IdentityName`) is **per-product** and must be reserved for this app.

## 2. Partner Center: account + identity

1. **Account settings → Programs → Windows → Get started** (NOT "Windows Desktop Applications").
   The SZA account already exists; registration was free (Individual).
2. **Create a new product → MSIX or PWA app** → reserve the app name (e.g. *doc-html-translate*).
3. **Product ▸ Product identity** → confirm the values match:
   - `Package/Identity/Name` → `-IdentityName` (**new, this product**)
   - `Package/Identity/Publisher` → must be `CN=F98ACEDB-1E22-4C39-AF63-F9FCFE807DCD` (the default)
   - `Package/Properties/PublisherDisplayName` → must be `SZA` (the default)

   They must match **exactly** or the upload is rejected.

## 3. Build the Store package (unsigned) & upload

`-Publisher` and `-PublisherDisplayName` already default to the SZA account values, so only the
reserved per-product Name is required:

```powershell
.\msix\build-msix.ps1 -IdentityName "<Package/Identity/Name from Partner Center>"
```

No `-SelfSign` - upload the **unsigned** `out\*.msix`; Microsoft signs it during certification.

### Version mapping (important)

The app's stamp is `YY.MMDD.HHmm` (e.g. `26.0612.0124`). The Store requires a **4-part**
`Major.Minor.Build.0` version with **revision = 0** and each part ≤ 65535. The script maps:

```
26.0612.0124  →  26.612.124.0       (YY . MMDD . HHmm . 0)
```

Monotonic over time and unique per minute. Override the stamp with `-Stamp 26.0612.0124` if needed.

## 4. Listing materials

| Item | Requirement | doc-html-translate |
| --- | --- | --- |
| **Privacy policy** | Required (the app reads your files & can call a translation API) | hosted page ready: paste `https://serzhyale.github.io/doc-html-translate/privacy.html` (source: [`../privacy.html`](../privacy.html)) |
| **Screenshots** | ≥ 1 PNG, **≥ 1366×768** | screenshot the GUI window converting a book |
| **Store logos** | Optional (falls back to package logos) | package logos are generated by the build script |
| **Description / features** | Required | templates below |
| **runFullTrust justification** | Required for every desktop MSIX, **~1000-char limit** | template below |
| **Age rating** | Short questionnaire → IARC rating | see IARC note below |
| **Pricing** | "Free" = base price in the Retail price dropdown | - |

### Age rating (IARC)

The SZA account has a portable **Global Rating ID `7d9b315a-f211-8505-80d0-3f4bee633770`**
(first generated for FastMediaSorter). You may paste it **only if** this app's functionality
does not change any questionnaire answer; otherwise run the short questionnaire again and let
IARC generate a fresh rating. doc-html-translate is a different app (document conversion, no IAP,
no ads, no UGC sharing), so verify the answers still hold before reusing the ID - when in doubt,
re-run the questionnaire.

## 5. Submit → certification

A full-trust app that spawns child processes and serves a local port can draw extra review. The
runFullTrust justification + a clear description pre-empt most questions. Certification typically
takes a few business days.

---

## Known limitations / follow-ups for the Store build

- **Google translation key (Store-safe).** The MSIX install directory is **read-only**, so a
  `google_api.key` cannot sit next to the exe there. The CLI therefore also looks for the key at
  `%LOCALAPPDATA%\doc-html-translate\google_api.key` (a writable per-user path) - that is where Store
  users place it. The next-to-exe location still works for the unpackaged build. The free
  browser-translation flow and the local **Ollama** flow need no key at all.
- **`-register` is a no-op under MSIX** by design - `HKCU\Software\Classes` writes are virtualized
  and ignored. File associations come from the manifest instead (already declared).
- **MOBI/AZW3 still need Calibre** installed on the user's machine (runtime dependency, non-DRM only).
- **Output location:** writing the HTML next to the source file or to a user-chosen folder works
  under `runFullTrust`. Only `%LOCALAPPDATA%`/`HKCU` writes are redirected into the container - the
  app doesn't depend on those for externally-read output.

---

## Text templates (Store listing)

### Description
```
doc-html-translate converts documents - EPUB, PDF, MOBI, AZW3, FB2, RTF, TXT, Markdown and HTML - into clean, local HTML you can read in any browser, with generated navigation and a table of contents.

Open a file and the app extracts it to a self-contained HTML folder and opens it in your browser. Read it offline, or use your browser's built-in page translation to read in your own language - completely free. For automated translation it can optionally use the Google Cloud Translation API or a local Ollama model (with Ollama your text never leaves your machine).

It runs as a small desktop app: pick a file, choose options, convert. Re-opening an already-converted book is instant. It needs nothing extra on Windows 10/11 (MOBI/AZW3 additionally require Calibre, for non-DRM files only).

doc-html-translate runs entirely on your device, has no accounts and no telemetry, and is open source: https://github.com/SerZhyAle/doc-html-translate
```

### Product features (one per line, ≤ 200 chars each)
```
Convert EPUB, PDF, MOBI, AZW3, FB2, RTF, TXT, Markdown and HTML to clean local HTML
Generated navigation and table of contents for comfortable reading
Read offline in any browser, or translate the page free with the browser's built-in translator
Optional automated translation via Google Cloud or a local Ollama model
Re-opens an already-converted book instantly
Opens documents straight from Explorer via file associations
Open source - no accounts, no telemetry, no ads, no data collection
```

### runFullTrust justification (keep under ~1000 chars)
```
doc-html-translate is a full-trust Win32 desktop app (Go), not a UWP app, so runFullTrust is required to run as a normal desktop process and to use the Win32 capabilities its features depend on:
- Reading the documents the user opens and writing the converted HTML next to them or to a folder the user chooses.
- Launching the bundled command-line converter and the user's web browser to display the result.
- Calling Calibre (for MOBI/AZW3) and a local Ollama server when those optional features are used.
These capabilities are available only to full-trust desktop apps. The app runs locally, makes no network connections except optional user-initiated translation (Google Cloud Translation API or a local Ollama model), and collects no user data. Open source: https://github.com/SerZhyAle/doc-html-translate
```

### Privacy policy (host as a page; paste the URL into Partner Center)
```
doc-html-translate does not collect, store, log, or transmit any personal data. It runs entirely on your device, has no servers of its own, and contains no telemetry, analytics, ads, or accounts.

What it accesses and why:
- The document files you open - to convert them to local HTML.
- Your web browser - to display the converted HTML.
- The internet - only if you explicitly choose Google translation: the extracted text is sent to the Google Cloud Translation API using a key you supply. With the local Ollama option, text is sent only to a translation server running on your own machine.

Local files it writes: the converted HTML output (next to the source file, or in a folder you choose). These never leave your device.

Data sharing: none. Children: no data collected. Open source: https://github.com/SerZhyAle/doc-html-translate. Contact: sza@ukr.net
```

---

_Adapted from the CyrFlip MSIX playbook (`P:\WINDOWS\CyrFlip\STORE_PUBLISHING.md`)._
