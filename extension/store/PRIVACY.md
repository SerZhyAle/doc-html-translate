# Privacy Policy - PDF / EPUB to translatable HTML

_Last updated: 2026-06-26_

_Hosted (paste this URL into the store forms): https://serzhyale.github.io/doc-html-translate/extension-privacy.html — rendered from this text as `extension-privacy.html` in the repo root._

## Summary
This extension processes PDFs and EPUBs entirely on your device. It does not collect, store, or transmit
your documents or any personal data to us or to any third party.

## What the extension does
When you open a PDF or EPUB, the extension redirects the page to a local viewer bundled with the
extension. The viewer reads the file's bytes in your browser and re-renders its text as HTML so your
browser's built-in "Translate page" feature can work on it. All extraction and rendering happen locally,
using bundled code (PDF.js for PDF; a self-contained unzip + HTML pipeline for EPUB) - no code is
downloaded at runtime.

## Data we collect
None. We have no servers and receive no data from the extension.

## Data stored on your device
The extension uses the browser's local extension storage only to remember your settings: whether reflow
is enabled (globally and per-site), and your reading preferences (font size, font family, theme). This
data never leaves your device and is removed if you uninstall the extension.

## Network access
The extension itself makes no network requests to any server we control. It fetches the PDF or EPUB you
opened (from the site or local file you chose) in order to render it. Translation is performed by your
browser's own built-in translation feature, which you invoke yourself; that feature is governed by your
browser vendor's privacy policy, not ours.

## Permissions
- declarativeNetRequest and host access: used solely to redirect PDF opens to the local viewer and to
  fetch the opened PDF for local rendering. No browsing data is read or transmitted.
- storage: used solely to save the settings described above.

## Children's privacy
The extension is a document-reading utility and does not knowingly collect any data from anyone,
including children.

## Changes
If this policy changes, the updated version will be published at the same URL with a new "last updated"
date.

## Contact
serzhyale@gmail.com
