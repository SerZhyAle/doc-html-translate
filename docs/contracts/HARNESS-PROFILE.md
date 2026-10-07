# Pointer: HARNESS-PROFILE

- **Id:** `HARNESS-PROFILE`
- **Version:** 0.11 draft
- **Home:** the shared contracts catalog, `rule-adoption/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** not applicable - the canon's shipped harness is never run against this repository
- **Wire carrier:** none - there is no `.sza-profile.json`

**Why not applicable.** The profile only configures the shipped harness, and nothing here invokes it: the
gate is [`scripts/check.ps1`](../../scripts/check.ps1), the spec flow is this repo's own `/spec*` commands.
No profile file exists at the root, on purpose.

**Rule 7 (0.10).** A repository that never runs the harness needs no profile. The non-use declaration
is deliberate; adopting the harness later requires mapping the ticket scheme first.

**What this repo owes it.** Nothing while the harness is unused. The day a harness-based skill is adopted
here, the same change adds `.sza-profile.json` mapping the ticket folder and id scheme above, and this
pointer's role becomes consumer.

**Conformance.** None to run - absence of `.sza-profile.json` is the declared state.

**Review 2026-10-07 (ticket 108).** Rule 7 read at 0.11: the shipped harness is unused, no profile is owed. No reader of that format is added. No catalog vectors exist.
