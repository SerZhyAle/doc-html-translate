# Pointer: SITE-FAMILY-MAP

- **Id:** `SITE-FAMILY-MAP`
- **Version:** 1.1
- **Home:** the shared contracts catalog, `product-web-pages/SITE-FAMILY-MAP.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - this product's row is `https://serzhyale.github.io/doc-html-translate/`
- **Wire carrier:** none - the footer tools grid of every site page

What this repo owes it:
- Every page's footer grid lists every other row of the map (the hub included) and not this product.
- One contact everywhere: `sza@ukr.net`, GitHub `SerZhyAle` - on the pages, the READMEs and the extension's
  store privacy source `extension/store/PRIVACY.md`.
- Adding a row to the catalog map is work for every page of this site; the list lives once, in
  `tests/site_test.go` (`familyURLs`), and the test fails until every footer carries it.

**Conformance.** `tests/site_test.go` checks the footer URLs and the contact string on every page. That each URL
answers is the contract's section 5 command, run by hand before a release.
