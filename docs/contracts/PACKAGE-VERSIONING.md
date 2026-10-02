# Pointer: PACKAGE-VERSIONING

- **Id:** `PACKAGE-VERSIONING`
- **Version:** 0.2 draft
- **Home:** the shared contracts catalog, `package-versioning/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - Windows desktop profile, registered `dotted` rendering
- **Owner:** FastMediaSorter Android

The desktop stamp keeps `YY.MMDD.HHmm`; the channel mappings and `-Stamp` pin derive from one
memoised instant per invocation. The 0.2 Windows profile explicitly registers this rendering and
requires no invented versionCode. Changing a shipped grammar still needs a channel-ordering plan.

**Open exception.** The browser extension publishes on its own clock. W5 leaves the independent-edition
question open; the old exception is narrowed to that part, until 2026-12-31.

**Conformance.** `scripts/verify-exe-version.ps1` reads every built executable's coherent, non-`dev`
version; the stamp and channel tests hold the renderings. No catalog vectors exist yet.
