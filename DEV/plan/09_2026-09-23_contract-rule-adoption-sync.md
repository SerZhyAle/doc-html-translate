# The canon stamp and the repo layout, declared against their contracts

**Status:** Draft
**Priority:** 54
**Date:** 2026-09-23

> Contract sync ticket, both directions. No product code.
> Contracts: `REPO-STAMP`, `HARNESS-PROFILE`, `REPO-LAYOUT`, `RULE-DELIVERY`, all 0.9 (domain
> `rule-adoption/`, owner sza-unified-rules, drafts). No pointer in
> [`docs/contracts/`](../../docs/contracts/) yet.

> **Remote execution (2026-09-25):** the contract text this ticket needs is quoted in "Contract snapshot" below, so every step not marked ⛔ runs in a cloud session from this repository alone. Steps marked **⛔ Local only** edit the shared contracts catalog (or another repository) and can run only on the owner's machine, where the catalog is mounted.

> **Re-verified 2026-09-25** against the catalog and the repo: the four pointers now exist and are tracked
> (commit `0c67c4e`, see Direction A step 1), and a registry row for this product was written 2026-09-24
> (Direction A step 4). The stamp is still at canon `2026.09.06.1`. Details in place below.

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
  *Still open 2026-09-25:* `.sza-canon.json` still carries `canon.version` `2026.09.06.1` and
  `adoptedOn` `2026-08-18`; the canon repository's `CANON_VERSION` now reads `2026.09.24.1`.
- **`REPO-LAYOUT` rule 2 - pointers untracked.** `docs/contracts/` exists only in the working tree, and
  the deletions of the two old documents it replaces are staged. HEAD has neither the old documents nor
  the pointers - the repo's committed state points at nothing.
  *Resolved 2026-09-25* - commit `0c67c4e` added all of `docs/contracts/` and deleted
  `docs/ocr-pipeline.md` and `docs/integration-image-translate.md` in the same commit;
  `git ls-files docs/contracts` lists 21 files, `git status` is clean.
- **`REPO-LAYOUT` rule 3 - unclear.** Tickets are `DEV/plan/<date>_<slug>.md` and research is
  `DEV/research/<topic>_<date>.md`, with no type prefix (B4). *2026-09-25:* tickets now carry a queue
  position, `DEV/plan/NN_<date>_<slug>.md` (CLAUDE.md "Spec / plan tickets"), still with no type prefix.
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
   **Done 2026-09-25** - commit `0c67c4e` ("Publish development specifications") adds the 21 files under
   `docs/contracts/` and deletes `docs/ocr-pipeline.md` and `docs/integration-image-translate.md`.
2. **⛔ Local only - needs the canon plugin/repo.** Reconcile the stamp through the canon's adopt-canon flow
   to the current published version - `canon.version` and `coreDigest` rewritten by the tool, not by hand -
   and reconcile the changed rule documents it brings. The digest is computed over the live rule documents
   (snapshot, `RULE-DELIVERY` rule 3) and must not be hand-written (`REPO-STAMP` rule 8), so a clone of
   this repository cannot do it. *Not done as of 2026-09-25* (stamp at `2026.09.06.1`, canon repository at
   `2026.09.24.1`).
3. Pointer files for the four ids, listed in the pointer README, with `HARNESS-PROFILE` marked not
   applicable and why.
   **Partly done 2026-09-25** - `docs/contracts/REPO-STAMP.md`, `HARNESS-PROFILE.md`, `REPO-LAYOUT.md`,
   `RULE-DELIVERY.md` exist, are tracked and are listed in `docs/contracts/README.md` (lines 31-34).
   *Still open (repo-only):* `HARNESS-PROFILE.md` and its README line call the repo a "consumer - tool
   runner profile and paths" and cite `configs/check-placement.jsonl` / `tests/placement_test.go`, which
   belong to `CHECK-PLACEMENT`, not to the harness profile. Rewrite it as not applicable: the shipped
   harness is never run here, there is no `.sza-profile.json`, and snapshot `HARNESS-PROFILE` rule 7
   (missing profile means the defaults) plus the latent-risk note above say why that matters.
4. **⛔ Local only - changes the contract catalog.** Registry: this product's row replacing the placeholder.
   **Done 2026-09-24, with errors** - the catalog's `_meta/REGISTRY.md` carries a doc-html-translate row
   for all four ids dated 2026-09-24 (quoted in the snapshot); the generic "every canon-adopting
   repository - pending" placeholder stays, as it serves the other repositories. The row misstates the
   repo on three points and needs a correction: it claims `.sza-profile.json` at the root (absent), overlay
   "A/C" (the stamp says `["A"]`), and a consumer role for `HARNESS-PROFILE` (not applicable, step 3). It
   also records no conformance evidence for the stale `RULE-DELIVERY` rule 5.

## Direction B - what the contracts need from this product

**⛔ Local only - changes the contract catalog** (every item below). Filed as
`PROPOSAL-2026-09-23-<topic>.md` in the catalog's `rule-adoption/` folder.

State of the catalog's `rule-adoption/` folder on 2026-09-25: four proposals, three of them filed by
FileDO on 2026-09-24 (`adoption-date`, `own-spec-scheme`, `stamp-defaults`) and one by
universal-agent-kit (`universal-agent-kit-adoption`). None names doc-html-translate.

- **B1** `RULE-DELIVERY` rule 5: say whether a reconciliation resets `canon.adoptedOn`. Here it was not
  reset on two earlier re-syncs, so the 180-day escalation runs from a date that no longer means
  "adopted".
  *Covered 2026-09-24 by FileDO's `PROPOSAL-2026-09-23-adoption-date.md`* (same question, FileDO's own
  history as evidence). What is left: co-sign it with this repo's evidence, or withdraw B1 here as covered.
- **B2** `HARNESS-PROFILE`: a repository that never runs the harness needs no profile - say so, so a
  reader does not flag the missing file.
  *Not filed as of 2026-09-25.* FileDO's `own-spec-scheme` touches the harness defaults but not this.
- **B3** `REPO-STAMP`: write down what absence means for `canon.adoptedOn`, `ledgerFile`,
  `exemptions[].path`, `versionShape.editionTagPrefixes`, `site.root`, `site.pages`, `privacy.page`,
  `privacy.sensitiveAccess` (`VERSIONING.md` §4 rule 7).
  *Mostly covered 2026-09-24 by FileDO's `PROPOSAL-2026-09-23-stamp-defaults.md`*, which lists the same
  seven keys except `exemptions[].path`. What is left: add `exemptions[].path` (co-sign or a short
  addendum), or withdraw the rest here as covered.
- **B4** `REPO-LAYOUT` rule 3: are date-slug tickets an allowed adaptation, or must new documents carry a
  type prefix?
  *Partly covered 2026-09-24 by FileDO's `PROPOSAL-2026-09-23-own-spec-scheme.md`*, which asks that a
  spec-id scheme declared in the agent-rules file be outside the prefix rule - but it is written for a
  scheme under `PLAN/`. This repo's `DEV/plan/NN_<date>_<slug>.md` is declared in `CLAUDE.md` and
  `AGENTS.md`; what is left is to ask that the carve-out not be tied to the `PLAN/` folder.
- **B5** staleness ladder: the skill says error at two versions behind, the gate says warn - one answer.
  *Not filed as of 2026-09-25.* Note the contract itself already answers from the digest and the adoption
  date, not from a version count (snapshot, `RULE-DELIVERY` rule 5) - so the skill is the side that
  disagrees with the contract.
- **B6** the gate running under the wrong PowerShell host should be "could not verify", not a failure.
  *Not filed as of 2026-09-25.* **⛔ Also needs the canon plugin/repo** to fix the gate itself.

## Done criteria

- [x] `git ls-files docs/contracts` lists every pointer; the two replaced documents are gone from HEAD in
      the same commit. *Done 2026-09-25 - commit `0c67c4e`, 21 tracked files.*
- [ ] **⛔ Waits on Direction A step 2 (local).** The compliance gate reports no `SZA-CANON03` finding -
      the gate ships with the canon plugin and the finding clears only after the tool-written stamp update.
- [ ] Pointer files and the registry row exist. *Pointer files exist (2026-09-25); `HARNESS-PROFILE.md`
      still to be rewritten as not applicable (step 3, repo-only).* **⛔ Local only - changes the contract
      catalog** for the registry half: the row exists since 2026-09-24 but needs the correction in step 4.
- [ ] **⛔ Local only - changes the contract catalog.** B1-B6 filed or withdrawn in writing here.
- [ ] `.sza-canon.json` `site.pages` is updated by
      [`12_2026-09-23_contract-product-web-pages-sync`](12_2026-09-23_contract-product-web-pages-sync.md), not
      here - listed so the two do not both edit it.

## Open questions

1. **⛔ Local only - needs the canon plugin/repo.** Reconcile to the published plugin version now, or wait
   for 2026.09.23.1 to be published? *2026-09-25:* the canon repository has moved on to `2026.09.24.1`,
   so the question is now "reconcile to whatever is published when step 2 runs".

## Contract snapshot (2026-09-25)

Quoted from the catalog on 2026-09-25 - REPO-STAMP 0.9, HARNESS-PROFILE 0.9, REPO-LAYOUT 0.9, RULE-DELIVERY 0.9. A working copy for executing this ticket without the catalog, not a second source: the catalog stays authoritative, and this section is deleted when the ticket moves to done/.

From the shared contracts catalog, `rule-adoption/README.md`:

**§2 `REPO-STAMP`**

> 3. **Three keys are required at the top level - `canon`, `overlay`, `ledgerShape` - and three inside
>    `canon`: `version`, `coreDigest`, `model`.** A missing one is an error against the stamp, not a default.
>    `overlay` may be an empty list and `ledgerShape` may be the string `none`; that is a declaration, and it
>    is a different thing from the key being absent.
> 4. **Every other key is optional and its absence has a written meaning**, which is the half of
>    [`VERSIONING.md`](../_meta/VERSIONING.md) §4 rule 7 this contract owes. The defaults: `role` absent
>    means `product`; `shapes`, `editions`, `channels`, `byteIdenticalPairs`, `vendorAllow` and `exemptions`
>    absent mean empty; `versionShape.tagRegex` null means the repository has no tags;
>    `site.kind` `none` means it serves no pages; `privacy.carveOut` null means the carve-out is not claimed;
>    `canon.contribRecord` null means no record is claimed and none is looked for.

> 8. **The stamp is written by the adoption run, not by hand**, and `canon.coreDigest` in particular is
>    recomputed against the live rule set rather than copied from another repository. A copied digest claims
>    a reconciliation that never happened, which is the one lie this file is able to tell.

**§3 `HARNESS-PROFILE`**

> 7. **A missing profile means the defaults, and an unresolvable project root is a terminating error naming
>    every attempt** - never a guess, because a harness that runs against the wrong tree writes into it.

**§4 `REPO-LAYOUT`**

> 2. **`docs/contracts/<ID>.md` holds pointers into this catalog and never a copy.** The file is named after
>    the contract id it points at - `STREAM-BANK.md`, `OCR-OVERLAY.md` - and a family that one product
>    implements as a unit may share one file (`FDSEC.md` for `FDSEC-FORMAT` and `FDSEC-BEHAVIOUR`) as long
>    as it names every id inside. `docs/contracts/README.md` is the index. A pointer carries the contract id,
>    its version, its home, this repository's role, and what the repository must do to stay conformant. A
>    file there that names no id and runs long is a copy, and a reader may say so. A reader also accepts the
>    older `CONTRACT_<ID>.md` spelling and reads it the same way.
> 3. **Outside `docs/contracts/`, the type prefix is the interface; the folder is not.** `SPECIFICATION_`,
>    `ROADMAP_`, `PROGRESS_`, `RESEARCH_`, `PLAN_`, uppercase and type first: that prefix is what a glob and
>    an index rely on when a folder is reorganized, and it is what may not be renamed casually. A pointer is
>    the exception - its folder says what it is, and its name is the id a reader looks it up by.

> 6. **An archive is frozen.** Documents under a `done/` or `archive/` tree may predate all of this and are
>    left as they are; the convention applies to new documents. A tool that "fixes" an archive is destroying
>    a record.

**§5 `RULE-DELIVERY`**

> 3. **The core digest covers the rule documents only.** It is a SHA-256 over each rule document's name and
>    its text with newlines normalized, in filename order, excluding the rule tree's own `README.md` and its
>    spread prompt - so one repository's own record changing can never mark every repository stale.

> 5. **The staleness ladder is judged from the digest, never from a commit id.** Equal digests mean nothing
>    to do. A differing digest is a warning, and becomes an error once the declared adoption date is more
>    than 180 days old. A tool states which of the two it is; a repository behind by a warning owes a
>    reconciliation of the changed documents, one behind by an error owes a full re-adoption.

> 7. **Nothing hand-edits the version pair.** It is written by the deploy path, which is the only writer of
>    `CANON_VERSION` and of the plugin version derived from it.

This product's row, from the shared contracts catalog, `_meta/REGISTRY.md` (step 4 lists what it gets wrong):

> | `REPO-STAMP`, `HARNESS-PROFILE`, `REPO-LAYOUT`, `RULE-DELIVERY` | doc-html-translate | P/C | 0.9 | 0.9 | 2026-09-24 | **consumer/producer.** Root `.sza-canon.json` (role: product, overlay A/C, ledgerShape: 2), `.sza-profile.json` at root, `AGENTS.md` authoritative canon pointer, pointers under `docs/contracts/<ID>.md` indexed by `docs/contracts/README.md`. |

No exception row for this product under these four ids.
