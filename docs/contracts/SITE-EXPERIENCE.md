# Pointer: SITE-EXPERIENCE

- **Id:** `SITE-EXPERIENCE`
- **Version:** 0.1 (draft)
- **Home:** the shared contracts catalog, `product-site/SITE-EXPERIENCE.md`, with the run-list `product-site/SITE-CHECKLIST.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - guide tier; there is no portal layer, the page layer is `assets/site.css`
- **Wire carrier:** none - a stylesheet layer, a set of components and a behaviour contract

What this repo owes it:
- The kit `assets/sza-kit.css` byte-identical and linked before `assets/site.css` (rule 1; pinned by
  `tests/site_test.go`). No layout or colour in a `style` attribute (rule 4).
- The pre-paint theme resolver in every `<head>` and `sza-lang` / `sza-theme` as `PAGE-STYLE` section 7 states
  (rules 7, 10).
- Every origin a page contacts is declared and named on the privacy pages (rule 14): `fonts.googleapis.com`,
  `fonts.gstatic.com`, and `api.github.com` for the release tag. Declared in `docs/security-posture.json`
  (`siteOrigins`, the `site` rows), rendered as a "This website" block on `privacy.html`,
  `extension-privacy.html` and `install-trust.html`, and held by `scripts/security-posture.ps1`.
- A skip link, a labelled `nav` per navigation, targets and contrast measured in both themes (rules 15-17):
  implemented locally by ticket 100 (`tests/site_a11y_test.go`); the exception closes after the re-run on the published site. One `main`, one `h1`, ordered headings, `lang`, `dir` and `alt` already hold on 18 of 18.

**Adopted 2026-10-06 (ticket 96), verdict `partial 0.1`.** Tab order, contrast numbers, reduced-motion
emulation and the 360 px layout were not run (ticket 102).
