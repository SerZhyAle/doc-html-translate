# Privacy Policy - Documents to translatable HTML

_Last updated: 2026-09-25_

_Hosted (paste this URL into the store forms): https://serzhyale.github.io/doc-html-translate/extension-privacy.html - the same policy as `extension-privacy.html` in the repo root. The *Network access* and *Permissions* sections of both are rendered from the rows in `docs/security-posture.json`; edit the rows, not the sections._

## Summary
This extension processes documents - PDF, EPUB, MOBI, AZW3, FB2, RTF, TXT, Markdown, and local HTML -
entirely on your device. It does not collect, store, or transmit your documents or any personal data to
us or to any third party.

## What the extension does
When you open a supported document, the extension opens it in a local viewer bundled with the extension.
The viewer reads the file's bytes in your browser and re-renders its text as HTML so your browser's
built-in "Translate page" feature can work on it. All extraction and rendering happen locally, using
bundled code (PDF.js for PDF; a self-contained unzip + HTML pipeline for EPUB; `marked` for Markdown;
`foliate-js` for MOBI/AZW3; and small readers for FB2/RTF/TXT/HTML) - no code is downloaded at runtime.

## Image text recognition (OCR)
The extension can recognize text baked into images - both images inside your PDFs/EPUBs and any image
you right-click with "OCR & translate this image" - so your browser's "Translate page" can translate
it. Recognition runs entirely on your device using a bundled engine (Tesseract). English recognition
data ships inside the extension and needs no network. Additional recognition languages are optional:
they are downloaded - as data files, only when you explicitly click "Download" - from a public
open-source data host (tessdata_fast on `tessdata.projectnaptha.com`) and cached locally on your
device for reuse. We operate no server; your images and their text are never sent to us or to any
third party.

## Data we collect
None. We have no servers and receive no data from the extension.

## Data stored on your device
The extension uses the browser's local extension storage only to remember your settings - whether the
viewer is on (globally and per site), your reading preferences (font size, font family, theme), the
interface language and whether remote images may load - which OCR languages you have downloaded, and the
short summary of the most recent document that "Copy diagnostics" reads. The recognition data of a
downloaded language is cached in the browser's storage for reuse. None of it leaves your device, and it
is removed if you uninstall the extension.

## Diagnostics you copy yourself
The options page carries a "Copy diagnostics" button. Pressing it writes a short English summary to your
clipboard: the extension version, your browser and platform, the interface language, your current
settings, and the format, page count and last error of the most recent document. It records no document
text, no file name and no URL, and your per-site exceptions are reported as a count rather than as host
names. The button adds no permission and sends nothing anywhere - it writes to the clipboard only, and
you decide whether to paste it into a mail to the author.

## Network access
<!-- security-posture:begin ext-network (rendered from docs/security-posture.json by scripts/security-posture.ps1 -Render; edit the rows there) -->
- **The document you open** - fetched from the site or local file you chose, with your sign-in for that
  site as an ordinary tab would send it, so that the viewer can render it.
- **The pictures you ask it to read** - the image you right-click, or the pictures of the page where you
  start *OCR every image on this page*, fetched from the sites that serve them.
- **Images a document points at on the internet** - blocked until you choose *Load them* for that
  document or turn on *Load remote images in documents*, so opening a document does not tell its author
  or a tracker that you opened it.
- **tessdata.projectnaptha.com, for extra OCR languages** - only when you click *Download* for a
  language: its data file is fetched, as data and not code, from that public open-source host and cached
  on your device. English ships inside the extension.

Nothing else leaves your browser: the extension has no server of its own. Translation is done by your
browser's built-in feature, which you start yourself and which your browser vendor's privacy policy
governs.
<!-- security-posture:end ext-network -->

## Permissions
<!-- security-posture:begin ext-permissions (rendered from docs/security-posture.json by scripts/security-posture.ps1 -Render; edit the rows there) -->
- **declarativeNetRequest** - used solely to open a document you open in the browser in the extension's
  local viewer instead, and to let *Open original* show it the browser's own way. The rules redirect
  addresses only; no page content is read.
- **Access to all sites** (host access) - used solely to fetch the document you opened and the pictures
  you ask it to read, from whichever site or folder they come from, and to read the current site's name
  for switching the viewer on or off per site. No browsing data is read or transmitted.
- **scripting** - used only when you choose *OCR every image on this page* from the right-click menu: the
  extension then adds one script and its stylesheet to that one tab, lays the recognized text over the
  page's pictures, and removes both when you stop. Nothing is added to a page without that click.
- **offscreen** - the recognition engine for a page runs in a hidden page of the extension, opened for
  one run and closed when it ends; only the picture areas and the recognized text pass through it, and
  none of it leaves your device.
- **contextMenus** - adds the extension's right-click actions: *OCR & translate this image* on a picture,
  *OCR every image on this page* on a page, and *Convert with doc-html-translate* on a document link or
  page.
- **storage** - used solely to remember, on this device, your settings (the viewer on or off, globally
  and per site, reading preferences, the interface language, whether remote images may load), which OCR
  languages you have downloaded, and the format, page count and last error of the most recent document
  for *Copy diagnostics*.
<!-- security-posture:end ext-permissions -->

## Children's privacy
The extension is a document-reading utility and does not knowingly collect any data from anyone,
including children.

## Changes
If this policy changes, the updated version will be published at the same URL with a new "last updated"
date.

## Contact
sza@ukr.net
