# What canon 2026.09.24.1 asks of this repository that it does not hold yet

**Status:** Draft
**Priority:** 50
**Date:** 2026-09-25

> Carried forward from the canon re-sync of 2026-09-25
> ([ticket 22](done/22_2026-09-23_contract-rule-adoption-sync.md), Direction A step 2). That run reconciled
> the 13 rule documents changed between canon `2026.09.06.1` and `2026.09.24.1`. Most asked for nothing
> new here or were already held; the four items below were not, and each is a standing artifact to build or
> a naming decision, not a fix. Owner decisions come first.

## What / why

The canon added three duties that bind every adopting repository, and the contract re-read left one naming
gap. None of them is a defect in shipped code. Each is a place where a claim this product makes (every
document has one home, the privacy page is true, the release honours its contracts, the layout is the
interface) is kept in prose rather than in a form a check can run against.

## Findings (measured 2026-09-25)

1. **No documentation registry** (DOCUMENTATION_CONCEPT §6). The mandatory half, items 1-5: no record per
   maintained document, no area / change-trigger facets, no validation from the registry to the tree or
   back. The site half, items 6-8, applies as well: Pages serves the repository root (stamp `site.root`
   `"."`), with seven declared pages and ten language folders (`ar/`, `bn/`, `de/`, `es/`, `fr/`, `hi/`,
   `it/`, `pt/`, `ur/`, `zh/`). `sitemap.xml` is tracked and nothing in `scripts/`, `tools/` or `.claude/`
   generates it, against item 7. Item 5 wants the record-shape number in the stamp beside the ledger
   shape; that is a stamp write and goes through the adopt-canon run. The stamp's `site.pages` list belongs
   to [ticket 26](26_2026-09-23_contract-product-web-pages-sync.md), which must not collide with this.
2. **No security posture inventories** (SECURITY_AND_PRIVACY §7). The facts exist only in prose -
   `privacy.html`, `extension-privacy.html`, `extension/store/PRIVACY.md` - and nothing derives those
   three from one set of rows (item 5). Candidate rows found this date, not yet an inventory:
   - Permissions, extension edition (`extension/manifest.json`): `declarativeNetRequest`, `scripting`,
     `offscreen`, `storage`, `contextMenus`, host `<all_urls>`. The app edition has no manifest, so item 8
     applies: capability rows for the Explorer registration (`internal/windowsreg`), the output folders it
     writes, and the bundled external tools it runs.
   - Network surfaces: Google Cloud Translation (`internal/translator/google.go`, opt-in, the user's key);
     Ollama (`internal/translator/ollama.go`, `localhost:11434` by default); the tessdata download
     (`internal/ocr/tessdata.go`, GitHub, only on request); the GUI's HTTP server
     (`cmd/doc-html-ui/main.go`, `127.0.0.1:0`, host and origin guard in `guard.go`); the extension's
     language-data fetch (`extension/src/ocr-lang.js`, `tessdata.projectnaptha.com`).
   - Item 6's consistency check (manifest against inventory both ways, justifications against the strings
     shown, "no telemetry" against the build's dependency set) has no home yet; the pre-release sweep here
     is `scripts/check.ps1` plus the preflight in `scripts/release.ps1`.
3. **No contract gate on the release path** (RELEASE_AND_DISTRIBUTION §2 "Contract gate", CONTRACTS §6).
   [`DEV/RELEASE.md`](../RELEASE.md) and `.claude/commands/release.md` never mention the catalog, and
   `scripts/release.ps1` blocks on gate evidence and a dirty tree but reads no registry row. The catalog is
   not reachable from a clone, so the gate has to report UNVERIFIED there rather than PASS, and the one
   tracked file allowed to name the catalog's path is `AGENTS.md`.
4. **Research notes have no declared name** (`REPO-LAYOUT` rule 3). Notes under `DEV/research/` carry no
   `RESEARCH_` prefix and follow no written scheme (`competitor_feature_research_ru.md`,
   `ocr_rescue_third_axis_2026-09-25.md`, `audit_2026-09-24/`). Recorded as this product's dated exception
   in the catalog registry, until 2026-12-31. The ticket-name half of the same rule is not this ticket's:
   it waits on the catalog's answer to `PROPOSAL-2026-09-23-own-spec-scheme.md`.

Checked in the same run and **held**, so not carried: the release step refuses without a verdict naming
what it judged (INVARIANTS 5 - `scripts/release.ps1` blocks on missing, failed, could-not-verify or stale
gate evidence); SUPPORT_AND_FEEDBACK §7 (no usage counter found; the GUI's report mail takes its address
from one backend value and its subject names the product and version).

## Done criteria

- [ ] The owner has decided, per item, build now or defer with a dated reason.
- [ ] 1: a registry covers every maintained document; its validation and reverse-coverage checks run in
      `scripts/check.ps1` and pass; `sitemap.xml` is generated from it.
- [ ] 2: both inventories exist, dated, and registered; the consistency check runs in the pre-release
      sweep; the three privacy texts are derived from the rows.
- [ ] 3: the release flow prints a contract-gate verdict (PASS / WARN / FAIL / UNVERIFIED) before the tag
      step, and a FAIL stops it.
- [ ] 4: the research-note naming is written in `CLAUDE.md`, or new notes carry `RESEARCH_`; the registry
      exception row is closed.
