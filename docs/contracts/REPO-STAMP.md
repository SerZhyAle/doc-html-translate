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
  An equal digest owes no stamp write at all: a version-only re-stamp by hand is the same breach (commit
  `e3f4301`, 2026-09-06, did exactly that).
- Every exemption carries an `id`, a `path` and a `reason`. The gate treats `path` as a wildcard and an
  exemption without one as covering every path, so `path` is never left out here.
- Other fields (`site.pages`, `channels`, ..) are edited in the change that makes them true.

**Conformance.** The canon plugin's compliance gate (shipped with the plugin, not in this repository),
run under PowerShell 7.
