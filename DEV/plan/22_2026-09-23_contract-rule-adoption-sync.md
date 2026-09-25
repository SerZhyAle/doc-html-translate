# The canon stamp and the repo layout, declared against their contracts

**Status:** Draft
**Priority:** 54
**Date:** 2026-09-23

> Contract sync ticket, both directions. No product code.
> Contracts: `REPO-STAMP`, `HARNESS-PROFILE`, `REPO-LAYOUT`, `RULE-DELIVERY`, all 0.9 (domain
> `rule-adoption/`, owner sza-unified-rules, drafts). No pointer in
> [`docs/contracts/`](../../docs/contracts/) yet.

## What / why

These are the machine-readable interface between the canon and a repository: the stamp file
`.sza-canon.json`, the names tools address a repo by, the harness profile, and the handshake through which
a rule set arrives and is judged stale. `REPO-STAMP`, `REPO-LAYOUT` and `RULE-DELIVERY` bind every
canon-adopting repository, and this one adopted. Its registry row is still the "every canon-adopting
repository - pending" placeholder.

Found by reading the stamp and running the canon's own digest tool on 2026-09-23.

## Findings that drive the work

**Held:** the stamp is at the root, valid, carries every required key and `role: product`; both
exemptions have an id and a reason; the stamped digest is the genuine digest of the 2026.09.06.1 core;
`CLAUDE.md` + `AGENTS.md`, `README.md`, `LICENSE` at the root; ledger shape 2 at `DEV/CHANGELOG.md`.

**Not held:**
- **`RULE-DELIVERY` rule 5 - stale.** The stamp says canon `2026.09.06.1`; the published plugin is
  2026.09.22.2 and the canon repository is at 2026.09.23.1. The compliance gate reports `SZA-CANON03` warn.
  Three versions behind, which the adopt-canon skill escalates to an error while the gate says warn (B5).
- **`REPO-LAYOUT` rule 2 - pointers untracked.** `docs/contracts/` exists only in the working tree, and
  the deletions of the two old documents it replaces are staged. HEAD has neither the old documents nor
  the pointers - the repo's committed state points at nothing.
- **`REPO-LAYOUT` rule 3 - unclear.** Tickets are `DEV/plan/<date>_<slug>.md` and research is
  `DEV/research/<topic>_<date>.md`, with no type prefix (B4).
- **`REPO-STAMP` rule 4 - gap in the contract, not in the stamp.** The stamp carries keys whose absence has
  no written meaning (B3).

**`HARNESS-PROFILE`:** not applicable - the shipped harness is never run here and there is no profile file.
Latent risk: the harness defaults (`PLAN/`, `^S\d{4}$` ticket ids) do not match this repo's layout, so a
harness-based skill run here would misread it.

**Canon-side finding, not this repo's to fix:** the compliance gate has no `#requires -Version 7`; under
Windows PowerShell 5.1 it reported 7 false errors and exited 1, under pwsh 7 it reported none on the same
tree.

## Direction A - the repo conforms

1. Commit `docs/contracts/` together with the staged deletions it replaces, in one commit (owner's go;
   commits only when asked).
2. Reconcile the stamp through the canon's adopt-canon flow to the current published version -
   `canon.version` and `coreDigest` rewritten by the tool, not by hand - and reconcile the changed rule
   documents it brings.
3. Pointer files for the four ids, listed in the pointer README, with `HARNESS-PROFILE` marked not
   applicable and why.
4. Registry: this product's row replacing the placeholder.

## Direction B - what the contracts need from this product

Filed as `PROPOSAL-2026-09-23-<topic>.md` in the catalog's `rule-adoption/` folder.

- **B1** `RULE-DELIVERY` rule 5: say whether a reconciliation resets `canon.adoptedOn`. Here it was not
  reset on two earlier re-syncs, so the 180-day escalation runs from a date that no longer means
  "adopted".
- **B2** `HARNESS-PROFILE`: a repository that never runs the harness needs no profile - say so, so a
  reader does not flag the missing file.
- **B3** `REPO-STAMP`: write down what absence means for `canon.adoptedOn`, `ledgerFile`,
  `exemptions[].path`, `versionShape.editionTagPrefixes`, `site.root`, `site.pages`, `privacy.page`,
  `privacy.sensitiveAccess` (`VERSIONING.md` §4 rule 7).
- **B4** `REPO-LAYOUT` rule 3: are date-slug tickets an allowed adaptation, or must new documents carry a
  type prefix?
- **B5** staleness ladder: the skill says error at two versions behind, the gate says warn - one answer.
- **B6** the gate running under the wrong PowerShell host should be "could not verify", not a failure.

## Done criteria

- [ ] `git ls-files docs/contracts` lists every pointer; the two replaced documents are gone from HEAD in
      the same commit.
- [ ] The compliance gate reports no `SZA-CANON03` finding.
- [ ] Pointer files and the registry row exist.
- [ ] B1-B6 filed or withdrawn in writing here.
- [ ] `.sza-canon.json` `site.pages` is updated by
      [`26_2026-09-23_contract-product-web-pages-sync`](26_2026-09-23_contract-product-web-pages-sync.md), not
      here - listed so the two do not both edit it.

## Open questions

1. Reconcile to the published plugin version now, or wait for 2026.09.23.1 to be published?
