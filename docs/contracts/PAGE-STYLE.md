# Pointer: PAGE-STYLE

- **Id:** `PAGE-STYLE`
- **Version:** 1.6
- **Home:** the shared contracts catalog, `product-web-pages/PAGE-STYLE.md`, reference kit `product-web-pages/reference/sza-kit.css` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - role "App - medium" (the distribution block is required)
- **Wire carrier:** `assets/sza-kit.css` - the one machine-checkable artifact of the domain

What this repo owes it:
- The kit byte-identical to the reference; everything this site adds lives in `assets/site.css`, linked after it.
  Shared body scripts (language, theme, copy, deep links, expand/collapse-all, back-to-top) are `assets/site.js`;
  the pre-paint resolver stays inline in each `<head>`.
- Every page: Outfit + Plus Jakarta Sans, light and dark with a persisted toggle and no flash, the glass header,
  the footer of [`SITE-FAMILY-MAP`](SITE-FAMILY-MAP.md), no emoji.
- RU / EN / UA switch keyed `ru|en|ua` in `data-lang`, `data-l` and `localStorage` `sza-lang`; a stored `uk` is
  read as `ua`. The ten locale landings and the three docs pages are separate per-language pages.

**1.6 review (2026-10-07, ticket 108).** The current reference kit still hashes to
`aea958f805249d300b7417373e4cad18b44d7eed9765954df19ed710276c4c9e`.
Section 5's all-pointer 44 px targets and pseudo-element reduced-motion coverage live in
the page layer until the hub revises the kit; only then re-vendor it and remove the
duplicate motion rule. Copied is a localized word (section 4.7). The new non-landing
picker is optional; existing language controls need no replacement (section 4.2).
Theme labeling/glyphs, secondary-locale order/storage and landing identity differences
remain dated exceptions in the catalog registry.

**Conformance.** `tests/site_test.go` pins the kit bytes, resolver and page width;
`tests/site_a11y_test.go` holds the target and motion rules. The rendered checklist
is run by `tools/sitecheck`; ticket 108 records its command and verdicts.
