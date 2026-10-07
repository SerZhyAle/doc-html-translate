# Pointer: PACKAGE-VERSIONING

- **Id:** `PACKAGE-VERSIONING`
- **Version:** 0.5 draft
- **Home:** the shared contracts catalog, `package-versioning/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - Windows desktop profile, registered `dotted` rendering
- **Owner:** FastMediaSorter Android

The desktop stamp keeps `YY.MMDD.HHmm`; the channel mappings and `-Stamp` pin derive from one
memoised instant per invocation. The 0.2 Windows profile explicitly registers this rendering and
requires no invented versionCode. Changing a shipped grammar still needs a channel-ordering plan.

**Independent edition (0.5 section 8 E1-E4, reviewed 2026-10-07).** The browser extension
of doc-html-translate publishes through Chrome Web Store and Edge Add-ons on its own
release run and instant, shared by its manifest and package metadata. It shares no pin
with desktop CLI, GUI, installer, winget or MSIX. The registered dotted stamp is rendered
as integer `YY.MDD.Hmm` parts (leading zeros removed) for the extension stores; `build.mjs`
computes it once per invocation. Versions order within each channel, never between editions.
The catalog exception that waited on W5 is closed; artifact read-back and fixed-instant
ordering checks for every remap remain an explicit exception until 2026-12-31.

**Conformance.** No catalog vectors. `scripts/verify-exe-version.ps1` reads the desktop
executable versions. No extension release or artifact build was performed by ticket 108.
