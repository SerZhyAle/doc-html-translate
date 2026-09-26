# The product site follows the portfolio page contracts

**Status:** In Progress
**Priority:** 50
**Date:** 2026-09-23

> Contract sync ticket, both directions. Site + docs, every authored locale in one edit.
> Contracts: `PAGE-CONTENT` 1.1, `PAGE-STYLE` 1.0, `SITE-FAMILY-MAP` 1.1 (domain `product-web-pages/`,
> owner the sza.od.ua hub, all active). Checked and **not applicable**: `WAVE-PARTICLES` 0.10.
> `PAGE-STYLE` is 1.1 in the catalog since 2026-09-24 (additive: StreamsPlayer joins the App - medium role).
> Pointers now exist in [`docs/contracts/`](../../docs/contracts/) (see "Re-verified 2026-09-25").

> **Remote execution (2026-09-25):** the contract text this ticket needs is quoted in "Contract snapshot" below, so every step not marked ⛔ runs in a cloud session from this repository alone. Steps marked **⛔ Local only** edit the shared contracts catalog (or another repository) and can run only on the owner's machine, where the catalog is mounted.

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

## Direction A - the site conforms

Phase order matters: the bug first, the kit before the pages that depend on it.

1. **Language value** - one value space `ru|en|ua` on every page, and every reader maps a stored `uk` to
   `ua` (forward tolerance). Small enough for `/fix` ahead of the rest.
2. **Kit** - `assets/sza-kit.css` byte-identical to the reference; every page-local rule moves into a
   separate page stylesheet. Or, if the width override stays, a recorded exception.
   Default to implement (no catalog needed): copy
   [`sza-kit.reference-2026-09-25.css`](26_2026-09-23_contract-product-web-pages-sync/sza-kit.reference-2026-09-25.css)
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
9. The install-trust page of [`27_2026-09-22_install-trust-page`](27_2026-09-22_install-trust-page.md) is built
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
  [`24_2026-09-23_contract-iconography-sync`](done/24_2026-09-23_contract-iconography-sync.md) B9.
  Site default until it is answered: the `PAGE-STYLE` §9 kit glyphs (`◐ ⤓ → ▸`) as the kit draws them, each
  with a text label or `aria-label`; a swap to `ICON-SET` ids follows the answer, not this ticket.

- **B14** (found 2026-09-25) reference kit: `.demo code{color:var(--text)}` sits on `--code-bg`, which stays
  dark in the light theme, so the demo strip's text disappears there. Ask: `.demo` text in `--code-ink`.
  Site default until answered: the override in `assets/site.css`.

## Done criteria

- [x] Pointer files `docs/contracts/PAGE-CONTENT.md`, `PAGE-STYLE.md`, `SITE-FAMILY-MAP.md` exist and are
      listed in [`docs/contracts/README.md`](../../docs/contracts/README.md), which also records that
      `WAVE-PARTICLES` was read and does not apply. (Files and listing done; the `PAGE-STYLE` version,
      the `WAVE-PARTICLES` note and the pointer bodies fixed 2026-09-25.)
- [ ] **⛔ Local only - changes the contract catalog.** Registry: this product's consumer row for the three ids (reads 1.1 / 1.0 / 1.1) replaces the
      "eight product pages" placeholder for this product; every remaining deviation is a dated exception.
      (A row dated 2026-09-24 exists but overstates conformance and covers `index.html` only - correct it
      against the finished site, and read 1.1 for `PAGE-STYLE` once the pages are checked against it.)
- [x] Choosing a language on any page of this site shows exactly one language on every other page.
- [x] `cmp` of the vendored kit against the reference is clean, or its exception is recorded. (Remote:
      `sha256sum assets/sza-kit.css` equals the snapshot hash; recording an exception is
      **⛔ Local only - changes the contract catalog.**)
- [x] The `SITE-FAMILY-MAP` §5 URL check passes for every footer on every page; one contact everywhere.
      (2026-09-25: all nine URLs 200; footers and contact guarded by `tests/site_test.go`.)
- [ ] The `PAGE-STYLE` §11 checklist is run on a rendered page at 360 / 768 / 1280 px for the landing,
      extension, docs and privacy pages, light and dark, and the result is recorded.
- [x] No docs link points at a branch file for a binary.
- [ ] **⛔ Local only - changes the contract catalog** (B12: **the hub repository**). B1-B12 filed or
      withdrawn in writing here.
- [x] `.sza-canon.json` `site.pages` lists every page this site serves, locale landings included.
- [x] Every surface changed in every authored locale in one edit.

## Open questions

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

## Contract snapshot (2026-09-25)

Quoted from the catalog on 2026-09-25 - SITE-FAMILY-MAP 1.1, PAGE-STYLE 1.1, PAGE-CONTENT 1.1, the domain README, the reference kit `sza-kit.css`, WAVE-PARTICLES 0.10 (header only). A working copy for executing this ticket without the catalog, not a second source: the catalog stays authoritative, and this section is deleted when the ticket moves to done/.

### The reference kit (working copy)

[`26_2026-09-23_contract-product-web-pages-sync/sza-kit.reference-2026-09-25.css`](26_2026-09-23_contract-product-web-pages-sync/sza-kit.reference-2026-09-25.css)
is a byte-for-byte copy of the shared contracts catalog, `product-web-pages/reference/sza-kit.css`, taken
2026-09-25: 13037 bytes, LF line endings, SHA-256

```
72bd903e7edd4d883106eb296c50b64a6e11731125fab89017320b250332593f
```

The same hash is recorded for the hub's and universal-agent-kit's byte-identical copies in the registry.
The vendored `assets/sza-kit.css` measured on 2026-09-25:
`b726620f697b7f6ed3f5b634fd5256098486e1a5eb26952b7d5ff0b5ce046ac9` (differs). Make it conform with
`cp <working copy> assets/sza-kit.css` and check `sha256sum assets/sza-kit.css` (or `cmp`). Keep LF: on a
Windows checkout with `core.autocrlf=true` a CRLF working copy changes the hash, so verify the committed
blob too (`git show HEAD:assets/sza-kit.css | sha256sum`). The tokens `PAGE-STYLE` §5 refers to are the
`:root` / `[data-theme]` blocks at the top of that file. The copy is a working copy for this ticket only
and is deleted together with the ticket folder when the ticket moves to done/.

### `SITE-FAMILY-MAP` 1.1 - the shared contracts catalog, `product-web-pages/SITE-FAMILY-MAP.md`

> ## 2. The map
>
> Each site lists **all the others** in its footer grid; it omits itself, or marks itself as current if it
> shows itself at all.
>
> | Tool | Type | URL |
> | --- | --- | --- |
> | FastMediaSorter v2 | Android media sorter | https://serzhyale.github.io/FastMediaSorter_mob_v2/ |
> | Fast Media Sorter for Windows | Windows media sorter | https://serzhyale.github.io/FastMediaSorter_Lite/ |
> | CyrFlip | Windows layout fixer | https://serzhyale.github.io/CyrFlip/ |
> | doc-html-translate | Windows ebook converter | https://serzhyale.github.io/doc-html-translate/ |
> | FileDO | Windows storage CLI | https://serzhyale.github.io/FileDO/ |
> | StreamsPlayer | Windows stream player | https://serzhyale.github.io/StreamsPlayer/ |
> | OneClickRunner | Windows tray launcher | https://serzhyale.github.io/OneClickRunner/ |
> | Universal Agent Kit | AI-dev methodology | https://serzhyale.github.io/universal-agent-kit/ |
> | SZA (hub) | Portfolio | https://sza.od.ua |
>
> ## 3. The rules
>
> 1. **The footer grid carries every sibling tool in this table.** A product that is in the table is in
>    every other product's footer. Adding a row to this table is work for every page, and that is the
>    cost the table exists to make visible.
> 2. **The hub is the exception and carries no tools grid.** Its project cards already are the full tool
>    index, so the grid would only repeat them; the hub footer is copyright plus the contact line.
> 3. **One contact, everywhere the same**: email **sza@ukr.net**, GitHub **SerZhyAle**. LinkedIn and the
>    phone **+356 9957 6364** are hub-only, and the phone is in a copy box. No page states a different
>    primary address, in any surface it owns - the page, its README, or a mirrored render of either.
> 4. **Contextual cross-links are body copy, not footer duplication.** A tool is named inside the text
>    only where it genuinely covers what this one does not: FMS-Lite and FMS-mob as a desktop/mobile
>    pair, FMS-Lite with doc-html-translate as companions, FileDO named by FMS for fake-capacity and
>    duplicate checks.
> 5. **A URL enters this table only once it answers.** A planned page is a GitHub repository link in the
>    footer, not a row here, until the page is live.
>
> ## 4. Compatibility
>
> [..] Adding a tool is additive:
> a page that has not caught up shows a shorter grid and nothing is wrong on it. Removing or repointing a
> tool is a MAJOR, because every other page then links somewhere that is gone.
>
> ## 5. Conformance
>
> Read off the rendered footer, and checkable from outside the implementation:
>
> - every row of section 2 but the product's own appears in its footer grid (rule 2 exempts the hub);
> - the contact strings in section 3 rule 3 are the ones the page and its README show;
> - each URL answers. The check is one command:
>
> ```
> for u in <the section 2 URLs>; do curl -s -o /dev/null -w "%{http_code} $u\n" -L "$u"; done
> ```
>
> ## 6. Out of scope
>
> The visual shape of the footer grid - that is `PAGE-STYLE` section 4.11. [..]

The contract fixes no footer heading text; `PAGE-STYLE` §4.11 names it "More tools by SZA" in English only.
This site's RU/EN/UA wording (not contract text) is in `index.html:141`: "Другие инструменты SZA" /
"More tools by SZA" / "Інші інструменти SZA".

### `PAGE-STYLE` 1.1 - the shared contracts catalog, `product-web-pages/PAGE-STYLE.md`

Section numbering is the document's own. The 1.1 amendment changed only §2 (StreamsPlayer added to App -
medium); the §2 row for this product reads:

> | **App — medium** | doc-html-translate, FastMediaSorter_Lite, FileDO, StreamsPlayer | Header, hero, **distribution block**, feature sections, copy boxes, footer. |

> 2. Add the font link in `<head>`:
>    ```html
>    <link rel="preconnect" href="https://fonts.googleapis.com">
>    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
>    <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@400;500;600;700;800&family=Plus+Jakarta+Sans:wght@300;400;500;600;700&display=swap" rel="stylesheet">
>    ```
> 3. Add the pre-paint script (Section 7) **before** the stylesheet so theme/language never flash.

> ## 3. Page order (the contract)
>
> **App sites — above the fold, in this exact order:**
>
> 1. **Sticky header** — brand · language (RU EN UA) · theme · primary Download.
> 2. **H1** — app name. One-line **tagline**.
> 3. **"What it is & who it's for"** — 1–2 sentences. Plain. (`.get .whatfor`)
> 4. **Get it block** (`.get`) — *immediately after #3, still above the fold:*
>    - **Official channels** that actually exist for this app: Microsoft Store / winget / Google Play / etc.
>    - **GitHub Releases** download (always present; dynamic latest where possible — Section 4.10).
>    - **Quickstart** — 2–3 numbered steps: install → run → first useful action.
> 5. Then: features, details, screenshots, FAQ, links.
> 6. **Footer** — "More tools by SZA" grid + contact.
>
> **Hub & Docs sites:** header → hero → content (grid / numbered sections) → footer. No distribution block on the Hub.

> ### 4.1 Sticky glass header
> ```html
> <header class="site-header">
>   <span class="brand">CyrFlip<span class="dot">.</span></span>
>   <div class="seg" role="group" aria-label="Language">
>     <button data-lang="ru">RU</button><button data-lang="en" aria-pressed="true">EN</button><button data-lang="ua">UA</button>
>   </div>
>   <button class="theme-btn" id="themeBtn" aria-label="Switch theme">◐</button>
>   <a class="btn btn-primary btn-sm" href="#get">Download</a>
> </header>
> ```
>
> ### 4.1a Product identity — use the Hub pattern
>
> Every project follows the SZA hub hierarchy:
>
> - **Header, left:** one compact brand unit — optional small monochrome product mark + product name.
> - **Hero:** optional small category/platform eyebrow, then the H1 with the visitor-facing outcome.
> - The hero does **not** repeat the product mark or product name from the header. It begins the explanation.
>
> This keeps identity stable while the first large heading answers “what does this give me?”
> The hub itself uses `SZA.` as its brand unit; product pages substitute their own name and, only when useful,
> their monochrome mark.
>
> ### 4.2 Language switcher — RU · EN · UA, **no flags**
> - Visible labels exactly: `RU` `EN` `UA` (note **UA**, not "UK"). Button order: **RU, EN, UA**.
> - Mechanism: pure-CSS `data-lang` (Section 6). Persist to `localStorage`. Pre-paint resolver (Section 7).
> - No flag emoji, no flag images. Text only.
>
> ### 4.3 Theme toggle — light/dark on **every** site
> - `◐` button flips `data-theme` on `<html>` between `dark`/`light`. Persist to `localStorage`.
> - First visit: `prefers-color-scheme` (default **dark**). Resolve before paint (Section 7).
> - Update `<meta name="theme-color">` and `aria-label` on toggle.

> ### 4.6 Numbered collapsible sections (TOC-as-content)
> ```html
> <details class="sec" id="install">
>   <summary><span class="secnum">02</span> Install &amp; start</summary>
>   <p>…</p>
> </details>
> ```
> Number sections `00, 01, 02…`. Deep links auto-open + scroll (Section 7). Provide **Expand all / Collapse all** when there are 4+.
>
> ### 4.7 Copy-to-clipboard box
> For commands, **and on the Hub for phone & email.**
> ```html
> <div class="copybox"><span class="tok">winget</span> install SerZhyAle.CyrFlip
>   <button class="copy" data-copy="winget install SerZhyAle.CyrFlip">Copy</button></div>
> ```
> Clipboard API + `execCommand` fallback; show "✓ Copied" for ~1.6s (`.copy.done`). Script in Section 7.

> ### 4.10 Distribution channels + dynamic GitHub release
> - Render a `btn` per existing official channel (Store / winget / Play). winget/CLI commands go in a `.copybox`.
> - **GitHub release:** fetch `https://api.github.com/repos/SerZhyAle/<repo>/releases/latest`, label the button with the
>   tag/asset, link to the asset. **Static fallback** to `/releases` if the fetch fails. Never hardcode a version string.
>
> ### 4.11 Footer — "More tools by SZA" + contact
> ```html
> <footer class="site-footer"><div class="container">
>   <div class="tools-grid"><!-- all sibling tools except the current one (Section 10) --></div>
>   <div class="footer-bottom">
>     <span>© 2026 Serhii Zhyhunenko</span>
>     <span><a href="https://sza.od.ua">sza.od.ua</a> · <a href="https://github.com/SerZhyAle">GitHub</a> · <a href="mailto:sza@ukr.net">sza@ukr.net</a></span>
>   </div>
> </div></footer>
> ```
>
> ### 4.12 Back-to-top
> `.to-top` button, shown after ~600px scroll (Section 7). Long pages only.

> ## 5. Visual language
>
> - **Color:** see tokens in [`sza-kit.css`](26_2026-09-23_contract-product-web-pages-sync/sza-kit.reference-2026-09-25.css). Green `--acc` is the primary/CTA color; gold `--gold` is the
>   *secondary* accent — use it for one thing at a time (a callout border, a highlight tag), never as a second CTA color.
>   Code blocks stay dark in both themes.
> - **Typography:** Outfit (headings, 700–800), Plus Jakarta Sans (body, 300–600), system mono for code. Tight letter-spacing on headings (`-0.02em`).
> - **Spacing & shape:** radius 16px cards / 10px small / pill buttons. Generous but not airy. `--wide: 1100px` container.
> - **Background:** subtle blurred blobs (`.bg-blobs`), opacity ~0.14. Decoration only; never competes with text.
> - **Motion:** short (≤0.3s), purposeful (hover lift, fade-in). All disabled under `prefers-reduced-motion`.
>
> ## 6. Internationalization (RU / EN / UA)
>
> - Three languages: **RU, EN, UA**. Labels text-only (no flags). Visible label for Ukrainian is **UA** (lang attribute may stay `uk`).
> - **Default mechanism (Hub, Docs, small/medium apps): pure-CSS toggle.** All languages in the DOM; show the active one:
>   ```css
>   html[data-lang="ru"] [data-l]:not([data-l="ru"]){display:none}
>   html[data-lang="en"] [data-l]:not([data-l="en"]){display:none}
>   html[data-lang="ua"] [data-l]:not([data-l="ua"]){display:none}
>   ```
>   `setLang()` sets `data-lang` + `lang` on `<html>`, persists, and swaps `<title>`/description/OG from a JS map.
> - **Big SEO app (FastMediaSorter_mob_v2): keep separate per-language pages** (`index.html`, `index-ru.html`, `index-uk.html`)
>   with `hreflang` — but relabel the switcher to RU/EN/UA, no flags, and restyle to the kit.
> - First visit: `navigator.language` → `ru*`→RU, `uk*`→UA, else EN. Persisted choice wins.
>
> ## 7. Required scripts (vanilla, no libraries)
>
> Place a **pre-paint** snippet in `<head>` *before* the stylesheet (prevents theme/lang flash), and the rest before `</body>`.
>
> ```html
> <!-- in <head>, before CSS -->
> <script>
> (function(){try{
>   var t=localStorage.getItem('sza-theme')||(matchMedia('(prefers-color-scheme: light)').matches?'light':'dark');
>   document.documentElement.setAttribute('data-theme',t);
>   var l=localStorage.getItem('sza-lang');
>   if(!l){var n=(navigator.language||'en').toLowerCase();l=n.indexOf('ru')==0?'ru':n.indexOf('uk')==0?'ua':'en';}
>   document.documentElement.setAttribute('data-lang',l);
> }catch(e){}})();
> </script>
> ```
>
> Body scripts: `setLang()`, `toggleTheme()`, copy buttons (`data-copy` → clipboard + "✓ Copied"),
> `openFromHash()` (auto-open the `<details>` a `#hash` targets + scroll, on load and `hashchange`),
> expand/collapse-all, and back-to-top show/hide. Keep it dependency-free
> (exception: `marked.js` is allowed only on the big app for markdown-as-data-source).

> ## 9. Emoji & icon policy
>
> - **No emoji.** Do not use emoji in headings, prose, buttons, badges, quickstart steps, or decorative UI.
> - For a section or action, prefer a **small, single-colour inline SVG** with a familiar meaning. It must support the
>   label, not replace it. Use one consistent stroke/weight family per page.
> - A small product icon is appropriate when it is already recognisable inside the product (for example, its app icon or
>   a familiar product symbol) **and has a monochrome variant**. Otherwise use a neutral outline SVG. Do not create
>   colourful illustrative icon sets just to decorate a page.
> - For UI affordances (theme, copy, download, arrows) prefer inline SVG icons or the few neutral glyphs already in the
>   kit (`◐ ⤓ → ▸`). Functional, not festive.

> ## 11. Acceptance checklist (per site)
>
> - [ ] `sza-kit.css` tokens in use; **no** leftover indigo/purple/cyan; green primary, gold secondary only.
> - [ ] Outfit + Plus Jakarta Sans loaded; no Inter / no theme-default fonts.
> - [ ] Light + dark both work; toggle persists; **no flash** on reload.
> - [ ] Language switcher shows **RU EN UA**, no flags; switching works and persists; metadata localizes.
> - [ ] (App sites) Above-the-fold order: name → tagline → "what & for whom" → Get-it (channels + GitHub release + quickstart).
> - [ ] GitHub release link is dynamic or, if static, points to `/releases/latest`.
> - [ ] Copy buttons work (incl. fallback) and show "✓ Copied".
> - [ ] Footer "More tools by SZA" grid present and correct; contact = sza@ukr.net.
> - [ ] No emoji; icons are familiar, monochrome, and support rather than replace labels.
> - [ ] Responsive at 360 / 768 / 1280+; 44px touch targets; `prefers-reduced-motion` respected; visible focus.
> - [ ] Existing functionality preserved (no broken downloads, anchors, or scripts).

### `PAGE-CONTENT` 1.1 - the shared contracts catalog, `product-web-pages/PAGE-CONTENT.md`

The document numbers only its page order. The ticket's shorthand maps as follows: **O*n*** = item *n* of
"Mandatory page order for apps"; **L*n*** = bullet *n* of "Layout contract" (L1 full width, L3 mark once /
outcome H1).

> Installation and download are neutral navigation actions. Do not use urgency,
> superlatives, marketing claims, invented channels, or repeated CTAs.

> Never invent an install method, command, capability, platform, release asset, or
> compatibility claim. If a fact is unclear, leave a clear TODO in the proposed
> copy and ask the maintainer rather than guessing.

> ## Mandatory page order for apps
>
> 1. Sticky header: brand, RU / EN / UA, theme, and one neutral link to `#get`.
> 2. H1 and one-line outcome tagline.
> 3. **What it is and who it is for**: one or two plain sentences.
> 4. **Proof of the use case**: one demo strip, screenshot, or three to five
>    scenario cards. The core functions/scenarios belong near the top. Use a
>    scenario before a feature catalogue.
> 5. **Get started** (`#get`): real channels; GitHub Releases; two or three
>    steps from install to the first useful result. Make the safe path primary.
> 6. Three to six feature groups, described through user outcomes. Put extensive
>    detail in disclosure sections or child documentation.
> 7. Variant/edition selector only when variants exist and the choice affects the
>    user.
> 8. Practical guides, FAQ, and troubleshooting.
> 9. Technical details, full compatibility information, source code, privacy,
>    licence, and honest caveats.
> 10. Footer: related SZA tools and contact.
>
> Use expandable `<details>` groups to keep the page dense and make good use of the
> available screen area. Put full feature inventories, compatibility, editions,
> FAQ, troubleshooting, technical implementation, legal material, and release
> history there. Never hide the project purpose, three to five core
> functions/scenarios, first safe action, or a material safety warning.
>
> ## Layout contract
>
> - Use the full available content width on wide screens. Ordinary text, headings,
>   and section introductions outside cards, callouts, and disclosure groups start
>   at the left edge of the container and have **no arbitrary narrow max-width**.
> - Cards and expandable groups may have their own constrained internal layout;
>   they are the exception, not the page default.
> - A product page shows its recognisable program mark **once** in the header or
>   hero, never both. Follow the SZA hub hierarchy: the header contains one compact
>   brand unit (optional small monochrome mark + product name); the hero starts with
>   a category/platform eyebrow and an outcome-focused H1, without repeating the
>   mark or product name. Do not duplicate an app icon just to fill space.
> - Do not use emoji. Use a small single-colour SVG only where it is familiar and
>   meaningful. A product mark is permitted only in a monochrome variant; otherwise
>   use a neutral outline icon. Icons accompany labels and never replace them.

> ## Variants
>
> [..]
> - **Medium app:** hero → what/for whom → scenarios → get started → features →
>   guides → details.

> - Use stable action labels: **Install**, **Get started**, **Guides**,
>   **Documentation**, **Source code**.
> - Put commands in copy boxes and show their expected effect.
> - Related tools occur in body copy only when contextually useful; the complete
>   family is in the footer.

> ## Acceptance test
>
> An unfamiliar visitor should be able to say, before a deep scroll: “This is
> ___; it is for me when ___; I start by ___.” They should then reach the first
> useful result in two or three documented steps.
>
> Also pass the technical checklist in `PAGE-STYLE` section 11.

The channels for the get-it block are this product's own facts, not contract text: `.sza-canon.json`
`channels` lists `github`, `winget`, `msstore`, `installer`, `chrome`, `edge`.

### Domain README - the shared contracts catalog, `product-web-pages/README.md`

> 1. **Content order before visual work.** `PAGE-CONTENT` decides meaning and order, `PAGE-STYLE` decides
>    rendering. Where the two disagree, `PAGE-CONTENT` wins on what and in what order, `PAGE-STYLE` on how
>    it is drawn. (`PAGE-CONTENT` opening; `PAGE-STYLE` section 3)

> 9. **The release link is never a hardcoded version.** Resolve the latest GitHub release at run time and
>    fall back to `/releases/latest`, never to a version string baked into the page.
>    (`PAGE-STYLE` section 4.10)

> The one machine-checkable artifact is the stylesheet. `reference/sza-kit.css` is canonical; a product's
> served copy conforms when it is byte-identical to it, or when the difference is a recorded exception:
>
> ```
> cmp <catalog>/product-web-pages/reference/sza-kit.css <repo>/<path>/sza-kit.css
> ```

### Proposals already in the catalog that cover B-items (not filed by this product)

`product-web-pages/PROPOSAL-2026-09-24-get-position.md` (FastMediaSorter_Lite) - covers B2:

> **Ask.** Decide which document owns section order - the natural answer is `PAGE-CONTENT`, which is about
> what a page says and in what order - and make the other cite it instead of restating it.

`product-web-pages/PROPOSAL-2026-09-23-universal-agent-kit-page-style.md` item 2 - covers B9:

> State one mapping: the switch
>    reads UA and keys `ua` in `data-lang` / `data-l` / `sza-lang`; `lang`, `hreflang`, URL parameters and
>    sitemaps use ISO `uk`; a resolver reading `sza-lang` accepts both values.

The same proposal's item 3 (`?lang=` wins over the stored choice) covers B8, item 7 (section 9 defers to
`ICON-SET`) covers B13, and `PROPOSAL-2026-09-23-documentation-method-start-and-actions.md` item 3 (width)
covers B3.

### `WAVE-PARTICLES` 0.10 - the shared contracts catalog, `animated-backdrop/README.md` (header)

> status: draft
>
> consumers: FastMediaSorter Android (phone and launcher), FastMediaSorter Android (watch), FastMediaSorter website, StreamsPlayer (Windows desktop)
>
> artifacts: reference/wave-particles.js (rung 2 of section 7); rung 3 not written
