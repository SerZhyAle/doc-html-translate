# Pointer: INSTALL-TRUST

- **Id:** `INSTALL-TRUST`
- **Version:** 1.0
- **Home:** the shared contracts catalog, `install-trust/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** producer - bound by the contract, **not yet adopted**
- **Reference rendering:** FMS Companion's trust guide, in the same catalog folder

What a user reads in the thirty seconds after Windows tells them it protected their PC: four sections in
order (what the warning is, why it appears, exactly what to click, what the app never does), the dialog
quoted, the real reason including the cost of a certificate, and never an instruction to weaken a
protection.

**Where this product stands.** It ships a per-user installer (`doc-html-translate-setup-<version>.exe`) and
portable executables from GitHub and winget, and none of them is code-signed
([`../../installer/doc-html-translate.iss`](../../installer/doc-html-translate.iss) configures no signing),
so a user downloading any of the three meets SmartScreen. No surface answers it: neither the READMEs, nor
the site pages, nor the first-run GUI says the word. The Store build is signed by Microsoft during
certification and is outside this contract's scope.

The gap is declared in the catalog registry with a dated exception rather than left silent. Ticket:
[`../../DEV/plan/13_2026-09-22_install-trust-page.md`](../../DEV/plan/13_2026-09-22_install-trust-page.md).

**What this repo owes it**

- One page carrying all four sections, authored in English and translated into the locales the other
  user-facing surfaces carry, reachable from every place the download is offered.
- What it says about "what the app never does" must match the privacy page and the Store data-safety form
  word for word in substance.
- When the artifacts become signed the page is updated, not deleted.
