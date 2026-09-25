# Pointer: REPO-STAMP

- **Id:** `REPO-STAMP`
- **Version:** 0.9 draft
- **Home:** the shared contracts catalog, `rule-adoption/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** producer - [`.sza-canon.json`](../../.sza-canon.json) at the repository root
- **Wire carrier:** none - `.sza-canon.json` JSON fields

The stamp declares canon version, core digest, adoption date, model (`reference`), overlay (`A`), editions
(`browser-extension`), channels, site pages, privacy and exemptions.

**What this repo owes it**

- The required keys stay present: `canon`, `overlay`, `ledgerShape`, and inside `canon` `version`,
  `coreDigest`, `model` (rule 3).
- `canon.version` and `canon.coreDigest` are written by the canon's adopt-canon run, never by hand and never
  copied from another repository (rule 8). A hand edit here claims a reconciliation that did not happen.
- Every exemption carries an `id` and a `reason`.
- Other fields (`site.pages`, `channels`, ..) are edited in the change that makes them true.

**Conformance.** The canon plugin's compliance gate (shipped with the plugin, not in this repository),
run under PowerShell 7.
