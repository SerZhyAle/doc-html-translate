# The product site follows the portfolio page contracts

**Status:** Draft
**Priority:** 50
**Date:** 2026-09-23

> Contract sync ticket, both directions. Site + docs, every authored locale in one edit.
> Contracts: `PAGE-CONTENT` 1.1, `PAGE-STYLE` 1.0, `SITE-FAMILY-MAP` 1.1 (domain `product-web-pages/`,
> owner the sza.od.ua hub, all active). Checked and **not applicable**: `WAVE-PARTICLES` 0.10.
> No pointer in [`docs/contracts/`](../../docs/contracts/) yet.

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
- the EN footer lacks StreamsPlayer and points OneClickRunner at its GitHub repo instead of its page;
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

## Direction A - the site conforms

Phase order matters: the bug first, the kit before the pages that depend on it.

1. **Language value** - one value space `ru|en|ua` on every page, and every reader maps a stored `uk` to
   `ua` (forward tolerance). Small enough for `/fix` ahead of the rest.
2. **Kit** - `assets/sza-kit.css` byte-identical to the reference; every page-local rule moves into a
   separate page stylesheet. Or, if the width override stays, a recorded exception.
3. **Family footer** - the full grid (all siblings, the hub, the heading, localized) on every page,
   locale pages included; contact `sza@ukr.net` everywhere including the extension privacy source.
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
   outside a `<title>`.
9. The install-trust page of [`2026-09-22_install-trust-page`](2026-09-22_install-trust-page.md) is built
   to the same child-page rules as 5-7.

## Direction B - what the contracts need from this product

Filed as `PROPOSAL-2026-09-23-<topic>.md` in the catalog's `product-web-pages/` folder, never as an edit.

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
- **B12** hub side: this product's card on the hub is stale (no comics, images, OCR, extension). Offer
  corrected copy to the hub owner.
- **B13** the `PAGE-STYLE` §9 glyph list conflicts with `ICON-SET` - joint proposal, see
  [`2026-09-23_contract-iconography-sync`](2026-09-23_contract-iconography-sync.md) B9.

## Done criteria

- [ ] Pointer files `docs/contracts/PAGE-CONTENT.md`, `PAGE-STYLE.md`, `SITE-FAMILY-MAP.md` exist and are
      listed in [`docs/contracts/README.md`](../../docs/contracts/README.md), which also records that
      `WAVE-PARTICLES` was read and does not apply.
- [ ] Registry: this product's consumer row for the three ids (reads 1.1 / 1.0 / 1.1) replaces the
      "eight product pages" placeholder for this product; every remaining deviation is a dated exception.
- [ ] Choosing a language on any page of this site shows exactly one language on every other page.
- [ ] `cmp` of the vendored kit against the reference is clean, or its exception is recorded.
- [ ] The `SITE-FAMILY-MAP` §5 URL check passes for every footer on every page; one contact everywhere.
- [ ] The `PAGE-STYLE` §11 checklist is run on a rendered page at 360 / 768 / 1280 px for the landing,
      extension, docs and privacy pages, light and dark, and the result is recorded.
- [ ] No docs link points at a branch file for a binary.
- [ ] B1-B12 filed or withdrawn in writing here.
- [ ] `.sza-canon.json` `site.pages` lists every page this site serves, locale landings included.
- [ ] Every surface changed in every authored locale in one edit.

## Open questions

1. Vendor the kit byte-identical and add a page stylesheet, or keep a standing exception for the width?
2. The outcome H1 in EN, RU and UA - the contract forbids inventing it; owner's wording.
3. The standing preference that a new feature leads the hero vs `PAGE-CONTENT`'s "no promotion" and
   outcome-first hero: which wins for the "New: 13 languages" paragraph?
4. Locale landings: add the RU/EN/UA switcher, keep a link list, or wait for B5?
5. Privacy pages: RU/UA versions, or English only (they are store-form URLs)?
6. Docs pages: restyle in place, or fold into the landing's numbered sections?
7. Store listings' own contact fields may also carry the old address - part of this work, or of the next
   release flow?
