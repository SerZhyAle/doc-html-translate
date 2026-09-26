# Internal documentation meets DOC-INTERNAL-QUALITY

**Status:** Partial
**Priority:** 40
**Date:** 2026-09-26

> Contract sync ticket. Contract: `DOC-INTERNAL-QUALITY` 0.9 draft (domain `documentation-quality/`, owner
> FastMediaSorter Android). Pointer: [`docs/contracts/DOC-QUALITY.md`](../../docs/contracts/DOC-QUALITY.md).
> Adopted 2026-09-26; the gaps below are the dated exception in the shared registry until this closes.

## What / why

The adoption run of 2026-09-26 read the seven rules against the tree. Rules 1 (registry, product areas,
update triggers, `generated` as the render role) and 2 (reverse coverage) are held and gated by
`scripts/doc-registry.ps1`, PASS on 2026-09-26 (39 records, 243 files, 18 pages). Rule 7 holds as measured
(no `http://` link, no embedded script in any tracked `.md`) but nothing keeps it so. Four rules are not met:

1. **Rule 3 - cross-links and anchors: no gate.** A scan of the 178 tracked `.md` files in the working tree
   of 2026-09-26 found **22 broken relative links** and 0 broken heading anchors. Two causes:
   - the uncommitted moves of tickets 28, 31 and 33 into `done/` left `../NN_...` links in the `done/`
     files that pointed at them (`done/2026-09-19_page-ocr-overlay.md`, `done/22_..`, `done/24_..`);
   - links still naming tickets by their pre-`NN_` names or by files that never landed:
     `_TEMPLATE_cross-edition.md`, `2026-07-01_cross-edition-parity.md`, `2026-07-17_comic-archives.md`,
     `2026-07-28_thirteen-ui-languages.md` (in `docs/PARITY.md`, `DEV/DOCS_SURFACES.md`,
     `DEV/research/format_verification_sweep.md`, `DEV/research/extension_formats_feasibility_ru.md` and
     three `done/` tickets), and `reference/sza-kit.css` in ticket 26.
2. **Rule 5 - house text style: the gate stops at code and UI.** `tests/typography_test.go` scans Go
   literals, extension `_locales`, splash resources and the site's author pages, not internal `.md` prose.
   Outside code spans and fences there are **311 en/em dashes in 30 files** (most in the imported
   `.claude/commands/*.md`, then `DEV/research/RESEARCH_INDEX.md`, `VALIDATION.md`, ticket 26) and
   **9 three-dot ellipses in 7 files**. The English-only comment rule has no gate either.
3. **Rule 6 - asset references: no gate.** 0 broken image references today; nothing fails a new one.
4. **Rule 4 - generated docs: partly gated.** Held: `sitemap.xml` (`doc-registry.ps1`), `SECURITY_POSTURE.md`
   and the privacy blocks (`security-posture.ps1`), `docs/GLYPH-MAP.md` coverage (`tests/iconography_test.go`),
   the published input limits (`tests/limits_parity_test.go`). Not held: the CLI flag lists in the README
   trio and `docs*.html` are hand-written against `internal/config/flags.go` with no drift check.

## Done when

- One check (a Go test under `tests/` or a step of `scripts/doc-registry.ps1`, CHECK-VERDICT exit codes)
  resolves every relative link, heading anchor and image path in every registry-claimed `.md`, and fails
  on `http://` links and remote `<script>` in them; it runs in `scripts/check.ps1`.
- The 22 links are fixed (a link repair in `done/` is allowed - it changes no recorded fact).
- The typography guard covers internal `.md` prose with the same author-language rule, and the existing
  dashes and ellipses are fixed or allowlisted with a reason.
- A test fails when a flag in `internal/config/flags.go` is missing from the README trio's flag list, or
  the drift is recorded as a proposal to the contract owner if a hand-written list is judged enough.
- The shared registry row is re-verified and the exception row closed.

## Evidence (2026-09-26)

- `pwsh -NoProfile -File scripts/doc-registry.ps1` - exit 0, `PASS (39 record(s), 243 document file(s) covered, 18 page(s) announced)`.
- Link, anchor, image, http and typography counts: a throwaway scan of `git ls-files '*.md'` in the adoption
  session (not committed); rebuild it as the check above rather than trusting these numbers.

## Implementation (2026-09-26)

All three checks are Go tests under `tests/`, so they run in `scripts/test.ps1`, which is the `gate`
record of `configs/check-placement.jsonl` that `scripts/check.ps1` runs. No new script and no verdict
vocabulary, so no new placement record (`tests/placement_test.go` asks one only of a script). The file
set is the tracked plus untracked-not-ignored files (the set `scripts/doc-registry.ps1` holds to the
registry, whose reverse coverage makes every `.md` in it registry-claimed). Shared reading (prose lines
with fences, code spans and HTML comments blanked; GitHub heading slugs) is in
`tests/markdown_scan_test.go`.

- **Rules 3, 6, 7 - links, anchors, images, `http://`, remote script.** `tests/doc_links_test.go`,
  `TestDocLinks`: every Markdown link, reference definition and `href`/`src` in the prose must resolve
  to a file or folder in that set (so a link into a gitignored folder fails too), a `#fragment` into a
  `.md` must be a heading slug or explicit id there, `http://` fails, and a `<script src>` from another
  origin fails. `TestDocLinkRules` pins the judgement on fixed inputs. Before the repairs it reported
  26 findings (the adoption scan counted 22; it did not count the gitignored `test_doc/CORPUS.md`
  target, and the `#get` anchor it would have flagged sits in a quoted HTML fence, now read as code).
  Repaired:
  - moved tickets: `done/2026-09-19_page-ocr-overlay.md` (28), `done/22_..` (31), `done/24_..` (33),
    `done/28_..` (`docs/PARITY.md` depth, ticket 30 one level up);
  - files never committed (no trace in git history): `_TEMPLATE_cross-edition.md`,
    `2026-07-01_cross-edition-parity.md`, `2026-07-28_thirteen-ui-languages.md`,
    `2026-07-17_comic-archives.md`, `2026-07-17_ocr-pool-per-book.md` and three `2026-07-17_` OCR
    tickets in `done/2026-07-01_app-ocr-image-overlay.md` - unlinked to a code-span name; in
    `docs/PARITY.md` and `DEV/DOCS_SURFACES.md` the text now says they were never committed;
  - `test_doc/CORPUS.md` (gitignored) unlinked in `DEV/research/format_verification_sweep.md`;
  - ticket 26's quoted `reference/sza-kit.css` now points at the kit snapshot in its own folder.
- **Rule 5 - house text style in Markdown prose.** `tests/typography_test.go`,
  `TestTypographyMarkdownProse`, applies `badTypography` per prose line, language from the file suffix
  (`_RU`, `_UK`, else `en` - all three are author languages). `badTypography` now also rejects the en
  dash for author languages (no Go literal and no `en`/`ru`/`uk` locale string carried one); the
  `<title>` exception is unchanged. 310 reported lines were fixed in 35 files (`.claude/commands/*.md`,
  `.claude/agents/*.md`, `DEV/research/RESEARCH_INDEX.md`, `VALIDATION.md`, `CODE_QUALITY.md`,
  `DEV/CHANGELOG.md`, `DEV/DOCS_SURFACES.md`, OCR phase files in `done/`); the quoted rule literals
  became code spans (`..` not `...`), and the changelog's `Alice in Wonderland — Page 5` example of the
  title exception is a code span. One allowlist: `quotedSections` skips the blockquoted lines of ticket
  26's "Contract snapshot" (verbatim catalog text, 17 lines); the entry fails once that section is gone.
- **Rule 4 - README flag tables.** `tests/readme_flags_test.go`, `TestReadmeFlagTables`: every flag
  defined in `internal/config/flags.go` must have a flag-table row in `README.md`, `README_RU.md` and
  `README_UK.md`, and no row may name an undefined flag. The trio was already complete (27 flags), so no
  README changed; verified to fail when a row is removed.
- Pointer `docs/contracts/DOC-QUALITY.md` lists the three tests.

**Open:**
- The shared registry row cannot be re-verified from this session (the catalog and the registry are
  not reachable here): the owner re-verifies it and closes the exception row. That is the one reason
  the status is `Partial`.
- Not gated, outside this ticket's checks: the English-only comment rule, and the `docs*.html` flag
  mentions (a curated subset in prose, not a table; the external rules are ticket 49's).
- `scripts/doc-registry.ps1` was not run here (no PowerShell); the edits add no document and change no
  registry claim.
