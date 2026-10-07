# Pointer: SITE-REPRESENTATION

- **Id:** `SITE-REPRESENTATION`
- **Version:** 0.1 (draft)
- **Home:** the shared contracts catalog, `product-site/SITE-REPRESENTATION.md`, with the run-list `product-site/SITE-CHECKLIST.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - guide tier; rules 1-5 and 12 bind, rules 6-10 are read against the manual as a guide page
- **Wire carrier:** none - a content model: facts, their sources and the shape of a function page

What this repo owes it:
- One positioning source with ordered pillars, and the public editions with one display name each (rules 1-4):
  [`docs/POSITIONING.md`](../POSITIONING.md) holds the pillars in order and the store-policy exceptions (none);
  [`docs/positioning.json`](../positioning.json) holds the editions, their display names in en, ru and uk, the
  channels, the interface-language count and the device classes. `tests/site_positioning_test.go` compares the
  facts with the code they derive from and with the README trio, the documentation trio and the landings;
  the README, documentation and listing leads are read by hand for the pillar order (ticket 101).
- The privacy page, the permission list and the store justifications say one thing (rule 12); they are rendered
  from `docs/security-posture.json` by `scripts/security-posture.ps1`.
- No showcase page exists, so rule 11 does not apply.

**Adopted 2026-10-06 (ticket 96), verdict `partial 0.1`.** Rules 5-10 were not run (ticket 102).

**Review 2026-10-07 (ticket 108).** Rules 1-12 reviewed against positioning sources and ticket 107 manual anatomy. The missing capability-to-edition matrix (rule 9) remains an explicit exception; no capability is invented by synchronization.
