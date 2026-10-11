# Glyph map

This product's inventory against the shared icon vocabulary (`ICON-SET` 0.29 carrier / 0.30 README,
`ICON-RENDER` 0.19, `ICON-EXTERNAL` 0.12 - pointers in [`contracts/`](contracts/)). It is the product's rung-2 record
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
| Previous page link | `◀` + "Back" | `media.previous` | Previous page / Предыдущая страница / Попередня сторінка | conforms; mirrored for RTL document paging (section 13 C) |
| Next page link | `▶` + "Forward" | `media.next` | Next page / Следующая страница / Наступна сторінка | conforms; mirrored for RTL document paging (section 13 C) |
| Table of contents link | `☰` | `nav.contents` | Table of contents / Оглавление / Зміст | conforms - "Оглавление" is the record's declared book-reader form (0.15) |
| Contents panel close (single page) | `×` + "Close contents" | - (`nav.close` as its word) | Close / Закрыть / Закрити | conforms - a text button, the search panel's close word; the typed cross was no vocabulary drawing (`ICON-RENDER` rule 7, 2026-10-06) |
| Continue reading (index) | `▸` | `feature.continue-reading` | Continue reading / Продолжить чтение / Продовжити читання | conforms |
| Text smaller | `A−` | `action.text-smaller` | Smaller text / Мельче / Дрібніше | conforms |
| Text larger | `A+` | `action.text-larger` | Larger text / Крупнее / Більше | conforms |
| Recognized text layer toggle | `▤` | `view.text-layer` | Text layer / Текстовый слой / Текстовий шар | conforms in glyph and name; its state is an exception (`ICON-RENDER` rule 4: the record's off form is owed, so `aria-pressed` and a pressed look carry it) |
| Night mode quick toggle | - (new 2026-09-29, ticket 59) | `app.night-mode` | Night mode / Ночной режим / Нічний режим | conforms; its state rides `aria-pressed` like the text layer's (pressed = a night-family theme is on) |
| Reset text size | - (new 2026-09-29, ticket 59) | `action.reset` | Reset text size / Сбросить размер текста / Скинути розмір тексту | conforms - the record's name qualified by its object, as the catalog qualifies `action.convert` by its target |
| Theme select | `☀ ◑ ☾ ●` on the options | `app.theme` | Theme / Тема / Тема; options Light / Sepia / Dark / Night as words (Светлая / Сепия / Тёмная / Ночная, Світла / Сепія / Темна / Нічна) | conforms (the record's choices are words, named as its note names them) |
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
| Night mode quick toggle | - (new 2026-09-29, ticket 59) | `app.night-mode` | Night mode (`ariaNightMode`, the desktop's words) | conforms; `aria-pressed` carries the state |
| Reset text size | - (new 2026-09-29, ticket 59) | `action.reset` | Reset text size (`ariaResetTextSize`, the desktop's words) | conforms |
| Theme select | - (text) | `app.theme` | Theme (`ariaTheme`) | conforms |
| Page jump | - (text) | `nav.go-to-page` | Go to page (`ttGoToPage`) | conforms |
| Resume-reading offer | - (new 2026-09-29, ticket 53) | `feature.continue-reading` | Continue reading (`vResumeContinue`, the desktop's words) | conforms |

## GUI launcher (`cmd/doc-html-ui/ui.html`)

Every command button of the launcher carries its meaning's glyph before its caption at the 16 px
tier (WINDOWS-UI 0.4 section 11, adopted 2026-10-11, ticket 112): mono in the window's own
colour everywhere, and on a dialog's answer buttons the colour look of `ICON-RENDER` 0.19
section 14 - the affirmative in the product's `state.ok` pair, the dismissing answer in
`state.error`, a caution in `state.warning`, each at about 12 percent tint over the control
face, the acting button of an irreversible confirmation alone on the `danger` face with the
dismissing answer's red glyph untinted.

| Control | Glyph before | Id | Name (en) | Status |
| --- | --- | --- | --- | --- |
| Section disclosure | `▸` / `▾` (did not mirror in RTL) | `nav.expand` / `nav.collapse` | section title | conforms (`rtl: fixed`, symmetric) |
| Drop zone | `⤓` (`nav.scroll-bottom`'s shape) | `content.document` | Drag a document here, or click to choose one | conforms - the zone is a drop target showing its content kind, not a bare command button |
| Swap source and target | `⇄` | `action.swap` | Swap source and target | conforms (glyph-only, tooltip and accessible name, 28 px target) |
| Convert | text only | `action.convert` | Convert | conforms (glyph on the accent fill, the primary action outside a dialog) |
| Cancel a run / Cancel the queue / No, thanks | text only | `action.cancel` | Cancel / Cancel queue / No, thanks | conforms |
| First-run add buttons | text only | `action.add` | Add the right-click entry / Add it and make this app the default | conforms (the caption's own verb, `rule 3`) |
| Open result / Open Default apps / Open the archive / a queue row's result | text only | `nav.open-external` | Open result / Open Default apps / Open the archive | conforms (`rtl: mirror` mirrored in the right-to-left window) |
| Delete previous result | text only | `action.delete` | Delete previous result | conforms |
| Clear all (recent) / Clear (queue) / Clear stored logs | text only | `action.clear-all` | Clear all / Clear / Clear stored logs | conforms |
| Retry failed / Try again (failure dialog) | text only | `action.retry` | Retry failed / Try again | provisional (`ICON-SET` section 15): drawn Material `replay`, proposed in `PROPOSAL-2026-10-11-doc-html-translate-button-meanings.md`, no catalog file yet |
| Browse.. (output folder) | text only | `action.choose-folder` | Browse.. | conforms (qualified form: the field names the folder) |
| Save (API key) | text only | `action.save` | Save | conforms |
| Download OCR data | text only | `action.download` | Download | conforms; while downloading the button is its Cancel and shows `action.cancel` |
| Check for updates | text only | `app.update` | Check for updates | provisional (`ICON-SET` section 15): drawn Material `update`, proposed with `action.retry` |
| Send logs to the author (button and failure dialog) | text only | `action.email` | Send logs to the author | conforms; first shipper of the `proposed` record (`ICON-SET` 0.19), named in the same proposal |
| Copy (command line) / Copy selection | text only | `action.copy` | Copy / Copy selection | conforms |
| Jump to latest | text only | `nav.scroll-bottom` | Jump to latest | conforms (the record's qualified journal form) |
| A queue row's Remove | text only | `action.remove` | Remove | conforms (the file stays; the accessible name adds the file after the visible word) |
| Readiness shortcuts (Choose OCR data / Save API key / Choose model) | text only | `nav.go-to` | Choose OCR data / Save API key / Choose model | conforms as a judgment call: the buttons jump the reader to the setting that answers the readiness issue, and `nav.go-to` is the vocabulary's to-destination meaning; recorded in the registry row (`rtl: mirror` mirrored) |
| Dialog OK / Open (secret file) | text only | `action.confirm` | OK / Open | conforms - the affirmative answer, `state.ok` glyph and tint |
| Dialog Cancel / No / Close / Don't translate | text only | `action.cancel` | Cancel / Don't translate / Close | conforms - the dismissing answer, `state.error` glyph; no tint in an irreversible confirmation |
| Dialog Delete / Delete the logs / Delete and rebuild | text only | `action.delete` | Delete / Delete the logs / Delete and rebuild | conforms - the acting button on the `danger` face, glyph in the face's own ink |
| Dialog Convert without translating | text only | `action.convert` | Convert without translating | conforms - the caution answer, `state.warning` glyph and tint |
| Dialog Translate and pay | text only | `action.translate` | Translate and pay | conforms - the acting button on the `danger` face (the product's registered cost confirmation), glyph in the face's own ink |

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
| `.ico` (16-256 px), extension action icons (16/32/48/128), MSIX tiles and `Square44x44Logo` `targetsize-*` / `altform-unplated` / `altform-lightunplated` through `resources.pri` | the product mark: a white sheet with a folded corner and `</>` cut into it, on the navy plate `#1E3A8A` (16-20 px drawing with `<>` without slash for 1px stroke clarity; 24-64 px with full `</>`; 96-256 px with "DOC" header and "HTML" footer branding typography framing `</>`) | artwork (`ICON-SET` rule 7) in the platform forms of `ICON-RENDER` rule 9 - the extension icon is the mark on its own plate (plate at least 9.3 : 1 on a light toolbar, sheet at least 11.2 : 1 on a dark one) |
| "Convert to HTML" shell verb | `action.convert`, mono, `#808080`, 16 px with 20, 24, 32 (`assets/convert-verb.ico`, exe icon resource 1) | conforms (rule 9: 3.9 : 1 on the light menu, 3.6 : 1 on `#2B2B2B`) |
| Registered document type (`-register`, and the MSIX file type association) | `content.document`, mono, `#808080`, 16-256 px (`assets/document-type.ico`, exe icon resource 2; `DocumentType` in the package) | conforms (rule 9) |
| A converted book's own images | the book's content | user content (`ICON-EXTERNAL` section 13 F); missing-image replacement is a dated exception |

No third-party mark is drawn anywhere (`ICON-EXTERNAL` rule 1 holds by absence): the store channels
are text links.

**Review 2026-10-07 (ticket 108).** The seven shipped records are active. Reader glyph targets
are 28 px for a fine pointer and 44 px for a coarse pointer; the site uses 44 px for every pointer.
The newly imported names in the ten further languages are not all aligned yet: the table's
glyph conformity does not claim that label audit, which is a dated catalog exception.
