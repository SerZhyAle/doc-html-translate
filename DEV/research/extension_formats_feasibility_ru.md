# Выполнимость: форматы EPUB/PDF/MOBI/AZW3/FB2/RTF/TXT/Markdown/HTML в расширении

**Тип:** research (шаг 1 из плана пользователя)
**Дата:** 2026-07-01
**Статус:** Готово - вердикт положительный

> Вопрос: можно ли перенести в браузерное расширение поддержку тех же форматов,
> что уже реализованы в EXE (CLI/GUI)? Как именно и с какими рисками?

## Краткий вердикт

Да, выполнимо целиком на клиентском JavaScript, без нативных зависимостей и без
Calibre. Расширение уже архитектурно готово: оно на 100% клиентское (моста к EXE
нет), уже вендорит тяжёлые библиотеки (PDF.js ~3 МБ, Tesseract.js ~3.8 МБ WASM) и
имеет конвейер reflow/render, не зависящий от формата. Самая сложная часть -
MOBI/AZW3, которые на десктопе требуют Calibre, - в браузере закрывается чистым
JS-парсером `foliate-js` (MIT). Ни один формат не требует WASM-компиляции Go.

## Как расширение устроено сейчас

- Manifest V3, `minimum_chrome_version: 105` (гарантирует WASM SIMD). Файл
  [extension/manifest.json](../../extension/manifest.json).
- Полностью клиентское: **нет** native messaging, нет вызова `doc-html-translate` /
  `doc-html-ui`. Весь парсинг идёт в браузере.
- Поддерживает сейчас **только PDF и EPUB**:
  - PDF - через вендоренный **PDF.js** плюс собственная эвристика reflow
    ([extension/src/reflow.js](../../extension/src/reflow.js)).
  - EPUB - собственный ZIP-ридер на `DecompressionStream`
    ([extension/src/epub.js](../../extension/src/epub.js)), **не** foliate-js.
- Конвейер не зависит от формата: парсер выдаёт промежуточную структуру блоков ->
  общий рендер (`renderBlocks` в [extension/src/viewer.js](../../extension/src/viewer.js)),
  общие детект языка ([lang.js](../../extension/src/lang.js)) и TOC ([toc.js](../../extension/src/toc.js)).
- Ввод файлов: (1) перехват URL через `declarativeNetRequest` только для `.pdf` и
  `.epub`; (2) файловый пикер на странице viewer (`accept="...pdf,...epub"`).
- Сборка без бандлера: нативные ES-модули, `node build.mjs` вендорит зависимости и
  пакует ZIP. Релиз - тег `ext-v*` -> GitHub Actions -> Chrome Web Store (+ Edge).
- Соответствие JS<->Go отслеживается в [docs/PARITY.md](../../docs/PARITY.md).

**Следствие:** правильная модель переноса - не "скомпилировать Go в WASM", а
"дописать JS-парсеры новых форматов по образцу уже существующих PDF/EPUB и
скормить их общему рендеру". Это совпадает с уже принятой в проекте практикой
(порт, а не компиляция).

## Природа парсеров на десктопе (Go)

Из [internal/pipeline/pipeline.go](../../internal/pipeline/pipeline.go) (диспетчер по
расширению) и per-format пакетов. Классификация по переносимости в браузер:

| Формат | Реализация в Go | Внешние зависимости | Перенос в браузер |
|---|---|---|---|
| EPUB | `archive/zip` + `encoding/xml`, чистый Go | нет | уже сделано (epub.js) |
| PDF | pdftotext (внешний) + fallback `ledongthuc/pdf` + `pdfcpu` | pdftotext (Poppler) | уже сделано (PDF.js) |
| TXT | чистый Go, ~155 стр | нет | тривиально |
| Markdown | `yuin/goldmark`, чистый Go | нет | JS-библиотека MD->HTML |
| FB2 | `encoding/xml`, чистый Go, ~194 стр | нет | легко (DOMParser / foliate-js) |
| RTF | собственный стриппер + cp1251, ~287 стр | нет | порт стриппера на JS |
| HTML | `golang.org/x/net/html`, чистый Go | нет | тривиально (DOMParser) |
| MOBI/AZW3 | делегирует `ebook-convert` -> EPUB | **Calibre** | foliate-js `mobi.js` |

Важно: MOBI/AZW3 на десктопе вообще не парсятся своим кодом - это shell-out в
Calibre. В браузере Calibre недоступен, поэтому единственный путь - независимый
JS-парсер.

## Стратегии переноса и выбор

**A. Порт парсеров на JS (выбрано).** Дописать JS-парсеры, отдающие блочную
структуру в общий рендер. Совпадает с текущей архитектурой (epub.js, reflow.js -
это уже порты), не тянет файловую модель Go-пайплайна (тот пишет `page_NNN.html` на
диск), даёт минимальный вес для простых форматов.

**B. Компиляция чистого Go в WASM (`GOOS=js`) (отклонено).** Плюс - переиспользование
точной логики Go. Минусы: (1) нет ни одного wasm-таргета в проекте; (2) Go-пайплайн
завязан на файловую систему (outputDir, `page_NNN.html`) - потребует рефакторинга
под in-memory; (3) EPUB/PDF в браузере уже на JS - wasm-версии дублировали бы их;
(4) предпочтительный путь PDF (pdftotext) и MOBI (Calibre) в wasm не собираются в
принципе; (5) вес Go-wasm бинаря. Слишком крупная архитектурная ставка ради
форматов, которые на JS делаются дёшево.

**C. Native messaging к EXE (отклонено).** Требует установленного EXE и
зарегистрированного host-манифеста; ломает ценность "поставил из Store и работает
без установок", а расширение - самостоятельный продукт Store. Кроссбраузерность
(Edge) тоже страдает.

## Ключевая находка: foliate-js закрывает MOBI/AZW3/FB2

`foliate-js` (johnfactotum) - чистый JS, MIT, нативные ES-модули, без шага сборки;
читает EPUB, MOBI, KF8/AZW3, FB2, CBZ прямо из File/Blob. Модули:
`mobi.js` (MOBI + KF8/AZW3), `fb2.js` (FictionBook 2), `epub.js`, `comic-book.js`,
`pdf.js`. Book-интерфейс: `.sections[]` (у секции `.createDocument()` -> `Document`,
`.load()` -> URL), `.metadata`, `.toc` (`.label`/`.href`/`.subitems`),
`.resolveHref()`. Зависимости: `zip.js` (BSD-3, для EPUB/CBZ), `fflate` (MIT, для
шрифтов KF8), опц. PDF.js (уже вендорится). Браузерные API: DOMParser, Blob,
TextDecoder, Web Crypto (деобфускация шрифтов EPUB, нужен HTTPS) - всё есть в MV3.
CSP менять не нужно (foliate-js - чистый JS, без wasm).

**Риск:** README прямо помечает библиотеку как "not stable". Митигирую: вендорить
зафиксированный коммит, вендорить только нужные модули (`mobi.js`, при желании
`fb2.js`) плюс `fflate`, покрыть реальными файлами в приёмочных тестах.
DRM-защищённые Kindle-файлы не парсятся (как и Calibre без плагинов) - вне охвата.

## Выполнимость по форматам (для расширения)

| Формат | Подход | Сложность/риск |
|---|---|---|
| TXT | нативный JS (абзацы + пагинация, порт логики txt) | низкая |
| Markdown | вендорить MD->HTML (marked/markdown-it, MIT) -> рендер | низкая |
| FB2 | foliate-js `fb2.js` **или** порт на DOMParser (~194 стр) | низкая/сред. |
| RTF | порт Go-стриппера на JS (~287 стр, cp1251) | средняя |
| HTML | DOMParser для локальных `.html`; ценность низкая (Chrome переводит HTML сам) | низкая |
| MOBI | foliate-js `mobi.js` (+ `fflate`) | средняя/высокая |
| AZW3 (KF8) | тот же `mobi.js` | средняя/высокая |

Оговорка по HTML: онлайн HTML-страницы Chrome переводит нативно, reflow им не нужен;
относительно к делу только локальные `.html`. Кандидат на низкий приоритет/опцию.

## Затрагиваемые области (вход в спецификацию)

- Диспетчер по расширению + ввод: файловый пикер `accept=...` и (осторожно) правила
  `declarativeNetRequest`. **Не** перехватывать `.txt`/`.html` через DNR (сломает
  обычный сёрфинг) - для них только пикер. Перехват DNR - для ebook-форматов.
- Адаптер foliate-js book -> блочная структура расширения (по образцу epub.js:
  секция -> `createDocument()` -> `<body>` -> санитайз -> `renderBlocks`, картинки в
  `blob:`), маппинг `.toc` в структуру [toc.js](../../extension/src/toc.js).
- Новые модули под `extension/src/` (напр. `mobi.js`-адаптер, `fb2.js`, `rtf.js`,
  `txt.js`, `md.js`) + записи в `web_accessible_resources` манифеста.
- `extension/build.mjs`: вендорить foliate-js (mobi.js/fb2.js + fflate) и MD-либу.
- [docs/PARITY.md](../../docs/PARITY.md): добавить строки соответствия JS<->Go для
  новых форматов.
- Документация и Store (шаг 4): имя/описание расширения (сейчас "PDF & EPUB Page
  Translator" - потребует переименования), [extension/store/LISTING.md](../../extension/store/LISTING.md),
  [extension/store/PRIVACY.md](../../extension/store/PRIVACY.md),
  [extension/README.md](../../extension/README.md), `_locales/{en,ru,uk}/messages.json`,
  корневой [README.md](../../README.md), `docs.html`, `DEV/CHANGELOG.md`, ассоциации
  файлов в манифесте.

## Риски и открытые вопросы

- Стабильность foliate-js ("not stable") - пин коммита + приёмочные тесты на живых
  файлах.
- Граф зависимостей модулей foliate-js: заявлено "в целом не зависят друг от друга",
  но `mobi.js` тянет `fflate`; уточнить фактический набор при написании плана.
- Переименование листинга Store влияет на идентичность/SEO - решение уровня релиза.
- Объём охвата: включать ли MOBI/AZW3 в первую поставку или фазировать
  (простые форматы -> потом ebook). Решается в спецификации.
- Пагинация/сплит больших страниц (в Go есть `htmlsplit` под лимит ~5000 симв.
  Chrome GT) - проверить, нужен ли аналог в расширении для крупных TXT/MOBI.

## Следующие полезные чтения (для шага 2 - спецификация)

1. [extension/src/epub.js](../../extension/src/epub.js) - эталон адаптера: ZIP ->
   секции -> санитайз -> blob-картинки -> рендер.
2. [extension/src/viewer.js](../../extension/src/viewer.js) - `renderBlocks`,
   диспетчер по типу, файловый пикер, детект языка.
3. [extension/src/background.js](../../extension/src/background.js) - правила DNR и
   контекст-меню (точка расширения перехвата).
4. [extension/build.mjs](../../extension/build.mjs) - как вендорятся зависимости.
5. [docs/PARITY.md](../../docs/PARITY.md) - формат строк соответствия JS<->Go.
6. [DEV/plan/_TEMPLATE_cross-edition.md](../plan/_TEMPLATE_cross-edition.md) - шаблон
   тикета (edition parity checklist).
