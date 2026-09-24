# Pointer: HARNESS-PROFILE

- **Id:** `HARNESS-PROFILE`
- **Version:** 0.9 draft
- **Home:** the shared contracts catalog, `rule-adoption/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer - tool runner profile and paths
- **Wire carrier:** `version` (JSON string)

Harness configuration rules:
- Falls back safely to default profile when root `.sza-profile.json` is omitted.
- Test runner and gate placement map configured via `configs/check-placement.jsonl`.

**Conformance.** `tests/placement_test.go` and `scripts/check.ps1`.
