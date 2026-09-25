# Pointer: HARNESS-PROFILE

- **Id:** `HARNESS-PROFILE`
- **Version:** 0.9 draft
- **Home:** the shared contracts catalog, `rule-adoption/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** not applicable - the canon's shipped harness is never run against this repository
- **Wire carrier:** none - there is no `.sza-profile.json`

**Why not applicable.** The profile only configures the shipped harness, and nothing here invokes it: the
gate is [`scripts/check.ps1`](../../scripts/check.ps1), the spec flow is this repo's own `/spec*` commands.
No profile file exists at the root, on purpose.

**Latent risk, and why it is written down.** The contract's rule 7 reads a missing profile as "use the
defaults", and the defaults assume a `PLAN/` folder and `^S\d{4}$` ticket ids. This repo keeps tickets as
`DEV/plan/NN_<date>_<slug>.md` (declared in [`AGENTS.md`](../../AGENTS.md) and
[`CLAUDE.md`](../../CLAUDE.md)), so a harness-based skill run here would find no tickets and report an
empty queue rather than fail. A reader must not "fix" this by adding a profile without also adopting the
harness; the catalog is asked to state that a repository which never runs the harness needs no profile.

**What this repo owes it.** Nothing while the harness is unused. The day a harness-based skill is adopted
here, the same change adds `.sza-profile.json` mapping the ticket folder and id scheme above, and this
pointer's role becomes consumer.

**Conformance.** None to run - absence of `.sza-profile.json` is the declared state.
