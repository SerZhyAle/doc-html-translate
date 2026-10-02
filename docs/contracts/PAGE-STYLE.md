# Pointer: PAGE-STYLE

- **Id:** `PAGE-STYLE`
- **Version:** 1.2
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

Deviations (each to be recorded as a dated registry exception in the catalog):
- `--wide` is `min(1760px,94vw)` in `assets/site.css`, not 1100px - `PAGE-CONTENT` asks for the full width
  (ticket 26, B3).
- The ten locale landings have the text-only secondary language list now admitted by 1.2. The remaining
  deviation is its EN-first order and lack of `sza-lang` writes on core-locale navigation; the documentation
  trio is admitted by section 6 and no longer excepted.
- The copy button's done state is the word "Copied" alone (`ICON-SET` 0.15), not section 4.7's "✓ Copied" (B13).

**Conformance.** `tests/site_test.go` pins the kit's SHA-256 and checks that every page links the kit then
`assets/site.css`, keys the language `ru|en|ua` and maps a stored `uk`. The section 11 checklist is a rendered
check (360 / 768 / 1280 px, light and dark) and is recorded in the ticket, not asserted here.
