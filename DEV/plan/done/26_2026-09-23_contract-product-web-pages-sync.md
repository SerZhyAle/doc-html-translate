# The product site follows the portfolio page contracts

**Status:** Implemented - Direction A in the repo 2026-09-25 (commit `26605b9`); the rendered section 11 walk, the registry row, the dated exceptions and the B1-B14 dispositions closed 2026-09-28
**Priority:** 50
**Date:** 2026-09-23

> Contract sync ticket, both directions. Site + docs, every authored locale in one edit.
> Contracts: `PAGE-CONTENT` 1.1, `PAGE-STYLE` 1.1, `SITE-FAMILY-MAP` 1.1 (domain `product-web-pages/`,
> owner the sza.od.ua hub, all active). Checked and **not applicable**: `WAVE-PARTICLES` (read at 0.10,
> re-read at 0.12 - still no canvas on any page of this product). Pointers live in
> [`docs/contracts/`](../../../docs/contracts/).

> **Remote execution (2026-09-25):** the contract text this ticket needs was quoted in a "Contract snapshot"
> section, so every step not marked ⛔ could run in a cloud session from this repository alone; the snapshot
> and the reference-kit working copy were deleted when the ticket moved to done/. Steps marked **⛔ Local
> only** edit the shared contracts catalog (or another repository) and ran on the owner's machine, where the
> catalog is mounted - the catalog half closed 2026-09-28 (see the section at the end).

## What / why

The three contracts say what every product page says and in what order, the one visual system it is built
from (the kit `sza-kit.css`, byte-identical), and the family map every footer carries. They were extracted
from the hub on 2026-09-22; the registry lists "the eight product pages" as bound and not yet declared.
This product's site is one of them and has never been read against them.

The site is three design systems at once: `index.html` and the ten locale landings are on the kit, but a
kit copy **edited in place**; `extension.html` and the two privacy pages are on an older token scheme; the
three `docs*.html` pages were never migrated (light-only teal, other fonts). The contracts, in turn, model
one page in three languages per product - this site has 13 interface languages, locale sub-pages and
child pages (extension, docs, privacy, and the coming install-trust page), which the contracts do not
describe.

Found by reading the pages on 2026-09-23; nothing was checked in a rendered browser yet.

## Findings that drive the work

**A real cross-page bug, user-visible (should not wait for the rest):** every SZA page shares one origin,
so `localStorage` `sza-lang` is shared. `index.html` stores and understands `ua`; `extension.html` stores
`uk` and understands only `uk`. After a visitor picks Ukrainian on the extension page, the landing page
matches no hide rule and shows all three languages at once, with English metadata - and the reverse on the
extension page (`index.html` pre-paint and `applyLang`; `extension.html` switcher and CSS).

**`SITE-FAMILY-MAP`:**
- ~~the EN footer lacks StreamsPlayer and points OneClickRunner at its GitHub repo instead of its page~~
  (fixed in `index.html:141`, found 2026-09-25);
- the ten locale footers list five siblings and no heading;
- `extension.html`, the docs pages and the privacy pages carry no family grid;
- `extension-privacy.html` and its source `extension/store/PRIVACY.md` give `serzhyale@gmail.com` instead
  of the one contact `sza@ukr.net`.

**`PAGE-STYLE`:**
- `assets/sza-kit.css` differs from the reference kit (page-local rules and a `--wide` override were added
  into it) - the contract requires byte identity or a recorded exception;
- docs pages: other fonts, light only, no toggle, "UK" label, links instead of the persisted switcher;
- privacy pages: no theme toggle, no switcher, English only;
- the product mark is coloured (`#22368a`) and shown twice (header and hero); pre-kit variable names
  `--accent-purple` / `--accent-cyan` on the extension page;
- `index.html`: no expand/collapse-all for its numbered sections; the theme button's `aria-label` and
  `theme-color` are not updated; locale landings show a bare `✓` instead of "✓ Copied".

**`PAGE-CONTENT`:**
- H1 is the product name, not an outcome, and the hero repeats the mark;
- the get-it block sits after four cards and a demo, below the fold, with three Install buttons above it;
- real channels are omitted: the setup installer and Edge Add-ons are on no landing page;
- the docs pages link binaries on `tree/master` / `blob/master` of a repo that has only `main`, bypassing
  Releases (`docs.html` and its ru/uk twins);
- the docs pages restate a sibling product (install + features) instead of one contextual cross-link.

**Other:** three em dashes in `index.html` prose (house style); `hreflang` clusters on the child pages
point at the homepage; no test guards any site rule.

**`WAVE-PARTICLES`:** no canvas, `requestAnimationFrame` or `getContext` on any site page or in the GUI;
the only background is the kit's CSS blobs. The product is not a consumer and owes no row. Adoption would
be a separate opt-in once the contract ships its reference script.

### Re-verified 2026-09-25

Re-read against the catalog and the working tree on 2026-09-25, by reading only (no rendered browser):

- **Done:** pointer files `docs/contracts/PAGE-CONTENT.md`, `PAGE-STYLE.md`, `SITE-FAMILY-MAP.md` exist and
  are listed in `docs/contracts/README.md` (rows 22-24). **Drift:** the `PAGE-STYLE` pointer and its README
  row say 1.0 (catalog 1.1); the README does not record that `WAVE-PARTICLES` was read and does not apply;
  the pointer bodies describe this page ("Four compact workflow cards", "`index.html` uses
  `assets/sza-kit.css`") and read as conformance claims rather than as a summary of the contracts.
- **Done (catalog side), but overstated:** the registry carries a consumer row `PAGE-CONTENT`,
  `PAGE-STYLE`, `SITE-FAMILY-MAP` / "doc-html-translate website", implements `1.1 / 1.0 / 1.1`, dated
  2026-09-24: "Landing page `index.html` conforms to SZA design kit, no-flag language switcher, pre-paint
  theme resolver, no emoji, and full `SITE-FAMILY-MAP` 1.1 §2 family tools grid in the footer." It covers
  `index.html` only, carries no exceptions, and the kit half is not true (next bullet). The "eight product
  pages" placeholder row is still in the registry.
- **Kit drift still present:** `assets/sza-kit.css` SHA-256 `b726620f..5ce046ac9` vs the reference
  `72bd903e..0332593f` (full hash in the snapshot). The diff: `--wide:min(1760px,94vw)`, `.container` /
  `.site-header` width rules, `.brand-mark` and `.product-icon` in `#22368a`, hero / section-title /
  info-card / install-grid / `#use-cases` rules, a `max-width:760px` block, and the reference's `.get`
  block rules replaced.
- **Language bug still present:** `extension.html:58` pre-paint resolves `uk` and never maps a stored
  value, its CSS keys `html[data-lang="uk"]` (`:200-202`) and `applyLang` iterates `['ru','en','uk']`
  (`:600-611`); `index.html:50` maps only `?l=uk` to `ua`, not a stored `sza-lang` of `uk`.
- **Done in the EN footer:** `index.html:141` now carries a localized heading (RU "Другие инструменты SZA",
  EN "More tools by SZA", UA "Інші інструменти SZA"), all seven siblings with StreamsPlayer, OneClickRunner at
  `https://serzhyale.github.io/OneClickRunner/`, the hub, and `sza@ukr.net`. The first `SITE-FAMILY-MAP`
  finding above is therefore closed for `index.html`.
- **Still open:** the ten locale footers (`ar bn de es fr hi it pt ur zh/index.html`) list four siblings plus
  the hub and no heading; `extension.html`, `docs*.html`, `privacy.html`, `extension-privacy.html` carry no
  `tools-grid`; `extension-privacy.html` and `extension/store/PRIVACY.md` still contain
  `serzhyale@gmail.com`; each `docs*.html` still has two `tree/master` / `blob/master` links; no child page
  links `sza-kit.css`; `index.html` still names neither the setup installer nor Edge Add-ons; em dashes remain
  in `index.html:119`, `:131` and the JS title map at `:144`.
- **`.sza-canon.json` `site.pages`** still lists seven pages and none of the ten locale landings.
- **`WAVE-PARTICLES`:** still no `<canvas>`, `requestAnimationFrame` or `getContext` on any site page. The
  contract now carries `reference/wave-particles.js` (rung 2), so the "once it ships its reference script"
  condition for an opt-in is met; the product still is not in its `consumers` and owes no row.
- **Direction B, already raised by other products** (none filed by this product; see the snapshot):
  B2 by FastMediaSorter_Lite (`PROPOSAL-2026-09-24-get-position.md`); B3 by universal-agent-kit
  (`PROPOSAL-2026-09-23-documentation-method-start-and-actions.md` item 3); B8 and B9 by
  universal-agent-kit (`PROPOSAL-2026-09-23-universal-agent-kit-page-style.md` items 3 and 2); B13 by
  the same proposal item 7.

### Implemented 2026-09-25 (repository side of Direction A)

Owner's answers to the open questions, 2026-09-25: Q2 H1 = EN "Turn any book, document or comic into a local
web page" / RU "Любая книга, документ или комикс - в локальную веб-страницу" / UA "Будь-яка книжка, документ чи
комікс - у локальну вебсторінку"; Q4 locale landings keep the link list; Q5 privacy pages in RU / EN / UA; Q6 docs
restyled in place. Q3 by the standing preference: the "New: 13 languages" paragraph stays in the hero, after
what/for-whom. Q7 (store listings' own contact fields) is left to the next release flow.

- **1 Language value:** one space `ru|en|ua` on every page; each pre-paint maps a stored `uk` (and `?l=uk`) to
  `ua`; `assets/site.js` does the same. `extension.html` now writes `ua`.
- **2 Kit:** `assets/sza-kit.css` = the reference (SHA-256 `72bd903e..0332593f`, LF, `-text` in `.gitattributes`).
  Every former diff line lives in `assets/site.css`, linked after the kit; the `--wide` override stays there
  (exception to record - catalog, local only). The mark is monochrome (`currentColor`), `#22368a` is gone.
  Shared body scripts in `assets/site.js` (theme-color + `aria-label` on toggle, copy with `execCommand`
  fallback, `openFromHash`, expand/collapse-all, back-to-top, release tag).
- **3 Footer:** the headed full grid (7 siblings + hub) on all 17 pages, the ten locale headings in their own
  language; `sza@ukr.net` in `extension-privacy.html` and `extension/store/PRIVACY.md`.
- **4 Landing order** (`index.html` + ten locales): eyebrow -> outcome H1 -> tagline -> what/for-whom -> one proof
  strip -> `section.get#get` (Microsoft Store, GitHub Releases with the live tag, setup installer, Chrome Web
  Store, Edge Add-ons, winget copy box, three-step quickstart) -> scenarios -> details (Expand all / Collapse
  all). Mark once in the header; hero icon and the three Install buttons removed. Em dashes gone from prose and
  the JS title map.
- **5 Extension page:** kit tokens, header/footer/back-to-top, self-referencing `hreflang`, page JS replaced by
  `site.js`, quickstart leads with the stores instead of Load unpacked.
- **6 Docs pages:** kit fonts/tokens, light/dark, header with RU EN UA (`data-href` to the sibling page),
  binaries -> `/releases/latest`, the Fast Media Sorter section -> one contextual note.
- **7 Privacy pages:** kit, RU / EN / UA in-page (the English text word for word as before), self-referencing
  `hreflang`, footer grid. Not a generated page: `PRIVACY.md` is copied by hand.
- **8 Guard:** `tests/site_test.go` - kit hash, kit-then-site.css on every page, `site.pages` = served pages,
  footer URLs + contact, `ru|en|ua` value space, no branch links for binaries, author-page typography.
- `sitemap.xml`: child pages carry their own `hreflang` clusters; `.sza-canon.json` `site.pages` lists all 17;
  `DEV/DOCS_SURFACES.md` rows updated; pointer files rewritten as contract summaries, `PAGE-STYLE` 1.1, README
  records `WAVE-PARTICLES` as read and not applicable.

Evidence: `go test ./tests/ -count=1` -> `ok doc-html-translate/tests 156.550s`, exit 0 (the first run died
with the known GOARCH=386 out-of-memory; the rerun passed). `TestSite*` 8/8 PASS.

Left open, found while implementing:
- `extension/store/PRIVACY.md` and `extension-privacy.html` already disagreed in substance before this ticket
  (remote images blocked until loaded; OCR languages stored; "PDF opens" vs "document opens") - owner to pick,
  then sync both and move "last updated" (still 2026-08-11 although the contact changed).
- The ten locale pages' `<title>` / description still read "document converter .."; only the root's follow the H1.
- Rendered spot check 2026-09-25 (headless Chrome, iframes at 360 / 768 and a 1280 window; dark for index,
  extension, docs.ru, privacy, ar; light for index, privacy, ar). Found and fixed: the Copy button let a long
  winget command show through (now opaque; `.install-grid` single-column below 1024px); the ar/ur demo strip
  ran right-to-left against its arrows (`dir="ltr"`); the kit's `.demo` text was invisible in the light theme
  (B14). Not yet done: the full section 11 checklist per page (touch targets, focus, reduced motion, 768 light,
  extension/docs light), so done criterion 6 stays open.

## Closed 2026-09-28 - the rendered walk and the catalog half

**The leftover items of 2026-09-25 were already resolved in commit `26605b9`:** the two extension privacy
surfaces agree (remote images blocked until loaded, downloaded OCR languages, contact `sza@ukr.net`, both
"last updated 2026-09-25"); the ten locale `<title>`s and descriptions follow the outcome H1 in their own
languages.

**Rendered `PAGE-STYLE` section 11 walk, 2026-09-28** (in-app Chromium; 360 / 768 / 1280 px, dark and light,
every page reloaded from its own pre-paint resolver): `index.html`, `extension.html`, `docs.html`,
`docs.ru.html`, `privacy.html`, `extension-privacy.html`, `install-trust.html` and `ar/index.html` (RTL) -
31 rendered combinations. Held on every page at every width and theme: no horizontal overflow; Outfit and
Plus Jakarta Sans resolve; `theme-color` matches the theme token both ways and follows the toggle;
`aria-label` on the toggle follows; the RU EN UA switcher keys `ua`, writes `lang="uk"`, persists and
localizes title and description; the footer grid carries all seven siblings plus the hub with the contact
`sza@ukr.net` and no stale address; no emoji; no branch links; the copy button works and shows the
localized done word; expand/collapse all and back-to-top work; the get-it block is above the fold at 768
and 1280 (at 360 it starts 73 px below the 740 px fold - recorded, the get-position proposal covers it);
the ar landing keeps `dir="rtl"` with no overflow and its own footer heading.

Found and recorded, nothing page-breaking:

- The reference kit hides `.site-header .btn` below 480 px and raises `.btn` / `.seg button` /
  `.theme-btn` to 44 px only under `pointer:coarse` - the kit's own rules, byte-identical reference
  (CyrFlip B4 / universal-agent-kit ask the reference to change it). The page layer now raises
  `.copybox .copy` and `.to-top` to 44 px for every pointer (`assets/site.css`), the two the kit misses
  entirely; `.tools-grid a` measures 68-90 px.
- The copy confirmation is the localized word without a dingbat ("Copied" / "Скопировано" /
  "Скопійовано" / "تم النسخ"): a `✓` prefix was tried and reverted the same day - `ICON-SET` 0.15
  `action.copy` rules the dingbat out for this product (a consumer since ticket 24). The `✓ Copied`
  wording of section 4.7/11 is new ask 7 of the proposal below, co-signed with universal-agent-kit item 4.

**Catalog half (⛔ Local only), done 2026-09-28:**

- The registry row of 2026-09-24 is rewritten from the finished site (reads `1.1 / 1.1 / 1.1`, declares
  this product out of the "eight product pages" placeholder, keeps the old row after "The row this
  replaces follows", quotes the walk).
- Three dated exceptions (until 2027-03-31): the `--wide` override in the page layer; the 13-locale shape
  with the per-language docs pages; the child-page order without a get-it block (`extension.html` has its
  own install block and stands outside it).
- `PROPOSAL-2026-09-28-doc-html-translate-page-sync.md`, filed beside the contracts in the catalog's
  `product-web-pages/` folder (the catalog is not a git repository, so the file itself is the record). New
  asks: the kit-untouched-plus-page-layer mechanism (B4), the child-page role (B6, extending CyrFlip B6),
  per-language documentation pages (B7), the two-public-entry-points map convention (B10), paid/metered
  paths under the safe-path rule (B11), the reference kit's `.demo` light-theme text (B14), and the
  `PAGE-STYLE` 4.7 vs `ICON-SET` copy-confirmation conflict (B13's sibling). Seconds: CyrFlip B1/B2/B3/B4/B6
  (our B5, B9, B1, the 44 px defect, B6), universal-agent-kit items 2/3/4/7 (our B9, B8, B13 + the
  confirmation, B13), `get-position` (our B2), `documentation-method` item 3 (our B3).

**B12 (the hub card), checked on the hub's working copy `P:/WEB/sites.google.comsiteszaodua`:** the card's
EN / RU / UA dictionaries already carry the corrected copy (browser extension, read/translate outcome), but
the static no-JS markup of card 4 still reads the old "Converter for Windows: EPUB, PDF, MOBI, AZW3, FB2,
RTF, TXT, and Markdown" text. The corrected copy is offered in the proposal's closing section; the hub
repository is not edited from this ticket.

**B-item register (criterion 8):** B1 -> seconded CyrFlip B3; B2 -> seconded `get-position` + CyrFlip B3;
B3 -> seconded `documentation-method` item 3 + dated exception; B4 -> new ask 1; B5 -> seconded CyrFlip B1 +
dated exception; B6 -> new ask 2 (extends CyrFlip B6) + dated exception; B7 -> new ask 3 + dated exception;
B8 -> seconded universal-agent-kit item 3; B9 -> seconded CyrFlip B2 + universal-agent-kit item 2; B10 ->
new ask 4; B11 -> new ask 5; B12 -> corrected copy offered (above); B13 -> seconded universal-agent-kit
item 7, the copy-confirmation half is new ask 7; B14 -> new ask 6, site default already in
`assets/site.css`.

## Direction A - the site conforms

Phase order matters: the bug first, the kit before the pages that depend on it.

1. **Language value** - one value space `ru|en|ua` on every page, and every reader maps a stored `uk` to
   `ua` (forward tolerance). Small enough for `/fix` ahead of the rest.
2. **Kit** - `assets/sza-kit.css` byte-identical to the reference; every page-local rule moves into a
   separate page stylesheet. Or, if the width override stays, a recorded exception.
   Default to implement (no catalog needed): copy
   `sza-kit.reference-2026-09-25.css`, a working copy of the reference kit that lived beside the ticket and
   was deleted with it
   over `assets/sza-kit.css` byte for byte, check its SHA-256 against the snapshot, and move every diff
   line (the `--wide` override included) into a page stylesheet linked after the kit - the shape
   universal-agent-kit ships. **⛔ Local only - changes the contract catalog.** Recording the width
   override that the page stylesheet keeps as a dated registry exception.
3. **Family footer** - the full grid (all siblings, the hub, the heading, localized) on every page,
   locale pages included; contact `sza@ukr.net` everywhere including the extension privacy source.
   `index.html:141` is the model (done, see "Re-verified 2026-09-25"). The contract fixes no heading text:
   RU/EN/UA reuse the `index.html` wording; the ten locale pages need the heading in their own language.
4. **Landing order** (`index.html` + ten locale pages) - outcome H1 in EN/RU/UA (wording from the owner),
   mark once in the header, get-it block right after what/for-whom with every real channel (Store, GitHub
   Releases, installer, winget copy box, Chrome Web Store, Edge Add-ons) and a 2-3 step quickstart, one
   neutral install link above it.
5. **Extension page** - kit tokens and names, mark once, footer grid, back-to-top, self-referencing
   `hreflang`.
6. **Docs pages** - rebuilt on the kit (fonts, tokens, light/dark, pre-paint, header with RU EN UA);
   binary links -> `/releases/latest`; the sibling section -> one contextual link via its map URL.
7. **Privacy pages** - kit, theme toggle, footer grid; locale decision per open question 5.
8. **Guard** - a repo test over the site pages: family-map URLs, contact string, kit hash, no em dash
   outside a `<title>`. The expected kit hash is the one in the snapshot; the test must hold the hash
   itself, not read the working copy under `DEV/plan/`, which is deleted with this ticket.
9. The install-trust page of [`27_2026-09-22_install-trust-page`](../27_2026-09-22_install-trust-page.md) is built
   to the same child-page rules as 5-7.

## Direction B - what the contracts need from this product

**⛔ Local only - changes the contract catalog.** Every item below is filed as
`PROPOSAL-2026-09-23-<topic>.md` in the catalog's `product-web-pages/` folder, never as an edit (B12 instead
changes the hub repository). None is blocking for Direction A: each site step states its default. Where
another product already filed the same point (see "Re-verified 2026-09-25"), co-sign that proposal instead
of opening a second one.

- **B1** `PAGE-STYLE` §3.2 / §11.5 say "H1 - app name"; §4.1a and `PAGE-CONTENT` L3 say the hero does not
  repeat the name and the H1 is the outcome. One rule, please (correction).
- **B2** `PAGE-STYLE` §3.4 vs `PAGE-CONTENT` O4/O5: get-it "immediately after what/for whom" vs proof before
  get-started. Clarify that a one-line proof may sit between while get-it stays above the fold.
- **B3** `--wide: 1100px` vs "full width" (`PAGE-CONTENT` L1): change the token, or give an override hook.
- **B4** kit extension mechanism: state "kit untouched + a page stylesheet" explicitly, so byte identity
  and page-local rules can coexist.
- **B5** more than three languages: RU/EN/UA stay the switcher; further locales are separate pages with
  the same footer and theme, linked from a secondary list and `hreflang` (MINOR).
- **B6** product child pages (extension, docs, privacy, install-trust): a sub-page role saying which
  checklist items bind (kit, theme, footer, contact: yes; get-it block: no).
- **B7** separate per-language documentation pages, today allowed only for the "Big SEO app" role.
- **B8** the `?l=` crawlable language override of `index.html` as an optional pre-paint feature.
- **B9** the `sza-lang` value space is a cross-product same-origin wire: write it down (`ru|en|ua`) and
  require readers to map `uk` -> `ua`.
- **B10** one product with two public entry points (landing + extension page): a map convention for it.
- **B11** paid/metered paths belong under "the costly path is never first" (this site already puts the
  free path first).
- **B12** **⛔ Local only - changes the hub repository.** Hub side: this product's card on the hub is stale (no comics, images, OCR, extension). Offer
  corrected copy to the hub owner.
- **B13** the `PAGE-STYLE` §9 glyph list conflicts with `ICON-SET` - joint proposal, see
  [`24_2026-09-23_contract-iconography-sync`](24_2026-09-23_contract-iconography-sync.md) B9.
  Site default until it is answered: the `PAGE-STYLE` §9 kit glyphs (`◐ ⤓ → ▸`) as the kit draws them, each
  with a text label or `aria-label`; a swap to `ICON-SET` ids follows the answer, not this ticket.

- **B14** (found 2026-09-25) reference kit: `.demo code{color:var(--text)}` sits on `--code-bg`, which stays
  dark in the light theme, so the demo strip's text disappears there. Ask: `.demo` text in `--code-ink`.
  Site default until answered: the override in `assets/site.css`.

## Done criteria

- [x] Pointer files `docs/contracts/PAGE-CONTENT.md`, `PAGE-STYLE.md`, `SITE-FAMILY-MAP.md` exist and are
      listed in [`docs/contracts/README.md`](../../../docs/contracts/README.md), which also records that
      `WAVE-PARTICLES` was read and does not apply. (Files and listing done; the `PAGE-STYLE` version,
      the `WAVE-PARTICLES` note and the pointer bodies fixed 2026-09-25.)
- [x] **⛔ Local only - changes the contract catalog.** Registry: this product's consumer row for the three ids (reads 1.1 / 1.1 / 1.1) replaces the
      "eight product pages" placeholder for this product; every remaining deviation is a dated exception.
      (Done 2026-09-28: the overstated row of 2026-09-24 rewritten from the finished site, the old row kept
      after "The row this replaces follows"; three dated exceptions until 2027-03-31.)
- [x] Choosing a language on any page of this site shows exactly one language on every other page.
- [x] `cmp` of the vendored kit against the reference is clean, or its exception is recorded. (Remote:
      `sha256sum assets/sza-kit.css` equals the snapshot hash `72bd903e..332593f`, re-verified 2026-09-28;
      the `--wide` page-layer override is a dated exception, the record is ⛔ Local only - done 2026-09-28.)
- [x] The `SITE-FAMILY-MAP` §5 URL check passes for every footer on every page; one contact everywhere.
      (2026-09-25: all nine URLs 200; footers and contact guarded by `tests/site_test.go`.)
- [x] The `PAGE-STYLE` §11 checklist is run on a rendered page at 360 / 768 / 1280 px for the landing,
      extension, docs and privacy pages, light and dark, and the result is recorded. (2026-09-28, see the
      closing section: 31 rendered combinations over eight pages including the RTL landing and both privacy
      pages, dark and light at every width; results in the registry row of the same date.)
- [x] No docs link points at a branch file for a binary.
- [x] **⛔ Local only - changes the contract catalog** (B12: **the hub repository**). B1-B12 filed or
      withdrawn in writing here. (Done 2026-09-28: the B-item register in the closing section -
      `PROPOSAL-2026-09-28-doc-html-translate-page-sync.md` in the catalog for the new asks, seconds
      elsewhere; B12's corrected copy offered in the proposal, the hub not edited.)
- [x] `.sza-canon.json` `site.pages` lists every page this site serves, locale landings included. (18 pages.)
- [x] Every surface changed in every authored locale in one edit.

## Open questions (all answered 2026-09-25 - the owner's wording and decisions are quoted in the "Implemented" section above)

1. Vendor the kit byte-identical and add a page stylesheet, or keep a standing exception for the width?
   (Either way the width override is a deviation from `PAGE-STYLE` §5 until B3 is answered; writing it down
   is **⛔ Local only - changes the contract catalog.**)
2. The outcome H1 in EN, RU and UA - the contract forbids inventing it; owner's wording.
3. The standing preference that a new feature leads the hero vs `PAGE-CONTENT`'s "no promotion" and
   outcome-first hero: which wins for the "New: 13 languages" paragraph?
4. Locale landings: add the RU/EN/UA switcher, keep a link list, or wait for B5?
5. Privacy pages: RU/UA versions, or English only (they are store-form URLs)?
6. Docs pages: restyle in place, or fold into the landing's numbered sections?
7. Store listings' own contact fields may also carry the old address - part of this work, or of the next
   release flow?

