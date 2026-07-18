# Privacy Policy - Documents to translatable HTML

_Last updated: 2026-07-01_

_Hosted (paste this URL into the store forms): https://serzhyale.github.io/doc-html-translate/extension-privacy.html - rendered from this text as `extension-privacy.html` in the repo root._

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
The extension uses the browser's local extension storage only to remember your settings: whether reflow
is enabled (globally and per-site), and your reading preferences (font size, font family, theme). This
data never leaves your device and is removed if you uninstall the extension.

## Network access
The extension itself makes no network requests to any server we control. It fetches the document you
opened (from the site or local file you chose) in order to render it, and it fetches an image you asked
it to OCR. The only outbound request beyond that is optional: when you explicitly download an extra OCR
language, its data file is fetched from the public open-source host named above and cached locally.
Translation is performed by your browser's own built-in translation feature, which you invoke yourself;
that feature is governed by your browser vendor's privacy policy, not ours.

## Permissions
- declarativeNetRequest and host access: used solely to redirect PDF opens to the local viewer, to
  fetch the opened PDF for local rendering, and to fetch an image you choose to OCR. No browsing data
  is read or transmitted.
- contextMenus: used solely to add the right-click "OCR & translate this image" action on images.
- storage: used solely to save the settings described above (including which OCR languages you have
  downloaded).

## Children's privacy
The extension is a document-reading utility and does not knowingly collect any data from anyone,
including children.

## Changes
If this policy changes, the updated version will be published at the same URL with a new "last updated"
date.

## Contact
serzhyale@gmail.com
