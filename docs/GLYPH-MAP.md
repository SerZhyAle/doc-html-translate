# Glyph map

This product's inventory against the shared icon vocabulary (`ICON-SET` 0.15, `ICON-RENDER` 0.13,
`ICON-EXTERNAL` 0.10 - pointers in [`contracts/`](contracts/)). It is the product's rung-2 record
(`ICON-SET` section 6: every glyph it shows maps to an id, or is declared artwork). Mapped on
2026-09-25 by ticket 24 (`DEV/plan/done/24_2026-09-23_contract-iconography-sync.md`).

Every glyph that conforms is drawn from the catalog's own file, vendored byte for byte under
[`../assets/glyphs/`](../assets/glyphs/) with its SHA-256 in `PROVENANCE.txt`. The two editions keep
one table each - `internal/htmlgen/glyphs.go` (desktop reader chrome) and `extension/src/glyphs.js`
(extension) - and the GUI and the site carry inline copies. `tests/iconography_test.go` holds all of
them to the vendored files, the vendored files to their provenance (and to the catalog when
`SZA_CONTRACTS_ROOT` names it), both editions' names for a shared control to each other, and this
file to the vendored set. `internal/htmlgen/glyphs_test.go` pins the paging pair and every
glyph-only reader control, glyph and name, in all thirteen languages.

Status: **conforms** - the vocabulary glyph under its canonical name (qualified where rule 3 allows);
**exception** - kept as it is under a dated exception in the catalog registry, with what would end
it; **artwork** - product-private, outside the vocabulary (`ICON-SET` rule 7). The catalog proposal
is `iconography/PROPOSAL-2026-09-25-doc-html-translate.md`; its items were decided in `ICON-SET` 0.15
and `ICON-RENDER` 0.13 on 2026-09-25, except the co-signed and open ones named below.

## Converted book (reader chrome, `internal/htmlgen`)

| Control | Glyph before | Id | Name (en / ru / uk) | Status |
| --- | --- | --- | --- | --- |
| Previous page link | `◀` + "Back" | `media.previous` | Previous page / Предыдущая страница / Попередня сторінка | conforms |
| Next page link | `▶` + "Forward" | `media.next` | Next page / Следующая страница / Наступна сторінка | conforms |
| Table of contents link | `☰` | `nav.contents` | Table of contents / Оглавление / Зміст | conforms - "Оглавление" is the record's declared book-reader form (0.15) |
| Continue reading (index) | `▸` | `feature.continue-reading` | Continue reading / Продолжить чтение / Продовжити читання | conforms |
| Text smaller | `A−` | `action.text-smaller` | Smaller text / Мельче / Дрібніше | conforms |
| Text larger | `A+` | `action.text-larger` | Larger text / Крупнее / Більше | conforms |
| Recognized text layer toggle | `▤` | `view.text-layer` | Text layer / Текстовый слой / Текстовий шар | conforms in glyph and name; its state is an exception (`ICON-RENDER` rule 4: the record's off form is owed, so `aria-pressed` and a pressed look carry it) |
| Theme select | `☀ ◑ ☾ ●` on the options | `app.theme` | Theme / Тема / Тема; options Light / Sepia / Dark / Night as words | conforms (the record's choices are words) |
| Page jump select | - (text) | `nav.go-to-page` | Go to page / Перейти к странице / Перейти до сторінки | conforms |
| Index TOC disclosure | browser's `<details>` marker | `nav.expand` / `nav.collapse` | branch title | conforms (CSS mask, `rtl: fixed`, symmetric) |
| Index TOC link colour | `#1a0dab` (1.4:1 on dark, 1.59:1 on night) | - | - | conforms: `--dht-link`, at least 6.3:1 on all four themes (`ICON-RENDER` rule 3) |

## Extension (viewer, popup, options)

| Control | Glyph before | Id | Name (en) | Status |
| --- | --- | --- | --- | --- |
| Table of contents button | `☰` + untranslated "Contents" | `nav.contents` | Table of contents (`ttToc`, the desktop's words) | conforms |
| TOC branch chevron | `▾` / `▸` | `nav.collapse` / `nav.expand` | Collapse / Expand | conforms |
| Save the original file | `↓` in 13 message strings | `action.save` | Save file | conforms |
| Export the view as HTML | `↓` in 13 message strings | `action.export` | Export HTML | conforms |
| Popup external links | `↗` | `nav.open-external` | (link text) | conforms, mirrors in RTL |
| Text smaller / larger | `A−` / `A+`, English aria-label | `action.text-smaller` / `action.text-larger` | `ariaSmallerText` / `ariaLargerText`, the desktop's words | conforms |
| Recognized text layer toggle | `▤`, English aria-label | `view.text-layer` | Text layer (`ttOcrLayer`) | conforms; state as in the book (exception) |
| Theme select | - (text) | `app.theme` | Theme (`ariaTheme`) | conforms |
| Page jump | - (text) | `nav.go-to-page` | Go to page (`ttGoToPage`) | conforms |

## GUI launcher (`cmd/doc-html-ui/ui.html`)

| Control | Glyph before | Id | Name (en) | Status |
| --- | --- | --- | --- | --- |
| Section disclosure | `▸` / `▾` (did not mirror in RTL) | `nav.expand` / `nav.collapse` | section title | conforms (`rtl: fixed`, symmetric) |
| Drop zone | `⤓` (`nav.scroll-bottom`'s shape) | `content.document` | Drag a document here, or click to choose one | conforms |
| Swap source and target | `⇄` | `action.swap` | Swap source and target | conforms |

## Site (`index.html`, the ten language pages, `extension.html`)

| Control | Glyph before | Id | Name (en / ru / uk) | Status |
| --- | --- | --- | --- | --- |
| To top button | `↑` + "Back to top" | `nav.scroll-top` | Scroll to top / В начало страницы / На початок сторінки | conforms (the record's qualified page form, 0.15) |
| Cross-link to the desktop app | `←` | `nav.go-to` | Desktop app | conforms |
| Section disclosure (`extension.html`) | `▸` / `▾` | `nav.expand` / `nav.collapse` | section title | conforms |
| Store install buttons | `⤓` | `action.install` | Install from Chrome Web Store / Edge Add-ons | conforms |
| Copy confirmation | `✓ Copied` (`✓` alone on the ten language pages) | - | Copied / Скопировано / Скопійовано | conforms: `action.copy`'s done state is the word; the copy button is text-only |
| Theme switch | `◐` | - | Switch theme | exception: the kit's glyph (`PAGE-STYLE` section 9), open in the co-signed kit proposals |
| Kit disclosure (`assets/sza-kit.css`) | `▸` | - | - | exception: vendored kit, not this product's to edit (same proposals) |

## System surfaces and artwork

| Surface | What it shows | Status |
| --- | --- | --- |
| `.ico` (16-256 px), extension action icons (16/32/48/128), MSIX tiles and `Square44x44Logo` `targetsize-*` / `altform-unplated` / `altform-lightunplated` through `resources.pri` | the product mark: a white sheet with a folded corner and `</>` cut into it, on the navy plate `#1E3A8A`; a simpler 16-20 px drawing without the slash | artwork (`ICON-SET` rule 7) in the platform forms of `ICON-RENDER` rule 9 - the extension icon is the mark on its own plate (plate at least 9.3 : 1 on a light toolbar, sheet at least 11.2 : 1 on a dark one) |
| "Convert to HTML" shell verb | `action.convert`, mono, `#808080`, 16 px with 20, 24, 32 (`assets/convert-verb.ico`, exe icon resource 1) | conforms (rule 9: 3.9 : 1 on the light menu, 3.6 : 1 on `#2B2B2B`) |
| Registered document type (`-register`, and the MSIX file type association) | `content.document`, mono, `#808080`, 16-256 px (`assets/document-type.ico`, exe icon resource 2; `DocumentType` in the package) | conforms (rule 9) |
| A converted book's own images | the book's content | outside the vocabulary (`ICON-EXTERNAL`; the proposal's item 12 is still open) |

No third-party mark is drawn anywhere (`ICON-EXTERNAL` rule 1 holds by absence): the store channels
are text links.
