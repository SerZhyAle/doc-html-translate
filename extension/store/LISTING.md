# Store listing assets (Steps 6-7)

Draft copy and metadata for the Chrome Web Store + Edge Add-ons listings. Fill in screenshots and the
hosted privacy-policy URL before submitting. Everything here is text; no account action is automated.

## Name
doc-html-translate

## Short description (<= 132 chars)
Opens PDFs and EPUBs as clean reflowed HTML so your browser's built-in Translate page works on them. 100% local - no API key.

## Category
Productivity

## Single-purpose statement
This extension has one purpose: when you open a PDF or EPUB, it re-renders the document's text as clean,
reflowed HTML in a local viewer so the browser's built-in "Translate page" feature can translate it. It
does not do anything else.

## Detailed description
Browsers can translate any web page for free, but not PDFs or EPUBs - the PDF viewer draws text on a
canvas that translation can't touch, and EPUBs don't open in the browser at all. This extension fixes
both. When you open a PDF, it extracts the text with PDF.js (bundled, offline) and lays it out as
ordinary HTML paragraphs and headings. When you open an EPUB, it unpacks the book and combines its
chapters into one clean HTML document. Either way you get a table of contents from the document's own
outline, and your browser's own "Translate page" works exactly as it does on any website.

- Works on web (https) and local (file://) PDFs and EPUBs.
- Detects the document language and sets it so the browser offers the right translation.
- Reading controls: font size/family, light/sepia/dark themes, page jump, collapsible contents.
- One click back to the original PDF in the native viewer.
- Turn it off globally or per-site from the toolbar.

Privacy: everything runs on your device. The extension never uploads your documents anywhere. The only
network access is your browser's own translation feature, which you trigger yourself.

Limitations: scanned/image-only PDFs have no text to translate (the extension detects this and offers
the original); files served without a ".pdf"/".epub" address aren't auto-detected; EPUBs are shown with
the viewer's own clean reading style rather than the book's original design; DRM-protected EPUBs can't be
read; ligatures in some PDF fonts may render imperfectly (a PDF.js limitation).

## Permission justifications (for the review form)
- declarativeNetRequest: used with dynamic rules created at runtime to redirect main_frame `*.pdf` and
  `*.epub` loads to the bundled local viewer. The extension does not read or modify the content of other
  sites; the rules are plain URL redirects.
- host_permissions (`<all_urls>`): required because the viewer fetches the opened document's bytes for
  local rendering and the file can live on any origin, and because the toolbar popup reads the active
  tab's hostname for the per-site on/off toggle. (The redirect itself does not consume host permissions.)
  No page content is read or transmitted. Because the opened document's origin is unknowable before the
  user opens the file, no fixed narrower match-pattern set is possible, and `activeTab` cannot grant the
  cross-origin fetch because the viewer is an extension page rather than the document's origin - so
  `<all_urls>` is the minimum that works for this single purpose.
- storage: to remember your on/off choice, per-site exceptions, and reading preferences (font, theme).

## Data use disclosures
- Does the extension collect or transmit user data? No.
- Remote code? No. All code (including PDF.js) is bundled in the package; nothing is loaded from a
  remote server and there is no eval of remote code.

## Assets still to produce (manual)
- Screenshots (1280x800 PNG, at least 1, ideally 3-5): a reflowed PDF mid-translation; a reflowed EPUB
  with the contents sidebar; the toolbar popup. Use the same set for Chrome and Edge. Avoid text-only
  marketing graphics (Chrome rejects non-representative images).
- Small promo tile 440x280 (optional), marquee 1400x560 (optional).
- Hosted privacy-policy URL: **already prepared** at `extension-privacy.html` in the repo root - once the
  GitHub Pages site is deployed it is served at
  `https://serzhyale.github.io/doc-html-translate/extension-privacy.html`. Paste that exact URL (a
  rendered HTML page, never the raw `.md` or a GitHub blob) into the Privacy policy field of both stores.
  (The existing `privacy.html` is the desktop app's policy and does NOT describe the extension - use the
  extension page.)
- Optional: a stable extension ID via a "key" field / uploaded .pem for a self-hosted CRX.

## Edge Add-ons notes (second submission)
Edge Partner Center does NOT inherit the Chrome answers - fill its forms independently with the same
content:
- Privacy policy URL: the same hosted `extension-privacy.html` URL.
- Data-collection questionnaire: "Does this extension collect personal data?" -> **No** (everything is
  processed locally; the document bytes are never sent to the developer).
- Description: paste the same detailed description from this file.
- Edge accepts the same store ZIP unchanged (`dist/doc-html-translate-extension.zip`).
