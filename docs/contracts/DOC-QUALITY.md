# Pointer: DOC-INTERNAL-QUALITY, DOC-EXTERNAL-QUALITY

- **Id:** `DOC-INTERNAL-QUALITY`, `DOC-EXTERNAL-QUALITY`
- **Version:** 0.9 (both, draft)
- **Home:** the shared contracts catalog, `documentation-quality/README.md` (its path is in [`AGENTS.md`](../../AGENTS.md))
- **Role:** consumer of both - internal engineering docs, and a published site, README trio and store listings
- **Wire carrier:** none - a documentation tree and the checks that hold it

What this repo does to stay conformant:
- `docs/DOCUMENT_REGISTRY.jsonl` declares every maintained document (product areas, update triggers,
  `generated` for a render); `scripts/doc-registry.ps1` holds it both ways, checks each announced page's SEO
  block and hreflang cluster, and keeps `sitemap.xml` equal to its render. Runs in `scripts/check.ps1`.
- Rendered texts come from one source and are gated: `scripts/security-posture.ps1` (privacy blocks,
  `docs/SECURITY_POSTURE.md`), `tests/iconography_test.go` (`docs/GLYPH-MAP.md`), `tests/limits_parity_test.go`.
- House text style in code and UI strings: `tests/typography_test.go`; on site pages: `tests/site_test.go`.
- External rules on the announced pages, as Go tests (run by `scripts/test.ps1` inside `scripts/check.ps1`):
  internal links and anchors resolve, no `http://` (`tests/site_links_test.go`, rule 7); title <= 60 and
  description <= 160 code points, script-switched strings included (`tests/site_seo_test.go`, rule 5);
  a source fingerprint on every localized page, stale = advisory (exit 3), missing = fail
  (`tests/site_l10n_test.go`, rule 3); the termbase `configs/termbase.json` with forbidden variants
  (`tests/site_termbase_test.go`, rule 4); subject index, glossary and workflow captures on the docs trio
  (`tests/site_docs_test.go`, rules 1 and 6).
- Where a rule reads as if it assumed a portal, a proposal is drafted in [`proposals/`](proposals/) until
  it can be filed in the catalog: `PROPOSAL-2026-09-26-doc-html-translate-doc-external-quality.md`.

**Adopted 2026-09-26 with gaps**, each a ticket and a dated exception in the shared registry:
internal rules 3-6 in [ticket 48](../../DEV/plan/48_2026-09-26_contract-doc-internal-quality-gaps.md),
external rules 1-7 in [ticket 49](../../DEV/plan/49_2026-09-26_contract-doc-external-quality-gaps.md).
