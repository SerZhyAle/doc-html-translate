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
