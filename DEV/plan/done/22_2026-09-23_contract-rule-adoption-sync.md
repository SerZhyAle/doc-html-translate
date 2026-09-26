# The canon stamp and the repo layout, declared against their contracts

**Status:** Implemented - 2026-09-25 on the owner's machine: stamp re-synced to canon `2026.09.24.1` by the adopt-canon run (compliance gate 0 errors, 0 warnings), registry rows corrected, B1-B6 filed or withdrawn in writing. What the re-sync found owed is carried by [ticket 31](31_2026-09-25_canon-resync-new-duties.md).
**Priority:** 54
**Date:** 2026-09-23

> Contract sync ticket, both directions. No product code.
> Contracts: `REPO-STAMP`, `HARNESS-PROFILE`, `REPO-LAYOUT`, `RULE-DELIVERY`, all 0.9 (domain
> `rule-adoption/`, owner sza-unified-rules, drafts). Pointers in
> [`docs/contracts/`](../../../docs/contracts/).

> **Remote execution (2026-09-25):** the steps marked ⛔ edit the shared contracts catalog or need the
> canon plugin, so they could run only on the owner's machine. All of them ran there on 2026-09-25; see
> "Implementation record" at the end. The contract snapshot quoted for cloud sessions was deleted on the
> move to `done/`, as it said it would be - the catalog is the source.

## What / why

These are the machine-readable interface between the canon and a repository: the stamp file
`.sza-canon.json`, the names tools address a repo by, the harness profile, and the handshake through which
a rule set arrives and is judged stale. `REPO-STAMP`, `REPO-LAYOUT` and `RULE-DELIVERY` bind every
canon-adopting repository, and this one adopted. Its registry row was still the "every canon-adopting
repository - pending" placeholder.

Found by reading the stamp and running the canon's own digest tool on 2026-09-23.

## Findings that drive the work

**Held:** the stamp is at the root, valid, carries every required key and `role: product`; both
exemptions have an id and a reason; the stamped digest is the genuine digest of the 2026.09.06.1 core;
`CLAUDE.md` + `AGENTS.md`, `README.md`, `LICENSE` at the root; ledger shape 2 at `DEV/CHANGELOG.md`.

**Not held:**
- **`RULE-DELIVERY` rule 5 - stale.** The stamp said canon `2026.09.06.1`. *Resolved 2026-09-25* - re-synced
  to `2026.09.24.1`, 13 versions later; `SZA-CANON03` is gone (Direction A step 2).
- **`REPO-LAYOUT` rule 2 - pointers untracked.** *Resolved 2026-09-25* - commit `0c67c4e` added all of
  `docs/contracts/` and deleted `docs/ocr-pipeline.md` and `docs/integration-image-translate.md` in the
  same commit.
- **`REPO-LAYOUT` rule 3 - no type prefix.** Tickets are `DEV/plan/NN_<YYYY-MM-DD>_<slug>.md`, declared in
  `CLAUDE.md`; research notes under `DEV/research/` follow no declared scheme. (The ticket's first draft
  said both schemes were declared in `AGENTS.md` and `CLAUDE.md`; only the ticket scheme is, and only in
  `CLAUDE.md`.) *Recorded 2026-09-25* as a dated exception in the catalog registry, until 2026-12-31; the
  ticket half waits on B4, the research half moved to ticket 31.
- **`REPO-STAMP` rule 4 - gap in the contract, not in the stamp.** The stamp carries keys whose absence has
  no written meaning (B3). *Filed 2026-09-25.*

**`HARNESS-PROFILE`:** not applicable - the shipped harness is never run here and there is no profile file.
Latent risk: the harness defaults (`PLAN/`, `^S\d{4}$` ticket ids, `PLAN/archive`) do not match this repo's
layout, so a harness-based skill run here would misread it (B2).

**Canon-side finding, not this repo's to fix:** the compliance gate declares no host; under Windows
PowerShell 5.1 it reports false errors and exits 1, under pwsh 7 it reports none on the same tree (B6).

## Direction A - the repo conforms

1. Commit `docs/contracts/` together with the staged deletions it replaces, in one commit.
   **Done 2026-09-25** - commit `0c67c4e`.
2. Reconcile the stamp through the canon's adopt-canon flow to the current published version.
   **Done 2026-09-25** - `2026.09.06.1` / `sha256:cdf49be6..` -> `2026.09.24.1` / `sha256:79ee3333..`, the
   digest from the tool (`check-compliance.ps1 -PrintDigest`), the 13 changed rule documents reconciled,
   plus a third `SZA-SEC04` exemption for a test fixture the gate found on the way. Record below.
3. Pointer files for the four ids, listed in the pointer README, with `HARNESS-PROFILE` marked not
   applicable and why. **Done 2026-09-25** (commit `2a536e7`), and brought up to date on the same day by
   this run: `RULE-DELIVERY.md` says current, `REPO-LAYOUT.md` names the exception and the proposals,
   `REPO-STAMP.md` gains the version-only re-stamp lesson and the `path` rule, `HARNESS-PROFILE.md` cites
   its proposal and no longer claims an `AGENTS.md` declaration.
4. Registry: this product's row replacing the placeholder. **Done 2026-09-25, corrected** - the
   2026-09-24 row (which claimed a `.sza-profile.json`, overlay "A/C" and a `HARNESS-PROFILE` role) is
   replaced by three rows: `REPO-STAMP` (P), `REPO-LAYOUT` + `RULE-DELIVERY` (C), `HARNESS-PROFILE` (not
   applicable), all verified 2026-09-25, plus one exception row for `REPO-LAYOUT` rule 3. The generic
   "every canon-adopting repository - pending" placeholder stays; it serves the other repositories.

## Direction B - what the contracts need from this product

Filed on 2026-09-25 as **one** proposal, `rule-adoption/PROPOSAL-2026-09-25-doc-html-translate-rule-adoption.md`,
following CyrFlip's precedent of seconding open proposals in its own file: the catalog lets a product edit
only its own rows and documents, so another product's proposal is not annotated in place.

- **B1** (`RULE-DELIVERY` rule 5, which run moves `canon.adoptedOn`) - **seconded** FileDO's
  `PROPOSAL-2026-09-23-adoption-date.md`, with this repo's history (`adoptedOn` unchanged through
  2026-09-03, the 2026-09-06 version-only re-stamp and 2026-09-25).
- **B2** (`HARNESS-PROFILE`, a repo that never runs the harness) - **filed new**, item 2, with the three
  wrong defaults cited by line.
- **B3** (`REPO-STAMP` absence meanings) - **seconded** FileDO's `PROPOSAL-2026-09-23-stamp-defaults.md`
  and added `exemptions[].path`, with what the reader already does (absent = every path, value = wildcard).
- **B4** (`REPO-LAYOUT` rule 3) - **seconded** FileDO's `PROPOSAL-2026-09-23-own-spec-scheme.md`, asking
  that the carve-out follow the declaration rather than the `PLAN/` folder. Scoped to tickets only.
- **B5** (staleness ladder) - **withdrawn as a separate request**: the canon already records the same
  divergence as its own exception (`RULE-DELIVERY` rule 5, registry section 3, 2026-09-22). This repo's
  case (13 versions behind, gate warn, skill error) is added there as evidence.
- **B6** (gate host) - **filed new**, item 3, re-measured 2026-09-25. The fix itself is the canon's, in its
  own repository.

Also seconded: CyrFlip's item 3 (a version-only re-stamp is a hand write) - this repo's `e3f4301` is a
second instance.

## Done criteria

- [x] `git ls-files docs/contracts` lists every pointer; the two replaced documents are gone from HEAD in
      the same commit. *Commit `0c67c4e`.*
- [x] The compliance gate reports no `SZA-CANON03` finding. *2026-09-25, pwsh 7: `0 error(s), 0 warning(s)`,
      exit 0.*
- [x] Pointer files and the registry row exist. *Pointers since `0c67c4e`, rewritten in `2a536e7`, updated
      2026-09-25; registry rows corrected 2026-09-25.*
- [x] B1-B6 filed or withdrawn in writing here. *See Direction B.*
- [x] `.sza-canon.json` `site.pages` was left to
      [`26_2026-09-23_contract-product-web-pages-sync`](../26_2026-09-23_contract-product-web-pages-sync.md);
      this run did not touch it.

## Open questions

1. Reconcile to the published plugin version now, or wait? *Answered 2026-09-25:* the installed plugin
   `2026.924.1` equals the canon repository's `CANON_VERSION` `2026.09.24.1`, so the run reconciled to it.

## Implementation record (2026-09-25, owner machine)

**Baseline.** `check-compliance.ps1` of plugin `2026.924.1` under pwsh 7: `1 error(s), 1 warning(s)`,
exit 1 - `SZA-SEC04 internal/translator/translator_test.go: live credential literal` and `SZA-CANON03`
(stamp digest `cdf49be6..` against canon `79ee3333..`).

**The `SZA-SEC04` finding** is `testKey = "AIzaTESTKEY-0123456789abcdefghijklmnopqrs"`, added that morning
by commit `1778dbb` for the tests proving the key travels only in the `X-Goog-Api-Key` header and never
survives into a transport error or an echoed error body. A made-up value in a real key's shape, so it was
exempted like the two redaction fixtures before it, with the reason in the stamp. No assertion needs the
shape; the owner may instead shorten the fixture out of the pattern and drop the exemption.

**Reconciliation, 13 changed rule documents plus `CONTRACTS.md`.**
- Held, nothing owed: `CONTRACTS.md`, `INVARIANTS` 10, `PLATFORM_OVERLAYS`, `REPOSITORY_LAYOUT`,
  `LOCALIZATION`, `NEW_PROJECT_CHECKLIST` (pointers named by id with an index, the catalog path in
  `AGENTS.md` only, registry rows and proposals in place - the contract-sync run of 2026-09-22 and its
  `contract-*` tickets are that work). `INVARIANTS` 5 and `TESTING_AND_QA` §5 (a missing verdict or a could-not-verify
  blocks the ship): `scripts/release.ps1` reports BLOCKED unless the last gate exited 0 or 3 on the tree
  HEAD holds and the working tree is clean. `SUPPORT_AND_FEEDBACK` §7: no usage counter; the GUI's report
  mail takes its address from one constant (`authorEmail`, `cmd/doc-html-ui/main.go`) and its subject names
  the product and version.
- No change owed: `AI_USAGE` (a prompt-submit hook's reach - this repo registers no hook of its own),
  `DEVELOPMENT` (an override that narrows a shipped default - no profile, no merge), `README` (index and
  glossary).
- Owed, carried to [ticket 31](31_2026-09-25_canon-resync-new-duties.md): the documentation registry
  (`DOCUMENTATION_CONCEPT` §6), the permission and network-surface inventories (`SECURITY_AND_PRIVACY` §7),
  a contract gate on the release path (`RELEASE_AND_DISTRIBUTION` §2, `CONTRACTS` §6), and the research
  half of `REPO-LAYOUT` rule 3.

**Stamp.** `canon.version` `2026.09.24.1`, `canon.coreDigest`
`sha256:79ee333369303df6fc85335bd67c08029f28356497c63b91e257d7e7b3f14d68` (from `-PrintDigest`), the third
exemption; `adoptedOn` stays `2026-08-18` as the skill directs (B1). No other key changed.

**Evidence.**
- Compliance gate after, pwsh 7: `check-compliance: EPUB_2_HTML - 0 error(s), 0 warning(s) (overlay A,
  canon 2026.09.24.1)`, exit 0.
- Same gate under Windows PowerShell 5.1.26100 on the same tree: `7 error(s), 3 warning(s)`, exit 1, all
  `SZA-STYLE01` / `SZA-STYLE02` from UTF-8 read in the ANSI code page (B6). A script carrying `#requires -Version 7`
  under 5.1 exits 1, so that line alone would not give "could not verify".
- Catalog gate `check-contracts.ps1 -CatalogRoot <catalog>` after the registry edit: `2 error(s),
  1 warning(s)`, exit 1 - both errors pre-existing and another product's (`CTR-KEY`: `app-activation` and
  `clipboard-guard` declare `0.9.1`), the warning catalog-wide (`CTR-STALE`, 7 of 135 rows unverified);
  nothing from this run's rows.
- The canon's own record: `contrib/epub_2_html.md` gained a "Canon re-sync 2026-09-25" section in the
  canon repository.
