# Glyph map

This product's inventory against the shared icon vocabulary (`ICON-SET` 0.36, `ICON-RENDER` 0.24,
`ICON-EXTERNAL` 0.12 - pointers in [`contracts/`](contracts/)). It is the product's rung-2 record
(`ICON-SET` section 6: every glyph it shows maps to an id, or is declared artwork). Mapped on
2026-09-25 by ticket 24 (`DEV/plan/done/24_2026-09-23_contract-iconography-sync.md`).

Every glyph that conforms is drawn from the catalog's own file, vendored byte for byte under
[`../assets/glyphs/`](../assets/glyphs/) with its SHA-256 in `PROVENANCE.txt`. The two editions keep
one table each - `internal/htmlgen/glyphs.go` (desktop reader chrome) and `extension/src/glyphs.js`
(extension) - and the GUI and the site carry inline copies. `tests/iconography_test.go` holds all of
them to the vendored files, the vendored files to their provenance (and to the catalog when
`SZA_CONTRACTS_ROOT` names it), both editions' names for a shared control to each other, and this
file to the vendored set. With the catalog mounted it also checks that no control's name is another
meaning's name in the catalog (`TestIconographyNamesAreNotAnotherMeaningsName`, `ICON-SET` rule 3). `internal/htmlgen/glyphs_test.go` pins the paging pair and every
glyph-only reader control, glyph and name, in all thirteen languages.

Status: **conforms** - the vocabulary glyph under its canonical name (qualified where rule 3 allows);
**exception** - kept as it is under a dated exception in the catalog registry, with what would end
it; **artwork** - product-private, outside the vocabulary (`ICON-SET` rule 7). The catalog proposal
is `iconography/PROPOSAL-2026-09-25-doc-html-translate.md`; its items were decided in `ICON-SET` 0.15
and `ICON-RENDER` 0.13 on 2026-09-25, except the co-signed and open ones named below.

## Converted book (reader chrome, `internal/htmlgen`)

| Control | Glyph before | Id | Name (en / ru / uk) | Status |
| --- | --- | --- | --- | --- |
| Previous page link | `◀` + "Back" | `media.previous` | Previous page / Предыдущая страница / Попередня сторінка | conforms; mirrored for RTL document paging (section 13 C); the glyph leads the caption (`BUTTONS` 2.0 section 2 rule 4) |
| Next page link | `▶` + "Forward" | `media.next` | Next page / Следующая страница / Наступна сторінка | conforms; mirrored for RTL document paging (section 13 C); since ticket 119 (slice R1, 2026-10-20) the glyph leads the caption here too, where it used to follow it |
| Table of contents link | `☰` | `nav.contents` | Table of contents / Оглавление / Зміст | conforms - "Оглавление" is the record's declared book-reader form (0.15) |
| Search button | word only | `action.search` | Search / Поиск / Пошук | conforms since ticket 119 (slice R1, 2026-10-20): the vendored glyph before its word; a disclosure, so it keeps its action glyph and shows the open state through `aria-expanded` and the selected face |
| Version link (project site) | text only | `nav.open-external` | (the version text) | conforms since ticket 119 (R1): the vendored drawing as a CSS mask after the text, mirrored in a right-to-left bar (`rtl: mirror`) |
| Fit width toggle (image-page books) | word only | - (no vocabulary meaning) | Fit width | no glyph on purpose: the vocabulary names no such meaning and a private picture is not an option (`ICON-SET` rule 5); text-only is permitted for a command without a recognisable meaning (`BUTTONS` 2.0 section 2 rule 1) |
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
| A document picture that cannot be shown (chapter page, single page) | the browser's broken-image icon | `content.image` | the picture's own alt text, no string of this product | conforms since ticket 122 (`ICON-EXTERNAL` section 13 F): a capture-phase `error` handler in the page head swaps the failed `<img>` for the glyph and its alt text, with no new request; a picture inside an OCR plate container is left alone, its plates being pinned to the picture's box |

## Extension (viewer, popup, options)

| Control | Glyph before | Id | Name (en) | Status |
| --- | --- | --- | --- | --- |
| Table of contents button | `☰` + untranslated "Contents" | `nav.contents` | Table of contents (`ttToc`, the desktop's words) | conforms |
| Search button | word only | `action.search` | Search (`vSearch`, the desktop's word) | conforms since ticket 119 (slice R1, 2026-10-20), as in the book |
| Project and feedback links | text only | `nav.open-external` | (link text) | conforms since ticket 119 (R1): an inline glyph after the text, mirrored in a right-to-left toolbar |
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
| Popup "Settings" command | - (was a bare `<a href="#">`) | `app.settings` | Settings (`popupOptions`) | conforms since ticket 119 (slice X1, 2026-10-20): a button with the glyph before the word; vendored with this slice |
| Popup "Open a document.." button | - | `content.document` | Open a document.. (`popupOpenDoc`) | conforms since slice X1: the primary command of the task panel shows the kind of content it opens, as the launcher's drop zone does |
| Options page rail (six destinations, monochrome) | - | `app.settings`, `action.extract-text`, `system.internet`, `action.export`, `action.text-settings`, `app.info` | General / Image OCR / Sites / Data / Interface / About (`pageGeneral`, `pageOcr`, `pageSites`, `pageData`, `pageInterface`, `optAbout`) | conforms since slice X1 - one glyph and one short name per destination, one monochrome treatment for the whole rail (`SETTINGS-NAVIGATION` 2.3); `system.internet` is the catalog's globe, vendored with this slice |
| Pages button (narrow options tab) | - | `nav.contents` | Pages (`pagesLabel`) | conforms - opens the same menu as a drawer (`SETTINGS-NAVIGATION` 2.6) |
| Export / Import / Reset settings | - | `action.export` / `action.import` / `action.reset` | Export settings / Import settings / Reset settings | conforms since slice X1; `action.import` is the catalog's arrow falling into a tray, vendored with this slice |
| Copy diagnostics / Save diagnostics as a file | - | `action.copy` / `action.save` | Copy diagnostics (`btnCopyDiag`) / Save diagnostics as a file (`btnSaveDiag`) | conforms since slice X1 |
| OCR language Download / Retry | - | `action.download` / `action.retry` | Download (`ocrDownload`) / Retry (`ocrRetry`) | conforms since slice X1; since ticket 120 the Russian and Ukrainian captions are the record's names (Повторить попытку / Повторити спробу), no longer the names of `action.redo` |
| Remove a site from a list | - | `action.remove` | Remove (`optHostRemove`) | conforms since slice X1; the accessible name adds the site after the visible word |
| Dialog Cancel | - | `action.cancel` | Cancel (`dlgCancel`) | conforms since slice X1 - the dismissing answer in the ordinary ink on the neutral face, first in the row; the Russian caption is the record's name "Отмена" since ticket 120 ("Отменить" is the name of `action.undo`) |
| Option-help Close | - | `nav.close` | Close (`dlgClose`) | conforms since ticket 120 - the plain cross of the record, in the ordinary ink on the neutral face; it drew `action.cancel` under the word Close before, and the glyph is vendored with that change |
| Option help button | - | `app.help` | Help: <option name> (`helpOptLabel`) | conforms since slice X1 - the shared help meaning named for its option (`BOOLEAN-OPTION` 4.4); the reader pages and the popup carry no surface help (ticket 118 decision 5) |
| Image-OCR option heading | - | `view.text-layer` | Use OCR for images | conforms (`BOOLEAN-OPTION` 2.2 heading glyph), as in the launcher |
| In-page OCR bar: Cancel / Scan new pictures / Hide or Show text / Remove | - | `action.cancel` / `action.extract-text` / `view.text-layer` / `action.remove` | Cancel (`pageOcrStop`, the words of `dlgCancel`: ending a running task is `action.cancel`, `ICON-SET` 0.14 decision (c); the caption was Stop, the name of `media.stop`, until ticket 120) / Scan new pictures / Hide text, Show text / Remove | conforms since slice X1 (the four drawings are carried inside `page-agent.js`, a content script that cannot import `glyphs.js`; `extension/test/page-agent.test.mjs` holds them to this table); "Select text" is text only (no vocabulary meaning) |
| Viewer export card: Cancel / Export partial / Prepare all | - | `action.cancel` / `action.export` / `action.export` | Cancel (`vExportCancel`) / Export partial (`vExportPartialBtn`) / Prepare all (`vExportPrepareBtn`) | conforms since slice X1 (second part, 2026-10-20): the dismissing answer first in the ordinary ink on the neutral face, the one primary last; both exports end in a file, so both draw the export meaning |
| Viewer status bar Cancel | - | `action.cancel` | Cancel (`vExportStop`) | conforms since slice X1 - ending a running task is `action.cancel`, as in the launcher |
| Viewer resume offer: Start over / Continue reading | - / `feature.continue-reading` | `action.cancel` / `feature.continue-reading` | Start over (`vResumeReset`) / Continue reading (`vResumeContinue`) | conforms since slice X1: the dismissing answer carries the launcher's dismissing glyph and reads first |
| Option checkbox tick (`.opt-box`: options page, popup) | the check mark, a CSS mask over the checked box | - | - | chrome, declared per `ICON-SET` section 13 E rule 3, as in the launcher: the tick says nothing beyond the control and is never the glyph of a command |
| Group header (popup language list, options About groups) | state triangle, drawn by the component (not a vocabulary glyph) | - | the group topic | conforms to `COLLAPSIBLE-GROUP` 3.2 (ticket 119, slice X1): a vector marker - right folded, down open, mirrored in a right-to-left interface - never the `nav.expand` / `nav.collapse` action chevrons |
| Viewer remote-images notice: Load them / Always load remote content | - | `action.download` | Load them (`vRemoteLoad`) / Always load remote content (`vRemoteAlways`) | conforms since slice X1 - the record's meaning is fetching from the internet to this device |
| Viewer empty-state and failure notices: Open a document | - | `content.document` | Open a document (`vOpenDocTitle`) | conforms since slice X1, the same picture the popup's button draws; Open original PDF and Unlock are words (no meaning vendored) |
| A document picture that cannot be shown (viewer) | the browser's broken-image icon | `content.image` | the picture's own alt text, no string of this product | conforms since ticket 122 (`ICON-EXTERNAL` section 13 F), as in the book: `missing-image.js` swaps a failed `<img>` for the glyph and its alt text on a capture-phase `error` listener; a picture the EPUB renderer cannot find inside the archive raises no `error`, so `rewriteImg` and `convertSvgImage` put the same stand-in in its place at render time |

## GUI launcher (`cmd/doc-html-ui/ui.html`)

Every command button of the launcher carries its meaning's glyph before its caption at the 16 px
tier (WINDOWS-UI 0.4 section 11, adopted 2026-10-11, ticket 112): mono in the window's own
colour everywhere. On a dialog's answer buttons the look is `BUTTONS` 2.0 section 3 (ticket 119):
every answer sits on the plain control face, the dismissing answer in the ordinary ink, the
affirmative with `state.ok` on its glyph, a caution with `state.warning` on its glyph, and the acting
button of an irreversible confirmation alone on the `danger` face with its glyph in the face's own
ink. The 12 percent tinted faces and the red dismissing glyph of `ICON-RENDER` 0.19 section 14 are
retired; `TestDialogAnswerFacesMeetContrast` bans them.

| Control | Glyph before | Id | Name (en) | Status |
| --- | --- | --- | --- | --- |
| Group header (About) | state triangle, drawn by the component (not a vocabulary glyph) | - | the group topic | conforms to `COLLAPSIBLE-GROUP` 3.2: a vector state marker - right folded, down open, leftward folded in a right-to-left window - never the `nav.expand` / `nav.collapse` action chevrons, which the launcher no longer draws (ticket 119) |
| Option checkbox tick (`.opt-box`, the boolean options) | the check mark, a CSS mask over the checked box | - | - | chrome, declared per `ICON-SET` section 13 E rule 3: the tick of a check box says nothing beyond the control; its path is the Material `check` that `action.confirm` also draws, used only as this part of this control and never as the glyph of a command |
| Drop zone | `⤓` (`nav.scroll-bottom`'s shape) | `content.document` | Drag a document here, or click to choose one | conforms - the zone is a drop target showing its content kind, not a bare command button |
| Swap source and target | `⇄` | `action.swap` | Swap source and target | conforms (glyph-only, tooltip and accessible name, 28 px target) |
| Convert | text only | `action.convert` | Convert | conforms (glyph on the accent fill, the primary action outside a dialog) |
| Cancel a run / Cancel the queue / No, thanks | `action.cancel` | `action.cancel` | Cancel / Cancel queue / No, thanks | conforms; the Russian base caption is the record's name "Отмена" since ticket 120 ("Отменить" is the name of `action.undo`); "Отменить очередь" and "Отменить текущий" keep the verb form of the name qualified by its object |
| First-run add buttons | text only | `action.add` | Add the right-click entry / Add it and make this app the default | conforms (the caption's own verb, `rule 3`) |
| Open result / Open Default apps / Open the archive / a queue row's result | text only | `nav.open-external` | Open result / Open Default apps / Open the archive | conforms (`rtl: mirror` mirrored in the right-to-left window) |
| Delete previous result | text only | `action.delete` | Delete previous result | conforms |
| Clear all (recent) / Clear all (queue) / Clear stored logs | `action.clear-all` | `action.clear-all` | Clear all / Clear all / Clear stored logs | conforms; since ticket 120 the queue button reads the record's name wherever the catalog has one (twelve of thirteen languages; "Clear" is the name of `action.clear-input`, a text field's) and Urdu, which the catalog has no name for, reads the faithful "سب صاف کریں" |
| Retry failed / Try again (failure dialog) | `action.retry` | `action.retry` | Retry failed / Try again | conforms - vendored since `ICON-SET` 0.36 (ticket 119); the drawing it replaces was the provisional Material `replay` |
| Browse.. (output folder) | text only | `action.choose-folder` | Browse.. | conforms (qualified form: the field names the folder) |
| Save (API key) | text only | `action.save` | Save | conforms |
| Download OCR data | text only | `action.download` | Download | conforms; while downloading the button is its Cancel and shows `action.cancel` |
| Check for updates | `app.update` | `app.update` | Check for updates | conforms - vendored since `ICON-SET` 0.36 (ticket 119); the drawing it replaces was the provisional Material `update` |
| Send logs to the author (button and failure dialog) | text only | `action.email` | Send logs to the author | conforms; first shipper of the `proposed` record (`ICON-SET` 0.19), named in the same proposal |
| Copy (command line) / Copy selection | text only | `action.copy` | Copy / Copy selection | conforms |
| Jump to latest | text only | `nav.scroll-bottom` | Jump to latest | conforms (the record's qualified journal form) |
| A queue row's Remove | text only | `action.remove` | Remove | conforms (the file stays; the accessible name adds the file after the visible word) |
| Readiness shortcuts (Choose OCR data / Save API key / Choose model) | text only | `nav.go-to` | Choose OCR data / Save API key / Choose model | conforms as a judgment call: the buttons jump the reader to the setting that answers the readiness issue, and `nav.go-to` is the vocabulary's to-destination meaning; recorded in the registry row (`rtl: mirror` mirrored) |
| Dialog OK / Open (secret file) | `action.confirm` | `action.confirm` | OK / Open | conforms - the affirmative answer: `state.ok` on the glyph over the neutral control face (`BUTTONS` 2.0 section 3 rule 5), last in the row; the caption "OK" is the name of `status.ok` over the glyph of `action.confirm` (named "Apply"), a permanent variant (ticket 119 decision D6) |
| Dialog Cancel / No / Don't translate | `action.cancel` | `action.cancel` | Cancel / Don't translate | conforms - the dismissing answer in the ordinary ink on the neutral face, first in the row (`BUTTONS` 2.0 section 3 rule 5 retired the red glyph) |
| Failure dialog Close / option-help Close | `nav.close` | `nav.close` | Close | conforms since ticket 120 - the plain cross of the record, in the ordinary ink on the neutral face, first in the row; both drew `action.cancel` under the word Close before (`ICON-SET` rule 2 sets `nav.close` against `action.cancel`), and the glyph is vendored with that change |
| Dialog Delete / Delete the logs / Delete and rebuild | text only | `action.delete` | Delete / Delete the logs / Delete and rebuild | conforms - the acting button on the `danger` face, glyph in the face's own ink |
| Dialog Convert without translating | `action.convert` | `action.convert` | Convert without translating | conforms - the caution answer: `state.warning` on the glyph, neutral face |
| Help button (header corner, and the corner of the cost, secret-file and failure dialogs) | - (new 2026-10-20, ticket 118) | `app.help` | Help (F1) / Справка (F1) / Довідка (F1) - the record's names in all thirteen languages | conforms - the glyph alone, name and tooltip from the vocabulary with the key appended (`CONTEXT-HELP` rule 2); F1 and the button open the same page through `cmd/doc-html-ui/help.js` |
| Dialog Translate and pay | text only | `action.translate` | Translate and pay | conforms - the acting button on the `danger` face (the product's registered cost confirmation), glyph in the face's own ink |
| Page rail (seven destinations, monochrome) | `action.convert`, `action.translate`, `action.extract-text`, `action.create-shortcut`, `action.text-settings`, `action.adjust`, `app.info` | one per page | Convert / Ollama / Image OCR / Windows / Interface / Advanced / About (13 languages, `pageConvert` ..) | conforms - one glyph and one short name per destination, one monochrome treatment for the whole rail (`SETTINGS-NAVIGATION` 2.3, decision D4 of ticket 119); `action.extract-text` is the catalog's OCR drawing, `action.create-shortcut` and `action.text-settings` are `proposed` records used as they stand |
| Pages button (narrow window) | `nav.contents` | `nav.contents` | Pages | conforms - opens the same menu as a drawer (`SETTINGS-NAVIGATION` 2.6) |
| Image-OCR option heading | `view.text-layer` | `view.text-layer` | Overlay translatable text on images | conforms - the recognized-text layer is exactly what the option adds (`BOOLEAN-OPTION` 2.2 heading glyph) |
| Option help button (OCR overlay, default handler) | `app.help` | `app.help` | Help: <option name> | conforms - the shared help meaning, named for its option (`BOOLEAN-OPTION` 4.4), distinct from the surface help in the header corner |

## Site (`index.html`, the ten language pages, `extension.html`)

| Control | Glyph before | Id | Name (en / ru / uk) | Status |
| --- | --- | --- | --- | --- |
| To top button | `↑` + "Back to top" | `nav.scroll-top` | Scroll to top / В начало страницы / На початок сторінки | conforms (the record's qualified page form, 0.15) |
| Cross-link to the desktop app | `←` | `nav.go-to` | Desktop app | conforms |
| Section disclosure (`details.sec`: the home page, the docs trio, `extension.html`) | a leading state triangle drawn by the page layer (not a vocabulary glyph) | - | section title | conforms to `COLLAPSIBLE-GROUP` 3.2 since ticket 119 (W1): right folded, down open, mirrored in RTL; the typed `▸` and the `nav.expand` / `nav.collapse` chevrons are no longer drawn on the site (the kit's own end marker is switched off in `assets/site.css`) |
| External-link button (`.btn[target="_blank"]`) | the `nav.open-external` mask after the caption | `nav.open-external` | (the link text) | conforms to `BUTTONS` 2.0 section 2 rule 3 since ticket 119 (W1): the vendored drawing as a CSS mask in `assets/site.css`; `rtl: mirror` honoured |
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
| A converted book's own images | the book's content | user content (`ICON-EXTERNAL` section 13 F); one that cannot be shown draws `content.image` with its alt text (rows above, ticket 122) |

No third-party mark is drawn anywhere (`ICON-EXTERNAL` rule 1 holds by absence): the store channels
are text links.

**Review 2026-10-07 (ticket 108).** The seven shipped records are active. The reader
glyph targets of that review (28 px for a fine pointer, 44 px for a coarse one; 44 px for the site) are
superseded: ticket 119 moved every surface to the shared 32 / 48 CSS px tokens (the site 44 / 48), see
[`UI_PROFILE.md`](UI_PROFILE.md).
The newly imported names in the ten further languages are not all aligned yet: the table's
glyph conformity does not claim that label audit, which is a dated catalog exception.
