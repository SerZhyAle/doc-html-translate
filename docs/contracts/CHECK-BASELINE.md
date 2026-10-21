# Pointer: CHECK-BASELINE

- **Id:** `CHECK-BASELINE`
- **Version:** 0.11 draft
- **Home:** the shared contracts catalog, `automated-checks/README.md` section 3 (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer, `partial` - one set baseline is in use: `configs/help-baseline.txt`
- **Owner:** FastMediaSorter Android

The file that carries accepted debt from run to run: a count ratchet or a set, one-way, reviewable in a
diff, and accepted only by an explicit act with a reason.

**State here (2026-10-20, ticket 120).** The earlier "none, dormant" is out of date: ticket 118 introduced
[`../../configs/help-baseline.txt`](../../configs/help-baseline.txt), a **set** baseline of 10 locale ids (the
GUI locales whose help falls back to the English pages), read by `internal/helpcheck` through
`scripts/help-check.ps1`. It is plain, sorted (checked), reviewable in a diff, and its count rides the verdict
line (`.. 10 locales in the baseline`); an unlisted locale without pages fails, and a pruned line is only an
advisory (`TestTheBaselineMayFallAndNeverRise`). It is not a suppression (rule 8). Lint and typos still run at
zero findings and the Go suite has no allow-list.

**Rules 3.2, 3.4 and 3.6: closed 2026-10-21 (ticket 122, item 5).** `internal/helpcheck` compares the file with
`git show HEAD:configs/help-baseline.txt` (supplied by `tools/helpcheck`): a line the committed file did not hold
fails with its amount and identifiers unless a dated journal line `# accepted <id> on <date>: <reason>` records why.
The update path is `go run ./tools/helpcheck -accept-baseline "<reason>"`: it prints every identifier it accepts
and writes the reason beside it. A committed state that cannot be read ends as COULD NOT VERIFY. Still asked of
the owner: whether a baseline whose identifiers are recomputed from the tree each run satisfies the absorption test
(the `automated-checks` proposal file of 2026-10-20, item 2), which could make the comparison unnecessary.

`DEV/ocrlab/thresholds.json` is **not** a baseline in this contract's sense: it holds metric acceptance
bounds, each with its measured value, which 0.11 puts out of scope.

**Conformance.** `go test ./internal/helpcheck` and `scripts/help-check.ps1`.
