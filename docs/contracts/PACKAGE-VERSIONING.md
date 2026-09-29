# Pointer: PACKAGE-VERSIONING

- **Id:** `PACKAGE-VERSIONING`
- **Version:** 0.1 draft
- **Home:** the shared contracts catalog, `package-versioning/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, declared (the contract's own header names this product); the version stamp itself
  does not conform and stands as a dated exception in the catalog's registry (ticket 69)
- **Owner:** FastMediaSorter Android

What it binds: the version name and code every shippable package carries - one `yyMMddHHmm` instant per
invocation, rule 2's `Y.YM.MDDH.Hmm` name grammar, and the 9-digit versionCode partition of rule 3.

**Where this product stands.** The stamp is the per-project frozen shape `YY.MMDD.HHmm`
(`versionShape.tagRegex` in the root `.sza-canon.json`; the same digits, a different rendering), the
browser-extension edition stamps on its own clock, and there is no versionCode - winget, the MSIX
`Version` and the Inno stamp order on the grammar itself. What does hold: one memoised instant per
packaging invocation (`scripts/lib/` stamp helpers), a pure-timestamp name with no semantic version, and
`-Stamp` as the orchestrator's pin.

**What this repo owes it**

- Nothing today beyond the recorded exception. A change to the stamp shape is a change to a frozen anchor
  (`AGENTS.md`) and needs an owner decision plus a channel-ordering plan before the grammar moves.

**Conformance.** `scripts/verify-exe-version.ps1` proves every built exe carries a coherent, non-`dev`
stamp; the extension manifests are pinned by `extension/test/`. No vectors exist in the catalog yet
(its section 4 names the ladder).
