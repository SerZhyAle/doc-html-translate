# Pointer: SITE-FAMILY-MAP

- **Id:** `SITE-FAMILY-MAP`
- **Version:** 1.10
- **Home:** the shared contracts catalog, `product-web-pages/SITE-FAMILY-MAP.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - this product's row is `https://serzhyale.github.io/doc-html-translate/`; read at 1.5 on 2026-10-20 (ticket 120)
- **Wire carrier:** none - the footer tools grid of every site page

What this repo owes it:
- Every page's footer grid lists every other row of the map (the hub included) and not this product. The product
  keeps the full grid on every page, a superset of the minimum a child page owes (footer-scope amendment).
- Each row carries the map's label: the StreamsPlayer row reads **STREAMS Player** (1.4) in all 20 footers, and the
  Android sibling reads Fast Media Sorter & Organizer; `tests/site_test.go` pins both labels.
- One contact everywhere: `sza@ukr.net`, GitHub `SerZhyAle` - on the pages, the READMEs and the extension's
  store privacy source `extension/store/PRIVACY.md`.
- Adding a row to the catalog map is work for every page of this site; the list lives once, in
  `tests/site_test.go` (`familyURLs`), and the test fails until every footer carries it.

**Conformance.** `tests/site_test.go` checks the footer URLs, the labels and the contact string on every page. That
each URL answers is the contract's section 5 command, run by hand before a release (all nine answered 200 on
2026-10-20). The shared pages are not cards and the extension page is a child page (amendment 3), so the old
remark that the dual public entry convention is undecided no longer applies.

**Re-read 2026-10-29 (ticket E0126) at 1.10.** 1.9 and 1.10 rename the Windows row: every footer card reads "Fast Media Sorter & Sharing for Windows" / "Windows media and file manager", the related-tool links use the 1.9 name, and `TestSiteFooterCarriesTheFamilyMap` pins both. 1.6 rule 6 holds (no footer links a help index); 1.8's section 2a pages are not linked from this site.

**2026-11-08 (ticket E0111, site audit F25).** The ten language landings render the *Type* cell in the page's language, the *Tool* cell and the URL untouched; section 2 does not call the column fixed text, and the catalog's `PROPOSAL-2026-11-08-family-map-type-in-the-page-language.md` asks it to say so. Every other page keeps the English cell. `tests/site_test.go` pins the English cell on those pages and the translated Windows cell on the landings. The footer's GitHub link reads "SerZhyAle on GitHub" in the page's language (site audit F29).
