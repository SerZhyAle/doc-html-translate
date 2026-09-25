# Internal documentation meets DOC-INTERNAL-QUALITY

**Status:** Draft
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
