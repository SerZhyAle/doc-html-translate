# Positioning

The one document every external surface of doc-html-translate is written from: the landing and its ten
language pages, the README trio, the documentation trio, the extension page, the store listings and the
extension listings. No surface is written from another surface; where two disagree this file decides
which one is wrong. Contract: `SITE-REPRESENTATION` rules 1 to 4 (pointer in
[contracts/SITE-REPRESENTATION.md](contracts/SITE-REPRESENTATION.md)).

The facts that are typed once - the public editions and their display names, the channels, the number of
interface languages, the device classes - are in [`positioning.json`](positioning.json). This file holds
what the product is for; that file holds what a program reads. `tests/site_positioning_test.go` reads both.

## What the product is

Turn any book, document or comic into a local web page, then read it, search it and translate it with the
browser's own page translation - free, with no key and no account.

## The pillars

In this order. A surface that lists what the product does lists these in this order; a surface too short
for all of them names the first ones.

<!-- pillars:begin -->
| # | Id | Pillar | What the visitor gets |
|---|---|---|---|
| 1 | `READ` | Open a book or document | A local page with a table of contents, navigation, themes and a remembered reading position. |
| 2 | `TRANSLATE` | Translate in the browser | The HTML opens in Chrome or Edge and the browser's own page translation works on it. |
| 3 | `OCR` | Recognise text in an image | Translatable text layers on a scan, a picture or a comic page. |
| 4 | `LOCAL` | Keep the result locally | An ordinary folder of HTML and resources that can be moved and reopened. |
<!-- pillars:end -->

`Id` is the token the surfaces carry: the landing marks each use-case card with it, on the root page and on
the ten language pages alike, and the test reads the order off those marks.

## Store-policy exceptions

None. No listing is required by its store to name one function ahead of the others.

## Where each surface stands

| Surface | Written from this file | Judged by |
|---|---|---|
| `index.html` and the ten `<code>/index.html` | the use-case cards carry the pillars in order | `tests/site_positioning_test.go` reads the `pill` marks of `#use-cases` |
| Editions, channels and language count on every page that states one | `positioning.json` | the same test, per page |
| README trio, documentation trio, extension page | the lead names pillars 1 and 2 first (convert and read, then translate in the browser) | read by hand at each release; no gate |
| Store listings (`tools/store/listing/*.txt`), extension listing (`extension/store/LISTING.md`) | the same | read by hand at each release; no gate |

A change to the pillar list or its order is made here first, then in every surface of the table in one edit
(`DEV/DOCS_SURFACES.md`).

## Public editions and the device classes

Declared with their display names in [`positioning.json`](positioning.json). A build variant that is not
listed there is not public and no page names it.

## The worksheet

The half of the canon worksheet (`PROMOTION.md` section 2) that the pillars do not cover. Written once by
ticket 111; every discoverability field below is derived from it. Public copy never names a competing
product (`PROMOTION.md` section 1 rule 4): the alternatives are generic, and what is known about named
products lives in the ticket's own research note.

### What the reader does today without this product

<!-- alternatives:begin -->
- an ebook manager's or reader's built-in lookup, which translates a selected phrase, not the book, and does
  not read a comic page;
- doing the round trip by hand: unpacking the book, translating its HTML files and rebuilding it;
- a translation add-on for an ebook manager, which needs a translation engine and often a paid key;
- a whole-book machine translator that rewrites the book through the user's own key or model;
- an online translation site or file converter, which needs the file uploaded, and often credits;
- the document mode of an online translator, which takes a few formats and a size-limited file and skips text in
  images;
- the browser's PDF viewer, where whole-page translation depends on the browser and its version;
- a phone camera's translation of a comic page, one picture at a time;
- not reading the book.

Opening a converted book in the browser's own page translation is not unique to this product, and no public
copy calls it that. What the product adds is the combination: the Windows app and the browser extension, OCR
text layers over scans and comic pages, comic archives and thirteen interface languages (the attributes
below).
<!-- alternatives:end -->

### Unique attributes, each with its pointer

An attribute that cannot be pinned to a test, a flag or a file is cut.

| Attribute | Pointer |
|---|---|
| The output is ordinary HTML that the browser's own page translation works on; the interface language never replaces the document's language, so the browser still offers to translate | `tests/smoke_test.go` `TestConvertedChromeLanguage`; `extension/src/viewer.js` |
| The file is not uploaded: conversion and reading run on the PC or in the browser | `docs/security-posture.json` (what each edition sends and to whom) |
| Translatable text layers over scans, pictures and comic pages (OCR) | `internal/ocr`; `docs/contracts/OCR-PIPELINE.md`; `extension/src/ocr.js` |
| Comic archives: CBZ, CBR, CB7, CBT on the desktop; CBZ and CBT in the extension | `internal/comic`; `extension/src/comic.js` |
| A reading layer: themes, text size, remembered position | `internal/htmlgen/navbar.go` `readerScript` |
| Thirteen interface languages | `docs/positioning.json` `interfaceLanguages`; `internal/i18n` |
| Three editions: desktop app, Microsoft Store app, browser extension | `docs/positioning.json` `editions` |
| Open source, MIT licence | `LICENSE` |

### Value

When I have a book or a comic in a language I do not read, I want to open it as a page my browser can
translate, so I can read it without uploading it anywhere or paying for a translator.

### Persona

A non-technical reader on Windows, or in desktop Chrome or Edge, with a foreign-language book, document or
comic file.

### Category

Desktop: "ebook converter", "EPUB to HTML converter". Extension: "PDF and EPUB translator", "document
translator". Both are terms people already search for (the per-locale terms are in the ticket's research
note and, once chosen, in [`discoverability.json`](discoverability.json)).

### Where the free path stops

"Free, with no key and no account" is true of the browser's page translation (pillar 2) only. Google Cloud
Translation (the user's own key, billed to the user) and a local Ollama model are optional engines and never
the main path. "Conversion and reading are local" is true; "nothing leaves your PC" is not (the user-started
log report and the optional Google Cloud path leave the machine), and "offline translation" is not (the
browser's page translation is online). Public copy uses the true lines.

## Discoverability fields

The values are in [`discoverability.json`](discoverability.json); this table says where each one is used and
what binds it. `tests/discoverability_test.go` holds the lists the build owns against that file.

| Field | Used in | Limit | Rule |
|---|---|---|---|
| `github.description`, `github.homepage`, `github.topics` | the GitHub repository (`gh repo edit`, an owner-approved step) | description 350 characters, 20 topics | category noun first, then what it does, then platform; topics from `allowedTags` |
| `winget.tags`, `winget.moniker` | the 13 `winget/*.locale.*.yaml` | 16 tags | identical in every locale; no dropped term |
| Store search terms | `@@SearchTerm1..7` of `tools/store/listing/*.txt` | 7 terms, 40 characters each, 21 words in all | researched per locale, never translated literally |
| Extension short description | `appDesc` of `extension/_locales/*/messages.json` | 132 characters | no format enumeration (Chrome rejected one in July 2026 as excessive keywords) |
| Dropped terms and phrases | every list above | - | `google-translate` and the other entries of `droppedTerms`, the over-claims of `droppedPhrases`, the comparison wording of `forbiddenPhrases` |
