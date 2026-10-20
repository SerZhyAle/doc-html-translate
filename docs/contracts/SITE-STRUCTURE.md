# Pointer: SITE-STRUCTURE

- **Id:** `SITE-STRUCTURE`
- **Version:** 0.3 (draft)
- **Home:** the shared contracts catalog, `product-site/SITE-STRUCTURE.md`, with the run-list `product-site/SITE-CHECKLIST.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, `partial` - tier **guide** (owner decision, 2026-10-06, ticket 96); locale sets as published; read at 0.3 on 2026-10-20 (ticket 120)
- **Wire carrier:** none - a page set, a navigation shape and an address scheme

What this repo owes it:
- The page set equals `sitemap.xml` both ways (rules 1, 2); `scripts/doc-registry.ps1` holds it. Every page is
  one step from the landing (rule 3).
- Locale sets as declared in the adoption row: landing 13, manual 3, trust, privacy and support pages 3 (rule 10).
  A language control offers only targets that answer (rule 9).
- The Support page type (section 2, 0.3) is `support.html` (RU / EN / UA in page), linked from every footer; the
  `#support` and `#logs` anchors of the landing and the manual stay.
- The visitor's language is kept (8.B): the in-page pages are opened in the language the visitor already
  chose, carried at run time as `?lang=` when one is known (2026-10-20).
- The list of addresses held outside the site is `configs/site-held-addresses.jsonl`, resolved against the page
  set by `scripts/site-addresses.ps1` inside `scripts/check.ps1`; the forwarder policy is
  [`../SITE_ADDRESSES.md`](../SITE_ADDRESSES.md) (rule 8, ticket 98). Permanent addresses and anchors never move
  (8.D); `?l=` addresses keep answering beside the canonical `?lang=`.
- The release-notes page (rules 2, 14) is `release-notes.html`, assembled at each release from
  `docs/release-notes.json` by `scripts/release-notes.ps1` and held against the app release tags inside
  `scripts/check.ps1` (ticket 103). The not-found page (rule 15) is `404.html` (ticket 97). Both registry
  exceptions are closed (2026-10-20): the live host answers an unknown address with 404 and the product page.
- Portal-only rules 4-7 and 12-14 are not bound at the guide tier.

**`partial` because** 8.C and 8.E (the app and the generated help link their page per language) are read and owned
with ticket 118, and the tree changes of 2026-10-20 are not published, so nothing is claimed about the live site
beyond status reads. Proposals: the `product-site` proposal file of 2026-10-20, items 1 and 2 (a titled section may
satisfy the Support page type at the guide tier).

**Conformance.** `tools/sitecheck --tree` (2026-10-20: `DONE`, info 10, pass 57, fail 1 = the kit `B4a`, na 2),
`go test ./tests -run TestSite` (47 PASS), `scripts/site-addresses.ps1` and `scripts/release-notes.ps1`.
