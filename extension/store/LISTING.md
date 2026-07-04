# Store listing assets

Copy and metadata for the Chrome Web Store + Edge Add-ons listings. Everything here is text; no account
action is automated.

**Live listing (Chrome Web Store):** https://chromewebstore.google.com/detail/nmcckamdocainafmmompkbmelkpbnmic
The Edge Add-ons submission is still pending - reuse the same copy below (see the Edge notes at the end).
Keep this file in sync with the published listing when the description changes.

> **Owner sign-off required (ticket 2026-07-01_extension-format-parity):** the extension now opens nine
> formats (PDF, EPUB, MOBI, AZW3, FB2, RTF, TXT, Markdown, local HTML), not just PDF/EPUB. The name,
> short and detailed copy below is a draft reflecting the wider range - review and finalize the marketing
> wording (especially the store **name**, which changes the listing identity) before the next publication.

## Localization (RU / UK)
The extension is localized via `_locales/{en,ru,uk}/messages.json` (`default_locale: en`). Chrome and Edge
auto-serve the localized **name** and **short description** from those files based on the user's browser
language - no per-language dashboard step is needed for those two fields. The long **detailed description**
and the **screenshot captions** are NOT auto-localized by the store, so paste them by hand from the same
message keys (`storeDescription`, `shot1Caption`..`shot3Caption`) into each store's localized-listing editor:
- Chrome Web Store: Store listing -> language dropdown (top right) -> add Russian and Ukrainian, paste the
  RU/UK `storeDescription` and the localized screenshots/captions.
- Edge Add-ons: Properties -> Availability -> add the same two languages, paste the same copy.
The RU/UK strings below are the canonical source; keep them in sync with `_locales`.

## Name
Document Page Translator - PDF, EPUB & more (Free & Local)
- RU: Переводчик страниц: PDF, EPUB и другие (бесплатно, локально)
- UK: Перекладач сторінок: PDF, EPUB та інші (безкоштовно, локально)

## Short description (<= 132 chars)
> Chrome rejected the earlier all-formats keyword list as "excessive keywords" (Spam / Placement,
> Jul 2026). Keep only the two headline formats (PDF, EPUB) + "other documents" - do NOT re-enumerate
> all nine formats here. The full list still lives in the detailed description below, in prose.

Re-render PDF, EPUB & other documents as clean HTML the browser's free Translate page can read - incl. image OCR. 100% local.
- RU: Открывайте PDF, EPUB и другие документы как чистый HTML - и браузер бесплатно переводит их встроенным переводчиком. Локально.
- UK: Відкривайте PDF, EPUB та інші документи як чистий HTML - і браузер безкоштовно перекладає їх вбудованим перекладачем. Локально.

## Category
Productivity

## Single-purpose statement
This extension has one purpose: when you open a supported document (PDF, EPUB, and other e-book and
document formats), it re-renders the document's text as clean, reflowed HTML in a local viewer so the
browser's built-in "Translate page" feature can translate it. It does not do anything else.

## Detailed description
> Chrome rejected this again on Jul 4 2026 (same "Yellow Argon" ref) - this time for the **detailed**
> description, not the short one. The offender was the full nine-format list repeated in a parenthetical
> `(PDF, EPUB, MOBI, AZW3, FB2, RTF, TXT, Markdown, or local HTML)` in both the single-purpose statement
> and "How it works". After a second rejection we stopped fighting it: the store copy now names **only
> PDF and EPUB** (the core formats) and refers to everything else generically as "other e-book and
> document formats" - **no MOBI/AZW3/FB2/RTF/TXT/Markdown/HTML enumeration anywhere in the listing**. The
> full explicit list lives only in README.md and PRIVACY.md, never in the store Description. This is a
> Store-listing **Description** field edit (dashboard only); the packaged manifest short description is
> already compliant, so no new upload or version bump is required to resubmit.

Chrome and Edge have excellent built-in translators that translate any webpage instantly and for free. But they don't work on books and documents:
1. **PDFs** are rendered on a canvas, making the text invisible to the browser's translator.
2. **EPUBs** cannot be opened in the browser at all; they just trigger a download.

**This extension quietly solves both problems - and opens seven more document formats too!**

### How it works:
When you open a supported document, this extension converts its text into a clean, reflowed HTML webpage. Since it is now a normal webpage, you can simply right-click and select **"Translate to [Your Language]"**!

### Key Features:
- 📖 **EPUB Reader**: Unpacks EPUB books and combines chapters into a single scrollable document for seamless translation.
- 📄 **PDF Reflow**: Extracts text and lays it out as readable paragraphs, making sure the translator doesn't break, overlap, or double text.
- 📚 **More formats**: Also opens many other e-book and document formats - each converted to clean, translatable text, with no Calibre or any other app required.
- 🔀 **Table of Contents**: Automatically imports the document's original outline as a collapsible navigation tree.
- 🎨 **Reading Mode**: Customize your reading experience with light, sepia, dark, and night themes, and adjustable font sizes.
- 🖼️ **Image OCR - translate text in pictures**: Right-click any image ("OCR & translate this image"), open a local image file with the "Open file" button, or turn on "Use OCR for images" for PDFs and EPUBs - the extension recognizes the text baked into pictures and lays it over them as real text, so the browser's "Translate page" translates that too. English works offline; more languages download on demand.
- 🔒 **100% Local & Private**: Everything runs entirely on your device. Your documents are never uploaded to any server.
- 🔄 **One-Click Original**: Instantly toggle back to the browser's native viewer with a single click.

### How to use:
1. Install the extension.
2. Open any supported document in your browser, or click the extension icon to select a local file.
3. Right-click anywhere on the page and select **"Translate"** to translate the entire document into your language for free!

Privacy: Everything runs on your device. The extension never uploads your documents anywhere — it has no server to send them to and no interest in your reading list. The only network access is your browser's own translation feature, which you trigger yourself.

Limitations: Scanned/image-only PDFs have no text to translate (the extension detects this and offers the original); files served without a ".pdf"/".epub" address aren't auto-detected; EPUBs are shown with the viewer's own clean reading style rather than the book's original design; DRM-protected EPUBs can't be read; ligatures in some PDF fonts may render imperfectly (a PDF.js limitation).

## Permission justifications (for the review form)
- declarativeNetRequest: used with dynamic rules created at runtime to redirect main_frame document loads
  (`*.pdf`, `*.epub`, `*.mobi`, `*.azw3`, `*.fb2`, `*.rtf`) to the bundled local viewer. The extension does not read or modify the content of other
  sites; the rules are plain URL redirects.
- host_permissions (`<all_urls>`): required because the viewer fetches the opened document's bytes for
  local rendering and the file can live on any origin, and because the toolbar popup reads the active
  tab's hostname for the per-site on/off toggle. (The redirect itself does not consume host permissions.)
  No page content is read or transmitted. Because the opened document's origin is unknowable before the
  user opens the file, no fixed narrower match-pattern set is possible, and `activeTab` cannot grant the
  cross-origin fetch because the viewer is an extension page rather than the document's origin - so
  `<all_urls>` is the minimum that works for this single purpose.
- contextMenus: to add the right-click "OCR & translate this image" action on images.
- storage: to remember your on/off choice, per-site exceptions, reading preferences (font, theme), and
  which OCR languages you have downloaded.

## Data use disclosures
- Does the extension collect or transmit user data? No.
- Remote code? No. All code (including PDF.js) is bundled in the package; nothing is loaded from a
  remote server and there is no eval of remote code.

## Screenshots (capture on Windows - manual)
Shoot all three in **Chrome on Windows 11** so the Windows window chrome (title bar + Chrome's translate
banner) is visible - that's the "у кого Windows" framing the listing should show. Export **1280x800 PNG**
(or 640x400). Show real application screens, not text-only marketing banners. Capture an EN and a RU/UK pass
so each localized listing gets matching shots. The captions below come verbatim from the `_locales`
`shot*Caption` keys - overlay them (or paste as the screenshot description) per language.

- **Screenshot 1 - Core value (PDF before -> after):** an English PDF reflowed into clean text with Chrome's
  "Translate this page" banner visible at the top of the Windows window. For the RU/UK pass, show it
  translated to that language.
  - EN: Open a PDF - the extension reflows it into clean text Chrome can translate in one click.
  - RU: Откройте PDF - расширение перевёрстывает его в чистый текст, который Chrome переводит в один клик.
  - UK: Відкрийте PDF - розширення переверстує його в чистий текст, який Chrome перекладає в один клік.
- **Screenshot 2 - EPUB reader + collapsible TOC:** an EPUB opened in dark or sepia theme, the collapsible
  table-of-contents sidebar on the left, a formatted translated chapter on the right.
  - EN: EPUB books open with a collapsible table of contents and comfortable reading themes.
  - RU: Книги EPUB открываются со сворачиваемым оглавлением и удобными темами чтения.
  - UK: Книжки EPUB відкриваються зі згортуваним змістом і зручними темами читання.
- **Screenshot 3 - Right-click flow + toolbar popup:** the right-click menu "Translate to [language]" over the
  reflowed document, with the extension's toolbar popup (toggles) in the corner.
  - EN: Right-click, "Translate to ...", done - 100% local, nothing leaves your device.
  - RU: Правый клик, «Перевести на ...», готово - 100% локально, ничего не покидает ваше устройство.
  - UK: Правий клік, «Перекласти на ...», готово - 100% локально, нічого не залишає ваш пристрій.
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
