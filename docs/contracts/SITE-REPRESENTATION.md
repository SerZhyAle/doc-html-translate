# Pointer: SITE-REPRESENTATION

- **Id:** `SITE-REPRESENTATION`
- **Version:** 0.2 (draft)
- **Home:** the shared contracts catalog, `product-site/SITE-REPRESENTATION.md`, with the run-list `product-site/SITE-CHECKLIST.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, `partial` - guide tier; rules 1-5 and 12 bind, rules 6-10 are read against the manual as a guide page; re-read at 0.1 on 2026-10-20 (ticket 120), unchanged
- **Wire carrier:** none - a content model: facts, their sources and the shape of a function page

What this repo owes it:
- One positioning source with ordered pillars, and the public editions with one display name each (rules 1-4):
  [`docs/POSITIONING.md`](../POSITIONING.md) holds the pillars in order and the store-policy exceptions (none);
  [`docs/positioning.json`](../positioning.json) holds the editions, their display names in en, ru and uk, the
  channels, the interface-language count and the device classes. `tests/site_positioning_test.go` compares the
  facts with the code they derive from and with the README trio, the documentation trio and the landings;
  the README, documentation and listing leads are read by hand for the pillar order (ticket 101). A channel name
  typed on a page is not compared with the declared list.
- The manual's four task sections carry the fixed function-page anatomy, limits beside the action, interface words
  in the page's language and one place for defaults (rules 5-8, 10; ticket 107; `tests/site_manual_anatomy_test.go`,
  sitecheck `C4` and `C5`). At the guide tier these anchored task sections are the function pages (owner ruling
  2026-10-07; confirmation asked in the `product-site` proposal file of 2026-10-20, item 3).
- The privacy page, the permission list and the store justifications say one thing (rule 12); they are rendered
  from `docs/security-posture.json` by `scripts/security-posture.ps1`.
- No showcase page exists, so rule 11 does not apply.

**Rule 9 closed 2026-11-10 (ticket E0130):** the generated capability matrix exists - capability-matrix.html / .ru.html / .uk.html, rendered from `docs/capabilities.json` by `scripts/capability-matrix.ps1` (a fast gate child that also holds the dependency facts against the README and the security posture). The registry exception and the adoption row are closed; the contract itself is unchanged at 0.2.

**`partial` because** rule 9, the capability-to-edition matrix, did not exist (a dated registry exception, closed 2026-11-10 by ticket E0130).

**Re-read 2026-10-29 (ticket E0126) at 0.2.** Item A confirms the manual's task sections as function pages; no gated part (B); no roadmap or planned label (D). Still partial on rule 9 (capability matrix).
