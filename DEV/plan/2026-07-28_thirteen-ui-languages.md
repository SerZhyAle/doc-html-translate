# Thirteen interface languages (CyrFlip parity)

**Status:** Partial (audited 2026-07-28) - every product surface and every distribution channel speaks 13
languages: CLI, converted page, GUI, extension, site, READMEs, installer (8, decision D2), MSIX, winget,
Store listing sources and 39 store screenshots. Phases 01-06 and 09 ✅ Done, 07 🚧 4/5, 08 🚧 6/7.

**Exactly two things are open, and both are recorded decisions rather than oversights:**
1. **Step 07.5** - the extension screenshot generator. Capturing the viewer needs a real
   `chrome-extension://` origin (its `chrome.i18n`/`chrome.storage` do not exist on `file://`). Two
   attempts failed: `--load-extension` under `--headless=new` returns an empty DOM for every
   `chrome-extension://` URL, including `manifest.json`, with the ID derived both ways. The route needs a
   Puppeteer dependency the extension does not have, or a hand-rolled CDP client. The four existing en/ru
   extension PNGs remain valid, so the listing keeps its images.
2. **Half of step 08.7** - the extension's **long** store description stays en/ru/uk. Chrome review
   rejected that exact copy twice in Jul 2026 for "excessive keywords" and the wording that finally passed
   is tuned word by word; machine-translating it is a submission risk, not a nicety. Its **name and short
   description are localized for all 13** and the stores serve them automatically from `_locales`. Overrule
   by having the ten written by a person.

Two plan predicates were corrected against reality during execution and the corrections are logged in the
phase files: the Partner Center export is UTF-8 **with** BOM (not without), and no locale column may be
invented in the CSV - an unknown one is dropped silently on import, so the 11 not-yet-created locales are
reported instead of fabricated.

`docs/PARITY.md` carries the new shared invariant. All six forks were settled by the owner on 2026-07-28
("делаем как правильно"), see **Decisions**.
**Tactical plan:** [`2026-07-28_thirteen-ui-languages/INDEX.md`](2026-07-28_thirteen-ui-languages/INDEX.md)
**Priority:** 30
**Date:** 2026-07-28

> **Cross-edition ticket** (template: [`_TEMPLATE_cross-edition.md`](_TEMPLATE_cross-edition.md)).
> One feature, every edition. Read [`docs/PARITY.md`](../../docs/PARITY.md) first; this ticket adds a new
> shared invariant to it (the UI language set), so PARITY.md must be edited before the work is called done.
>
> Reference implementation in the portfolio: **CyrFlip** (`p:\WINDOWS\CyrFlip`), which shipped the same
> 13-language move on 2026-07-25 - spec `PLAN/done/WorldLanguages_Spec_Idea_v0.1.md`, layer
> `src/CyrFlip/Localization*.cs`, site `docs/<lang>/index.html`, listing `msix/listing/<lang>.txt` +
> `msix/build-store-listing-csv.ps1`, screenshots `tools/uitest/Save-StoreScreenshots.ps1`. Where CyrFlip
> already paid for a lesson, this ticket cites it rather than re-deriving it.

## What / why

Today the product speaks three languages at best and two in the places a user actually reads: the converted
page and the console. A German, Spanish or Arabic user installs a converter whose whole promise is "read and
translate anything in your own language" and meets an English-only interface. This ticket raises the
interface, the primitive per-language documentation, the store listings and the release screenshots to the
same 13 languages CyrFlip ships, so the portfolio has one language set instead of a per-product accident.

Scope is the **product's own chrome**, not the documents it converts. The converter already handles any
content language; nothing here changes extraction, translation or OCR behaviour.

## Do not confuse the four language axes

The word "language" means four unrelated things in this repo. Only the first is in scope.

| Axis | What it selects | Where it lives today | In scope |
|---|---|---|---|
| **UI language** | The words the product itself shows | `syslocale`, `internal/app`, `internal/htmlgen`, `ui.html` `I18N`, `_locales`, `.iss` | **yes - this ticket** |
| Content language | The document's own language, written to `<html lang>` | `internal/htmlgen/singlepage.go` `htmlLang`, [`extension/src/lang.js`](../../extension/src/lang.js) | no |
| Translation source / target | `-src` / `-dst` for Google / Ollama | [`internal/config/flags.go`](../../internal/config/flags.go), `LANGS` in `ui.html` (18 entries) | no |
| OCR language | Tesseract traineddata | [`internal/ocr/tessdata.go`](../../internal/ocr/tessdata.go), [`extension/src/ocr-lang.js`](../../extension/src/ocr-lang.js) | no |

The OCR catalog also happens to have **13** entries (`eng rus ukr jpn jpn_vert deu fra spa ita por pol
chi_sim kor`, a PARITY.md invariant). That is a coincidence of counting, not the same list - the UI set has
no Japanese, Polish or Korean, and the OCR set has no Arabic, Hindi, Bengali or Urdu. Never derive one from
the other.

## The language set

The same 13 as CyrFlip, in the same order, so the portfolio has one list. CyrFlip derived it from keyboard
layouts (the 12 most-typed world languages plus Ukrainian); here the justification is simply portfolio
consistency plus reach - the set covers the large majority of Windows users by first language, and it is a
set the author already has vetted translations for in a sibling repository.

| # | Code | Endonym | Script | Dir | Store locale | Extension `_locales` dir | Inno `.isl` | Notes |
|---|---|---|---|---|---|---|---|---|
| 1 | `en` | English | Latin | LTR | `en-us` | `en` | `Default.isl` | **Key language** - source strings live in English (see below) |
| 2 | `ru` | Русский | Cyrillic | LTR | `ru` | `ru` | `Russian.isl` | author-proofread |
| 3 | `uk` | Українська | Cyrillic | LTR | `uk` | `uk` | `Ukrainian.isl` | author-proofread |
| 4 | `de` | Deutsch | Latin | LTR | `de` | `de` | `German.isl` | longest strings - layout risk |
| 5 | `it` | Italiano | Latin | LTR | `it` | `it` | `Italian.isl` | |
| 6 | `es` | Español | Latin | LTR | `es` | `es` | `Spanish.isl` | |
| 7 | `fr` | Français | Latin | LTR | `fr` | `fr` | `French.isl` | |
| 8 | `pt` | Português | Latin | LTR | `pt-br` | `pt` | `BrazilianPortuguese.isl` | one neutral translation; Brazilian variant wherever a channel forces a choice (D1) |
| 9 | `ar` | العربية | Arabic | **RTL** | `ar` | `ar` | none official | mirroring work |
| 10 | `hi` | हिन्दी | Devanagari | LTR | `hi` | `hi` | none official | needs Nirmala UI |
| 11 | `bn` | বাংলা | Bengali | LTR | `bn` | `bn` | none official | needs Nirmala UI |
| 12 | `ur` | اردو | Arabic | **RTL** | `ur` | `ur` | none official | CyrFlip lost this column once - see C1 |
| 13 | `zh` | 中文 | Han (Simplified) | LTR | `zh-hans` | `zh_CN` | none official | needs Microsoft YaHei UI |

Every decision that shaped this table - the Portuguese variant, the five languages the installer cannot
speak, what is translated per release and what is not - is settled in **Decisions** below, with the
reasoning. There are no open questions in this ticket.

**Key language is English, not Russian.** CyrFlip keys its dictionary on the Russian source string because
its call sites were written in Russian. Here every artifact is English by canon (hard invariant 19) and the
existing string literals in `internal/app`, `internal/htmlgen` and `ui.html` are English, so the key is the
**English** source string. Same mechanism, mirrored: `Translate(lang, en)` returns `en` unchanged for
`en`, falls back to English for a missing translation, and returns the key itself for an unknown key so a
new string is visible but never crashes.

## Current state - the honest inventory

Measured 2026-07-28 by reading the files, not by memory.

| Surface | File(s) | Languages today | Size of the job | Mechanism today |
|---|---|---|---|---|
| CLI splash, registration, prompts | [`internal/app/app.go`](../../internal/app/app.go) | en, ru | 9 `IsRussian` branches + two ~49-line splash blocks | `if syslocale.IsRussian()` |
| Converted page chrome (navbar + reader controls) | [`internal/htmlgen/navbar.go`](../../internal/htmlgen/navbar.go), [`htmlgen.go`](../../internal/htmlgen/htmlgen.go) | en, ru | 12 strings (`Back` `Forward` `Contents`, `Smaller text` `Larger text` `Font` `Theme`, `Light` `Sepia` `Dark` `Night`, `Continue reading`); `Chapters:` and `Serif/Sans/Mono` are not localized at all | `if syslocale.IsRussian()` |
| GUI | [`cmd/doc-html-ui/ui.html`](../../cmd/doc-html-ui/ui.html) | en, ru, uk | **87 keys** per language | `I18N` dict + `data-i18n` / `-ph` / `-title` / `-html` |
| GUI language resolution | [`internal/syslocale/locale_windows.go`](../../internal/syslocale/locale_windows.go), `main.go` `/api/env` | en, ru, uk | `Lang()` maps 2 LANGIDs, everything else is `en` | `GetUserDefaultUILanguage` |
| Extension UI (viewer, options, popup, OCR overlay) | `extension/src/*.html`, `*.js` | **en only** | ~58 visible HTML strings + ~50 JS strings; only `background.js`, `ocr.js`, `options.js`, `popup.js` call `chrome.i18n.getMessage`, and only for **20** keys | hardcoded English with `getMessage(key) \|\| fallback` |
| Extension manifest + store copy | [`extension/_locales/`](../../extension/_locales/) | en, ru, uk | 20 keys × 3 | `__MSG_*__` + `chrome.i18n` |
| Installer | [`installer/doc-html-translate.iss`](../../installer/doc-html-translate.iss) | en, ru, uk | 3 `[Languages]` rows + 3 `CustomMessages` keys × 3 | Inno `.isl` + `[CustomMessages]` |
| Landing page | [`index.html`](../../index.html) | en, ru, ua in-page | ~40 `data-l` triplets | `data-l` spans, CSS hides the others |
| Extension page | [`extension.html`](../../extension.html) | en, ru, ua in-page | 586 lines | same |
| App docs | [`docs.html`](../../docs.html) · [`docs.ru.html`](../../docs.ru.html) · [`docs.uk.html`](../../docs.uk.html) | en / ru / uk, separate files | 373 / 184 / 184 lines | one file per language |
| Privacy pages | [`privacy.html`](../../privacy.html), [`extension-privacy.html`](../../extension-privacy.html) | en, ru, ua in-page | | `data-l` |
| Repo README | [`README.md`](../../README.md) | **en only** | 262 lines | - (CyrFlip has `README_RU.md` + `README_UK.md`) |
| MSIX Store listing | [`tools/store/listingData.csv`](../../tools/store/listingData.csv) | `en-us`, `ru` | one column per locale | hand-maintained Partner Center export |
| MSIX manifest | [`msix/AppxManifest.xml`](../../msix/AppxManifest.xml) | `en-us` | 1 `<Resource Language>` row | |
| winget | [`winget/SerZhyAle.DocHtmlTranslate.locale.en-US.yaml`](../../winget/) | `en-US` | 1 locale manifest | |
| Chrome / Edge listing | [`extension/store/LISTING.md`](../../extension/store/LISTING.md) | en (+ ru/uk blocks) | name + short desc auto-served from `_locales`; long description and captions pasted per language by hand | |
| App screenshots | [`tools/store/make-screenshot.ps1`](../../tools/store/make-screenshot.ps1) → `reading-view.png`, `table-of-contents.png` | effectively **en** (whatever the dev machine's OS language is) | 2 shots | headless Edge over a converted sample |
| Extension screenshots | `extension/store/screenshots/shot{1,2}-{en,ru}.png` | en, ru | 4 shots | by hand |
| Docs surfaces manifest | [`DEV/DOCS_SURFACES.md`](../DOCS_SURFACES.md) | says "3-language parity is mandatory" | | must be rewritten by this ticket |

**Two facts worth pulling out of that table.** First, the surface a user spends the most time in - the
converted page - is the *least* localized (en/ru, 12 strings, chosen by the OS language with no way to
override). Second, the extension edition has an i18n mechanism wired up and uses it for 20 keys, none of
which are the viewer UI; its reader is English for everyone.

## Block A - one localization layer per codebase

Two codebases, so two layers, kept in lockstep by PARITY.md and a test (Block E).

### A1. Go: `internal/i18n`

New package. Public surface, deliberately small and modelled on CyrFlip's `Localization.cs`:

- `Codes` / `Names` - the 13 codes and endonyms, in the table's order; index 0 is `en`, the key language.
- `T(lang, en string, args ...any) string` - translate an English source string; unknown key returns the
  key, empty translation falls back to English.
- `IsRTL(lang) bool` - `ar`, `ur`.
- `Dir(lang) string` - `"rtl"` / `"ltr"`, for the `dir` attribute of generated HTML.
- `FontStack(lang) string` - system UI font for scripts Segoe UI does not cover: `Nirmala UI` (hi, bn),
  `Microsoft YaHei UI` (zh), default otherwise. Same choice as CyrFlip, same reason: these ship with
  Windows, and the GUI runs offline in a local browser window, so a webfont is not an option.
- `Resolve(explicit, saved string) string` - the resolution order below.

Strings are registered in `i18n_<area>.go` files (`i18n_cli.go`, `i18n_reader.go`), one call per source
string with 12 translations, so a new language is one column and not a diff across the tree - the exact
reason CyrFlip replaced its `ru ? .. : uk ? ..` ternaries.

**Language resolution order** (first hit wins):

1. `-ui-lang <code>` - new CLI flag (see A5), also what the screenshot tooling drives.
2. GUI: the user's saved choice in the settings file (already persisted for the current 3).
3. `syslocale.Lang()` - extended from 2 mapped LANGIDs to all 13 primary language IDs.
4. `en`.

`syslocale.IsRussian()` is deleted after the last caller moves; keeping it would invite a fourteenth
`if IsRussian()`. `Lang()` keeps its contract ("limited to the languages the app ships strings for") with a
longer switch: `0x19 ru`, `0x22 uk`, `0x07 de`, `0x10 it`, `0x0A es`, `0x0C fr`, `0x16 pt`, `0x01 ar`,
`0x39 hi`, `0x45 bn`, `0x20 ur`, `0x04 zh`, default `en`.

### A2. The converted page

All 12 chrome strings go through `i18n.T`, plus the two that are English-only today (`Chapters:` and the
font-family option labels). The reader-control markup gains `dir` handling for `ar` / `ur`.

**One invariant this must not break, and it is the product's whole free flow.** The generated page carries
`<html lang="<content language>">` because that is what makes Chrome offer "Translate page". If the chrome
around the text is in a *different* language than the body, Chrome's own language detection can be pulled
towards the chrome - and with 13 UI languages that risk is 13× more likely than it is today. Requirements:

- The navbar / toolbar containers carry an explicit `lang="<ui>"` (and `dir` when RTL) attribute so the
  detector attributes those words to the UI language and leaves the body's language alone.
- `<html lang>` keeps meaning **content**, never UI. This must be stated in `docs/PARITY.md`.
- Verified per language by the `/verify-view` flow, not by reading the HTML (Block E).

RTL is a chrome-only concern here: an Arabic *interface* over a left-to-right book must not mirror the
book's text. Only the navbar and the toolbar mirror; the content area keeps the document's own direction.

### A3. GUI (`cmd/doc-html-ui/ui.html`)

- `I18N` grows from 3 to 13 dictionaries of 87 keys. Machine-assisted like CyrFlip's; en/ru/uk stay
  author-proofread.
- The three-button `#langSwitch` becomes a `<select>` of endonyms - 13 buttons do not fit the header row,
  and CyrFlip made the same call for the same reason.
- `document.documentElement.dir` set from the language; check the flex rows, the `.langs` row, the swap
  button (`⇄` points the wrong way when mirrored) and the copy-box.
- Long-string layout: German is the stress case (CyrFlip measured ~1.15× the Russian width for the same
  phrase). The GUI is CSS flex rather than fixed-width WinForms columns, so this is likely to be free -
  but it is a *check*, not an assumption, and it needs one screenshot per language to close (Block D).
- Font stack per language, from `FontStack` above, delivered through `/api/env`.

### A4. Extension

The extension is not a follower here - it is the edition with the biggest gap (English-only UI). Work:

- Move the ~108 hardcoded viewer / options / popup / OCR strings to `chrome.i18n.getMessage` with the
  existing `getMessage(key) || fallback` idiom, so a missing key degrades to English instead of blank.
- `_locales/<code>/messages.json` for all 13 (`pt_PT`, `zh_CN` per Chrome's locale naming).
- `dir="rtl"` on the viewer for `ar` / `ur`, chrome-only, with the same "do not mirror the document"
  rule as A2.
- The viewer's UI language follows `chrome.i18n.getUILanguage()` by default, with an options-page override
  (decision D6) - the same shape as `-ui-lang` plus the GUI switcher on the desktop side.

### A5. CLI flag and GUI parity

New flag `-ui-lang <code>` in [`internal/config/flags.go`](../../internal/config/flags.go), defaulting to
`""` (meaning "resolve from the OS"). It selects the language of the console output **and** of the
generated page chrome.

`tests/ui_cli_parity_test.go` fails the build if the GUI does not expose a CLI flag, so the GUI must either
forward `-ui-lang` or claim it as GUI-native (its own language switcher already is the control - the
allow-list entry should say so, with the switcher also driving the converter's flag so the page chrome
matches the GUI).

### A6. Installer

- `[Languages]`: the 8 of 13 that Inno Setup ships official `.isl` files for (en, ru, uk, de, it, es, fr,
  pt-BR). The other five (ar, hi, bn, ur, zh) have no official translation in the Inno distribution and
  **fall back to English in the installer only** (decision D2) - the app itself still opens in their
  language.
- `[CustomMessages]`: the 3 existing keys for every language actually listed - 8 languages × 3 keys.
- The MSIX and portable channels are unaffected: this limit is Inno's, and the Store app never sees it.

## Block B - primitive documentation and the site

The ask is explicitly *primitive* documentation per language: a first page, not the full guide (decision
D4). CyrFlip settled it the same way - `docs/<lang>/index.html` only, full guide en/ru/uk.

1. **Per-language landing directories.** `<lang>/index.html` for the 10 new languages, keeping the current
   root `index.html` as the en/ru/ua in-page entry. This mixes two mechanisms (in-page `data-l` for three,
   directories for ten) - accepted, because converting the root page to directories would break every
   existing inbound link and the store/winget URLs that point at it.
2. **Content of each per-language page:** hero, the four use-case cards, the format pills, the install
   block, the "important" note, and links out to the full English guide plus the extension page. One page,
   translated - not a truncated one.
3. **`lang` and `dir="rtl"`** correct per page; verify the glass/blob styling and the header row survive
   mirroring (CyrFlip found the mirrored capture and the mirrored layout are different problems).
4. **Language switcher** in the header of every page, listing all 13 endonyms, plus `hreflang`
   alternates on all of them (root included) and a `sitemap.xml` - neither exists in this repo today,
   while CyrFlip already ships both. Without `hreflang` the ten new pages compete with the root page in
   search instead of complementing it.
5. **Full docs stay en/ru/uk** (`docs.html` trio). The per-language pages link to the English one.
6. **README:** add `README_RU.md` and `README_UK.md` (CyrFlip has both; this repo has neither). All three,
   the guide and every per-language landing page carry one plain sentence saying the ten non-author
   languages are machine-translated and unproofread, with an address for corrections (decision D5 - the
   caveat lives in prose, never as a "beta" tag in the language list). CyrFlip published exactly that
   admission; copying the honesty is part of copying the feature.
7. **`.nojekyll` must survive.** It exists and it is load-bearing: without it GitHub Pages runs Jekyll,
   the build errors, and the live site silently freezes at the last good commit. Ten new directories are
   exactly the kind of change that tempts someone to "clean up" the root.
8. **`DEV/DOCS_SURFACES.md` is rewritten by this ticket.** It currently states "3-language parity is
   mandatory" and "en/ru/uk is one atomic edit, not three requests" as an invariant. After this ticket the
   manifest has to say which surfaces are 13-language and which stay 3-language, or the next `/docs-sync`
   will confidently leave ten languages behind.

## Block C - listings and package metadata

1. **Microsoft Store (Partner Center).** Port CyrFlip's tooling rather than inventing it:
   `tools/store/listing/<lang>.txt` as `@@Field / value` blocks + a `build-store-listing-csv.ps1` that
   fills **only empty cells** of a fresh Partner Center export and never touches asset URLs. The lessons
   that tooling encodes, all of which apply here unchanged:
   - A language must **already exist on the submission** before import; a column Partner Center does not
     know is dropped silently. That is how CyrFlip's Urdu copy went missing on the first attempt.
   - Write UTF-8 **without** BOM and quote every field - the shape of the import that actually went
     through.
   - A listing counts as complete only with a description **and** at least one screenshot, so Block D is
     not optional decoration for the ten new locales.
   - Search terms are capped per language (memory: the 21-word limit that fails the import per-language).
   - `ReleaseNotes` ("What's new") stays en/ru/uk (decision D3); the other language files carry no
     `@@ReleaseNotes` block, and the tooling says so in a comment - it is a deliberate absence, not an
     omission waiting to be fixed.
   - Keep a `-FillNothing` round-trip switch: the output must be byte-identical to the export, which is
     the only cheap proof the writer is not corrupting the live listing.
2. **`msix/AppxManifest.xml`** `<Resources>`: one `<Resource Language="..."/>` per shipped language.
3. **winget:** one `SerZhyAle.DocHtmlTranslate.locale.<tag>.yaml` per language, generated from the same
   `listing/<lang>.txt` source. The default-locale manifest and `PackageIdentifier` are frozen anchors -
   adding locale manifests must not touch either. CRLF manifests, as winget requires.
4. **Chrome Web Store / Edge Add-ons:** `_locales` already auto-serves name and short description per
   browser language - that comes free with A4's 13 locale dirs. The long description and screenshot
   captions are **not** auto-localized; they are pasted per language in each dashboard.
   `extension/store/LISTING.md` grows a per-language section, generated from the same source as the Store
   copy so the two never diverge.

## Block D - screenshots for the release

Today: `tools/store/make-screenshot.ps1` converts a public-domain sample with the CLI and captures two
1366×768 PNGs with headless Edge. The captured chrome is in whatever language the dev machine's OS is - the
script has no language control at all, which is why the shipped shots are English.

1. **`-Language` parameter**, defaulting to all 13, driving `-ui-lang` on the CLI call. Each pass writes
   `<shot>-<locale>.png` using the Partner Center locale code, so the Store import folder can point each
   listing at its own file by relative path.
2. **Three frames per language**, not two: the reading view, the table of contents, and the GUI window.
   The GUI frame matters most for this feature - it is the surface where a translated interface is
   visible as an interface - and it is capturable headlessly because the GUI is a local page.
3. **RTL check built in:** for `ar` / `ur` the shot must show a mirrored chrome over unmirrored book text.
   Unlike CyrFlip we capture a browser page, not a `WS_EX_LAYOUTRTL` window, so CyrFlip's
   flip-the-PrintWindow-bitmap workaround is **not** needed here - do not port it.
4. **Extension screenshots** (`extension/store/screenshots/shot{1,2}-<lang>.png`) get the same treatment
   once A4 lands; currently en/ru only and made by hand.
5. Outputs land in the repo's gitignored `temp/` while iterating, never in a deep scratchpad path - a
   long path silently breaks OCR through MAX_PATH and fakes a quality bug (recorded in memory).

## Block E - guards, tests and the typography conflict

1. **Go:** every registered key has a non-empty translation in all 13 (or an explicit English fallback
   marker), and no user-visible literal is left outside the layer. CyrFlip's equivalent test also renders
   each form per language and fails on surviving Cyrillic; the Go analogue is to convert a fixture per
   language and assert no untranslated key survives in the output HTML.
2. **GUI:** all 13 `I18N` dictionaries carry an identical key set (a missing key is invisible at runtime -
   it falls back to English and looks fine).
3. **Extension:** `npm test` case asserting the same for `_locales/*/messages.json`, extending the rule
   `DOCS_SURFACES.md` already states for the current trio ("add a key to all three or none").
4. **`tests/ui_cli_parity_test.go`** covers `-ui-lang` (see A5) - it will go red the moment the flag lands
   without GUI exposure.
5. **The typography guard conflicts with this feature and must be amended first.**
   [`tests/typography_test.go`](../../tests/typography_test.go) fails any Go string literal or `_locales`
   message containing `—`, `…` or `...`. That rule encodes the author's *house style for Russian and
   English prose*. It is wrong as a global rule for translations: Chinese uses `——` as its dash and `……`
   as its ellipsis, German and French use `–`, and Arabic and Urdu use their own punctuation
   (`،` `؟` `۔`). Leaving the guard as-is guarantees either a red build or 13 mistranslated languages.
   Required change: scope the house-style assertion to the **en / ru / uk** values and allow
   script-appropriate punctuation elsewhere, with the exception list written down and tested.
   Do not weaken the guard globally - it exists because this drift already happened once
   ([`done/2026-07-17_output-typography.md`](done/2026-07-17_output-typography.md), ~42 violations swept).

## Edition parity checklist

| Edition | Status | Notes / rationale |
|---|---|---|
| CLI (`doc-html-translate`) | `[ ]` | A1, A2, A5 - splash + registration output + `-ui-lang` |
| GUI (`doc-html-ui`) | `[ ]` | A3 + A5 - 87 keys × 13, `<select>` switcher, RTL, font stack |
| MSIX Store app | `[ ]` | C2 manifest resources; inherits the GUI otherwise |
| Browser extension | `[ ]` | A4 - the largest gap: viewer UI is English-only today |
| Website / docs | `[ ]` | B - 10 new landing pages, hreflang, sitemap, README_RU/UK, DOCS_SURFACES rewrite |
| Installer (`setup.exe`) | `[ ]` | A6 - 8 official Inno languages; ar/hi/bn/ur/zh fall back to English there only (D2) |
| Store listings + screenshots | `[ ]` | C1/C3/C4 + D - the release-facing half of the ask |

## Shared invariants touched

Add to [`docs/PARITY.md`](../../docs/PARITY.md):

- **UI language set = 13**, one list, both editions: `internal/i18n` `Codes` ↔ `extension/_locales/*`.
  This becomes a checked invariant like the OCR catalog already is.
- **`<html lang>` is the content language, never the UI language**; UI chrome carries its own `lang` /
  `dir` (A2). Today's PARITY.md line "Reader features that are Go-only: .. Russian localization
  (`syslocale.IsRussian`)" becomes "UI localization, 13 languages, both editions" once A4 lands.
- **RTL applies to chrome only**, never to the converted document body.
- The **default-UI-language rule** (explicit flag → saved choice → OS → English) as a settings default,
  next to the existing default table.

## Cross-references (Go ↔ JS)

| Concern | Go | JS |
|---|---|---|
| Language set + endonyms | `internal/i18n/i18n.go` | `extension/_locales/*/` dir list |
| Reader chrome strings | `internal/htmlgen/navbar.go` | `extension/src/viewer.html` + `viewer.js` |
| RTL / direction | `i18n.Dir` | viewer `dir` attribute |
| UI language resolution | `internal/syslocale` + `-ui-lang` | `chrome.i18n.getUILanguage()` + options override |

## Decisions (settled 2026-07-28)

Six forks were open when this ticket was drafted. All six are closed here, by the owner's instruction to
take the correct option rather than to choose between them. Each is recorded with its reasoning, because a
decision without its reason gets re-litigated by the next reader.

**D1. Portuguese ships as one neutral translation, Brazilian where a channel forces a variant.**
The interface has no business carrying two Portuguese dictionaries: the differences that matter to a
converter's UI (`ficheiro` vs `arquivo`, `ecrã` vs `tela`) are a handful of nouns, and two dictionaries
would double the maintenance for them. So: one translation under the code `pt`, written in a form a reader
in either country accepts, using the more widely-read Brazilian term where the two genuinely diverge.
Chrome resolves `_locales/pt/` for both `pt-BR` and `pt-PT` browsers, so the extension needs one directory.
Where a channel demands a specific tag - Partner Center and Inno Setup both do - it gets the Brazilian one
(`pt-br`, `BrazilianPortuguese.isl`), because Brazil is roughly four fifths of the Portuguese-speaking
Windows population.

**D2. The five languages Inno Setup cannot speak fall back to English in the installer, and only there.**
Arabic, Hindi, Bengali, Urdu and Chinese have no official `.isl` in the Inno distribution; only third-party
translations exist. Vendoring those would mean shipping five files this repo cannot license cleanly, cannot
proofread, and cannot re-verify when Inno's message set changes between versions - and a *wrong*
installer is worse than an English one, because the installer is the moment a user decides whether to
trust the program. An English installer followed by an app that opens in Bengali is a small, honest seam.
The `setup.exe` channel is also the smallest of the four: MSIX, winget and portable users never meet it.

**D3. "What's new" stays en / ru / uk; everything else in a listing is translated into all 13.**
The description, the feature bullets and the screenshots are written once per product state and re-read by
every visitor - they carry the 13 languages. Release notes are rewritten at *every* release and read by
almost nobody: at 13 locales that is a recurring per-release cost that would either slow releases down or,
more likely, quietly rot into stale copy in ten languages. Partner Center is fine with empty
`ReleaseNotes` cells for a locale; those language files simply carry no `@@ReleaseNotes` block, which is a
deliberate absence and must be commented as one in the tooling so nobody "fixes" it later. This matches
CyrFlip, which arrived at the same split after living with it.

**D4. Per-language site pages are one landing page each - no FAQ, no partial guide.**
The ask was primitive documentation per language, and a half-translated guide is worse than a clearly
English one: it strands the reader mid-topic in a language they cannot continue in. Each of the ten new
pages is complete in itself (hero, use cases, formats, install, the "important" note) and links onward to
the full English guide. The full docs trio stays en / ru / uk.

**D5. No "beta" marker on the language list; the honesty goes in prose instead.**
Ten of the thirteen will be machine-translated and unproofread. Tagging them "beta" in a dropdown tells the
user nothing actionable - it does not say what might be wrong or what to do about it - while making the
product look unfinished in exactly those markets it is trying to enter. The correct place for the caveat is
the README (en/ru/uk), the guide and each per-language landing page: one plain sentence saying the
translation is machine-made and unproofread, with an address to send corrections to. That is a statement a
reader can act on. If a specific language later collects real complaints, that language gets fixed - which
is more useful than a permanent badge on all ten.

**D6. The extension gets an explicit UI-language override in its options, defaulting to the browser's.**
`chrome.i18n.getUILanguage()` is the right default and wrong as the only option: a user running an English
browser at work still reads books in their own language, and the language of the *browser chrome* is not
the language they want the *reader* in. The override is also what makes Block D reproducible - screenshots
in 13 languages cannot require 13 browser profiles. This mirrors the desktop side, where `-ui-lang` and the
GUI switcher already override the OS language, so the two editions behave the same way.

## Done criteria

- [ ] Every edition row above is `Done` or `Declined (reason)`.
- [ ] `internal/i18n` is the only place a user-visible Go string is chosen; `syslocale.IsRussian` is gone.
- [ ] All 13 languages render: GUI, converted page, extension viewer, installer, site landing.
- [ ] RTL verified for `ar` + `ur` on chrome, with document text unmirrored.
- [ ] Chrome still offers "Translate page" on a converted document in every UI language (A2 invariant),
      checked with `/verify-view`, not by inspection.
- [ ] `docs/PARITY.md` carries the new invariants; `DEV/DOCS_SURFACES.md` rewritten for 13 vs 3 languages.
- [ ] Typography guard amended and green (Block E5), `scripts/test.ps1` green, `npm test` green.
- [ ] Store listing CSV round-trips byte-identically with `-FillNothing`; every new locale exists on the
      submission **before** import.
- [ ] 13 × 3 app screenshots and the extension screenshots regenerated by script, not by hand.
- [ ] Changelog entry in `DEV/CHANGELOG.md`; machine-translation disclosure present in README (en/ru/uk)
      and on the site.

## Suggested sequencing

1. **E5 first** - amend the typography guard, otherwise every later commit is red.
2. **A1 + A2** - the Go layer and the converted page. This is the surface with the most user-time and the
   riskiest interaction (page-language detection), so it gets the earliest feedback.
3. **A3 + A5** - GUI and the flag; parity test keeps them honest.
4. **A4** - extension, the largest single block of new strings.
5. **B** - site, README, DOCS_SURFACES.
6. **D** - screenshot tooling (needs `-ui-lang` from step 3 and the extension UI from step 4).
7. **C** - listings last, because they consume the screenshots and the finished copy.
8. **A6** - installer; independent of everything above, any time.

This ordering is now realised as 9 numbered phases in
[`2026-07-28_thirteen-ui-languages/INDEX.md`](2026-07-28_thirteen-ui-languages/INDEX.md), which is the
executable source of truth; the installer (A6) lands there with the other packaging work in Phase 08.
Execute the phases, not this file.
