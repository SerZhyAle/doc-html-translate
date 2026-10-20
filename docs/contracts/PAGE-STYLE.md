# Pointer: PAGE-STYLE

- **Id:** `PAGE-STYLE`
- **Version:** 3.1
- **Home:** the shared contracts catalog, `product-web-pages/PAGE-STYLE.md`, reference kit `product-web-pages/reference/sza-kit.css` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, `partial` - role "App - medium" (the distribution block is required); read at 3.1 on 2026-10-20 (ticket 120)
- **Wire carrier:** `assets/sza-kit.css` - the one machine-checkable artifact of the domain

What this repo owes it:
- The kit byte-identical to the reference; everything this site adds lives in `assets/site.css`, linked after it.
  The identified adapter over the kit (sections 4.4 and 12) is the "UI CONTRACTS" block of `assets/site.css`
  (ticket 119, slice W1): rectangular buttons, the 44 / 48 px profile, the state triangle of a group, the
  external-link marker. Shared body scripts (language, theme, copy, deep links, expand/collapse-all,
  back-to-top, carrying the visitor's language) are `assets/site.js`; the pre-paint resolver stays inline in each
  `<head>`.
- Every page: Outfit + Plus Jakarta Sans, light and dark with a persisted toggle and no flash, the glass header,
  the footer of [`SITE-FAMILY-MAP`](SITE-FAMILY-MAP.md), no emoji.
- RU / EN / UA switch keyed `ru|en|ua` in `data-lang`, `data-l` and `localStorage` `sza-lang`; a stored `uk` is
  read as `ua`. The ten locale landings and the three docs pages are separate per-language pages.
- Locale URLs (amendment 1): `?lang=<ISO code>` is canonical and `?l=` is still read as the legacy form; a valid
  `lang` wins, a page ignores a locale it does not ship, canonical and hreflang addresses use `lang`. Done in the
  resolver of the seven pages that carry one, the sixteen hreflang pages, the sitemap and the held-address gate
  (2026-10-20, ticket 120); `?l=` links keep answering.

**Open, in the catalog registry (section 3).** The kit: the reference is now `e438fef2` (15926 bytes) and the
vendored `assets/sza-kit.css` is still `aea958f8` (14521 bytes), so the browser check `B4a` fails; version 3.1
gives no grace period. Re-vendor with the copies under `help/` and move the pin in `tests/site_test.go` - ticket
119, milestone 2026-12-20 (the page overrides already meet the contrast the new kit asks for; the adapter over
the new kit is read, not rendered). Also dated: the theme control's visible label and glyph (ticket 119), and the
locale-list order and `sza-lang` write on the ten locale landings (until 2027-03-31).

**Conformance.** `tests/site_test.go` pins the kit bytes, resolver and page width; `tests/site_a11y_test.go` holds
the target and motion rules; `tools/sitecheck --tree` and `tools/uicheck --only site` run the rendered checklist
(2026-10-20: sitecheck `DONE` with the one `B4a` fail, uicheck 655 checks PASS).
