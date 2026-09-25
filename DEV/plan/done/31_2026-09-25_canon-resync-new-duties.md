# What canon 2026.09.24.1 asks of this repository that it does not hold yet

**Status:** Implemented - all four items built on 2026-09-25 and moved to `done/`. Still open outside this
ticket: the owner reads the rendered privacy texts before `main` is pushed (Pages publishes on push), because
their `scripting` and `offscreen` sentences are the store permission text the page-OCR ticket reserved for
owner sign-off.
**Priority:** 50
**Date:** 2026-09-25

> Carried forward from the canon re-sync of 2026-09-25
> ([ticket 22](22_2026-09-23_contract-rule-adoption-sync.md), Direction A step 2). That run reconciled
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
   to [ticket 26](../26_2026-09-23_contract-product-web-pages-sync.md), which must not collide with this.
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
   [`DEV/RELEASE.md`](../../RELEASE.md) and `.claude/commands/release.md` never mention the catalog, and
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

- [x] The owner has decided, per item, build now or defer with a dated reason. Build now, all four: the
      owner's instruction on 2026-09-25 was to implement this ticket.
- [x] 1: a registry covers every maintained document; its validation and reverse-coverage checks run in
      `scripts/check.ps1` and pass; `sitemap.xml` is generated from it.
- [x] 2: both inventories exist, dated, and registered; the consistency check runs in the pre-release
      sweep; the three privacy texts are derived from the rows.
- [x] 3: the release flow prints a contract-gate verdict (PASS / WARN / FAIL / UNVERIFIED) before the tag
      step, and a FAIL stops it.
- [x] 4: the research-note naming is written in `CLAUDE.md`, and new notes carry `RESEARCH_`; the registry
      exception row is closed for its research half. Its ticket half stays open under its own reason - it
      was never this ticket's (finding 4).

## Implementation (2026-09-25)

**1. Documentation registry.** [`docs/DOCUMENT_REGISTRY.jsonl`](../../../docs/DOCUMENT_REGISTRY.jsonl), 39 records in
the reference's record shape 1, at the path the canon's own harness defaults to, built from the tree as it
is: 18 announced pages in seven site records, the store and legal texts, every engineering, agent, plan
and research document, and the frozen early notes and ticket archive declared as frozen. Two
program-source HTML groups (the GUI page, the extension's pages) are excluded with reasons in
`configs/doc-registry-exclusions.jsonl`. [`scripts/doc-registry.ps1`](../../../scripts/doc-registry.ps1), in
`scripts/check.ps1`, holds items 1-7: the shape-1 fields and one home per file; registry to tree; tree to
registry over every `.md` and `.html` git would commit; every `.html` served by Pages either announced by
its own `<link rel="canonical">` (this site's permalink) or excluded with a reason; the SEO block of canon
section 3 on every announced page; `sitemap.xml` equal to its render. `-Generate` writes the sitemap, and
its first output was byte-identical to the hand-kept file. [`scripts/doc-query.ps1`](../../../scripts/doc-query.ps1)
answers the two facets (`-Area`, `-Trigger`, `-Path`, `-List`). The stamp declares `docRegistryShape: 1`
and `docRegistryFile` beside the ledger shape, written by an `adopt-canon` run at the unchanged digest
(item 5). Item 8 is a once-per-release step in `DEV/RELEASE.md`, `scripts/release.ps1` and `/release`.
The first run found real gaps, closed here: the ten per-language landings had no `og:image`, Twitter card
or JSON-LD (added from each page's own description and localized screenshot - no new translation), and
four author pages used a `summary` card where the canon asks for `summary_large_image`.

**2. Security posture inventories.** [`docs/security-posture.json`](../../../docs/security-posture.json), dated
2026-09-25: 13 permission rows (the extension's six declarations, the MSIX `runFullTrust`, and six
capability rows for the app, which has no permission manifest - item 8) and 9 network surfaces. Each row
cites its consumers in the code and its shown strings in the strings files, and carries one public
sentence per author language. Every public form renders from the rows: two blocks of `privacy.html`, two
of `extension-privacy.html`, two of `extension/store/PRIVACY.md`, the permission justifications of
`extension/store/LISTING.md`, the runFullTrust justification of `msix/README.md`, and all of
[`docs/SECURITY_POSTURE.md`](../../../docs/SECURITY_POSTURE.md). [`scripts/security-posture.ps1`](../../../scripts/security-posture.ps1),
in `scripts/check.ps1` (item 6), checks the manifest and MSIX declarations against the rows both ways,
every citation against the code, network call sites by reverse coverage, the dependency set of `go.sum`
and `extension/package-lock.json` against a telemetry denylist, the stores' 1000-character limit, and
every render. The first reconciliation found and fixed five divergences the prose had kept:
- both extension privacy texts named neither `scripting` nor `offscreen`, declared since 2026-09-19, and
  still said `contextMenus` serves one menu item where it serves three;
- `LISTING.md` named two menu items by labels the extension does not show ("Read the pictures on this
  page", "Convert to readable HTML" - the real ones are "OCR every image on this page" and "Convert with
  doc-html-translate"), and its host-access justification left out the OCR image fetch;
- `privacy.html` did not mention the OCR language download from GitHub, the GUI's loopback server, the
  Explorer registration or the helper programs, and its list of written files was short;
- `extension-privacy.html` did not say that a document's remote images stay blocked until the reader
  loads them, which `PRIVACY.md` did;
- `msix/README.md` held a third, stale copy of the privacy text and a runFullTrust justification claiming
  no network use but translation; the copy is now a pointer to the page, the justification a render.

**3. Contract gate.** [`scripts/contract-gate.ps1`](../../../scripts/contract-gate.ps1) reads the pointers in
`docs/contracts/` against the catalog's registry. FAIL: a missing or `pending` row, a row verified before
the last `v*` tag, a pointer ahead of the catalog, two MAJOR versions behind, an absent adoption with no
open exception, an expired exception. WARN: behind within one MAJOR, a row and a pointer that disagree,
an absence under a dated exception, a row without a pointer. It finds the catalog through `-Catalog`,
`SZA_CONTRACTS_CATALOG` or the one sentence of `AGENTS.md` that names it, and says UNVERIFIED (exit 2),
never PASS, when it cannot. `scripts/release.ps1` runs it, prints the verdict and each finding under the
gate-evidence line, blocks the tag step on FAIL or UNVERIFIED, and now exits 1 while either line blocks.
It is placed in a new `release` class of `configs/check-placement.jsonl`, held by
`tests/placement_test.go`. Its first run on the live catalog: WARN, five rows owned by other open work -
`APP-BEHAVIOUR` and `APP-STYLE` rows at 0.9 against pointers at 0.10 (ticket 23), `PAGE-STYLE` read at 1.0
against 1.1 (ticket 26), `CHECK-VERDICT` one minor behind, `INSTALL-TRUST` under its exception (ticket 27).

**4. Research-note naming.** `CLAUDE.md` ("Research notes") declares
`DEV/research/RESEARCH_<topic-slug>_<YYYY-MM-DD>.md`; the eighteen entries named before keep their names,
because tickets and the frozen archive link to them. [`tests/research_naming_test.go`](../../../tests/research_naming_test.go)
holds that list and fails on any new unprefixed name (probed with a throwaway `new_note_2026-09-25.md`:
FAIL). The `/research` command names the same scheme. In the catalog, the `REPO-LAYOUT` exception row is
narrowed to the ticket half, and the adoption row says rule 3 is held for research notes.

**Tests.** `tests/posture_scripts_test.go` drives the three new scripts over scratch trees to every
documented outcome - PASS, each FAIL rule, COULD NOT VERIFY.

**Evidence, 2026-09-25.** `scripts/check.ps1`: `doc-registry: PASS (39 record(s), 204 document file(s)
covered, 18 page(s) announced)`, `security-posture: PASS (13 permission row(s), 9 network surface(s), 7
declaration(s), 57 dependencies, 9 render(s))`, `test-extension: PASS (run=271 skipped=0)`,
`parity-check: PASS`, and `ok doc-html-translate/tests` (the new tests with the site, placement and
typography suites). The aggregate was `check: FAIL (3: test, lint, typo)`, all three from files this ticket
did not touch - `git diff -- internal/ cmd/` is empty: `internal/pdf` `TestExtract_Volume3ImagesOnTheirPages`
and `internal/procrun` `TestRunMissingBinaryKeepsThePathError` fail again on a rerun, `internal/bundledtools`
`TestExtractSetConcurrent` passed on it; lint reports three unchecked `windows.CloseHandle` in the
`proc_windows.go` files; typos reports 85 findings, none in a file changed here. `scripts/contract-gate.ps1`:
`PASS WITH ADVISORIES (5 warning ..)`, exit 3. Canon compliance: 1 error before (`SZA-SEC04` on the
archived ticket 22), 0 errors and 1 warning after (`SZA-CTR01`, ticket 23), exit 0.
