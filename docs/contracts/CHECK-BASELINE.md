# Pointer: CHECK-BASELINE

- **Id:** `CHECK-BASELINE`
- **Version:** 0.9 (draft)
- **Home:** the shared contracts catalog, `automated-checks/README.md` section 3 (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, dormant - no check in this repo keeps a baseline today
- **Owner:** FastMediaSorter Android

The file that carries accepted debt from run to run: a count ratchet or a set, one-way, reviewable in a
diff, and accepted only by an explicit act with a reason.

**State here.** No accepted-findings file exists: lint and typos run at zero findings, and the Go suite
has no allow-list. The contract binds the day one is introduced - and it must then say which of the two
shapes it is (rule 3), use set semantics if debt is ever re-frozen wholesale (rule 4), and journal each
acceptance.

`DEV/ocrlab/thresholds.json` is **not** a baseline in this contract's sense: it holds metric acceptance
bounds, each with the measured value it was derived from. Whether that is a third shape of this contract
or out of its scope is an open question put to the owner in writing.

**Conformance.** Nothing to run.
