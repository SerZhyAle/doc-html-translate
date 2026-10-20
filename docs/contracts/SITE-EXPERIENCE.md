# Pointer: SITE-EXPERIENCE

- **Id:** `SITE-EXPERIENCE`
- **Version:** 0.4 (draft)
- **Home:** the shared contracts catalog, `product-site/SITE-EXPERIENCE.md`, with the run-list `product-site/SITE-CHECKLIST.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, `partial` - guide tier; there is no portal layer, the page layer is `assets/site.css`; read at 0.4 on 2026-10-20 (ticket 120)
- **Wire carrier:** none - a stylesheet layer, a set of components and a behaviour contract

What this repo owes it:
- The kit `assets/sza-kit.css` byte-identical and linked before `assets/site.css` (rule 1; pinned by
  `tests/site_test.go`). No layout or colour in a `style` attribute (rule 4).
- The pre-paint theme resolver in every `<head>` and `sza-lang` / `sza-theme` as `PAGE-STYLE` section 7 states
  (rules 7, 10); a full-width content box at 1920 and 2560 px (rule 20).
- Every origin a page contacts is declared and named on the privacy pages (rule 14): `fonts.googleapis.com`,
  `fonts.gstatic.com`, and `api.github.com` for the release tag. Declared in `docs/security-posture.json`
  (`siteOrigins`, the `site` rows), rendered as a "This website" block on `privacy.html`,
  `extension-privacy.html` and `install-trust.html`, and held by `scripts/security-posture.ps1`.
- A skip link, a labelled `nav` per navigation, targets and contrast measured in both themes (rules 15-17), reduced
  motion including the back-to-top glide (rule 19), one `main`, one `h1`, ordered headings, `lang`, `dir` and `alt`
  on every page. Version 0.4 adds the Support page linked from every footer and release history from versioned
  records (`SITE-STRUCTURE`).

**Browser run, 2026-10-20 (`tools/sitecheck --tree`): `DONE`, info 10, pass 57, fail 1, na 2.** Rules 4, 7, 10, 14,
15, 16, 18, 19 and 20 and rule 17 for text and targets pass (0 of 1909 text runs under 4.5:1 in both themes; no
control under 44 px). The one fail is rule 1: the kit is `aea958f8` and the catalog's reference is `e438fef2` -
a dated exception, ticket 119, until 2026-12-20. The component-edge contrast probe (information only) reads
1.00-1.26:1 on `.btn`, `.seg button` and `.theme-btn` with the catalog kit's own palette; a proposal asks the owner
to define "components" (the `product-site` proposal file of 2026-10-20, item 4).

**`partial` because** the screen reader pass (rule 12) is undriven (dated exception, until 2026-12-31). Not
verified: NVDA, OS high contrast, real touch and DPI, the 360 px fold, the adapter over the new kit.
