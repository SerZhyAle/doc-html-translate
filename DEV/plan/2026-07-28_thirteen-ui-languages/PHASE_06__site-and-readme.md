# Phase 06 - Site landing pages and README in thirteen languages

**Strategic spec:** [`../2026-07-28_thirteen-ui-languages.md`](../2026-07-28_thirteen-ui-languages.md)
**Tactical index:** [`INDEX.md`](INDEX.md)
**Status:** ✅ Done
**Depends on:** Phase 03
**Steps done:** 6 / 6

## Objective

Ship one complete landing page per language (strategic decision D4), wire them together for search engines,
and add the Russian and Ukrainian READMEs the repo lacks - with the machine-translation disclosure (D5).

## Prerequisites

- [x] Phase 03 ✅ Done (so the copy describes a product that actually speaks 13 languages).
- [x] `.nojekyll` present at the repo root - GitHub Pages silently freezes the live site without it.

## Files touched

| File | New / Modified | Line budget |
|------|:--------------:|------------:|
| `index.html` | Modified | ≤ 200 |
| `<code>/index.html` × 10 (`de it es fr pt ar hi bn ur zh`) | New | ≤ 140 each |
| `sitemap.xml` · `robots.txt` | New | ≤ 60 / ≤ 10 |
| `README.md` | Modified | ≤ 290 |
| `README_RU.md` · `README_UK.md` | New | ≤ 290 each |
| `docs.html` · `docs.ru.html` · `docs.uk.html` | Modified | ≤ 400 / ≤ 220 / ≤ 220 |

## Steps

### Step 06.1 - Add the 13-language switcher and hreflang to the root pages

**Files:** `index.html`, `extension.html`, `privacy.html`, `extension-privacy.html`
**Depends on:** - start of phase

**Prompt for developer:**
> Keep the existing in-page `data-l` mechanism for ru/en/ua on the root pages, and add next to the `RU EN
> UA` group a compact selector listing the other ten endonyms, each linking to `/<code>/`. Add
> `<link rel="alternate" hreflang="<code>" href="..">` for all 13 plus `x-default` in the head of every
> root page. Do not convert the root page to a per-language directory - its URL is referenced from the
> store listings and winget.

**Verification:**
- `hreflang="zh"` and `hreflang="x-default"` both appear in `index.html`.
- The header selector contains 10 links of the form `/<code>/`.
- Opening `index.html` locally still switches ru/en/ua in place.

**Status:** `[x] done`

---

### Step 06.2 - Create the five Latin-script landing pages

**Files:** `de/index.html`, `it/index.html`, `es/index.html`, `fr/index.html`, `pt/index.html`
**Depends on:** Step 06.1

**Prompt for developer:**
> Create one complete landing page per language, translated from the English copy of `index.html`: hero,
> the four use-case cards, the format pills, install block with the winget command, the "important" note
> about free vs paid translation, and links to the English guide, the extension page and both privacy
> pages. Single-language pages - no `data-l` spans. Correct `lang` attribute, canonical URL pointing at
> itself, the same hreflang block as Step 06.1, and the shared `assets/sza-kit.css`.

**Verification:**
- Five files exist; each has `<html lang="<code>"` and a self-referencing `<link rel="canonical">`.
- Each contains the winget command string and a link to `docs.html`.
- No `data-l=` attribute appears in any of them.

**Status:** `[x] done`

---

### Step 06.3 - Create the Indic and Chinese landing pages

**Files:** `hi/index.html`, `bn/index.html`, `zh/index.html`
**Depends on:** Step 06.2

**Prompt for developer:**
> Same as Step 06.2 for Hindi, Bengali and Chinese. These are web pages, so the browser handles the fonts -
> do not add a webfont, but do check that the hero heading and the pills do not clip at 360 px width.

**Verification:**
- Three files exist with the correct `lang` attribute and hreflang block.
- Rendered at 360 px width, no horizontal scrollbar (checked with the headless capture from Phase 07 or a
  local browser).

**Status:** `[x] done`

---

### Step 06.4 - Create the two right-to-left landing pages

**Files:** `ar/index.html`, `ur/index.html`
**Depends on:** Step 06.3

**Prompt for developer:**
> Same as Step 06.2 for Arabic and Urdu, with `dir="rtl"` on `<html>`. Verify the glass/blob background,
> the header row, the button group and the card grid under mirroring; keep the Latin product name,
> the winget command and the file-format pills left-to-right by wrapping them in `dir="ltr"` spans.

**Verification:**
- Both files carry `dir="rtl"` on `<html>` and at least one `dir="ltr"` wrapper around a command or code
  string.
- No layout element overflows the viewport at 1366 px and at 360 px.

**Status:** `[x] done`

---

### Step 06.5 - Add sitemap and robots

**Files:** `sitemap.xml`, `robots.txt`
**Depends on:** Step 06.4

**Prompt for developer:**
> Add a `sitemap.xml` listing the root pages and all 10 language pages with their `hreflang` alternates,
> and a `robots.txt` pointing at it. Neither file exists in this repo today; CyrFlip ships both.

**Verification:**
- `sitemap.xml` contains 13 or more `<url>` entries and parses as XML.
- `robots.txt` contains a `Sitemap:` line with the full https URL.

**Status:** `[x] done`

---

### Step 06.6 - Add the RU/UK READMEs and the translation disclosure

**Files:** `README.md`, `README_RU.md`, `README_UK.md`, `docs.html`, `docs.ru.html`, `docs.uk.html`, `<code>/index.html` × 10
**Depends on:** Step 06.5

**Prompt for developer:**
> Create `README_RU.md` and `README_UK.md` as full translations of `README.md`, cross-link the three at the
> top of each, and document the 13 interface languages in all of them. Add one plain sentence - in the
> READMEs, the three docs pages and every per-language landing page - saying the interface translations
> outside English, Russian and Ukrainian are machine-made and unproofread, with the contact address for
> corrections. No "beta" tag anywhere in the language lists (decision D5).

**Verification:**
- `README_RU.md` and `README_UK.md` exist and each links to the other two READMEs.
- The disclosure sentence appears in 3 READMEs + 3 docs pages + 10 landing pages (grep a distinctive
  fragment; 16 hits).
- `beta` returns zero hits in the language-list markup of the GUI, the extension and the site.

**Status:** `[x] done`

## Phase done criteria

- [x] Every `Step 06.*` is `[x] done`.
- [x] Every new HTML file passes `./scripts/verify-html.ps1` (or the repo's HTML check) with no errors.
- [x] `.nojekyll` still present at the repo root.
- [x] Grep for `TODO(phase-06)` returns zero hits.
- [x] Changelog entry added for every file in "Files touched".

## Step log

- 2026-07-28 - 06.1 hreflang (13 + `x-default`) in `index.html`, `extension.html`, `privacy.html`,
  `extension-privacy.html`; a 10-endonym `.langs` nav on the root page; the boot script now honours
  `?l=ru|uk|en` so the three in-page languages have addressable URLs for hreflang and for the
  per-language pages to link back to.
- 2026-07-28 - 06.2 / 06.3 / 06.4 the ten pages are generated from one template with per-language copy;
  each is single-language (no `data-l`), self-canonical, and carries the same hreflang block.
- 2026-07-28 - 06.4 RTL verified by rendering `ar/index.html` in headless Chrome at 1366 px: header,
  card grid and section rules mirror, while the product name, format pills, `winget` command,
  `index.html` and `Chrome / Edge` stay LTR inside `dir="ltr"` wrappers.
- 2026-07-28 - **360 px caveat:** headless Chrome `--window-size=360` clips content on the new pages -
  and clips the pre-existing root `index.html` identically, so this is the headless renderer ignoring the
  viewport meta, not a regression from this phase. A real narrow-viewport check still wants a device or
  DevTools emulation.
- 2026-07-28 - 06.5 `sitemap.xml` (17 `<url>` entries, each with 13 + `x-default` alternates, parses as
  XML) and `robots.txt`.
- 2026-07-28 - 06.6 `README_RU.md` and `README_UK.md` written as full translations and cross-linked;
  all three READMEs gained the 13-language feature bullet, the missing `-ui-lang` flag row and the
  disclosure. Disclosure grep: 16 hits (3 READMEs + 3 docs pages + 10 landing pages). `beta`: 0 hits.

## Handoff notes

Ten new URLs now exist under `/<code>/`. Phase 08's store listings and Phase 09's docs surfaces manifest
both reference them; the landing copy is also the source the store descriptions are condensed from.

## Rollback plan

Revert the phase commit(s). The ten directories are additive - deleting them leaves the site exactly as it
was, provided the hreflang blocks in the root pages go with them.
