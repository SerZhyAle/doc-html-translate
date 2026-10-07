# Pointer: SITE-STRUCTURE

- **Id:** `SITE-STRUCTURE`
- **Version:** 0.1 (draft)
- **Home:** the shared contracts catalog, `product-site/SITE-STRUCTURE.md`, with the run-list `product-site/SITE-CHECKLIST.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - tier **guide** (owner decision, 2026-10-06, ticket 96); locale sets as published
- **Wire carrier:** none - a page set, a navigation shape and an address scheme

What this repo owes it:
- The page set equals `sitemap.xml` both ways (rules 1, 2); `scripts/doc-registry.ps1` holds it. Every page is
  one step from the landing (rule 3, crawled 2026-10-06).
- Locale sets as declared in the adoption row: landing 13, manual 3, trust and privacy pages 3 (rule 10).
  A language control offers only targets that answer (rule 9).
- The list of addresses held outside the site is `configs/site-held-addresses.jsonl`, resolved against the page
  set by `scripts/site-addresses.ps1` inside `scripts/check.ps1`; the forwarder policy is
  [`../SITE_ADDRESSES.md`](../SITE_ADDRESSES.md) (rule 8, ticket 98).
- The release-notes page (rules 2, 14) is `release-notes.html`, assembled at each release from
  `docs/release-notes.json` by `scripts/release-notes.ps1` and held against the app release tags inside
  `scripts/check.ps1` (owner decision, 2026-10-06, ticket 103: a page on the site, not the GitHub Releases
  address). The not-found page (rule 15) was ticket 97. The shared registry's dated exception for the missing
  release-notes page is closed by this page.
- Portal-only rules 4-7 and 12-14 are not bound at the guide tier.

**Adopted 2026-10-06 (ticket 96), verdict `partial 0.1`.** Measured on the published site, not on the source;
the boxes that need real browser input are recorded as not run (ticket 102).

**Review 2026-10-07 (ticket 108).** Guide tier, existing locale groups and held-address scheme retained (sections 2 and 3). The working tree now includes the release-notes page from ticket 103; not-found and capability-source gaps remain separately excepted.
