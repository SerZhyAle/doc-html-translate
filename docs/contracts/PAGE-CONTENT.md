# Pointer: PAGE-CONTENT

- **Id:** `PAGE-CONTENT`
- **Version:** 1.5
- **Home:** the shared contracts catalog, `product-web-pages/PAGE-CONTENT.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - variant "Medium app"; read at 1.5 on 2026-10-20 (ticket 120)
- **Wire carrier:** none - rendered page markup

What this repo owes it (the landing `index.html` and the ten locale landings):
- Page order: sticky header with the brand, the language switcher, the theme control, the documentation link and
  one neutral link to `#get` -> identity block, eyebrow, outcome H1 and tagline -> what it is and who it is for
  -> one proof strip -> the get-started block `#get` with every real channel (Microsoft Store, GitHub Releases,
  setup installer, winget copy box, Chrome Web Store, Edge Add-ons) and a three-step quickstart -> scenarios ->
  details in numbered disclosure groups -> footer. The documentation link in the header is new (2026-10-20,
  landing item 4).
- The application icon belongs in the hero identity block; the header carries text only (1.3).
- The free, key-less path first; paid and metered modes are named after it, never as the first action.
- Related tools only as contextual body copy (Fast Media Sorter for Windows as a companion); the full family is
  the footer.
- Never an invented channel, command or claim; the release link resolves the latest release at run time and
  falls back to `/releases/latest`.

**1.5 (read 2026-10-20).** The footer scope, the shared-page rule and the locale query are carried by
[`PAGE-STYLE`](PAGE-STYLE.md) and [`SITE-FAMILY-MAP`](SITE-FAMILY-MAP.md). The landing identity still repeats the
header mark and spells the hero name "Doc-HTML-Translate" against the family-map spelling - a dated registry
exception (until 2026-12-31). The "Download kit (.zip)" copy variant does not apply: this application has several
distribution channels. Child pages keep their own subject order (1.2); the extension page has its own
distribution block.

**Conformance.** No vectors. The rendered page checklist (`tools/sitecheck`), `tools/uicheck --only site` and the
product-site tests are the evidence.
